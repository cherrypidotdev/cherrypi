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
-- Project access (phase 5). A project's effective role for an actor is the
-- strongest of a direct grant in `project_actor_memberships` (the exception)
-- and the best role among the teams attached through
-- `project_team_memberships`. Roles are ordered viewer < member < owner:
--   viewer  — view and comment; cannot prompt or edit,
--   member  — prompt (create tasks) and edit,
--   owner   — everything, including managing access.
--
-- Team access is consented to: attaching a team is a
-- `project_team_invitations` row that one of the invitee team's owners approves
-- (admins may override). Direct actor grants are added and removed by a project
-- owner without the target's consent.
--
-- `project_agents` carried per-agent task flags; phase 5 replaces them with the
-- role model, so every membership is migrated into a direct grant and the table
-- is dropped. Agents with create/update flags become `member`, the rest
-- `viewer` (list-only).
CREATE TABLE project_team_memberships (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member', 'viewer')),
  granted_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_project_team_memberships_unique
  ON project_team_memberships (project_id, team_id);
CREATE INDEX idx_project_team_memberships_team ON project_team_memberships (team_id);
CREATE INDEX idx_project_team_memberships_project
  ON project_team_memberships (project_id, role);

CREATE TABLE project_actor_memberships (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member', 'viewer')),
  granted_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_project_actor_memberships_unique
  ON project_actor_memberships (project_id, actor_id);
CREATE INDEX idx_project_actor_memberships_actor ON project_actor_memberships (actor_id);
CREATE INDEX idx_project_actor_memberships_project
  ON project_actor_memberships (project_id, role);

-- A pending invitation reserves the (project, team) pair; one row per pair can
-- be pending at a time. `status` moves pending -> accepted / declined and a
-- revoked invite keeps its row for the log. The role is the role the team will
-- hold once accepted.
CREATE TABLE project_team_invitations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  invited_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member', 'viewer')),
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'accepted', 'declined', 'revoked')),
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  responded_at INTEGER,
  responded_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX idx_project_team_invitations_pending
  ON project_team_invitations (project_id, team_id) WHERE status = 'pending';
CREATE INDEX idx_project_team_invitations_team ON project_team_invitations (team_id, status);
CREATE INDEX idx_project_team_invitations_project
  ON project_team_invitations (project_id, status);
CREATE INDEX idx_project_team_invitations_expires ON project_team_invitations (expires_at);

-- Migrate the per-agent memberships into direct grants. `project_agents.agent_id`
-- references `agents.id`, so the grant is keyed by the agent's actor id.
INSERT INTO project_actor_memberships (
  project_id, actor_id, role, created_at, updated_at
)
SELECT
  pa.project_id,
  a.actor_id,
  CASE
    WHEN pa.can_create_tasks = 1 OR pa.can_update_tasks = 1 THEN 'member'
    ELSE 'viewer'
  END,
  pa.created_at,
  pa.created_at
FROM project_agents pa
JOIN agents a ON a.id = pa.agent_id;

DROP TABLE project_agents;

-- +goose Down
-- Rebuild `project_agents` from the direct agent grants, mapping roles back to
-- the old flags, then drop the phase-5 tables.
CREATE TABLE project_agents (
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  agent_id INTEGER NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  can_list_tasks INTEGER NOT NULL DEFAULT 1,
  can_create_tasks INTEGER NOT NULL DEFAULT 0,
  can_update_tasks INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (project_id, agent_id)
);
INSERT INTO project_agents (
  project_id, agent_id, created_at,
  can_list_tasks, can_create_tasks, can_update_tasks
)
SELECT
  m.project_id,
  a.id,
  m.created_at,
  1,
  CASE WHEN m.role IN ('owner', 'member') THEN 1 ELSE 0 END,
  CASE WHEN m.role IN ('owner', 'member') THEN 1 ELSE 0 END
FROM project_actor_memberships m
JOIN agents a ON a.actor_id = m.actor_id;
CREATE INDEX idx_project_agents_agent_id ON project_agents (agent_id);

DROP TABLE project_team_invitations;
DROP TABLE project_actor_memberships;
DROP TABLE project_team_memberships;
