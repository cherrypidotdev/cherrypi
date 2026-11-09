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



ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_state_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_state_check CHECK (state IN (
  'backlog', 'waiting', 'queued', 'running', 'input', 'settled', 'failed',
  'completed', 'cancelled'
));
ALTER TABLE tasks ADD COLUMN parent_task_id TEXT
  REFERENCES tasks (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN hidden BIGINT NOT NULL DEFAULT 0
  CHECK (hidden IN (0, 1));
ALTER TABLE tasks ADD COLUMN pending_steering TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_tasks_parent ON tasks (parent_task_id);
CREATE INDEX idx_tasks_hidden ON tasks (hidden);

-- +goose Down
UPDATE tasks SET state = 'queued' WHERE state = 'waiting';
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_state_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_state_check CHECK (state IN (
  'backlog', 'queued', 'running', 'input', 'settled', 'failed',
  'completed', 'cancelled'
));
DROP INDEX IF EXISTS idx_tasks_hidden;
DROP INDEX IF EXISTS idx_tasks_parent;
ALTER TABLE tasks DROP COLUMN pending_steering;
ALTER TABLE tasks DROP COLUMN hidden;
ALTER TABLE tasks DROP COLUMN parent_task_id;
