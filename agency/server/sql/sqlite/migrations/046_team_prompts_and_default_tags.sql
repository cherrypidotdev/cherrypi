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
-- Team-level prompts and default project tags.
--
--  * A prompt may now belong to a team (`team`) or to a team × project
--    (`team_project`). The owning team is `prompts.team_id`; the project stays
--    in `prompts.project_id`. A task pins the versions it started with in
--    `task_team_prompts`, one row per owning team, so a later edit or a team
--    change never rewrites a started task's system prompt.
--  * `tags.is_default` pre-selects a tag in the task composer for its project.
--    It is only a default: a creator may still remove it from one task.
--
-- SQLite cannot alter a CHECK constraint, so the prompts table is rebuilt.
-- Foreign keys are off at runtime and row ids are copied verbatim, so the
-- tasks/session_context prompt pins keep pointing at the same rows.
CREATE TABLE prompts_new (
  id INTEGER PRIMARY KEY,
  scope TEXT NOT NULL CHECK (
    scope IN ('org', 'agent', 'project', 'agent_project', 'team', 'team_project')
  ),
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
  team_id INTEGER REFERENCES teams (id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects (id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  version INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);
INSERT INTO prompts_new (id, scope, agent_id, team_id, project_id, content, version, created_at)
SELECT id, scope, agent_id, NULL, project_id, content, version, created_at FROM prompts;
DROP TABLE prompts;
ALTER TABLE prompts_new RENAME TO prompts;
CREATE INDEX idx_prompts_key
  ON prompts (scope, agent_id, team_id, project_id, version DESC);

ALTER TABLE tags ADD COLUMN is_default INTEGER NOT NULL DEFAULT 0;

CREATE TABLE task_team_prompts (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  project_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  PRIMARY KEY (task_rowid, team_id)
);
CREATE INDEX idx_task_team_prompts_team ON task_team_prompts (team_id);

-- +goose Down
DROP INDEX idx_task_team_prompts_team;
DROP TABLE task_team_prompts;

ALTER TABLE tags DROP COLUMN is_default;

CREATE TABLE prompts_old (
  id INTEGER PRIMARY KEY,
  scope TEXT NOT NULL CHECK (
    scope IN ('org', 'agent', 'project', 'agent_project')
  ),
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects (id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  version INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);
INSERT INTO prompts_old (id, scope, agent_id, project_id, content, version, created_at)
SELECT id, scope, agent_id, project_id, content, version, created_at
FROM prompts
WHERE scope IN ('org', 'agent', 'project', 'agent_project');
DROP TABLE prompts;
ALTER TABLE prompts_old RENAME TO prompts;
CREATE INDEX idx_prompts_key
  ON prompts (scope, agent_id, project_id, version DESC);
