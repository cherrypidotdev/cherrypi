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
-- Labeled URLs the running agent attached to its task (e.g. a PR or an issue).
-- The agent replaces the whole set each time it calls its setTaskUrls tool, so
-- rows are not archived; they cascade with the task. (task_id, url) is the
-- identity so a re-sent URL updates its label instead of duplicating.
CREATE TABLE task_links (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  url TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  position INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (task_id, url)
);
CREATE INDEX idx_task_links_task ON task_links (task_id, position);

-- +goose Down
DROP TABLE task_links;
