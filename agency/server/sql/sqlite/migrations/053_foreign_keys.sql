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



























UPDATE teams SET created_by_actor_id = (SELECT MIN(id) FROM actors)
WHERE NOT EXISTS (SELECT 1 FROM actors a WHERE a.id = teams.created_by_actor_id)
  AND EXISTS (SELECT 1 FROM actors);



DELETE FROM agents WHERE NOT EXISTS (SELECT 1 FROM actors a WHERE a.id = agents.actor_id);
DELETE FROM users WHERE NOT EXISTS (SELECT 1 FROM actors a WHERE a.id = users.actor_id);
DELETE FROM tags WHERE project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = tags.project_id);
DELETE FROM prompts WHERE (agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents a WHERE a.id = prompts.agent_id)) OR (team_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM teams t WHERE t.id = prompts.team_id)) OR (project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = prompts.project_id));
DELETE FROM email_credentials WHERE NOT EXISTS (SELECT 1 FROM actors a WHERE a.id = email_credentials.actor_id);
DELETE FROM teams WHERE NOT EXISTS (SELECT 1 FROM actors a WHERE a.id = teams.created_by_actor_id);


DELETE FROM actor_settings WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = actor_settings.actor_id);
UPDATE actors SET picture_attachment_id = NULL WHERE picture_attachment_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM attachments p WHERE p.id = actors.picture_attachment_id);
UPDATE agent_domain_logs SET actor_id = NULL WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = agent_domain_logs.actor_id);
DELETE FROM agent_domain_logs WHERE agent_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = agent_domain_logs.agent_actor_id);
UPDATE agent_logs SET agent_id = NULL WHERE agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents p WHERE p.id = agent_logs.agent_id);
DELETE FROM agent_owners WHERE owner_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = agent_owners.owner_actor_id);
DELETE FROM agent_owners WHERE agent_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = agent_owners.agent_actor_id);
DELETE FROM agent_project_limits WHERE project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = agent_project_limits.project_id);
DELETE FROM agent_project_limits WHERE agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents p WHERE p.id = agent_project_limits.agent_id);
DELETE FROM agent_project_tag_limits WHERE tag_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tags p WHERE p.id = agent_project_tag_limits.tag_id);
DELETE FROM agent_project_tag_limits WHERE project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = agent_project_tag_limits.project_id);
DELETE FROM agent_project_tag_limits WHERE agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents p WHERE p.id = agent_project_tag_limits.agent_id);
DELETE FROM agent_tag_limits WHERE agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents p WHERE p.id = agent_tag_limits.agent_id);
DELETE FROM agent_task_limits WHERE agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents p WHERE p.id = agent_task_limits.agent_id);
UPDATE agent_usage SET agent_id = NULL WHERE agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents p WHERE p.id = agent_usage.agent_id);
UPDATE artifact_parts SET attachment_id = NULL WHERE attachment_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM attachments p WHERE p.id = artifact_parts.attachment_id);
DELETE FROM external_credentials WHERE provider_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM identity_providers p WHERE p.id = external_credentials.provider_id);
DELETE FROM external_credentials WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = external_credentials.actor_id);
UPDATE invitations SET invited_by_actor_id = NULL WHERE invited_by_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = invitations.invited_by_actor_id);
DELETE FROM invitations WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = invitations.actor_id);
DELETE FROM ldap_credentials WHERE provider_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM ldap_providers p WHERE p.id = ldap_credentials.provider_id);
DELETE FROM ldap_credentials WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = ldap_credentials.actor_id);
DELETE FROM notifications WHERE recipient_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = notifications.recipient_actor_id);
DELETE FROM notifications WHERE agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents p WHERE p.id = notifications.agent_id);
DELETE FROM personal_access_tokens WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = personal_access_tokens.actor_id);
UPDATE project_actor_memberships SET granted_by_actor_id = NULL WHERE granted_by_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = project_actor_memberships.granted_by_actor_id);
DELETE FROM project_actor_memberships WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = project_actor_memberships.actor_id);
DELETE FROM project_actor_memberships WHERE project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = project_actor_memberships.project_id);
DELETE FROM project_ancestors WHERE ancestor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = project_ancestors.ancestor_id);
DELETE FROM project_ancestors WHERE project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = project_ancestors.project_id);
UPDATE project_logs SET actor_id = NULL WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = project_logs.actor_id);
DELETE FROM project_logs WHERE project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = project_logs.project_id);
UPDATE project_parent_invitations SET responded_by_actor_id = NULL WHERE responded_by_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = project_parent_invitations.responded_by_actor_id);
UPDATE project_parent_invitations SET invited_by_actor_id = NULL WHERE invited_by_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = project_parent_invitations.invited_by_actor_id);
DELETE FROM project_parent_invitations WHERE child_project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = project_parent_invitations.child_project_id);
DELETE FROM project_parent_invitations WHERE parent_project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = project_parent_invitations.parent_project_id);
UPDATE project_team_invitations SET responded_by_actor_id = NULL WHERE responded_by_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = project_team_invitations.responded_by_actor_id);
UPDATE project_team_invitations SET invited_by_actor_id = NULL WHERE invited_by_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = project_team_invitations.invited_by_actor_id);
DELETE FROM project_team_invitations WHERE team_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM teams p WHERE p.id = project_team_invitations.team_id);
DELETE FROM project_team_invitations WHERE project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = project_team_invitations.project_id);
UPDATE project_team_memberships SET granted_by_actor_id = NULL WHERE granted_by_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = project_team_memberships.granted_by_actor_id);
DELETE FROM project_team_memberships WHERE team_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM teams p WHERE p.id = project_team_memberships.team_id);
DELETE FROM project_team_memberships WHERE project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = project_team_memberships.project_id);
UPDATE projects SET parent_project_id = NULL WHERE parent_project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = projects.parent_project_id);
UPDATE projects SET picture_attachment_id = NULL WHERE picture_attachment_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM attachments p WHERE p.id = projects.picture_attachment_id);
DELETE FROM sessions WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = sessions.actor_id);
DELETE FROM task_artifacts WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_artifacts.task_rowid);
DELETE FROM task_attachments WHERE attachment_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM attachments p WHERE p.id = task_attachments.attachment_id);
DELETE FROM task_attachments WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_attachments.task_rowid);
DELETE FROM task_favorites WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = task_favorites.actor_id);
DELETE FROM task_favorites WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_favorites.task_rowid);
DELETE FROM task_leads WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = task_leads.actor_id);
DELETE FROM task_leads WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_leads.task_rowid);
UPDATE task_logs SET actor_id = NULL WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = task_logs.actor_id);
DELETE FROM task_logs WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_logs.task_rowid);
UPDATE task_messages SET actor_id = NULL WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = task_messages.actor_id);
DELETE FROM task_messages WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_messages.task_rowid);
DELETE FROM task_session_entries WHERE task_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.id = task_session_entries.task_id);
DELETE FROM task_subscriptions WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = task_subscriptions.actor_id);
DELETE FROM task_subscriptions WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_subscriptions.task_rowid);
DELETE FROM task_tags WHERE tag_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tags p WHERE p.id = task_tags.tag_id);
UPDATE task_team_prompts SET project_prompt_id = NULL WHERE project_prompt_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = task_team_prompts.project_prompt_id);
UPDATE task_team_prompts SET prompt_id = NULL WHERE prompt_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = task_team_prompts.prompt_id);
DELETE FROM task_team_prompts WHERE team_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM teams p WHERE p.id = task_team_prompts.team_id);
DELETE FROM task_team_prompts WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_team_prompts.task_rowid);
DELETE FROM task_teams WHERE team_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM teams p WHERE p.id = task_teams.team_id);
DELETE FROM task_teams WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_teams.task_rowid);
UPDATE tasks SET pending_prompt_actor_id = NULL WHERE pending_prompt_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = tasks.pending_prompt_actor_id);
UPDATE tasks SET creator_actor_id = NULL WHERE creator_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = tasks.creator_actor_id);
UPDATE tasks SET agent_project_prompt_id = NULL WHERE agent_project_prompt_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = tasks.agent_project_prompt_id);
UPDATE tasks SET project_prompt_id = NULL WHERE project_prompt_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = tasks.project_prompt_id);
UPDATE tasks SET agent_prompt_id = NULL WHERE agent_prompt_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = tasks.agent_prompt_id);
UPDATE tasks SET org_prompt_id = NULL WHERE org_prompt_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = tasks.org_prompt_id);
UPDATE tasks SET session_agent_id = NULL WHERE session_agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents p WHERE p.id = tasks.session_agent_id);
UPDATE tasks SET assignee_actor_id = NULL WHERE assignee_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = tasks.assignee_actor_id);
UPDATE tasks SET project_id = NULL WHERE project_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = tasks.project_id);
UPDATE team_invitations SET responded_by_actor_id = NULL WHERE responded_by_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = team_invitations.responded_by_actor_id);
UPDATE team_invitations SET invited_by_actor_id = NULL WHERE invited_by_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = team_invitations.invited_by_actor_id);
DELETE FROM team_invitations WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = team_invitations.actor_id);
DELETE FROM team_invitations WHERE team_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM teams p WHERE p.id = team_invitations.team_id);
UPDATE team_logs SET actor_id = NULL WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = team_logs.actor_id);
DELETE FROM team_logs WHERE team_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM teams p WHERE p.id = team_logs.team_id);
DELETE FROM team_members WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = team_members.actor_id);
DELETE FROM team_members WHERE team_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM teams p WHERE p.id = team_members.team_id);
DELETE FROM tokens WHERE email_credential_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM email_credentials p WHERE p.id = tokens.email_credential_id);
DELETE FROM tokens WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = tokens.actor_id);
DELETE FROM totp_recovery_codes WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = totp_recovery_codes.actor_id);
UPDATE user_logs SET actor_id = NULL WHERE actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = user_logs.actor_id);
DELETE FROM user_logs WHERE user_actor_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM actors p WHERE p.id = user_logs.user_actor_id);
DELETE FROM webhooks WHERE agent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM agents p WHERE p.id = webhooks.agent_id);
DELETE FROM notifications WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = notifications.task_rowid);
DELETE FROM task_links WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_links.task_rowid);
DELETE FROM task_references WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_references.task_rowid);
DELETE FROM task_references WHERE referenced_task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_references.referenced_task_rowid);
DELETE FROM task_tags WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = task_tags.task_rowid);
DELETE FROM artifact_parts WHERE task_rowid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tasks p WHERE p.rowid = artifact_parts.task_rowid);
DELETE FROM message_attachments WHERE attachment_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM attachments p WHERE p.id = message_attachments.attachment_id);
DELETE FROM message_attachments WHERE message_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM task_messages p WHERE p.id = message_attachments.message_id);
DELETE FROM artifact_parts WHERE NOT EXISTS (SELECT 1 FROM task_artifacts a WHERE a.task_rowid = artifact_parts.task_rowid AND a.artifact_id = artifact_parts.artifact_id);



CREATE TABLE notifications_new (
  id INTEGER PRIMARY KEY,
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
  task_rowid INTEGER REFERENCES tasks (rowid) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'info',
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  read_at INTEGER
, recipient_actor_id INTEGER REFERENCES actors (id) ON DELETE CASCADE);
INSERT INTO notifications_new
  (id, agent_id, task_rowid, kind, severity, title, body, created_at, read_at, recipient_actor_id)
  SELECT id, agent_id, task_rowid, kind, severity, title, body, created_at, read_at, recipient_actor_id
  FROM notifications;
DROP TABLE notifications;
ALTER TABLE notifications_new RENAME TO notifications;
CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);
CREATE INDEX idx_notifications_recipient ON notifications (recipient_actor_id, read_at);
CREATE INDEX idx_notifications_task_rowid ON notifications (task_rowid);

CREATE TABLE task_links_new (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  url TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  position INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (task_rowid, url)
);
INSERT INTO task_links_new (task_rowid, url, label, position)
  SELECT task_rowid, url, label, position FROM task_links;
DROP TABLE task_links;
ALTER TABLE task_links_new RENAME TO task_links;
CREATE INDEX idx_task_links_task ON task_links (task_rowid, position);

CREATE TABLE task_references_new (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  referenced_task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  created_at INTEGER NOT NULL, kind TEXT NOT NULL DEFAULT 'blocking'
  CHECK (kind IN ('blocking', 'informational', 'supersedes')),
  PRIMARY KEY (task_rowid, referenced_task_rowid),
  CHECK (task_rowid <> referenced_task_rowid)
);
INSERT INTO task_references_new (task_rowid, referenced_task_rowid, created_at, kind)
  SELECT task_rowid, referenced_task_rowid, created_at, kind FROM task_references;
DROP TABLE task_references;
ALTER TABLE task_references_new RENAME TO task_references;
CREATE INDEX idx_task_references_on ON task_references (referenced_task_rowid);

CREATE TABLE task_tags_new (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags (id),
  PRIMARY KEY (task_rowid, tag_id)
);
INSERT INTO task_tags_new (task_rowid, tag_id) SELECT task_rowid, tag_id FROM task_tags;
DROP TABLE task_tags;
ALTER TABLE task_tags_new RENAME TO task_tags;
CREATE INDEX idx_task_tags_tag ON task_tags (tag_id);

CREATE TABLE artifact_parts_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  artifact_id TEXT NOT NULL,
  position INTEGER NOT NULL DEFAULT 0,
  kind TEXT NOT NULL CHECK (kind IN ('text', 'file', 'data')),
  text TEXT NOT NULL DEFAULT '',
  data_json TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL,
  filename TEXT NOT NULL DEFAULT '',
  media_type TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (task_rowid, artifact_id)
    REFERENCES task_artifacts (task_rowid, artifact_id) ON DELETE CASCADE
);
INSERT INTO artifact_parts_new
  (id, task_rowid, artifact_id, position, kind, text, data_json, url,
   attachment_id, filename, media_type, metadata_json)
  SELECT id, task_rowid, artifact_id, position, kind, text, data_json, url,
         attachment_id, filename, media_type, metadata_json
  FROM artifact_parts;
DROP TABLE artifact_parts;
ALTER TABLE artifact_parts_new RENAME TO artifact_parts;
CREATE INDEX idx_artifact_parts_artifact ON artifact_parts (task_rowid, artifact_id, position);
CREATE INDEX idx_artifact_parts_attachment ON artifact_parts (attachment_id);





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
         assignee_pinned, priority, state, session_id, session_agent_id,
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



CREATE TABLE task_session_entries_new (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE ON UPDATE CASCADE,
  seq INTEGER NOT NULL,
  entry_id TEXT NOT NULL,
  parent_id TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL,
  raw TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (task_id, seq)
);
INSERT INTO task_session_entries_new
  (task_id, seq, entry_id, parent_id, type, raw, created_at)
  SELECT task_id, seq, entry_id, parent_id, type, raw, created_at
  FROM task_session_entries;
DROP TABLE task_session_entries;
ALTER TABLE task_session_entries_new RENAME TO task_session_entries;
CREATE UNIQUE INDEX idx_task_session_entries_entry
  ON task_session_entries (task_id, entry_id);

-- +goose Down


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

CREATE TABLE notifications_old (
  id INTEGER PRIMARY KEY,
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
  task_rowid INTEGER,
  kind TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'info',
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  read_at INTEGER
, recipient_actor_id INTEGER REFERENCES actors (id) ON DELETE CASCADE);
INSERT INTO notifications_old
  (id, agent_id, task_rowid, kind, severity, title, body, created_at, read_at, recipient_actor_id)
  SELECT id, agent_id, task_rowid, kind, severity, title, body, created_at, read_at, recipient_actor_id
  FROM notifications;
DROP TABLE notifications;
ALTER TABLE notifications_old RENAME TO notifications;
CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);
CREATE INDEX idx_notifications_recipient ON notifications (recipient_actor_id, read_at);
CREATE INDEX idx_notifications_task_rowid ON notifications (task_rowid);

CREATE TABLE task_links_old (
  task_rowid INTEGER NOT NULL,
  url TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  position INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (task_rowid, url)
);
INSERT INTO task_links_old (task_rowid, url, label, position)
  SELECT task_rowid, url, label, position FROM task_links;
DROP TABLE task_links;
ALTER TABLE task_links_old RENAME TO task_links;
CREATE INDEX idx_task_links_task ON task_links (task_rowid, position);

CREATE TABLE task_references_old (
  task_rowid INTEGER NOT NULL,
  referenced_task_rowid INTEGER NOT NULL,
  created_at INTEGER NOT NULL, kind TEXT NOT NULL DEFAULT 'blocking'
  CHECK (kind IN ('blocking', 'informational', 'supersedes')),
  PRIMARY KEY (task_rowid, referenced_task_rowid),
  CHECK (task_rowid <> referenced_task_rowid)
);
INSERT INTO task_references_old (task_rowid, referenced_task_rowid, created_at, kind)
  SELECT task_rowid, referenced_task_rowid, created_at, kind FROM task_references;
DROP TABLE task_references;
ALTER TABLE task_references_old RENAME TO task_references;
CREATE INDEX idx_task_references_on ON task_references (referenced_task_rowid);

CREATE TABLE task_tags_old (
  task_rowid INTEGER NOT NULL,
  tag_id TEXT NOT NULL REFERENCES tags (id),
  PRIMARY KEY (task_rowid, tag_id)
);
INSERT INTO task_tags_old (task_rowid, tag_id) SELECT task_rowid, tag_id FROM task_tags;
DROP TABLE task_tags;
ALTER TABLE task_tags_old RENAME TO task_tags;
CREATE INDEX idx_task_tags_tag ON task_tags (tag_id);

CREATE TABLE artifact_parts_old (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_rowid INTEGER NOT NULL,
  artifact_id TEXT NOT NULL,
  position INTEGER NOT NULL DEFAULT 0,
  kind TEXT NOT NULL CHECK (kind IN ('text', 'file', 'data')),
  text TEXT NOT NULL DEFAULT '',
  data_json TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL,
  filename TEXT NOT NULL DEFAULT '',
  media_type TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT ''
);
INSERT INTO artifact_parts_old
  (id, task_rowid, artifact_id, position, kind, text, data_json, url,
   attachment_id, filename, media_type, metadata_json)
  SELECT id, task_rowid, artifact_id, position, kind, text, data_json, url,
         attachment_id, filename, media_type, metadata_json
  FROM artifact_parts;
DROP TABLE artifact_parts;
ALTER TABLE artifact_parts_old RENAME TO artifact_parts;
CREATE INDEX idx_artifact_parts_artifact ON artifact_parts (task_rowid, artifact_id, position);
CREATE INDEX idx_artifact_parts_attachment ON artifact_parts (attachment_id);

CREATE TABLE task_session_entries_old (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  entry_id TEXT NOT NULL,
  parent_id TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL,
  raw TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (task_id, seq)
);
INSERT INTO task_session_entries_old
  (task_id, seq, entry_id, parent_id, type, raw, created_at)
  SELECT task_id, seq, entry_id, parent_id, type, raw, created_at
  FROM task_session_entries;
DROP TABLE task_session_entries;
ALTER TABLE task_session_entries_old RENAME TO task_session_entries;
CREATE UNIQUE INDEX idx_task_session_entries_entry
  ON task_session_entries (task_id, entry_id);
