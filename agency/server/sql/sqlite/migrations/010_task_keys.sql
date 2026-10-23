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

-- +goose Up
-- Custom, human-readable task ids. A project carries a task prefix (defaulting
-- to its uppercased slug) and a per-project counter, so a task id reads
-- `<prefix>-<number>` (e.g. `MYPROJECT-1`). Changing the prefix renames the
-- project's existing tasks, so `tasks.id` is not derived at read time but
-- stored. Legacy orphan tasks (created before a project was required) share
-- the reserved `TASK` prefix and one global sequence.

ALTER TABLE projects ADD COLUMN prefix TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN task_counter INTEGER NOT NULL DEFAULT 0;
UPDATE projects SET prefix = UPPER(slug);
-- `TASK` is reserved for orphan tasks; keep projects off it. Pick a suffix past
-- any existing TASK-N prefix so the rename can never collide.
UPDATE projects
SET prefix = 'TASK-' || (
  COALESCE(
    (SELECT MAX(CAST(substr(p2.prefix, 6) AS INTEGER))
     FROM projects p2
     WHERE p2.prefix GLOB 'TASK-[0-9]*' AND p2.id <> projects.id),
    1
  ) + 1
)
WHERE prefix = 'TASK';
CREATE UNIQUE INDEX idx_projects_prefix ON projects (prefix);

ALTER TABLE tasks ADD COLUMN number INTEGER NOT NULL DEFAULT 0;

-- Number each task within its project, oldest first. Orphan tasks (no project)
-- form a single partition.
UPDATE tasks
SET number = (
  SELECT rn
  FROM (
    SELECT id AS tid,
           ROW_NUMBER() OVER (PARTITION BY project_id ORDER BY created_at, id) AS rn
    FROM tasks
  ) AS numbered
  WHERE numbered.tid = tasks.id
);

-- Seed each project's counter past its current tasks; the next task is the
-- counter plus one.
UPDATE projects
SET task_counter = COALESCE(
  (SELECT MAX(number) FROM tasks WHERE tasks.project_id = projects.id),
  0
);

-- Rewrite every legacy UUID into a `<prefix>-<number>` id. The old -> new map
-- is kept so the child tables can be updated too (foreign key enforcement is
-- off, and the tag/link tables have no ON UPDATE clause).
CREATE TEMP TABLE task_key_map (old_id TEXT PRIMARY KEY, new_id TEXT NOT NULL);
INSERT INTO task_key_map (old_id, new_id)
SELECT t.id,
       COALESCE((SELECT p.prefix FROM projects p WHERE p.id = t.project_id), 'TASK')
         || '-' || t.number
FROM tasks t;

UPDATE task_tags
SET task_id = (SELECT new_id FROM task_key_map m WHERE m.old_id = task_tags.task_id)
WHERE task_id IN (SELECT old_id FROM task_key_map);

UPDATE task_links
SET task_id = (SELECT new_id FROM task_key_map m WHERE m.old_id = task_links.task_id)
WHERE task_id IN (SELECT old_id FROM task_key_map);

UPDATE notifications
SET task_id = (SELECT new_id FROM task_key_map m WHERE m.old_id = notifications.task_id)
WHERE task_id IN (SELECT old_id FROM task_key_map);

UPDATE tasks
SET id = (SELECT new_id FROM task_key_map m WHERE m.old_id = tasks.id);

DROP TABLE task_key_map;

CREATE UNIQUE INDEX idx_tasks_project_number ON tasks (project_id, number)
WHERE project_id IS NOT NULL;

-- +goose Down
-- The original random ids cannot be recovered; mint fresh UUIDs and rewrite the
-- references so the schema drops cleanly.
CREATE TEMP TABLE task_key_revert (old_id TEXT PRIMARY KEY, new_id TEXT NOT NULL);
INSERT INTO task_key_revert (old_id, new_id)
SELECT id, lower(hex(randomblob(16))) FROM tasks;

UPDATE task_tags
SET task_id = (
  SELECT new_id FROM task_key_revert r WHERE r.old_id = task_tags.task_id
)
WHERE task_id IN (SELECT old_id FROM task_key_revert);

UPDATE task_links
SET task_id = (
  SELECT new_id FROM task_key_revert r WHERE r.old_id = task_links.task_id
)
WHERE task_id IN (SELECT old_id FROM task_key_revert);

UPDATE notifications
SET task_id = (
  SELECT new_id FROM task_key_revert r WHERE r.old_id = notifications.task_id
)
WHERE task_id IN (SELECT old_id FROM task_key_revert);

UPDATE tasks SET id = (SELECT new_id FROM task_key_revert r WHERE r.old_id = tasks.id);

DROP TABLE task_key_revert;

DROP INDEX idx_tasks_project_number;
DROP INDEX idx_projects_prefix;
ALTER TABLE tasks DROP COLUMN number;
ALTER TABLE projects DROP COLUMN task_counter;
ALTER TABLE projects DROP COLUMN prefix;
