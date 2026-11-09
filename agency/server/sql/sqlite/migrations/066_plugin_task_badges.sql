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





CREATE TABLE plugin_task_badges (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  plugin_id TEXT NOT NULL REFERENCES plugins (id) ON DELETE CASCADE,
  badges_json TEXT NOT NULL DEFAULT '[]',
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (task_id, plugin_id)
);
CREATE INDEX idx_plugin_task_badges_plugin ON plugin_task_badges (plugin_id);

-- +goose Down
DROP TABLE plugin_task_badges;
