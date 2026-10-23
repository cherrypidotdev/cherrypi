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
  CAST(sqlc.narg(creator_actor_id) AS INTEGER),
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
SELECT * FROM tasks WHERE id = ?;

-- name: GetTaskByProjectSeq :one
SELECT * FROM tasks WHERE project_id = ? AND seq = ?;

-- ListTasks returns tasks newest-first (by insertion order), narrowed by the
-- optional filters.
-- Tasks whose project is archived are hidden from the all-projects list
-- unless a project is named explicitly, matching the "archive hides its
-- tasks" rule. `archived` selects the active list (0) or the archive (1).
-- name: ListTasks :many
SELECT t.* FROM tasks t
WHERE (
    CAST(sqlc.narg(project_id) AS TEXT) IS NULL
    OR t.project_id = sqlc.narg(project_id)
  )
  AND (
    CAST(sqlc.narg(agent_id) AS INTEGER) IS NULL
    OR t.assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.narg(agent_id))
  )
  AND (
    CAST(sqlc.narg(state) AS TEXT) IS NULL
    OR t.state = sqlc.narg(state)
  )
  AND (
    (CAST(sqlc.arg(archived) AS INTEGER) = 1 AND t.archived_at IS NOT NULL)
    OR (CAST(sqlc.arg(archived) AS INTEGER) = 0 AND t.archived_at IS NULL)
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
    -- The human-readable `<prefix>-<seq>` id, so "CHERRYPI-61" finds that
    -- task (the comparison is case-insensitive).
    OR lower(t.id) LIKE '%' || lower(sqlc.narg(search)) || '%'
    OR (
      -- A purely numeric query also matches the per-project task sequence, so
      -- "61" finds CHERRYPI-61 and CHERRYPI-612.
      sqlc.narg(search) NOT GLOB '*[^0-9]*'
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
-- The scheduler's queue: unblocked queued tasks only. A queued task with an
-- incomplete blocking reference stays queued (reported as blocked) but is
-- skipped here until every blocking reference is completed.
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
-- The scheduler's reconcile reads every in-flight run (any agent) to
-- materialize settlement when a run finishes without a watcher.
SELECT * FROM tasks
WHERE state IN ('running', 'input') AND archived_at IS NULL
ORDER BY created_at ASC;

-- name: UpdateDraftTask :one
-- `new_id` and `seq` carry a task that moved to another project (the id's
-- prefix is the project's), and equal the current values otherwise.
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
-- Draft -> backlog: very likely to run, but not queued yet.
UPDATE tasks SET state = 'backlog', updated_at = ?, error = ''
WHERE id = ? AND state = 'draft';

-- name: DemoteTask :execrows
-- Backlog -> draft.
UPDATE tasks SET state = 'draft', updated_at = ?, error = ''
WHERE id = ? AND state = 'backlog';

-- name: QueueTask :execrows
-- Backlog -> queued. A queued task with incomplete blocking references is
-- reported as blocked but stays queued; the scheduler skips it.
UPDATE tasks SET state = 'queued', queued_at = COALESCE(queued_at, ?), updated_at = ?, error = ''
WHERE id = ? AND state = 'backlog';

-- name: DequeueTask :execrows
-- Queued -> backlog, without starting it.
UPDATE tasks SET state = 'backlog', updated_at = ?, error = ''
WHERE id = ? AND state = 'queued';

-- ClaimTask flips a queued task to running and assigns it. It is conditional
-- on the state so two drains cannot both start the same task; 0 rows means
-- someone else claimed it. Prompt ids are only written on the first run
-- (the caller passes NULL otherwise), hence COALESCE.
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
UPDATE tasks SET state = 'settled', settled_at = ?, updated_at = ?, error = ''
WHERE id = ? AND state IN ('running', 'input');

-- name: MarkTaskInput :execrows
-- A running task is blocked on an extension dialog. It keeps its slot.
UPDATE tasks SET state = 'input', updated_at = ?
WHERE id = ? AND state = 'running';

-- name: ClearTaskInput :execrows
-- The dialog was answered; the run is in flight again.
UPDATE tasks SET state = 'running', updated_at = ?
WHERE id = ? AND state = 'input';

-- name: ResumeTask :execrows
-- A settled or failed task resumes to running when a follow-up prompt is
-- accepted. Its slot was never released, so no re-claim is needed.
UPDATE tasks SET state = 'running', settled_at = ?, updated_at = ?
WHERE id = ? AND state IN ('settled', 'failed');

-- name: FailTask :execrows
UPDATE tasks SET state = 'failed', error = ?, settled_at = ?, updated_at = ?
WHERE id = ? AND state IN ('running', 'input');

-- name: CompleteTask :execrows
-- name: CompleteTask :execrows
-- A user closes out a non-running task as done. A cancelled task can be
-- switched back to completed (and vice versa).
UPDATE tasks SET state = 'completed', settled_at = ?, updated_at = ?, error = '',
  pending_prompt = '', pending_prompt_actor_id = NULL
WHERE id = ? AND state IN ('draft', 'backlog', 'queued', 'settled', 'failed', 'cancelled');

-- name: CancelTask :execrows
-- A user closes out a non-running task as abandoned. A completed task can be
-- switched back to cancelled (and vice versa).
UPDATE tasks SET state = 'cancelled', settled_at = ?, updated_at = ?,
  pending_prompt = '', pending_prompt_actor_id = NULL
WHERE id = ? AND state IN ('draft', 'backlog', 'queued', 'settled', 'failed', 'completed');

-- name: ArchiveTask :execrows
-- Hide a completed or cancelled task. Archive is a flag, so the lifecycle
-- state is preserved; no other state can be archived.
UPDATE tasks SET archived_at = ?, updated_at = ?
WHERE id = ? AND archived_at IS NULL AND state IN ('completed', 'cancelled');

-- name: UnarchiveTask :execrows
UPDATE tasks SET archived_at = NULL, updated_at = ?
WHERE id = ? AND archived_at IS NOT NULL;

-- name: RequeueTaskWithPrompt :execrows
-- A follow-up to a resting task is stored as the pending prompt and the task
-- is queued; the scheduler delivers the prompt when the task is next started.
-- Every resting state requeues here, so sending a follow-up never waits on
-- the agent tunnel; a stale queued_at is kept for ordering.
UPDATE tasks SET
  state = 'queued', error = '', pending_prompt = sqlc.arg(pending_prompt),
  pending_prompt_actor_id = sqlc.narg(pending_prompt_actor_id),
  queued_at = COALESCE(queued_at, sqlc.arg(queued_at)), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND state IN ('completed', 'cancelled', 'settled', 'failed');

-- name: ClearTaskPendingPrompt :execrows
UPDATE tasks SET pending_prompt = '', pending_prompt_actor_id = NULL, updated_at = ?
WHERE id = ?;

-- name: SetTaskPrompt :execrows
-- Replace the prompt of a task that has not been queued yet (draft or
-- backlog). A queued or started task's prompt is pinned.
UPDATE tasks SET prompt = ?, updated_at = ?
WHERE id = ? AND state IN ('draft', 'backlog');

-- name: SetTaskPendingPrompt :execrows
-- Replace a draft task's pending follow-up (the next message a re-queue
-- delivers). An empty value clears it.
UPDATE tasks SET pending_prompt = sqlc.arg(pending_prompt),
  pending_prompt_actor_id = sqlc.narg(pending_prompt_actor_id),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND state = 'draft';

-- name: RequeueTask :execrows
UPDATE tasks SET
  state = 'queued', error = ?, queued_at = COALESCE(queued_at, ?), updated_at = ?
WHERE id = ? AND state = 'running';

-- name: SetTaskError :execrows
UPDATE tasks SET error = ?, updated_at = ? WHERE id = ?;

-- name: ReopenTask :execrows
UPDATE tasks SET state = ?, settled_at = NULL, error = '', pending_prompt = '',
  pending_prompt_actor_id = NULL, updated_at = ?
WHERE id = ? AND state IN ('settled', 'failed', 'completed', 'cancelled');

-- name: SetTaskSession :execrows
-- Record the agent-side Pi session a task started and the agent that owns it.
-- session_agent_id survives a later reassignment: the scheduler compares it
-- with the picked agent to tell a resume from a cross-agent continuation.
-- A new session also clears the cleanup mark: the task has a live session
-- again and a future follow-up resumes it instead of starting fresh.
UPDATE tasks SET
  session_id = ?, session_agent_id = ?, session_deleted_at = NULL, updated_at = ?
WHERE id = ?;

-- name: RenameTask :one
UPDATE tasks SET name = ?, updated_at = ? WHERE id = ? RETURNING *;

-- name: SetTaskPriority :one
UPDATE tasks SET priority = ?, updated_at = ? WHERE id = ? RETURNING *;

-- name: SetTaskSize :one
UPDATE tasks SET size = ?, updated_at = ? WHERE id = ? RETURNING *;

-- name: MarkTaskSessionDeleted :one
-- Record that a task's agent-side Pi session and workspace were cleaned up.
-- The end state is idempotent: the first cleanup wins.
UPDATE tasks SET session_deleted_at = ?, updated_at = ?
WHERE id = ? AND session_deleted_at IS NULL
RETURNING *;
-- name: SetTaskAssignee :one
-- Pin a task to an agent, or clear the pin (assignee_actor_id NULL,
-- assignee_pinned 0) so the scheduler picks one. Allowed in any state; the
-- service rejects a live run before calling this.
UPDATE tasks
SET
  assignee_actor_id = sqlc.narg(assignee_actor_id),
  assignee_pinned = sqlc.arg(assignee_pinned),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

-- name: RenameProjectTasks :exec
-- Rewrite a project's rendered task ids after its prefix changed. `seq` and
-- every child reference are untouched: they key on the task rowid, so only
-- `tasks.id` (the denormalized display key) moves.
UPDATE tasks
SET id = sqlc.arg(prefix) || '-' || seq
WHERE project_id = sqlc.arg(project_id);

-- name: ListTasksByProject :many
SELECT * FROM tasks WHERE project_id = ?;

-- name: ListTasksByProjectAgent :many
SELECT t.* FROM tasks t
JOIN agents g ON g.actor_id = t.assignee_actor_id
WHERE t.project_id = sqlc.arg(project_id) AND g.id = sqlc.arg(agent_id);

-- name: CountTasksByProject :one
SELECT count(*) FROM tasks WHERE project_id = ?;

-- name: DeleteTasksByProjectAgent :exec
DELETE FROM tasks WHERE rowid IN (
  SELECT t.rowid FROM tasks t
  JOIN agents g ON g.actor_id = t.assignee_actor_id
  WHERE t.project_id = sqlc.arg(project_id) AND g.id = sqlc.arg(agent_id)
);

-- name: ListRunningTasksByAgent :many
-- Running, input, settled, and failed tasks all hold an agent slot; archived
-- tasks never do.
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
