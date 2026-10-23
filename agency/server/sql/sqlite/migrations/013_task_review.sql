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
-- Adds the `review` state, drops task archiving, and adds `pending_prompt`.
--
-- `review` parks a task whose run finished but whose result is not yet
-- accepted. It is slot-free (unlike `settled`/`failed`), so the agent can take
-- other work; a follow-up prompt re-queues it. `pending_prompt` carries a
-- follow-up sent to a resting task (review/completed/cancelled): the agency
-- stores it here and the scheduler delivers it when the task is next started.
-- SQLite cannot alter a CHECK constraint, so the table is rebuilt (which also
-- drops the retired `archived_at` column). A leftover table from an earlier
-- failed attempt is dropped first so the migration can be retried.
DROP TABLE IF EXISTS tasks_new;
CREATE TABLE tasks_new (
  id TEXT PRIMARY KEY,
  prompt TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  project_id TEXT REFERENCES projects (id),
  assignee_agent_id TEXT REFERENCES agents (id) ON DELETE SET NULL,
  assignee_pinned INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL CHECK (state IN (
    'draft', 'queued', 'running', 'input', 'settled', 'failed', 'review',
    'completed', 'cancelled'
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
  number INTEGER NOT NULL DEFAULT 0,
  size TEXT NOT NULL DEFAULT 'medium' CHECK (size IN ('small', 'medium', 'big')),
  blocked INTEGER NOT NULL DEFAULT 0 CHECK (blocked IN (0, 1)),
  pending_prompt TEXT NOT NULL DEFAULT ''
);

INSERT INTO tasks_new (
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority,
  state, session_id, org_prompt_id, agent_prompt_id, project_prompt_id,
  agent_project_prompt_id, error, queued_at, started_at, settled_at, created_at,
  updated_at, model_provider, model_id, thinking_level, interactive, workspace,
  auto_title, number, size, blocked
)
SELECT
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority,
  state, session_id, org_prompt_id, agent_prompt_id, project_prompt_id,
  agent_project_prompt_id, error, queued_at, started_at, settled_at, created_at,
  updated_at, model_provider, model_id, thinking_level, interactive, workspace,
  auto_title, number,
  -- A database written under an earlier revision can hold a `size` outside the
  -- final set (or a legacy empty default), and `blocked` outside 0/1.
  -- Normalize before the rebuild so it can never fail on old rows.
  CASE WHEN size IN ('small', 'medium', 'big') THEN size ELSE 'medium' END,
  CASE WHEN blocked IN (0, 1) THEN blocked ELSE 0 END
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_agent_id, state);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE UNIQUE INDEX idx_tasks_project_number ON tasks (project_id, number)
WHERE project_id IS NOT NULL;

-- +goose Down
-- Restores the pre-013 schema: `review` folds back to `settled`,
-- `pending_prompt` is dropped, and `archived_at` returns (all tasks active).
DROP TABLE IF EXISTS tasks_old;
CREATE TABLE tasks_old (
  id TEXT PRIMARY KEY,
  prompt TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  project_id TEXT REFERENCES projects (id),
  assignee_agent_id TEXT REFERENCES agents (id) ON DELETE SET NULL,
  assignee_pinned INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL CHECK (state IN (
    'draft', 'queued', 'running', 'input', 'settled', 'failed', 'completed',
    'cancelled'
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
  archived_at INTEGER,
  number INTEGER NOT NULL DEFAULT 0,
  size TEXT NOT NULL DEFAULT 'medium' CHECK (size IN ('small', 'medium', 'big')),
  blocked INTEGER NOT NULL DEFAULT 0 CHECK (blocked IN (0, 1))
);

INSERT INTO tasks_old (
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority,
  state, session_id, org_prompt_id, agent_prompt_id, project_prompt_id,
  agent_project_prompt_id, error, queued_at, started_at, settled_at, created_at,
  updated_at, model_provider, model_id, thinking_level, interactive, workspace,
  auto_title, archived_at, number, size, blocked
)
SELECT
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority,
  CASE WHEN state = 'review' THEN 'settled' ELSE state END,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id,
  agent_project_prompt_id, error, queued_at, started_at, settled_at, created_at,
  updated_at, model_provider, model_id, thinking_level, interactive, workspace,
  auto_title, NULL, number,
  CASE WHEN size IN ('small', 'medium', 'big') THEN size ELSE 'medium' END,
  CASE WHEN blocked IN (0, 1) THEN blocked ELSE 0 END
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_old RENAME TO tasks;

CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_agent_id, state);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE INDEX idx_tasks_archived ON tasks (archived_at);
CREATE UNIQUE INDEX idx_tasks_project_number ON tasks (project_id, number)
WHERE project_id IS NOT NULL;
