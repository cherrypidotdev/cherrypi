-- CherryPi: A multi-agent-multi-human self-hosted agency
-- Copyright (C) 2026 CherryPi authors <cherrypi.dev>
--
-- This program is free software: you can redistribute it and/or modify
-- it under the terms of the GNU Affero General Public License as published
-- by the Free Software Foundation, either version 3 of the License, or
-- (at your option) any later version.
--
-- This program is distributed in the hope that it will be useful,
-- but WITHOUT ANY WARRANTY; without even the implied warranty of
-- MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
-- GNU Affero General Public License for more details.
--
-- You should have received a copy of the GNU Affero General Public License
-- along with this program.  If not, see <https://www.gnu.org/licenses/>.

-- +goose Up
-- Personal access tokens (`pat_…`), the SDK/CLI credential. A PAT carries the
-- actor's full permissions (minus admin capabilities, enforced in the
-- service). Stored hashed; `prefix` is the displayable fragment.
CREATE TABLE personal_access_tokens (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  token_hash BLOB NOT NULL,
  prefix TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  last_used_at INTEGER,
  expires_at INTEGER
);
CREATE UNIQUE INDEX idx_personal_access_tokens_token_hash
  ON personal_access_tokens (token_hash);
CREATE INDEX idx_personal_access_tokens_actor ON personal_access_tokens (actor_id);
CREATE INDEX idx_personal_access_tokens_expires ON personal_access_tokens (expires_at);

-- +goose Down
DROP TABLE personal_access_tokens;
