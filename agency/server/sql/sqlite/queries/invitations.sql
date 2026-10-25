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



-- name: CreateInvitation :one
INSERT INTO invitations (
  kind, username, actor_id, email, provider_id, external_username, role,
  token_hash, invited_by_actor_id, created_at, expires_at
)
VALUES (
  sqlc.arg(kind), sqlc.arg(username), sqlc.arg(actor_id), sqlc.narg(email),
  sqlc.narg(provider_id), sqlc.narg(external_username), sqlc.arg(role),
  sqlc.narg(token_hash), sqlc.narg(invited_by_actor_id),
  sqlc.arg(created_at), sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetInvitationByID :one
SELECT * FROM invitations WHERE id = sqlc.arg(id);

-- name: GetInvitationByTokenHash :one
SELECT * FROM invitations WHERE token_hash = sqlc.arg(token_hash);

-- name: ListInvitations :many
SELECT
  i.*,
  COALESCE(a.name, '') AS actor_name,
  u.status AS actor_status,
  ib.username AS invited_by_username
FROM invitations i
LEFT JOIN actors a ON a.id = i.actor_id
LEFT JOIN users u ON u.actor_id = i.actor_id
LEFT JOIN actors ib ON ib.id = i.invited_by_actor_id
ORDER BY i.created_at DESC;

-- name: GetPendingInvitationByUsername :one
SELECT * FROM invitations
WHERE username = sqlc.arg(username)
  AND accepted_at IS NULL AND revoked_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPendingInvitationByActor :one
SELECT * FROM invitations
WHERE actor_id = sqlc.arg(actor_id)
  AND accepted_at IS NULL AND revoked_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPendingExternalInvitation :one
-- Matches a provider-scoped OIDC/LDAP invite at the first federated login.
SELECT * FROM invitations
WHERE kind = sqlc.arg(kind)
  AND provider_id = sqlc.arg(provider_id)
  AND external_username = sqlc.arg(external_username)
  AND accepted_at IS NULL AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC
LIMIT 1;

-- name: AcceptInvitation :execrows
UPDATE invitations SET accepted_at = sqlc.arg(accepted_at)
WHERE id = sqlc.arg(id) AND accepted_at IS NULL AND revoked_at IS NULL;

-- name: UpdateInvitationTokenHash :exec
UPDATE invitations SET token_hash = sqlc.arg(token_hash)
WHERE id = sqlc.arg(id) AND accepted_at IS NULL AND revoked_at IS NULL;

-- name: RevokeInvitation :execrows
UPDATE invitations SET revoked_at = sqlc.arg(revoked_at)
WHERE id = sqlc.arg(id) AND accepted_at IS NULL AND revoked_at IS NULL;

-- name: DeleteInvitationsByActor :exec
DELETE FROM invitations WHERE actor_id = sqlc.arg(actor_id);

-- name: DeleteExpiredInvitedActors :execrows

-- still be in the invited state (an accepted invite leaves an active actor).
DELETE FROM actors
WHERE id IN (
  SELECT i.actor_id FROM invitations i
  JOIN users u ON u.actor_id = i.actor_id
  WHERE i.accepted_at IS NULL AND i.revoked_at IS NULL
    AND i.expires_at < sqlc.arg(now)
    AND u.status = 'invited'
);

-- name: DeleteExpiredInvitations :execrows
DELETE FROM invitations
WHERE accepted_at IS NULL AND revoked_at IS NULL AND expires_at < sqlc.arg(now);
