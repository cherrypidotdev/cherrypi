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
-- A task can depend on other tasks: every dependency must reach `completed`
-- before the dependent is queued. A blocked task stays in `draft` at the SQL
-- level with `blocked = 1`, so the state CHECK is untouched (SQLite cannot
-- alter a CHECK without rebuilding the table) and this migration composes with
-- any other `ADD COLUMN` on `tasks`. The service exposes `blocked = 1` as the
-- `blocked` task state.
ALTER TABLE tasks ADD COLUMN blocked INTEGER NOT NULL DEFAULT 0
  CHECK (blocked IN (0, 1));

-- Dependency edges. Both sides cascade with their task; foreign-key
-- enforcement is off in this SQLite setup, so deletes clean them explicitly.
CREATE TABLE task_dependencies (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  depends_on_task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (task_id, depends_on_task_id),
  CHECK (task_id <> depends_on_task_id)
);

-- Reverse lookups: "what depends on this task?" drives the promotion, and the
-- cascade walks this index.
CREATE INDEX idx_task_dependencies_on ON task_dependencies (depends_on_task_id);

-- +goose Down
DROP TABLE task_dependencies;
ALTER TABLE tasks DROP COLUMN blocked;
