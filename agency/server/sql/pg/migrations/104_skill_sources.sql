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


CREATE TABLE git_credentials (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  kind TEXT NOT NULL CHECK (kind IN ('token', 'ssh_key')),
  host TEXT NOT NULL,
  port BIGINT NOT NULL DEFAULT 0,
  username TEXT NOT NULL DEFAULT '',
  secret_sealed BYTEA,
  known_hosts TEXT NOT NULL DEFAULT '',
  public_key TEXT NOT NULL DEFAULT '',
  fingerprint TEXT NOT NULL DEFAULT '',
  created_by_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);

CREATE TABLE skill_sources (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  owner_kind TEXT NOT NULL CHECK (owner_kind IN ('org', 'project', 'team', 'agent', 'group')),
  owner_id TEXT NOT NULL DEFAULT '',
  repo_url TEXT NOT NULL,
  ref TEXT NOT NULL DEFAULT '',
  path_glob TEXT NOT NULL DEFAULT '.well-known/skills/*/SKILL.md',
  credential_id BIGINT REFERENCES git_credentials (id) ON DELETE RESTRICT,
  update_mode TEXT NOT NULL DEFAULT 'staged' CHECK (update_mode IN ('staged', 'auto')),
  poll_interval_s BIGINT NOT NULL DEFAULT 900,
  paused BIGINT NOT NULL DEFAULT 0,
  synced_commit TEXT NOT NULL DEFAULT '',
  synced_at BIGINT NOT NULL DEFAULT 0,
  checked_at BIGINT NOT NULL DEFAULT 0,
  sync_state TEXT NOT NULL DEFAULT '' CHECK (sync_state IN ('', 'ok', 'pending', 'error')),
  sync_error TEXT NOT NULL DEFAULT '',
  pending_commit TEXT NOT NULL DEFAULT '',
  pending_json TEXT NOT NULL DEFAULT '',
  
  
  
  problems_json TEXT NOT NULL DEFAULT '',
  created_by_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);
CREATE INDEX idx_skill_sources_owner ON skill_sources (owner_kind, owner_id);

CREATE TABLE skill_source_installs (
  id BIGSERIAL PRIMARY KEY,
  source_id BIGINT NOT NULL REFERENCES skill_sources (id) ON DELETE CASCADE,
  scope_kind TEXT NOT NULL CHECK (scope_kind IN ('org', 'project', 'team', 'agent', 'group')),
  scope_id TEXT NOT NULL DEFAULT '',
  enabled BIGINT NOT NULL DEFAULT 1,
  created_by_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL,
  created_at BIGINT NOT NULL,
  UNIQUE (source_id, scope_kind, scope_id)
);
CREATE INDEX idx_skill_source_installs_scope
  ON skill_source_installs (scope_kind, scope_id);

ALTER TABLE skills ADD COLUMN source_id BIGINT REFERENCES skill_sources (id) ON DELETE CASCADE;
ALTER TABLE skills ADD COLUMN source_path TEXT NOT NULL DEFAULT '';
ALTER TABLE skills ADD COLUMN source_commit TEXT NOT NULL DEFAULT '';
ALTER TABLE skills ADD COLUMN source_missing BIGINT NOT NULL DEFAULT 0;
CREATE UNIQUE INDEX idx_skills_source_path ON skills (source_id, source_path);

-- +goose Down
DROP INDEX idx_skills_source_path;
ALTER TABLE skills DROP COLUMN source_missing;
ALTER TABLE skills DROP COLUMN source_commit;
ALTER TABLE skills DROP COLUMN source_path;
ALTER TABLE skills DROP COLUMN source_id;
DROP INDEX idx_skill_source_installs_scope;
DROP TABLE skill_source_installs;
DROP INDEX idx_skill_sources_owner;
DROP TABLE skill_sources;
DROP TABLE git_credentials;