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




-- name: GetPrompt :one
SELECT * FROM prompts WHERE id = sqlc.arg(id);

-- name: LatestPrompt :one
SELECT * FROM prompts
WHERE scope = sqlc.arg(scope)
  AND coalesce(agent_id, 0) = sqlc.arg(agent_id)
  AND coalesce(team_id, 0) = sqlc.arg(team_id)
  AND coalesce(group_id, 0) = sqlc.arg(group_id)
  AND coalesce(project_id, '') = sqlc.arg(project_id)
ORDER BY version DESC
LIMIT 1;

-- name: LatestPromptVersion :one
SELECT CAST(coalesce(max(version), 0) AS BIGINT) FROM prompts
WHERE scope = sqlc.arg(scope)
  AND coalesce(agent_id, 0) = sqlc.arg(agent_id)
  AND coalesce(team_id, 0) = sqlc.arg(team_id)
  AND coalesce(group_id, 0) = sqlc.arg(group_id)
  AND coalesce(project_id, '') = sqlc.arg(project_id);

-- name: ListPromptVersions :many
SELECT * FROM prompts
WHERE scope = sqlc.arg(scope)
  AND coalesce(agent_id, 0) = sqlc.arg(agent_id)
  AND coalesce(team_id, 0) = sqlc.arg(team_id)
  AND coalesce(group_id, 0) = sqlc.arg(group_id)
  AND coalesce(project_id, '') = sqlc.arg(project_id)
  AND (sqlc.arg(before_version) = 0 OR version < sqlc.arg(before_version))
ORDER BY version DESC
LIMIT sqlc.arg(max_count);

-- name: InsertPrompt :one
INSERT INTO prompts (
  scope, agent_id, team_id, group_id, project_id, content, version, created_at,
  author_actor_id, approved_by_actor_id, note, proposal_id, auto_approved
)
VALUES (
  sqlc.arg(scope),
  NULLIF(sqlc.arg(agent_id), 0),
  NULLIF(sqlc.arg(team_id), 0),
  NULLIF(sqlc.arg(group_id), 0),
  NULLIF(sqlc.arg(project_id), ''),
  sqlc.arg(content),
  sqlc.arg(version),
  sqlc.arg(created_at),
  NULLIF(sqlc.arg(author_actor_id), 0),
  NULLIF(sqlc.arg(approved_by_actor_id), 0),
  sqlc.arg(note),
  NULLIF(sqlc.arg(proposal_id), 0),
  sqlc.arg(auto_approved)
)
RETURNING *;
