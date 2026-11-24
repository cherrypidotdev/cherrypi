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
  id, seq, prompt, name, project_id, origin_task_id,
  assignee_actor_id, assignee_pinned, priority, size, state,
  creator_actor_id,
  session_id, org_prompt_id, agent_prompt_id, project_prompt_id, agent_project_prompt_id,
  error, model_provider, model_id, thinking_level, automated, reaction, workspace,
  auto_title, queued_at, started_at, settled_at, created_at, updated_at
) VALUES (
  sqlc.arg(id),
  sqlc.arg(seq),
  sqlc.arg(prompt),
  sqlc.arg(name),
  NULLIF(CAST(sqlc.arg(project_id) AS TEXT), ''),
  NULLIF(CAST(sqlc.arg(origin_task_id) AS TEXT), ''),
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
  sqlc.arg(automated),
  sqlc.arg(reaction),
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
WITH reference_scope AS (
  SELECT CAST(sqlc.narg(reference_project_id) AS TEXT) AS id
  WHERE CAST(sqlc.narg(reference_project_id) AS TEXT) IS NOT NULL
  UNION
  SELECT pa.project_id
  FROM project_ancestors pa
  WHERE pa.ancestor_id = sqlc.narg(reference_project_id)
  UNION
  SELECT pa.ancestor_id
  FROM project_ancestors pa
  WHERE pa.project_id = sqlc.narg(reference_project_id)
  UNION
  SELECT p.id
  FROM projects p
  JOIN projects origin ON origin.id = sqlc.narg(reference_project_id)
  WHERE p.id <> origin.id
    AND (
      (p.parent_project_id IS NOT NULL AND p.parent_project_id = origin.parent_project_id)
      OR (p.parent_project_id IS NULL AND origin.parent_project_id IS NULL)
    )
)
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
    OR (
      CAST(sqlc.arg(archived) AS BIGINT) = 0
      AND (
        t.archived_at IS NULL
        OR CAST(sqlc.arg(include_archived) AS BIGINT) = 1
      )
    )
  )
  AND (
    CAST(sqlc.narg(automated) AS BIGINT) IS NULL
    OR t.automated = sqlc.narg(automated)
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
    CAST(sqlc.narg(priority) AS BIGINT) IS NULL
    OR t.priority = sqlc.narg(priority)
  )
  AND (
    CAST(sqlc.narg(project_id) AS TEXT) IS NOT NULL
    OR t.project_id IS NULL
    OR EXISTS (
      SELECT 1 FROM projects p WHERE p.id = t.project_id AND p.archived_at IS NULL
    )
  )
  AND (
    CAST(sqlc.narg(reference_project_id) AS TEXT) IS NULL
    OR t.project_id IN (SELECT id FROM reference_scope)
  )
ORDER BY t.rowid DESC
LIMIT CAST(sqlc.narg(page_limit) AS BIGINT)
OFFSET CAST(sqlc.arg(page_offset) AS BIGINT);
-- name: ListTasksRanked :many








WITH reference_scope AS (
  SELECT CAST(sqlc.narg(reference_project_id) AS TEXT) AS id
  WHERE CAST(sqlc.narg(reference_project_id) AS TEXT) IS NOT NULL
  UNION
  SELECT pa.project_id
  FROM project_ancestors pa
  WHERE pa.ancestor_id = sqlc.narg(reference_project_id)
  UNION
  SELECT pa.ancestor_id
  FROM project_ancestors pa
  WHERE pa.project_id = sqlc.narg(reference_project_id)
  UNION
  SELECT p.id
  FROM projects p
  JOIN projects origin ON origin.id = sqlc.narg(reference_project_id)
  WHERE p.id <> origin.id
    AND (
      (p.parent_project_id IS NOT NULL AND p.parent_project_id = origin.parent_project_id)
      OR (p.parent_project_id IS NULL AND origin.parent_project_id IS NULL)
    )
),
matched AS (
  SELECT
    t.rowid AS rid,
    CASE
      WHEN CAST(sqlc.narg(search) AS TEXT) IS NULL THEN 0
      WHEN lower(t.id) = lower(sqlc.narg(search)) THEN 0
      WHEN sqlc.narg(search) ~ '^[0-9]*$'
        AND CAST(t.seq AS TEXT) = sqlc.narg(search) THEN 1
      WHEN sqlc.narg(search) ~ '^[0-9]*$'
        AND CAST(t.seq AS TEXT) LIKE '%' || sqlc.narg(search) || '%' THEN 2
      ELSE 3
    END AS match_rank,
    
    
    CASE
      WHEN sqlc.narg(search) ~ '^[0-9]*$'
        AND CAST(sqlc.narg(reference_project_id) AS TEXT) IS NOT NULL
        AND t.project_id = sqlc.narg(reference_project_id) THEN 0
      ELSE 1
    END AS project_rank
  FROM tasks t
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
      OR (
        CAST(sqlc.arg(archived) AS BIGINT) = 0
        AND (
          t.archived_at IS NULL
          OR CAST(sqlc.arg(include_archived) AS BIGINT) = 1
        )
      )
    )
    AND (
      CAST(sqlc.narg(automated) AS BIGINT) IS NULL
      OR t.automated = sqlc.narg(automated)
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
      CAST(sqlc.narg(priority) AS BIGINT) IS NULL
      OR t.priority = sqlc.narg(priority)
    )
    AND (
      CAST(sqlc.narg(project_id) AS TEXT) IS NOT NULL
      OR t.project_id IS NULL
      OR EXISTS (
        SELECT 1 FROM projects p WHERE p.id = t.project_id AND p.archived_at IS NULL
      )
    )
    AND (
      CAST(sqlc.narg(reference_project_id) AS TEXT) IS NULL
      OR t.project_id IN (SELECT id FROM reference_scope)
    )
)
SELECT t.* FROM tasks t
JOIN matched m ON m.rid = t.rowid
ORDER BY m.match_rank, m.project_rank, t.rowid DESC
LIMIT CAST(sqlc.narg(page_limit) AS BIGINT)
OFFSET CAST(sqlc.arg(page_offset) AS BIGINT);

-- name: ListQueuedTasks :many








SELECT * FROM tasks
WHERE state = 'queued' AND archived_at IS NULL
ORDER BY priority DESC, created_at ASC;

-- name: ListRunningTasks :many


SELECT * FROM tasks
WHERE state IN ('running', 'input') AND archived_at IS NULL
ORDER BY created_at ASC;

-- name: ListSessionTasks :many




SELECT * FROM tasks
WHERE state IN ('running', 'input', 'settled', 'failed')
  AND session_id <> ''
  AND session_deleted_at IS NULL
  AND archived_at IS NULL
ORDER BY created_at ASC;

-- name: UpdateBacklogTask :one



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
  workspace = sqlc.arg(workspace),
  auto_title = sqlc.arg(auto_title),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND state = 'backlog' AND started_at IS NULL
RETURNING *;

-- name: QueueTask :execrows


UPDATE tasks SET state = 'queued', queued_at = COALESCE(queued_at, $1), updated_at = $2, error = ''
WHERE id = $3 AND state = 'backlog';

-- name: WaitTask :execrows




UPDATE tasks SET state = 'waiting', queued_at = COALESCE(queued_at, sqlc.arg(queued_at)),
  updated_at = sqlc.arg(updated_at), error = ''
WHERE id = sqlc.arg(id)
  AND state IN ('backlog', 'queued', 'running', 'input', 'settled', 'failed');

-- name: UnwaitTask :execrows

UPDATE tasks SET state = 'queued', queued_at = COALESCE(queued_at, sqlc.arg(queued_at)),
  updated_at = sqlc.arg(updated_at), error = ''
WHERE id = sqlc.arg(id) AND state = 'waiting';

-- name: WakeWaitingTaskWithPrompt :execrows








UPDATE tasks SET state = 'queued', pending_prompt = sqlc.arg(pending_prompt),
  pending_prompt_actor_id = sqlc.narg(pending_prompt_actor_id),
  queued_at = COALESCE(queued_at, sqlc.arg(queued_at)),
  updated_at = sqlc.arg(updated_at), error = ''
WHERE id = sqlc.arg(id) AND state = 'waiting'
  AND NOT EXISTS (SELECT 1 FROM task_holds WHERE task_id = sqlc.arg(id));

-- name: DequeueTask :execrows

UPDATE tasks SET state = 'backlog', updated_at = $1, error = ''
WHERE id = $2 AND state IN ('queued', 'waiting');





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


UPDATE tasks SET
  state = CASE
    WHEN EXISTS (SELECT 1 FROM task_holds WHERE task_id = sqlc.arg(id))
      THEN 'waiting'
    ELSE 'settled'
  END,
  settled_at = sqlc.arg(settled_at),
  updated_at = sqlc.arg(updated_at),
  error = ''
WHERE id = sqlc.arg(id) AND state IN ('running', 'input');

-- name: ReleaseSettleTask :execrows


UPDATE tasks SET state = 'settled', settled_at = COALESCE(settled_at, $1),
  updated_at = $2, error = ''
WHERE id = $3 AND state = 'waiting';

-- name: ReleaseQueueTask :execrows


UPDATE tasks SET state = 'queued', pending_prompt = $1,
  pending_prompt_actor_id = $2, queued_at = COALESCE(queued_at, $3),
  updated_at = $4, error = ''
WHERE id = $5 AND state = 'waiting';

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
WHERE id = $3
  AND state IN ('backlog', 'waiting', 'queued', 'settled', 'failed', 'cancelled');

-- name: CancelTask :execrows


UPDATE tasks SET state = 'cancelled', settled_at = $1, updated_at = $2,
  pending_prompt = '', pending_prompt_actor_id = NULL
WHERE id = $3
  AND state IN ('backlog', 'waiting', 'queued', 'settled', 'failed', 'completed');

-- name: ArchiveTask :execrows


UPDATE tasks SET archived_at = $1, updated_at = $2
WHERE id = $3 AND archived_at IS NULL AND state IN ('completed', 'cancelled');

-- name: UnarchiveTask :execrows
UPDATE tasks SET archived_at = NULL, updated_at = $1
WHERE id = $2 AND archived_at IS NOT NULL;

-- name: RequeueLostSessionTask :execrows





UPDATE tasks SET state = 'queued', error = '',
  queued_at = COALESCE(queued_at, sqlc.arg(queued_at)),
  updated_at = sqlc.arg(updated_at), archived_at = NULL
WHERE id = sqlc.arg(id) AND state IN ('running', 'input');

-- name: RequeueTaskWithPrompt :execrows







UPDATE tasks SET
  state = 'queued', error = '', pending_prompt = sqlc.arg(pending_prompt),
  pending_prompt_actor_id = CASE
    WHEN pending_prompt = '' THEN sqlc.narg(pending_prompt_actor_id)
    ELSE COALESCE(pending_prompt_actor_id, sqlc.narg(pending_prompt_actor_id))
  END,
  queued_at = COALESCE(queued_at, sqlc.arg(queued_at)), updated_at = sqlc.arg(updated_at),
  
  
  archived_at = NULL
WHERE id = sqlc.arg(id)
  AND state IN ('completed', 'cancelled', 'settled', 'failed', 'backlog');

-- name: ClearTaskPendingPrompt :execrows
UPDATE tasks SET pending_prompt = '', pending_prompt_actor_id = NULL, updated_at = $1
WHERE id = $2;

-- name: SetTaskPrompt :execrows


UPDATE tasks SET prompt = $1, updated_at = $2
WHERE id = $3 AND state = 'backlog' AND started_at IS NULL;

-- name: SetTaskModel :execrows


UPDATE tasks SET model_provider = $1, model_id = $2, thinking_level = $3, updated_at = $4
WHERE id = $5 AND state = 'backlog' AND started_at IS NULL;

-- name: SetTaskPendingPrompt :execrows


UPDATE tasks SET pending_prompt = sqlc.arg(pending_prompt),
  pending_prompt_actor_id = sqlc.narg(pending_prompt_actor_id),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND state = 'backlog' AND started_at IS NULL;

-- name: RequeueTask :execrows
UPDATE tasks SET
  state = 'queued', error = $1, queued_at = COALESCE(queued_at, $2), updated_at = $3
WHERE id = $4 AND state = 'running';

-- name: SetTaskError :execrows
UPDATE tasks SET error = $1, updated_at = $2 WHERE id = $3;

-- name: ReopenTask :execrows
UPDATE tasks SET state = $1, settled_at = NULL, error = '', pending_prompt = '',
  pending_prompt_actor_id = NULL, updated_at = $2
WHERE id = $3
  AND state IN ('settled', 'failed', 'completed', 'cancelled', 'waiting');

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
  AND t.state = 'running';


-- name: ListRunningTaskTagsByAgent :many
SELECT t.id AS task_id, tt.tag_id, tags.name
FROM task_tags tt
JOIN tasks t ON t.rowid = tt.task_rowid
JOIN agents g ON g.actor_id = t.assignee_actor_id
JOIN tags ON tags.id = tt.tag_id
WHERE g.id = sqlc.arg(agent_id)
  AND t.archived_at IS NULL
  AND t.state = 'running';

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
    AND (
      t.state IN ('running', 'input')
      OR (t.state = 'settled' AND t.settled_at >= sqlc.arg(not_before))
    )
);

-- name: TaskIDBySession :one
SELECT t.id FROM tasks t
JOIN agents g ON g.actor_id = t.assignee_actor_id
WHERE g.id = sqlc.arg(agent_id)
  AND t.session_id = sqlc.arg(session_id)
  AND t.session_id <> ''
LIMIT 1;

-- name: SessionAgentID :one




SELECT COALESCE(t.session_agent_id, g.id) AS agent_id
FROM tasks t
LEFT JOIN agents g ON g.actor_id = t.assignee_actor_id
WHERE t.session_id = sqlc.arg(session_id)
  AND t.session_id <> ''
  AND COALESCE(t.session_agent_id, g.id) IS NOT NULL
LIMIT 1;

-- name: SetTaskNamePrompt :exec
UPDATE tasks SET
  name = CASE WHEN name = '' THEN sqlc.arg(name) ELSE name END,
  prompt = CASE WHEN prompt = '' THEN sqlc.arg(prompt) ELSE prompt END
WHERE id = sqlc.arg(id) AND (name = '' OR prompt = '');

-- name: FailRestingTask :execrows


UPDATE tasks SET state = 'failed', error = sqlc.arg(error),
  settled_at = sqlc.arg(settled_at), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND state IN ('backlog', 'queued', 'waiting', 'settled', 'failed');

-- name: SetTaskPendingSteering :execrows


UPDATE tasks SET pending_steering = sqlc.arg(pending_steering),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: SetTaskPendingPromptAny :execrows






UPDATE tasks SET pending_prompt = sqlc.arg(pending_prompt),
  pending_prompt_actor_id = CASE
    WHEN pending_prompt = '' THEN sqlc.narg(pending_prompt_actor_id)
    ELSE COALESCE(pending_prompt_actor_id, sqlc.narg(pending_prompt_actor_id))
  END,
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: ListRunningTasksByProject :many


SELECT t.* FROM tasks t
WHERE t.project_id = sqlc.arg(project_id)
  AND t.archived_at IS NULL
  AND t.state = 'running';

-- name: ListRunningTaskTagsByProject :many
SELECT t.id AS task_id, tt.tag_id, tags.name
FROM task_tags tt
JOIN tasks t ON t.rowid = tt.task_rowid
JOIN tags ON tags.id = tt.tag_id
WHERE t.project_id = sqlc.arg(project_id)
  AND t.archived_at IS NULL
  AND t.state = 'running';

-- name: ListInheritedAutoArchivableTasks :many




SELECT t.* FROM tasks t
LEFT JOIN projects p ON p.id = t.project_id
WHERE t.state IN ('completed', 'cancelled')
  AND t.archived_at IS NULL
  AND t.settled_at IS NOT NULL
  AND t.settled_at <= sqlc.arg(cutoff_ms)
  AND (p.id IS NULL OR p.archive_after_days IS NULL)
ORDER BY t.settled_at ASC;

-- name: ListOverriddenAutoArchivableTasks :many



SELECT t.* FROM tasks t
JOIN projects p ON p.id = t.project_id
WHERE t.state IN ('completed', 'cancelled')
  AND t.archived_at IS NULL
  AND t.settled_at IS NOT NULL
  AND p.archive_after_days > 0
  AND t.settled_at <= sqlc.arg(now_ms) - CAST(p.archive_after_days AS BIGINT) * 86400000
ORDER BY t.settled_at ASC;

-- name: RecordTaskSession :exec


INSERT INTO task_sessions (task_id, agent_id, session_id, created_at)
VALUES (sqlc.arg(task_id), sqlc.arg(agent_id), sqlc.arg(session_id), sqlc.arg(created_at))
ON CONFLICT (task_id, agent_id, session_id) DO NOTHING;

-- name: ListLiveTaskSessions :many

SELECT task_id, agent_id, session_id, created_at, deleted_at
FROM task_sessions
WHERE task_id = sqlc.arg(task_id) AND deleted_at IS NULL
ORDER BY created_at ASC;

-- name: MarkTaskSessionCleaned :exec

UPDATE task_sessions SET deleted_at = sqlc.arg(deleted_at)
WHERE task_id = sqlc.arg(task_id)
  AND agent_id = sqlc.arg(agent_id)
  AND session_id = sqlc.arg(session_id);



-- name: SetTaskNeedsOperator :execrows


UPDATE tasks SET needs_operator = sqlc.arg(reason), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: SetTaskReactionReminders :execrows


UPDATE tasks SET reaction_reminders = sqlc.arg(count), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: ListTaskOriginChildren :many



SELECT id FROM tasks
WHERE origin_task_id = sqlc.arg(origin_task_id)
ORDER BY rowid;

-- name: ListLiveReactionTasksByOrigin :many





SELECT * FROM tasks
WHERE origin_task_id = sqlc.arg(origin_task_id)
  AND reaction = 1
  AND automated = 1
  AND archived_at IS NULL
  AND state NOT IN ('completed', 'cancelled', 'failed')
ORDER BY rowid DESC;

-- name: ListStalledReactionTasks :many




SELECT t.* FROM tasks t
JOIN agents g ON g.actor_id = t.assignee_actor_id
WHERE t.reaction = 1
  AND t.automated = 1
  AND t.origin_task_id IS NOT NULL
  AND t.archived_at IS NULL
  AND t.state IN ('waiting', 'queued', 'input', 'settled')
  AND g.last_online_at IS NOT NULL
  AND g.last_online_at <= sqlc.arg(cutoff_ms)
ORDER BY t.rowid;

-- name: ListPendingSteeringTasks :many


SELECT id, pending_steering FROM tasks
WHERE pending_steering <> ''
  AND state IN ('waiting', 'queued', 'input', 'settled', 'failed');
