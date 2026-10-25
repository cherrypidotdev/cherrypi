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

















CREATE TABLE user_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  at INTEGER NOT NULL,
  actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  kind TEXT NOT NULL,
  field TEXT NOT NULL DEFAULT '',
  old_value TEXT NOT NULL DEFAULT '',
  new_value TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  
  data_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_user_logs_user ON user_logs (user_actor_id, at DESC, id DESC);
CREATE INDEX idx_user_logs_at ON user_logs (at DESC, id DESC);
CREATE INDEX idx_user_logs_actor ON user_logs (actor_id, at DESC, id DESC);

CREATE TABLE agent_domain_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  at INTEGER NOT NULL,
  actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  kind TEXT NOT NULL,
  field TEXT NOT NULL DEFAULT '',
  old_value TEXT NOT NULL DEFAULT '',
  new_value TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  
  data_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_agent_domain_logs_agent ON agent_domain_logs (agent_actor_id, at DESC, id DESC);
CREATE INDEX idx_agent_domain_logs_at ON agent_domain_logs (at DESC, id DESC);
CREATE INDEX idx_agent_domain_logs_actor ON agent_domain_logs (actor_id, at DESC, id DESC);



CREATE INDEX idx_task_logs_at ON task_logs (at DESC, id DESC);
CREATE INDEX idx_project_logs_at ON project_logs (at DESC, id DESC);
CREATE INDEX idx_project_logs_actor ON project_logs (actor_id, at DESC, id DESC);
CREATE INDEX idx_team_logs_at ON team_logs (at DESC, id DESC);
CREATE INDEX idx_team_logs_actor ON team_logs (actor_id, at DESC, id DESC);

-- +goose Down
DROP INDEX idx_team_logs_actor;
DROP INDEX idx_team_logs_at;
DROP INDEX idx_project_logs_actor;
DROP INDEX idx_project_logs_at;
DROP INDEX idx_task_logs_at;
DROP TABLE agent_domain_logs;
DROP TABLE user_logs;
