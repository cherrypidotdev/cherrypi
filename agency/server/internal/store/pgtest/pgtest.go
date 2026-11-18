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

// goose migration with SQLSTATE XX000. It surfaces on the rootless CI runner's

// their schemas at once. goose runs each migration in its own transaction, so

package pgtest

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	attempts = 4
	backoff  = 100 * time.Millisecond
)

const DSNEnv = "CHERRYPI_AGENCY_TEST_DATABASE_DSN"

func Open(t testing.TB, what string, open func() error) {
	t.Helper()

	for attempt := 1; ; attempt++ {
		err := open()
		if err == nil {
			return
		}

		if attempt == attempts || !interruptedFileOpen(err) {
			t.Fatalf("%s: %v", what, err)
		}

		time.Sleep(time.Duration(attempt) * backoff)
	}
}

func interruptedFileOpen(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "XX000" &&
			strings.Contains(pgErr.Message, "Interrupted system call")
	}

	msg := err.Error()

	return strings.Contains(msg, "SQLSTATE XX000") &&
		strings.Contains(msg, "Interrupted system call")
}
