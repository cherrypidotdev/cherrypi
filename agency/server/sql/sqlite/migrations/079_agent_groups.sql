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











CREATE TABLE agent_groups (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL,
  archived_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX idx_agent_groups_archived ON agent_groups (archived_at);

CREATE TABLE agent_group_members (
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  agent_id INTEGER NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  
  added_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (group_id, agent_id)
);
CREATE INDEX idx_agent_group_members_agent ON agent_group_members (agent_id);


CREATE TABLE agent_group_owners (
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (group_id, actor_id)
);
CREATE INDEX idx_agent_group_owners_actor ON agent_group_owners (actor_id);





CREATE TABLE agent_group_member_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  agent_id INTEGER NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  requested_by_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'accepted', 'declined', 'revoked')),
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  responded_at INTEGER,
  responded_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_agent_group_member_requests_pending
  ON agent_group_member_requests (group_id, agent_id) WHERE status = 'pending';
CREATE INDEX idx_agent_group_member_requests_agent
  ON agent_group_member_requests (agent_id, status);
CREATE INDEX idx_agent_group_member_requests_group
  ON agent_group_member_requests (group_id, status);


CREATE TABLE agent_group_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  at INTEGER NOT NULL,
  actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  kind TEXT NOT NULL,
  field TEXT NOT NULL DEFAULT '',
  old_value TEXT NOT NULL DEFAULT '',
  new_value TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  data_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_agent_group_logs_group ON agent_group_logs (group_id, at DESC, id DESC);
CREATE INDEX idx_agent_group_logs_at ON agent_group_logs (at DESC, id DESC);
CREATE INDEX idx_agent_group_logs_actor ON agent_group_logs (actor_id, at DESC, id DESC);






CREATE TABLE prompts_new (
  id INTEGER PRIMARY KEY,
  scope TEXT NOT NULL CHECK (
    scope IN ('org', 'agent', 'project', 'agent_project', 'team', 'team_project',
              'group', 'group_project', 'group_team')
  ),
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
  team_id INTEGER REFERENCES teams (id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects (id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  version INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  author_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  approved_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  note TEXT NOT NULL DEFAULT '',
  proposal_id INTEGER REFERENCES prompt_proposals (id) ON DELETE SET NULL,
  auto_approved INTEGER NOT NULL DEFAULT 0,
  group_id INTEGER REFERENCES agent_groups (id) ON DELETE CASCADE
);
INSERT INTO prompts_new (
  id, scope, agent_id, team_id, group_id, project_id, content, version,
  created_at, author_actor_id, approved_by_actor_id, note, proposal_id,
  auto_approved
)
SELECT
  id, scope, agent_id, team_id, NULL, project_id, content, version,
  created_at, author_actor_id, approved_by_actor_id, note, proposal_id,
  auto_approved
FROM prompts;
DROP TABLE prompts;
ALTER TABLE prompts_new RENAME TO prompts;
CREATE INDEX idx_prompts_key
  ON prompts (scope, agent_id, team_id, group_id, project_id, version DESC);




CREATE TABLE prompt_proposals_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scope TEXT NOT NULL CHECK (
    scope IN ('org', 'agent', 'project', 'agent_project', 'team', 'team_project',
              'group', 'group_project', 'group_team')
  ),
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
  team_id INTEGER REFERENCES teams (id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects (id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  base_version INTEGER NOT NULL,
  author_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  note TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending' CHECK (
    status IN ('pending', 'approved', 'rejected', 'withdrawn')
  ),
  auto_approved INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  decided_at INTEGER,
  decided_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  decision_note TEXT NOT NULL DEFAULT '',
  group_id INTEGER REFERENCES agent_groups (id) ON DELETE CASCADE
);
INSERT INTO prompt_proposals_new (
  id, scope, agent_id, team_id, group_id, project_id, content, base_version,
  author_actor_id, note, status, auto_approved, created_at, decided_at,
  decided_by_actor_id, decision_note
)
SELECT
  id, scope, agent_id, team_id, NULL, project_id, content, base_version,
  author_actor_id, note, status, auto_approved, created_at, decided_at,
  decided_by_actor_id, decision_note
FROM prompt_proposals;
DROP TABLE prompt_proposals;
ALTER TABLE prompt_proposals_new RENAME TO prompt_proposals;
CREATE INDEX idx_prompt_proposals_target
  ON prompt_proposals (scope, agent_id, team_id, group_id, project_id, status, created_at DESC);
CREATE INDEX idx_prompt_proposals_author
  ON prompt_proposals (author_actor_id, created_at DESC);
CREATE INDEX idx_prompt_proposals_status
  ON prompt_proposals (status, created_at DESC);




ALTER TABLE notifications
  ADD COLUMN prompt_group_id INTEGER REFERENCES agent_groups (id) ON DELETE SET NULL;




CREATE TABLE task_group_prompts (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  project_prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  PRIMARY KEY (task_rowid, group_id)
);
CREATE INDEX idx_task_group_prompts_group ON task_group_prompts (group_id);

CREATE TABLE task_group_team_prompts (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  prompt_id INTEGER REFERENCES prompts (id) ON DELETE SET NULL,
  PRIMARY KEY (task_rowid, group_id, team_id)
);
CREATE INDEX idx_task_group_team_prompts_group
  ON task_group_team_prompts (group_id);

-- +goose Down
ALTER TABLE notifications DROP COLUMN prompt_group_id;

DROP INDEX idx_task_group_team_prompts_group;
DROP TABLE task_group_team_prompts;
DROP INDEX idx_task_group_prompts_group;
DROP TABLE task_group_prompts;

DROP INDEX idx_prompt_proposals_status;
DROP INDEX idx_prompt_proposals_author;
DROP INDEX idx_prompt_proposals_target;
CREATE TABLE prompt_proposals_old (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scope TEXT NOT NULL CHECK (
    scope IN ('org', 'agent', 'project', 'agent_project', 'team', 'team_project')
  ),
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
  team_id INTEGER REFERENCES teams (id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects (id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  base_version INTEGER NOT NULL,
  author_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  note TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending' CHECK (
    status IN ('pending', 'approved', 'rejected', 'withdrawn')
  ),
  auto_approved INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  decided_at INTEGER,
  decided_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  decision_note TEXT NOT NULL DEFAULT ''
);
INSERT INTO prompt_proposals_old (
  id, scope, agent_id, team_id, project_id, content, base_version,
  author_actor_id, note, status, auto_approved, created_at, decided_at,
  decided_by_actor_id, decision_note
)
SELECT
  id, scope, agent_id, team_id, project_id, content, base_version,
  author_actor_id, note, status, auto_approved, created_at, decided_at,
  decided_by_actor_id, decision_note
FROM prompt_proposals
WHERE scope IN ('org', 'agent', 'project', 'agent_project', 'team', 'team_project');
DROP TABLE prompt_proposals;
ALTER TABLE prompt_proposals_old RENAME TO prompt_proposals;
CREATE INDEX idx_prompt_proposals_target
  ON prompt_proposals (scope, agent_id, team_id, project_id, status, created_at DESC);
CREATE INDEX idx_prompt_proposals_author
  ON prompt_proposals (author_actor_id, created_at DESC);
CREATE INDEX idx_prompt_proposals_status
  ON prompt_proposals (status, created_at DESC);

DROP INDEX idx_prompts_key;
CREATE TABLE prompts_old (
  id INTEGER PRIMARY KEY,
  scope TEXT NOT NULL CHECK (
    scope IN ('org', 'agent', 'project', 'agent_project', 'team', 'team_project')
  ),
  agent_id INTEGER REFERENCES agents (id) ON DELETE CASCADE,
  team_id INTEGER REFERENCES teams (id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects (id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  version INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  author_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  approved_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  note TEXT NOT NULL DEFAULT '',
  proposal_id INTEGER REFERENCES prompt_proposals (id) ON DELETE SET NULL,
  auto_approved INTEGER NOT NULL DEFAULT 0
);
INSERT INTO prompts_old (
  id, scope, agent_id, team_id, project_id, content, version, created_at,
  author_actor_id, approved_by_actor_id, note, proposal_id, auto_approved
)
SELECT
  id, scope, agent_id, team_id, project_id, content, version, created_at,
  author_actor_id, approved_by_actor_id, note, proposal_id, auto_approved
FROM prompts
WHERE scope IN ('org', 'agent', 'project', 'agent_project', 'team', 'team_project');
DROP TABLE prompts;
ALTER TABLE prompts_old RENAME TO prompts;
CREATE INDEX idx_prompts_key
  ON prompts (scope, agent_id, team_id, project_id, version DESC);

DROP INDEX idx_agent_group_logs_actor;
DROP INDEX idx_agent_group_logs_at;
DROP INDEX idx_agent_group_logs_group;
DROP TABLE agent_group_logs;
DROP INDEX idx_agent_group_member_requests_group;
DROP INDEX idx_agent_group_member_requests_agent;
DROP INDEX idx_agent_group_member_requests_pending;
DROP TABLE agent_group_member_requests;
DROP INDEX idx_agent_group_owners_actor;
DROP TABLE agent_group_owners;
DROP INDEX idx_agent_group_members_agent;
DROP TABLE agent_group_members;
DROP INDEX idx_agent_groups_archived;
DROP TABLE agent_groups;
