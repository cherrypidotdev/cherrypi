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
-- A human-friendly display name. It defaults to a truncated form of the
-- session's initial prompt and can be edited; empty falls back to the prompt
-- for rows written before this migration.
ALTER TABLE sessions ADD COLUMN name TEXT NOT NULL DEFAULT '';

-- When set, the session is archived: hidden from the main list under
-- "Archived sessions". Archiving also ends the session; unarchiving only
-- un-hides it (the session stays ended until a new message).
ALTER TABLE sessions ADD COLUMN archived_at INTEGER;

UPDATE sessions SET name = substr(replace(prompt, char(10), ' '), 1, 80);

CREATE INDEX idx_sessions_archived_at ON sessions (archived_at);

-- +goose Down
DROP INDEX idx_sessions_archived_at;
ALTER TABLE sessions DROP COLUMN archived_at;
ALTER TABLE sessions DROP COLUMN name;
