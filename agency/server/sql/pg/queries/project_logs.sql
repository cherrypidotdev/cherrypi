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




-- name: InsertProjectLog :exec
INSERT INTO project_logs (
  project_id, at, actor_id, kind, field, old_value, new_value, message, data_json
)
VALUES (
  sqlc.arg(project_id), sqlc.arg(at), sqlc.narg(actor_id), sqlc.arg(kind),
  sqlc.arg(field), sqlc.arg(old_value), sqlc.arg(new_value), sqlc.arg(message),
  sqlc.arg(data_json)
);

-- name: ListProjectLogs :many
SELECT
  l.id, l.project_id, l.at, l.actor_id, l.kind, l.field, l.old_value,
  l.new_value, l.message, l.data_json,
  a.kind AS actor_kind,
  a.username AS actor_username,
  a.name AS actor_name
FROM project_logs l
LEFT JOIN actors a ON a.id = l.actor_id
WHERE l.project_id = sqlc.arg(project_id)
  AND l.id < sqlc.arg(before_id)
ORDER BY l.at DESC, l.id DESC
LIMIT CAST(sqlc.arg(max_count) AS BIGINT);

-- name: DeleteProjectLogs :exec
DELETE FROM project_logs WHERE project_id = sqlc.arg(project_id);
