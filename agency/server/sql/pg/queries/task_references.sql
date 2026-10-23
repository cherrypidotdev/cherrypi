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

-- Task references: an edge from one task to another with a `kind`
-- (`blocking`, `informational`, or `supersedes`). A blocking edge gates
-- queuing (the task waits for `completed`); an informational edge is a plain
-- relation; a supersedes edge cancels a draft/backlog/queued/failed task.
--
-- Edges key on the two task rowids, never on the mutable `<prefix>-<seq>` id,
-- so renaming a task (project move, title change, or project prefix change)
-- leaves the graph untouched. The public queries still take and return the
-- string id; the subqueries resolve it to the rowid through `tasks.id`.

-- name: UpsertTaskReference :exec
-- Add an edge, or flip an existing edge's kind. An existing edge is otherwise
-- left alone (its created_at is preserved).
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
-- Every reference id, regardless of blocking, in id order.
SELECT ref.id FROM task_references r
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
WHERE r.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1)
ORDER BY ref.id;

-- name: ListTaskReferences :many
-- The referenced tasks (outgoing edges), newest id order.
SELECT ref.* FROM tasks ref
JOIN task_references r ON r.referenced_task_rowid = ref.rowid
WHERE r.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1)
ORDER BY ref.created_at DESC;

-- name: ListTaskReferencedBy :many
-- The tasks that reference this one (incoming edges), newest id order.
SELECT t.* FROM tasks t
JOIN task_references r ON r.task_rowid = t.rowid
WHERE r.referenced_task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1)
ORDER BY t.created_at DESC;

-- name: CountIncompleteBlockingReferences :one
-- How many of a task's blocking references have not reached `completed`.
SELECT count(*) FROM task_references r
JOIN tasks t ON t.rowid = r.referenced_task_rowid
WHERE r.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = $1)
  AND r.kind = 'blocking'
  AND t.state <> 'completed';

-- name: ListAllBlockingTaskReferences :many
-- Every blocking edge, used for in-process cycle detection (task -> referenced).
SELECT t.id AS task_id, ref.id AS referenced_task_id
FROM task_references r
JOIN tasks t ON t.rowid = r.task_rowid
JOIN tasks ref ON ref.rowid = r.referenced_task_rowid
WHERE r.kind = 'blocking'
ORDER BY t.id, ref.id;

-- name: DeleteTaskReferenceEdges :exec
-- Remove every edge that touches a task (as referencing or referenced). Foreign
-- key enforcement is off in this SQLite setup, so the cascade does not run.
DELETE FROM task_references
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
   OR referenced_task_rowid = (
     SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)
   );

-- name: DeleteProjectAgentReferenceEdges :exec
-- Remove every edge that touches a task deleted with a project membership.
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
-- Lightweight outgoing references (id, name, state, kind) for every edge,
-- so a task list can render the references column without loading whole tasks.
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
-- Lightweight incoming references for every edge, keyed by the referenced task,
-- so a task detail can render "referenced by" without loading whole tasks.
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
