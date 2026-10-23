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

-- name: AppendTaskSessionEntry :execrows
-- Append one Pi session entry to a task's canonical history. The next `seq` is
-- derived from the current maximum, and the unique `(task_id, entry_id)` index
-- makes a replayed push (after a lost acknowledgment) a no-op.
INSERT INTO task_session_entries (
  task_id, seq, entry_id, parent_id, type, raw, created_at
) VALUES (
  sqlc.arg(task_id),
  (SELECT COALESCE(MAX(seq), 0) + 1
    FROM task_session_entries
    WHERE task_id = sqlc.arg(task_id)),
  sqlc.arg(entry_id),
  sqlc.arg(parent_id),
  sqlc.arg(type),
  sqlc.arg(raw),
  sqlc.arg(created_at)
)
ON CONFLICT DO NOTHING;

-- name: ListTaskSessionEntries :many
-- The full ordered conversation, used to reconstruct a resumable Pi session
-- file on a fresh agent.
SELECT * FROM task_session_entries
WHERE task_id = $1
ORDER BY seq;

-- name: CountTaskSessionEntries :one
SELECT COUNT(*) FROM task_session_entries WHERE task_id = $1;

-- name: ListTaskHistoryTaskIDs :many
-- The ids of every task with at least one stored session entry, so a list can
-- mark which tasks are migrated without a query per task.
SELECT DISTINCT task_id FROM task_session_entries;

-- name: TaskSessionEntryTip :one
-- The newest entry's Pi entry id, or '' when the task has no stored history.
SELECT
  CAST(COALESCE(MAX(seq), 0) AS BIGINT) AS seq,
  CAST(COALESCE(
    (SELECT e2.entry_id
      FROM task_session_entries e2
      WHERE e2.task_id = sqlc.arg(task_id)
      ORDER BY e2.seq DESC
      LIMIT 1),
    ''
  ) AS TEXT) AS entry_id
FROM task_session_entries e
WHERE e.task_id = sqlc.arg(task_id);
