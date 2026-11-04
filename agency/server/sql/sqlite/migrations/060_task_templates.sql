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




CREATE TABLE task_templates (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  archived_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);


CREATE UNIQUE INDEX idx_task_templates_active_name
  ON task_templates (project_id, lower(name)) WHERE archived_at IS NULL;
CREATE INDEX idx_task_templates_project ON task_templates (project_id, archived_at);

CREATE TABLE task_template_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  template_id TEXT NOT NULL REFERENCES task_templates (id) ON DELETE CASCADE,
  version INTEGER NOT NULL,
  body TEXT NOT NULL,
  title_template TEXT NOT NULL DEFAULT '',
  args_json TEXT NOT NULL DEFAULT '[]',
  default_priority INTEGER NOT NULL DEFAULT 0,
  default_size TEXT NOT NULL DEFAULT 'medium'
    CHECK (default_size IN ('small', 'medium', 'big', 'epic')),
  default_tag_ids_json TEXT NOT NULL DEFAULT '[]',
  default_team_ids_json TEXT NOT NULL DEFAULT '[]',
  default_lead_actor_ids_json TEXT NOT NULL DEFAULT '[]',
  created_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  UNIQUE (template_id, version)
);
CREATE INDEX idx_task_template_versions_template
  ON task_template_versions (template_id, version DESC);




CREATE TABLE task_template_applications (
  task_rowid INTEGER PRIMARY KEY REFERENCES tasks (rowid) ON DELETE CASCADE,
  template_id TEXT REFERENCES task_templates (id) ON DELETE SET NULL,
  template_version INTEGER NOT NULL,
  template_name TEXT NOT NULL,
  args_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_task_template_applications_template
  ON task_template_applications (template_id);

-- +goose Down
DROP TABLE task_template_applications;
DROP TABLE task_template_versions;
DROP TABLE task_templates;
