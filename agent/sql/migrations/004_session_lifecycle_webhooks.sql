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
-- Persist the waiting-for-input state so a session blocked on an extension
-- dialog survives a restart. `pending_dialog` holds the raw
-- extension_ui_request JSON; on the user's answer the backend resumes Pi and
-- replays the answer (Pi re-emits the dangling dialog).
ALTER TABLE sessions ADD COLUMN waiting_for_input INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN pending_dialog TEXT;

-- +goose Down
ALTER TABLE sessions DROP COLUMN pending_dialog;
ALTER TABLE sessions DROP COLUMN waiting_for_input;
