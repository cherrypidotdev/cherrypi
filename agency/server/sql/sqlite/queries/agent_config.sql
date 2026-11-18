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

-- name: ListModelProviders :many
SELECT * FROM model_providers ORDER BY slug;

-- name: GetModelProvider :one
SELECT * FROM model_providers WHERE slug = sqlc.arg(slug);

-- name: UpsertModelProvider :exec
INSERT INTO model_providers (
  slug, name, api, base_url, api_key_sealed, headers_json, enabled,
  created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(slug) DO UPDATE SET
  name = excluded.name,
  api = excluded.api,
  base_url = excluded.base_url,
  api_key_sealed = excluded.api_key_sealed,
  headers_json = excluded.headers_json,
  enabled = excluded.enabled,
  updated_at = excluded.updated_at;

-- name: DeleteModelProvider :execrows
DELETE FROM model_providers WHERE slug = sqlc.arg(slug);

-- name: ListModels :many
SELECT * FROM models ORDER BY provider_id, model_id;

-- name: UpsertModel :exec
INSERT INTO models (
  provider_id, model_id, name, reasoning, input_json, context_window,
  max_tokens, cost_json, thinking_levels_json, enabled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider_id, model_id) DO UPDATE SET
  name = excluded.name,
  reasoning = excluded.reasoning,
  input_json = excluded.input_json,
  context_window = excluded.context_window,
  max_tokens = excluded.max_tokens,
  cost_json = excluded.cost_json,
  thinking_levels_json = excluded.thinking_levels_json,
  enabled = excluded.enabled;

-- name: DeleteModelsForProvider :exec
DELETE FROM models WHERE provider_id = sqlc.arg(provider_id);

-- name: ListModelGrants :many
SELECT * FROM model_grants ORDER BY provider_id, scope_kind, scope_id;

-- name: UpsertModelGrant :exec
INSERT INTO model_grants (provider_id, model_id, scope_kind, scope_id, created_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(provider_id, model_id, scope_kind, scope_id) DO NOTHING;

-- name: DeleteModelGrant :exec
DELETE FROM model_grants
WHERE provider_id = sqlc.arg(provider_id)
  AND model_id = sqlc.arg(model_id)
  AND scope_kind = sqlc.arg(scope_kind)
  AND scope_id = sqlc.arg(scope_id);

-- name: DeleteModelGrantsForProvider :exec
DELETE FROM model_grants WHERE provider_id = sqlc.arg(provider_id);

-- name: GetAgentConfigRevision :one
SELECT revision FROM agent_config_revision WHERE id = 1;

-- name: BumpAgentConfigRevision :exec
UPDATE agent_config_revision SET revision = sqlc.arg(revision) WHERE id = 1;

-- name: ListAgentGroupIDsForAgentActor :many
SELECT g.id
FROM agent_groups g
JOIN agent_group_members m ON m.group_id = g.id
JOIN agents a ON a.id = m.agent_id
WHERE a.actor_id = sqlc.arg(actor_id) AND g.archived_at IS NULL
ORDER BY g.id;
