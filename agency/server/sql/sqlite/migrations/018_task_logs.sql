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















ALTER TABLE tasks ADD COLUMN state_updated_at INTEGER NOT NULL DEFAULT 0;
UPDATE tasks SET state_updated_at = updated_at;

CREATE TABLE task_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  at INTEGER NOT NULL,
  actor TEXT NOT NULL DEFAULT 'system',
  kind TEXT NOT NULL,
  field TEXT NOT NULL DEFAULT '',
  old_value TEXT NOT NULL DEFAULT '',
  new_value TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  
  
  data_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_task_logs_task ON task_logs (task_rowid, at DESC, id DESC);




-- +goose StatementBegin
CREATE TRIGGER tasks_state_updated_at_on_update
AFTER UPDATE OF state, blocked ON tasks
FOR EACH ROW
WHEN OLD.state <> NEW.state OR OLD.blocked <> NEW.blocked
BEGIN
  UPDATE tasks SET state_updated_at = NEW.updated_at WHERE rowid = NEW.rowid;
END;
-- +goose StatementEnd


-- +goose StatementBegin
CREATE TRIGGER tasks_state_updated_at_on_insert
AFTER INSERT ON tasks
FOR EACH ROW
BEGIN
  UPDATE tasks SET state_updated_at = NEW.created_at WHERE rowid = NEW.rowid;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER tasks_state_updated_at_on_insert;
DROP TRIGGER tasks_state_updated_at_on_update;
DROP TABLE task_logs;
ALTER TABLE tasks DROP COLUMN state_updated_at;
