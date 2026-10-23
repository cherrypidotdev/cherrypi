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
-- Team-level prompts and default project tags. See the SQLite migration with
-- the same number for the rationale.
ALTER TABLE prompts DROP CONSTRAINT IF EXISTS prompts_scope_check;
ALTER TABLE prompts ADD CONSTRAINT prompts_scope_check CHECK (
  scope IN ('org', 'agent', 'project', 'agent_project', 'team', 'team_project')
);
ALTER TABLE prompts ADD COLUMN team_id BIGINT;
DROP INDEX IF EXISTS idx_prompts_key;
CREATE INDEX idx_prompts_key
  ON prompts (scope, agent_id, team_id, project_id, version DESC);

ALTER TABLE tags ADD COLUMN is_default BIGINT NOT NULL DEFAULT 0;

CREATE TABLE task_team_prompts (
  task_rowid BIGINT NOT NULL,
  team_id BIGINT NOT NULL,
  prompt_id BIGINT,
  project_prompt_id BIGINT,
  PRIMARY KEY (task_rowid, team_id)
);
CREATE INDEX idx_task_team_prompts_team ON task_team_prompts (team_id);

-- +goose Down
DROP INDEX IF EXISTS idx_task_team_prompts_team;
DROP TABLE task_team_prompts;

ALTER TABLE tags DROP COLUMN is_default;

DROP INDEX IF EXISTS idx_prompts_key;
ALTER TABLE prompts DROP CONSTRAINT IF EXISTS prompts_scope_check;
DELETE FROM prompts WHERE scope IN ('team', 'team_project');
ALTER TABLE prompts DROP COLUMN team_id;
ALTER TABLE prompts ADD CONSTRAINT prompts_scope_check CHECK (
  scope IN ('org', 'agent', 'project', 'agent_project')
);
CREATE INDEX idx_prompts_key
  ON prompts (scope, agent_id, project_id, version DESC);
