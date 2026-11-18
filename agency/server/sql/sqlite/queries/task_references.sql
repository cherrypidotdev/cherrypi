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


INSERT INTO task_references (
  task_rowid, referenced_task_rowid, kind, wake_on_resolve_when_many, created_at
)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(referenced_task_id)),
  sqlc.arg(kind),
  sqlc.arg(wake_on_resolve_when_many),
  sqlc.arg(created_at)
)
ON CONFLICT (task_rowid, referenced_task_rowid)
DO UPDATE SET
  kind = excluded.kind,
  wake_on_resolve_when_many = excluded.wake_on_resolve_when_many;

-- name: RemoveTaskReference :execrows
DELETE FROM task_references
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND referenced_task_rowid = (
    SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(referenced_task_id)
  );

-- name: ListTaskReferenceIDs :many

SELECT ref.id FROM task_references r
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
WHERE r.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = ?)
ORDER BY ref.id;

-- name: ListTaskReferences :many

SELECT ref.* FROM tasks ref
JOIN task_references r ON r.referenced_task_rowid = ref.rowid
WHERE r.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = ?)
ORDER BY ref.created_at DESC;

-- name: ListTaskReferencedBy :many

SELECT t.* FROM tasks t
JOIN task_references r ON r.task_rowid = t.rowid
WHERE r.referenced_task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = ?)
ORDER BY t.created_at DESC;

-- name: ListBlockedEdgesByBlocker :many


SELECT r.task_rowid AS task_rowid,
       t.id AS blocked_task_id,
       r.wake_on_resolve_when_many AS wake_on_resolve_when_many
FROM task_references r
JOIN tasks t ON t.rowid = r.task_rowid
WHERE r.referenced_task_rowid = (
  SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(blocker_id)
)
  AND r.kind = 'blocking'
ORDER BY r.created_at;

-- name: CountIncompleteBlockingReferences :one


SELECT count(*) FROM task_references r
JOIN tasks t ON t.rowid = r.referenced_task_rowid
WHERE r.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = ?)
  AND r.kind = 'blocking'
  AND t.state NOT IN ('completed', 'cancelled');

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
  ref.priority AS reference_priority,
  ref.size AS reference_size,
  r.kind AS kind,
  r.wake_on_resolve_when_many AS wake_on_resolve_when_many
FROM task_references r
JOIN tasks t ON t.rowid = r.task_rowid
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
ORDER BY t.id, ref.created_at DESC, ref.id;

-- name: ListTaskReferenceRefsByTask :many




SELECT
  t.id AS task_id,
  ref.id AS reference_id,
  ref.name AS reference_name,
  ref.prompt AS reference_prompt,
  ref.state AS reference_state,
  ref.priority AS reference_priority,
  ref.size AS reference_size,
  r.kind AS kind,
  r.wake_on_resolve_when_many AS wake_on_resolve_when_many
FROM task_references r
JOIN tasks t ON t.rowid = r.task_rowid
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
WHERE t.id = sqlc.arg(task_id)
ORDER BY ref.created_at DESC, ref.id;

-- name: ListTaskReferencedByRefsByTask :many

SELECT
  ref.id AS task_id,
  t.id AS reference_id,
  t.name AS reference_name,
  t.prompt AS reference_prompt,
  t.state AS reference_state,
  t.priority AS reference_priority,
  t.size AS reference_size,
  r.kind AS kind,
  r.wake_on_resolve_when_many AS wake_on_resolve_when_many
FROM task_references r
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
JOIN tasks t ON t.rowid = r.task_rowid
WHERE ref.id = sqlc.arg(task_id)
ORDER BY t.created_at DESC, t.id;
