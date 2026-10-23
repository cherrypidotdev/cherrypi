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

import "strings"

// Compile-time proof that both dialect implementations satisfy the interface.
var (
	_ Store = (*SqliteStore)(nil)
	_ Store = (*PgStore)(nil)
)

// IsPostgresDSN reports whether dsn selects the PostgreSQL implementation.
// Anything else (a path, `file:...`, `sqlite://...`) is SQLite, which keeps the
// default DSN and every existing deployment working unchanged.
func IsPostgresDSN(dsn string) bool {
	return strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://")
}
