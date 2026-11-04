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











DROP TABLE IF EXISTS tasks_new;
CREATE TABLE tasks_new (
  rowid INTEGER PRIMARY KEY,
  id TEXT NOT NULL UNIQUE,
  seq INTEGER NOT NULL DEFAULT 0,
  prompt TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  project_id TEXT REFERENCES projects (id),
  assignee_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  assignee_pinned INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL CHECK (state IN (
    'backlog', 'queued', 'running', 'input', 'settled', 'failed',
    'completed', 'cancelled'
  )),
  session_id TEXT NOT NULL DEFAULT '',
  session_agent_id INTEGER REFERENCES agents (id) ON DELETE SET NULL,
  org_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  agent_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  project_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  agent_project_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
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
  pending_prompt TEXT NOT NULL DEFAULT '',
  state_updated_at INTEGER NOT NULL DEFAULT 0,
  session_deleted_at INTEGER,
  size TEXT NOT NULL DEFAULT 'medium'
    CHECK (size IN ('small', 'medium', 'big', 'epic')),
  archived_at INTEGER,
  creator_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  participation TEXT NOT NULL DEFAULT 'project'
    CHECK (participation IN ('project', 'teams', 'assignee')),
  pending_prompt_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL
);
INSERT INTO tasks_new
  (rowid, id, seq, prompt, name, project_id, assignee_actor_id,
   assignee_pinned, priority, state, session_id, session_agent_id,
   org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
   error, queued_at, started_at, settled_at, created_at, updated_at,
   model_provider, model_id, thinking_level, interactive, workspace,
   auto_title, pending_prompt, state_updated_at, session_deleted_at, size,
   archived_at, creator_actor_id, participation, pending_prompt_actor_id)
  SELECT rowid, id, seq, prompt, name, project_id, assignee_actor_id,
         assignee_pinned, priority,
         CASE WHEN state = 'draft' THEN 'backlog' ELSE state END,
         session_id, session_agent_id,
         org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
         error, queued_at, started_at, settled_at, created_at, updated_at,
         model_provider, model_id, thinking_level, interactive, workspace,
         auto_title, pending_prompt, state_updated_at, session_deleted_at, size,
         archived_at, creator_actor_id, participation, pending_prompt_actor_id
  FROM tasks;
DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;
CREATE INDEX idx_tasks_archived ON tasks (archived_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_actor_id, state);
CREATE INDEX idx_tasks_creator ON tasks (creator_actor_id);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE UNIQUE INDEX idx_tasks_project_seq ON tasks (project_id, seq)
  WHERE project_id IS NOT NULL;
CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);

-- +goose StatementBegin
CREATE TRIGGER tasks_state_updated_at_on_insert
AFTER INSERT ON tasks
FOR EACH ROW
BEGIN
  UPDATE tasks SET state_updated_at = NEW.created_at WHERE rowid = NEW.rowid;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER tasks_state_updated_at_on_update
AFTER UPDATE OF state ON tasks
FOR EACH ROW
WHEN OLD.state <> NEW.state
BEGIN
  UPDATE tasks SET state_updated_at = NEW.updated_at WHERE rowid = NEW.rowid;
END;
-- +goose StatementEnd

-- +goose Down



DROP TABLE IF EXISTS tasks_old;
CREATE TABLE tasks_old (
  rowid INTEGER PRIMARY KEY,
  id TEXT NOT NULL UNIQUE,
  seq INTEGER NOT NULL DEFAULT 0,
  prompt TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  project_id TEXT REFERENCES projects (id),
  assignee_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  assignee_pinned INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL CHECK (state IN (
    'draft', 'backlog', 'queued', 'running', 'input', 'settled', 'failed',
    'completed', 'cancelled'
  )),
  session_id TEXT NOT NULL DEFAULT '',
  session_agent_id INTEGER REFERENCES agents (id) ON DELETE SET NULL,
  org_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  agent_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  project_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  agent_project_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
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
  pending_prompt TEXT NOT NULL DEFAULT '',
  state_updated_at INTEGER NOT NULL DEFAULT 0,
  session_deleted_at INTEGER,
  size TEXT NOT NULL DEFAULT 'medium'
    CHECK (size IN ('small', 'medium', 'big', 'epic')),
  archived_at INTEGER,
  creator_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  participation TEXT NOT NULL DEFAULT 'project'
    CHECK (participation IN ('project', 'teams', 'assignee')),
  pending_prompt_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL
);
INSERT INTO tasks_old
  (rowid, id, seq, prompt, name, project_id, assignee_actor_id,
   assignee_pinned, priority, state, session_id, session_agent_id,
   org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
   error, queued_at, started_at, settled_at, created_at, updated_at,
   model_provider, model_id, thinking_level, interactive, workspace,
   auto_title, pending_prompt, state_updated_at, session_deleted_at, size,
   archived_at, creator_actor_id, participation, pending_prompt_actor_id)
  SELECT rowid, id, seq, prompt, name, project_id, assignee_actor_id,
         assignee_pinned, priority, state, session_id, session_agent_id,
         org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
         error, queued_at, started_at, settled_at, created_at, updated_at,
         model_provider, model_id, thinking_level, interactive, workspace,
         auto_title, pending_prompt, state_updated_at, session_deleted_at, size,
         archived_at, creator_actor_id, participation, pending_prompt_actor_id
  FROM tasks;
DROP TABLE tasks;
ALTER TABLE tasks_old RENAME TO tasks;
CREATE INDEX idx_tasks_archived ON tasks (archived_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_actor_id, state);
CREATE INDEX idx_tasks_creator ON tasks (creator_actor_id);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE UNIQUE INDEX idx_tasks_project_seq ON tasks (project_id, seq)
  WHERE project_id IS NOT NULL;
CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);

-- +goose StatementBegin
CREATE TRIGGER tasks_state_updated_at_on_insert
AFTER INSERT ON tasks
FOR EACH ROW
BEGIN
  UPDATE tasks SET state_updated_at = NEW.created_at WHERE rowid = NEW.rowid;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER tasks_state_updated_at_on_update
AFTER UPDATE OF state ON tasks
FOR EACH ROW
WHEN OLD.state <> NEW.state
BEGIN
  UPDATE tasks SET state_updated_at = NEW.updated_at WHERE rowid = NEW.rowid;
END;
-- +goose StatementEnd
