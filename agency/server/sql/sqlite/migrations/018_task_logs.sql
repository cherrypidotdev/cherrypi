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
-- Task change log and state-change timestamp.
--
-- `task_logs` records who changed what, when: appended on creation and on
-- every meaningful mutation (state, priority, size, assignee, tags,
-- dependencies, links, title, project, prompt, model). Entries key on
-- `tasks.rowid`, so a project prefix rename never rewrites them. `old_value`
-- and `new_value` are stored pre-rendered for display; `message` carries the
-- human summary.
--
-- `tasks.state_updated_at` is the moment the effective state last changed.
-- It is separate from `updated_at`, which any edit bumps, so a list can order
-- by "recently changed state" without a priority tweak reordering it. Two
-- triggers keep it in sync on every state/blocked transition, so no query has
-- to remember to write it. Existing rows are backfilled from `updated_at`.

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
  -- Kind-specific structured data as a JSON object, so a client can render
  -- badges and links instead of parsing the message.
  data_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_task_logs_task ON task_logs (task_rowid, at DESC, id DESC);

-- A state change (state column or the `blocked` flag that renders as blocked)
-- stamps state_updated_at with the transition's updated_at. The inner UPDATE
-- does not touch state/blocked, so it cannot re-fire the trigger.
-- +goose StatementBegin
CREATE TRIGGER tasks_state_updated_at_on_update
AFTER UPDATE OF state, blocked ON tasks
FOR EACH ROW
WHEN OLD.state <> NEW.state OR OLD.blocked <> NEW.blocked
BEGIN
  UPDATE tasks SET state_updated_at = NEW.updated_at WHERE rowid = NEW.rowid;
END;
-- +goose StatementEnd

-- A new task's state is its state at creation.
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
