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








CREATE TABLE knowledge_concepts (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL CHECK (scope IN ('org', 'project', 'agent')),
  
  scope_id TEXT NOT NULL DEFAULT '',
  path TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'Note',
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  resource TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '[]',
  body TEXT NOT NULL DEFAULT '',
  frontmatter_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'stable'
    CHECK (status IN ('draft', 'stable', 'deprecated')),
  stale_after INTEGER,
  generated_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  generated_by TEXT NOT NULL DEFAULT '',
  generated_at INTEGER NOT NULL,
  trust_tier TEXT NOT NULL DEFAULT 'unverified'
    CHECK (trust_tier IN ('unverified', 'machine-confirmed', 'human-reviewed')),
  revision INTEGER NOT NULL DEFAULT 1,
  protected INTEGER NOT NULL DEFAULT 0,
  deleted_at INTEGER,
  created_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  UNIQUE (scope, scope_id, path)
);
CREATE INDEX idx_knowledge_concepts_scope
  ON knowledge_concepts (scope, scope_id, updated_at DESC);
CREATE INDEX idx_knowledge_concepts_path
  ON knowledge_concepts (path, scope, scope_id);
CREATE INDEX idx_knowledge_concepts_updated
  ON knowledge_concepts (updated_at DESC, id);



CREATE TABLE knowledge_revisions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  concept_id TEXT NOT NULL REFERENCES knowledge_concepts (id) ON DELETE CASCADE,
  revision INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  frontmatter_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'stable',
  actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  actor_ref TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  UNIQUE (concept_id, revision)
);
CREATE INDEX idx_knowledge_revisions_concept
  ON knowledge_revisions (concept_id, revision DESC);



CREATE TABLE knowledge_verifications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  concept_id TEXT NOT NULL REFERENCES knowledge_concepts (id) ON DELETE CASCADE,
  actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  actor_ref TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  revoked_at INTEGER,
  revoked_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL
);
CREATE INDEX idx_knowledge_verifications_concept
  ON knowledge_verifications (concept_id, created_at DESC);



CREATE TABLE knowledge_sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  concept_id TEXT NOT NULL REFERENCES knowledge_concepts (id) ON DELETE CASCADE,
  source_id TEXT NOT NULL DEFAULT '',
  resource TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  author_ref TEXT NOT NULL DEFAULT '',
  usage_count INTEGER,
  last_modified INTEGER,
  position INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_knowledge_sources_concept
  ON knowledge_sources (concept_id, position);



CREATE TABLE knowledge_links (
  concept_id TEXT NOT NULL REFERENCES knowledge_concepts (id) ON DELETE CASCADE,
  target_path TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'link'
    CHECK (kind IN ('link', 'source', 'decision')),
  position INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (concept_id, target_path, kind)
);
CREATE INDEX idx_knowledge_links_target ON knowledge_links (target_path, kind);



CREATE TABLE knowledge_memory_recovery (
  id TEXT PRIMARY KEY,
  agent_actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  payload_json TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  restored_at INTEGER
);
CREATE INDEX idx_knowledge_memory_recovery_agent
  ON knowledge_memory_recovery (agent_actor_id, created_at DESC);

-- +goose Down
DROP TABLE knowledge_memory_recovery;
DROP TABLE knowledge_links;
DROP TABLE knowledge_sources;
DROP TABLE knowledge_verifications;
DROP TABLE knowledge_revisions;
DROP TABLE knowledge_concepts;
