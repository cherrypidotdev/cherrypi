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




-- name: GetPromptProposal :one
SELECT * FROM prompt_proposals WHERE id = sqlc.arg(id);

-- name: InsertPromptProposal :one
INSERT INTO prompt_proposals (
  scope, agent_id, team_id, project_id, content, base_version,
  author_actor_id, note, status, auto_approved, created_at
)
VALUES (
  sqlc.arg(scope),
  NULLIF(sqlc.arg(agent_id), 0),
  NULLIF(sqlc.arg(team_id), 0),
  NULLIF(sqlc.arg(project_id), ''),
  sqlc.arg(content),
  sqlc.arg(base_version),
  sqlc.arg(author_actor_id),
  sqlc.arg(note),
  'pending',
  0,
  sqlc.arg(created_at)
)
RETURNING *;

-- name: ListPromptProposalsByTarget :many
SELECT * FROM prompt_proposals
WHERE scope = sqlc.arg(scope)
  AND coalesce(agent_id, 0) = sqlc.arg(agent_id)
  AND coalesce(team_id, 0) = sqlc.arg(team_id)
  AND coalesce(project_id, '') = sqlc.arg(project_id)
  AND (CAST(sqlc.narg(status) AS TEXT) IS NULL OR status = CAST(sqlc.narg(status) AS TEXT))
  AND (sqlc.arg(before_id) = 0 OR id < sqlc.arg(before_id))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(max_count);

-- name: ListPromptProposalsByAuthor :many
SELECT * FROM prompt_proposals
WHERE author_actor_id = sqlc.arg(author_actor_id)
  AND (CAST(sqlc.narg(status) AS TEXT) IS NULL OR status = CAST(sqlc.narg(status) AS TEXT))
  AND (sqlc.arg(before_id) = 0 OR id < sqlc.arg(before_id))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(max_count);

-- name: SetPromptProposalDecision :one
UPDATE prompt_proposals
SET status = sqlc.arg(status),
    decided_at = sqlc.arg(decided_at),
    decided_by_actor_id = NULLIF(sqlc.arg(decided_by_actor_id), 0),
    decision_note = sqlc.arg(decision_note),
    auto_approved = sqlc.arg(auto_approved)
WHERE id = sqlc.arg(id) AND status = 'pending'
RETURNING *;

-- name: WithdrawPromptProposal :one
UPDATE prompt_proposals
SET status = 'withdrawn',
    decided_at = sqlc.arg(decided_at),
    decision_note = sqlc.arg(decision_note)
WHERE id = sqlc.arg(id)
  AND author_actor_id = sqlc.arg(author_actor_id)
  AND status = 'pending'
RETURNING *;
