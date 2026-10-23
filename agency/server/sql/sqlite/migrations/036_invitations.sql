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
-- Invitations: the only way a user joins an installation (registration is
-- closed). Four kinds share one shape: the inviter always reserves the
-- globally-unique CherryPi username, and the invitee actor is created up-front
-- with status `invited` so the namespace is claimed before acceptance.
--
--   username  a generated password is shown once; the first login must reset it
--   email     a single-use token is mailed; acceptance sets the password
--   oidc      provider-scoped; linked on the first federated login
--   ldap      provider-scoped; linked on the first LDAP bind
CREATE TABLE invitations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL CHECK (kind IN ('username', 'email', 'oidc', 'ldap')),
  -- The reserved CherryPi username (globally unique across users and agents).
  username TEXT NOT NULL,
  -- The pre-created user actor; deleting the invitation removes it.
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  -- E-mail invites: the invited address.
  email TEXT,
  -- OIDC/LDAP invites: the provider and the provider-side username to match.
  provider_id INTEGER,
  external_username TEXT,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
  -- E-mail invites: the hashed single-use acceptance token.
  token_hash BLOB,
  invited_by_actor_id INTEGER REFERENCES actors (id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  accepted_at INTEGER,
  revoked_at INTEGER
);
-- The single-use token is looked up by hash; unique so a collision cannot
-- resolve to the wrong invitation.
CREATE UNIQUE INDEX idx_invitations_token_hash ON invitations (token_hash)
  WHERE token_hash IS NOT NULL;
CREATE INDEX idx_invitations_actor ON invitations (actor_id);
CREATE INDEX idx_invitations_provider ON invitations (kind, provider_id);
CREATE INDEX idx_invitations_expires ON invitations (expires_at);

-- A username invite hands the invitee a generated password; the first login
-- must replace it before anything else.
ALTER TABLE users ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN must_change_password;
DROP TABLE invitations;
