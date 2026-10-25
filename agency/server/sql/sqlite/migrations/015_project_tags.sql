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













ALTER TABLE tags ADD COLUMN project_id TEXT REFERENCES projects (id);
DROP INDEX idx_tags_name;

CREATE TEMP TABLE tag_fork_map (
  old_tag_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  new_tag_id TEXT NOT NULL,
  PRIMARY KEY (old_tag_id, project_id)
);

INSERT INTO tag_fork_map (old_tag_id, project_id, new_tag_id)
SELECT DISTINCT tt.tag_id, t.project_id, lower(hex(randomblob(16)))
FROM task_tags tt
JOIN tasks t ON t.rowid = tt.task_rowid
WHERE t.project_id IS NOT NULL;

INSERT INTO tags (
  id, name, description, color, archived_at, created_at, updated_at, project_id
)
SELECT
  m.new_tag_id, tg.name, tg.description, tg.color, tg.archived_at,
  tg.created_at, tg.updated_at, m.project_id
FROM tag_fork_map m
JOIN tags tg ON tg.id = m.old_tag_id;


UPDATE task_tags
SET tag_id = (
  SELECT m.new_tag_id
  FROM tag_fork_map m
  JOIN tasks t ON t.rowid = task_tags.task_rowid
  WHERE m.old_tag_id = task_tags.tag_id AND m.project_id = t.project_id
)
WHERE task_rowid IN (SELECT rowid FROM tasks WHERE project_id IS NOT NULL);


DELETE FROM task_tags
WHERE task_rowid IN (SELECT rowid FROM tasks WHERE project_id IS NULL);


UPDATE agent_project_tag_limits
SET tag_id = (
  SELECT m.new_tag_id
  FROM tag_fork_map m
  WHERE m.old_tag_id = agent_project_tag_limits.tag_id
    AND m.project_id = agent_project_tag_limits.project_id
);



DELETE FROM agent_project_tag_limits
WHERE tag_id NOT IN (SELECT new_tag_id FROM tag_fork_map);

DROP TABLE tag_fork_map;



ALTER TABLE agent_tag_limits RENAME TO agent_tag_limits_old;
CREATE TABLE agent_tag_limits (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  tag_name TEXT NOT NULL,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, tag_name)
);
INSERT INTO agent_tag_limits (agent_id, tag_name, max_concurrent)
SELECT o.agent_id, lower(tg.name), o.max_concurrent
FROM agent_tag_limits_old o
JOIN tags tg ON tg.id = o.tag_id;
DROP TABLE agent_tag_limits_old;

DELETE FROM tags WHERE project_id IS NULL;

CREATE UNIQUE INDEX idx_tags_project_name ON tags (project_id, lower(name));

-- +goose Down


DROP INDEX idx_tags_project_name;

CREATE TEMP TABLE tag_collapse_map AS
SELECT
  id AS old_tag_id,
  (SELECT MIN(t2.id) FROM tags t2 WHERE lower(t2.name) = lower(tags.name))
    AS new_tag_id
FROM tags;

UPDATE task_tags
SET tag_id = (
  SELECT m.new_tag_id FROM tag_collapse_map m WHERE m.old_tag_id = task_tags.tag_id
);

DELETE FROM task_tags
WHERE rowid NOT IN (SELECT MIN(rowid) FROM task_tags GROUP BY task_rowid, tag_id);

UPDATE agent_project_tag_limits
SET tag_id = (
  SELECT m.new_tag_id
  FROM tag_collapse_map m
  WHERE m.old_tag_id = agent_project_tag_limits.tag_id
);

DELETE FROM agent_project_tag_limits
WHERE rowid NOT IN (
  SELECT MIN(rowid)
  FROM agent_project_tag_limits
  GROUP BY agent_id, project_id, tag_id
);

DELETE FROM tags WHERE id NOT IN (SELECT new_tag_id FROM tag_collapse_map);

ALTER TABLE tags DROP COLUMN project_id;

CREATE UNIQUE INDEX idx_tags_name ON tags (lower(name));

DROP TABLE tag_collapse_map;

DROP TABLE agent_tag_limits;
CREATE TABLE agent_tag_limits (
  agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
  max_concurrent INTEGER NOT NULL,
  PRIMARY KEY (agent_id, tag_id)
);
