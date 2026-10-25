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










CREATE TABLE task_session_entries (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  entry_id TEXT NOT NULL,
  parent_id TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL,
  raw TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (task_id, seq)
);

CREATE UNIQUE INDEX idx_task_session_entries_entry
  ON task_session_entries (task_id, entry_id);





ALTER TABLE tasks ADD COLUMN session_agent_id TEXT NOT NULL DEFAULT '';




UPDATE tasks
SET session_agent_id = assignee_agent_id
WHERE session_id <> '' AND assignee_agent_id IS NOT NULL;

-- +goose Down
ALTER TABLE tasks DROP COLUMN session_agent_id;
DROP INDEX idx_task_session_entries_entry;
DROP TABLE task_session_entries;
