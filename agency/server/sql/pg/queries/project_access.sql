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







-- name: UpsertProjectActorGrant :exec
INSERT INTO project_actor_memberships (
  project_id, actor_id, role, granted_by_actor_id, created_at, updated_at
)
VALUES (
  sqlc.arg(project_id), sqlc.arg(actor_id), sqlc.arg(role),
  sqlc.narg(granted_by_actor_id), sqlc.arg(created_at), sqlc.arg(updated_at)
)
ON CONFLICT (project_id, actor_id) DO UPDATE SET
  role = excluded.role,
  granted_by_actor_id = excluded.granted_by_actor_id,
  updated_at = excluded.updated_at;

-- name: UpdateProjectActorGrantRole :exec
UPDATE project_actor_memberships
SET role = sqlc.arg(role), updated_at = sqlc.arg(updated_at)
WHERE project_id = sqlc.arg(project_id) AND actor_id = sqlc.arg(actor_id);

-- name: RemoveProjectActorGrant :exec
DELETE FROM project_actor_memberships
WHERE project_id = sqlc.arg(project_id) AND actor_id = sqlc.arg(actor_id);

-- name: GetProjectActorGrant :one
SELECT * FROM project_actor_memberships
WHERE project_id = sqlc.arg(project_id) AND actor_id = sqlc.arg(actor_id);

-- name: ListProjectActorGrants :many
SELECT
  m.*,
  a.kind AS actor_kind,
  a.username AS actor_username,
  a.name AS actor_name,
  a.description AS actor_description
FROM project_actor_memberships m
JOIN actors a ON a.id = m.actor_id
WHERE m.project_id = sqlc.arg(project_id)
ORDER BY lower(a.username);

-- name: ListProjectActorMembershipsForActor :many
SELECT
  m.*,
  p.name AS project_name,
  p.slug AS project_slug
FROM project_actor_memberships m
JOIN projects p ON p.id = m.project_id
WHERE m.actor_id = sqlc.arg(actor_id)
ORDER BY lower(p.name);



-- name: UpsertProjectTeamMembership :exec
INSERT INTO project_team_memberships (
  project_id, team_id, role, granted_by_actor_id, created_at, updated_at
)
VALUES (
  sqlc.arg(project_id), sqlc.arg(team_id), sqlc.arg(role),
  sqlc.narg(granted_by_actor_id), sqlc.arg(created_at), sqlc.arg(updated_at)
)
ON CONFLICT (project_id, team_id) DO UPDATE SET
  role = excluded.role,
  granted_by_actor_id = excluded.granted_by_actor_id,
  updated_at = excluded.updated_at;

-- name: UpdateProjectTeamMembershipRole :exec
UPDATE project_team_memberships
SET role = sqlc.arg(role), updated_at = sqlc.arg(updated_at)
WHERE project_id = sqlc.arg(project_id) AND team_id = sqlc.arg(team_id);

-- name: RemoveProjectTeamMembership :exec
DELETE FROM project_team_memberships
WHERE project_id = sqlc.arg(project_id) AND team_id = sqlc.arg(team_id);

-- name: GetProjectTeamMembership :one
SELECT * FROM project_team_memberships
WHERE project_id = sqlc.arg(project_id) AND team_id = sqlc.arg(team_id);

-- name: ListProjectTeamMemberships :many
SELECT
  m.*,
  t.name AS team_name,
  t.description AS team_description,
  (SELECT COUNT(*) FROM team_members tm WHERE tm.team_id = m.team_id)
    AS member_count
FROM project_team_memberships m
JOIN teams t ON t.id = m.team_id
WHERE m.project_id = sqlc.arg(project_id)
ORDER BY lower(t.name);

-- name: ListProjectTeamMembershipsForTeam :many
SELECT
  m.*,
  p.name AS project_name,
  p.slug AS project_slug
FROM project_team_memberships m
JOIN projects p ON p.id = m.project_id
WHERE m.team_id = sqlc.arg(team_id)
ORDER BY lower(p.name);

-- name: CountProjectsOwnedByTeam :one
SELECT COUNT(*) FROM project_team_memberships
WHERE team_id = sqlc.arg(team_id) AND role = 'owner';



-- name: CreateProjectTeamInvitation :one
INSERT INTO project_team_invitations (
  project_id, team_id, invited_by_actor_id, role, note, created_at, expires_at
)
VALUES (
  sqlc.arg(project_id), sqlc.arg(team_id), sqlc.narg(invited_by_actor_id),
  sqlc.arg(role), sqlc.arg(note), sqlc.arg(created_at), sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetProjectTeamInvitation :one
SELECT * FROM project_team_invitations WHERE id = sqlc.arg(id);

-- name: GetPendingProjectTeamInvitation :one
SELECT * FROM project_team_invitations
WHERE project_id = sqlc.arg(project_id) AND team_id = sqlc.arg(team_id)
  AND status = 'pending';

-- name: ListProjectTeamInvitations :many
SELECT
  i.*,
  t.name AS team_name,
  t.description AS team_description,
  inviter.username AS invited_by_username,
  responder.username AS responded_by_username
FROM project_team_invitations i
JOIN teams t ON t.id = i.team_id
LEFT JOIN actors inviter ON inviter.id = i.invited_by_actor_id
LEFT JOIN actors responder ON responder.id = i.responded_by_actor_id
WHERE i.project_id = sqlc.arg(project_id)
ORDER BY i.created_at DESC, i.id DESC;

-- name: ListPendingProjectInvitationsForTeamOwner :many
SELECT
  i.*,
  p.name AS project_name,
  p.slug AS project_slug,
  t.name AS team_name,
  inviter.username AS invited_by_username
FROM project_team_invitations i
JOIN projects p ON p.id = i.project_id
JOIN teams t ON t.id = i.team_id
JOIN team_members tm
  ON tm.team_id = i.team_id AND tm.actor_id = sqlc.arg(actor_id) AND tm.role = 'owner'
LEFT JOIN actors inviter ON inviter.id = i.invited_by_actor_id
WHERE i.status = 'pending'
ORDER BY i.created_at DESC, i.id DESC;

-- name: RespondProjectTeamInvitation :execrows
UPDATE project_team_invitations
SET status = sqlc.arg(status),
    responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: RevokeProjectTeamInvitation :execrows
UPDATE project_team_invitations
SET status = 'revoked',
    responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: ExpireProjectTeamInvitations :execrows
UPDATE project_team_invitations
SET status = 'revoked', responded_at = sqlc.arg(responded_at)
WHERE status = 'pending' AND expires_at < sqlc.arg(now);



-- name: GetEffectiveProjectRole :one
SELECT COALESCE(MAX(weight), 0) AS weight
FROM (
  
  SELECT CASE m.role
      WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight
  FROM project_actor_memberships m
  WHERE m.project_id = sqlc.arg(project_id) AND m.actor_id = sqlc.arg(actor_id)
  UNION ALL
  SELECT CASE ptm.role
      WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight
  FROM project_team_memberships ptm
  JOIN team_members tm ON tm.team_id = ptm.team_id
  WHERE ptm.project_id = sqlc.arg(project_id) AND tm.actor_id = sqlc.arg(actor_id)
  UNION ALL
  
  SELECT 1 AS weight
  FROM project_ancestors pa
  WHERE pa.project_id = sqlc.arg(project_id)
    AND (
      EXISTS (
        SELECT 1 FROM project_actor_memberships m
        WHERE m.project_id = pa.ancestor_id AND m.actor_id = sqlc.arg(actor_id)
      )
      OR EXISTS (
        SELECT 1 FROM project_team_memberships ptm
        JOIN team_members tm ON tm.team_id = ptm.team_id
        WHERE ptm.project_id = pa.ancestor_id AND tm.actor_id = sqlc.arg(actor_id)
      )
    )
);

-- name: ListProjectsForActor :many
WITH roots AS (
  SELECT m.project_id,
    CASE m.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight
  FROM project_actor_memberships m
  WHERE m.actor_id = sqlc.arg(actor_id)
  UNION ALL
  SELECT ptm.project_id,
    CASE ptm.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight
  FROM project_team_memberships ptm
  JOIN team_members tm ON tm.team_id = ptm.team_id
  WHERE tm.actor_id = sqlc.arg(actor_id)
),
readable AS (
  SELECT project_id, weight FROM roots
  UNION ALL
  
  SELECT pa.project_id, 1 AS weight
  FROM project_ancestors pa
  JOIN roots ON roots.project_id = pa.ancestor_id
),
best AS (
  SELECT project_id, MAX(weight) AS weight FROM readable GROUP BY project_id
)
SELECT p.*
FROM projects p
JOIN best ON best.project_id = p.id
ORDER BY lower(p.name);

-- name: ListAgentProjects :many
WITH agent_actor AS (
  SELECT a.actor_id FROM agents a WHERE a.id = sqlc.arg(agent_id)
),
roots AS (
  SELECT m.project_id,
    CASE m.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight
  FROM project_actor_memberships m
  JOIN agent_actor ON agent_actor.actor_id = m.actor_id
  UNION ALL
  SELECT ptm.project_id,
    CASE ptm.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight
  FROM project_team_memberships ptm
  JOIN team_members tm ON tm.team_id = ptm.team_id
  JOIN agent_actor ON agent_actor.actor_id = tm.actor_id
),
readable AS (
  SELECT project_id, weight FROM roots
  UNION ALL
  SELECT pa.project_id, 1 AS weight
  FROM project_ancestors pa
  JOIN roots ON roots.project_id = pa.ancestor_id
),
best AS (
  SELECT project_id, MAX(weight) AS weight FROM readable GROUP BY project_id
)
SELECT p.*
FROM projects p
JOIN best ON best.project_id = p.id
ORDER BY lower(p.name);

-- name: ListAgentProjectMemberships :many
WITH agent_actor AS (
  SELECT a.actor_id FROM agents a WHERE a.id = sqlc.arg(agent_id)
),
roots AS (
  SELECT m.project_id,
    CASE m.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight
  FROM project_actor_memberships m
  JOIN agent_actor ON agent_actor.actor_id = m.actor_id
  UNION ALL
  SELECT ptm.project_id,
    CASE ptm.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight
  FROM project_team_memberships ptm
  JOIN team_members tm ON tm.team_id = ptm.team_id
  JOIN agent_actor ON agent_actor.actor_id = tm.actor_id
),
readable AS (
  SELECT project_id, weight FROM roots
  UNION ALL
  SELECT pa.project_id, 1 AS weight
  FROM project_ancestors pa
  JOIN roots ON roots.project_id = pa.ancestor_id
),
best AS (
  SELECT project_id, MAX(weight) AS weight FROM readable GROUP BY project_id
)
SELECT
  p.id, p.name, p.slug, p.prefix, p.description, p.archived_at,
  p.parent_project_id, p.created_at, p.updated_at,
  CASE best.weight WHEN 3 THEN 'owner' WHEN 2 THEN 'member' ELSE 'viewer' END AS role,
  CAST(1 AS BIGINT) AS can_list_tasks,
  CAST(CASE WHEN best.weight >= 2 THEN 1 ELSE 0 END AS BIGINT) AS can_create_tasks,
  CAST(CASE WHEN best.weight >= 2 THEN 1 ELSE 0 END AS BIGINT) AS can_update_tasks
FROM projects p
JOIN best ON best.project_id = p.id
ORDER BY lower(p.name);

-- name: ListProjectMemberActorIDs :many
SELECT DISTINCT actor_id FROM (
  SELECT m.actor_id FROM project_actor_memberships m
  WHERE m.project_id = sqlc.arg(project_id)
  UNION ALL
  SELECT tm.actor_id FROM project_team_memberships ptm
  JOIN team_members tm ON tm.team_id = ptm.team_id
  WHERE ptm.project_id = sqlc.arg(project_id)
)
ORDER BY actor_id;





-- name: DeleteProjectTeamMembershipsByProject :exec
DELETE FROM project_team_memberships WHERE project_id = sqlc.arg(project_id);

-- name: DeleteProjectActorGrantsByProject :exec
DELETE FROM project_actor_memberships WHERE project_id = sqlc.arg(project_id);

-- name: DeleteProjectTeamInvitationsByProject :exec
DELETE FROM project_team_invitations WHERE project_id = sqlc.arg(project_id);

-- name: DeleteProjectTeamMembershipsByTeam :exec
DELETE FROM project_team_memberships WHERE team_id = sqlc.arg(team_id);

-- name: DeleteProjectTeamInvitationsByTeam :exec
DELETE FROM project_team_invitations WHERE team_id = sqlc.arg(team_id);

-- name: DeleteProjectActorGrantsByActor :exec
DELETE FROM project_actor_memberships WHERE actor_id = sqlc.arg(actor_id);
