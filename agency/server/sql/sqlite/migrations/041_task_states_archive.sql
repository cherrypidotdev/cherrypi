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
-- Rework the task states and the reference edge type, and add per-user
-- settings.
--
--  * `review` is removed: a parked-for-review task is now just `settled`.
--  * `backlog` joins the lifecycle between draft and queued. Queuing is only
--    allowed from backlog (or at creation), and de-queuing returns a task to
--    backlog.
--  * `blocked` stops being a stored draft flag: a queued task with incomplete
--    blocking references is reported as blocked (a computed property), so the
--    column is dropped. Rows that were blocked drafts migrate to `queued`.
--  * per-task archiving returns: `archived_at` hides a task and cleanup is
--    triggered on archive. It is a flag, not a state.
--  * references gain a `kind` (`blocking`, `informational`, `supersedes`) in
--    place of the `blocking` boolean.
--
-- SQLite cannot alter a CHECK, so `tasks` is rebuilt; rowids are preserved
-- verbatim because every child table keys on them.
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
    'draft', 'backlog', 'queued', 'running', 'input', 'settled', 'failed',
    'completed', 'cancelled'
  )),
  session_id TEXT NOT NULL DEFAULT '',
  session_agent_id INTEGER REFERENCES agents (id) ON DELETE SET NULL,
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
  pending_prompt TEXT NOT NULL DEFAULT '',
  state_updated_at INTEGER NOT NULL DEFAULT 0,
  session_deleted_at INTEGER,
  size TEXT NOT NULL DEFAULT 'medium'
    CHECK (size IN ('small', 'medium', 'big', 'epic')),
  archived_at INTEGER
);
INSERT INTO tasks_new (
  rowid, id, seq, prompt, name, project_id, assignee_actor_id, assignee_pinned,
  priority, state, session_id, session_agent_id, org_prompt_id, agent_prompt_id,
  project_prompt_id, agent_project_prompt_id, error, queued_at, started_at,
  settled_at, created_at, updated_at, model_provider, model_id, thinking_level,
  interactive, workspace, auto_title, pending_prompt, state_updated_at,
  session_deleted_at, size, archived_at
)
SELECT
  rowid, id, seq, prompt, name, project_id, assignee_actor_id, assignee_pinned,
  priority,
  -- A stored `review` maps to settled; a blocked draft was waiting on a
  -- blocking reference, which is now a queued (blocked) task.
  CASE
    WHEN state = 'review' THEN 'settled'
    WHEN state = 'draft' AND blocked = 1 THEN 'queued'
    ELSE state
  END,
  session_id, session_agent_id, org_prompt_id, agent_prompt_id,
  project_prompt_id, agent_project_prompt_id, error, queued_at, started_at,
  settled_at, created_at, updated_at, model_provider, model_id, thinking_level,
  interactive, workspace, auto_title, pending_prompt, state_updated_at,
  session_deleted_at, size, NULL
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_actor_id, state);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE INDEX idx_tasks_archived ON tasks (archived_at);
CREATE UNIQUE INDEX idx_tasks_project_seq ON tasks (project_id, seq)
  WHERE project_id IS NOT NULL;

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

-- References: the boolean `blocking` becomes a three-way `kind`. Every
-- existing blocking edge stays blocking; every informational edge stays
-- informational.
ALTER TABLE task_references ADD COLUMN kind TEXT NOT NULL DEFAULT 'blocking'
  CHECK (kind IN ('blocking', 'informational', 'supersedes'));
UPDATE task_references SET kind = 'informational' WHERE blocking = 0;
ALTER TABLE task_references DROP COLUMN blocking;

-- Per-user key/value settings, e.g. whether to show the first-archive notice.
CREATE TABLE actor_settings (
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  value TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (actor_id, key)
);

-- +goose Down
-- Reverse to the pre-rework schema: review returns (pending_prompt is kept),
-- backlog folds to draft, blocked drafts come back, archive is dropped, and
-- references return to a boolean `blocking`.
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
    'draft', 'queued', 'running', 'input', 'settled', 'failed', 'review',
    'completed', 'cancelled'
  )),
  session_id TEXT NOT NULL DEFAULT '',
  session_agent_id INTEGER REFERENCES agents (id) ON DELETE SET NULL,
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
  blocked INTEGER NOT NULL DEFAULT 0 CHECK (blocked IN (0, 1)),
  pending_prompt TEXT NOT NULL DEFAULT '',
  state_updated_at INTEGER NOT NULL DEFAULT 0,
  session_deleted_at INTEGER,
  size TEXT NOT NULL DEFAULT 'medium'
    CHECK (size IN ('small', 'medium', 'big', 'epic'))
);
INSERT INTO tasks_old (
  rowid, id, seq, prompt, name, project_id, assignee_actor_id, assignee_pinned,
  priority, state, session_id, session_agent_id, org_prompt_id, agent_prompt_id,
  project_prompt_id, agent_project_prompt_id, error, queued_at, started_at,
  settled_at, created_at, updated_at, model_provider, model_id, thinking_level,
  interactive, workspace, auto_title, blocked, pending_prompt, state_updated_at,
  session_deleted_at, size
)
SELECT
  rowid, id, seq, prompt, name, project_id, assignee_actor_id, assignee_pinned,
  priority,
  CASE WHEN state = 'backlog' THEN 'draft' ELSE state END,
  session_id, session_agent_id, org_prompt_id, agent_prompt_id,
  project_prompt_id, agent_project_prompt_id, error, queued_at, started_at,
  settled_at, created_at, updated_at, model_provider, model_id, thinking_level,
  interactive, workspace, auto_title, 0, pending_prompt, state_updated_at,
  session_deleted_at, size
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_old RENAME TO tasks;

CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_actor_id, state);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE UNIQUE INDEX idx_tasks_project_seq ON tasks (project_id, seq)
  WHERE project_id IS NOT NULL;

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
AFTER UPDATE OF state, blocked ON tasks
FOR EACH ROW
WHEN OLD.state <> NEW.state OR OLD.blocked <> NEW.blocked
BEGIN
  UPDATE tasks SET state_updated_at = NEW.updated_at WHERE rowid = NEW.rowid;
END;
-- +goose StatementEnd

ALTER TABLE task_references ADD COLUMN blocking INTEGER NOT NULL DEFAULT 1
  CHECK (blocking IN (0, 1));
UPDATE task_references SET blocking = CASE WHEN kind = 'blocking' THEN 1 ELSE 0 END;
ALTER TABLE task_references DROP COLUMN kind;

DROP TABLE actor_settings;
