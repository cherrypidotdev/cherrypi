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

-- name: InsertUsage :exec
INSERT INTO usage (
  session_id, provider, model, input_tokens, output_tokens, cost, active_ms,
  tool_ms, model_wait_ms, ttft_ms, at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SumUsageBySession :one
SELECT
  CAST(COALESCE(SUM(input_tokens), 0) AS INTEGER) AS input_tokens,
  CAST(COALESCE(SUM(output_tokens), 0) AS INTEGER) AS output_tokens,
  CAST(COALESCE(SUM(cost), 0.0) AS REAL) AS cost,
  CAST(COALESCE(SUM(active_ms), 0) AS INTEGER) AS active_ms,
  CAST(COALESCE(SUM(tool_ms), 0) AS INTEGER) AS tool_ms,
  CAST(COALESCE(SUM(model_wait_ms), 0) AS INTEGER) AS model_wait_ms,
  CAST(COALESCE(SUM(ttft_ms), 0) AS INTEGER) AS ttft_ms,
  CAST(COUNT(CASE WHEN ttft_ms > 0 THEN 1 END) AS INTEGER) AS ttft_runs
FROM usage
WHERE session_id = ?;

-- name: ListUsageBySession :many
SELECT
  session_id,
  CAST(COALESCE(SUM(input_tokens), 0) AS INTEGER) AS input_tokens,
  CAST(COALESCE(SUM(output_tokens), 0) AS INTEGER) AS output_tokens,
  CAST(COALESCE(SUM(cost), 0.0) AS REAL) AS cost,
  CAST(COALESCE(SUM(active_ms), 0) AS INTEGER) AS active_ms,
  CAST(COALESCE(SUM(tool_ms), 0) AS INTEGER) AS tool_ms,
  CAST(COALESCE(SUM(model_wait_ms), 0) AS INTEGER) AS model_wait_ms,
  CAST(COALESCE(SUM(ttft_ms), 0) AS INTEGER) AS ttft_ms,
  CAST(COUNT(CASE WHEN ttft_ms > 0 THEN 1 END) AS INTEGER) AS ttft_runs,
  CAST(COALESCE(MAX(at), 0) AS INTEGER) AS last_at
FROM usage
GROUP BY session_id
ORDER BY last_at DESC;

-- name: ListUsageByDay :many
SELECT
  CAST(at / 86400000 AS INTEGER) AS day,
  CAST(COALESCE(SUM(input_tokens), 0) AS INTEGER) AS input_tokens,
  CAST(COALESCE(SUM(output_tokens), 0) AS INTEGER) AS output_tokens,
  CAST(COALESCE(SUM(cost), 0.0) AS REAL) AS cost
FROM usage
GROUP BY day
ORDER BY day;
