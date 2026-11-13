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




-- name: ListProjectTagReservations :many
SELECT * FROM project_tag_reservations
WHERE project_id = $1
ORDER BY tag_name;

-- name: DeleteProjectTagReservations :exec
DELETE FROM project_tag_reservations WHERE project_id = $1;

-- name: InsertProjectTagReservation :exec
INSERT INTO project_tag_reservations (
  project_id, tag_name, reserved, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5);

-- name: ListAgentProjectTagReservations :many


WITH agent_actor AS (
  SELECT a.actor_id FROM agents a WHERE a.id = $1
),
roots AS (
  SELECT g.project_id, g.weight
  FROM effective_project_grants g
  JOIN agent_actor ON agent_actor.actor_id = g.actor_id
),
readable AS (
  SELECT project_id, weight FROM roots
  UNION ALL
  SELECT pa.project_id, roots.weight
  FROM project_ancestors pa
  JOIN roots ON roots.project_id = pa.ancestor_id
),
best AS (
  SELECT project_id, MAX(weight) AS weight FROM readable GROUP BY project_id
)
SELECT r.project_id, r.tag_name, r.reserved
FROM project_tag_reservations r
JOIN best ON best.project_id = r.project_id
WHERE best.weight >= 2
ORDER BY r.tag_name;
