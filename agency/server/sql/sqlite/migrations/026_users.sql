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






ALTER TABLE users RENAME TO users_legacy;

CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NOT NULL UNIQUE REFERENCES actors (id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
  status TEXT NOT NULL DEFAULT 'invited'
    CHECK (status IN ('invited', 'active', 'disabled')),
  last_login_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX idx_users_role ON users (role);

INSERT INTO actors (
  kind, username, name, description, password_hash, password_updated_at,
  created_at, updated_at
)
SELECT 'user', username, username, '', password_hash, updated_at,
       created_at, updated_at
FROM users_legacy;

INSERT INTO users (actor_id, role, status, created_at, updated_at)
SELECT a.id, 'admin', 'active', a.created_at, a.updated_at
FROM actors a
JOIN users_legacy u ON u.username = a.username
WHERE a.kind = 'user';

DROP TABLE users_legacy;

-- +goose Down

CREATE TABLE users_old (
  id INTEGER PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
INSERT INTO users_old (username, password_hash, created_at, updated_at)
SELECT a.username, CAST(COALESCE(a.password_hash, '') AS TEXT),
       a.created_at, a.updated_at
FROM actors a
JOIN users u ON u.actor_id = a.id;

DROP TABLE users;
ALTER TABLE users_old RENAME TO users;
