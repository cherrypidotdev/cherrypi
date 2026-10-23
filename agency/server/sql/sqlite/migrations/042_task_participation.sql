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
-- Task collaboration (auth phase 6): a task records its creator, the
-- participation level that gates who may prompt its agent, the teams that
-- level names, and the real actor behind every message and log entry.
--
-- Comments were dropped from the plan (see docs/auth.md), so no comment table
-- is created here.
--
--  * `tasks.creator_actor_id` is the actor that created the task. It is
--    nullable: rows that predate this migration keep no creator, and the
--    participation floor then falls back to the project rule.
--  * `tasks.participation` is `project` (default), `teams`, or `assignee`.
--    The creator may always prompt regardless of the level (the locked
--    participation floor).
--  * `task_participation_teams` lists the teams allowed to prompt at the
--    `teams` level.
--  * `task_messages.actor_id` and `task_logs.actor_id` attribute a prompt or
--    a change to the actor that made it. Both are nullable so deleting an
--    actor keeps the history and legacy rows stay readable (foreign keys are
--    off at runtime; see 040).
--  * `tasks.pending_prompt_actor_id` carries the sender of a follow-up
--    stashed on a resting task, so the scheduler can attribute it when it
--    finally delivers the prompt.

ALTER TABLE tasks ADD COLUMN creator_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN participation TEXT NOT NULL DEFAULT 'project'
  CHECK (participation IN ('project', 'teams', 'assignee'));
ALTER TABLE tasks ADD COLUMN pending_prompt_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL;
CREATE INDEX idx_tasks_creator ON tasks (creator_actor_id);

-- The teams a task is restricted to at the `teams` participation level.
CREATE TABLE task_participation_teams (
  task_rowid INTEGER NOT NULL REFERENCES tasks (rowid) ON DELETE CASCADE,
  team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  PRIMARY KEY (task_rowid, team_id)
);
CREATE INDEX idx_task_participation_teams_team ON task_participation_teams (team_id);

ALTER TABLE task_messages ADD COLUMN actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL;
CREATE INDEX idx_task_messages_actor ON task_messages (actor_id);

ALTER TABLE task_logs ADD COLUMN actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL;
CREATE INDEX idx_task_logs_actor ON task_logs (actor_id);

-- +goose Down
DROP INDEX idx_task_logs_actor;
ALTER TABLE task_logs DROP COLUMN actor_id;
DROP INDEX idx_task_messages_actor;
ALTER TABLE task_messages DROP COLUMN actor_id;
DROP TABLE task_participation_teams;
DROP INDEX idx_tasks_creator;
ALTER TABLE tasks DROP COLUMN pending_prompt_actor_id;
ALTER TABLE tasks DROP COLUMN participation;
ALTER TABLE tasks DROP COLUMN creator_actor_id;
