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

-- name: ListVariables :many
SELECT * FROM variables ORDER BY scope_kind, scope_id, name;

-- name: GetVariable :one
SELECT * FROM variables
WHERE scope_kind = sqlc.arg(scope_kind)
  AND scope_id = sqlc.arg(scope_id)
  AND name = sqlc.arg(name);

-- name: UpsertVariable :exec
INSERT INTO variables (
  scope_kind, scope_id, name, note, secret, value, value_sealed,
  created_by_actor_id, updated_by_actor_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(scope_kind, scope_id, name) DO UPDATE SET
  note = excluded.note,
  secret = excluded.secret,
  value = excluded.value,
  value_sealed = excluded.value_sealed,
  updated_by_actor_id = excluded.updated_by_actor_id,
  updated_at = excluded.updated_at;

-- name: DeleteVariable :execrows
DELETE FROM variables
WHERE scope_kind = sqlc.arg(scope_kind)
  AND scope_id = sqlc.arg(scope_id)
  AND name = sqlc.arg(name);

-- name: DeleteVariablesForScope :exec
DELETE FROM variables
WHERE scope_kind = sqlc.arg(scope_kind) AND scope_id = sqlc.arg(scope_id);

-- name: ListTeamIDsForAgentActor :many
SELECT DISTINCT etm.team_id
FROM effective_team_members etm
JOIN agents a ON a.actor_id = etm.actor_id
WHERE a.actor_id = sqlc.arg(actor_id)
ORDER BY etm.team_id;
