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


ALTER TABLE actor_settings ADD CONSTRAINT fk_actor_settings_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE actors ADD CONSTRAINT fk_actors_picture_attachment_id FOREIGN KEY (picture_attachment_id) REFERENCES attachments (id) ON DELETE SET NULL;
ALTER TABLE agent_domain_logs ADD CONSTRAINT fk_agent_domain_logs_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE agent_domain_logs ADD CONSTRAINT fk_agent_domain_logs_agent_actor_id FOREIGN KEY (agent_actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE agent_logs ADD CONSTRAINT fk_agent_logs_agent_id FOREIGN KEY (agent_id) REFERENCES agents (id) ON DELETE SET NULL;
ALTER TABLE agent_owners ADD CONSTRAINT fk_agent_owners_owner_actor_id FOREIGN KEY (owner_actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE agent_owners ADD CONSTRAINT fk_agent_owners_agent_actor_id FOREIGN KEY (agent_actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE agent_project_limits ADD CONSTRAINT fk_agent_project_limits_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE agent_project_limits ADD CONSTRAINT fk_agent_project_limits_agent_id FOREIGN KEY (agent_id) REFERENCES agents (id) ON DELETE CASCADE;
ALTER TABLE agent_project_tag_limits ADD CONSTRAINT fk_agent_project_tag_limits_tag_id FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE CASCADE;
ALTER TABLE agent_project_tag_limits ADD CONSTRAINT fk_agent_project_tag_limits_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE agent_project_tag_limits ADD CONSTRAINT fk_agent_project_tag_limits_agent_id FOREIGN KEY (agent_id) REFERENCES agents (id) ON DELETE CASCADE;
ALTER TABLE agent_tag_limits ADD CONSTRAINT fk_agent_tag_limits_agent_id FOREIGN KEY (agent_id) REFERENCES agents (id) ON DELETE CASCADE;
ALTER TABLE agent_task_limits ADD CONSTRAINT fk_agent_task_limits_agent_id FOREIGN KEY (agent_id) REFERENCES agents (id) ON DELETE CASCADE;
ALTER TABLE agent_usage ADD CONSTRAINT fk_agent_usage_agent_id FOREIGN KEY (agent_id) REFERENCES agents (id) ON DELETE SET NULL;
ALTER TABLE agents ADD CONSTRAINT fk_agents_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE artifact_parts ADD CONSTRAINT fk_artifact_parts_attachment_id FOREIGN KEY (attachment_id) REFERENCES attachments (id) ON DELETE SET NULL;
ALTER TABLE email_credentials ADD CONSTRAINT fk_email_credentials_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE external_credentials ADD CONSTRAINT fk_external_credentials_provider_id FOREIGN KEY (provider_id) REFERENCES identity_providers (id) ON DELETE CASCADE;
ALTER TABLE external_credentials ADD CONSTRAINT fk_external_credentials_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE invitations ADD CONSTRAINT fk_invitations_invited_by_actor_id FOREIGN KEY (invited_by_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE invitations ADD CONSTRAINT fk_invitations_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE ldap_credentials ADD CONSTRAINT fk_ldap_credentials_provider_id FOREIGN KEY (provider_id) REFERENCES ldap_providers (id) ON DELETE CASCADE;
ALTER TABLE ldap_credentials ADD CONSTRAINT fk_ldap_credentials_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE message_attachments ADD CONSTRAINT fk_message_attachments_attachment_id FOREIGN KEY (attachment_id) REFERENCES attachments (id) ON DELETE CASCADE;
ALTER TABLE message_attachments ADD CONSTRAINT fk_message_attachments_message_id FOREIGN KEY (message_id) REFERENCES task_messages (id) ON DELETE CASCADE;
ALTER TABLE notifications ADD CONSTRAINT fk_notifications_recipient_actor_id FOREIGN KEY (recipient_actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE notifications ADD CONSTRAINT fk_notifications_agent_id FOREIGN KEY (agent_id) REFERENCES agents (id) ON DELETE CASCADE;
ALTER TABLE personal_access_tokens ADD CONSTRAINT fk_personal_access_tokens_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE project_actor_memberships ADD CONSTRAINT fk_project_actor_memberships_granted_by_actor_id FOREIGN KEY (granted_by_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE project_actor_memberships ADD CONSTRAINT fk_project_actor_memberships_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE project_actor_memberships ADD CONSTRAINT fk_project_actor_memberships_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE project_ancestors ADD CONSTRAINT fk_project_ancestors_ancestor_id FOREIGN KEY (ancestor_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE project_ancestors ADD CONSTRAINT fk_project_ancestors_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE project_logs ADD CONSTRAINT fk_project_logs_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE project_logs ADD CONSTRAINT fk_project_logs_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE project_parent_invitations ADD CONSTRAINT fk_project_parent_invitations_responded_by_actor_id FOREIGN KEY (responded_by_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE project_parent_invitations ADD CONSTRAINT fk_project_parent_invitations_invited_by_actor_id FOREIGN KEY (invited_by_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE project_parent_invitations ADD CONSTRAINT fk_project_parent_invitations_child_project_id FOREIGN KEY (child_project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE project_parent_invitations ADD CONSTRAINT fk_project_parent_invitations_parent_project_id FOREIGN KEY (parent_project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE project_team_invitations ADD CONSTRAINT fk_project_team_invitations_responded_by_actor_id FOREIGN KEY (responded_by_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE project_team_invitations ADD CONSTRAINT fk_project_team_invitations_invited_by_actor_id FOREIGN KEY (invited_by_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE project_team_invitations ADD CONSTRAINT fk_project_team_invitations_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE project_team_invitations ADD CONSTRAINT fk_project_team_invitations_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE project_team_memberships ADD CONSTRAINT fk_project_team_memberships_granted_by_actor_id FOREIGN KEY (granted_by_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE project_team_memberships ADD CONSTRAINT fk_project_team_memberships_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE project_team_memberships ADD CONSTRAINT fk_project_team_memberships_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE projects ADD CONSTRAINT fk_projects_parent_project_id FOREIGN KEY (parent_project_id) REFERENCES projects (id) ON DELETE RESTRICT;
ALTER TABLE projects ADD CONSTRAINT fk_projects_picture_attachment_id FOREIGN KEY (picture_attachment_id) REFERENCES attachments (id) ON DELETE SET NULL;
ALTER TABLE prompts ADD CONSTRAINT fk_prompts_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE prompts ADD CONSTRAINT fk_prompts_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE prompts ADD CONSTRAINT fk_prompts_agent_id FOREIGN KEY (agent_id) REFERENCES agents (id) ON DELETE CASCADE;
ALTER TABLE sessions ADD CONSTRAINT fk_sessions_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE tags ADD CONSTRAINT fk_tags_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE RESTRICT;
ALTER TABLE task_artifacts ADD CONSTRAINT fk_task_artifacts_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_attachments ADD CONSTRAINT fk_task_attachments_attachment_id FOREIGN KEY (attachment_id) REFERENCES attachments (id) ON DELETE CASCADE;
ALTER TABLE task_attachments ADD CONSTRAINT fk_task_attachments_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_favorites ADD CONSTRAINT fk_task_favorites_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE task_favorites ADD CONSTRAINT fk_task_favorites_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_leads ADD CONSTRAINT fk_task_leads_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE task_leads ADD CONSTRAINT fk_task_leads_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_logs ADD CONSTRAINT fk_task_logs_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE task_logs ADD CONSTRAINT fk_task_logs_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_messages ADD CONSTRAINT fk_task_messages_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE task_messages ADD CONSTRAINT fk_task_messages_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_session_entries ADD CONSTRAINT fk_task_session_entries_task_id FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE task_subscriptions ADD CONSTRAINT fk_task_subscriptions_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE task_subscriptions ADD CONSTRAINT fk_task_subscriptions_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_tags ADD CONSTRAINT fk_task_tags_tag_id FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE RESTRICT;
ALTER TABLE task_team_prompts ADD CONSTRAINT fk_task_team_prompts_project_prompt_id FOREIGN KEY (project_prompt_id) REFERENCES prompts (id) ON DELETE SET NULL;
ALTER TABLE task_team_prompts ADD CONSTRAINT fk_task_team_prompts_prompt_id FOREIGN KEY (prompt_id) REFERENCES prompts (id) ON DELETE SET NULL;
ALTER TABLE task_team_prompts ADD CONSTRAINT fk_task_team_prompts_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE task_team_prompts ADD CONSTRAINT fk_task_team_prompts_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_teams ADD CONSTRAINT fk_task_teams_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE task_teams ADD CONSTRAINT fk_task_teams_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_pending_prompt_actor_id FOREIGN KEY (pending_prompt_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_creator_actor_id FOREIGN KEY (creator_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_agent_project_prompt_id FOREIGN KEY (agent_project_prompt_id) REFERENCES prompts (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_project_prompt_id FOREIGN KEY (project_prompt_id) REFERENCES prompts (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_agent_prompt_id FOREIGN KEY (agent_prompt_id) REFERENCES prompts (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_org_prompt_id FOREIGN KEY (org_prompt_id) REFERENCES prompts (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_session_agent_id FOREIGN KEY (session_agent_id) REFERENCES agents (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_assignee_actor_id FOREIGN KEY (assignee_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE RESTRICT;
ALTER TABLE team_invitations ADD CONSTRAINT fk_team_invitations_responded_by_actor_id FOREIGN KEY (responded_by_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE team_invitations ADD CONSTRAINT fk_team_invitations_invited_by_actor_id FOREIGN KEY (invited_by_actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE team_invitations ADD CONSTRAINT fk_team_invitations_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE team_invitations ADD CONSTRAINT fk_team_invitations_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE team_logs ADD CONSTRAINT fk_team_logs_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE team_logs ADD CONSTRAINT fk_team_logs_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE team_members ADD CONSTRAINT fk_team_members_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE team_members ADD CONSTRAINT fk_team_members_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE teams ADD CONSTRAINT fk_teams_created_by_actor_id FOREIGN KEY (created_by_actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE teams ADD CONSTRAINT fk_teams_picture_attachment_id FOREIGN KEY (picture_attachment_id) REFERENCES attachments (id) ON DELETE SET NULL;
ALTER TABLE tokens ADD CONSTRAINT fk_tokens_email_credential_id FOREIGN KEY (email_credential_id) REFERENCES email_credentials (id) ON DELETE CASCADE;
ALTER TABLE tokens ADD CONSTRAINT fk_tokens_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE totp_recovery_codes ADD CONSTRAINT fk_totp_recovery_codes_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE user_logs ADD CONSTRAINT fk_user_logs_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE user_logs ADD CONSTRAINT fk_user_logs_user_actor_id FOREIGN KEY (user_actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE users ADD CONSTRAINT fk_users_actor_id FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;
ALTER TABLE webhooks ADD CONSTRAINT fk_webhooks_agent_id FOREIGN KEY (agent_id) REFERENCES agents (id) ON DELETE CASCADE;
ALTER TABLE notifications ADD CONSTRAINT fk_notifications_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_links ADD CONSTRAINT fk_task_links_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_references ADD CONSTRAINT fk_task_references_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_references ADD CONSTRAINT fk_task_references_referenced_task_rowid FOREIGN KEY (referenced_task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE task_tags ADD CONSTRAINT fk_task_tags_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE artifact_parts ADD CONSTRAINT fk_artifact_parts_task_rowid FOREIGN KEY (task_rowid) REFERENCES tasks (rowid) ON DELETE CASCADE;
ALTER TABLE artifact_parts ADD CONSTRAINT fk_artifact_parts_artifact FOREIGN KEY (task_rowid, artifact_id) REFERENCES task_artifacts (task_rowid, artifact_id) ON DELETE CASCADE;
-- +goose Down
ALTER TABLE actor_settings DROP CONSTRAINT IF EXISTS fk_actor_settings_actor_id;
ALTER TABLE actors DROP CONSTRAINT IF EXISTS fk_actors_picture_attachment_id;
ALTER TABLE agent_domain_logs DROP CONSTRAINT IF EXISTS fk_agent_domain_logs_actor_id;
ALTER TABLE agent_domain_logs DROP CONSTRAINT IF EXISTS fk_agent_domain_logs_agent_actor_id;
ALTER TABLE agent_logs DROP CONSTRAINT IF EXISTS fk_agent_logs_agent_id;
ALTER TABLE agent_owners DROP CONSTRAINT IF EXISTS fk_agent_owners_owner_actor_id;
ALTER TABLE agent_owners DROP CONSTRAINT IF EXISTS fk_agent_owners_agent_actor_id;
ALTER TABLE agent_project_limits DROP CONSTRAINT IF EXISTS fk_agent_project_limits_project_id;
ALTER TABLE agent_project_limits DROP CONSTRAINT IF EXISTS fk_agent_project_limits_agent_id;
ALTER TABLE agent_project_tag_limits DROP CONSTRAINT IF EXISTS fk_agent_project_tag_limits_tag_id;
ALTER TABLE agent_project_tag_limits DROP CONSTRAINT IF EXISTS fk_agent_project_tag_limits_project_id;
ALTER TABLE agent_project_tag_limits DROP CONSTRAINT IF EXISTS fk_agent_project_tag_limits_agent_id;
ALTER TABLE agent_tag_limits DROP CONSTRAINT IF EXISTS fk_agent_tag_limits_agent_id;
ALTER TABLE agent_task_limits DROP CONSTRAINT IF EXISTS fk_agent_task_limits_agent_id;
ALTER TABLE agent_usage DROP CONSTRAINT IF EXISTS fk_agent_usage_agent_id;
ALTER TABLE agents DROP CONSTRAINT IF EXISTS fk_agents_actor_id;
ALTER TABLE artifact_parts DROP CONSTRAINT IF EXISTS fk_artifact_parts_attachment_id;
ALTER TABLE email_credentials DROP CONSTRAINT IF EXISTS fk_email_credentials_actor_id;
ALTER TABLE external_credentials DROP CONSTRAINT IF EXISTS fk_external_credentials_provider_id;
ALTER TABLE external_credentials DROP CONSTRAINT IF EXISTS fk_external_credentials_actor_id;
ALTER TABLE invitations DROP CONSTRAINT IF EXISTS fk_invitations_invited_by_actor_id;
ALTER TABLE invitations DROP CONSTRAINT IF EXISTS fk_invitations_actor_id;
ALTER TABLE ldap_credentials DROP CONSTRAINT IF EXISTS fk_ldap_credentials_provider_id;
ALTER TABLE ldap_credentials DROP CONSTRAINT IF EXISTS fk_ldap_credentials_actor_id;
ALTER TABLE message_attachments DROP CONSTRAINT IF EXISTS fk_message_attachments_attachment_id;
ALTER TABLE message_attachments DROP CONSTRAINT IF EXISTS fk_message_attachments_message_id;
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS fk_notifications_recipient_actor_id;
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS fk_notifications_agent_id;
ALTER TABLE personal_access_tokens DROP CONSTRAINT IF EXISTS fk_personal_access_tokens_actor_id;
ALTER TABLE project_actor_memberships DROP CONSTRAINT IF EXISTS fk_project_actor_memberships_granted_by_actor_id;
ALTER TABLE project_actor_memberships DROP CONSTRAINT IF EXISTS fk_project_actor_memberships_actor_id;
ALTER TABLE project_actor_memberships DROP CONSTRAINT IF EXISTS fk_project_actor_memberships_project_id;
ALTER TABLE project_ancestors DROP CONSTRAINT IF EXISTS fk_project_ancestors_ancestor_id;
ALTER TABLE project_ancestors DROP CONSTRAINT IF EXISTS fk_project_ancestors_project_id;
ALTER TABLE project_logs DROP CONSTRAINT IF EXISTS fk_project_logs_actor_id;
ALTER TABLE project_logs DROP CONSTRAINT IF EXISTS fk_project_logs_project_id;
ALTER TABLE project_parent_invitations DROP CONSTRAINT IF EXISTS fk_project_parent_invitations_responded_by_actor_id;
ALTER TABLE project_parent_invitations DROP CONSTRAINT IF EXISTS fk_project_parent_invitations_invited_by_actor_id;
ALTER TABLE project_parent_invitations DROP CONSTRAINT IF EXISTS fk_project_parent_invitations_child_project_id;
ALTER TABLE project_parent_invitations DROP CONSTRAINT IF EXISTS fk_project_parent_invitations_parent_project_id;
ALTER TABLE project_team_invitations DROP CONSTRAINT IF EXISTS fk_project_team_invitations_responded_by_actor_id;
ALTER TABLE project_team_invitations DROP CONSTRAINT IF EXISTS fk_project_team_invitations_invited_by_actor_id;
ALTER TABLE project_team_invitations DROP CONSTRAINT IF EXISTS fk_project_team_invitations_team_id;
ALTER TABLE project_team_invitations DROP CONSTRAINT IF EXISTS fk_project_team_invitations_project_id;
ALTER TABLE project_team_memberships DROP CONSTRAINT IF EXISTS fk_project_team_memberships_granted_by_actor_id;
ALTER TABLE project_team_memberships DROP CONSTRAINT IF EXISTS fk_project_team_memberships_team_id;
ALTER TABLE project_team_memberships DROP CONSTRAINT IF EXISTS fk_project_team_memberships_project_id;
ALTER TABLE projects DROP CONSTRAINT IF EXISTS fk_projects_parent_project_id;
ALTER TABLE projects DROP CONSTRAINT IF EXISTS fk_projects_picture_attachment_id;
ALTER TABLE prompts DROP CONSTRAINT IF EXISTS fk_prompts_project_id;
ALTER TABLE prompts DROP CONSTRAINT IF EXISTS fk_prompts_team_id;
ALTER TABLE prompts DROP CONSTRAINT IF EXISTS fk_prompts_agent_id;
ALTER TABLE sessions DROP CONSTRAINT IF EXISTS fk_sessions_actor_id;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS fk_tags_project_id;
ALTER TABLE task_artifacts DROP CONSTRAINT IF EXISTS fk_task_artifacts_task_rowid;
ALTER TABLE task_attachments DROP CONSTRAINT IF EXISTS fk_task_attachments_attachment_id;
ALTER TABLE task_attachments DROP CONSTRAINT IF EXISTS fk_task_attachments_task_rowid;
ALTER TABLE task_favorites DROP CONSTRAINT IF EXISTS fk_task_favorites_actor_id;
ALTER TABLE task_favorites DROP CONSTRAINT IF EXISTS fk_task_favorites_task_rowid;
ALTER TABLE task_leads DROP CONSTRAINT IF EXISTS fk_task_leads_actor_id;
ALTER TABLE task_leads DROP CONSTRAINT IF EXISTS fk_task_leads_task_rowid;
ALTER TABLE task_logs DROP CONSTRAINT IF EXISTS fk_task_logs_actor_id;
ALTER TABLE task_logs DROP CONSTRAINT IF EXISTS fk_task_logs_task_rowid;
ALTER TABLE task_messages DROP CONSTRAINT IF EXISTS fk_task_messages_actor_id;
ALTER TABLE task_messages DROP CONSTRAINT IF EXISTS fk_task_messages_task_rowid;
ALTER TABLE task_session_entries DROP CONSTRAINT IF EXISTS fk_task_session_entries_task_id;
ALTER TABLE task_subscriptions DROP CONSTRAINT IF EXISTS fk_task_subscriptions_actor_id;
ALTER TABLE task_subscriptions DROP CONSTRAINT IF EXISTS fk_task_subscriptions_task_rowid;
ALTER TABLE task_tags DROP CONSTRAINT IF EXISTS fk_task_tags_tag_id;
ALTER TABLE task_team_prompts DROP CONSTRAINT IF EXISTS fk_task_team_prompts_project_prompt_id;
ALTER TABLE task_team_prompts DROP CONSTRAINT IF EXISTS fk_task_team_prompts_prompt_id;
ALTER TABLE task_team_prompts DROP CONSTRAINT IF EXISTS fk_task_team_prompts_team_id;
ALTER TABLE task_team_prompts DROP CONSTRAINT IF EXISTS fk_task_team_prompts_task_rowid;
ALTER TABLE task_teams DROP CONSTRAINT IF EXISTS fk_task_teams_team_id;
ALTER TABLE task_teams DROP CONSTRAINT IF EXISTS fk_task_teams_task_rowid;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_pending_prompt_actor_id;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_creator_actor_id;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_agent_project_prompt_id;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_project_prompt_id;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_agent_prompt_id;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_org_prompt_id;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_session_agent_id;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_assignee_actor_id;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_project_id;
ALTER TABLE team_invitations DROP CONSTRAINT IF EXISTS fk_team_invitations_responded_by_actor_id;
ALTER TABLE team_invitations DROP CONSTRAINT IF EXISTS fk_team_invitations_invited_by_actor_id;
ALTER TABLE team_invitations DROP CONSTRAINT IF EXISTS fk_team_invitations_actor_id;
ALTER TABLE team_invitations DROP CONSTRAINT IF EXISTS fk_team_invitations_team_id;
ALTER TABLE team_logs DROP CONSTRAINT IF EXISTS fk_team_logs_actor_id;
ALTER TABLE team_logs DROP CONSTRAINT IF EXISTS fk_team_logs_team_id;
ALTER TABLE team_members DROP CONSTRAINT IF EXISTS fk_team_members_actor_id;
ALTER TABLE team_members DROP CONSTRAINT IF EXISTS fk_team_members_team_id;
ALTER TABLE teams DROP CONSTRAINT IF EXISTS fk_teams_created_by_actor_id;
ALTER TABLE teams DROP CONSTRAINT IF EXISTS fk_teams_picture_attachment_id;
ALTER TABLE tokens DROP CONSTRAINT IF EXISTS fk_tokens_email_credential_id;
ALTER TABLE tokens DROP CONSTRAINT IF EXISTS fk_tokens_actor_id;
ALTER TABLE totp_recovery_codes DROP CONSTRAINT IF EXISTS fk_totp_recovery_codes_actor_id;
ALTER TABLE user_logs DROP CONSTRAINT IF EXISTS fk_user_logs_actor_id;
ALTER TABLE user_logs DROP CONSTRAINT IF EXISTS fk_user_logs_user_actor_id;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_actor_id;
ALTER TABLE webhooks DROP CONSTRAINT IF EXISTS fk_webhooks_agent_id;
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS fk_notifications_task_rowid;
ALTER TABLE task_links DROP CONSTRAINT IF EXISTS fk_task_links_task_rowid;
ALTER TABLE task_references DROP CONSTRAINT IF EXISTS fk_task_references_task_rowid;
ALTER TABLE task_references DROP CONSTRAINT IF EXISTS fk_task_references_referenced_task_rowid;
ALTER TABLE task_tags DROP CONSTRAINT IF EXISTS fk_task_tags_task_rowid;
ALTER TABLE artifact_parts DROP CONSTRAINT IF EXISTS fk_artifact_parts_task_rowid;
ALTER TABLE artifact_parts DROP CONSTRAINT IF EXISTS fk_artifact_parts_artifact;