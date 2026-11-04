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

-- name: CreateTaskTemplate :one
INSERT INTO task_templates (
  id, project_id, name, description, created_by_actor_id, created_at, updated_at
) VALUES (
  sqlc.arg(id), sqlc.arg(project_id), sqlc.arg(name), sqlc.arg(description),
  sqlc.narg(created_by_actor_id), sqlc.arg(created_at), sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetTaskTemplate :one
SELECT t.*,
  CAST(COALESCE((SELECT MAX(v.version) FROM task_template_versions v WHERE v.template_id = t.id), 0) AS INTEGER) AS latest_version
FROM task_templates t
WHERE t.id = sqlc.arg(id);

-- name: GetTaskTemplateByProjectName :one
SELECT t.*,
  CAST(COALESCE((SELECT MAX(v.version) FROM task_template_versions v WHERE v.template_id = t.id), 0) AS INTEGER) AS latest_version
FROM task_templates t
WHERE t.project_id = sqlc.arg(project_id)
  AND lower(t.name) = lower(sqlc.arg(name))
  AND t.archived_at IS NULL;

-- name: ListTaskTemplates :many
SELECT t.*,
  CAST(COALESCE((SELECT MAX(v.version) FROM task_template_versions v WHERE v.template_id = t.id), 0) AS INTEGER) AS latest_version
FROM task_templates t
WHERE t.project_id = sqlc.arg(project_id)
  AND (CAST(sqlc.arg(include_archived) AS INTEGER) = 1 OR t.archived_at IS NULL)
ORDER BY (t.archived_at IS NOT NULL), lower(t.name);

-- name: UpdateTaskTemplate :one
UPDATE task_templates
SET name = sqlc.arg(name), description = sqlc.arg(description), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SetTaskTemplateArchived :one
UPDATE task_templates
SET archived_at = sqlc.narg(archived_at), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: CreateTaskTemplateVersion :one



INSERT INTO task_template_versions (
  template_id, version, body, title_template, args_json,
  default_priority, default_size,
  default_tag_ids_json, default_team_ids_json, default_lead_actor_ids_json,
  created_by_actor_id, created_at
)
SELECT
  sqlc.arg(template_id),
  COALESCE(MAX(version), 0) + 1,
  sqlc.arg(body),
  sqlc.arg(title_template),
  sqlc.arg(args_json),
  sqlc.arg(default_priority),
  sqlc.arg(default_size),
  sqlc.arg(default_tag_ids_json),
  sqlc.arg(default_team_ids_json),
  sqlc.arg(default_lead_actor_ids_json),
  sqlc.narg(created_by_actor_id),
  sqlc.arg(created_at)
FROM task_template_versions
WHERE template_id = sqlc.arg(template_id)
RETURNING *;

-- name: GetTaskTemplateVersion :one
SELECT * FROM task_template_versions
WHERE template_id = sqlc.arg(template_id) AND version = sqlc.arg(version);

-- name: ListTaskTemplateVersions :many
SELECT * FROM task_template_versions
WHERE template_id = sqlc.arg(template_id)
ORDER BY version DESC;

-- name: CreateTaskTemplateApplication :execrows
INSERT INTO task_template_applications (
  task_rowid, template_id, template_version, template_name, args_json, created_at
)
SELECT rowid, sqlc.narg(template_id), sqlc.arg(template_version), sqlc.arg(template_name),
  sqlc.arg(args_json), sqlc.arg(created_at)
FROM tasks WHERE id = sqlc.arg(task_id);

-- name: GetTaskTemplateApplication :one
SELECT a.* FROM task_template_applications a
JOIN tasks t ON t.rowid = a.task_rowid
WHERE t.id = sqlc.arg(task_id);
