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





-- name: GetTaskProgress :one
SELECT summary, updated_at
FROM task_progress
WHERE task_rowid = (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id));

-- name: ListTaskProgress :many
SELECT t.id AS task_id, p.summary, p.updated_at
FROM task_progress p
JOIN tasks t ON t.rowid = p.task_rowid;

-- name: ListTaskProgressItems :many
SELECT t.id AS task_id, i.position, i.text, i.status
FROM task_progress_items i
JOIN tasks t ON t.rowid = i.task_rowid
ORDER BY i.task_rowid, i.position;

-- name: ListTaskProgressItemsByTask :many
SELECT position, text, status
FROM task_progress_items
WHERE task_rowid = (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id))
ORDER BY position;

-- name: UpsertTaskProgress :exec
INSERT INTO task_progress (task_rowid, summary, updated_at)
VALUES (
  (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id)),
  sqlc.arg(summary), sqlc.arg(updated_at)
)
ON CONFLICT (task_rowid) DO UPDATE SET
  summary = excluded.summary,
  updated_at = excluded.updated_at;

-- name: DeleteTaskProgressItems :exec
DELETE FROM task_progress_items
WHERE task_rowid = (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id));

-- name: InsertTaskProgressItem :exec
INSERT INTO task_progress_items (task_rowid, position, text, status)
VALUES (
  (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id)),
  sqlc.arg(position), sqlc.arg(text), sqlc.arg(status)
);

-- name: DeleteTaskProgress :exec
DELETE FROM task_progress
WHERE task_rowid = (SELECT rowid FROM tasks WHERE id = sqlc.arg(task_id));

-- name: DeleteTaskProgressItemsByProjectAgent :exec
DELETE FROM task_progress_items WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id)
    AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteTaskProgressByProjectAgent :exec
DELETE FROM task_progress WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id)
    AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);
