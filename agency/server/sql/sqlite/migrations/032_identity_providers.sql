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
-- Federated identity providers (phase 2): generic OIDC discovery and a
-- generic OAuth2 shim, plus the external identities linked to actors.
--
-- `identity_providers` is org-wide configuration, admin-only. `kind = oidc`
-- uses `issuer` for discovery; `kind = oauth2` is the shim and uses the
-- explicit `authorization_endpoint` / `token_endpoint` / `userinfo_endpoint`
-- with dot-separated `*_path`s into the userinfo JSON. An OIDC provider may
-- also override endpoints (discovery fills the blanks) and always verifies the
-- ID token against `jwks_uri`.
--
-- `external_credentials` keys a linked identity by `(provider, subject)`, the
-- provider's stable user id. Invite-only: a row exists only because an actor
-- linked the provider from their account (or a phase-3 invite pre-created it),
-- so login is a lookup, never an auto-provision.
CREATE TABLE identity_providers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL CHECK (kind IN ('oidc', 'oauth2')),
  slug TEXT NOT NULL,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  -- OIDC: discovery issuer. OAuth2: ignored.
  issuer TEXT NOT NULL DEFAULT '',
  -- Explicit endpoints. OIDC fills the blanks from discovery; OAuth2 requires
  -- them.
  authorization_endpoint TEXT NOT NULL DEFAULT '',
  token_endpoint TEXT NOT NULL DEFAULT '',
  userinfo_endpoint TEXT NOT NULL DEFAULT '',
  jwks_uri TEXT NOT NULL DEFAULT '',
  client_id TEXT NOT NULL DEFAULT '',
  -- Write-only secret; never returned by the API.
  client_secret TEXT NOT NULL DEFAULT '',
  -- Space-separated scopes sent on the authorization request.
  scopes TEXT NOT NULL DEFAULT '',
  -- Generic OAuth2 shim: dot-separated JSON paths into the userinfo response.
  subject_path TEXT NOT NULL DEFAULT '',
  email_path TEXT NOT NULL DEFAULT '',
  name_path TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_identity_providers_slug ON identity_providers (slug);

CREATE TABLE external_credentials (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  provider_id INTEGER NOT NULL REFERENCES identity_providers (id) ON DELETE CASCADE,
  subject TEXT NOT NULL,
  email TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_external_credentials_provider_subject
  ON external_credentials (provider_id, subject);
CREATE INDEX idx_external_credentials_actor ON external_credentials (actor_id);

-- +goose Down
DROP TABLE external_credentials;
DROP TABLE identity_providers;
