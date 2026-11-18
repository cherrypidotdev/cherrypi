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



















DROP INDEX IF EXISTS idx_tasks_parent;
ALTER TABLE tasks DROP COLUMN parent_task_id;

ALTER TABLE tasks DROP COLUMN interactive;
ALTER TABLE tasks ADD COLUMN automated INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN needs_operator TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN reaction_reminders INTEGER NOT NULL DEFAULT 0;

ALTER TABLE task_references
  ADD COLUMN wake_on_resolve_when_many INTEGER NOT NULL DEFAULT 0;



DELETE FROM task_leads WHERE (task_rowid, actor_id) IN (
  SELECT task_rowid, actor_id FROM (
    SELECT task_rowid, actor_id,
           ROW_NUMBER() OVER (
             PARTITION BY task_rowid ORDER BY added_at DESC, actor_id DESC
           ) AS rn
    FROM task_leads
  ) WHERE rn > 1
);
INSERT OR IGNORE INTO task_leads (task_rowid, actor_id, added_at)
SELECT t.rowid, t.creator_actor_id, t.created_at
FROM tasks t
WHERE t.creator_actor_id IS NOT NULL
  AND t.state NOT IN ('backlog', 'cancelled')
  AND NOT EXISTS (SELECT 1 FROM task_leads l WHERE l.task_rowid = t.rowid);
CREATE UNIQUE INDEX idx_task_leads_single ON task_leads (task_rowid);



ALTER TABLE agents ADD COLUMN last_online_at INTEGER;

-- +goose Down



DROP INDEX IF EXISTS idx_task_leads_single;
ALTER TABLE agents DROP COLUMN last_online_at;
ALTER TABLE task_references DROP COLUMN wake_on_resolve_when_many;
ALTER TABLE tasks DROP COLUMN reaction_reminders;
ALTER TABLE tasks DROP COLUMN needs_operator;
ALTER TABLE tasks DROP COLUMN automated;
ALTER TABLE tasks ADD COLUMN interactive INTEGER NOT NULL DEFAULT 1;
ALTER TABLE tasks ADD COLUMN parent_task_id TEXT REFERENCES tasks (id)
  ON DELETE SET NULL;
CREATE INDEX idx_tasks_parent ON tasks (parent_task_id);
