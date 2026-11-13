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







-- name: CreateAgentGroup :one
INSERT INTO agent_groups (
  name, slug, description, picture_attachment_id, created_at, updated_at
)
VALUES (
  sqlc.arg(name), sqlc.arg(slug), sqlc.arg(description),
  sqlc.narg(picture_attachment_id), sqlc.arg(created_at), sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetAgentGroup :one
SELECT * FROM agent_groups WHERE id = sqlc.arg(id);

-- name: GetAgentGroupBySlug :one
SELECT * FROM agent_groups WHERE lower(slug) = lower(sqlc.arg(slug));

-- name: ListAgentGroups :many
SELECT
  g.*,
  (SELECT COUNT(*) FROM agent_group_members m WHERE m.group_id = g.id) AS member_count
FROM agent_groups g
ORDER BY lower(g.name);

-- name: ListAgentGroupsForAgent :many
SELECT
  g.*,
  (SELECT COUNT(*) FROM agent_group_members m WHERE m.group_id = g.id) AS member_count
FROM agent_groups g
JOIN agent_group_members m ON m.group_id = g.id
WHERE m.agent_id = sqlc.arg(agent_id)
ORDER BY g.id;

-- name: UpdateAgentGroup :one
UPDATE agent_groups
SET name = sqlc.arg(name), slug = sqlc.arg(slug),
    description = sqlc.arg(description), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SetAgentGroupArchived :one
UPDATE agent_groups
SET archived_at = sqlc.narg(archived_at), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SetAgentGroupPicture :exec
UPDATE agent_groups
SET picture_attachment_id = sqlc.arg(picture_attachment_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: ClearAgentGroupPicture :exec
UPDATE agent_groups
SET picture_attachment_id = NULL, updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DeleteAgentGroup :exec
DELETE FROM agent_groups WHERE id = sqlc.arg(id);



-- name: AddAgentGroupMember :exec
INSERT INTO agent_group_members (group_id, agent_id, added_by_actor_id, created_at)
VALUES (
  sqlc.arg(group_id), sqlc.arg(agent_id), sqlc.narg(added_by_actor_id),
  sqlc.arg(created_at)
)
ON CONFLICT (group_id, agent_id) DO NOTHING;

-- name: RemoveAgentGroupMember :execrows
DELETE FROM agent_group_members
WHERE group_id = sqlc.arg(group_id) AND agent_id = sqlc.arg(agent_id);

-- name: DeleteAgentGroupMembers :exec
DELETE FROM agent_group_members WHERE group_id = sqlc.arg(group_id);

-- name: DeleteAgentGroupMembershipsByAgent :exec
DELETE FROM agent_group_members WHERE agent_id = sqlc.arg(agent_id);

-- name: IsAgentGroupMember :one
SELECT COUNT(*) FROM agent_group_members
WHERE group_id = sqlc.arg(group_id) AND agent_id = sqlc.arg(agent_id);

-- name: GetAgentGroupMember :one
SELECT * FROM agent_group_members
WHERE group_id = sqlc.arg(group_id) AND agent_id = sqlc.arg(agent_id);

-- name: CountAgentGroupMembers :one
SELECT COUNT(*) FROM agent_group_members WHERE group_id = sqlc.arg(group_id);

-- name: ListAgentGroupMembers :many
SELECT
  m.group_id, m.agent_id, m.added_by_actor_id, m.created_at,
  a.actor_id,
  act.username AS agent_username,
  act.name AS agent_name,
  act.description AS agent_description,
  act.picture_attachment_id AS agent_picture_attachment_id,
  g.last_seen_at,
  ab.username AS added_by_username,
  ab.name AS added_by_name
FROM agent_group_members m
JOIN agents a ON a.id = m.agent_id
JOIN actors act ON act.id = a.actor_id
LEFT JOIN agents g ON g.id = m.agent_id
LEFT JOIN actors ab ON ab.id = m.added_by_actor_id
WHERE m.group_id = sqlc.arg(group_id)
ORDER BY lower(act.username);

-- name: ListAgentGroupMembersForAgent :many
SELECT
  m.group_id, m.agent_id, m.added_by_actor_id, m.created_at,
  a.actor_id,
  act.username AS agent_username,
  act.name AS agent_name,
  act.description AS agent_description,
  act.picture_attachment_id AS agent_picture_attachment_id,
  g.last_seen_at,
  ab.username AS added_by_username,
  ab.name AS added_by_name
FROM agent_group_members m
JOIN agents a ON a.id = m.agent_id
JOIN actors act ON act.id = a.actor_id
LEFT JOIN agents g ON g.id = m.agent_id
LEFT JOIN actors ab ON ab.id = m.added_by_actor_id
WHERE m.agent_id = sqlc.arg(agent_id)
ORDER BY m.group_id;

-- name: ListAllAgentGroupMembers :many
SELECT
  m.group_id, m.agent_id, m.added_by_actor_id, m.created_at,
  a.actor_id,
  act.username AS agent_username,
  act.name AS agent_name,
  act.description AS agent_description,
  act.picture_attachment_id AS agent_picture_attachment_id,
  g.last_seen_at,
  ab.username AS added_by_username,
  ab.name AS added_by_name
FROM agent_group_members m
JOIN agents a ON a.id = m.agent_id
JOIN actors act ON act.id = a.actor_id
LEFT JOIN agents g ON g.id = m.agent_id
LEFT JOIN actors ab ON ab.id = m.added_by_actor_id
ORDER BY m.group_id, lower(act.username);



-- name: AddAgentGroupOwner :exec
INSERT INTO agent_group_owners (group_id, actor_id, created_at)
VALUES (sqlc.arg(group_id), sqlc.arg(actor_id), sqlc.arg(created_at))
ON CONFLICT (group_id, actor_id) DO NOTHING;

-- name: RemoveAgentGroupOwner :execrows
DELETE FROM agent_group_owners
WHERE group_id = sqlc.arg(group_id) AND actor_id = sqlc.arg(actor_id);

-- name: CountAgentGroupOwners :one
SELECT COUNT(*) FROM agent_group_owners WHERE group_id = sqlc.arg(group_id);

-- name: IsAgentGroupOwner :one
SELECT COUNT(*) FROM agent_group_owners
WHERE group_id = sqlc.arg(group_id) AND actor_id = sqlc.arg(actor_id);

-- name: ListAgentGroupOwners :many
SELECT
  o.group_id, o.actor_id, o.created_at,
  a.kind AS actor_kind,
  a.username AS actor_username,
  a.name AS actor_name,
  a.description AS actor_description,
  a.picture_attachment_id AS actor_picture_attachment_id
FROM agent_group_owners o
JOIN actors a ON a.id = o.actor_id
WHERE o.group_id = sqlc.arg(group_id)
ORDER BY lower(a.username);

-- name: ListAgentGroupIDsOwnedBy :many
SELECT group_id FROM agent_group_owners WHERE actor_id = sqlc.arg(actor_id);

-- name: DeleteAgentGroupOwnerEdgesByActor :exec

DELETE FROM agent_group_owners WHERE actor_id = sqlc.arg(actor_id);

-- name: DeleteAgentGroupMembershipsByActor :exec

DELETE FROM agent_group_members
WHERE agent_id IN (SELECT id FROM agents WHERE actor_id = sqlc.arg(actor_id));

-- name: DeleteAgentGroupMemberRequestsByActor :exec
DELETE FROM agent_group_member_requests
WHERE requested_by_actor_id = sqlc.arg(actor_id)
  OR agent_id IN (SELECT id FROM agents WHERE actor_id = sqlc.arg(actor_id));



-- name: CreateAgentGroupMemberRequest :one
INSERT INTO agent_group_member_requests (
  group_id, agent_id, requested_by_actor_id, note, created_at, expires_at
)
VALUES (
  sqlc.arg(group_id), sqlc.arg(agent_id), sqlc.arg(requested_by_actor_id),
  sqlc.arg(note), sqlc.arg(created_at), sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetAgentGroupMemberRequest :one
SELECT * FROM agent_group_member_requests WHERE id = sqlc.arg(id);

-- name: GetPendingAgentGroupMemberRequest :one
SELECT * FROM agent_group_member_requests
WHERE group_id = sqlc.arg(group_id)
  AND agent_id = sqlc.arg(agent_id)
  AND status = 'pending';

-- name: ListAgentGroupMemberRequests :many
SELECT
  r.*,
  g.name AS group_name,
  g.slug AS group_slug,
  act.username AS agent_username,
  act.name AS agent_name,
  rb.username AS requested_by_username,
  rb.name AS requested_by_name,
  db.username AS responded_by_username,
  db.name AS responded_by_name
FROM agent_group_member_requests r
JOIN agent_groups g ON g.id = r.group_id
JOIN agents a ON a.id = r.agent_id
JOIN actors act ON act.id = a.actor_id
LEFT JOIN actors rb ON rb.id = r.requested_by_actor_id
LEFT JOIN actors db ON db.id = r.responded_by_actor_id
WHERE r.group_id = sqlc.arg(group_id)
  AND r.status = sqlc.arg(status)
ORDER BY r.created_at DESC;

-- name: ListPendingAgentGroupMemberRequestsForAgent :many
SELECT
  r.*,
  g.name AS group_name,
  g.slug AS group_slug,
  act.username AS agent_username,
  act.name AS agent_name,
  rb.username AS requested_by_username,
  rb.name AS requested_by_name,
  db.username AS responded_by_username,
  db.name AS responded_by_name
FROM agent_group_member_requests r
JOIN agent_groups g ON g.id = r.group_id
JOIN agents a ON a.id = r.agent_id
JOIN actors act ON act.id = a.actor_id
LEFT JOIN actors rb ON rb.id = r.requested_by_actor_id
LEFT JOIN actors db ON db.id = r.responded_by_actor_id
WHERE r.agent_id = sqlc.arg(agent_id) AND r.status = 'pending'
ORDER BY r.created_at DESC;

-- name: RespondAgentGroupMemberRequest :execrows
UPDATE agent_group_member_requests
SET status = sqlc.arg(status), responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: ExpireAgentGroupMemberRequests :execrows
UPDATE agent_group_member_requests
SET status = 'revoked', responded_at = sqlc.arg(responded_at)
WHERE status = 'pending' AND expires_at < sqlc.arg(now);

-- name: DeleteAgentGroupMemberRequestsByGroup :exec
DELETE FROM agent_group_member_requests WHERE group_id = sqlc.arg(group_id);

-- name: DeleteAgentGroupMemberRequestsByAgent :exec
DELETE FROM agent_group_member_requests WHERE agent_id = sqlc.arg(agent_id);



-- name: ListAgentGroupMemberProjects :many



WITH roots AS (
  SELECT g.actor_id, g.project_id
  FROM effective_project_grants g
  WHERE g.actor_id IN (
    SELECT a.actor_id FROM agent_group_members gm
    JOIN agents a ON a.id = gm.agent_id
    WHERE gm.group_id = sqlc.arg(group_id)
  )
),
reach AS (
  SELECT actor_id, project_id FROM roots
  UNION
  SELECT r.actor_id, pa.project_id
  FROM roots r
  JOIN project_ancestors pa ON pa.ancestor_id = r.project_id
)
SELECT
  p.id, p.name, p.slug, p.prefix, p.description, p.archived_at,
  p.parent_project_id, p.created_at, p.updated_at, p.archive_after_days,
  COUNT(DISTINCT reach.actor_id) AS member_count
FROM projects p
JOIN reach ON reach.project_id = p.id
GROUP BY p.id, p.name, p.slug, p.prefix, p.description, p.archived_at,
  p.parent_project_id, p.created_at, p.updated_at, p.archive_after_days
ORDER BY lower(p.name);

-- name: ListAgentGroupMemberTeams :many



WITH roots AS (
  SELECT g.actor_id, g.project_id
  FROM effective_project_grants g
  WHERE g.actor_id IN (
    SELECT a.actor_id FROM agent_group_members gm
    JOIN agents a ON a.id = gm.agent_id
    WHERE gm.group_id = sqlc.arg(group_id)
  )
),
reach AS (
  SELECT actor_id, project_id FROM roots
  UNION
  SELECT r.actor_id, pa.project_id
  FROM roots r
  JOIN project_ancestors pa ON pa.ancestor_id = r.project_id
),
attached AS (
  SELECT ptm.team_id, ptm.project_id
  FROM project_team_memberships ptm
  JOIN reach ON reach.project_id = ptm.project_id
  UNION
  SELECT ptm.team_id, pa.project_id
  FROM project_team_memberships ptm
  JOIN project_ancestors pa ON pa.ancestor_id = ptm.project_id
  JOIN reach ON reach.project_id = pa.project_id
)
SELECT t.id AS team_id, t.name AS team_name, attached.project_id
FROM teams t
JOIN attached ON attached.team_id = t.id
GROUP BY t.id, t.name, attached.project_id
ORDER BY lower(t.name), attached.project_id;



-- name: UpsertProjectGroupMembership :exec
INSERT INTO project_group_memberships (
  project_id, group_id, role, granted_by_actor_id, created_at, updated_at
)
VALUES (
  sqlc.arg(project_id), sqlc.arg(group_id), sqlc.arg(role),
  sqlc.narg(granted_by_actor_id), sqlc.arg(created_at), sqlc.arg(updated_at)
)
ON CONFLICT (project_id, group_id) DO UPDATE SET
  role = excluded.role,
  granted_by_actor_id = excluded.granted_by_actor_id,
  updated_at = excluded.updated_at;

-- name: GetProjectGroupMembership :one
SELECT * FROM project_group_memberships
WHERE project_id = sqlc.arg(project_id) AND group_id = sqlc.arg(group_id);

-- name: DeleteProjectGroupMembership :execrows
DELETE FROM project_group_memberships
WHERE project_id = sqlc.arg(project_id) AND group_id = sqlc.arg(group_id);

-- name: DeleteProjectGroupMembershipsByGroup :exec
DELETE FROM project_group_memberships WHERE group_id = sqlc.arg(group_id);

-- name: DeleteProjectGroupMembershipsByProject :exec
DELETE FROM project_group_memberships WHERE project_id = sqlc.arg(project_id);

-- name: GetEffectiveProjectGroupRole :one


SELECT CAST(COALESCE(MAX(weight), 0) AS BIGINT) AS weight
FROM (
  SELECT CASE pgm.role
      WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight
  FROM project_ancestors pa
  JOIN project_group_memberships pgm ON pgm.project_id = pa.ancestor_id
  WHERE pa.project_id = sqlc.arg(project_id)
    AND pgm.group_id = sqlc.arg(group_id)
) AS inherited_roles;

-- name: ListProjectGroupLinks :many
SELECT
  m.group_id, m.role, m.granted_by_actor_id, m.created_at, m.updated_at,
  g.name AS group_name,
  g.slug AS group_slug,
  g.description AS group_description,
  (SELECT COUNT(*) FROM agent_group_members agm WHERE agm.group_id = m.group_id)
    AS member_count,
  a.username AS granted_by_username,
  a.name AS granted_by_name
FROM project_group_memberships m
JOIN agent_groups g ON g.id = m.group_id
LEFT JOIN actors a ON a.id = m.granted_by_actor_id
WHERE m.project_id = sqlc.arg(project_id)
ORDER BY lower(g.name);

-- name: ListInheritedProjectGroupLinks :many


WITH candidates AS (
  SELECT
    pgm.group_id AS group_id,
    CASE pgm.role WHEN 'owner' THEN 3 WHEN 'member' THEN 2 ELSE 1 END AS weight,
    pa.depth AS depth,
    pa.ancestor_id AS source_project_id
  FROM project_ancestors pa
  JOIN project_group_memberships pgm ON pgm.project_id = pa.ancestor_id
  WHERE pa.project_id = sqlc.arg(project_id)
),
best AS (
  SELECT group_id, MAX(weight) AS weight FROM candidates GROUP BY group_id
),
chosen AS (
  SELECT c.group_id, c.weight, c.depth, c.source_project_id
  FROM candidates c
  JOIN best b ON b.group_id = c.group_id AND b.weight = c.weight
  WHERE c.depth = (
    SELECT MIN(c2.depth) FROM candidates c2
    WHERE c2.group_id = c.group_id AND c2.weight = c.weight
  )
)
SELECT
  g.id AS group_id,
  g.name AS group_name,
  g.slug AS group_slug,
  g.description AS group_description,
  CAST(chosen.weight AS BIGINT) AS role_weight,
  chosen.source_project_id AS source_project_id,
  source.name AS source_project_name,
  source.slug AS source_project_slug,
  (SELECT COUNT(*) FROM agent_group_members agm WHERE agm.group_id = g.id)
    AS member_count
FROM chosen
JOIN agent_groups g ON g.id = chosen.group_id
JOIN projects source ON source.id = chosen.source_project_id
ORDER BY lower(g.name);

-- name: ListProjectGroupInvitations :many
SELECT
  i.*,
  g.name AS group_name,
  g.slug AS group_slug,
  g.description AS group_description,
  inviter.username AS invited_by_username,
  inviter.name AS invited_by_name,
  responder.username AS responded_by_username,
  responder.name AS responded_by_name
FROM project_group_invitations i
JOIN agent_groups g ON g.id = i.group_id
LEFT JOIN actors inviter ON inviter.id = i.invited_by_actor_id
LEFT JOIN actors responder ON responder.id = i.responded_by_actor_id
WHERE i.project_id = sqlc.arg(project_id)
ORDER BY i.created_at DESC, i.id DESC;

-- name: GetProjectGroupInvitation :one
SELECT * FROM project_group_invitations WHERE id = sqlc.arg(id);

-- name: GetPendingProjectGroupInvitation :one
SELECT * FROM project_group_invitations
WHERE project_id = sqlc.arg(project_id) AND group_id = sqlc.arg(group_id)
  AND status = 'pending';

-- name: CreateProjectGroupInvitation :one
INSERT INTO project_group_invitations (
  project_id, group_id, invited_by_actor_id, role, note, created_at, expires_at
)
VALUES (
  sqlc.arg(project_id), sqlc.arg(group_id), sqlc.narg(invited_by_actor_id),
  sqlc.arg(role), sqlc.arg(note), sqlc.arg(created_at), sqlc.arg(expires_at)
)
RETURNING *;

-- name: RespondProjectGroupInvitation :execrows
UPDATE project_group_invitations
SET status = sqlc.arg(status),
    responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: RevokeProjectGroupInvitation :execrows
UPDATE project_group_invitations
SET status = 'revoked',
    responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: ExpireProjectGroupInvitations :execrows
UPDATE project_group_invitations
SET status = 'revoked', responded_at = sqlc.arg(responded_at)
WHERE status = 'pending' AND expires_at < sqlc.arg(now);

-- name: DeleteProjectGroupInvitationsByGroup :exec
DELETE FROM project_group_invitations WHERE group_id = sqlc.arg(group_id);

-- name: DeleteProjectGroupInvitationsByProject :exec
DELETE FROM project_group_invitations WHERE project_id = sqlc.arg(project_id);

-- name: ListAgentGroupProjectAccess :many
SELECT
  m.project_id, m.group_id, m.role, m.granted_by_actor_id, m.created_at,
  m.updated_at,
  p.name AS project_name,
  p.slug AS project_slug,
  a.username AS granted_by_username,
  a.name AS granted_by_name
FROM project_group_memberships m
JOIN projects p ON p.id = m.project_id
LEFT JOIN actors a ON a.id = m.granted_by_actor_id
WHERE m.group_id = sqlc.arg(group_id)
ORDER BY lower(p.name);

-- name: ListAgentGroupProjectInvitations :many
SELECT
  i.*,
  p.name AS project_name,
  p.slug AS project_slug,
  inviter.username AS invited_by_username,
  inviter.name AS invited_by_name,
  responder.username AS responded_by_username,
  responder.name AS responded_by_name
FROM project_group_invitations i
JOIN projects p ON p.id = i.project_id
LEFT JOIN actors inviter ON inviter.id = i.invited_by_actor_id
LEFT JOIN actors responder ON responder.id = i.responded_by_actor_id
WHERE i.group_id = sqlc.arg(group_id)
ORDER BY i.created_at DESC, i.id DESC;



-- name: UpsertTeamGroupMembership :exec
INSERT INTO team_group_memberships (
  team_id, group_id, granted_by_actor_id, created_at
)
VALUES (
  sqlc.arg(team_id), sqlc.arg(group_id), sqlc.narg(granted_by_actor_id),
  sqlc.arg(created_at)
)
ON CONFLICT (team_id, group_id) DO UPDATE SET
  granted_by_actor_id = excluded.granted_by_actor_id;

-- name: GetTeamGroupMembership :one
SELECT * FROM team_group_memberships
WHERE team_id = sqlc.arg(team_id) AND group_id = sqlc.arg(group_id);

-- name: DeleteTeamGroupMembership :execrows
DELETE FROM team_group_memberships
WHERE team_id = sqlc.arg(team_id) AND group_id = sqlc.arg(group_id);

-- name: DeleteTeamGroupMembershipsByGroup :exec
DELETE FROM team_group_memberships WHERE group_id = sqlc.arg(group_id);

-- name: DeleteTeamGroupMembershipsByTeam :exec
DELETE FROM team_group_memberships WHERE team_id = sqlc.arg(team_id);

-- name: ListTeamGroupLinks :many
SELECT
  m.group_id, m.granted_by_actor_id, m.created_at,
  g.name AS group_name,
  g.slug AS group_slug,
  g.description AS group_description,
  (SELECT COUNT(*) FROM agent_group_members agm WHERE agm.group_id = m.group_id)
    AS member_count,
  a.username AS granted_by_username,
  a.name AS granted_by_name
FROM team_group_memberships m
JOIN agent_groups g ON g.id = m.group_id
LEFT JOIN actors a ON a.id = m.granted_by_actor_id
WHERE m.team_id = sqlc.arg(team_id)
ORDER BY lower(g.name);

-- name: ListTeamGroupInvitations :many
SELECT
  i.*,
  g.name AS group_name,
  g.slug AS group_slug,
  g.description AS group_description,
  inviter.username AS invited_by_username,
  inviter.name AS invited_by_name,
  responder.username AS responded_by_username,
  responder.name AS responded_by_name
FROM team_group_invitations i
JOIN agent_groups g ON g.id = i.group_id
LEFT JOIN actors inviter ON inviter.id = i.invited_by_actor_id
LEFT JOIN actors responder ON responder.id = i.responded_by_actor_id
WHERE i.team_id = sqlc.arg(team_id)
ORDER BY i.created_at DESC, i.id DESC;

-- name: GetTeamGroupInvitation :one
SELECT * FROM team_group_invitations WHERE id = sqlc.arg(id);

-- name: GetPendingTeamGroupInvitation :one
SELECT * FROM team_group_invitations
WHERE team_id = sqlc.arg(team_id) AND group_id = sqlc.arg(group_id)
  AND status = 'pending';

-- name: CreateTeamGroupInvitation :one
INSERT INTO team_group_invitations (
  team_id, group_id, invited_by_actor_id, note, created_at, expires_at
)
VALUES (
  sqlc.arg(team_id), sqlc.arg(group_id), sqlc.narg(invited_by_actor_id),
  sqlc.arg(note), sqlc.arg(created_at), sqlc.arg(expires_at)
)
RETURNING *;

-- name: RespondTeamGroupInvitation :execrows
UPDATE team_group_invitations
SET status = sqlc.arg(status),
    responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: RevokeTeamGroupInvitation :execrows
UPDATE team_group_invitations
SET status = 'revoked',
    responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: ExpireTeamGroupInvitations :execrows
UPDATE team_group_invitations
SET status = 'revoked', responded_at = sqlc.arg(responded_at)
WHERE status = 'pending' AND expires_at < sqlc.arg(now);

-- name: DeleteTeamGroupInvitationsByGroup :exec
DELETE FROM team_group_invitations WHERE group_id = sqlc.arg(group_id);

-- name: DeleteTeamGroupInvitationsByTeam :exec
DELETE FROM team_group_invitations WHERE team_id = sqlc.arg(team_id);

-- name: ListAgentGroupTeamAccess :many
SELECT
  m.team_id, m.group_id, m.granted_by_actor_id, m.created_at,
  t.name AS team_name,
  t.description AS team_description,
  (SELECT COUNT(*) FROM agent_group_members agm WHERE agm.group_id = m.group_id)
    AS member_count,
  a.username AS granted_by_username,
  a.name AS granted_by_name
FROM team_group_memberships m
JOIN teams t ON t.id = m.team_id
LEFT JOIN actors a ON a.id = m.granted_by_actor_id
WHERE m.group_id = sqlc.arg(group_id)
ORDER BY lower(t.name);

-- name: ListAgentGroupTeamInvitations :many
SELECT
  i.*,
  t.name AS team_name,
  t.description AS team_description,
  inviter.username AS invited_by_username,
  inviter.name AS invited_by_name,
  responder.username AS responded_by_username,
  responder.name AS responded_by_name
FROM team_group_invitations i
JOIN teams t ON t.id = i.team_id
LEFT JOIN actors inviter ON inviter.id = i.invited_by_actor_id
LEFT JOIN actors responder ON responder.id = i.responded_by_actor_id
WHERE i.group_id = sqlc.arg(group_id)
ORDER BY i.created_at DESC, i.id DESC;



-- name: ListProjectOwnerActorIDs :many


SELECT DISTINCT actor_id FROM (
  SELECT m.actor_id FROM project_actor_memberships m
  WHERE m.project_id = sqlc.arg(project_id) AND m.role = 'owner'
  UNION ALL
  SELECT etm.actor_id
  FROM project_team_memberships ptm
  JOIN effective_team_members etm ON etm.team_id = ptm.team_id
  WHERE ptm.project_id = sqlc.arg(project_id)
    AND ptm.role = 'owner'
    AND etm.role = 'owner'
) AS project_owners
ORDER BY actor_id;

-- name: ListTeamOwnerActorIDs :many
SELECT DISTINCT actor_id FROM effective_team_members
WHERE team_id = sqlc.arg(team_id) AND role = 'owner'
ORDER BY actor_id;



-- name: ListLatestGroupPrompts :many


SELECT p.*
FROM prompts p
WHERE coalesce(p.group_id, 0) = sqlc.arg(group_id)
  AND p.version = (
    SELECT MAX(x.version) FROM prompts x
    WHERE x.scope = p.scope
      AND coalesce(x.agent_id, 0) = coalesce(p.agent_id, 0)
      AND coalesce(x.team_id, 0) = coalesce(p.team_id, 0)
      AND coalesce(x.group_id, 0) = coalesce(p.group_id, 0)
      AND coalesce(x.project_id, '') = coalesce(p.project_id, '')
  )
ORDER BY p.scope, p.team_id, p.project_id;



-- name: InsertAgentGroupLog :exec
INSERT INTO agent_group_logs (
  group_id, at, actor_id, kind, field, old_value, new_value, message, data_json
)
VALUES (
  sqlc.arg(group_id), sqlc.arg(at), sqlc.narg(actor_id), sqlc.arg(kind),
  sqlc.arg(field), sqlc.arg(old_value), sqlc.arg(new_value), sqlc.arg(message),
  sqlc.arg(data_json)
);

-- name: ListAgentGroupLogs :many
SELECT
  l.id, l.group_id, l.at, l.actor_id, l.kind, l.field, l.old_value, l.new_value,
  l.message, l.data_json,
  a.kind AS actor_kind,
  a.username AS actor_username,
  a.name AS actor_name
FROM agent_group_logs l
LEFT JOIN actors a ON a.id = l.actor_id
WHERE l.group_id = sqlc.arg(group_id)
  AND l.id < sqlc.arg(before_id)
ORDER BY l.at DESC, l.id DESC
LIMIT CAST(sqlc.arg(max_count) AS BIGINT);



-- name: AddTaskGroupPrompt :exec
INSERT INTO task_group_prompts (task_rowid, group_id, prompt_id, project_prompt_id)
VALUES (
  (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id)),
  sqlc.arg(group_id), sqlc.narg(prompt_id), sqlc.narg(project_prompt_id)
);

-- name: AddTaskGroupTeamPrompt :exec
INSERT INTO task_group_team_prompts (task_rowid, group_id, team_id, prompt_id)
VALUES (
  (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id)),
  sqlc.arg(group_id), sqlc.arg(team_id), sqlc.narg(prompt_id)
);

-- name: DeleteTaskGroupPrompts :exec
DELETE FROM task_group_prompts
WHERE task_rowid = (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id));

-- name: DeleteTaskGroupTeamPrompts :exec
DELETE FROM task_group_team_prompts
WHERE task_rowid = (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id));

-- name: ListTaskGroupPromptsByTask :many
SELECT * FROM task_group_prompts
WHERE task_rowid = (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id))
ORDER BY group_id;

-- name: ListTaskGroupTeamPromptsByTask :many
SELECT * FROM task_group_team_prompts
WHERE task_rowid = (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id))
ORDER BY group_id, team_id;
