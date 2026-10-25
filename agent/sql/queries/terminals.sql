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

-- name: CreateTerminalRecord :exec
INSERT INTO terminals (
  id, kind, owner, cwd, cols, rows, state, created_at
) VALUES (
  ?, ?, ?, ?, ?, ?, 'open', ?
);

-- name: SetTerminalState :exec
UPDATE terminals
SET state = ?, end_reason = ?, ended_at = ?
WHERE id = ?;

-- name: ListTerminalsByState :many
SELECT * FROM terminals WHERE state = ? ORDER BY created_at;

-- name: InsertTerminalLog :exec
INSERT INTO terminal_logs (
  terminal_id, kind, owner, event, reason, detail, at
) VALUES (
  ?, ?, ?, ?, ?, ?, ?
);

-- name: ListTerminalLogs :many
SELECT * FROM terminal_logs ORDER BY at DESC, id DESC LIMIT ?;
