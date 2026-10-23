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

-- Persistent inbox. A notification points at a task by its stable rowid when
-- it comes from one; reads render the public `<prefix>-<seq>` id for the UI.
-- A `recipient_actor_id` targets an entry at one actor; NULL broadcasts it to
-- every reader (legacy entries and org-wide signals).

-- name: CreateNotification :one
INSERT INTO notifications (
  agent_id, task_rowid, kind, severity, title, body, created_at, recipient_actor_id
)
VALUES (
  sqlc.arg(agent_id),
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(kind),
  sqlc.arg(severity),
  sqlc.arg(title),
  sqlc.arg(body),
  sqlc.arg(created_at),
  sqlc.narg(recipient_actor_id)
)
RETURNING *;

-- name: RefreshUnreadNotification :one
-- Bumps the newest unread notification of a kind for one task and one
-- recipient and rewrites its content, so a burst of attention signals (a
-- questionnaire's questions) stays one inbox entry instead of one per signal.
UPDATE notifications
SET
  title = sqlc.arg(title),
  body = sqlc.arg(body),
  created_at = sqlc.arg(created_at)
WHERE id = (
  SELECT n.id FROM notifications n
  WHERE
    n.agent_id = sqlc.arg(agent_id)
    AND n.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
    AND n.kind = sqlc.arg(kind)
    AND n.read_at IS NULL
    AND n.recipient_actor_id IS sqlc.arg(recipient_actor_id)
  ORDER BY n.created_at DESC
  LIMIT 1
)
RETURNING *;

-- name: ListNotifications :many
SELECT n.*, t.id AS task_id, t.project_id AS project_id
FROM notifications n
LEFT JOIN tasks t ON t.rowid = n.task_rowid
WHERE n.recipient_actor_id IS NULL OR n.recipient_actor_id = sqlc.arg(recipient_actor_id)
ORDER BY n.created_at DESC
LIMIT CAST(sqlc.arg(max_count) AS INTEGER);

-- name: CountUnreadNotifications :one
SELECT CAST(COUNT(*) AS INTEGER) FROM notifications
WHERE read_at IS NULL
  AND (recipient_actor_id IS NULL OR recipient_actor_id = sqlc.arg(recipient_actor_id));

-- name: MarkNotificationRead :exec
UPDATE notifications SET read_at = sqlc.arg(read_at)
WHERE id = sqlc.arg(id)
  AND (recipient_actor_id IS NULL OR recipient_actor_id = sqlc.arg(recipient_actor_id));

-- name: MarkTaskNotificationsRead :exec
-- Visiting a task is enough to consider its signals seen.
UPDATE notifications SET read_at = sqlc.arg(read_at)
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND read_at IS NULL
  AND (recipient_actor_id IS NULL OR recipient_actor_id = sqlc.arg(recipient_actor_id));

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read_at = sqlc.arg(read_at)
WHERE read_at IS NULL
  AND (recipient_actor_id IS NULL OR recipient_actor_id = sqlc.arg(recipient_actor_id));

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE id = sqlc.arg(id)
  AND (recipient_actor_id IS NULL OR recipient_actor_id = sqlc.arg(recipient_actor_id));

-- name: DeleteAllNotifications :exec
DELETE FROM notifications
WHERE recipient_actor_id IS NULL OR recipient_actor_id = sqlc.arg(recipient_actor_id);

-- name: ClearTaskNotifications :exec
-- Detach an inbox entry from a task that is being deleted, so a reused rowid
-- cannot make it point at a different task.
UPDATE notifications SET task_rowid = NULL
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: ClearTaskNotificationsByProjectAgent :exec
-- The same detach for every task a project membership removes.
UPDATE notifications SET task_rowid = NULL WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id) AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);
