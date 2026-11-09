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
INSERT INTO agent_usage (
  agent_id, session_id, provider, model, input_tokens, output_tokens, cost,
  active_ms, tool_ms, model_wait_ms, ttft_ms, at
)
VALUES (
  sqlc.arg(agent_id), sqlc.arg(session_id), sqlc.arg(provider), sqlc.arg(model),
  sqlc.arg(input_tokens), sqlc.arg(output_tokens), sqlc.arg(cost),
  sqlc.arg(active_ms), sqlc.arg(tool_ms), sqlc.arg(model_wait_ms),
  sqlc.arg(ttft_ms), sqlc.arg(at)
);

-- name: ListAgentUsage :many


SELECT
  ag.agent_id,
  ag.session_id,
  CAST(COALESCE(SUM(ag.input_tokens), 0) AS BIGINT) AS input_tokens,
  CAST(COALESCE(SUM(ag.output_tokens), 0) AS BIGINT) AS output_tokens,
  CAST(COALESCE(SUM(ag.cost), 0.0) AS DOUBLE PRECISION) AS cost,
  CAST(COALESCE(SUM(ag.active_ms), 0) AS BIGINT) AS active_ms,
  CAST(COALESCE(SUM(ag.tool_ms), 0) AS BIGINT) AS tool_ms,
  CAST(COALESCE(SUM(ag.model_wait_ms), 0) AS BIGINT) AS model_wait_ms,
  CAST(COALESCE(SUM(ag.ttft_ms), 0) AS BIGINT) AS ttft_ms,
  CAST(COUNT(CASE WHEN ag.ttft_ms > 0 THEN 1 END) AS BIGINT) AS ttft_runs,
  CAST(COUNT(*) AS BIGINT) AS runs,
  CAST(COALESCE(MAX(ag.at), 0) AS BIGINT) AS last_at
FROM agent_usage AS ag
LEFT JOIN tasks AS t
  ON t.session_id = ag.session_id
 AND t.assignee_actor_id = (SELECT g.actor_id FROM agents g WHERE g.id = ag.agent_id)
WHERE (CAST(sqlc.narg(agent_id) AS BIGINT) IS NULL OR ag.agent_id = sqlc.narg(agent_id))
  AND (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR t.project_id = sqlc.narg(project_id))
  AND (CAST(sqlc.narg(provider) AS TEXT) IS NULL OR ag.provider = sqlc.narg(provider))
  AND (CAST(sqlc.narg(model) AS TEXT) IS NULL OR ag.model = sqlc.narg(model))
  AND (CAST(sqlc.narg(since_ms) AS BIGINT) IS NULL OR ag.at >= sqlc.narg(since_ms))
  AND (CAST(sqlc.narg(until_ms) AS BIGINT) IS NULL OR ag.at <= sqlc.narg(until_ms))
GROUP BY ag.agent_id, ag.session_id
ORDER BY last_at DESC;

-- name: SumAgentUsage :one
SELECT
  CAST(COALESCE(SUM(input_tokens), 0) AS BIGINT) AS input_tokens,
  CAST(COALESCE(SUM(output_tokens), 0) AS BIGINT) AS output_tokens,
  CAST(COALESCE(SUM(cost), 0.0) AS DOUBLE PRECISION) AS cost
FROM agent_usage;

-- name: ListAgentUsageByDay :many


SELECT
  CAST(ag.at / 86400000 AS BIGINT) AS day,
  CAST(COALESCE(SUM(ag.input_tokens), 0) AS BIGINT) AS input_tokens,
  CAST(COALESCE(SUM(ag.output_tokens), 0) AS BIGINT) AS output_tokens,
  CAST(COALESCE(SUM(ag.cost), 0.0) AS DOUBLE PRECISION) AS cost,
  CAST(COUNT(*) AS BIGINT) AS runs
FROM agent_usage AS ag
LEFT JOIN tasks AS t
  ON t.session_id = ag.session_id
 AND t.assignee_actor_id = (SELECT g.actor_id FROM agents g WHERE g.id = ag.agent_id)
WHERE (CAST(sqlc.narg(agent_id) AS BIGINT) IS NULL OR ag.agent_id = sqlc.narg(agent_id))
  AND (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR t.project_id = sqlc.narg(project_id))
  AND (CAST(sqlc.narg(provider) AS TEXT) IS NULL OR ag.provider = sqlc.narg(provider))
  AND (CAST(sqlc.narg(model) AS TEXT) IS NULL OR ag.model = sqlc.narg(model))
  AND (CAST(sqlc.narg(since_ms) AS BIGINT) IS NULL OR ag.at >= sqlc.narg(since_ms))
  AND (CAST(sqlc.narg(until_ms) AS BIGINT) IS NULL OR ag.at <= sqlc.narg(until_ms))
GROUP BY day
ORDER BY day;

-- name: ListAgentUsageByModel :many




SELECT
  ag.provider,
  ag.model,
  CAST(COALESCE(SUM(ag.input_tokens), 0) AS BIGINT) AS input_tokens,
  CAST(COALESCE(SUM(ag.output_tokens), 0) AS BIGINT) AS output_tokens,
  CAST(COALESCE(SUM(ag.cost), 0.0) AS DOUBLE PRECISION) AS cost,
  CAST(COALESCE(SUM(ag.active_ms), 0) AS BIGINT) AS active_ms,
  CAST(COALESCE(SUM(ag.tool_ms), 0) AS BIGINT) AS tool_ms,
  CAST(COALESCE(SUM(ag.model_wait_ms), 0) AS BIGINT) AS model_wait_ms,
  CAST(COALESCE(SUM(ag.ttft_ms), 0) AS BIGINT) AS ttft_ms,
  CAST(COUNT(CASE WHEN ag.ttft_ms > 0 THEN 1 END) AS BIGINT) AS ttft_runs,
  CAST(COUNT(*) AS BIGINT) AS runs,
  CAST(COUNT(DISTINCT ag.session_id) AS BIGINT) AS tasks
FROM agent_usage AS ag
LEFT JOIN tasks AS t
  ON t.session_id = ag.session_id
 AND t.assignee_actor_id = (SELECT g.actor_id FROM agents g WHERE g.id = ag.agent_id)
WHERE (CAST(sqlc.narg(agent_id) AS BIGINT) IS NULL OR ag.agent_id = sqlc.narg(agent_id))
  AND (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR t.project_id = sqlc.narg(project_id))
  AND (CAST(sqlc.narg(provider) AS TEXT) IS NULL OR ag.provider = sqlc.narg(provider))
  AND (CAST(sqlc.narg(model) AS TEXT) IS NULL OR ag.model = sqlc.narg(model))
  AND (CAST(sqlc.narg(since_ms) AS BIGINT) IS NULL OR ag.at >= sqlc.narg(since_ms))
  AND (CAST(sqlc.narg(until_ms) AS BIGINT) IS NULL OR ag.at <= sqlc.narg(until_ms))
GROUP BY ag.provider, ag.model
ORDER BY cost DESC, ag.provider, ag.model;

-- name: ListAgentUsageModels :many


SELECT DISTINCT ag.provider, ag.model
FROM agent_usage AS ag
LEFT JOIN tasks AS t
  ON t.session_id = ag.session_id
 AND t.assignee_actor_id = (SELECT g.actor_id FROM agents g WHERE g.id = ag.agent_id)
WHERE (CAST(sqlc.narg(agent_id) AS BIGINT) IS NULL OR ag.agent_id = sqlc.narg(agent_id))
  AND (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR t.project_id = sqlc.narg(project_id))
ORDER BY ag.provider, ag.model;

-- name: ListAgentUsageRuns :many




SELECT * FROM agent_usage
WHERE session_id = sqlc.arg(session_id)
ORDER BY at ASC, id ASC;
