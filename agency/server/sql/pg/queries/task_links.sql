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

-- Labeled URLs an agent attaches to a task. Links key on the task rowid so a
-- project prefix change never has to rewrite them.

-- name: ListTaskLinks :many
SELECT t.id AS task_id, l.url, l.label, l.position
FROM task_links l
JOIN tasks t ON t.rowid = l.task_rowid
ORDER BY l.task_rowid, l.position;

-- name: ListTaskLinksByTask :many
SELECT l.url, l.label, l.position
FROM task_links l
WHERE l.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY l.position;

-- name: DeleteTaskLinks :exec
DELETE FROM task_links
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskLinksByProjectAgent :exec
-- Clean up published links when a project membership takes its tasks with it.
DELETE FROM task_links WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id) AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: InsertTaskLink :exec
INSERT INTO task_links (task_rowid, url, label, position)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(url), sqlc.arg(label), sqlc.arg(position)
);
