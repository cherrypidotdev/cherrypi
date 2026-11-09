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


CREATE TABLE IF NOT EXISTS plugin_log_surfaces (
  log_id BIGINT NOT NULL REFERENCES plugin_logs (id) ON DELETE CASCADE,
  plugin_id TEXT NOT NULL REFERENCES plugins (id) ON DELETE CASCADE,
  scope TEXT NOT NULL CHECK (scope IN ('org', 'project', 'agent', 'user')),
  target_id TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (log_id, scope, target_id)
);
CREATE INDEX IF NOT EXISTS idx_plugin_log_surfaces_target
  ON plugin_log_surfaces (plugin_id, scope, target_id, log_id DESC);

-- +goose Down
DROP TABLE IF EXISTS plugin_log_surfaces;
