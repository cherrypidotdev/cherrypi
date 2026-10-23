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

-- Agent queries. `actors` holds the shared identity; `agents(id, actor_id)`
-- holds the agent surrogate. Agent-specific tables reference `agents.id`;
-- actor-level references use the actor id.

-- name: CreateActor :one
INSERT INTO actors (
  kind, username, name, description, created_at, updated_at
)
VALUES (
  'agent',
  sqlc.arg(username),
  sqlc.arg(name),
  sqlc.arg(description),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING id;

-- name: CreateAgentRow :one
INSERT INTO agents (actor_id) VALUES (sqlc.arg(actor_id))
RETURNING id, actor_id, last_seen_at;

-- name: GetAgent :one
SELECT
  g.id, g.actor_id, g.last_seen_at,
  a.username, a.name, a.description, a.picture_attachment_id,
  a.created_at, a.updated_at
FROM agents g
JOIN actors a ON a.id = g.actor_id
WHERE g.id = sqlc.arg(id);

-- name: GetAgentByActor :one
SELECT
  g.id, g.actor_id, g.last_seen_at,
  a.username, a.name, a.description, a.picture_attachment_id,
  a.created_at, a.updated_at
FROM agents g
JOIN actors a ON a.id = g.actor_id
WHERE g.actor_id = sqlc.arg(actor_id);

-- name: GetAgentByUsername :one
SELECT
  g.id, g.actor_id, g.last_seen_at,
  a.username, a.name, a.description, a.picture_attachment_id,
  a.created_at, a.updated_at
FROM agents g
JOIN actors a ON a.id = g.actor_id
WHERE a.kind = 'agent' AND a.username = sqlc.arg(username);

-- name: ListAgents :many
SELECT
  g.id, g.actor_id, g.last_seen_at,
  a.username, a.name, a.description, a.picture_attachment_id,
  a.created_at, a.updated_at
FROM agents g
JOIN actors a ON a.id = g.actor_id
ORDER BY a.name, a.username;

-- name: UpdateAgent :exec
UPDATE actors
SET name = sqlc.arg(name),
    description = sqlc.arg(description),
    updated_at = sqlc.arg(updated_at)
WHERE actors.id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(id))
  AND kind = 'agent';

-- name: DeleteAgent :exec
DELETE FROM agents WHERE id = sqlc.arg(id);

-- name: DeleteActor :exec
DELETE FROM actors WHERE id = sqlc.arg(actor_id) AND kind = 'agent';

-- name: TouchAgentSeen :exec
UPDATE agents SET last_seen_at = sqlc.arg(last_seen_at) WHERE id = sqlc.arg(id);

-- name: CreateAgentCredential :one
INSERT INTO agent_credentials (agent_id, username, password_hash, created_at)
VALUES (
  sqlc.arg(agent_id), sqlc.arg(username), sqlc.arg(password_hash), sqlc.arg(created_at)
)
RETURNING *;

-- name: ListAgentCredentials :many
SELECT * FROM agent_credentials WHERE agent_id = sqlc.arg(agent_id)
ORDER BY username;

-- name: GetAgentCredential :one
SELECT * FROM agent_credentials WHERE username = sqlc.arg(username);

-- name: RevokeAgentCredential :execrows
DELETE FROM agent_credentials
WHERE agent_id = sqlc.arg(agent_id) AND username = sqlc.arg(username);

-- name: TouchAgentCredentialUsed :exec
UPDATE agent_credentials SET last_used_at = sqlc.arg(last_used_at)
WHERE username = sqlc.arg(username);

-- name: UpdateAgentCredentialPassword :exec
UPDATE agent_credentials SET password_hash = sqlc.arg(password_hash)
WHERE username = sqlc.arg(username);
