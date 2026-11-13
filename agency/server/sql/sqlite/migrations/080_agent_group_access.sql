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
-- Agent groups as access principals. A group invited to a team or project









--  * A project/team owner invites a group; a group owner accepts. An invitation

CREATE TABLE team_group_memberships (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  granted_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_team_group_memberships_unique
  ON team_group_memberships (team_id, group_id);
CREATE INDEX idx_team_group_memberships_group ON team_group_memberships (group_id);

CREATE TABLE team_group_invitations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  invited_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'accepted', 'declined', 'revoked')),
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  responded_at INTEGER,
  responded_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX idx_team_group_invitations_pending
  ON team_group_invitations (team_id, group_id) WHERE status = 'pending';
CREATE INDEX idx_team_group_invitations_group ON team_group_invitations (group_id, status);
CREATE INDEX idx_team_group_invitations_team ON team_group_invitations (team_id, status);

CREATE TABLE project_group_memberships (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('viewer', 'member')),
  granted_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_project_group_memberships_unique
  ON project_group_memberships (project_id, group_id);
CREATE INDEX idx_project_group_memberships_group
  ON project_group_memberships (group_id);

CREATE TABLE project_group_invitations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  group_id INTEGER NOT NULL REFERENCES agent_groups (id) ON DELETE CASCADE,
  invited_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('viewer', 'member')),
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'accepted', 'declined', 'revoked')),
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  responded_at INTEGER,
  responded_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX idx_project_group_invitations_pending
  ON project_group_invitations (project_id, group_id) WHERE status = 'pending';
CREATE INDEX idx_project_group_invitations_group
  ON project_group_invitations (group_id, status);
CREATE INDEX idx_project_group_invitations_project
  ON project_group_invitations (project_id, status);




CREATE VIEW effective_team_members AS
SELECT
  CAST(MIN(m.id) AS INTEGER) AS id,
  m.team_id AS team_id,
  m.actor_id AS actor_id,
  CASE MAX(CASE m.role WHEN 'owner' THEN 2 ELSE 1 END)
    WHEN 2 THEN 'owner' ELSE 'member' END AS role,
  CAST(MIN(m.joined_at) AS INTEGER) AS joined_at,
  CAST(CASE WHEN SUM(CASE WHEN m.via_group_id = 0 THEN 1 ELSE 0 END) > 0
    THEN 0 ELSE MIN(m.via_group_id) END AS INTEGER) AS via_group_id
FROM (
  SELECT
    tm.id AS id,
    tm.team_id AS team_id,
    tm.actor_id AS actor_id,
    tm.role AS role,
    tm.joined_at AS joined_at,
    CAST(0 AS INTEGER) AS via_group_id
  FROM team_members tm
  UNION ALL
  SELECT
    CAST(0 AS INTEGER) AS id,
    tgm.team_id AS team_id,
    a.actor_id AS actor_id,
    'member' AS role,
    tgm.created_at AS joined_at,
    tgm.group_id AS via_group_id
  FROM team_group_memberships tgm
  JOIN agent_group_members agm ON agm.group_id = tgm.group_id
  JOIN agents a ON a.id = agm.agent_id
) AS m
GROUP BY m.team_id, m.actor_id;



CREATE VIEW effective_project_grants AS
SELECT
  m.project_id AS project_id,
  m.actor_id AS actor_id,
  CASE m.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight,
  m.project_id AS source_project_id
FROM project_actor_memberships m
UNION ALL
SELECT
  ptm.project_id AS project_id,
  etm.actor_id AS actor_id,
  CASE ptm.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight,
  ptm.project_id AS source_project_id
FROM project_team_memberships ptm
JOIN effective_team_members etm ON etm.team_id = ptm.team_id
UNION ALL
SELECT
  pgm.project_id AS project_id,
  a.actor_id AS actor_id,
  CASE pgm.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight,
  pgm.project_id AS source_project_id
FROM project_group_memberships pgm
JOIN agent_group_members agm ON agm.group_id = pgm.group_id
JOIN agents a ON a.id = agm.agent_id;

-- +goose Down
DROP VIEW effective_project_grants;
DROP VIEW effective_team_members;

DROP INDEX idx_project_group_invitations_project;
DROP INDEX idx_project_group_invitations_group;
DROP INDEX idx_project_group_invitations_pending;
DROP TABLE project_group_invitations;

DROP INDEX idx_project_group_memberships_group;
DROP INDEX idx_project_group_memberships_unique;
DROP TABLE project_group_memberships;

DROP INDEX idx_team_group_invitations_team;
DROP INDEX idx_team_group_invitations_group;
DROP INDEX idx_team_group_invitations_pending;
DROP TABLE team_group_invitations;

DROP INDEX idx_team_group_memberships_group;
DROP INDEX idx_team_group_memberships_unique;
DROP TABLE team_group_memberships;
