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

-- name: InsertAgentUsage :exec
INSERT INTO agent_usage (agent_id, session_id, model, input_tokens, output_tokens, cost, active_ms, at)
VALUES (
  sqlc.arg(agent_id), sqlc.arg(session_id), sqlc.arg(model), sqlc.arg(input_tokens),
  sqlc.arg(output_tokens), sqlc.arg(cost), sqlc.arg(active_ms), sqlc.arg(at)
);

-- name: ListAgentUsage :many


SELECT
  ag.agent_id,
  ag.session_id,
  CAST(COALESCE(SUM(ag.input_tokens), 0) AS INTEGER) AS input_tokens,
  CAST(COALESCE(SUM(ag.output_tokens), 0) AS INTEGER) AS output_tokens,
  CAST(COALESCE(SUM(ag.cost), 0.0) AS REAL) AS cost,
  CAST(COALESCE(SUM(ag.active_ms), 0) AS INTEGER) AS active_ms,
  CAST(COALESCE(MAX(ag.at), 0) AS INTEGER) AS last_at
FROM agent_usage AS ag
LEFT JOIN tasks AS t
  ON t.session_id = ag.session_id
 AND t.assignee_actor_id = (SELECT g.actor_id FROM agents g WHERE g.id = ag.agent_id)
WHERE (CAST(sqlc.narg(agent_id) AS INTEGER) IS NULL OR ag.agent_id = sqlc.narg(agent_id))
  AND (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR t.project_id = sqlc.narg(project_id))
GROUP BY ag.agent_id, ag.session_id
ORDER BY last_at DESC;

-- name: SumAgentUsage :one
SELECT
  CAST(COALESCE(SUM(input_tokens), 0) AS INTEGER) AS input_tokens,
  CAST(COALESCE(SUM(output_tokens), 0) AS INTEGER) AS output_tokens,
  CAST(COALESCE(SUM(cost), 0.0) AS REAL) AS cost
FROM agent_usage;

-- name: ListAgentUsageByDay :many


SELECT
  CAST(ag.at / 86400000 AS INTEGER) AS day,
  CAST(COALESCE(SUM(ag.input_tokens), 0) AS INTEGER) AS input_tokens,
  CAST(COALESCE(SUM(ag.output_tokens), 0) AS INTEGER) AS output_tokens,
  CAST(COALESCE(SUM(ag.cost), 0.0) AS REAL) AS cost
FROM agent_usage AS ag
LEFT JOIN tasks AS t
  ON t.session_id = ag.session_id
 AND t.assignee_actor_id = (SELECT g.actor_id FROM agents g WHERE g.id = ag.agent_id)
WHERE (CAST(sqlc.narg(agent_id) AS INTEGER) IS NULL OR ag.agent_id = sqlc.narg(agent_id))
  AND (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR t.project_id = sqlc.narg(project_id))
GROUP BY day
ORDER BY day;
