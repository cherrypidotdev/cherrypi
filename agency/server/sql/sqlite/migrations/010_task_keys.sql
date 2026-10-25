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







ALTER TABLE projects ADD COLUMN prefix TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN task_counter INTEGER NOT NULL DEFAULT 0;
UPDATE projects SET prefix = UPPER(slug);


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



UPDATE projects
SET task_counter = COALESCE(
  (SELECT MAX(number) FROM tasks WHERE tasks.project_id = projects.id),
  0
);




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
