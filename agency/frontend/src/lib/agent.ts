// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { AgentStatus, AgentVersionStatus } from "@cherrypi/sdk";

export function agentStatusLabel(status: AgentStatus | undefined): string {
  return status === AgentStatus.ONLINE ? "Online" : "Offline";
}

export function agentStatusClass(status: AgentStatus | undefined): string {
  return status === AgentStatus.ONLINE
    ? "border-emerald-500/32 bg-emerald-500/12 text-emerald-600 dark:text-emerald-400"
    : "border-border bg-secondary text-secondary-foreground";
}

export function agentVersionStatusLabel(
  status: AgentVersionStatus | undefined,
): string {
  switch (status) {
    case AgentVersionStatus.CURRENT:
      return "Up to date";
    case AgentVersionStatus.OUTDATED:
      return "Binary should be updated";
    case AgentVersionStatus.INCOMPATIBLE:
      return "Binary incompatible";
    default:
      return "Unknown";
  }
}

export function agentVersionStatusClass(
  status: AgentVersionStatus | undefined,
): string {
  switch (status) {
    case AgentVersionStatus.CURRENT:
      return "border-emerald-500/32 bg-emerald-500/12 text-emerald-600 dark:text-emerald-400";
    case AgentVersionStatus.OUTDATED:
      return "border-amber-500/35 bg-amber-500/12 text-amber-600 dark:text-amber-400";
    case AgentVersionStatus.INCOMPATIBLE:
      return "border-red-500/32 bg-red-500/12 text-red-600 dark:text-red-400";
    default:
      return "border-border bg-secondary text-secondary-foreground";
  }
}

export function agentVersionNeedsAttention(
  status: AgentVersionStatus | undefined,
): boolean {
  return (
    status === AgentVersionStatus.OUTDATED ||
    status === AgentVersionStatus.INCOMPATIBLE
  );
}

export function agentVersionTitle(agent: {
  version?: string;
  protocolVersion?: string;
}): string {
  const parts: string[] = [];
  if (agent.version) {
    parts.push(`Build ${agent.version}`);
  }
  if (agent.protocolVersion) {
    parts.push(`Protocol ${agent.protocolVersion}`);
  }
  return parts.length ? parts.join(" · ") : "No version reported";
}
