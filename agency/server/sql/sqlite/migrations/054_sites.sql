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



CREATE TABLE sites (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  only_creator_may_edit INTEGER NOT NULL DEFAULT 1,
  agency_wide INTEGER NOT NULL DEFAULT 0,
  creator_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  origin_task_id TEXT NOT NULL DEFAULT '',
  attachment_id INTEGER NOT NULL REFERENCES attachments (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX idx_sites_origin_task ON sites (origin_task_id);
CREATE INDEX idx_sites_creator ON sites (creator_actor_id);
CREATE INDEX idx_sites_attachment ON sites (attachment_id);



ALTER TABLE personal_access_tokens ADD COLUMN site_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_personal_access_tokens_site ON personal_access_tokens (site_id);

-- +goose Down
DROP INDEX idx_personal_access_tokens_site;
ALTER TABLE personal_access_tokens DROP COLUMN site_id;
DROP TABLE sites;
