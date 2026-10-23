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
-- Team domain log: one row per mutating team action (created, renamed, member
-- added/removed, invitation invited/accepted/approved/revoked, deleted), with
-- the acting actor and a pre-rendered summary. The phase-7 explorer unions the
-- domain logs, so the shape mirrors `task_logs` (minus the rowid indirection,
-- which teams do not need). `actor_id` is nullable so deleting an actor keeps
-- the history (see the phase-3 note on disabled foreign keys).
CREATE TABLE team_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  at INTEGER NOT NULL,
  actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  kind TEXT NOT NULL,
  field TEXT NOT NULL DEFAULT '',
  old_value TEXT NOT NULL DEFAULT '',
  new_value TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  -- Kind-specific structured data as a JSON object.
  data_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_team_logs_team ON team_logs (team_id, at DESC, id DESC);

-- +goose Down
DROP TABLE team_logs;
