// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// Connection parameters carried by every store connection.
//
//   - busy_timeout makes a contended statement wait instead of failing
//     immediately. modernc applies it before the other pragmas, so switching
//     journal_mode under contention does not itself hit a lock.
//   - WAL lets readers run against the last committed snapshot while a writer
//     is active. The reported task-chat failure was a read (`GetAgent` /
//     `GetSession`) colliding with the event/scheduler writes; rollback-journal
//     mode made them mutually exclusive. WAL removes that collision and leaves
//     only writer-vs-writer, which busy_timeout (and the retry below) covers.
const (
	busyTimeoutMs = 5000
	busyRetries   = 3
	busyRetryWait = 100 * time.Millisecond
)

// sqliteDSN appends the store's connection parameters to a DSN that may
// already carry a query string.
func sqliteDSN(dsn string) string {
	sep := "?"
	if strings.ContainsRune(dsn, '?') {
		sep = "&"
	}

	return dsn + sep + "mode=rwc&_busy_timeout=" + strconv.Itoa(busyTimeoutMs) +
		"&_journal_mode=WAL&_synchronous=NORMAL"
}

// openSQLite opens a *sql.DB whose connections retry a transient SQLite lock.
// It goes through sqlite.NewConnector rather than sql.Open so the registered
// driver's functions and collations are preserved (see the modernc docs).
func openSQLite(dsn string) (*sql.DB, error) {
	connector, err := sqlite.NewConnector(sqliteDSN(dsn))
	if err != nil {
		return nil, err
	}

	return sql.OpenDB(busyConnector{Connector: connector}), nil
}

// isBusy reports whether err is SQLite's BUSY or LOCKED result, the transient
// contention error a retry can clear. Extended result codes (BUSY_SNAPSHOT,
// BUSY_RECOVERY, LOCKED_SHAREDCACHE, …) share the primary code in the low byte.
func isBusy(err error) bool {
	if err == nil {
		return false
	}

	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xff {
		case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
			return true
		}

		return false
	}

	// A driver or wrapper may only expose the message; match it as a fallback.
	return strings.Contains(err.Error(), "SQLITE_BUSY") ||
		strings.Contains(err.Error(), "database is locked")
}

// retryBusy runs fn until it stops returning a busy error, waiting with linear
// backoff between attempts. A cancelled context short-circuits the wait.
func retryBusy(ctx context.Context, fn func() error) error {
	var err error

	for attempt := 0; ; attempt++ {
		if err = fn(); !isBusy(err) {
			return err
		}
		if attempt >= busyRetries-1 {
			return err
		}

		select {
		case <-ctx.Done():
			return err
		case <-time.After(busyRetryWait * time.Duration(attempt+1)):
		}
	}
}

// busyConnector interposes on the modernc connector so every statement run on a
// pooled connection retries a transient lock instead of surfacing it.
// database/sql reaches queries through the connection or its prepared
// statement, so wrapping both is what makes the retry cover every store method
// — including the QueryRow-based reads whose error would otherwise only appear
// at Scan, past any wrapper.
type busyConnector struct{ driver.Connector }

// Connect implements driver.Connector.
func (c busyConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &busyConn{Conn: conn}, nil
}

// busyConn implements the optional driver interfaces database/sql prefers so it
// can retry on the direct path. A method returns driver.ErrSkip when the
// underlying connection lacks the interface, handing the call back to
// database/sql's prepare/exec fallback (which lands on busyStmt).
type busyConn struct{ driver.Conn }

func (c *busyConn) ExecContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Result, error) {
	execer, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	var res driver.Result

	err := retryBusy(ctx, func() error {
		var err error

		res, err = execer.ExecContext(ctx, query, args)

		return err
	})

	return res, err
}

func (c *busyConn) QueryContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	var rows driver.Rows

	err := retryBusy(ctx, func() error {
		var err error

		rows, err = queryer.QueryContext(ctx, query, args)

		return err
	})

	return rows, err
}

func (c *busyConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	preparer, ok := c.Conn.(driver.ConnPrepareContext)
	if !ok {
		return c.Prepare(query)
	}

	var stmt driver.Stmt

	err := retryBusy(ctx, func() error {
		var err error

		stmt, err = preparer.PrepareContext(ctx, query)

		return err
	})
	if err != nil {
		return nil, err
	}

	return &busyStmt{Stmt: stmt}, nil
}

func (c *busyConn) BeginTx(
	ctx context.Context,
	opts driver.TxOptions,
) (driver.Tx, error) {
	beginner, ok := c.Conn.(driver.ConnBeginTx)
	if !ok {
		return nil, driver.ErrSkip
	}

	var tx driver.Tx

	err := retryBusy(ctx, func() error {
		var err error

		tx, err = beginner.BeginTx(ctx, opts)

		return err
	})

	return tx, err
}

func (c *busyConn) Ping(ctx context.Context) error {
	if pinger, ok := c.Conn.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}

	return nil
}

func (c *busyConn) ResetSession(ctx context.Context) error {
	if resetter, ok := c.Conn.(driver.SessionResetter); ok {
		return resetter.ResetSession(ctx)
	}

	return nil
}

func (c *busyConn) IsValid() bool {
	if validator, ok := c.Conn.(driver.Validator); ok {
		return validator.IsValid()
	}

	return true
}

// busyStmt retries the context-aware statement methods, covering queries that
// database/sql runs through a prepared statement.
type busyStmt struct{ driver.Stmt }

func (s *busyStmt) ExecContext(
	ctx context.Context,
	args []driver.NamedValue,
) (driver.Result, error) {
	execer, ok := s.Stmt.(driver.StmtExecContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	var res driver.Result

	err := retryBusy(ctx, func() error {
		var err error

		res, err = execer.ExecContext(ctx, args)

		return err
	})

	return res, err
}

func (s *busyStmt) QueryContext(
	ctx context.Context,
	args []driver.NamedValue,
) (driver.Rows, error) {
	queryer, ok := s.Stmt.(driver.StmtQueryContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	var rows driver.Rows

	err := retryBusy(ctx, func() error {
		var err error

		rows, err = queryer.QueryContext(ctx, args)

		return err
	})

	return rows, err
}
