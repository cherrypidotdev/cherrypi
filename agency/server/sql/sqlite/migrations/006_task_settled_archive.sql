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







CREATE TABLE tasks_new (
  id TEXT PRIMARY KEY,
  prompt TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  project_id TEXT REFERENCES projects (id),
  assignee_agent_id TEXT REFERENCES agents (id) ON DELETE SET NULL,
  assignee_pinned INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL CHECK (state IN (
    'draft', 'queued', 'running', 'input', 'settled', 'failed', 'completed', 'cancelled'
  )),
  session_id TEXT NOT NULL DEFAULT '',
  org_prompt_id INTEGER REFERENCES prompts (id),
  agent_prompt_id INTEGER REFERENCES prompts (id),
  project_prompt_id INTEGER REFERENCES prompts (id),
  agent_project_prompt_id INTEGER REFERENCES prompts (id),
  error TEXT NOT NULL DEFAULT '',
  queued_at INTEGER,
  started_at INTEGER,
  settled_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  model_provider TEXT NOT NULL DEFAULT '',
  model_id TEXT NOT NULL DEFAULT '',
  thinking_level TEXT NOT NULL DEFAULT '',
  interactive INTEGER NOT NULL DEFAULT 0,
  workspace TEXT NOT NULL DEFAULT '',
  auto_title INTEGER NOT NULL DEFAULT 0,
  archived_at INTEGER
);

INSERT INTO tasks_new (
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority, state,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, queued_at, started_at, settled_at, created_at, updated_at,
  model_provider, model_id, thinking_level, interactive, workspace, auto_title, archived_at
)
SELECT
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority,
  CASE
    WHEN state = 'completed' THEN 'settled'
    WHEN state = 'cancelled' THEN 'settled'
    ELSE state
  END,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, queued_at, started_at, completed_at, created_at, updated_at,
  model_provider, model_id, thinking_level, interactive, workspace, auto_title, NULL
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_agent_id, state);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE INDEX idx_tasks_archived ON tasks (archived_at);

-- +goose Down


CREATE TABLE tasks_old (
  id TEXT PRIMARY KEY,
  prompt TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  project_id TEXT REFERENCES projects (id),
  assignee_agent_id TEXT REFERENCES agents (id) ON DELETE SET NULL,
  assignee_pinned INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL CHECK (state IN ('draft', 'queued', 'running', 'completed', 'failed')),
  session_id TEXT NOT NULL DEFAULT '',
  org_prompt_id INTEGER REFERENCES prompts (id),
  agent_prompt_id INTEGER REFERENCES prompts (id),
  project_prompt_id INTEGER REFERENCES prompts (id),
  agent_project_prompt_id INTEGER REFERENCES prompts (id),
  error TEXT NOT NULL DEFAULT '',
  queued_at INTEGER,
  started_at INTEGER,
  completed_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  model_provider TEXT NOT NULL DEFAULT '',
  model_id TEXT NOT NULL DEFAULT '',
  thinking_level TEXT NOT NULL DEFAULT '',
  interactive INTEGER NOT NULL DEFAULT 0,
  workspace TEXT NOT NULL DEFAULT '',
  auto_title INTEGER NOT NULL DEFAULT 0
);

INSERT INTO tasks_old (
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority, state,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, queued_at, started_at, completed_at, created_at, updated_at,
  model_provider, model_id, thinking_level, interactive, workspace, auto_title
)
SELECT
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority,
  CASE
    WHEN state IN ('settled', 'cancelled') THEN 'completed'
    WHEN state = 'input' THEN 'running'
    ELSE state
  END,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, queued_at, started_at, settled_at, created_at, updated_at,
  model_provider, model_id, thinking_level, interactive, workspace, auto_title
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_old RENAME TO tasks;

CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_agent_id, state);
CREATE INDEX idx_tasks_project ON tasks (project_id);
