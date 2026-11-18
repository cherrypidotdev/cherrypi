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












CREATE TABLE model_providers (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL DEFAULT '',
  
  
  api TEXT NOT NULL,
  base_url TEXT NOT NULL,
  
  
  api_key_sealed BYTEA,
  headers_json TEXT NOT NULL DEFAULT '{}',
  enabled BIGINT NOT NULL DEFAULT 1,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);

CREATE TABLE models (
  id BIGSERIAL PRIMARY KEY,
  provider_id BIGINT NOT NULL REFERENCES model_providers (id) ON DELETE CASCADE,
  model_id TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  reasoning BIGINT NOT NULL DEFAULT 0,
  input_json TEXT NOT NULL DEFAULT '["text"]',
  context_window BIGINT NOT NULL DEFAULT 0,
  max_tokens BIGINT NOT NULL DEFAULT 0,
  cost_json TEXT NOT NULL DEFAULT '{}',
  enabled BIGINT NOT NULL DEFAULT 1,
  UNIQUE (provider_id, model_id)
);



CREATE TABLE model_grants (
  id BIGSERIAL PRIMARY KEY,
  provider_id BIGINT NOT NULL REFERENCES model_providers (id) ON DELETE CASCADE,
  model_id TEXT NOT NULL DEFAULT '',
  scope_kind TEXT NOT NULL CHECK (scope_kind IN ('org', 'agent', 'team', 'project', 'group')),
  scope_id TEXT NOT NULL DEFAULT '',
  created_at BIGINT NOT NULL,
  UNIQUE (provider_id, model_id, scope_kind, scope_id)
);



CREATE TABLE agent_config_revision (
  id BIGINT PRIMARY KEY CHECK (id = 1),
  revision TEXT NOT NULL
);
INSERT INTO agent_config_revision (id, revision) VALUES (1, '0');

-- +goose Down
DROP TABLE agent_config_revision;
DROP TABLE model_grants;
DROP TABLE models;
DROP TABLE model_providers;
