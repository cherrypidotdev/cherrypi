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










PRAGMA foreign_keys=OFF;

CREATE TABLE actors_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL CHECK (kind IN ('user', 'agent', 'plugin')),
  username TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL,
  password_hash BLOB,
  password_updated_at INTEGER,
  totp_secret BLOB,
  totp_enabled_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
INSERT INTO actors_new (
  id, kind, username, name, description, picture_attachment_id,
  password_hash, password_updated_at, totp_secret, totp_enabled_at,
  created_at, updated_at
)
SELECT
  id, kind, username, name, description, picture_attachment_id,
  password_hash, password_updated_at, totp_secret, totp_enabled_at,
  created_at, updated_at
FROM actors;
DROP TABLE actors;
ALTER TABLE actors_new RENAME TO actors;
CREATE UNIQUE INDEX idx_actors_username ON actors (username);
CREATE INDEX idx_actors_kind ON actors (kind);




CREATE TABLE plugins (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  version TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  manifest_json TEXT NOT NULL,
  runtime TEXT NOT NULL DEFAULT 'lua',
  entry TEXT NOT NULL DEFAULT '',
  checksum TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 0,
  installed_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE plugin_actors (
  plugin_id TEXT PRIMARY KEY REFERENCES plugins (id) ON DELETE CASCADE,
  actor_id INTEGER NOT NULL UNIQUE REFERENCES actors (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL
);





CREATE TABLE plugin_settings (
  plugin_id TEXT NOT NULL REFERENCES plugins (id) ON DELETE CASCADE,
  scope TEXT NOT NULL CHECK (scope IN ('org', 'agent', 'user')),
  target_id TEXT NOT NULL DEFAULT '',
  values_json TEXT NOT NULL DEFAULT '{}',
  enabled INTEGER NOT NULL DEFAULT 0,
  updated_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (plugin_id, scope, target_id)
);

-- The project surface. A project owner invites the plugin actor to the project



CREATE TABLE plugin_project_settings (
  plugin_id TEXT NOT NULL REFERENCES plugins (id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  enabled INTEGER NOT NULL DEFAULT 0,
  allow_task_creation INTEGER NOT NULL DEFAULT 1,
  max_priority INTEGER NOT NULL DEFAULT 0,
  values_json TEXT NOT NULL DEFAULT '{}',
  updated_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (plugin_id, project_id)
);
CREATE INDEX idx_plugin_project_settings_project ON plugin_project_settings (project_id);




CREATE TABLE task_plugins (
  task_id TEXT PRIMARY KEY REFERENCES tasks (id) ON DELETE CASCADE,
  plugin_id TEXT NOT NULL REFERENCES plugins (id) ON DELETE CASCADE,
  tools_json TEXT NOT NULL DEFAULT '[]',
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_task_plugins_plugin ON task_plugins (plugin_id);

CREATE TABLE plugin_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  plugin_id TEXT NOT NULL REFERENCES plugins (id) ON DELETE CASCADE,
  at INTEGER NOT NULL,
  level TEXT NOT NULL,
  kind TEXT NOT NULL,
  invocation_id TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL,
  data_json TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX idx_plugin_logs_plugin ON plugin_logs (plugin_id, at DESC, id DESC);

-- +goose Down
DROP TABLE plugin_logs;
DROP TABLE task_plugins;
DROP TABLE plugin_project_settings;
DROP TABLE plugin_settings;
DROP TABLE plugin_actors;
DROP TABLE plugins;

PRAGMA foreign_keys=OFF;

CREATE TABLE actors_old (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL CHECK (kind IN ('user', 'agent')),
  username TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL,
  password_hash BLOB,
  password_updated_at INTEGER,
  totp_secret BLOB,
  totp_enabled_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
INSERT INTO actors_old
SELECT id, kind, username, name, description, picture_attachment_id,
       password_hash, password_updated_at, totp_secret, totp_enabled_at,
       created_at, updated_at
FROM actors WHERE kind != 'plugin';
DROP TABLE actors;
ALTER TABLE actors_old RENAME TO actors;
CREATE UNIQUE INDEX idx_actors_username ON actors (username);
CREATE INDEX idx_actors_kind ON actors (kind);
