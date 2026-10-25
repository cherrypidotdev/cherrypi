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

-- name: CreateCredential :one
INSERT INTO credentials (username, password_hash, created_at, updated_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetCredential :one
SELECT * FROM credentials WHERE username = ?;

-- name: ListCredentials :many
SELECT * FROM credentials ORDER BY username;

-- name: DeleteCredential :exec
DELETE FROM credentials WHERE username = ?;
