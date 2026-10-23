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

-- name: CreateWebhook :one
INSERT INTO webhooks (agent_id, url, events, secret, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListWebhooks :many
SELECT * FROM webhooks ORDER BY id;

-- name: GetWebhook :one
SELECT * FROM webhooks WHERE id = $1;

-- name: UpdateWebhook :one
UPDATE webhooks
SET agent_id = $1, url = $2, events = $3, secret = $4, updated_at = $5
WHERE id = $6
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhooks WHERE id = $1;
