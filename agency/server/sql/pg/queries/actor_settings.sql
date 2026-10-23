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

-- Per-user settings: a small key/value store scoped to an actor (the signed-in
-- user). Values are opaque strings the HTTP layer encodes; a missing key means
-- the caller's default applies.

-- name: GetActorSetting :one
SELECT value FROM actor_settings WHERE actor_id = $1 AND key = $2;

-- name: ListActorSettings :many
SELECT key, value FROM actor_settings WHERE actor_id = $1 ORDER BY key;

-- name: UpsertActorSetting :exec
INSERT INTO actor_settings (actor_id, key, value) VALUES ($1, $2, $3)
ON CONFLICT(actor_id, key) DO UPDATE SET value = excluded.value;

-- name: DeleteActorSetting :exec
DELETE FROM actor_settings WHERE actor_id = $1 AND key = $2;
