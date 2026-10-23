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
-- Drop the Phase 0 placeholder table.
DROP TABLE users;

-- Basic-auth username/password pairs created through the SDK (`AuthService`).
-- Startup env credentials are transient (in-memory only) and are NOT seeded
-- here.
CREATE TABLE credentials (
  id INTEGER PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

-- One row per agent session. `status` is a SessionStatus integer enum
-- (active=1, done=2, failed=3, cancelled=4). `pending_prompt` holds the most
-- recent prompt whose run has not settled; it lets a restarted server
-- auto-continue an interrupted run. If several prompts were queued, only the
-- most recent one is recovered.
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  status INTEGER NOT NULL,
  prompt TEXT NOT NULL,
  workspace TEXT NOT NULL,
  pi_session_id TEXT NOT NULL,
  pid INTEGER,
  pending_prompt TEXT,
  error TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  ended_at INTEGER
);

CREATE INDEX idx_sessions_status ON sessions (status);

-- Append-only per-run cost accounting. Tokens/cost are the delta since the
-- previous row for the same session.
CREATE TABLE usage (
  id INTEGER PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
  model TEXT NOT NULL DEFAULT '',
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cost REAL NOT NULL DEFAULT 0,
  at INTEGER NOT NULL
);

CREATE INDEX idx_usage_session_id ON usage (session_id);

-- +goose Down
DROP TABLE usage;
DROP TABLE sessions;
DROP TABLE credentials;

CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  username TEXT NOT NULL UNIQUE
);
