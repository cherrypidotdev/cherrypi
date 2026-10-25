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























ALTER TABLE tasks ADD COLUMN creator_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN participation TEXT NOT NULL DEFAULT 'project'
  CHECK (participation IN ('project', 'teams', 'assignee'));
ALTER TABLE tasks ADD COLUMN pending_prompt_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL;
CREATE INDEX idx_tasks_creator ON tasks (creator_actor_id);


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
