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












ALTER TABLE task_dependencies RENAME TO task_references;
ALTER TABLE task_references RENAME COLUMN depends_on_task_rowid TO referenced_task_rowid;
ALTER TABLE task_references ADD COLUMN blocking INTEGER NOT NULL DEFAULT 1
  CHECK (blocking IN (0, 1));

DROP INDEX idx_task_dependencies_on;
CREATE INDEX idx_task_references_on ON task_references (referenced_task_rowid);

-- +goose Down
DROP INDEX idx_task_references_on;
CREATE INDEX idx_task_dependencies_on ON task_references (referenced_task_rowid);
ALTER TABLE task_references DROP COLUMN blocking;
ALTER TABLE task_references RENAME COLUMN referenced_task_rowid TO depends_on_task_rowid;
ALTER TABLE task_references RENAME TO task_dependencies;
