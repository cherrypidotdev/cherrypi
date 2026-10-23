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

-- +goose Up
-- Agent operational log lines.
--
-- Agents push structured log records (task/session lifecycle, connectivity) on
-- their event stream; the agency persists them here so the dashboard can show
-- what an agent did without shelling into its container. The agent's own
-- stdout keeps the same records.
--
-- `level` is an integer so a `min_level` filter is a simple comparison:
-- 0 debug, 1 info, 2 warn, 3 error. `agent_id` is the registered agent; the
-- public queries take it directly, and history is pruned per agent on insert.
CREATE TABLE agent_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id TEXT NOT NULL,
  at INTEGER NOT NULL,
  level INTEGER NOT NULL DEFAULT 1,
  message TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  session_id TEXT NOT NULL DEFAULT '',
  -- Remaining structured fields as a JSON object, so a client can render them
  -- without parsing the message.
  attrs_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_agent_logs_agent ON agent_logs (agent_id, id DESC);

-- +goose Down
DROP TABLE agent_logs;
