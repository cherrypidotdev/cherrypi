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
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"time"

	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	_sql "cherrypi.vladfaust.com/agency/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func NewPgStore(dsn string) (Store, error) {
	log.Printf("Using agency PostgreSQL database at %s", redactDSN(dsn))

	db, err := openPostgres(dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to agency PostgreSQL: %w", err)
	}

	if err := migratePostgres(db); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("migrate agency PostgreSQL: %w", err)
	}

	return &PgStore{db: db, q: pgdb.New(db)}, nil
}

func openPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, err
	}

	return db, nil
}

func migratePostgres(db *sql.DB) error {
	migrations, err := fs.Sub(_sql.PgMigrations, "pg/migrations")
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations)
	if err != nil {
		return err
	}

	_, err = provider.Up(context.Background())

	return err
}

func redactDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme == "" {
		return "(non-URL DSN)"
	}

	if u.User != nil {
		u.User = url.User(u.User.Username())
	}

	return u.String()
}
