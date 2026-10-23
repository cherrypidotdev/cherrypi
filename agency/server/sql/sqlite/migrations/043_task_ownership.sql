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
-- Task ownership (the phase 6 follow-up): a task is owned by one or more of
-- its project's teams, has zero or more leads drawn from those teams, and any
-- project actor may subscribe to its updates. This supersedes the
-- `participation` level: its link table is dropped and every task follows one
-- implicit rule. The vestigial `tasks.participation` column is left in place
-- for old rows (see docs/LEGACY.md).
--
--  * `task_teams` lists the teams that own a task. A task created after this
--    migration has at least one; legacy tasks have none and fall back to the
--    project-visibility rule.
--  * `task_leads` lists the accountable actors, each a member of an owning
--    team. Leads are shown on the task, notified of its updates, and may
--    prompt it.
--  * `task_subscriptions` is the per-actor opt-in to a task's updates. Any
--    actor with view access may subscribe; leads are implicit recipients.
--  * `notifications.recipient_actor_id` targets an inbox entry at one actor;
--    NULL keeps the pre-ownership broadcast behaviour for legacy and
--    non-task signals.
CREATE TABLE task_teams (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  PRIMARY KEY (task_rowid, team_id)
);
CREATE INDEX idx_task_teams_team ON task_teams (team_id);

CREATE TABLE task_leads (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  added_at INTEGER NOT NULL,
  PRIMARY KEY (task_rowid, actor_id)
);
CREATE INDEX idx_task_leads_actor ON task_leads (actor_id);

CREATE TABLE task_subscriptions (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (task_rowid, actor_id)
);
CREATE INDEX idx_task_subscriptions_actor ON task_subscriptions (actor_id);

ALTER TABLE notifications
  ADD COLUMN recipient_actor_id INTEGER REFERENCES actors (id) ON DELETE CASCADE;
CREATE INDEX idx_notifications_recipient
  ON notifications (recipient_actor_id, read_at);

DROP TABLE task_participation_teams;

-- +goose Down
CREATE TABLE task_participation_teams (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  PRIMARY KEY (task_rowid, team_id)
);
CREATE INDEX idx_task_participation_teams_team ON task_participation_teams (team_id);

DROP INDEX idx_notifications_recipient;
ALTER TABLE notifications DROP COLUMN recipient_actor_id;
DROP TABLE task_subscriptions;
DROP TABLE task_leads;
DROP TABLE task_teams;
