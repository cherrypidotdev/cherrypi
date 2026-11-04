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


CREATE TABLE prompt_proposals (
  id BIGSERIAL PRIMARY KEY,
  scope TEXT NOT NULL CHECK (
    scope IN ('org', 'agent', 'project', 'agent_project', 'team', 'team_project')
  ),
  agent_id BIGINT REFERENCES agents (id) ON DELETE CASCADE,
  team_id BIGINT REFERENCES teams (id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects (id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  base_version BIGINT NOT NULL,
  author_actor_id BIGINT NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  note TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending' CHECK (
    status IN ('pending', 'approved', 'rejected', 'withdrawn')
  ),
  auto_approved BIGINT NOT NULL DEFAULT 0,
  created_at BIGINT NOT NULL,
  decided_at BIGINT,
  decided_by_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL,
  decision_note TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_prompt_proposals_target
  ON prompt_proposals (scope, agent_id, team_id, project_id, status, created_at DESC);
CREATE INDEX idx_prompt_proposals_author
  ON prompt_proposals (author_actor_id, created_at DESC);
CREATE INDEX idx_prompt_proposals_status
  ON prompt_proposals (status, created_at DESC);

ALTER TABLE prompts ADD COLUMN author_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE prompts ADD COLUMN approved_by_actor_id BIGINT REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE prompts ADD COLUMN note TEXT NOT NULL DEFAULT '';
ALTER TABLE prompts ADD COLUMN proposal_id BIGINT REFERENCES prompt_proposals (id) ON DELETE SET NULL;
ALTER TABLE prompts ADD COLUMN auto_approved BIGINT NOT NULL DEFAULT 0;

ALTER TABLE agents ADD COLUMN prompt_auto_approve BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE agents DROP COLUMN prompt_auto_approve;

ALTER TABLE prompts DROP COLUMN auto_approved;
ALTER TABLE prompts DROP COLUMN proposal_id;
ALTER TABLE prompts DROP COLUMN note;
ALTER TABLE prompts DROP COLUMN approved_by_actor_id;
ALTER TABLE prompts DROP COLUMN author_actor_id;

DROP INDEX idx_prompt_proposals_status;
DROP INDEX idx_prompt_proposals_author;
DROP INDEX idx_prompt_proposals_target;
DROP TABLE prompt_proposals;
