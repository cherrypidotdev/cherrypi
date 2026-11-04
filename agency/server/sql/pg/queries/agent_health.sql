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





-- name: InsertAgentHealthSample :exec
INSERT INTO agent_health_samples (
  agent_id, sampled_at, scope, hostname, os, platform, kernel_version,
  kernel_arch, cpu_percent, cpu_cores_json, load1, load5, load15,
  mem_total, mem_used, mem_available, mem_used_percent,
  swap_total, swap_used, swap_available, swap_used_percent,
  disks_json, networks_json, gpus_json, uptime_seconds, boot_time,
  process_count, agent_rss_bytes, agent_cpu_percent, go_heap_bytes
) VALUES (
  sqlc.arg(agent_id), sqlc.arg(sampled_at), sqlc.arg(scope),
  sqlc.arg(hostname), sqlc.arg(os), sqlc.arg(platform),
  sqlc.arg(kernel_version), sqlc.arg(kernel_arch), sqlc.arg(cpu_percent),
  sqlc.arg(cpu_cores_json), sqlc.arg(load1), sqlc.arg(load5),
  sqlc.arg(load15), sqlc.arg(mem_total), sqlc.arg(mem_used),
  sqlc.arg(mem_available), sqlc.arg(mem_used_percent), sqlc.arg(swap_total),
  sqlc.arg(swap_used), sqlc.arg(swap_available), sqlc.arg(swap_used_percent),
  sqlc.arg(disks_json), sqlc.arg(networks_json), sqlc.arg(gpus_json),
  sqlc.arg(uptime_seconds), sqlc.arg(boot_time), sqlc.arg(process_count),
  sqlc.arg(agent_rss_bytes), sqlc.arg(agent_cpu_percent),
  sqlc.arg(go_heap_bytes)
)
ON CONFLICT (agent_id, sampled_at) DO NOTHING;

-- name: PruneAgentHealthSamples :exec
DELETE FROM agent_health_samples
WHERE agent_id = sqlc.arg(agent_id)
  AND sampled_at < sqlc.arg(before);

-- name: GetLatestAgentHealthSample :one
SELECT * FROM agent_health_samples
WHERE agent_id = sqlc.arg(agent_id)
ORDER BY sampled_at DESC
LIMIT 1;

-- name: ListAgentHealthSamples :many

SELECT * FROM agent_health_samples
WHERE agent_id = sqlc.arg(agent_id)
  AND sampled_at >= sqlc.arg(since)
  AND sampled_at <= sqlc.arg(until)
ORDER BY sampled_at ASC;

-- name: DeleteAgentHealthSamples :exec
DELETE FROM agent_health_samples WHERE agent_id = sqlc.arg(agent_id);
