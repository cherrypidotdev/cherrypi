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














CREATE TABLE variables (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scope_kind TEXT NOT NULL CHECK (
    scope_kind IN ('org', 'project', 'team', 'group', 'agent')
  ),
  
  
  scope_id TEXT NOT NULL DEFAULT '',
  
  name TEXT NOT NULL,
  
  note TEXT NOT NULL DEFAULT '',
  secret INTEGER NOT NULL DEFAULT 0,
  
  value TEXT,
  
  value_sealed BLOB,
  created_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  updated_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  UNIQUE (scope_kind, scope_id, name),
  CHECK (
    (secret = 1 AND value IS NULL AND value_sealed IS NOT NULL)
    OR (secret = 0 AND value IS NOT NULL AND value_sealed IS NULL)
  )
);
CREATE INDEX idx_variables_scope ON variables (scope_kind, scope_id);

-- +goose Down
DROP TABLE variables;
