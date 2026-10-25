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




ALTER TABLE project_agents ADD COLUMN can_list_tasks INTEGER NOT NULL DEFAULT 1;
ALTER TABLE project_agents ADD COLUMN can_create_tasks INTEGER NOT NULL DEFAULT 0;
ALTER TABLE project_agents ADD COLUMN can_update_tasks INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE project_agents DROP COLUMN can_update_tasks;
ALTER TABLE project_agents DROP COLUMN can_create_tasks;
ALTER TABLE project_agents DROP COLUMN can_list_tasks;
