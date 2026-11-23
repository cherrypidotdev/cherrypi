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


-- message, in send order. The texts joined with a blank line are the task's



-- name: CreateTaskPendingMessage :exec
INSERT INTO task_pending_messages (
  id, task_rowid, message_id, text, actor_id, cancelable, position, created_at
)
VALUES (
  sqlc.arg(id),
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.narg(message_id),
  sqlc.arg(text),
  sqlc.narg(actor_id),
  sqlc.arg(cancelable),
  
  
  COALESCE((
    SELECT MAX(position) + 1 FROM task_pending_messages
    WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  ), 0),
  sqlc.arg(created_at)
);

-- name: ListTaskPendingMessages :many
SELECT
  m.id, m.message_id, m.text, m.actor_id, m.cancelable, m.position, m.created_at
FROM task_pending_messages m
WHERE m.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY m.position;

-- name: DeleteTaskPendingMessage :execrows

DELETE FROM task_pending_messages
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND task_pending_messages.id = sqlc.arg(id);

-- name: DeleteTaskPendingMessages :execrows


DELETE FROM task_pending_messages
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));
