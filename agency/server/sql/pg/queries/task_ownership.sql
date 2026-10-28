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







-- name: ListTaskTeams :many
SELECT t.id AS task_id, tt.team_id
FROM task_teams tt
JOIN tasks t ON t.rowid = tt.task_rowid
ORDER BY t.id, tt.team_id;

-- name: ListTaskTeamsByTask :many
SELECT tt.team_id
FROM task_teams tt
WHERE tt.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY tt.team_id;

-- name: DeleteTaskTeams :exec
DELETE FROM task_teams
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskTeamsByProjectAgent :exec


DELETE FROM task_teams WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id)
    AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteTaskTeamsByTeam :exec
DELETE FROM task_teams WHERE team_id = sqlc.arg(team_id);

-- name: DeleteTaskTeamsByProjectTeam :exec

DELETE FROM task_teams
WHERE team_id = sqlc.arg(team_id) AND task_rowid IN (
  SELECT rowid FROM tasks WHERE project_id = sqlc.arg(project_id)
);

-- name: AddTaskTeam :exec
INSERT INTO task_teams (task_rowid, team_id)
VALUES ((SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)), sqlc.arg(team_id))
ON CONFLICT DO NOTHING;

-- name: IsActorInTaskTeam :one
SELECT EXISTS (
  SELECT 1
  FROM task_teams tt
  JOIN team_members tm ON tm.team_id = tt.team_id
  WHERE tt.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
    AND tm.actor_id = sqlc.arg(actor_id)
);

-- name: ListTaskTeamMemberActorIDs :many

SELECT DISTINCT tm.actor_id
FROM task_teams tt
JOIN team_members tm ON tm.team_id = tt.team_id
WHERE tt.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: CountTasksOwnedByTeam :one
SELECT CAST(COUNT(*) AS BIGINT) FROM task_teams WHERE team_id = sqlc.arg(team_id);

-- name: ListActorProjectTeams :many


SELECT ptm.team_id
FROM project_team_memberships ptm
JOIN team_members tm ON tm.team_id = ptm.team_id
WHERE ptm.project_id = sqlc.arg(project_id) AND tm.actor_id = sqlc.arg(actor_id)
ORDER BY ptm.team_id;



-- name: ListTaskLeads :many
SELECT t.id AS task_id, tl.actor_id
FROM task_leads tl
JOIN tasks t ON t.rowid = tl.task_rowid
ORDER BY t.id, tl.actor_id;

-- name: ListTaskLeadsByTask :many
SELECT tl.actor_id
FROM task_leads tl
WHERE tl.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY tl.actor_id;

-- name: DeleteTaskLeads :exec
DELETE FROM task_leads
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskLeadsByProjectAgent :exec
DELETE FROM task_leads WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id)
    AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteTaskLeadsByActor :exec
DELETE FROM task_leads WHERE actor_id = sqlc.arg(actor_id);

-- name: AddTaskLead :exec
INSERT INTO task_leads (task_rowid, actor_id, added_at)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(actor_id),
  sqlc.arg(added_at)
)
ON CONFLICT DO NOTHING;

-- name: IsActorTaskLead :one
SELECT EXISTS (
  SELECT 1 FROM task_leads
  WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
    AND actor_id = sqlc.arg(actor_id)
);



-- name: AddTaskSubscription :exec
INSERT INTO task_subscriptions (task_rowid, actor_id, created_at)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(actor_id),
  sqlc.arg(created_at)
)
ON CONFLICT DO NOTHING;

-- name: DeleteTaskSubscription :exec
DELETE FROM task_subscriptions
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND actor_id = sqlc.arg(actor_id);

-- name: DeleteTaskSubscriptions :exec
DELETE FROM task_subscriptions
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskSubscriptionsByProjectAgent :exec
DELETE FROM task_subscriptions WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id)
    AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteTaskSubscriptionsByActor :exec
DELETE FROM task_subscriptions WHERE actor_id = sqlc.arg(actor_id);

-- name: IsActorSubscribedToTask :one
SELECT EXISTS (
  SELECT 1 FROM task_subscriptions
  WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
    AND actor_id = sqlc.arg(actor_id)
);

-- name: ListTaskSubscriptionsByTask :many
SELECT ts.actor_id
FROM task_subscriptions ts
WHERE ts.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY ts.actor_id;

-- name: ListTaskNotificationRecipients :many

SELECT DISTINCT actor_id FROM (
  SELECT tl.actor_id
  FROM task_leads tl
  WHERE tl.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  UNION
  SELECT ts.actor_id
  FROM task_subscriptions ts
  WHERE ts.task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
) AS recipients
ORDER BY actor_id;



-- name: DeleteTaskTeamPrompts :exec
DELETE FROM task_team_prompts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: AddTaskTeamPrompt :exec
INSERT INTO task_team_prompts (task_rowid, team_id, prompt_id, project_prompt_id)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(team_id),
  sqlc.narg(prompt_id),
  sqlc.narg(project_prompt_id)
)
ON CONFLICT (task_rowid, team_id) DO UPDATE SET
  prompt_id = excluded.prompt_id,
  project_prompt_id = excluded.project_prompt_id;

-- name: ListTaskTeamPromptsByTask :many
SELECT team_id, prompt_id, project_prompt_id
FROM task_team_prompts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY team_id;

-- name: DeleteTaskTeamPromptsByProjectAgent :exec


DELETE FROM task_team_prompts WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id)
    AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);
