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




-- name: ListTaskTags :many
SELECT
  t.id AS task_id, tags.id, tags.project_id, tags.name, tags.description,
  tags.color, tags.archived_at
FROM task_tags tt
JOIN tasks t ON t.rowid = tt.task_rowid
JOIN tags ON tags.id = tt.tag_id
ORDER BY t.id, lower(tags.name);

-- name: ListTaskTagsByTask :many
SELECT
  tags.id, tags.project_id, tags.name, tags.description, tags.color,
  tags.archived_at
FROM task_tags tt
JOIN tags ON tags.id = tt.tag_id
WHERE tt.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1)
ORDER BY lower(tags.name);

-- name: DeleteTaskTags :exec
DELETE FROM task_tags WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1);

-- name: DeleteTaskTagsByProjectAgent :exec


DELETE FROM task_tags WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id) AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: AddTaskTag :exec
INSERT INTO task_tags (task_rowid, tag_id)
VALUES ((SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)), sqlc.arg(tag_id))
ON CONFLICT DO NOTHING;

-- name: DeleteTaskTag :exec
DELETE FROM task_tags
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND tag_id = sqlc.arg(tag_id);
