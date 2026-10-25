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

-- name: CreateTask :one
INSERT INTO tasks (
  id, seq, prompt, name, project_id, assignee_actor_id, assignee_pinned, priority, size, state,
  creator_actor_id,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, model_provider, model_id, thinking_level, interactive, workspace, auto_title,
  queued_at, started_at, settled_at, created_at, updated_at
) VALUES (
  sqlc.arg(id),
  sqlc.arg(seq),
  sqlc.arg(prompt),
  sqlc.arg(name),
  NULLIF(CAST(sqlc.arg(project_id) AS TEXT), ''),
  sqlc.narg(assignee_actor_id),
  sqlc.arg(assignee_pinned),
  sqlc.arg(priority),
  COALESCE(NULLIF(CAST(sqlc.arg(size) AS TEXT), ''), 'medium'),
  sqlc.arg(state),
  CAST(sqlc.narg(creator_actor_id) AS BIGINT),
  sqlc.arg(session_id),
  sqlc.arg(org_prompt_id),
  sqlc.arg(agent_prompt_id),
  sqlc.arg(project_prompt_id),
  sqlc.arg(agent_project_prompt_id),
  sqlc.arg(error),
  sqlc.arg(model_provider),
  sqlc.arg(model_id),
  sqlc.arg(thinking_level),
  sqlc.arg(interactive),
  sqlc.arg(workspace),
  sqlc.arg(auto_title),
  sqlc.arg(queued_at),
  sqlc.arg(started_at),
  sqlc.arg(settled_at),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1;

-- name: GetTaskByProjectSeq :one
SELECT * FROM tasks WHERE project_id = $1 AND seq = $2;






-- name: ListTasks :many
SELECT t.* FROM tasks t
WHERE (
    CAST(sqlc.narg(project_id) AS TEXT) IS NULL
    OR t.project_id = sqlc.narg(project_id)
  )
  AND (
    CAST(sqlc.narg(agent_id) AS BIGINT) IS NULL
    OR t.assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.narg(agent_id))
  )
  AND (
    CAST(sqlc.narg(state) AS TEXT) IS NULL
    OR t.state = sqlc.narg(state)
  )
  AND (
    (CAST(sqlc.arg(archived) AS BIGINT) = 1 AND t.archived_at IS NOT NULL)
    OR (CAST(sqlc.arg(archived) AS BIGINT) = 0 AND t.archived_at IS NULL)
  )
  AND (
    CAST(sqlc.narg(tag_id) AS TEXT) IS NULL
    OR EXISTS (
      SELECT 1 FROM task_tags tt WHERE tt.task_rowid = t.rowid AND tt.tag_id = sqlc.narg(tag_id)
    )
  )
  AND (
    CAST(sqlc.narg(search) AS TEXT) IS NULL
    OR lower(t.name) LIKE '%' || lower(sqlc.narg(search)) || '%'
    OR lower(t.prompt) LIKE '%' || lower(sqlc.narg(search)) || '%'
    
    
    OR lower(t.id) LIKE '%' || lower(sqlc.narg(search)) || '%'
    OR (
      
      
      sqlc.narg(search) ~ '^[0-9]*$'
      AND CAST(t.seq AS TEXT) LIKE '%' || sqlc.narg(search) || '%'
    )
  )
  AND (
    CAST(sqlc.narg(size) AS TEXT) IS NULL
    OR t.size = sqlc.narg(size)
  )
  AND (
    CAST(sqlc.narg(project_id) AS TEXT) IS NOT NULL
    OR t.project_id IS NULL
    OR EXISTS (
      SELECT 1 FROM projects p WHERE p.id = t.project_id AND p.archived_at IS NULL
    )
  )
ORDER BY t.rowid DESC;

-- name: ListQueuedTasks :many



SELECT * FROM tasks
WHERE state = 'queued' AND archived_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM task_references r
    JOIN tasks dep ON dep.rowid = r.referenced_task_rowid
    WHERE r.task_rowid = tasks.rowid
      AND r.kind = 'blocking'
      AND dep.state <> 'completed'
  )
ORDER BY priority DESC, created_at ASC;

-- name: ListRunningTasks :many


SELECT * FROM tasks
WHERE state IN ('running', 'input') AND archived_at IS NULL
ORDER BY created_at ASC;

-- name: UpdateDraftTask :one


UPDATE tasks SET
  id = sqlc.arg(new_id),
  seq = sqlc.arg(seq),
  prompt = sqlc.arg(prompt),
  name = sqlc.arg(name),
  project_id = NULLIF(CAST(sqlc.arg(project_id) AS TEXT), ''),
  assignee_actor_id = sqlc.narg(assignee_actor_id),
  assignee_pinned = sqlc.arg(assignee_pinned),
  priority = sqlc.arg(priority),
  size = COALESCE(NULLIF(CAST(sqlc.arg(size) AS TEXT), ''), 'medium'),
  model_provider = sqlc.arg(model_provider),
  model_id = sqlc.arg(model_id),
  thinking_level = sqlc.arg(thinking_level),
  interactive = sqlc.arg(interactive),
  workspace = sqlc.arg(workspace),
  auto_title = sqlc.arg(auto_title),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND state = 'draft'
RETURNING *;

-- name: PromoteTask :execrows

UPDATE tasks SET state = 'backlog', updated_at = $1, error = ''
WHERE id = $2 AND state = 'draft';

-- name: DemoteTask :execrows

UPDATE tasks SET state = 'draft', updated_at = $1, error = ''
WHERE id = $2 AND state = 'backlog';

-- name: QueueTask :execrows


UPDATE tasks SET state = 'queued', queued_at = COALESCE(queued_at, $1), updated_at = $2, error = ''
WHERE id = $3 AND state = 'backlog';

-- name: DequeueTask :execrows

UPDATE tasks SET state = 'backlog', updated_at = $1, error = ''
WHERE id = $2 AND state = 'queued';





-- name: ClaimTask :execrows
UPDATE tasks SET
  state = 'running',
  assignee_actor_id = sqlc.arg(assignee_actor_id),
  assignee_pinned = sqlc.arg(assignee_pinned),
  started_at = COALESCE(started_at, sqlc.arg(started_at)),
  updated_at = sqlc.arg(updated_at),
  error = '',
  org_prompt_id = COALESCE(sqlc.narg(org_prompt_id), org_prompt_id),
  agent_prompt_id = COALESCE(sqlc.narg(agent_prompt_id), agent_prompt_id),
  project_prompt_id = COALESCE(sqlc.narg(project_prompt_id), project_prompt_id),
  agent_project_prompt_id = COALESCE(
    sqlc.narg(agent_project_prompt_id), agent_project_prompt_id
  )
WHERE id = sqlc.arg(id) AND state = 'queued';

-- name: SettleTask :execrows
UPDATE tasks SET state = 'settled', settled_at = $1, updated_at = $2, error = ''
WHERE id = $3 AND state IN ('running', 'input');

-- name: MarkTaskInput :execrows

UPDATE tasks SET state = 'input', updated_at = $1
WHERE id = $2 AND state = 'running';

-- name: ClearTaskInput :execrows

UPDATE tasks SET state = 'running', updated_at = $1
WHERE id = $2 AND state = 'input';

-- name: ResumeTask :execrows


UPDATE tasks SET state = 'running', settled_at = $1, updated_at = $2
WHERE id = $3 AND state IN ('settled', 'failed');

-- name: FailTask :execrows
UPDATE tasks SET state = 'failed', error = $1, settled_at = $2, updated_at = $3
WHERE id = $4 AND state IN ('running', 'input');

-- name: CompleteTask :execrows
-- name: CompleteTask :execrows


UPDATE tasks SET state = 'completed', settled_at = $1, updated_at = $2, error = '',
  pending_prompt = '', pending_prompt_actor_id = NULL
WHERE id = $3 AND state IN ('draft', 'backlog', 'queued', 'settled', 'failed', 'cancelled');

-- name: CancelTask :execrows


UPDATE tasks SET state = 'cancelled', settled_at = $1, updated_at = $2,
  pending_prompt = '', pending_prompt_actor_id = NULL
WHERE id = $3 AND state IN ('draft', 'backlog', 'queued', 'settled', 'failed', 'completed');

-- name: ArchiveTask :execrows


UPDATE tasks SET archived_at = $1, updated_at = $2
WHERE id = $3 AND archived_at IS NULL AND state IN ('completed', 'cancelled');

-- name: UnarchiveTask :execrows
UPDATE tasks SET archived_at = NULL, updated_at = $1
WHERE id = $2 AND archived_at IS NOT NULL;

-- name: RequeueTaskWithPrompt :execrows




UPDATE tasks SET
  state = 'queued', error = '', pending_prompt = sqlc.arg(pending_prompt),
  pending_prompt_actor_id = sqlc.narg(pending_prompt_actor_id),
  queued_at = COALESCE(queued_at, sqlc.arg(queued_at)), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND state IN ('completed', 'cancelled', 'settled', 'failed');

-- name: ClearTaskPendingPrompt :execrows
UPDATE tasks SET pending_prompt = '', pending_prompt_actor_id = NULL, updated_at = $1
WHERE id = $2;

-- name: SetTaskPrompt :execrows


UPDATE tasks SET prompt = $1, updated_at = $2
WHERE id = $3 AND state IN ('draft', 'backlog');

-- name: SetTaskPendingPrompt :execrows


UPDATE tasks SET pending_prompt = sqlc.arg(pending_prompt),
  pending_prompt_actor_id = sqlc.narg(pending_prompt_actor_id),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND state = 'draft';

-- name: RequeueTask :execrows
UPDATE tasks SET
  state = 'queued', error = $1, queued_at = COALESCE(queued_at, $2), updated_at = $3
WHERE id = $4 AND state = 'running';

-- name: SetTaskError :execrows
UPDATE tasks SET error = $1, updated_at = $2 WHERE id = $3;

-- name: ReopenTask :execrows
UPDATE tasks SET state = $1, settled_at = NULL, error = '', pending_prompt = '',
  pending_prompt_actor_id = NULL, updated_at = $2
WHERE id = $3 AND state IN ('settled', 'failed', 'completed', 'cancelled');

-- name: SetTaskSession :execrows





UPDATE tasks SET
  session_id = $1, session_agent_id = $2, session_deleted_at = NULL, updated_at = $3
WHERE id = $4;

-- name: RenameTask :one
UPDATE tasks SET name = $1, updated_at = $2 WHERE id = $3 RETURNING *;

-- name: SetTaskPriority :one
UPDATE tasks SET priority = $1, updated_at = $2 WHERE id = $3 RETURNING *;

-- name: SetTaskSize :one
UPDATE tasks SET size = $1, updated_at = $2 WHERE id = $3 RETURNING *;

-- name: MarkTaskSessionDeleted :one


UPDATE tasks SET session_deleted_at = $1, updated_at = $2
WHERE id = $3 AND session_deleted_at IS NULL
RETURNING *;
-- name: SetTaskAssignee :one



UPDATE tasks
SET
  assignee_actor_id = sqlc.narg(assignee_actor_id),
  assignee_pinned = sqlc.arg(assignee_pinned),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1;

-- name: RenameProjectTasks :exec



UPDATE tasks
SET id = sqlc.arg(prefix) || '-' || seq
WHERE project_id = sqlc.arg(project_id);

-- name: ListTasksByProject :many
SELECT * FROM tasks WHERE project_id = $1;

-- name: ListTasksByProjectAgent :many
SELECT t.* FROM tasks t
JOIN agents g ON g.actor_id = t.assignee_actor_id
WHERE t.project_id = sqlc.arg(project_id) AND g.id = sqlc.arg(agent_id);

-- name: CountTasksByProject :one
SELECT count(*) FROM tasks WHERE project_id = $1;

-- name: DeleteTasksByProjectAgent :exec
DELETE FROM tasks WHERE rowid IN (
  SELECT t.rowid FROM tasks t
  JOIN agents g ON g.actor_id = t.assignee_actor_id
  WHERE t.project_id = sqlc.arg(project_id) AND g.id = sqlc.arg(agent_id)
);

-- name: ListRunningTasksByAgent :many


SELECT t.* FROM tasks t
JOIN agents g ON g.actor_id = t.assignee_actor_id
WHERE g.id = sqlc.arg(agent_id)
  AND t.archived_at IS NULL
  AND t.state IN ('running', 'input', 'settled', 'failed');


-- name: ListRunningTaskTagsByAgent :many
SELECT t.id AS task_id, tt.tag_id, tags.name
FROM task_tags tt
JOIN tasks t ON t.rowid = tt.task_rowid
JOIN agents g ON g.actor_id = t.assignee_actor_id
JOIN tags ON tags.id = tt.tag_id
WHERE g.id = sqlc.arg(agent_id)
  AND t.archived_at IS NULL
  AND t.state IN ('running', 'input', 'settled', 'failed');

-- name: FailTaskBySession :execrows
UPDATE tasks SET
  state = 'failed',
  error = sqlc.arg(error),
  settled_at = sqlc.arg(settled_at),
  updated_at = sqlc.arg(updated_at)
WHERE rowid IN (
  SELECT t.rowid FROM tasks t
  JOIN agents g ON g.actor_id = t.assignee_actor_id
  WHERE g.id = sqlc.arg(agent_id)
    AND t.session_id = sqlc.arg(session_id)
    AND t.state IN ('running', 'input')
);

-- name: TaskIDBySession :one
SELECT t.id FROM tasks t
JOIN agents g ON g.actor_id = t.assignee_actor_id
WHERE g.id = sqlc.arg(agent_id)
  AND t.session_id = sqlc.arg(session_id)
  AND t.session_id <> ''
LIMIT 1;

-- name: SetTaskNamePrompt :exec
UPDATE tasks SET
  name = CASE WHEN name = '' THEN sqlc.arg(name) ELSE name END,
  prompt = CASE WHEN prompt = '' THEN sqlc.arg(prompt) ELSE prompt END
WHERE id = sqlc.arg(id) AND (name = '' OR prompt = '');
