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




DROP TABLE IF EXISTS task_size_backup;
CREATE TEMP TABLE task_size_backup AS SELECT rowid, size FROM tasks;

ALTER TABLE tasks DROP COLUMN size;
ALTER TABLE tasks ADD COLUMN size TEXT NOT NULL DEFAULT 'medium'
  CHECK (size IN ('small', 'medium', 'big', 'epic'));

UPDATE tasks SET size = (
  SELECT b.size FROM task_size_backup b WHERE b.rowid = tasks.rowid
) WHERE rowid IN (SELECT rowid FROM task_size_backup);

DROP TABLE task_size_backup;

-- +goose Down


DROP TABLE IF EXISTS task_size_backup;
CREATE TEMP TABLE task_size_backup AS SELECT rowid, size FROM tasks;

ALTER TABLE tasks DROP COLUMN size;
ALTER TABLE tasks ADD COLUMN size TEXT NOT NULL DEFAULT 'medium'
  CHECK (size IN ('small', 'medium', 'big'));

UPDATE tasks SET size = (
  SELECT CASE WHEN b.size = 'epic' THEN 'medium' ELSE b.size END
  FROM task_size_backup b WHERE b.rowid = tasks.rowid
) WHERE rowid IN (SELECT rowid FROM task_size_backup);

DROP TABLE task_size_backup;
