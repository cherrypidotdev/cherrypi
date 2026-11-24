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





-- name: ListMcpServers :many
SELECT * FROM mcp_servers ORDER BY name;

-- name: GetMcpServer :one
SELECT * FROM mcp_servers WHERE id = sqlc.arg(id);

-- name: GetMcpServerByName :one
SELECT * FROM mcp_servers WHERE name = sqlc.arg(name);

-- name: CreateMcpServer :one
INSERT INTO mcp_servers (
  name, description, owner_kind, owner_id, url, headers_json,
  connect_timeout_ms, request_timeout_ms, tools_json, direct_tools, enabled,
  created_by_actor_id, created_at, updated_at
) VALUES (
  sqlc.arg(name),
  sqlc.arg(description),
  sqlc.arg(owner_kind),
  sqlc.arg(owner_id),
  sqlc.arg(url),
  sqlc.arg(headers_json),
  sqlc.arg(connect_timeout_ms),
  sqlc.arg(request_timeout_ms),
  sqlc.arg(tools_json),
  sqlc.arg(direct_tools),
  sqlc.arg(enabled),
  NULLIF(sqlc.arg(created_by_actor_id), 0),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING *;

-- name: UpdateMcpServer :one
UPDATE mcp_servers SET
  name = sqlc.arg(name),
  description = sqlc.arg(description),
  owner_kind = sqlc.arg(owner_kind),
  owner_id = sqlc.arg(owner_id),
  url = sqlc.arg(url),
  headers_json = sqlc.arg(headers_json),
  connect_timeout_ms = sqlc.arg(connect_timeout_ms),
  request_timeout_ms = sqlc.arg(request_timeout_ms),
  tools_json = sqlc.arg(tools_json),
  direct_tools = sqlc.arg(direct_tools),
  enabled = sqlc.arg(enabled),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteMcpServer :execrows
DELETE FROM mcp_servers WHERE id = sqlc.arg(id);

-- name: ListSkillEntries :many
SELECT * FROM skills ORDER BY name;

-- name: GetSkillEntry :one
SELECT * FROM skills WHERE id = sqlc.arg(id);

-- name: CreateSkillEntry :one
INSERT INTO skills (
  name, description, body, files_json, disable_model_invocation, owner_kind,
  owner_id, enabled, created_by_actor_id, created_at, updated_at
) VALUES (
  sqlc.arg(name),
  sqlc.arg(description),
  sqlc.arg(body),
  sqlc.arg(files_json),
  sqlc.arg(disable_model_invocation),
  sqlc.arg(owner_kind),
  sqlc.arg(owner_id),
  sqlc.arg(enabled),
  NULLIF(sqlc.arg(created_by_actor_id), 0),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING *;

-- name: UpdateSkillEntry :one
UPDATE skills SET
  name = sqlc.arg(name),
  description = sqlc.arg(description),
  body = sqlc.arg(body),
  files_json = sqlc.arg(files_json),
  disable_model_invocation = sqlc.arg(disable_model_invocation),
  owner_kind = sqlc.arg(owner_kind),
  owner_id = sqlc.arg(owner_id),
  enabled = sqlc.arg(enabled),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteSkillEntry :execrows
DELETE FROM skills WHERE id = sqlc.arg(id);

-- name: ListMcpInstalls :many
SELECT * FROM mcp_installs ORDER BY entry_id, scope_kind, scope_id;

-- name: UpsertMcpInstall :exec
INSERT INTO mcp_installs (
  entry_id, scope_kind, scope_id, enabled, created_by_actor_id, created_at
) VALUES (
  sqlc.arg(entry_id),
  sqlc.arg(scope_kind),
  sqlc.arg(scope_id),
  sqlc.arg(enabled),
  NULLIF(sqlc.arg(created_by_actor_id), 0),
  sqlc.arg(created_at)
)
ON CONFLICT(entry_id, scope_kind, scope_id) DO UPDATE SET
  enabled = excluded.enabled;

-- name: DeleteMcpInstall :execrows
DELETE FROM mcp_installs
WHERE entry_id = sqlc.arg(entry_id)
  AND scope_kind = sqlc.arg(scope_kind)
  AND scope_id = sqlc.arg(scope_id);

-- name: ListSkillInstalls :many
SELECT * FROM skill_installs ORDER BY entry_id, scope_kind, scope_id;

-- name: UpsertSkillInstall :exec
INSERT INTO skill_installs (
  entry_id, scope_kind, scope_id, enabled, created_by_actor_id, created_at
) VALUES (
  sqlc.arg(entry_id),
  sqlc.arg(scope_kind),
  sqlc.arg(scope_id),
  sqlc.arg(enabled),
  NULLIF(sqlc.arg(created_by_actor_id), 0),
  sqlc.arg(created_at)
)
ON CONFLICT(entry_id, scope_kind, scope_id) DO UPDATE SET
  enabled = excluded.enabled;

-- name: DeleteSkillInstall :execrows
DELETE FROM skill_installs
WHERE entry_id = sqlc.arg(entry_id)
  AND scope_kind = sqlc.arg(scope_kind)
  AND scope_id = sqlc.arg(scope_id);

-- name: GetTaskIntegrationReport :one
SELECT * FROM task_integration_reports WHERE task_id = sqlc.arg(task_id);

-- name: UpsertTaskIntegrationReport :exec
INSERT INTO task_integration_reports (
  task_id, payload_json, config_revision, reported_at
) VALUES (
  sqlc.arg(task_id),
  sqlc.arg(payload_json),
  sqlc.arg(config_revision),
  sqlc.arg(reported_at)
)
ON CONFLICT(task_id) DO UPDATE SET
  payload_json = excluded.payload_json,
  config_revision = excluded.config_revision,
  reported_at = excluded.reported_at;




-- name: ListGitCredentials :many
SELECT * FROM git_credentials ORDER BY name;

-- name: GetGitCredential :one
SELECT * FROM git_credentials WHERE id = sqlc.arg(id);

-- name: CountGitCredentialSources :one
SELECT COUNT(*) FROM skill_sources WHERE credential_id = sqlc.arg(credential_id);

-- name: CreateGitCredential :one
INSERT INTO git_credentials (
  name, kind, host, port, username, secret_sealed, known_hosts, public_key,
  fingerprint, created_by_actor_id, created_at, updated_at
) VALUES (
  sqlc.arg(name),
  sqlc.arg(kind),
  sqlc.arg(host),
  sqlc.arg(port),
  sqlc.arg(username),
  sqlc.arg(secret_sealed),
  sqlc.arg(known_hosts),
  sqlc.arg(public_key),
  sqlc.arg(fingerprint),
  NULLIF(sqlc.arg(created_by_actor_id), 0),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING *;

-- name: UpdateGitCredential :one
UPDATE git_credentials SET
  name = sqlc.arg(name),
  kind = sqlc.arg(kind),
  host = sqlc.arg(host),
  port = sqlc.arg(port),
  username = sqlc.arg(username),
  secret_sealed = sqlc.arg(secret_sealed),
  known_hosts = sqlc.arg(known_hosts),
  public_key = sqlc.arg(public_key),
  fingerprint = sqlc.arg(fingerprint),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteGitCredential :execrows
DELETE FROM git_credentials WHERE id = sqlc.arg(id);

-- name: ListSkillSources :many
SELECT * FROM skill_sources ORDER BY slug;

-- name: GetSkillSource :one
SELECT * FROM skill_sources WHERE id = sqlc.arg(id);

-- name: CreateSkillSource :one
INSERT INTO skill_sources (
  slug, name, description, owner_kind, owner_id, repo_url, ref, path_glob,
  credential_id, update_mode, poll_interval_s, paused, created_by_actor_id,
  created_at, updated_at
) VALUES (
  sqlc.arg(slug),
  sqlc.arg(name),
  sqlc.arg(description),
  sqlc.arg(owner_kind),
  sqlc.arg(owner_id),
  sqlc.arg(repo_url),
  sqlc.arg(ref),
  sqlc.arg(path_glob),
  sqlc.arg(credential_id),
  sqlc.arg(update_mode),
  sqlc.arg(poll_interval_s),
  sqlc.arg(paused),
  NULLIF(sqlc.arg(created_by_actor_id), 0),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING *;

-- name: UpdateSkillSource :one
UPDATE skill_sources SET
  slug = sqlc.arg(slug),
  name = sqlc.arg(name),
  description = sqlc.arg(description),
  owner_kind = sqlc.arg(owner_kind),
  owner_id = sqlc.arg(owner_id),
  repo_url = sqlc.arg(repo_url),
  ref = sqlc.arg(ref),
  path_glob = sqlc.arg(path_glob),
  credential_id = sqlc.arg(credential_id),
  update_mode = sqlc.arg(update_mode),
  poll_interval_s = sqlc.arg(poll_interval_s),
  paused = sqlc.arg(paused),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteSkillSource :execrows
DELETE FROM skill_sources WHERE id = sqlc.arg(id);

-- name: SetSkillSourceCheck :exec
UPDATE skill_sources SET
  checked_at = sqlc.arg(checked_at),
  sync_state = sqlc.arg(sync_state),
  sync_error = sqlc.arg(sync_error),
  pending_commit = sqlc.arg(pending_commit),
  pending_json = sqlc.arg(pending_json),
  problems_json = sqlc.arg(problems_json)
WHERE id = sqlc.arg(id);

-- name: SetSkillSourceApplied :exec
UPDATE skill_sources SET
  synced_commit = sqlc.arg(synced_commit),
  synced_at = sqlc.arg(synced_at),
  checked_at = sqlc.arg(checked_at),
  sync_state = 'ok',
  sync_error = sqlc.arg(sync_error),
  pending_commit = '',
  pending_json = '',
  problems_json = sqlc.arg(problems_json)
WHERE id = sqlc.arg(id);

-- name: ListSkillSourceInstalls :many
SELECT * FROM skill_source_installs ORDER BY source_id, scope_kind, scope_id;

-- name: UpsertSkillSourceInstall :exec
INSERT INTO skill_source_installs (
  source_id, scope_kind, scope_id, enabled, created_by_actor_id, created_at
) VALUES (
  sqlc.arg(source_id),
  sqlc.arg(scope_kind),
  sqlc.arg(scope_id),
  sqlc.arg(enabled),
  NULLIF(sqlc.arg(created_by_actor_id), 0),
  sqlc.arg(created_at)
)
ON CONFLICT(source_id, scope_kind, scope_id) DO UPDATE SET
  enabled = excluded.enabled;

-- name: DeleteSkillSourceInstall :execrows
DELETE FROM skill_source_installs
WHERE source_id = sqlc.arg(source_id)
  AND scope_kind = sqlc.arg(scope_kind)
  AND scope_id = sqlc.arg(scope_id);

-- name: ListSkillsBySource :many
SELECT * FROM skills WHERE source_id = sqlc.arg(source_id) ORDER BY name;

-- name: GetSkillEntryByName :one
SELECT * FROM skills WHERE name = sqlc.arg(name);

-- name: CreateSourcedSkill :one
INSERT INTO skills (
  name, description, body, files_json, disable_model_invocation, owner_kind,
  owner_id, enabled, source_id, source_path, source_commit, source_missing,
  created_by_actor_id, created_at, updated_at
) VALUES (
  sqlc.arg(name),
  sqlc.arg(description),
  sqlc.arg(body),
  sqlc.arg(files_json),
  sqlc.arg(disable_model_invocation),
  sqlc.arg(owner_kind),
  sqlc.arg(owner_id),
  sqlc.arg(enabled),
  sqlc.arg(source_id),
  sqlc.arg(source_path),
  sqlc.arg(source_commit),
  0,
  NULLIF(sqlc.arg(created_by_actor_id), 0),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
RETURNING *;

-- name: UpdateSourcedSkill :one
UPDATE skills SET
  name = sqlc.arg(name),
  description = sqlc.arg(description),
  body = sqlc.arg(body),
  files_json = sqlc.arg(files_json),
  disable_model_invocation = sqlc.arg(disable_model_invocation),
  source_path = sqlc.arg(source_path),
  source_commit = sqlc.arg(source_commit),
  source_missing = 0,
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: MarkSourcedSkillMissing :exec
UPDATE skills SET
  source_commit = sqlc.arg(source_commit),
  source_missing = 1,
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DetachSourcedSkill :exec
UPDATE skills SET
  source_id = NULL,
  source_path = '',
  source_commit = '',
  source_missing = 0,
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: ReownSourcedSkills :exec
UPDATE skills SET owner_kind = sqlc.arg(owner_kind), owner_id = sqlc.arg(owner_id)
WHERE source_id = sqlc.arg(source_id);
