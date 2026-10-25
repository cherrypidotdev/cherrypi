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











CREATE TABLE task_artifacts (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  artifact_id TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '',
  streaming INTEGER NOT NULL DEFAULT 0 CHECK (streaming IN (0, 1)),
  position INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (task_rowid, artifact_id)
);
CREATE INDEX idx_task_artifacts_task ON task_artifacts (task_rowid, position);

CREATE TABLE artifact_parts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_rowid INTEGER NOT NULL,
  artifact_id TEXT NOT NULL,
  position INTEGER NOT NULL DEFAULT 0,
  kind TEXT NOT NULL CHECK (kind IN ('text', 'file', 'data')),
  text TEXT NOT NULL DEFAULT '',
  data_json TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL,
  filename TEXT NOT NULL DEFAULT '',
  media_type TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_artifact_parts_artifact
  ON artifact_parts (task_rowid, artifact_id, position);
CREATE INDEX idx_artifact_parts_attachment ON artifact_parts (attachment_id);

-- +goose Down
DROP TABLE artifact_parts;
DROP TABLE task_artifacts;
