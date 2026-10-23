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

-- name: ListSettings :many
SELECT key, value FROM settings ORDER BY key;

-- name: GetSetting :one
SELECT value FROM settings WHERE key = sqlc.arg(key);

-- name: UpsertSetting :exec
INSERT INTO settings (key, value) VALUES ($1, $2)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: ClearSettingValue :exec
-- Blank a setting only when it still holds the given value, so a stale
-- cleanup never clobbers a newer write.
UPDATE settings SET value = '' WHERE key = sqlc.arg(key) AND value = sqlc.arg(value);
