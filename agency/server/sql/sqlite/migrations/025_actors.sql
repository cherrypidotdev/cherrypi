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











CREATE TABLE actors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL CHECK (kind IN ('user', 'agent')),
  username TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL,
  
  
  password_hash BLOB,
  password_updated_at INTEGER,
  totp_secret BLOB,
  totp_enabled_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_actors_username ON actors (username);
CREATE INDEX idx_actors_kind ON actors (kind);

INSERT INTO actors (
  kind, username, name, description, picture_attachment_id, created_at, updated_at
)
SELECT
  'agent', username, display_name, description, picture_attachment_id,
  created_at, updated_at
FROM agents;

CREATE TEMP TABLE agent_actor_map (old_id TEXT PRIMARY KEY, new_actor_id INTEGER NOT NULL);
INSERT INTO agent_actor_map (old_id, new_actor_id)
SELECT a.id, act.id
FROM agents a
JOIN actors act ON act.kind = 'agent' AND act.username = a.username;


CREATE TABLE agents_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NOT NULL UNIQUE REFERENCES actors (id) ON DELETE CASCADE,
  last_seen_at INTEGER
);
INSERT INTO agents_new (actor_id, last_seen_at)
SELECT m.new_actor_id, a.last_seen_at
FROM agents a JOIN agent_actor_map m ON m.old_id = a.id;

CREATE TEMP TABLE agent_id_map (old_id TEXT PRIMARY KEY, new_id INTEGER NOT NULL);
INSERT INTO agent_id_map (old_id, new_id)
SELECT m.old_id, g.id
FROM agent_actor_map m JOIN agents_new g ON g.actor_id = m.new_actor_id;

DROP TABLE agents;
ALTER TABLE agents_new RENAME TO agents;


CREATE TABLE agent_credentials_new (
  id INTEGER PRIMARY KEY,
  agent_id INTEGER NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  last_used_at INTEGER
);
INSERT INTO agent_credentials_new (
  id, agent_id, username, password_hash, created_at, last_used_at
)
SELECT c.id, m.new_id, c.username, c.password_hash, c.created_at, c.last_used_at
FROM agent_credentials c JOIN agent_id_map m ON m.old_id = c.agent_id;
DROP TABLE agent_credentials;
ALTER TABLE agent_credentials_new RENAME TO agent_credentials;
CREATE INDEX idx_agent_credentials_agent_id ON agent_credentials (agent_id);


CREATE TABLE agent_task_limits_new (
  agent_id INTEGER PRIMARY KEY REFERENCES agents (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL DEFAULT 1,
  max_concurrent_per_project INTEGER NOT NULL DEFAULT 1
);
INSERT INTO agent_task_limits_new (
  agent_id, max_concurrent, max_concurrent_per_project
)
SELECT m.new_id, l.max_concurrent, l.max_concurrent_per_project
FROM agent_task_limits l JOIN agent_id_map m ON m.old_id = l.agent_id;
DROP TABLE agent_task_limits;
ALTER TABLE agent_task_limits_new RENAME TO agent_task_limits;

CREATE TABLE agent_tag_limits_new (
  agent_id INTEGER NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  tag_name TEXT NOT NULL,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, tag_name)
);
INSERT INTO agent_tag_limits_new (agent_id, tag_name, max_concurrent)
SELECT m.new_id, l.tag_name, l.max_concurrent
FROM agent_tag_limits l JOIN agent_id_map m ON m.old_id = l.agent_id;
DROP TABLE agent_tag_limits;
ALTER TABLE agent_tag_limits_new RENAME TO agent_tag_limits;

CREATE TABLE agent_project_limits_new (
  agent_id INTEGER NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, project_id)
);
INSERT INTO agent_project_limits_new (agent_id, project_id, max_concurrent)
SELECT m.new_id, l.project_id, l.max_concurrent
FROM agent_project_limits l JOIN agent_id_map m ON m.old_id = l.agent_id;
DROP TABLE agent_project_limits;
ALTER TABLE agent_project_limits_new RENAME TO agent_project_limits;

CREATE TABLE agent_project_tag_limits_new (
  agent_id INTEGER NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, project_id, tag_id)
);
INSERT INTO agent_project_tag_limits_new (
  agent_id, project_id, tag_id, max_concurrent
)
SELECT m.new_id, l.project_id, l.tag_id, l.max_concurrent
FROM agent_project_tag_limits l JOIN agent_id_map m ON m.old_id = l.agent_id;
DROP TABLE agent_project_tag_limits;
ALTER TABLE agent_project_tag_limits_new RENAME TO agent_project_tag_limits;


CREATE TABLE project_agents_new (
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  agent_id INTEGER NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  can_list_tasks INTEGER NOT NULL DEFAULT 1,
  can_create_tasks INTEGER NOT NULL DEFAULT 0,
  can_update_tasks INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (project_id, agent_id)
);
INSERT INTO project_agents_new (
  project_id, agent_id, created_at,
  can_list_tasks, can_create_tasks, can_update_tasks
)
SELECT
  pa.project_id, m.new_id, pa.created_at,
  pa.can_list_tasks, pa.can_create_tasks, pa.can_update_tasks
FROM project_agents pa JOIN agent_id_map m ON m.old_id = pa.agent_id;
DROP TABLE project_agents;
ALTER TABLE project_agents_new RENAME TO project_agents;
CREATE INDEX idx_project_agents_agent_id ON project_agents (agent_id);


CREATE TABLE prompts_new (
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
INSERT INTO prompts_new (id, scope, agent_id, project_id, content, version, created_at)
SELECT p.id, p.scope, m.new_id, p.project_id, p.content, p.version, p.created_at
FROM prompts p LEFT JOIN agent_id_map m ON m.old_id = p.agent_id;
DROP TABLE prompts;
ALTER TABLE prompts_new RENAME TO prompts;
CREATE INDEX idx_prompts_key ON prompts (scope, agent_id, project_id, version DESC);




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
  session_deleted_at INTEGER
);
INSERT INTO tasks_new (
  rowid, id, seq, prompt, name, project_id, assignee_actor_id, assignee_pinned,
  priority, state, session_id, session_agent_id, org_prompt_id, agent_prompt_id,
  project_prompt_id, agent_project_prompt_id, error, queued_at, started_at,
  settled_at, created_at, updated_at, model_provider, model_id, thinking_level,
  interactive, workspace, auto_title, blocked, pending_prompt,
  state_updated_at, session_deleted_at
)
SELECT
  t.rowid, t.id, t.seq, t.prompt, t.name, t.project_id, m.new_actor_id,
  t.assignee_pinned, t.priority, t.state, t.session_id,
  (SELECT m2.new_id FROM agent_id_map m2 WHERE m2.old_id = t.session_agent_id),
  t.org_prompt_id,
  t.agent_prompt_id, t.project_prompt_id, t.agent_project_prompt_id, t.error,
  t.queued_at, t.started_at, t.settled_at, t.created_at, t.updated_at,
  t.model_provider, t.model_id, t.thinking_level, t.interactive, t.workspace,
  t.auto_title, t.blocked, t.pending_prompt, t.state_updated_at,
  t.session_deleted_at
FROM tasks t LEFT JOIN agent_actor_map m ON m.old_id = t.assignee_agent_id;
DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;
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


CREATE TABLE notifications_new (
  id INTEGER PRIMARY KEY,
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
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
SELECT n.id, m.new_id, n.task_rowid, n.kind, n.severity, n.title, n.body,
       n.created_at, n.read_at
FROM notifications n LEFT JOIN agent_id_map m ON m.old_id = n.agent_id;
DROP TABLE notifications;
ALTER TABLE notifications_new RENAME TO notifications;
CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);
CREATE INDEX idx_notifications_task_rowid ON notifications (task_rowid);

CREATE TABLE webhooks_new (
  id INTEGER PRIMARY KEY,
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
  url TEXT NOT NULL,
  events TEXT NOT NULL,
  secret TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
INSERT INTO webhooks_new (id, agent_id, url, events, secret, created_at, updated_at)
SELECT w.id, m.new_id, w.url, w.events, w.secret, w.created_at, w.updated_at
FROM webhooks w LEFT JOIN agent_id_map m ON m.old_id = w.agent_id;
DROP TABLE webhooks;
ALTER TABLE webhooks_new RENAME TO webhooks;




CREATE TABLE agent_usage_new (
  id INTEGER PRIMARY KEY,
  agent_id INTEGER REFERENCES agents (id) ON DELETE SET NULL,
  session_id TEXT NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cost REAL NOT NULL DEFAULT 0,
  active_ms INTEGER NOT NULL DEFAULT 0,
  at INTEGER NOT NULL
);
INSERT INTO agent_usage_new (
  id, agent_id, session_id, model, input_tokens, output_tokens, cost, active_ms, at
)
SELECT u.id, m.new_id, u.session_id, u.model, u.input_tokens, u.output_tokens,
       u.cost, u.active_ms, u.at
FROM agent_usage u LEFT JOIN agent_id_map m ON m.old_id = u.agent_id;
DROP TABLE agent_usage;
ALTER TABLE agent_usage_new RENAME TO agent_usage;
CREATE INDEX idx_agent_usage_agent_id ON agent_usage (agent_id);
CREATE INDEX idx_agent_usage_session_id ON agent_usage (session_id);

CREATE TABLE agent_logs_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id INTEGER REFERENCES agents (id) ON DELETE SET NULL,
  at INTEGER NOT NULL,
  level INTEGER NOT NULL DEFAULT 1,
  message TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  session_id TEXT NOT NULL DEFAULT '',
  attrs_json TEXT NOT NULL DEFAULT ''
);
INSERT INTO agent_logs_new (
  id, agent_id, at, level, message, task_id, session_id, attrs_json
)
SELECT l.id, m.new_id, l.at, l.level, l.message, l.task_id, l.session_id, l.attrs_json
FROM agent_logs l LEFT JOIN agent_id_map m ON m.old_id = l.agent_id;
DROP TABLE agent_logs;
ALTER TABLE agent_logs_new RENAME TO agent_logs;
CREATE INDEX idx_agent_logs_agent ON agent_logs (agent_id, id DESC);

DROP TABLE agent_id_map;
DROP TABLE agent_actor_map;

-- +goose Down


CREATE TABLE agents_old (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  last_seen_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL
);
INSERT INTO agents_old (
  id, username, display_name, description, last_seen_at, created_at, updated_at,
  picture_attachment_id
)
SELECT
  CAST(g.id AS TEXT), a.username, a.name, a.description, g.last_seen_at,
  a.created_at, a.updated_at, a.picture_attachment_id
FROM agents g JOIN actors a ON a.id = g.actor_id;

CREATE TEMP TABLE agent_down_map (
  new_id INTEGER PRIMARY KEY,
  old_id TEXT NOT NULL,
  actor_id INTEGER NOT NULL
);
INSERT INTO agent_down_map (new_id, old_id, actor_id)
SELECT g.id, CAST(g.id AS TEXT), g.actor_id FROM agents g;

CREATE TABLE agent_credentials_old (
  id INTEGER PRIMARY KEY,
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  last_used_at INTEGER
);
INSERT INTO agent_credentials_old (
  id, agent_id, username, password_hash, created_at, last_used_at
)
SELECT c.id, m.old_id, c.username, c.password_hash, c.created_at, c.last_used_at
FROM agent_credentials c JOIN agent_down_map m ON m.new_id = c.agent_id;
DROP TABLE agent_credentials;
ALTER TABLE agent_credentials_old RENAME TO agent_credentials;
CREATE INDEX idx_agent_credentials_agent_id ON agent_credentials (agent_id);

CREATE TABLE agent_task_limits_old (
  agent_id TEXT PRIMARY KEY REFERENCES agents (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL DEFAULT 1,
  max_concurrent_per_project INTEGER NOT NULL DEFAULT 1
);
INSERT INTO agent_task_limits_old (agent_id, max_concurrent, max_concurrent_per_project)
SELECT m.old_id, l.max_concurrent, l.max_concurrent_per_project
FROM agent_task_limits l JOIN agent_down_map m ON m.new_id = l.agent_id;
DROP TABLE agent_task_limits;
ALTER TABLE agent_task_limits_old RENAME TO agent_task_limits;

CREATE TABLE agent_tag_limits_old (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  tag_name TEXT NOT NULL,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, tag_name)
);
INSERT INTO agent_tag_limits_old (agent_id, tag_name, max_concurrent)
SELECT m.old_id, l.tag_name, l.max_concurrent
FROM agent_tag_limits l JOIN agent_down_map m ON m.new_id = l.agent_id;
DROP TABLE agent_tag_limits;
ALTER TABLE agent_tag_limits_old RENAME TO agent_tag_limits;

CREATE TABLE agent_project_limits_old (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, project_id)
);
INSERT INTO agent_project_limits_old (agent_id, project_id, max_concurrent)
SELECT m.old_id, l.project_id, l.max_concurrent
FROM agent_project_limits l JOIN agent_down_map m ON m.new_id = l.agent_id;
DROP TABLE agent_project_limits;
ALTER TABLE agent_project_limits_old RENAME TO agent_project_limits;

CREATE TABLE agent_project_tag_limits_old (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, project_id, tag_id)
);
INSERT INTO agent_project_tag_limits_old (agent_id, project_id, tag_id, max_concurrent)
SELECT m.old_id, l.project_id, l.tag_id, l.max_concurrent
FROM agent_project_tag_limits l JOIN agent_down_map m ON m.new_id = l.agent_id;
DROP TABLE agent_project_tag_limits;
ALTER TABLE agent_project_tag_limits_old RENAME TO agent_project_tag_limits;

CREATE TABLE project_agents_old (
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  can_list_tasks INTEGER NOT NULL DEFAULT 1,
  can_create_tasks INTEGER NOT NULL DEFAULT 0,
  can_update_tasks INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (project_id, agent_id)
);
INSERT INTO project_agents_old (
  project_id, agent_id, created_at, can_list_tasks, can_create_tasks, can_update_tasks
)
SELECT pa.project_id, m.old_id, pa.created_at, pa.can_list_tasks,
       pa.can_create_tasks, pa.can_update_tasks
FROM project_agents pa JOIN agent_down_map m ON m.new_id = pa.agent_id;
DROP TABLE project_agents;
ALTER TABLE project_agents_old RENAME TO project_agents;
CREATE INDEX idx_project_agents_agent_id ON project_agents (agent_id);

CREATE TABLE prompts_old (
  id INTEGER PRIMARY KEY,
  scope TEXT NOT NULL CHECK (scope IN ('org', 'agent', 'project', 'agent_project')),
  agent_id TEXT REFERENCES agents (id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects (id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  version INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);
INSERT INTO prompts_old (id, scope, agent_id, project_id, content, version, created_at)
SELECT p.id, p.scope, m.old_id, p.project_id, p.content, p.version, p.created_at
FROM prompts p LEFT JOIN agent_down_map m ON m.new_id = p.agent_id;
DROP TABLE prompts;
ALTER TABLE prompts_old RENAME TO prompts;
CREATE INDEX idx_prompts_key ON prompts (scope, agent_id, project_id, version DESC);

CREATE TABLE tasks_old (
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
    'draft', 'queued', 'running', 'input', 'settled', 'failed', 'review',
    'completed', 'cancelled'
  )),
  session_id TEXT NOT NULL DEFAULT '',
  session_agent_id TEXT NOT NULL DEFAULT '',
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
  session_deleted_at INTEGER
);
INSERT INTO tasks_old (
  rowid, id, seq, prompt, name, project_id, assignee_agent_id, assignee_pinned,
  priority, state, session_id, session_agent_id, org_prompt_id, agent_prompt_id,
  project_prompt_id, agent_project_prompt_id, error, queued_at, started_at,
  settled_at, created_at, updated_at, model_provider, model_id, thinking_level,
  interactive, workspace, auto_title, blocked, pending_prompt,
  state_updated_at, session_deleted_at
)
SELECT
  t.rowid, t.id, t.seq, t.prompt, t.name, t.project_id, m.old_id,
  t.assignee_pinned, t.priority, t.state, t.session_id,
  COALESCE((SELECT m2.old_id FROM agent_down_map m2 WHERE m2.new_id = t.session_agent_id), ''),
  t.org_prompt_id,
  t.agent_prompt_id, t.project_prompt_id, t.agent_project_prompt_id, t.error,
  t.queued_at, t.started_at, t.settled_at, t.created_at, t.updated_at,
  t.model_provider, t.model_id, t.thinking_level, t.interactive, t.workspace,
  t.auto_title, t.blocked, t.pending_prompt, t.state_updated_at,
  t.session_deleted_at
FROM tasks t
LEFT JOIN agent_down_map m ON m.actor_id = t.assignee_actor_id;
DROP TABLE tasks;
ALTER TABLE tasks_old RENAME TO tasks;
CREATE INDEX idx_tasks_state ON tasks (state, priority DESC, created_at);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_agent_id, state);
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

CREATE TABLE notifications_old (
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
INSERT INTO notifications_old (
  id, agent_id, task_rowid, kind, severity, title, body, created_at, read_at
)
SELECT n.id, m.old_id, n.task_rowid, n.kind, n.severity, n.title, n.body,
       n.created_at, n.read_at
FROM notifications n LEFT JOIN agent_down_map m ON m.new_id = n.agent_id;
DROP TABLE notifications;
ALTER TABLE notifications_old RENAME TO notifications;
CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);
CREATE INDEX idx_notifications_task_rowid ON notifications (task_rowid);

CREATE TABLE webhooks_old (
  id INTEGER PRIMARY KEY,
  agent_id TEXT REFERENCES agents (id) ON DELETE CASCADE,
  url TEXT NOT NULL,
  events TEXT NOT NULL,
  secret TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
INSERT INTO webhooks_old (id, agent_id, url, events, secret, created_at, updated_at)
SELECT w.id, m.old_id, w.url, w.events, w.secret, w.created_at, w.updated_at
FROM webhooks w LEFT JOIN agent_down_map m ON m.new_id = w.agent_id;
DROP TABLE webhooks;
ALTER TABLE webhooks_old RENAME TO webhooks;

CREATE TABLE agent_usage_old (
  id INTEGER PRIMARY KEY,
  agent_id TEXT NOT NULL,
  session_id TEXT NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cost REAL NOT NULL DEFAULT 0,
  active_ms INTEGER NOT NULL DEFAULT 0,
  at INTEGER NOT NULL
);
INSERT INTO agent_usage_old (
  id, agent_id, session_id, model, input_tokens, output_tokens, cost, active_ms, at
)
SELECT u.id, COALESCE(m.old_id, ''), u.session_id, u.model, u.input_tokens,
       u.output_tokens, u.cost, u.active_ms, u.at
FROM agent_usage u LEFT JOIN agent_down_map m ON m.new_id = u.agent_id;
DROP TABLE agent_usage;
ALTER TABLE agent_usage_old RENAME TO agent_usage;
CREATE INDEX idx_agent_usage_agent_id ON agent_usage (agent_id);
CREATE INDEX idx_agent_usage_session_id ON agent_usage (session_id);

CREATE TABLE agent_logs_old (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id TEXT NOT NULL,
  at INTEGER NOT NULL,
  level INTEGER NOT NULL DEFAULT 1,
  message TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  session_id TEXT NOT NULL DEFAULT '',
  attrs_json TEXT NOT NULL DEFAULT ''
);
INSERT INTO agent_logs_old (
  id, agent_id, at, level, message, task_id, session_id, attrs_json
)
SELECT l.id, COALESCE(m.old_id, ''), l.at, l.level, l.message, l.task_id,
       l.session_id, l.attrs_json
FROM agent_logs l LEFT JOIN agent_down_map m ON m.new_id = l.agent_id;
DROP TABLE agent_logs;
ALTER TABLE agent_logs_old RENAME TO agent_logs;
CREATE INDEX idx_agent_logs_agent ON agent_logs (agent_id, id DESC);

DROP TABLE agents;
ALTER TABLE agents_old RENAME TO agents;
DROP TABLE agent_down_map;
DROP TABLE actors;
