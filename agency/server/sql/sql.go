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

// Package sql embeds the agency schema and query trees.
//
// There are two dialect trees: `sqlite`, the original and default store, and
// `pg`, a PostgreSQL baseline plus the migrations both dialects share from now
// on. Keeping the trees side by side mirrors sqlc's two engines; see
// sqlc.yaml.
package sql

import (
	"embed"
	"io/fs"
)

// SqliteMigrations is the SQLite goose tree.
//
//go:embed sqlite/migrations/*.sql
var SqliteMigrations embed.FS

// Migrations is the SQLite migration tree rooted where goose expects it. It is
// kept for the historical-migration tests, which walk the SQLite history one
// version at a time.
var Migrations fs.FS = mustSub(SqliteMigrations, "sqlite")

// PgMigrations is the PostgreSQL goose tree, rooted at `pg/migrations`.
//
//go:embed pg/migrations/*.sql
var PgMigrations embed.FS

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}

	return sub
}
