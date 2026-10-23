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

-- Teams, members, invitations and agent owners. See docs/auth.md phase 4.
-- Policy lives in internal/teams; this file is the query surface.

-- name: CreateTeam :one
INSERT INTO teams (
  name, description, picture_attachment_id, created_by_actor_id, created_at,
  updated_at
)
VALUES (
  sqlc.arg(name), sqlc.arg(description), sqlc.narg(picture_attachment_id),
  sqlc.arg(created_by_actor_id), sqlc.arg(created_at), sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams WHERE id = sqlc.arg(id);

-- name: GetTeamByName :one
SELECT * FROM teams WHERE lower(name) = lower(sqlc.arg(name));

-- name: ListTeams :many
SELECT
  t.*,
  (SELECT COUNT(*) FROM team_members m WHERE m.team_id = t.id) AS member_count
FROM teams t
ORDER BY lower(t.name);

-- name: ListTeamsForActor :many
SELECT
  t.*,
  m.role AS member_role,
  (SELECT COUNT(*) FROM team_members tm WHERE tm.team_id = t.id) AS member_count
FROM teams t
JOIN team_members m ON m.team_id = t.id
WHERE m.actor_id = sqlc.arg(actor_id)
ORDER BY lower(t.name);

-- name: UpdateTeam :one
UPDATE teams
SET name = sqlc.arg(name), description = sqlc.arg(description),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SetTeamPicture :exec
UPDATE teams SET picture_attachment_id = sqlc.arg(picture_attachment_id),
                 updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: ClearTeamPicture :exec
UPDATE teams SET picture_attachment_id = NULL, updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = sqlc.arg(id);

-- name: CountTeams :one
SELECT COUNT(*) FROM teams;

-- Members -------------------------------------------------------------------

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, actor_id, role, joined_at)
VALUES (sqlc.arg(team_id), sqlc.arg(actor_id), sqlc.arg(role), sqlc.arg(joined_at))
ON CONFLICT (team_id, actor_id) DO UPDATE SET role = excluded.role;

-- name: RemoveTeamMember :execrows
DELETE FROM team_members
WHERE team_id = sqlc.arg(team_id) AND actor_id = sqlc.arg(actor_id);

-- name: DeleteTeamMembers :exec
DELETE FROM team_members WHERE team_id = sqlc.arg(team_id);

-- name: GetTeamMember :one
SELECT * FROM team_members
WHERE team_id = sqlc.arg(team_id) AND actor_id = sqlc.arg(actor_id);

-- name: ListTeamMembers :many
SELECT
  m.id, m.team_id, m.actor_id, m.role, m.joined_at,
  a.kind AS actor_kind,
  a.username AS actor_username,
  a.name AS actor_name,
  a.description AS actor_description
FROM team_members m
JOIN actors a ON a.id = m.actor_id
WHERE m.team_id = sqlc.arg(team_id)
ORDER BY m.role DESC, lower(a.username);

-- name: CountTeamMembers :one
SELECT COUNT(*) FROM team_members WHERE team_id = sqlc.arg(team_id);

-- name: CountTeamOwners :one
SELECT COUNT(*) FROM team_members
WHERE team_id = sqlc.arg(team_id) AND role = 'owner';

-- Invitations ---------------------------------------------------------------

-- name: CreateTeamInvitation :one
INSERT INTO team_invitations (
  team_id, actor_id, invited_by_actor_id, role, note, created_at, expires_at
)
VALUES (
  sqlc.arg(team_id), sqlc.arg(actor_id), sqlc.narg(invited_by_actor_id),
  sqlc.arg(role), sqlc.arg(note), sqlc.arg(created_at), sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetTeamInvitation :one
SELECT * FROM team_invitations WHERE id = sqlc.arg(id);

-- name: GetPendingTeamInvitation :one
SELECT * FROM team_invitations
WHERE team_id = sqlc.arg(team_id)
  AND actor_id = sqlc.arg(actor_id)
  AND status = 'pending';

-- name: ListTeamInvitations :many
SELECT
  i.*,
  a.kind AS actor_kind,
  a.username AS actor_username,
  a.name AS actor_name,
  ib.username AS invited_by_username,
  rb.username AS responded_by_username
FROM team_invitations i
JOIN actors a ON a.id = i.actor_id
LEFT JOIN actors ib ON ib.id = i.invited_by_actor_id
LEFT JOIN actors rb ON rb.id = i.responded_by_actor_id
WHERE i.team_id = sqlc.arg(team_id)
ORDER BY i.created_at DESC;

-- name: ListPendingTeamInvitationsForActor :many
-- The teams an actor has been invited to and can act on: a user accepts, an
-- agent counts here until one of its owners approves.
SELECT
  i.*,
  t.name AS team_name,
  t.description AS team_description,
  ib.username AS invited_by_username
FROM team_invitations i
JOIN teams t ON t.id = i.team_id
LEFT JOIN actors ib ON ib.id = i.invited_by_actor_id
WHERE i.actor_id = sqlc.arg(actor_id) AND i.status = 'pending'
ORDER BY i.created_at DESC;

-- name: RespondTeamInvitation :execrows
UPDATE team_invitations
SET status = sqlc.arg(status), responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: RevokeTeamInvitation :execrows
UPDATE team_invitations
SET status = 'revoked', responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: ExpireTeamInvitations :execrows
UPDATE team_invitations
SET status = 'revoked', responded_at = sqlc.arg(responded_at)
WHERE status = 'pending' AND expires_at < sqlc.arg(now);

-- name: DeleteTeamInvitationsByTeam :exec
DELETE FROM team_invitations WHERE team_id = sqlc.arg(team_id);

-- name: DeleteTeamMembershipsByActor :exec
-- Foreign keys are off, so deleting an actor removes its memberships and
-- pending invitations explicitly.
DELETE FROM team_members WHERE actor_id = sqlc.arg(actor_id);

-- name: DeleteTeamInvitationsByActor :exec
DELETE FROM team_invitations WHERE actor_id = sqlc.arg(actor_id);

-- Agent owners --------------------------------------------------------------

-- name: AddAgentOwner :exec
INSERT INTO agent_owners (agent_actor_id, owner_actor_id, created_at)
VALUES (sqlc.arg(agent_actor_id), sqlc.arg(owner_actor_id), sqlc.arg(created_at))
ON CONFLICT (agent_actor_id, owner_actor_id) DO NOTHING;

-- name: RemoveAgentOwner :execrows
DELETE FROM agent_owners
WHERE agent_actor_id = sqlc.arg(agent_actor_id)
  AND owner_actor_id = sqlc.arg(owner_actor_id);

-- name: IsAgentOwner :one
SELECT COUNT(*) FROM agent_owners
WHERE agent_actor_id = sqlc.arg(agent_actor_id)
  AND owner_actor_id = sqlc.arg(owner_actor_id);

-- name: CountAgentOwners :one
SELECT COUNT(*) FROM agent_owners WHERE agent_actor_id = sqlc.arg(agent_actor_id);

-- name: ListAgentOwners :many
SELECT
  o.agent_actor_id, o.owner_actor_id, o.created_at,
  a.kind AS actor_kind,
  a.username AS actor_username,
  a.name AS actor_name
FROM agent_owners o
JOIN actors a ON a.id = o.owner_actor_id
WHERE o.agent_actor_id = sqlc.arg(agent_actor_id)
ORDER BY lower(a.username);

-- name: ListAgentsOwnedBy :many
-- The agent actor ids an actor owns, for the "my agents" filter.
SELECT agent_actor_id FROM agent_owners WHERE owner_actor_id = sqlc.arg(owner_actor_id);

-- name: DeleteAgentOwners :exec
DELETE FROM agent_owners WHERE agent_actor_id = sqlc.arg(agent_actor_id);

-- name: DeleteAgentOwnerEdgesByActor :exec
-- Drop every ownership edge that touches an actor: the agents it owned and any
-- ownership of it as an agent.
DELETE FROM agent_owners
WHERE owner_actor_id = sqlc.arg(actor_id) OR agent_actor_id = sqlc.arg(actor_id);

-- Team logs -----------------------------------------------------------------

-- name: InsertTeamLog :exec
INSERT INTO team_logs (
  team_id, at, actor_id, kind, field, old_value, new_value, message, data_json
)
VALUES (
  sqlc.arg(team_id), sqlc.arg(at), sqlc.narg(actor_id), sqlc.arg(kind),
  sqlc.arg(field), sqlc.arg(old_value), sqlc.arg(new_value), sqlc.arg(message),
  sqlc.arg(data_json)
);

-- name: ListTeamLogs :many
SELECT
  l.id, l.team_id, l.at, l.actor_id, l.kind, l.field, l.old_value, l.new_value,
  l.message, l.data_json,
  a.kind AS actor_kind,
  a.username AS actor_username,
  a.name AS actor_name
FROM team_logs l
LEFT JOIN actors a ON a.id = l.actor_id
WHERE l.team_id = sqlc.arg(team_id)
  AND l.id < sqlc.arg(before_id)
ORDER BY l.at DESC, l.id DESC
LIMIT CAST(sqlc.arg(max_count) AS INTEGER);

-- name: DeleteTeamLogs :exec
DELETE FROM team_logs WHERE team_id = sqlc.arg(team_id);
