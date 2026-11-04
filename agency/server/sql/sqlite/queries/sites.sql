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




-- name: CreateSite :one
INSERT INTO sites (
  id, title, description, only_creator_may_edit, agency_wide,
  creator_actor_id, origin_task_id, attachment_id, created_at, updated_at
) VALUES (
  sqlc.arg(id), sqlc.arg(title), sqlc.arg(description),
  sqlc.arg(only_creator_may_edit), sqlc.arg(agency_wide),
  sqlc.arg(creator_actor_id), sqlc.arg(origin_task_id),
  sqlc.arg(attachment_id), sqlc.arg(created_at), sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetSite :one
SELECT * FROM sites WHERE id = sqlc.arg(id);

-- name: ListSites :many
SELECT * FROM sites ORDER BY lower(title), id;

-- name: ListSitesByOriginTask :many
SELECT * FROM sites
WHERE origin_task_id = sqlc.arg(origin_task_id)
ORDER BY created_at, id;

-- name: UpdateSiteSettings :one
UPDATE sites
SET title = sqlc.arg(title),
    description = sqlc.arg(description),
    only_creator_may_edit = sqlc.arg(only_creator_may_edit),
    agency_wide = sqlc.arg(agency_wide),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateSite :one
UPDATE sites
SET title = sqlc.arg(title),
    description = sqlc.arg(description),
    only_creator_may_edit = sqlc.arg(only_creator_may_edit),
    agency_wide = sqlc.arg(agency_wide),
    attachment_id = sqlc.arg(attachment_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateSiteContent :one
UPDATE sites
SET attachment_id = sqlc.arg(attachment_id), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteSite :exec
DELETE FROM sites WHERE id = sqlc.arg(id);

-- name: SiteIDByAttachment :one
SELECT id FROM sites WHERE attachment_id = sqlc.arg(attachment_id) LIMIT 1;
