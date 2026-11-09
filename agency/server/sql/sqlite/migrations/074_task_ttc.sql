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










ALTER TABLE tasks ADD COLUMN ttc_ms INTEGER NOT NULL DEFAULT 0;







UPDATE tasks SET ttc_ms = CASE
  WHEN COALESCE(queued_at, 0) = 0 THEN 0
  WHEN state IN ('completed', 'cancelled')
    THEN MAX(COALESCE(settled_at, 0) - queued_at, 0)
  WHEN state = 'backlog' THEN 0
  ELSE MAX(COALESCE(state_updated_at, 0) - queued_at, 0)
END;

-- +goose StatementBegin
CREATE TRIGGER tasks_ttc_on_update
AFTER UPDATE OF state ON tasks
FOR EACH ROW
WHEN OLD.state <> NEW.state
BEGIN
  UPDATE tasks
  SET ttc_ms = OLD.ttc_ms + CASE
    WHEN OLD.state NOT IN ('backlog', 'completed', 'cancelled')
      THEN MAX(NEW.updated_at - OLD.state_updated_at, 0)
    ELSE 0
  END
  WHERE rowid = NEW.rowid;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER tasks_ttc_on_update;
ALTER TABLE tasks DROP COLUMN ttc_ms;
