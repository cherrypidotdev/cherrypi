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
-- Teams group actors (users and agents) and are the primary way project access
-- is granted; direct project grants (phase 5) are the exception. A team is
-- flat apart from a membership role: the creator is the first `owner`, every
-- other member is a plain `member`. Any member may invite an actor; a user
-- accepts their own invitation, and an agent's owner approves it (admins may
-- override either).
CREATE TABLE teams (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL COLLATE NOCASE,
  description TEXT NOT NULL DEFAULT '',
  -- Reserved for a team avatar; set through the attachment surface.
  picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL,
  created_by_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_teams_name ON teams (name);
CREATE INDEX idx_teams_created_by ON teams (created_by_actor_id);

CREATE TABLE team_members (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
  joined_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_team_members_unique ON team_members (team_id, actor_id);
CREATE INDEX idx_team_members_actor ON team_members (actor_id);
CREATE INDEX idx_team_members_team ON team_members (team_id);

-- A pending invitation reserves the (team, actor) pair; one row per pair can
-- be pending at a time. `status` moves pending -> accepted / declined and a
-- revoked invite keeps its row for the log.
CREATE TABLE team_invitations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  invited_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'accepted', 'declined', 'revoked')),
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  responded_at INTEGER,
  responded_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX idx_team_invitations_pending
  ON team_invitations (team_id, actor_id) WHERE status = 'pending';
CREATE INDEX idx_team_invitations_actor ON team_invitations (actor_id, status);
CREATE INDEX idx_team_invitations_team ON team_invitations (team_id, status);
CREATE INDEX idx_team_invitations_expires ON team_invitations (expires_at);

-- Agent ownership: an agent actor may be owned by any actor kind. Owners may
-- edit the agent, approve its team invitations and manage other owners; an
-- agent with no owner is admin-only.
CREATE TABLE agent_owners (
  agent_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  owner_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (agent_actor_id, owner_actor_id)
);
CREATE INDEX idx_agent_owners_owner ON agent_owners (owner_actor_id);

-- +goose Down
DROP TABLE agent_owners;
DROP TABLE team_invitations;
DROP TABLE team_members;
DROP TABLE teams;
