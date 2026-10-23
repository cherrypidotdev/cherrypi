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

-- Accounts: actors, users, e-mail credentials, sessions, single-use tokens,
-- personal access tokens and TOTP recovery codes. See docs/auth.md.

-- Actors --------------------------------------------------------------------

-- name: CreateUserActor :one
INSERT INTO actors (
  kind, username, name, description, password_hash, password_updated_at,
  created_at, updated_at
)
VALUES (
  'user',
  sqlc.arg(username),
  sqlc.arg(name),
  sqlc.arg(description),
  sqlc.narg(password_hash),
  sqlc.narg(password_updated_at),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING id;

-- name: GetActorByID :one
SELECT * FROM actors WHERE id = sqlc.arg(id);

-- name: GetActorByUsername :one
SELECT * FROM actors WHERE username = sqlc.arg(username);

-- name: GetActorByEmail :one
SELECT a.* FROM actors a
JOIN email_credentials e ON e.actor_id = a.id
WHERE e.email = sqlc.arg(email) AND e.confirmed_at IS NOT NULL;

-- name: UpdateActorPassword :exec
UPDATE actors
SET password_hash = sqlc.arg(password_hash),
    password_updated_at = sqlc.arg(password_updated_at),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: UpdateActorTOTP :exec
UPDATE actors
SET totp_secret = sqlc.arg(totp_secret),
    totp_enabled_at = sqlc.arg(totp_enabled_at),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: ClearActorTOTP :exec
UPDATE actors
SET totp_secret = NULL, totp_enabled_at = NULL, updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: UpdateActorProfile :exec
UPDATE actors
SET name = sqlc.arg(name),
    description = sqlc.arg(description),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DeleteAnyActor :exec
DELETE FROM actors WHERE id = sqlc.arg(id);

-- Users ---------------------------------------------------------------------

-- name: CreateUserRow :one
INSERT INTO users (actor_id, role, status, created_at, updated_at)
VALUES (
  sqlc.arg(actor_id), sqlc.arg(role), sqlc.arg(status),
  sqlc.arg(created_at), sqlc.arg(updated_at)
)
RETURNING *;

-- name: GetUserByActor :one
SELECT
  u.id, u.actor_id, u.role, u.status, u.last_login_at, u.must_change_password,
  u.created_at, u.updated_at,
  a.username, a.name, a.description, a.picture_attachment_id, a.password_hash,
  a.totp_enabled_at
FROM users u
JOIN actors a ON a.id = u.actor_id
WHERE u.actor_id = sqlc.arg(actor_id);

-- name: GetUserByUsername :one
SELECT
  u.id, u.actor_id, u.role, u.status, u.last_login_at, u.must_change_password,
  u.created_at, u.updated_at,
  a.username, a.name, a.description, a.picture_attachment_id, a.password_hash,
  a.totp_enabled_at
FROM users u
JOIN actors a ON a.id = u.actor_id
WHERE a.username = sqlc.arg(username);

-- name: ListUsers :many
SELECT
  u.id, u.actor_id, u.role, u.status, u.last_login_at, u.must_change_password,
  u.created_at, u.updated_at,
  a.username, a.name, a.description, a.picture_attachment_id, a.password_hash,
  a.totp_enabled_at
FROM users u
JOIN actors a ON a.id = u.actor_id
ORDER BY a.username;

-- name: SetUserMustChangePassword :exec
UPDATE users SET must_change_password = sqlc.arg(must_change_password),
  updated_at = sqlc.arg(updated_at)
WHERE actor_id = sqlc.arg(actor_id);

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountAdmins :one
SELECT COUNT(*) FROM users WHERE role = 'admin';

-- name: UpdateUserRole :exec
UPDATE users SET role = sqlc.arg(role), updated_at = sqlc.arg(updated_at)
WHERE actor_id = sqlc.arg(actor_id);

-- name: UpdateUserStatus :exec
UPDATE users SET status = sqlc.arg(status), updated_at = sqlc.arg(updated_at)
WHERE actor_id = sqlc.arg(actor_id);

-- name: TouchUserLogin :exec
UPDATE users SET last_login_at = sqlc.arg(last_login_at) WHERE actor_id = sqlc.arg(actor_id);

-- name: DeleteUserByActor :exec
DELETE FROM users WHERE actor_id = sqlc.arg(actor_id);

-- E-mail credentials --------------------------------------------------------

-- name: CreateEmailCredential :one
INSERT INTO email_credentials (actor_id, email, confirmed_at, created_at, updated_at)
VALUES (
  sqlc.arg(actor_id), sqlc.arg(email), sqlc.narg(confirmed_at),
  sqlc.arg(created_at), sqlc.arg(updated_at)
)
RETURNING *;

-- name: ListEmailCredentials :many
SELECT * FROM email_credentials
WHERE actor_id = sqlc.arg(actor_id)
ORDER BY confirmed_at IS NULL, email;

-- name: GetEmailCredentialByEmail :one
SELECT * FROM email_credentials WHERE email = sqlc.arg(email);

-- name: GetEmailCredentialByID :one
SELECT * FROM email_credentials WHERE id = sqlc.arg(id);

-- name: ConfirmEmailCredential :exec
UPDATE email_credentials
SET confirmed_at = sqlc.arg(confirmed_at), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND confirmed_at IS NULL;

-- name: DeleteEmailCredential :exec
DELETE FROM email_credentials
WHERE id = sqlc.arg(id) AND actor_id = sqlc.arg(actor_id);

-- name: CountConfirmedEmails :one
SELECT COUNT(*) FROM email_credentials
WHERE actor_id = sqlc.arg(actor_id) AND confirmed_at IS NOT NULL;

-- Sessions ------------------------------------------------------------------

-- name: CreateSession :one
INSERT INTO sessions (
  actor_id, token_hash, method, user_agent, ip, remember, totp_verified,
  created_at, last_seen_at, expires_at
)
VALUES (
  sqlc.arg(actor_id), sqlc.arg(token_hash), sqlc.arg(method),
  sqlc.arg(user_agent), sqlc.arg(ip), sqlc.arg(remember),
  sqlc.arg(totp_verified), sqlc.arg(created_at), sqlc.arg(last_seen_at),
  sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetSessionByHash :one
SELECT
  s.id, s.actor_id, s.token_hash, s.method, s.user_agent, s.ip, s.remember,
  s.totp_verified, s.created_at, s.last_seen_at, s.expires_at,
  a.kind, a.username, a.name, a.description, a.picture_attachment_id,
  a.password_hash, a.totp_secret, a.totp_enabled_at,
  u.id AS user_id, u.role, u.status, u.must_change_password
FROM sessions s
JOIN actors a ON a.id = s.actor_id
LEFT JOIN users u ON u.actor_id = s.actor_id
WHERE s.token_hash = sqlc.arg(token_hash);

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = sqlc.arg(last_seen_at), expires_at = sqlc.arg(expires_at)
WHERE id = sqlc.arg(id);

-- name: DeleteSessionByHash :exec
DELETE FROM sessions WHERE token_hash = sqlc.arg(token_hash);

-- name: DeleteSessionByID :exec
DELETE FROM sessions WHERE id = sqlc.arg(id) AND actor_id = sqlc.arg(actor_id);

-- name: DeleteSessionsByActor :exec
DELETE FROM sessions WHERE actor_id = sqlc.arg(actor_id);

-- name: DeleteOtherSessions :exec
DELETE FROM sessions
WHERE actor_id = sqlc.arg(actor_id) AND id <> sqlc.arg(id);

-- name: ListSessionsByActor :many
SELECT * FROM sessions
WHERE actor_id = sqlc.arg(actor_id)
ORDER BY last_seen_at DESC;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at < sqlc.arg(now);

-- Tokens --------------------------------------------------------------------

-- name: CreateToken :one
INSERT INTO tokens (
  token_hash, kind, actor_id, email_credential_id, created_at, expires_at
)
VALUES (
  sqlc.arg(token_hash), sqlc.arg(kind), sqlc.narg(actor_id),
  sqlc.narg(email_credential_id), sqlc.arg(created_at), sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetTokenByHash :one
SELECT * FROM tokens WHERE token_hash = sqlc.arg(token_hash);

-- name: ConsumeToken :execrows
UPDATE tokens SET consumed_at = sqlc.arg(consumed_at)
WHERE id = sqlc.arg(id) AND consumed_at IS NULL;

-- name: DeleteTokensByActorKind :exec
DELETE FROM tokens
WHERE actor_id = sqlc.arg(actor_id) AND kind = sqlc.arg(kind) AND consumed_at IS NULL;

-- name: DeleteExpiredTokens :execrows
DELETE FROM tokens WHERE expires_at < sqlc.arg(now);

-- Personal access tokens ----------------------------------------------------

-- name: CreatePersonalAccessToken :one
INSERT INTO personal_access_tokens (
  actor_id, name, token_hash, prefix, created_at, expires_at
)
VALUES (
  sqlc.arg(actor_id), sqlc.arg(name), sqlc.arg(token_hash), sqlc.arg(prefix),
  sqlc.arg(created_at), sqlc.narg(expires_at)
)
RETURNING *;

-- name: GetPersonalAccessTokenByHash :one
SELECT
  p.id, p.actor_id, p.name, p.token_hash, p.prefix, p.created_at,
  p.last_used_at, p.expires_at,
  a.kind, a.username, a.name AS actor_name, a.description,
  a.picture_attachment_id, a.password_hash, a.totp_secret, a.totp_enabled_at,
  u.id AS user_id, u.role, u.status
FROM personal_access_tokens p
JOIN actors a ON a.id = p.actor_id
LEFT JOIN users u ON u.actor_id = p.actor_id
WHERE p.token_hash = sqlc.arg(token_hash);

-- name: ListPersonalAccessTokens :many
SELECT * FROM personal_access_tokens
WHERE actor_id = sqlc.arg(actor_id)
ORDER BY created_at DESC;

-- name: TouchPersonalAccessToken :exec
UPDATE personal_access_tokens SET last_used_at = sqlc.arg(last_used_at)
WHERE id = sqlc.arg(id);

-- name: DeletePersonalAccessToken :exec
DELETE FROM personal_access_tokens
WHERE id = sqlc.arg(id) AND actor_id = sqlc.arg(actor_id);

-- name: DeleteExpiredPersonalAccessTokens :execrows
DELETE FROM personal_access_tokens
WHERE expires_at IS NOT NULL AND expires_at < sqlc.arg(now);

-- TOTP recovery codes -------------------------------------------------------

-- name: CreateRecoveryCode :one
INSERT INTO totp_recovery_codes (actor_id, code_hash, created_at)
VALUES (sqlc.arg(actor_id), sqlc.arg(code_hash), sqlc.arg(created_at))
RETURNING *;

-- name: ListUnusedRecoveryCodes :many
SELECT * FROM totp_recovery_codes
WHERE actor_id = sqlc.arg(actor_id) AND used_at IS NULL
ORDER BY id;

-- name: ConsumeRecoveryCode :execrows
UPDATE totp_recovery_codes SET used_at = sqlc.arg(used_at)
WHERE id = sqlc.arg(id) AND used_at IS NULL;

-- name: DeleteRecoveryCodes :exec
DELETE FROM totp_recovery_codes WHERE actor_id = sqlc.arg(actor_id);

-- name: CountUnusedRecoveryCodes :one
SELECT COUNT(*) FROM totp_recovery_codes
WHERE actor_id = sqlc.arg(actor_id) AND used_at IS NULL;
