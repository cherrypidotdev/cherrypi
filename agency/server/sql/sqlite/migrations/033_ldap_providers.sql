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
-- LDAP providers (phase 2). Multiple providers per org; the login form lists
-- them and the login request names one by slug.
--
-- `ldap_providers` is org-wide configuration, admin-only. `encryption` is
-- `none`, `starttls` (upgrade a plain connection) or `ldaps` (TLS from the
-- start); `insecure` skips certificate verification for self-signed test/dev
-- servers only.
--
-- Bind resolution: when `bind_dn` contains `{username}` the server binds
-- directly as the substituted DN (glauth-style uid bind); otherwise it binds
-- with `bind_dn`/`bind_password`, searches `base_dn` for `user_filter` and
-- re-binds as the user's DN to verify the password.
--
-- `ldap_credentials` maps a linked LDAP username to an actor. Invite-only: a
-- row exists only because the actor linked the provider from their account (or
-- a phase-3 invite pre-created it), so login is a lookup, never an
-- auto-provision.
CREATE TABLE ldap_providers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  host TEXT NOT NULL,
  port INTEGER NOT NULL DEFAULT 389,
  encryption TEXT NOT NULL DEFAULT 'none'
    CHECK (encryption IN ('none', 'starttls', 'ldaps')),
  insecure INTEGER NOT NULL DEFAULT 0 CHECK (insecure IN (0, 1)),
  base_dn TEXT NOT NULL DEFAULT '',
  bind_dn TEXT NOT NULL DEFAULT '',
  -- Write-only secret; never returned by the API.
  bind_password TEXT NOT NULL DEFAULT '',
  user_filter TEXT NOT NULL DEFAULT '(uid={username})',
  username_attr TEXT NOT NULL DEFAULT 'uid',
  email_attr TEXT NOT NULL DEFAULT 'mail',
  name_attr TEXT NOT NULL DEFAULT 'cn',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_ldap_providers_slug ON ldap_providers (slug);

CREATE TABLE ldap_credentials (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NOT NULL REFERENCES actors (id) ON DELETE CASCADE,
  provider_id INTEGER NOT NULL REFERENCES ldap_providers (id) ON DELETE CASCADE,
  username TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_ldap_credentials_provider_username
  ON ldap_credentials (provider_id, username);
CREATE INDEX idx_ldap_credentials_actor ON ldap_credentials (actor_id);

-- +goose Down
DROP TABLE ldap_credentials;
DROP TABLE ldap_providers;
