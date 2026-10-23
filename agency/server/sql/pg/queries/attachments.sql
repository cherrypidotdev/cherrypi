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

-- The blob registry and its links. Attachments store where an object lives;
-- `task_attachments` and `message_attachments` give it meaning, and a
-- project/agent picture is a `picture_attachment_id` column. Task links key on
-- `tasks.rowid` so a project prefix change never touches them.

-- name: CreateAttachment :one
INSERT INTO attachments (storage_key, filename, content_type, size, created_by, created_at)
VALUES (sqlc.arg(storage_key), sqlc.arg(filename), sqlc.arg(content_type),
        sqlc.arg(size), sqlc.arg(created_by), sqlc.arg(created_at))
RETURNING *;

-- name: GetAttachment :one
SELECT * FROM attachments WHERE id = sqlc.arg(id);

-- name: SetAttachmentSize :exec
UPDATE attachments SET size = sqlc.arg(size) WHERE id = sqlc.arg(id);

-- name: DeleteAttachment :exec
DELETE FROM attachments WHERE id = sqlc.arg(id);

-- name: ListOrphanAttachments :many
-- Blobs no longer referenced anywhere. The GC deletes their object and row.
-- The agency's logo and favicon live as settings keys holding an attachment id
-- (see internal/settings); a blob they name is referenced.
SELECT a.* FROM attachments a
WHERE NOT EXISTS (SELECT 1 FROM task_attachments ta WHERE ta.attachment_id = a.id)
  AND NOT EXISTS (SELECT 1 FROM message_attachments ma WHERE ma.attachment_id = a.id)
  AND NOT EXISTS (SELECT 1 FROM artifact_parts ap WHERE ap.attachment_id = a.id)
  AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.picture_attachment_id = a.id)
  AND NOT EXISTS (SELECT 1 FROM actors ag WHERE ag.picture_attachment_id = a.id)
  AND NOT EXISTS (
    SELECT 1 FROM settings s
    WHERE s.key IN ('agency_logo_attachment_id', 'agency_favicon_attachment_id')
      AND s.value = CAST(a.id AS TEXT)
  );

-- name: LinkTaskAttachment :exec
INSERT INTO task_attachments (task_rowid, attachment_id, position, created_at)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(attachment_id),
  COALESCE((
    SELECT MAX(position) + 1 FROM task_attachments
    WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  ), 0),
  sqlc.arg(created_at)
);

-- name: UnlinkTaskAttachment :exec
DELETE FROM task_attachments
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND attachment_id = sqlc.arg(attachment_id);

-- name: ListTaskAttachments :many
-- A task's tray, each row carrying the message it was sent with (NULL for a
-- tray-only upload). An attachment belongs to at most one message.
SELECT
  a.id, a.storage_key, a.filename, a.content_type, a.size, a.created_by,
  a.created_at, ta.position, ma.message_id
FROM task_attachments ta
JOIN attachments a ON a.id = ta.attachment_id
LEFT JOIN message_attachments ma ON ma.attachment_id = a.id
WHERE ta.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY ta.position, ta.created_at, a.id;

-- name: ListTaskAttachmentIDs :many
SELECT attachment_id FROM task_attachments
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: CreateTaskMessage :exec
INSERT INTO task_messages (id, task_rowid, text, actor_id, created_at)
VALUES (
  sqlc.arg(id),
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(text),
  sqlc.narg(actor_id),
  sqlc.arg(created_at)
)
ON CONFLICT DO NOTHING;

-- name: ListTaskMessages :many
SELECT m.id, m.text, m.actor_id, m.created_at
FROM task_messages m
WHERE m.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY m.created_at, m.id;

-- name: LinkMessageAttachment :exec
INSERT INTO message_attachments (message_id, attachment_id, position)
VALUES (
  sqlc.arg(message_id),
  sqlc.arg(attachment_id),
  COALESCE((
    SELECT MAX(position) + 1 FROM message_attachments
    WHERE message_id = sqlc.arg(message_id)
  ), 0)
);

-- name: SetProjectPicture :exec
UPDATE projects
SET picture_attachment_id = sqlc.arg(picture_attachment_id), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: ClearProjectPicture :exec
UPDATE projects
SET picture_attachment_id = NULL, updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: SetAgentPicture :exec
UPDATE actors
SET picture_attachment_id = sqlc.arg(picture_attachment_id), updated_at = sqlc.arg(updated_at)
WHERE id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
  AND kind = 'agent';

-- name: ClearAgentPicture :exec
UPDATE actors
SET picture_attachment_id = NULL, updated_at = sqlc.arg(updated_at)
WHERE id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
  AND kind = 'agent';

-- name: SetUserPicture :exec
UPDATE actors
SET picture_attachment_id = sqlc.arg(picture_attachment_id), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(actor_id) AND kind = 'user';

-- name: ClearUserPicture :exec
UPDATE actors
SET picture_attachment_id = NULL, updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(actor_id) AND kind = 'user';

-- name: GetProjectPicture :one
SELECT a.id, a.storage_key, a.filename, a.content_type, a.size, a.created_by, a.created_at
FROM projects p
JOIN attachments a ON a.id = p.picture_attachment_id
WHERE p.id = sqlc.arg(project_id);

-- name: GetAgentPicture :one
SELECT a.id, a.storage_key, a.filename, a.content_type, a.size, a.created_by, a.created_at
FROM agents g
JOIN actors ag ON ag.id = g.actor_id
JOIN attachments a ON a.id = ag.picture_attachment_id
WHERE g.id = sqlc.arg(agent_id);

-- name: GetUserPicture :one
SELECT a.id, a.storage_key, a.filename, a.content_type, a.size, a.created_by, a.created_at
FROM actors ag
JOIN attachments a ON a.id = ag.picture_attachment_id
WHERE ag.id = sqlc.arg(actor_id) AND ag.kind = 'user';

-- name: ListProjectPictures :many
SELECT p.id AS owner_id, a.id, a.storage_key, a.filename, a.content_type, a.size,
       a.created_by, a.created_at
FROM projects p
JOIN attachments a ON a.id = p.picture_attachment_id;

-- name: ListAgentPictures :many
SELECT g.id AS owner_id, a.id, a.storage_key, a.filename, a.content_type, a.size,
       a.created_by, a.created_at
FROM agents g
JOIN actors ag ON ag.id = g.actor_id
JOIN attachments a ON a.id = ag.picture_attachment_id;

-- name: ListUserPictures :many
SELECT ag.id AS owner_id, a.id, a.storage_key, a.filename, a.content_type, a.size,
       a.created_by, a.created_at
FROM actors ag
JOIN attachments a ON a.id = ag.picture_attachment_id
WHERE ag.kind = 'user';

-- name: SetAttachmentStorageKey :exec
UPDATE attachments SET storage_key = sqlc.arg(storage_key) WHERE id = sqlc.arg(id);

-- name: DeleteTaskAttachments :exec
DELETE FROM task_attachments
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskAttachmentsByProjectAgent :exec
DELETE FROM task_attachments WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id) AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteMessageAttachmentsByTask :exec
DELETE FROM message_attachments WHERE message_id IN (
  SELECT m.id FROM task_messages m
  WHERE m.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
);

-- name: DeleteMessageAttachmentsByProjectAgent :exec
DELETE FROM message_attachments WHERE message_id IN (
  SELECT m.id FROM task_messages m WHERE m.task_rowid IN (
    SELECT rowid FROM tasks
    WHERE project_id = sqlc.arg(project_id) AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
  )
);

-- name: DeleteTaskMessages :exec
DELETE FROM task_messages
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskMessagesByProjectAgent :exec
DELETE FROM task_messages WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id) AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteAttachmentTaskLinks :exec
DELETE FROM task_attachments WHERE attachment_id = sqlc.arg(attachment_id);

-- name: DeleteAttachmentMessageLinks :exec
DELETE FROM message_attachments WHERE attachment_id = sqlc.arg(attachment_id);

-- name: ClearProjectPictureByAttachment :exec
UPDATE projects SET picture_attachment_id = NULL
WHERE picture_attachment_id = sqlc.arg(attachment_id);

-- name: ClearAgentPictureByAttachment :exec
UPDATE actors SET picture_attachment_id = NULL
WHERE picture_attachment_id = sqlc.arg(attachment_id) AND kind = 'agent';

-- name: ClearUserPictureByAttachment :exec
UPDATE actors SET picture_attachment_id = NULL
WHERE picture_attachment_id = sqlc.arg(attachment_id) AND kind = 'user';
