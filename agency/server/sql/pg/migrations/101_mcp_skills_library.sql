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








CREATE TABLE mcp_servers (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  owner_kind TEXT NOT NULL CHECK (owner_kind IN ('org', 'project', 'team', 'agent', 'group')),
  owner_id TEXT NOT NULL DEFAULT '',
  
  url TEXT NOT NULL DEFAULT '',
  headers_json TEXT NOT NULL DEFAULT '{}',
  connect_timeout_ms BIGINT NOT NULL DEFAULT 15000,
  request_timeout_ms BIGINT NOT NULL DEFAULT 900000,
  tools_json TEXT NOT NULL DEFAULT '{}',
  direct_tools BIGINT NOT NULL DEFAULT 1,
  enabled BIGINT NOT NULL DEFAULT 1,
  created_by_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);
CREATE INDEX idx_mcp_servers_owner ON mcp_servers (owner_kind, owner_id);

CREATE TABLE skills (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  files_json TEXT NOT NULL DEFAULT '[]',
  disable_model_invocation BIGINT NOT NULL DEFAULT 0,
  owner_kind TEXT NOT NULL CHECK (owner_kind IN ('org', 'project', 'team', 'agent', 'group')),
  owner_id TEXT NOT NULL DEFAULT '',
  enabled BIGINT NOT NULL DEFAULT 1,
  created_by_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);
CREATE INDEX idx_skills_owner ON skills (owner_kind, owner_id);

CREATE TABLE mcp_installs (
  id BIGSERIAL PRIMARY KEY,
  entry_id BIGINT NOT NULL REFERENCES mcp_servers (id) ON DELETE CASCADE,
  scope_kind TEXT NOT NULL CHECK (scope_kind IN ('org', 'project', 'team', 'agent', 'group')),
  scope_id TEXT NOT NULL DEFAULT '',
  enabled BIGINT NOT NULL DEFAULT 1,
  created_by_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL,
  created_at BIGINT NOT NULL,
  UNIQUE (entry_id, scope_kind, scope_id)
);
CREATE INDEX idx_mcp_installs_scope ON mcp_installs (scope_kind, scope_id);

CREATE TABLE skill_installs (
  id BIGSERIAL PRIMARY KEY,
  entry_id BIGINT NOT NULL REFERENCES skills (id) ON DELETE CASCADE,
  scope_kind TEXT NOT NULL CHECK (scope_kind IN ('org', 'project', 'team', 'agent', 'group')),
  scope_id TEXT NOT NULL DEFAULT '',
  enabled BIGINT NOT NULL DEFAULT 1,
  created_by_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL,
  created_at BIGINT NOT NULL,
  UNIQUE (entry_id, scope_kind, scope_id)
);
CREATE INDEX idx_skill_installs_scope ON skill_installs (scope_kind, scope_id);

CREATE TABLE task_integration_reports (
  task_id TEXT PRIMARY KEY REFERENCES tasks (id) ON DELETE CASCADE,
  payload_json TEXT NOT NULL,
  config_revision TEXT NOT NULL DEFAULT '',
  reported_at BIGINT NOT NULL
);

-- +goose Down
DROP TABLE task_integration_reports;
DROP TABLE skill_installs;
DROP TABLE mcp_installs;
DROP TABLE skills;
DROP TABLE mcp_servers;