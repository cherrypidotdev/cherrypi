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



CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  archived_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX idx_projects_archived_at ON projects (archived_at);




CREATE TABLE prompts (
  id INTEGER PRIMARY KEY,
  scope TEXT NOT NULL CHECK (scope IN ('org', 'agent', 'project', 'agent_project')),
  agent_id TEXT REFERENCES agents (id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects (id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  version INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE INDEX idx_prompts_key ON prompts (scope, agent_id, project_id, version DESC);





CREATE TABLE session_context (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  session_id TEXT NOT NULL,
  project_id TEXT REFERENCES projects (id),
  org_prompt_id INTEGER REFERENCES prompts (id),
  agent_prompt_id INTEGER REFERENCES prompts (id),
  project_prompt_id INTEGER REFERENCES prompts (id),
  agent_project_prompt_id INTEGER REFERENCES prompts (id),
  created_at INTEGER NOT NULL,
  PRIMARY KEY (agent_id, session_id)
);

CREATE INDEX idx_session_context_project_id ON session_context (project_id);

-- +goose Down
DROP TABLE session_context;
DROP TABLE prompts;
DROP TABLE projects;
