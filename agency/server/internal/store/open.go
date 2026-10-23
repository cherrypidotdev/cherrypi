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

import "log"

// Open opens the store selected by dsn and migrates it to the latest version.
// PostgreSQL is selected by a postgres:// or postgresql:// URL; everything
// else is a SQLite DSN.
func Open(dsn string) (Store, error) {
	if IsPostgresDSN(dsn) {
		return NewPgStore(dsn)
	}

	return NewSqliteStore(dsn)
}

// NewStore opens the store selected by dsn, exiting on error. It is the
// startup path; tests use Open or the explicit NewSqliteStore/NewPgStore.
func NewStore(dsn string) Store {
	st, err := Open(dsn)
	if err != nil {
		log.Fatalf("Failed to open the agency database: %v", err)
	}

	return st
}
