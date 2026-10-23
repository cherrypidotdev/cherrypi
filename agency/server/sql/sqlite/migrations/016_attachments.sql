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
-- File storage: a blob registry plus the links that give blobs meaning.
--
-- `attachments` is storage data only: it records where an object lives, never
-- who owns it. Ownership lives in `task_attachments` (a task's file tray,
-- including agent artifacts) and `message_attachments` (a user prompt's
-- files), while a project's/agent's picture is a `picture_attachment_id`
-- column. Because the registry has no owner, deletes only remove links; the
-- attachments service garbage-collects blobs that lost their last reference
-- and deletes their objects.
--
-- Task links key on `tasks.rowid`, not the mutable `<prefix>-<seq>` string id,
-- so editing a project's prefix never touches them.

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

-- The agency's ordered log of the user prompts it sent a task. `id` is the
-- client-supplied (or agency-generated) message id; the nth row lines up with
-- the nth UserMessage in Pi's history so a console can align them.
CREATE TABLE task_messages (
  id TEXT PRIMARY KEY,
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  text TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_task_messages_task ON task_messages (task_rowid, created_at, id);

-- A message's files (many per message). The link is created when the prompt is
-- accepted; the attachment must already exist.
CREATE TABLE message_attachments (
  message_id TEXT NOT NULL REFERENCES task_messages (id) ON DELETE CASCADE,
  attachment_id INTEGER NOT NULL REFERENCES attachments (id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (message_id, attachment_id)
);
CREATE INDEX idx_message_attachments_attachment ON message_attachments (attachment_id);

-- A project's / agent's picture is a foreign key to one attachment row, NULL
-- when it uses the jdenticon fallback.
ALTER TABLE projects ADD COLUMN picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL;
ALTER TABLE agents ADD COLUMN picture_attachment_id INTEGER REFERENCES attachments (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE agents DROP COLUMN picture_attachment_id;
ALTER TABLE projects DROP COLUMN picture_attachment_id;
DROP TABLE message_attachments;
DROP TABLE task_messages;
DROP TABLE task_attachments;
DROP TABLE attachments;
