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

-- name: CreateTag :one
INSERT INTO tags (id, project_id, name, description, color, is_default, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetTag :one
SELECT * FROM tags WHERE id = $1;

-- name: GetTagByName :one
SELECT * FROM tags
WHERE project_id = sqlc.arg(project_id) AND lower(name) = lower(sqlc.arg(name))
LIMIT 1;

-- name: ListTags :many
-- Every tag, or one project's tags when project_id is set.
SELECT tags.*, (
  SELECT count(*) FROM task_tags tt WHERE tt.tag_id = tags.id
) AS task_count
FROM tags
WHERE (
    CAST(sqlc.narg(project_id) AS TEXT) IS NULL
    OR tags.project_id = sqlc.narg(project_id)
  )
  AND (
    archived_at IS NULL
    OR CAST(sqlc.arg(include_archived) AS BIGINT) = 1
  )
ORDER BY tags.project_id, lower(name);

-- name: GetTagWithCount :one
SELECT tags.*, (
  SELECT count(*) FROM task_tags tt WHERE tt.tag_id = tags.id
) AS task_count
FROM tags WHERE id = $1;

-- name: UpdateTag :one
UPDATE tags SET name = $1, description = $2, color = $3, is_default = $4, updated_at = $5
WHERE id = $6
RETURNING *;

-- name: SetTagArchived :one
UPDATE tags SET archived_at = $1, updated_at = $2 WHERE id = $3 RETURNING *;

-- name: DeleteTagsByProject :exec
DELETE FROM tags WHERE project_id = $1;
