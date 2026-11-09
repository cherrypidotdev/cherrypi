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

package plugins

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/telemetry"
	_ "github.com/jackc/pgx/v5/stdlib"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	_ "modernc.org/sqlite"
)

type PluginDB struct {
	db       *sql.DB
	postgres bool
	schema   string
	path     string
}

func OpenPluginDB(
	ctx context.Context,
	dsn, pluginID, dataDir string,
	requireDatabase bool,
) (*PluginDB, error) {
	if !isPostgres(dsn) {
		dir := filepath.Join(dataDir, "plugins", pluginID, "data")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		path := filepath.Join(dir, "plugin.sqlite3")
		db, err := telemetry.SQLOpen(
			"sqlite",
			path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)",
			semconv.DBSystemNameSQLite,
			telemetry.SQLPoolAttribute("plugin:"+pluginID),
		)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(1)
		return &PluginDB{db: db, path: path}, nil
	}

	if db, err := openPluginPGDatabase(ctx, dsn, pluginID); err == nil {
		return &PluginDB{db: db, postgres: true}, nil
	} else if db, err := createPluginPGDatabase(ctx, dsn, pluginID); err == nil {
		return &PluginDB{db: db, postgres: true}, nil
	} else if requireDatabase {
		return nil, fmt.Errorf("plugin PostgreSQL database required: %w", err)
	}

	db, err := telemetry.SQLOpen(
		"pgx",
		dsn,
		semconv.DBSystemNamePostgreSQL,
		telemetry.SQLPoolAttribute("plugin:"+pluginID),
	)
	if err != nil {
		return nil, err
	}
	schema := "plugin_" + strings.ReplaceAll(pluginID, "-", "_")
	if _, err := db.ExecContext(
		ctx,
		`CREATE SCHEMA IF NOT EXISTS `+quoteIdent(schema),
	); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &PluginDB{db: db, postgres: true, schema: schema}, nil
}

func openPluginPGDatabase(ctx context.Context, dsn, pluginID string) (*sql.DB, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	u.Path = "/plugins_" + strings.ReplaceAll(pluginID, "-", "_")
	db, err := telemetry.SQLOpen(
		"pgx",
		u.String(),
		semconv.DBSystemNamePostgreSQL,
		telemetry.SQLPoolAttribute("plugin:"+pluginID),
	)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func createPluginPGDatabase(ctx context.Context, dsn, pluginID string) (*sql.DB, error) {
	admin, err := telemetry.SQLOpen(
		"pgx",
		dsn,
		semconv.DBSystemNamePostgreSQL,
		telemetry.SQLPoolAttribute("plugin:"+pluginID),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = admin.Close() }()
	name := "plugins_" + strings.ReplaceAll(pluginID, "-", "_")
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE `+quoteIdent(name)); err != nil {
		return nil, err
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	u.Path = "/" + name
	db, err := telemetry.SQLOpen(
		"pgx",
		u.String(),
		semconv.DBSystemNamePostgreSQL,
		telemetry.SQLPoolAttribute("plugin:"+pluginID),
	)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func isPostgres(dsn string) bool {
	return strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://")
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func (p *PluginDB) Close() error { return p.db.Close() }

func migrationsDir(versionDir string, postgres bool) string {
	root := filepath.Join(versionDir, "migrations")
	dialect := "sqlite"
	if postgres {
		dialect = "pg"
	}
	info, err := os.Stat(filepath.Join(root, dialect))
	if err == nil && info.IsDir() {
		return filepath.Join(root, dialect)
	}
	return root
}

func (p *PluginDB) Migrate(ctx context.Context, dir string) error {
	if _, err := p.execRaw(
		ctx,
		`CREATE TABLE IF NOT EXISTS plugin_migrations (version TEXT PRIMARY KEY, applied_at BIGINT NOT NULL)`,
	); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) || dir == "" {
		return nil
	}
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		applied, err := p.migrationApplied(ctx, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if _, err := p.execRaw(ctx, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := p.execRaw(
			ctx,
			`INSERT INTO plugin_migrations (version,applied_at) VALUES (?,?)`,
			name,
			time.Now().UnixMilli(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (p *PluginDB) migrationApplied(ctx context.Context, name string) (bool, error) {
	rows, err := p.Query(ctx, `SELECT version FROM plugin_migrations WHERE version=?`, name)
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func (p *PluginDB) conn(ctx context.Context) (*sql.Conn, error) {
	c, err := p.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if p.schema != "" {
		if _, err := c.ExecContext(
			ctx,
			`SET search_path TO `+quoteIdent(p.schema)+`,public`,
		); err != nil {
			_ = c.Close()
			return nil, err
		}
	}
	return c, nil
}

func (p *PluginDB) execRaw(ctx context.Context, query string, args ...any) (sql.Result, error) {
	query = p.q(query)
	if p.schema == "" {
		return p.db.ExecContext(ctx, query, args...)
	}
	c, err := p.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = c.Close() }()
	return c.ExecContext(ctx, query, args...)
}

func (p *PluginDB) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := p.execRaw(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (p *PluginDB) Query(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	query = p.q(query)
	var rows *sql.Rows
	var err error
	if p.schema == "" {
		rows, err = p.db.QueryContext(ctx, query, args...)
	} else {
		c, cerr := p.conn(ctx)
		if cerr != nil {
			return nil, cerr
		}
		defer func() { _ = c.Close() }()
		rows, err = c.QueryContext(ctx, query, args...)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			switch v := vals[i].(type) {
			case []byte:
				row[col] = string(v)
			case time.Time:
				row[col] = v.UTC().Format(time.RFC3339Nano)
			default:
				row[col] = v
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (p *PluginDB) q(query string) string {
	if !p.postgres {
		return query
	}
	var b strings.Builder
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			fmt.Fprintf(&b, "$%d", n)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
