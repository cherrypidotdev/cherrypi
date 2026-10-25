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


CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);



CREATE TABLE agents (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  last_seen_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);



CREATE TABLE agent_credentials (
  id INTEGER PRIMARY KEY,
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  last_used_at INTEGER
);

CREATE INDEX idx_agent_credentials_agent_id ON agent_credentials (agent_id);


CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);


CREATE TABLE webhooks (
  id INTEGER PRIMARY KEY,
  agent_id TEXT REFERENCES agents (id) ON DELETE CASCADE,
  url TEXT NOT NULL,
  events TEXT NOT NULL,
  secret TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);



CREATE TABLE notifications (
  id INTEGER PRIMARY KEY,
  agent_id TEXT REFERENCES agents (id) ON DELETE CASCADE,
  session_id TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'info',
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  read_at INTEGER
);

CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);



CREATE TABLE agent_usage (
  id INTEGER PRIMARY KEY,
  agent_id TEXT NOT NULL,
  session_id TEXT NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cost REAL NOT NULL DEFAULT 0,
  active_ms INTEGER NOT NULL DEFAULT 0,
  at INTEGER NOT NULL
);

CREATE INDEX idx_agent_usage_agent_id ON agent_usage (agent_id);
CREATE INDEX idx_agent_usage_session_id ON agent_usage (session_id);

-- +goose Down
DROP TABLE agent_usage;
DROP TABLE notifications;
DROP TABLE webhooks;
DROP TABLE settings;
DROP TABLE agent_credentials;
DROP TABLE agents;
DROP TABLE users;
