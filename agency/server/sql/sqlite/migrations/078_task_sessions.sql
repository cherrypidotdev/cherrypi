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






CREATE TABLE task_sessions (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  agent_id INTEGER NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  session_id TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  deleted_at INTEGER,
  PRIMARY KEY (task_id, agent_id, session_id)
);

CREATE INDEX idx_task_sessions_live ON task_sessions (task_id)
WHERE deleted_at IS NULL;



INSERT INTO task_sessions (task_id, agent_id, session_id, created_at, deleted_at)
SELECT id, session_agent_id, session_id, COALESCE(updated_at, created_at),
  session_deleted_at
FROM tasks
WHERE session_id <> '' AND session_agent_id IS NOT NULL;

-- +goose Down
DROP INDEX idx_task_sessions_live;
DROP TABLE task_sessions;
