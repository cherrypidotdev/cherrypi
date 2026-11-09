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



-- name: UpsertTaskHold :exec
INSERT INTO task_holds (task_id, plugin_id, key, reason, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (task_id, plugin_id, key)
DO UPDATE SET reason = excluded.reason;

-- name: DeleteTaskHold :execrows
DELETE FROM task_holds
WHERE task_id = $1 AND plugin_id = $2 AND key = $3;

-- name: DeleteTaskHolds :execrows
DELETE FROM task_holds WHERE task_id = $1;

-- name: ListTaskHolds :many
SELECT
  h.task_id,
  h.plugin_id,
  p.name AS plugin_name,
  h.key,
  h.reason,
  h.created_at
FROM task_holds h
JOIN plugins p ON p.id = h.plugin_id
WHERE h.task_id = $1
ORDER BY h.created_at, h.plugin_id, h.key;

-- name: ListAllTaskHolds :many
SELECT
  h.task_id,
  h.plugin_id,
  p.name AS plugin_name,
  h.key,
  h.reason,
  h.created_at
FROM task_holds h
JOIN plugins p ON p.id = h.plugin_id
ORDER BY h.task_id, h.created_at, h.plugin_id, h.key;

-- name: CountTaskHolds :one
SELECT COUNT(*) FROM task_holds WHERE task_id = $1;

-- name: ListTaskHoldTaskIDsByPlugin :many
SELECT DISTINCT task_id FROM task_holds WHERE plugin_id = $1;
