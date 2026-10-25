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



ALTER TABLE tasks ADD COLUMN model_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN model_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN thinking_level TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN interactive INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN workspace TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN auto_title INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tasks DROP COLUMN auto_title;
ALTER TABLE tasks DROP COLUMN workspace;
ALTER TABLE tasks DROP COLUMN interactive;
ALTER TABLE tasks DROP COLUMN thinking_level;
ALTER TABLE tasks DROP COLUMN model_id;
ALTER TABLE tasks DROP COLUMN model_provider;
