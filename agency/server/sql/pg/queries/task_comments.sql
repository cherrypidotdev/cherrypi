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






-- name: CreateTaskComment :one
INSERT INTO task_comments (
  task_rowid, author_actor_id, body, created_at, updated_at
) VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.narg(author_actor_id),
  sqlc.arg(body),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetTaskComment :one
SELECT * FROM task_comments WHERE id = sqlc.arg(id);

-- name: GetTaskCommentTaskID :one
SELECT t.id FROM task_comments c
JOIN tasks t ON t.rowid = c.task_rowid
WHERE c.id = sqlc.arg(id);

-- name: UpdateTaskCommentBody :exec
UPDATE task_comments
SET body = sqlc.arg(body),
    updated_at = sqlc.arg(updated_at),
    edited_at = sqlc.arg(edited_at)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: SoftDeleteTaskComment :exec
UPDATE task_comments
SET deleted_at = sqlc.arg(deleted_at),
    deleted_by_actor_id = sqlc.narg(deleted_by_actor_id)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ListTaskComments :many
SELECT * FROM task_comments
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY created_at, id;

-- name: LinkTaskCommentAttachment :exec
INSERT INTO task_comment_attachments (comment_id, attachment_id, position)
VALUES (sqlc.arg(comment_id), sqlc.arg(attachment_id), sqlc.arg(position))
ON CONFLICT DO NOTHING;

-- name: DeleteTaskCommentAttachments :exec
DELETE FROM task_comment_attachments WHERE comment_id = sqlc.arg(comment_id);

-- name: DeleteTaskCommentAttachmentsByAttachment :exec
DELETE FROM task_comment_attachments WHERE attachment_id = sqlc.arg(attachment_id);

-- name: ListTaskCommentAttachments :many


SELECT
  tca.comment_id,
  a.id AS attachment_id,
  a.storage_key,
  a.filename,
  a.content_type,
  a.size,
  a.created_by,
  a.created_at,
  tca.position
FROM task_comment_attachments tca
JOIN attachments a ON a.id = tca.attachment_id
JOIN task_comments c ON c.id = tca.comment_id
WHERE c.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY tca.comment_id, tca.position, a.id;

-- name: GetTaskCommentRead :one
SELECT last_read_comment_id FROM task_comment_reads
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND actor_id = sqlc.arg(actor_id);

-- name: UpsertTaskCommentRead :exec
INSERT INTO task_comment_reads (
  task_rowid, actor_id, last_read_comment_id, read_at
) VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(actor_id),
  sqlc.arg(last_read_comment_id),
  sqlc.arg(read_at)
)
ON CONFLICT(task_rowid, actor_id) DO UPDATE SET
  last_read_comment_id = excluded.last_read_comment_id,
  read_at = excluded.read_at;

-- name: CountUnreadTaskComments :one
SELECT CAST(COUNT(*) AS BIGINT) FROM task_comments c
LEFT JOIN task_comment_reads r
  ON r.task_rowid = c.task_rowid AND r.actor_id = sqlc.arg(actor_id)
WHERE c.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND c.deleted_at IS NULL
  AND COALESCE(c.author_actor_id, -1) <> sqlc.arg(actor_id)
  AND c.id > COALESCE(r.last_read_comment_id, 0);

-- name: ListUnreadTaskCommentCounts :many


SELECT t.id AS task_id, CAST(COUNT(*) AS BIGINT) AS unread
FROM task_comments c
JOIN tasks t ON t.rowid = c.task_rowid
LEFT JOIN task_comment_reads r
  ON r.task_rowid = c.task_rowid AND r.actor_id = sqlc.arg(actor_id)
WHERE c.deleted_at IS NULL
  AND COALESCE(c.author_actor_id, -1) <> sqlc.arg(actor_id)
  AND c.id > COALESCE(r.last_read_comment_id, 0)
GROUP BY t.id;

-- name: DeleteTaskCommentAttachmentsByTask :exec
DELETE FROM task_comment_attachments WHERE comment_id IN (
  SELECT id FROM task_comments
  WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
);

-- name: DeleteTaskComments :exec
DELETE FROM task_comments
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskCommentReads :exec
DELETE FROM task_comment_reads
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskCommentAttachmentsByProjectAgent :exec
DELETE FROM task_comment_attachments WHERE comment_id IN (
  SELECT c.id FROM task_comments c WHERE c.task_rowid IN (
    SELECT rowid FROM tasks
    WHERE project_id = sqlc.arg(project_id)
      AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
  )
);

-- name: DeleteTaskCommentsByProjectAgent :exec
DELETE FROM task_comments WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id)
    AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteTaskCommentReadsByProjectAgent :exec
DELETE FROM task_comment_reads WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id)
    AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteTaskCommentReadsByActor :exec
DELETE FROM task_comment_reads WHERE actor_id = sqlc.arg(actor_id);
