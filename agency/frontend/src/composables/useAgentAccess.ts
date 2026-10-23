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
import { useDirectory } from "@/composables/useDirectory";
import { currentAccount } from "@/lib/account";
import { computed } from "vue";

/**
 * Resolves whether the signed-in actor may manage an agent.
 *
 * Ownership is loaded once by the shell into `useDirectory`; administrators
 * can manage every agent. Agents the actor only shares a team with are
 * visible but not manageable.
 */
export function useAgentAccess() {
  const { agents } = useAgents();
  const { ownedAgentActorIds } = useDirectory();

  const isAdmin = computed(() => currentAccount.value?.is_admin === true);
  const owned = computed(() => new Set(ownedAgentActorIds.value));

  function isOwner(actorId: string | undefined): boolean {
    if (!actorId) return false;
    return isAdmin.value || owned.value.has(actorId);
  }

  /** Accepts an agent id or an agent actor id. */
  function canManageAgent(agentIdOrActorId: string | undefined): boolean {
    if (!agentIdOrActorId) return false;
    const agent = agents.value.find(
      (item) =>
        item.id === agentIdOrActorId || item.actorId === agentIdOrActorId,
    );
    return isOwner(agent?.actorId ?? agentIdOrActorId);
  }

  return { isAdmin, isOwner, canManageAgent };
}
