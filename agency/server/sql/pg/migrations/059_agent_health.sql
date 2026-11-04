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



CREATE TABLE agent_health_samples (
  id BIGSERIAL PRIMARY KEY,
  agent_id BIGINT NOT NULL,
  sampled_at BIGINT NOT NULL,
  scope TEXT NOT NULL DEFAULT '',
  hostname TEXT NOT NULL DEFAULT '',
  os TEXT NOT NULL DEFAULT '',
  platform TEXT NOT NULL DEFAULT '',
  kernel_version TEXT NOT NULL DEFAULT '',
  kernel_arch TEXT NOT NULL DEFAULT '',
  cpu_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
  cpu_cores_json TEXT NOT NULL DEFAULT '[]',
  load1 DOUBLE PRECISION NOT NULL DEFAULT 0,
  load5 DOUBLE PRECISION NOT NULL DEFAULT 0,
  load15 DOUBLE PRECISION NOT NULL DEFAULT 0,
  mem_total BIGINT NOT NULL DEFAULT 0,
  mem_used BIGINT NOT NULL DEFAULT 0,
  mem_available BIGINT NOT NULL DEFAULT 0,
  mem_used_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
  swap_total BIGINT NOT NULL DEFAULT 0,
  swap_used BIGINT NOT NULL DEFAULT 0,
  swap_available BIGINT NOT NULL DEFAULT 0,
  swap_used_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
  disks_json TEXT NOT NULL DEFAULT '[]',
  networks_json TEXT NOT NULL DEFAULT '[]',
  gpus_json TEXT NOT NULL DEFAULT '[]',
  uptime_seconds BIGINT NOT NULL DEFAULT 0,
  boot_time BIGINT NOT NULL DEFAULT 0,
  process_count BIGINT NOT NULL DEFAULT 0,
  agent_rss_bytes BIGINT NOT NULL DEFAULT 0,
  agent_cpu_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
  go_heap_bytes BIGINT NOT NULL DEFAULT 0,
  UNIQUE (agent_id, sampled_at)
);

CREATE INDEX idx_agent_health_samples_agent_time
  ON agent_health_samples (agent_id, sampled_at);

-- +goose Down
DROP INDEX idx_agent_health_samples_agent_time;
DROP TABLE agent_health_samples;
