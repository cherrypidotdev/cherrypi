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

-- name: GetTaskSessionState :one



SELECT * FROM task_session_state WHERE task_id = ?;

-- name: CreateTaskSessionState :execrows



INSERT INTO task_session_state (
  task_id, revision, model_provider, model_id, thinking_level,
  entries_json, updated_at
) VALUES (
  sqlc.arg(task_id), 1, sqlc.arg(model_provider), sqlc.arg(model_id),
  sqlc.arg(thinking_level), sqlc.arg(entries_json),
  sqlc.arg(updated_at)
)
ON CONFLICT DO NOTHING;

-- name: UpdateTaskSessionState :execrows


UPDATE task_session_state SET
  revision = revision + 1,
  model_provider = sqlc.arg(model_provider),
  model_id = sqlc.arg(model_id),
  thinking_level = sqlc.arg(thinking_level),
  entries_json = sqlc.arg(entries_json),
  updated_at = sqlc.arg(updated_at)
WHERE task_id = sqlc.arg(task_id)
  AND revision = sqlc.arg(expected_revision);

-- name: AppendTaskEvent :execrows


INSERT INTO task_events (task_id, seq, at, kind, payload_json)
VALUES (
  sqlc.arg(task_id),
  sqlc.arg(seq),
  sqlc.arg(at),
  sqlc.arg(kind),
  sqlc.arg(payload_json)
)
ON CONFLICT DO NOTHING;

-- name: ListTaskEvents :many

SELECT * FROM task_events WHERE task_id = ? ORDER BY seq;

-- name: LastTaskEventSeq :one


SELECT CAST(COALESCE(MAX(seq), 0) AS INTEGER) AS last_seq
FROM task_events
WHERE task_id = ?;

-- name: PageTaskEvents :many


SELECT * FROM task_events
WHERE task_id = sqlc.arg(task_id)
  AND (
    CAST(sqlc.narg(before) AS INTEGER) IS NULL
    OR seq < CAST(sqlc.narg(before) AS INTEGER)
  )
ORDER BY seq DESC
LIMIT sqlc.arg(page_limit);

-- name: DeleteTaskSessionState :exec


DELETE FROM task_session_state WHERE task_id = ?;

-- name: DeleteTaskEventsForTask :exec

DELETE FROM task_events WHERE task_id = ?;

-- name: DeleteTaskSessionEntriesForTask :exec

DELETE FROM task_session_entries WHERE task_id = ?;
