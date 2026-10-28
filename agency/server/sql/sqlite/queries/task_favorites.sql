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





-- name: AddTaskFavorite :exec
INSERT INTO task_favorites (task_rowid, actor_id, created_at)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(actor_id),
  sqlc.arg(created_at)
)
ON CONFLICT(task_rowid, actor_id) DO NOTHING;

-- name: DeleteTaskFavorite :exec
DELETE FROM task_favorites
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND actor_id = sqlc.arg(actor_id);

-- name: IsActorFavoriteTask :one
SELECT EXISTS (
  SELECT 1 FROM task_favorites
  WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
    AND actor_id = sqlc.arg(actor_id)
);

-- name: ListFavoriteTaskIDs :many
SELECT t.id
FROM task_favorites tf
JOIN tasks t ON t.rowid = tf.task_rowid
WHERE tf.actor_id = sqlc.arg(actor_id)
ORDER BY tf.created_at DESC, t.id;

-- name: DeleteTaskFavoritesByActor :exec
DELETE FROM task_favorites WHERE actor_id = sqlc.arg(actor_id);

-- name: DeleteTaskFavoritesByTask :exec
DELETE FROM task_favorites
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));
