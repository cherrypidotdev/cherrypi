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

-- +goose Up
-- Project hierarchy. Every project may have at most one parent, forming a
-- strict tree (the service rejects cycles). Access flows down the tree:
-- anyone who can view a parent also gets the viewer role (never member or
-- owner) on every descendant, unless they hold a stronger explicit grant
-- there.
ALTER TABLE projects ADD COLUMN parent_project_id TEXT REFERENCES projects (id);
CREATE INDEX idx_projects_parent ON projects (parent_project_id);

-- Materialized ancestor closure, maintained by the service whenever a parent
-- link changes. It keeps access and reference queries non-recursive: a row
-- (project_id, ancestor_id, depth) means `ancestor_id` is `depth` levels above
-- `project_id`.
CREATE TABLE project_ancestors (
  project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  ancestor_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  depth INTEGER NOT NULL CHECK (depth > 0),
  PRIMARY KEY (project_id, ancestor_id)
);
CREATE INDEX idx_project_ancestors_ancestor ON project_ancestors (ancestor_id);

-- Becoming a child widens a project's audience to the parent's viewers, so it
-- is consented to like a team attachment: the parent's owner creates an
-- invitation and an owner of the child (direct or through an owner-role team)
-- accepts it. When both projects share an owner-role team, a member of that
-- team may link them directly; administrators skip both steps.
CREATE TABLE project_parent_invitations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  parent_project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  child_project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  invited_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'accepted', 'declined', 'revoked')),
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  responded_at INTEGER,
  responded_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX idx_project_parent_invitations_pair
  ON project_parent_invitations (parent_project_id, child_project_id)
  WHERE status = 'pending';
-- A project can only have one parent, so it can only have one pending
-- parent invitation at a time.
CREATE UNIQUE INDEX idx_project_parent_invitations_child_pending
  ON project_parent_invitations (child_project_id) WHERE status = 'pending';
CREATE INDEX idx_project_parent_invitations_parent
  ON project_parent_invitations (parent_project_id, status);
CREATE INDEX idx_project_parent_invitations_child
  ON project_parent_invitations (child_project_id, status);
CREATE INDEX idx_project_parent_invitations_expires
  ON project_parent_invitations (expires_at);

-- +goose Down
DROP TABLE project_parent_invitations;
DROP TABLE project_ancestors;
DROP INDEX idx_projects_parent;
ALTER TABLE projects DROP COLUMN parent_project_id;
