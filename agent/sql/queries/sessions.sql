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

-- name: CreateSession :one
INSERT INTO tasks (
  id, status, prompt, name, workspace, pi_session_id, pending_prompt, interactive,
  append_system_prompt, agency_project_id, agency_task_seq, created_at, updated_at
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetSession :one
SELECT * FROM tasks WHERE id = ?;

-- name: ListSessions :many
SELECT * FROM tasks ORDER BY created_at DESC;

-- name: ListActiveSessions :many
SELECT * FROM tasks WHERE status = ? ORDER BY created_at;

-- name: SetSessionPendingPrompt :exec
UPDATE tasks
SET pending_prompt = ?, updated_at = ?
WHERE id = ?;

-- name: ClearSessionPendingPrompt :exec
UPDATE tasks
SET pending_prompt = NULL, updated_at = ?
WHERE id = ?;

-- name: SetSessionAgencyRef :exec
UPDATE tasks
SET agency_project_id = ?, agency_task_seq = ?, updated_at = ?
WHERE id = ?;

-- name: SetSessionPid :exec
UPDATE tasks
SET pid = ?, updated_at = ?
WHERE id = ?;

-- name: SetSessionWaiting :exec
UPDATE tasks
SET waiting_for_input = 1, pending_dialog = ?, updated_at = ?
WHERE id = ?;

-- name: ReactivateSession :exec
UPDATE tasks
SET status = ?, error = NULL, ended_at = NULL, updated_at = ?
WHERE id = ?;

-- name: ClearSessionWaiting :exec
UPDATE tasks
SET waiting_for_input = 0, pending_dialog = NULL, updated_at = ?
WHERE id = ?;

-- name: SetSessionName :exec
UPDATE tasks
SET name = ?, updated_at = ?
WHERE id = ?;

-- name: EndSession :exec
UPDATE tasks
SET status = ?, error = ?, ended_at = ?, updated_at = ?, pid = NULL, pending_prompt = NULL,
    waiting_for_input = 0, pending_dialog = NULL
WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM tasks WHERE id = ?;
