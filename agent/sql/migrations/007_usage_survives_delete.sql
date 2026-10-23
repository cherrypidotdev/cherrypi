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
-- Deleting a session must keep its token/cost history, but the original
-- `usage` table cascaded the delete via its foreign key. Rebuild it without
-- the reference: usage rows now outlive their session (the dashboard still
-- aggregates them, labelled by session id).
CREATE TABLE usage_new (
  id INTEGER PRIMARY KEY,
  session_id TEXT NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cost REAL NOT NULL DEFAULT 0,
  at INTEGER NOT NULL
);

INSERT INTO usage_new (id, session_id, model, input_tokens, output_tokens, cost, at)
SELECT id, session_id, model, input_tokens, output_tokens, cost, at FROM usage;

DROP TABLE usage;

ALTER TABLE usage_new RENAME TO usage;

CREATE INDEX idx_usage_session_id ON usage (session_id);

-- +goose Down
CREATE TABLE usage_old (
  id INTEGER PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
  model TEXT NOT NULL DEFAULT '',
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cost REAL NOT NULL DEFAULT 0,
  at INTEGER NOT NULL
);

INSERT INTO usage_old (id, session_id, model, input_tokens, output_tokens, cost, at)
SELECT id, session_id, model, input_tokens, output_tokens, cost, at FROM usage;

DROP TABLE usage;

ALTER TABLE usage_old RENAME TO usage;

CREATE INDEX idx_usage_session_id ON usage (session_id);
