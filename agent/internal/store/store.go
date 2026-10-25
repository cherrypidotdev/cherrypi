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
	"log"

	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
	_sql "cherrypi.vladfaust.com/agent/sql"
	"github.com/pressly/goose/v3"
)

func NewStore(
	dbDsn string,
	targetMigration *uint16,
) Store {
	log.Printf("Using SQLite database at %s", dbDsn)

	db, err := openSQLite(dbDsn)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}

	if err := migrateSqlite(db, targetMigration); err != nil {
		log.Fatalf("Failed to run SQLite migrations: %v", err)
	}

	return Store{
		db:      db,
		queries: sqlitedb.New(db),
	}
}

func migrateSqlite(db *sql.DB, targetMigration *uint16) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	goose.SetBaseFS(_sql.Migrations)

	if targetMigration != nil {
		target := int64(*targetMigration)

		current, err := goose.GetDBVersion(db)
		if err != nil {
			return err
		}

		log.Printf("Migrating SQLite database from version %d to %d", current, target)
		if current <= target {
			if err := goose.UpTo(db, "migrations", target); err != nil {
				return err
			}
		} else {
			if err := goose.DownTo(db, "migrations", target); err != nil {
				return err
			}
		}
	} else {
		log.Printf("Migrating SQLite database to latest version")
		if err := goose.Up(db, "migrations"); err != nil {
			return err
		}
	}

	return nil
}

type Store struct {
	db      *sql.DB
	queries *sqlitedb.Queries
}

func (s Store) CreateCredential(
	ctx context.Context,
	p sqlitedb.CreateCredentialParams,
) (sqlitedb.Credential, error) {
	return s.queries.CreateCredential(ctx, p)
}

func (s Store) GetCredential(ctx context.Context, username string) (sqlitedb.Credential, error) {
	return s.queries.GetCredential(ctx, username)
}

func (s Store) ListCredentials(ctx context.Context) ([]sqlitedb.Credential, error) {
	return s.queries.ListCredentials(ctx)
}

func (s Store) DeleteCredential(ctx context.Context, username string) error {
	return s.queries.DeleteCredential(ctx, username)
}

func (s Store) ListSettings(ctx context.Context) ([]sqlitedb.Setting, error) {
	return s.queries.ListSettings(ctx)
}

func (s Store) UpsertSetting(ctx context.Context, p sqlitedb.UpsertSettingParams) error {
	return s.queries.UpsertSetting(ctx, p)
}

func (s Store) CreateSession(
	ctx context.Context,
	p sqlitedb.CreateSessionParams,
) (sqlitedb.Task, error) {
	return s.queries.CreateSession(ctx, p)
}

func (s Store) GetSession(ctx context.Context, id string) (sqlitedb.Task, error) {
	return s.queries.GetSession(ctx, id)
}

func (s Store) ListSessions(ctx context.Context) ([]sqlitedb.Task, error) {
	return s.queries.ListSessions(ctx)
}

func (s Store) ListActiveSessions(ctx context.Context, status int64) ([]sqlitedb.Task, error) {
	return s.queries.ListActiveSessions(ctx, status)
}

func (s Store) SetSessionPendingPrompt(
	ctx context.Context,
	p sqlitedb.SetSessionPendingPromptParams,
) error {
	return s.queries.SetSessionPendingPrompt(ctx, p)
}

func (s Store) ClearSessionPendingPrompt(
	ctx context.Context,
	p sqlitedb.ClearSessionPendingPromptParams,
) error {
	return s.queries.ClearSessionPendingPrompt(ctx, p)
}

func (s Store) SetSessionPid(ctx context.Context, p sqlitedb.SetSessionPidParams) error {
	return s.queries.SetSessionPid(ctx, p)
}

func (s Store) SetSessionAgencyRef(
	ctx context.Context,
	p sqlitedb.SetSessionAgencyRefParams,
) error {
	return s.queries.SetSessionAgencyRef(ctx, p)
}

func (s Store) EndSession(ctx context.Context, p sqlitedb.EndSessionParams) error {
	return s.queries.EndSession(ctx, p)
}

func (s Store) DeleteSession(ctx context.Context, id string) error {
	return s.queries.DeleteSession(ctx, id)
}

func (s Store) SetSessionName(ctx context.Context, p sqlitedb.SetSessionNameParams) error {
	return s.queries.SetSessionName(ctx, p)
}

func (s Store) SetSessionWaiting(ctx context.Context, p sqlitedb.SetSessionWaitingParams) error {
	return s.queries.SetSessionWaiting(ctx, p)
}

func (s Store) ReactivateSession(ctx context.Context, p sqlitedb.ReactivateSessionParams) error {
	return s.queries.ReactivateSession(ctx, p)
}

func (s Store) ClearSessionWaiting(
	ctx context.Context,
	p sqlitedb.ClearSessionWaitingParams,
) error {
	return s.queries.ClearSessionWaiting(ctx, p)
}

func (s Store) InsertUsage(ctx context.Context, p sqlitedb.InsertUsageParams) error {
	return s.queries.InsertUsage(ctx, p)
}

func (s Store) SumUsageBySession(
	ctx context.Context,
	sessionID string,
) (sqlitedb.SumUsageBySessionRow, error) {
	return s.queries.SumUsageBySession(ctx, sessionID)
}

func (s Store) ListUsageBySession(ctx context.Context) ([]sqlitedb.ListUsageBySessionRow, error) {
	return s.queries.ListUsageBySession(ctx)
}

func (s Store) ListUsageByDay(ctx context.Context) ([]sqlitedb.ListUsageByDayRow, error) {
	return s.queries.ListUsageByDay(ctx)
}

func (s Store) CreateTerminalRecord(
	ctx context.Context,
	id string,
	kind int64,
	owner, cwd string,
	cols, rows, createdAt int64,
) error {
	return s.queries.CreateTerminalRecord(ctx, sqlitedb.CreateTerminalRecordParams{
		ID:        id,
		Kind:      kind,
		Owner:     owner,
		Cwd:       cwd,
		Cols:      cols,
		Rows:      rows,
		CreatedAt: createdAt,
	})
}

func (s Store) SetTerminalState(
	ctx context.Context,
	id, state, reason string,
	endedAt int64,
) error {
	return s.queries.SetTerminalState(ctx, sqlitedb.SetTerminalStateParams{
		State:     state,
		EndReason: reason,
		EndedAt:   sql.NullInt64{Int64: endedAt, Valid: endedAt != 0},
		ID:        id,
	})
}

func (s Store) ListTerminalsByState(
	ctx context.Context,
	state string,
) ([]sqlitedb.Terminal, error) {
	return s.queries.ListTerminalsByState(ctx, state)
}

func (s Store) InsertTerminalLog(
	ctx context.Context,
	terminalID string,
	kind int64,
	owner, event, reason, detail string,
	at int64,
) error {
	return s.queries.InsertTerminalLog(ctx, sqlitedb.InsertTerminalLogParams{
		TerminalID: terminalID,
		Kind:       kind,
		Owner:      owner,
		Event:      event,
		Reason:     reason,
		Detail:     detail,
		At:         at,
	})
}

func (s Store) ListTerminalLogs(
	ctx context.Context,
	limit int64,
) ([]sqlitedb.TerminalLog, error) {
	return s.queries.ListTerminalLogs(ctx, limit)
}
