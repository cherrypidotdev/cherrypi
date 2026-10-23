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
-- Reference tasks by an integer rowid internally.
--
-- `tasks.id` is the human-readable `<prefix>-<seq>` key and it *changes* when a
-- project's prefix is edited. Every child table used to store that mutable
-- string, so a rename had to rewrite the whole dependency graph, tag links,
-- published links and notifications in one fragile transaction. This migration
-- gives every task a stable, table-unique integer `rowid` and has the child
-- tables reference it instead; a prefix change then only touches `tasks.id`.
--
-- `number` is renamed to `seq`: the per-project sequence that only ever
-- increases, independent of the project's (mutable) prefix. The rendered `id`
-- stays as a denormalized, uniquely indexed column so the public string id
-- remains a cheap lookup; it is always `COALESCE(prefix, 'TASK') || '-' || seq`
-- and is the single column a prefix change rewrites.
--
-- SQLite cannot change a primary key in place, so `tasks` is rebuilt. The old
-- rowids are captured first and copied verbatim into the new table, so the
-- child tables can be remapped by the old `id` without another lookup table.

CREATE TEMP TABLE task_rowid_map (old_id TEXT PRIMARY KEY, rowid INTEGER NOT NULL);

INSERT INTO task_rowid_map (old_id, rowid) SELECT id, rowid FROM tasks;

CREATE TABLE tasks_new (
  rowid INTEGER PRIMARY KEY,
  id TEXT NOT NULL UNIQUE,
  seq INTEGER NOT NULL DEFAULT 0,
  prompt TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  project_id TEXT REFERENCES projects (id),
  assignee_agent_id TEXT REFERENCES agents (id) ON DELETE SET NULL,
  assignee_pinned INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL CHECK (state IN (
    'draft', 'queued', 'running', 'input', 'settled', 'failed', 'review', 'completed', 'cancelled'
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
  size TEXT NOT NULL DEFAULT 'medium' CHECK (size IN ('small', 'medium', 'big')),
  blocked INTEGER NOT NULL DEFAULT 0 CHECK (blocked IN (0, 1)),
  pending_prompt TEXT NOT NULL DEFAULT ''
);

INSERT INTO tasks_new (
  rowid, id, seq, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority, state,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, queued_at, started_at, settled_at, created_at, updated_at,
  model_provider, model_id, thinking_level, interactive, workspace, auto_title,
  size, blocked, pending_prompt
)
SELECT
  rowid, id, number, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority, state,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, queued_at, started_at, settled_at, created_at, updated_at,
  model_provider, model_id, thinking_level, interactive, workspace, auto_title,
  size, blocked, pending_prompt
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_agent_id, state);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE UNIQUE INDEX idx_tasks_project_seq ON tasks (project_id, seq)
WHERE project_id IS NOT NULL;

-- Tag links key on the task rowid.
CREATE TABLE task_tags_new (
  task_rowid INTEGER NOT NULL,
  tag_id TEXT NOT NULL REFERENCES tags (id),
  PRIMARY KEY (task_rowid, tag_id)
);

INSERT INTO task_tags_new (task_rowid, tag_id)
SELECT m.rowid, tt.tag_id FROM task_tags tt
JOIN task_rowid_map m ON m.old_id = tt.task_id;

DROP TABLE task_tags;
ALTER TABLE task_tags_new RENAME TO task_tags;
CREATE INDEX idx_task_tags_tag ON task_tags (tag_id);

-- Published links key on the task rowid.
CREATE TABLE task_links_new (
  task_rowid INTEGER NOT NULL,
  url TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  position INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (task_rowid, url)
);

INSERT INTO task_links_new (task_rowid, url, label, position)
SELECT m.rowid, l.url, l.label, l.position FROM task_links l
JOIN task_rowid_map m ON m.old_id = l.task_id;

DROP TABLE task_links;
ALTER TABLE task_links_new RENAME TO task_links;
CREATE INDEX idx_task_links_task ON task_links (task_rowid, position);

-- Dependency edges key on both task rowids.
CREATE TABLE task_dependencies_new (
  task_rowid INTEGER NOT NULL,
  depends_on_task_rowid INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (task_rowid, depends_on_task_rowid),
  CHECK (task_rowid <> depends_on_task_rowid)
);

INSERT INTO task_dependencies_new (task_rowid, depends_on_task_rowid, created_at)
SELECT t.rowid, d.rowid, td.created_at FROM task_dependencies td
JOIN task_rowid_map t ON t.old_id = td.task_id
JOIN task_rowid_map d ON d.old_id = td.depends_on_task_id;

DROP TABLE task_dependencies;
ALTER TABLE task_dependencies_new RENAME TO task_dependencies;
CREATE INDEX idx_task_dependencies_on ON task_dependencies (depends_on_task_rowid);

-- Notifications point at a task when they come from one (attention signals);
-- `NULL` replaces the old empty string for notification kinds that have none.
CREATE TABLE notifications_new (
  id INTEGER PRIMARY KEY,
  agent_id TEXT REFERENCES agents (id) ON DELETE CASCADE,
  task_rowid INTEGER,
  kind TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'info',
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  read_at INTEGER
);

INSERT INTO notifications_new (
  id, agent_id, task_rowid, kind, severity, title, body, created_at, read_at
)
SELECT n.id, n.agent_id, m.rowid, n.kind, n.severity, n.title, n.body, n.created_at, n.read_at
FROM notifications n
LEFT JOIN task_rowid_map m ON m.old_id = n.task_id;

DROP TABLE notifications;
ALTER TABLE notifications_new RENAME TO notifications;
CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);
CREATE INDEX idx_notifications_task_rowid ON notifications (task_rowid);

DROP TABLE task_rowid_map;

-- +goose Down
-- Rebuild the pre-rowid layout: `tasks.id` is the primary key again, `seq`
-- folds back to `number`, and every child table returns to the mutable string
-- task id. `rowid` values are dropped, so ids are the only mapping available.
CREATE TEMP TABLE task_id_map (rowid INTEGER PRIMARY KEY, old_id TEXT NOT NULL);

INSERT INTO task_id_map (rowid, old_id) SELECT rowid, id FROM tasks;

CREATE TABLE tasks_old (
  id TEXT PRIMARY KEY,
  prompt TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  project_id TEXT REFERENCES projects (id),
  assignee_agent_id TEXT REFERENCES agents (id) ON DELETE SET NULL,
  assignee_pinned INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL CHECK (state IN (
    'draft', 'queued', 'running', 'input', 'settled', 'failed', 'review', 'completed', 'cancelled'
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

INSERT INTO tasks_old (
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority, state,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, queued_at, started_at, settled_at, created_at, updated_at,
  model_provider, model_id, thinking_level, interactive, workspace, auto_title,
  number, size, blocked, pending_prompt
)
SELECT
  id, prompt, name, project_id, assignee_agent_id, assignee_pinned, priority, state,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, queued_at, started_at, settled_at, created_at, updated_at,
  model_provider, model_id, thinking_level, interactive, workspace, auto_title,
  seq, size, blocked, pending_prompt
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_old RENAME TO tasks;

CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_agent_id, state);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE UNIQUE INDEX idx_tasks_project_number ON tasks (project_id, number)
WHERE project_id IS NOT NULL;

CREATE TABLE task_tags_old (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags (id),
  PRIMARY KEY (task_id, tag_id)
);

INSERT INTO task_tags_old (task_id, tag_id)
SELECT m.old_id, tt.tag_id FROM task_tags tt
JOIN task_id_map m ON m.rowid = tt.task_rowid;

DROP TABLE task_tags;
ALTER TABLE task_tags_old RENAME TO task_tags;
CREATE INDEX idx_task_tags_tag ON task_tags (tag_id);

CREATE TABLE task_links_old (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  url TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  position INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (task_id, url)
);

INSERT INTO task_links_old (task_id, url, label, position)
SELECT m.old_id, l.url, l.label, l.position FROM task_links l
JOIN task_id_map m ON m.rowid = l.task_rowid;

DROP TABLE task_links;
ALTER TABLE task_links_old RENAME TO task_links;
CREATE INDEX idx_task_links_task ON task_links (task_id, position);

CREATE TABLE task_dependencies_old (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  depends_on_task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (task_id, depends_on_task_id),
  CHECK (task_id <> depends_on_task_id)
);

INSERT INTO task_dependencies_old (task_id, depends_on_task_id, created_at)
SELECT t.old_id, d.old_id, td.created_at FROM task_dependencies td
JOIN task_id_map t ON t.rowid = td.task_rowid
JOIN task_id_map d ON d.rowid = td.depends_on_task_rowid;

DROP TABLE task_dependencies;
ALTER TABLE task_dependencies_old RENAME TO task_dependencies;
CREATE INDEX idx_task_dependencies_on ON task_dependencies (depends_on_task_id);

CREATE TABLE notifications_old (
  id INTEGER PRIMARY KEY,
  agent_id TEXT REFERENCES agents (id) ON DELETE CASCADE,
  task_id TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'info',
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  read_at INTEGER
);

INSERT INTO notifications_old (
  id, agent_id, task_id, kind, severity, title, body, created_at, read_at
)
SELECT
  n.id, n.agent_id, COALESCE(m.old_id, ''), n.kind, n.severity, n.title, n.body,
  n.created_at, n.read_at
FROM notifications n
LEFT JOIN task_id_map m ON m.rowid = n.task_rowid;

DROP TABLE notifications;
ALTER TABLE notifications_old RENAME TO notifications;
CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);

DROP TABLE task_id_map;
