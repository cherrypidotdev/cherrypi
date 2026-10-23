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

-- Agent operational logs (see the 019 migration). Newest first, keyed by the
-- autoincrement id, which doubles as the pagination cursor.

-- name: InsertAgentLog :exec
INSERT INTO agent_logs (
  agent_id, at, level, message, task_id, session_id, attrs_json
)
VALUES (
  sqlc.arg(agent_id),
  sqlc.arg(at),
  sqlc.arg(level),
  sqlc.arg(message),
  sqlc.arg(task_id),
  sqlc.arg(session_id),
  sqlc.arg(attrs_json)
);

-- name: PruneAgentLogs :exec
-- Keep only the newest `keep` lines for one agent; a no-op until it exceeds
-- the cap. Run right after InsertAgentLog, in the same transaction.
DELETE FROM agent_logs
WHERE agent_logs.agent_id = sqlc.arg(agent_id)
  AND agent_logs.id NOT IN (
    SELECT kept.id FROM agent_logs AS kept
    WHERE kept.agent_id = sqlc.arg(agent_id)
    ORDER BY kept.id DESC
    LIMIT CAST(sqlc.arg(keep) AS INTEGER)
  );

-- name: ListAgentLogs :many
-- `min_level` filters by severity (0 = every level); `before_id` pages older
-- than a cursor (0 = newest first).
SELECT id, agent_id, at, level, message, task_id, session_id, attrs_json
FROM agent_logs
WHERE agent_id = sqlc.arg(agent_id)
  AND level >= sqlc.arg(min_level)
  AND (sqlc.arg(before_id) = 0 OR id < sqlc.arg(before_id))
ORDER BY id DESC
LIMIT CAST(sqlc.arg(max_count) AS INTEGER);

-- name: DeleteAgentLogs :exec
DELETE FROM agent_logs WHERE agent_id = ?;
