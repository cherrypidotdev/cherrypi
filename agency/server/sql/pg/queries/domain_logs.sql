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

-- The admin domain-log explorer (phase 7): one UNION over the five scoped
-- domain logs, newest first. There is no central event table. Pagination is a
-- keyset cursor on (at, scope, entry_id), which is stable across the union
-- because `at` is milliseconds and ties are broken by scope then row id.

-- name: ListDomainLogs :many
WITH entries AS (
  SELECT
    'task' AS scope,
    l.id AS entry_id,
    t.id AS scope_id,
    CAST(CASE WHEN t.name <> '' THEN t.name ELSE t.id END AS TEXT) AS scope_label,
    l.at AS at,
    l.actor_id AS actor_id,
    l.kind AS kind,
    l.field AS field,
    l.old_value AS old_value,
    l.new_value AS new_value,
    l.message AS message,
    l.data_json AS data_json
  FROM task_logs l
  JOIN tasks t ON t.rowid = l.task_rowid
  UNION ALL
  SELECT
    'agent', l.id, CAST(l.agent_actor_id AS TEXT), CAST(COALESCE(a.name, a.username, '') AS TEXT),
    l.at, l.actor_id, l.kind, l.field, l.old_value, l.new_value, l.message, l.data_json
  FROM agent_domain_logs l
  LEFT JOIN actors a ON a.id = l.agent_actor_id
  UNION ALL
  SELECT
    'project', l.id, l.project_id, CAST(COALESCE(p.name, l.project_id) AS TEXT),
    l.at, l.actor_id, l.kind, l.field, l.old_value, l.new_value, l.message, l.data_json
  FROM project_logs l
  LEFT JOIN projects p ON p.id = l.project_id
  UNION ALL
  SELECT
    'team', l.id, CAST(l.team_id AS TEXT), CAST(COALESCE(tm.name, '') AS TEXT),
    l.at, l.actor_id, l.kind, l.field, l.old_value, l.new_value, l.message, l.data_json
  FROM team_logs l
  LEFT JOIN teams tm ON tm.id = l.team_id
  UNION ALL
  SELECT
    'user', l.id, CAST(l.user_actor_id AS TEXT), CAST(COALESCE(au.username, '') AS TEXT),
    l.at, l.actor_id, l.kind, l.field, l.old_value, l.new_value, l.message, l.data_json
  FROM user_logs l
  LEFT JOIN actors au ON au.id = l.user_actor_id
)
SELECT
  e.scope, e.entry_id, e.scope_id, e.scope_label, e.at, e.actor_id, e.kind,
  e.field, e.old_value, e.new_value, e.message, e.data_json,
  a.kind AS actor_kind,
  a.username AS actor_username,
  a.name AS actor_name
FROM entries e
LEFT JOIN actors a ON a.id = e.actor_id
WHERE
  (CAST(sqlc.narg('actor') AS TEXT) IS NULL OR lower(a.username) = lower(CAST(sqlc.narg('actor') AS TEXT)))
  AND (CAST(sqlc.narg('scope') AS TEXT) IS NULL OR e.scope = CAST(sqlc.narg('scope') AS TEXT))
  AND (CAST(sqlc.narg('kind') AS TEXT) IS NULL OR e.kind = CAST(sqlc.narg('kind') AS TEXT))
  AND (CAST(sqlc.narg('since') AS BIGINT) IS NULL OR e.at >= CAST(sqlc.narg('since') AS BIGINT))
  AND (CAST(sqlc.narg('until') AS BIGINT) IS NULL OR e.at <= CAST(sqlc.narg('until') AS BIGINT))
  AND (CAST(sqlc.narg('search') AS TEXT) IS NULL OR lower(e.message) LIKE '%' || lower(CAST(sqlc.narg('search') AS TEXT)) || '%')
  AND (
    CAST(sqlc.narg('cursor_at') AS BIGINT) IS NULL
    OR e.at < CAST(sqlc.narg('cursor_at') AS BIGINT)
    OR (
      e.at = CAST(sqlc.narg('cursor_at') AS BIGINT)
      AND (
        e.scope > CAST(sqlc.narg('cursor_scope') AS TEXT)
        OR (
          e.scope = CAST(sqlc.narg('cursor_scope') AS TEXT)
          AND e.entry_id < CAST(sqlc.narg('cursor_id') AS BIGINT)
        )
      )
    )
  )
ORDER BY e.at DESC, e.scope ASC, e.entry_id DESC
LIMIT CAST(sqlc.arg('max_count') AS BIGINT);

-- name: PruneTaskLogs :exec
DELETE FROM task_logs WHERE at < sqlc.arg(before_at);

-- name: PruneProjectLogs :exec
DELETE FROM project_logs WHERE at < sqlc.arg(before_at);

-- name: PruneTeamLogs :exec
DELETE FROM team_logs WHERE at < sqlc.arg(before_at);
