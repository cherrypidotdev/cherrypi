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




CREATE TABLE terminals (
  id TEXT PRIMARY KEY,
  kind INTEGER NOT NULL,
  task_id TEXT NOT NULL DEFAULT '',
  owner TEXT NOT NULL DEFAULT '',
  cwd TEXT NOT NULL DEFAULT '',
  cols INTEGER NOT NULL DEFAULT 0,
  rows INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL CHECK (state IN ('open', 'interrupted', 'closed')),
  end_reason TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  ended_at INTEGER
);

CREATE INDEX idx_terminals_state ON terminals (state);



CREATE TABLE terminal_logs (
  id INTEGER PRIMARY KEY,
  terminal_id TEXT NOT NULL,
  kind INTEGER NOT NULL DEFAULT 0,
  task_id TEXT NOT NULL DEFAULT '',
  owner TEXT NOT NULL DEFAULT '',
  event TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT '',
  at INTEGER NOT NULL
);

CREATE INDEX idx_terminal_logs_at ON terminal_logs (at DESC);

-- +goose Down
DROP TABLE terminal_logs;
DROP TABLE terminals;
