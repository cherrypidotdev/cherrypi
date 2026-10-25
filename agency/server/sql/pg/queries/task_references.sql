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











-- name: UpsertTaskReference :exec


INSERT INTO task_references (task_rowid, referenced_task_rowid, kind, created_at)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(referenced_task_id)),
  sqlc.arg(kind),
  sqlc.arg(created_at)
)
ON CONFLICT (task_rowid, referenced_task_rowid)
DO UPDATE SET kind = excluded.kind;

-- name: RemoveTaskReference :execrows
DELETE FROM task_references
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND referenced_task_rowid = (
    SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(referenced_task_id)
  );

-- name: ListTaskReferenceIDs :many

SELECT ref.id FROM task_references r
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
WHERE r.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1)
ORDER BY ref.id;

-- name: ListTaskReferences :many

SELECT ref.* FROM tasks ref
JOIN task_references r ON r.referenced_task_rowid = ref.rowid
WHERE r.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1)
ORDER BY ref.created_at DESC;

-- name: ListTaskReferencedBy :many

SELECT t.* FROM tasks t
JOIN task_references r ON r.task_rowid = t.rowid
WHERE r.referenced_task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1)
ORDER BY t.created_at DESC;

-- name: CountIncompleteBlockingReferences :one

SELECT count(*) FROM task_references r
JOIN tasks t ON t.rowid = r.referenced_task_rowid
WHERE r.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1)
  AND r.kind = 'blocking'
  AND t.state <> 'completed';

-- name: ListAllBlockingTaskReferences :many

SELECT t.id AS task_id, ref.id AS referenced_task_id
FROM task_references r
JOIN tasks t ON t.rowid = r.task_rowid
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
WHERE r.kind = 'blocking'
ORDER BY t.id, ref.id;

-- name: DeleteTaskReferenceEdges :exec


DELETE FROM task_references
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
   OR referenced_task_rowid = (
     SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)
   );

-- name: DeleteProjectAgentReferenceEdges :exec

DELETE FROM task_references
WHERE task_rowid IN (
    SELECT t.rowid FROM tasks t WHERE t.project_id = sqlc.arg(project_id)
      AND t.assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
  )
   OR referenced_task_rowid IN (
    SELECT t.rowid FROM tasks t WHERE t.project_id = sqlc.arg(project_id)
      AND t.assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
  );

-- name: ListTaskReferenceRefs :many


SELECT
  t.id AS task_id,
  ref.id AS reference_id,
  ref.name AS reference_name,
  ref.prompt AS reference_prompt,
  ref.state AS reference_state,
  r.kind AS kind
FROM task_references r
JOIN tasks t ON t.rowid = r.task_rowid
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
ORDER BY t.id, ref.created_at DESC;

-- name: ListTaskReferencedByRefs :many


SELECT
  ref.id AS task_id,
  t.id AS reference_id,
  t.name AS reference_name,
  t.prompt AS reference_prompt,
  t.state AS reference_state,
  r.kind AS kind
FROM task_references r
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
JOIN tasks t ON t.rowid = r.task_rowid
ORDER BY ref.id, t.created_at DESC;
