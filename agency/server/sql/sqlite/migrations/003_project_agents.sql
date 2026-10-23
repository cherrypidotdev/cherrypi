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
-- Explicit agent ↔ project membership. A project's agents are its members:
-- only a member agent can create a session in the project, and an agent can be
-- added to (or removed from) a project independently of whether it has
-- sessions. Removing a member requires deleting its sessions in the project
-- first, so the link cascades only with the project or the agent.
CREATE TABLE project_agents (
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (project_id, agent_id)
);

CREATE INDEX idx_project_agents_agent_id ON project_agents (agent_id);

-- Backfill memberships from sessions created before memberships existed, so
-- existing project sessions keep their agent on the project.
INSERT OR IGNORE INTO project_agents (project_id, agent_id, created_at)
SELECT project_id, agent_id, MIN(created_at)
FROM session_context
WHERE project_id IS NOT NULL
GROUP BY project_id, agent_id;

-- +goose Down
DROP TABLE project_agents;
