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






-- name: CreateIdentityProvider :one
INSERT INTO identity_providers (
  kind, slug, name, enabled, issuer, authorization_endpoint, token_endpoint,
  userinfo_endpoint, jwks_uri, client_id, client_secret, scopes,
  subject_path, email_path, name_path, created_at, updated_at
)
VALUES (
  sqlc.arg(kind), sqlc.arg(slug), sqlc.arg(name), sqlc.arg(enabled),
  sqlc.arg(issuer), sqlc.arg(authorization_endpoint), sqlc.arg(token_endpoint),
  sqlc.arg(userinfo_endpoint), sqlc.arg(jwks_uri), sqlc.arg(client_id),
  sqlc.arg(client_secret), sqlc.arg(scopes), sqlc.arg(subject_path),
  sqlc.arg(email_path), sqlc.arg(name_path), sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetIdentityProviderByID :one
SELECT * FROM identity_providers WHERE id = sqlc.arg(id);

-- name: GetIdentityProviderBySlug :one
SELECT * FROM identity_providers WHERE slug = sqlc.arg(slug);

-- name: ListIdentityProviders :many
SELECT * FROM identity_providers ORDER BY name, id;

-- name: ListEnabledIdentityProviders :many
SELECT * FROM identity_providers WHERE enabled = 1 ORDER BY name, id;

-- name: UpdateIdentityProvider :exec
UPDATE identity_providers
SET kind = sqlc.arg(kind),
    slug = sqlc.arg(slug),
    name = sqlc.arg(name),
    enabled = sqlc.arg(enabled),
    issuer = sqlc.arg(issuer),
    authorization_endpoint = sqlc.arg(authorization_endpoint),
    token_endpoint = sqlc.arg(token_endpoint),
    userinfo_endpoint = sqlc.arg(userinfo_endpoint),
    jwks_uri = sqlc.arg(jwks_uri),
    client_id = sqlc.arg(client_id),
    client_secret = sqlc.arg(client_secret),
    scopes = sqlc.arg(scopes),
    subject_path = sqlc.arg(subject_path),
    email_path = sqlc.arg(email_path),
    name_path = sqlc.arg(name_path),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DeleteIdentityProvider :exec
DELETE FROM identity_providers WHERE id = sqlc.arg(id);



-- name: CreateExternalCredential :one
INSERT INTO external_credentials (
  actor_id, provider_id, subject, email, created_at, updated_at
)
VALUES (
  sqlc.arg(actor_id), sqlc.arg(provider_id), sqlc.arg(subject), sqlc.arg(email),
  sqlc.arg(created_at), sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetExternalCredential :one
SELECT * FROM external_credentials
WHERE provider_id = sqlc.arg(provider_id) AND subject = sqlc.arg(subject);

-- name: ListExternalCredentialsByActor :many
SELECT * FROM external_credentials
WHERE actor_id = sqlc.arg(actor_id)
ORDER BY id;

-- name: ListExternalCredentialsByProvider :many
SELECT * FROM external_credentials
WHERE provider_id = sqlc.arg(provider_id)
ORDER BY id;

-- name: DeleteExternalCredential :exec
DELETE FROM external_credentials
WHERE id = sqlc.arg(id) AND actor_id = sqlc.arg(actor_id);

-- name: DeleteExternalCredentialsByProvider :exec
DELETE FROM external_credentials WHERE provider_id = sqlc.arg(provider_id);



-- name: CreateLDAPProvider :one
INSERT INTO ldap_providers (
  slug, name, enabled, host, port, encryption, insecure, base_dn, bind_dn,
  bind_password, user_filter, username_attr, email_attr, name_attr,
  created_at, updated_at
)
VALUES (
  sqlc.arg(slug), sqlc.arg(name), sqlc.arg(enabled), sqlc.arg(host),
  sqlc.arg(port), sqlc.arg(encryption), sqlc.arg(insecure), sqlc.arg(base_dn),
  sqlc.arg(bind_dn), sqlc.arg(bind_password), sqlc.arg(user_filter),
  sqlc.arg(username_attr), sqlc.arg(email_attr), sqlc.arg(name_attr),
  sqlc.arg(created_at), sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetLDAPProviderByID :one
SELECT * FROM ldap_providers WHERE id = sqlc.arg(id);

-- name: GetLDAPProviderBySlug :one
SELECT * FROM ldap_providers WHERE slug = sqlc.arg(slug);

-- name: ListLDAPProviders :many
SELECT * FROM ldap_providers ORDER BY name, id;

-- name: ListEnabledLDAPProviders :many
SELECT * FROM ldap_providers WHERE enabled = 1 ORDER BY name, id;

-- name: UpdateLDAPProvider :exec
UPDATE ldap_providers
SET slug = sqlc.arg(slug),
    name = sqlc.arg(name),
    enabled = sqlc.arg(enabled),
    host = sqlc.arg(host),
    port = sqlc.arg(port),
    encryption = sqlc.arg(encryption),
    insecure = sqlc.arg(insecure),
    base_dn = sqlc.arg(base_dn),
    bind_dn = sqlc.arg(bind_dn),
    bind_password = sqlc.arg(bind_password),
    user_filter = sqlc.arg(user_filter),
    username_attr = sqlc.arg(username_attr),
    email_attr = sqlc.arg(email_attr),
    name_attr = sqlc.arg(name_attr),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DeleteLDAPProvider :exec
DELETE FROM ldap_providers WHERE id = sqlc.arg(id);



-- name: CreateLDAPCredential :one
INSERT INTO ldap_credentials (
  actor_id, provider_id, username, created_at, updated_at
)
VALUES (
  sqlc.arg(actor_id), sqlc.arg(provider_id), sqlc.arg(username),
  sqlc.arg(created_at), sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetLDAPCredential :one
SELECT * FROM ldap_credentials
WHERE provider_id = sqlc.arg(provider_id) AND username = sqlc.arg(username);

-- name: ListLDAPCredentialsByActor :many
SELECT * FROM ldap_credentials
WHERE actor_id = sqlc.arg(actor_id)
ORDER BY id;

-- name: DeleteLDAPCredential :exec
DELETE FROM ldap_credentials
WHERE id = sqlc.arg(id) AND actor_id = sqlc.arg(actor_id);

-- name: DeleteLDAPCredentialsByProvider :exec
DELETE FROM ldap_credentials WHERE provider_id = sqlc.arg(provider_id);
