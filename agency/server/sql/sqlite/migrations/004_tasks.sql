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
-- Org-wide task tags. Soft-deleted via archived_at so completed tasks keep
-- rendering the tag; the name is unique case-insensitively.
CREATE TABLE tags (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  color TEXT NOT NULL DEFAULT '',
  archived_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_tags_name ON tags (lower(name));

-- The task work item. It absorbs the old session_context: a task owns its
-- prompt and is bound to its agent (assignee) only when it starts, with the
-- Pi session id recorded then. Prompt versions are pinned at first run start
-- and never change afterwards.
CREATE TABLE tasks (
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
  updated_at INTEGER NOT NULL
);

CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_agent_id, state);
CREATE INDEX idx_tasks_project ON tasks (project_id);

CREATE TABLE task_tags (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags (id),
  PRIMARY KEY (task_id, tag_id)
);

CREATE INDEX idx_task_tags_tag ON task_tags (tag_id);

-- Per-agent concurrency limits. The overall totals live in one row per agent;
-- the tag and project limits are sparse overrides.
CREATE TABLE agent_task_limits (
  agent_id TEXT PRIMARY KEY REFERENCES agents (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL DEFAULT 1,
  max_concurrent_per_project INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE agent_tag_limits (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, tag_id)
);

CREATE TABLE agent_project_limits (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, project_id)
);

CREATE TABLE agent_project_tag_limits (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, project_id, tag_id)
);

-- Backfill: every session_context row becomes a completed, slot-free task that
-- keeps its project, agent, Pi session and pinned prompt versions. The prompt
-- and name are enriched lazily from the agent's task listing.
INSERT INTO tasks (
  id, prompt, name, project_id, assignee_agent_id, state, session_id,
  org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  created_at, updated_at, completed_at
)
SELECT
  lower(hex(randomblob(16))), '', '', project_id, agent_id, 'completed', session_id,
  org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  created_at, created_at, created_at
FROM session_context;

DROP TABLE session_context;

-- +goose Down
CREATE TABLE session_context (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  session_id TEXT NOT NULL,
  project_id TEXT REFERENCES projects (id),
  org_prompt_id INTEGER REFERENCES prompts (id),
  agent_prompt_id INTEGER REFERENCES prompts (id),
  project_prompt_id INTEGER REFERENCES prompts (id),
  agent_project_prompt_id INTEGER REFERENCES prompts (id),
  created_at INTEGER NOT NULL,
  PRIMARY KEY (agent_id, session_id)
);

CREATE INDEX idx_session_context_project_id ON session_context (project_id);

DROP TABLE agent_project_tag_limits;
DROP TABLE agent_project_limits;
DROP TABLE agent_tag_limits;
DROP TABLE agent_task_limits;
DROP TABLE task_tags;
DROP TABLE tasks;
DROP TABLE tags;
