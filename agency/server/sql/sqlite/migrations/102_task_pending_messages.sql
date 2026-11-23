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
















CREATE TABLE task_pending_messages (
  id TEXT PRIMARY KEY,
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  message_id TEXT,
  text TEXT NOT NULL DEFAULT '',
  actor_id BIGINT,
  cancelable INTEGER NOT NULL DEFAULT 1,
  
  
  position INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_task_pending_messages_task
  ON task_pending_messages (task_rowid, position);








INSERT INTO task_pending_messages (
  id, task_rowid, message_id, text, actor_id, cancelable, position, created_at
)
SELECT
  'legacy:' || t.id, t.rowid, NULL, t.pending_prompt,
  t.pending_prompt_actor_id, 0, 0, t.updated_at
FROM tasks t
WHERE t.pending_prompt <> '';

-- +goose Down
DROP TABLE task_pending_messages;
