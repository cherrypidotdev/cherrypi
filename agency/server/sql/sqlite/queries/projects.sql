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

-- name: CreateProject :one
INSERT INTO projects (id, name, slug, prefix, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects WHERE id = ?;

-- name: GetProjectBySlug :one
SELECT * FROM projects WHERE slug = ?;

-- name: GetProjectByPrefix :one
SELECT * FROM projects WHERE prefix = ?;

-- name: ListProjects :many
SELECT * FROM projects ORDER BY created_at ASC;

-- name: UpdateProject :one
UPDATE projects
SET name = ?, description = ?, prefix = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: AllocateTaskNumber :one
-- Atomically hands out the next task number for a project.
UPDATE projects
SET task_counter = task_counter + 1
WHERE id = ?
RETURNING task_counter;

-- name: SetProjectArchived :one
UPDATE projects
SET archived_at = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: SetProjectParent :one
UPDATE projects
SET parent_project_id = sqlc.narg(parent_project_id), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetProjectParent :one
SELECT parent.* FROM projects parent
JOIN projects child ON child.parent_project_id = parent.id
WHERE child.id = sqlc.arg(child_id);

-- name: ListProjectChildren :many
SELECT * FROM projects
WHERE parent_project_id = sqlc.arg(parent_project_id)
ORDER BY name COLLATE NOCASE;

-- name: CountProjectChildren :one
SELECT COUNT(*) FROM projects WHERE parent_project_id = sqlc.arg(parent_project_id);

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;
