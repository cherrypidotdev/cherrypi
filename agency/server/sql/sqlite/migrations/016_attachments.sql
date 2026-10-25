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













CREATE TABLE attachments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  storage_key TEXT NOT NULL UNIQUE,
  filename TEXT NOT NULL,
  content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
  size INTEGER NOT NULL DEFAULT 0,
  created_by TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);

CREATE TABLE task_attachments (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  attachment_id INTEGER NOT NULL REFERENCES attachments (id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (task_rowid, attachment_id)
);
CREATE INDEX idx_task_attachments_attachment ON task_attachments (attachment_id);




CREATE TABLE task_messages (
  id TEXT PRIMARY KEY,
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  text TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_task_messages_task ON task_messages (task_rowid, created_at, id);



CREATE TABLE message_attachments (
  message_id TEXT NOT NULL REFERENCES task_messages (id) ON DELETE CASCADE,
  attachment_id INTEGER NOT NULL REFERENCES attachments (id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (message_id, attachment_id)
);
CREATE INDEX idx_message_attachments_attachment ON message_attachments (attachment_id);



ALTER TABLE projects ADD COLUMN picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL;
ALTER TABLE agents ADD COLUMN picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE agents DROP COLUMN picture_attachment_id;
ALTER TABLE projects DROP COLUMN picture_attachment_id;
DROP TABLE message_attachments;
DROP TABLE task_messages;
DROP TABLE task_attachments;
DROP TABLE attachments;
