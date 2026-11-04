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




CREATE TABLE task_progress (
  task_rowid BIGINT PRIMARY KEY REFERENCES tasks (rowid) ON DELETE CASCADE,
  summary TEXT NOT NULL DEFAULT '',
  updated_at BIGINT NOT NULL
);

CREATE TABLE task_progress_items (
  task_rowid BIGINT NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  position INTEGER NOT NULL,
  text TEXT NOT NULL,
  status INTEGER NOT NULL DEFAULT 1 CHECK (status IN (1, 2, 3)),
  PRIMARY KEY (task_rowid, position)
);

CREATE INDEX idx_task_progress_items_task
  ON task_progress_items (task_rowid, position);

-- +goose Down
DROP TABLE task_progress_items;
DROP TABLE task_progress;
