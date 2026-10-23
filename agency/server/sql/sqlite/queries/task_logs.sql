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

-- Task change log. Entries key on the task rowid so a project prefix change
-- never has to rewrite them; the public queries still take the string id.

-- name: InsertTaskLog :exec
INSERT INTO task_logs (
  task_rowid, at, actor, actor_id, kind, field, old_value, new_value, message, data_json
)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(at),
  sqlc.arg(actor),
  sqlc.narg(actor_id),
  sqlc.arg(kind),
  sqlc.arg(field),
  sqlc.arg(old_value),
  sqlc.arg(new_value),
  sqlc.arg(message),
  sqlc.arg(data_json)
);

-- name: ListTaskLogsByTask :many
SELECT id, at, actor, actor_id, kind, field, old_value, new_value, message, data_json
FROM task_logs
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY at DESC, id DESC;

-- name: DeleteTaskLogs :exec
DELETE FROM task_logs
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskLogsByProjectAgent :exec
-- Clean up logs when a project membership takes its tasks with it (foreign
-- keys are off, so nothing cascades).
DELETE FROM task_logs WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id)
    AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);
