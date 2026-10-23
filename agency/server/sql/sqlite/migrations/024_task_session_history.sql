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
-- The canonical copy of each task's Pi session. Pi stores a task's
-- conversation as an append-only JSONL tree on the agent's disk; the agency
-- keeps every non-header entry here, keyed by the agency task rather than by
-- the agent-side Pi session id. That makes the agency the durable home of the
-- conversation: an agent (and its workspace) is treated as ephemeral, so a
-- task can be replayed onto a fresh agent under a new Pi session id.
--
-- `seq` is the agency's own 1-based order; `entry_id` is Pi's entry id and is
-- unique per task so replayed pushes are idempotent. `raw` is the verbatim
-- JSONL line, handed back to an agent to reconstruct a resumable session file.
CREATE TABLE task_session_entries (
  task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  entry_id TEXT NOT NULL,
  parent_id TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL,
  raw TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (task_id, seq)
);

CREATE UNIQUE INDEX idx_task_session_entries_entry
  ON task_session_entries (task_id, entry_id);

-- The agent that owns the task's current Pi session. It is set when the task
-- starts and survives a later reassignment, so the scheduler can tell a
-- same-agent resume (reuse the warm Pi session) from a cross-agent
-- continuation (start a fresh Pi session seeded from the stored history).
ALTER TABLE tasks ADD COLUMN session_agent_id TEXT NOT NULL DEFAULT '';

-- Backfill: a session predates cross-agent continuation, so it always lived
-- on the task's current assignee. Without this an existing started task would
-- look like a cross-agent continuation and be restarted instead of resumed.
UPDATE tasks
SET session_agent_id = assignee_agent_id
WHERE session_id <> '' AND assignee_agent_id IS NOT NULL;

-- +goose Down
ALTER TABLE tasks DROP COLUMN session_agent_id;
DROP INDEX idx_task_session_entries_entry;
DROP TABLE task_session_entries;
