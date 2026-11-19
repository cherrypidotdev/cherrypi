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




ALTER TABLE actors ADD COLUMN emoji TEXT NOT NULL DEFAULT '';
ALTER TABLE actors ADD COLUMN links TEXT NOT NULL DEFAULT '[]';

ALTER TABLE projects ADD COLUMN emoji TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN links TEXT NOT NULL DEFAULT '[]';

ALTER TABLE teams ADD COLUMN emoji TEXT NOT NULL DEFAULT '';
ALTER TABLE teams ADD COLUMN links TEXT NOT NULL DEFAULT '[]';

ALTER TABLE agent_groups ADD COLUMN emoji TEXT NOT NULL DEFAULT '';
ALTER TABLE agent_groups ADD COLUMN links TEXT NOT NULL DEFAULT '[]';











UPDATE projects SET slug = 'tmp!' || id;



WITH candidates AS (
  SELECT id,
         lower(prefix) AS base,
         ROW_NUMBER() OVER (
           PARTITION BY lower(prefix) ORDER BY created_at, id
         ) AS rn
  FROM projects
)
UPDATE projects
SET slug = (
  SELECT CASE WHEN candidates.rn = 1 THEN candidates.base
              ELSE candidates.base || '-' || candidates.rn END
  FROM candidates
  WHERE candidates.id = projects.id
);



CREATE UNIQUE INDEX idx_projects_slug_nocase ON projects (lower(slug));
DROP INDEX idx_projects_prefix;
ALTER TABLE projects DROP COLUMN prefix;

-- +goose Down


CREATE UNIQUE INDEX idx_projects_prefix ON projects (upper(slug));
ALTER TABLE projects ADD COLUMN prefix TEXT NOT NULL DEFAULT '';
UPDATE projects SET prefix = upper(slug);
DROP INDEX idx_projects_slug_nocase;

ALTER TABLE agent_groups DROP COLUMN links;
ALTER TABLE agent_groups DROP COLUMN emoji;

ALTER TABLE teams DROP COLUMN links;
ALTER TABLE teams DROP COLUMN emoji;

ALTER TABLE projects DROP COLUMN links;
ALTER TABLE projects DROP COLUMN emoji;

ALTER TABLE actors DROP COLUMN links;
ALTER TABLE actors DROP COLUMN emoji;