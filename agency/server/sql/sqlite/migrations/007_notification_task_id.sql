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



ALTER TABLE notifications RENAME COLUMN session_id TO task_id;

UPDATE notifications
SET task_id = COALESCE(
  (
    SELECT t.id FROM tasks t
    WHERE t.assignee_agent_id = notifications.agent_id
      AND t.session_id = notifications.task_id
  ),
  ''
)
WHERE task_id <> ''
  AND task_id NOT IN (SELECT id FROM tasks);

-- +goose Down
UPDATE notifications
SET task_id = COALESCE(
  (
    SELECT t.session_id FROM tasks t
    WHERE t.id = notifications.task_id AND t.session_id <> ''
  ),
  ''
)
WHERE task_id <> '';

ALTER TABLE notifications RENAME COLUMN task_id TO session_id;
