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

const (
	busyTimeoutMs = 5000
	busyRetries   = 3
	busyRetryWait = 100 * time.Millisecond
)

func sqliteDSN(dsn string) string {
	sep := "?"
	if strings.ContainsRune(dsn, '?') {
		sep = "&"
	}

	return dsn + sep + "mode=rwc&_busy_timeout=" + strconv.Itoa(busyTimeoutMs) +
		"&_journal_mode=WAL&_synchronous=NORMAL"
}

func openSQLite(dsn string) (*sql.DB, error) {
	connector, err := sqlite.NewConnector(sqliteDSN(dsn))
	if err != nil {
		return nil, err
	}

	return sql.OpenDB(busyConnector{Connector: connector}), nil
}

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

	return strings.Contains(err.Error(), "SQLITE_BUSY") ||
		strings.Contains(err.Error(), "database is locked")
}

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

type busyConnector struct{ driver.Connector }

func (c busyConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &busyConn{Conn: conn}, nil
}

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
