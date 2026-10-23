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
-- Project domain log: one row per mutating project action (created, renamed,
-- archived, access changed, deleted), with the acting actor and a pre-rendered
-- summary. The phase-7 explorer unions the domain logs, so the shape mirrors
-- `team_logs` (minus the rowid indirection, which projects do not need).
-- `actor_id` is nullable so deleting an actor keeps the history (foreign keys
-- are off at runtime; see the phase-3 note).
CREATE TABLE project_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
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
CREATE INDEX idx_project_logs_project ON project_logs (project_id, at DESC, id DESC);

-- +goose Down
DROP TABLE project_logs;
