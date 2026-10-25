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




-- name: GetAgentTaskLimits :one
SELECT * FROM agent_task_limits WHERE agent_id = ?;

-- name: UpsertAgentTaskLimits :exec
INSERT INTO agent_task_limits (agent_id, max_concurrent, max_concurrent_per_project)
VALUES (?, ?, ?)
ON CONFLICT (agent_id) DO UPDATE SET
  max_concurrent = excluded.max_concurrent,
  max_concurrent_per_project = excluded.max_concurrent_per_project;

-- name: ListAgentTagLimits :many
SELECT * FROM agent_tag_limits WHERE agent_id = ? ORDER BY tag_name;

-- name: DeleteAgentTagLimits :exec
DELETE FROM agent_tag_limits WHERE agent_id = ?;

-- name: InsertAgentTagLimit :exec
INSERT INTO agent_tag_limits (agent_id, tag_name, max_concurrent) VALUES (?, ?, ?);

-- name: ListAgentProjectLimits :many
SELECT * FROM agent_project_limits WHERE agent_id = ? ORDER BY project_id;

-- name: DeleteAgentProjectLimits :exec
DELETE FROM agent_project_limits WHERE agent_id = ?;

-- name: InsertAgentProjectLimit :exec
INSERT INTO agent_project_limits (agent_id, project_id, max_concurrent) VALUES (?, ?, ?);

-- name: ListAgentProjectTagLimits :many
SELECT * FROM agent_project_tag_limits
WHERE agent_id = ?
ORDER BY project_id, tag_id;

-- name: DeleteAgentProjectTagLimits :exec
DELETE FROM agent_project_tag_limits WHERE agent_id = ?;

-- name: InsertAgentProjectTagLimit :exec
INSERT INTO agent_project_tag_limits (
  agent_id, project_id, tag_id, max_concurrent
) VALUES (?, ?, ?, ?);
