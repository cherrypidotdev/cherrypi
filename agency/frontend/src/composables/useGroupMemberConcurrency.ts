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

import { useAgents } from "@/composables/useAgents";
import { useFleetConcurrency } from "@/composables/useFleetConcurrency";
import type { AgentGroupMember } from "@cherrypi/sdk";
import { computed, toValue, type MaybeRefOrGetter } from "vue";

export function useGroupMemberConcurrency(
  members: MaybeRefOrGetter<AgentGroupMember[]>,
) {
  const { agents } = useAgents();
  const memberAgents = computed(() =>
    toValue(members)
      .map((member) =>
        agents.value.find((agent) => agent.id === member.agentId),
      )
      .filter((agent): agent is NonNullable<typeof agent> => !!agent),
  );
  const { stats } = useFleetConcurrency(() => memberAgents.value);

  function fleetStat(agentId: string) {
    return stats.value[agentId];
  }

  return { stats, fleetStat };
}
