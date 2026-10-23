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

-- Project hierarchy: parent links, parent invitations, shared owner teams and
-- the reference-shape helpers. The policy lives in internal/projects.

-- name: ListProjectAncestorIDs :many
-- Every strict ancestor of a project, nearest first.
SELECT ancestor_id FROM project_ancestors
WHERE project_id = sqlc.arg(project_id)
ORDER BY depth ASC;

-- name: ListProjectDescendantIDs :many
-- Every strict descendant of a project, nearest first.
SELECT project_id FROM project_ancestors
WHERE ancestor_id = sqlc.arg(project_id)
ORDER BY depth ASC, project_id ASC;

-- name: IsProjectDescendant :one
-- Reports whether `ancestor_id` is a strict ancestor of `project_id`.
SELECT EXISTS (
  SELECT 1 FROM project_ancestors
  WHERE project_id = sqlc.arg(project_id)
    AND ancestor_id = sqlc.arg(ancestor_id)
);

-- name: DeleteAllProjectAncestors :exec
DELETE FROM project_ancestors;

-- name: InsertProjectAncestor :exec
INSERT INTO project_ancestors (project_id, ancestor_id, depth)
VALUES (sqlc.arg(project_id), sqlc.arg(ancestor_id), sqlc.arg(depth));

-- name: DeleteProjectAncestorRowsByProject :exec
DELETE FROM project_ancestors
WHERE project_id = sqlc.arg(project_id) OR ancestor_id = sqlc.arg(project_id);

-- name: ListProjectSharedOwnerTeams :many
-- Teams with the owner role on both projects that the actor belongs to. A
-- member of one may link the pair directly.
SELECT t.id AS team_id, t.name AS team_name
FROM project_team_memberships parent_link
JOIN project_team_memberships child_link
  ON child_link.team_id = parent_link.team_id
JOIN teams t ON t.id = parent_link.team_id
JOIN team_members tm ON tm.team_id = parent_link.team_id
WHERE parent_link.project_id = sqlc.arg(parent_project_id)
  AND parent_link.role = 'owner'
  AND child_link.project_id = sqlc.arg(child_project_id)
  AND child_link.role = 'owner'
  AND tm.actor_id = sqlc.arg(actor_id)
ORDER BY t.name COLLATE NOCASE;

-- name: CreateProjectParentInvitation :one
INSERT INTO project_parent_invitations (
  parent_project_id, child_project_id, invited_by_actor_id, note,
  created_at, expires_at
)
VALUES (
  sqlc.arg(parent_project_id), sqlc.arg(child_project_id),
  sqlc.narg(invited_by_actor_id), sqlc.arg(note),
  sqlc.arg(created_at), sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetProjectParentInvitation :one
SELECT * FROM project_parent_invitations WHERE id = sqlc.arg(id);

-- name: GetPendingProjectParentInvitationForChild :one
SELECT * FROM project_parent_invitations
WHERE child_project_id = sqlc.arg(child_project_id) AND status = 'pending'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPendingProjectParentInvitation :one
SELECT * FROM project_parent_invitations
WHERE parent_project_id = sqlc.arg(parent_project_id)
  AND child_project_id = sqlc.arg(child_project_id)
  AND status = 'pending';

-- name: ListProjectParentInvitationsByParent :many
SELECT
  i.*,
  parent.name AS parent_name,
  parent.slug AS parent_slug,
  child.name AS child_name,
  child.slug AS child_slug,
  inviter.username AS invited_by_username,
  responder.username AS responded_by_username
FROM project_parent_invitations i
JOIN projects parent ON parent.id = i.parent_project_id
JOIN projects child ON child.id = i.child_project_id
LEFT JOIN actors inviter ON inviter.id = i.invited_by_actor_id
LEFT JOIN actors responder ON responder.id = i.responded_by_actor_id
WHERE i.parent_project_id = sqlc.arg(parent_project_id)
ORDER BY i.created_at DESC, i.id DESC;

-- name: ListProjectParentInvitationsByChild :many
SELECT
  i.*,
  parent.name AS parent_name,
  parent.slug AS parent_slug,
  child.name AS child_name,
  child.slug AS child_slug,
  inviter.username AS invited_by_username,
  responder.username AS responded_by_username
FROM project_parent_invitations i
JOIN projects parent ON parent.id = i.parent_project_id
JOIN projects child ON child.id = i.child_project_id
LEFT JOIN actors inviter ON inviter.id = i.invited_by_actor_id
LEFT JOIN actors responder ON responder.id = i.responded_by_actor_id
WHERE i.child_project_id = sqlc.arg(child_project_id)
ORDER BY i.created_at DESC, i.id DESC;

-- name: RespondProjectParentInvitation :execrows
UPDATE project_parent_invitations
SET status = sqlc.arg(status),
    responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: RevokeProjectParentInvitation :execrows
UPDATE project_parent_invitations
SET status = 'revoked',
    responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: RevokePendingProjectParentInvitationsForChild :execrows
UPDATE project_parent_invitations
SET status = 'revoked',
    responded_at = sqlc.arg(responded_at),
    responded_by_actor_id = sqlc.narg(responded_by_actor_id)
WHERE child_project_id = sqlc.arg(child_project_id) AND status = 'pending';

-- name: ExpireProjectParentInvitations :execrows
UPDATE project_parent_invitations
SET status = 'revoked', responded_at = sqlc.arg(responded_at)
WHERE status = 'pending' AND expires_at < sqlc.arg(now);

-- name: DeleteProjectParentInvitationsByProject :exec
DELETE FROM project_parent_invitations
WHERE parent_project_id = sqlc.arg(project_id)
   OR child_project_id = sqlc.arg(project_id);

-- name: ListCrossProjectTaskReferences :many
-- Every reference edge whose endpoints live in different projects, used to
-- reject a hierarchy change that would make an existing edge illegal.
SELECT
  task.id AS task_id,
  task.project_id AS task_project_id,
  referenced.id AS referenced_task_id,
  referenced.project_id AS referenced_project_id
FROM task_references r
JOIN tasks task ON task.rowid = r.task_rowid
JOIN tasks referenced ON referenced.rowid = r.referenced_task_rowid
WHERE task.project_id IS NOT NULL
  AND referenced.project_id IS NOT NULL
  AND task.project_id <> referenced.project_id
ORDER BY task.id, referenced.id;
