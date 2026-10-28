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



DELETE FROM task_references
WHERE (task_rowid, referenced_task_rowid) IN (
  SELECT r.task_rowid, r.referenced_task_rowid
  FROM task_references r
  JOIN tasks task ON task.rowid = r.task_rowid
  JOIN tasks referenced ON referenced.rowid = r.referenced_task_rowid
  JOIN projects task_project ON task_project.id = task.project_id
  JOIN projects referenced_project ON referenced_project.id = referenced.project_id
  WHERE task_project.parent_project_id IS NULL
    AND referenced_project.parent_project_id IS NULL
    AND task_project.id <> referenced_project.id
);

-- +goose Down

SELECT 1;
