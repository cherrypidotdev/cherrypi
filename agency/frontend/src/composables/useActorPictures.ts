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
import { useUsers } from "@/composables/useUsers";
import type { MessageSender } from "@/lib/console";
import type { Attachment } from "@cherrypi/sdk";
import { computed } from "vue";

export function useActorPictures() {
  const { users, loadUsers } = useUsers();
  const { agents, loadAgents } = useAgents();

  const byActorId = computed(() => {
    const map = new Map<string, Attachment>();
    for (const user of users.value) {
      if (user.picture) map.set(String(user.id), user.picture);
    }
    for (const agent of agents.value) {
      if (agent.picture) map.set(agent.actorId, agent.picture);
    }
    return map;
  });

  const byUsername = computed(() => {
    const map = new Map<string, Attachment>();
    for (const user of users.value) {
      if (user.picture) map.set(user.username, user.picture);
    }
    for (const agent of agents.value) {
      if (agent.picture) map.set(agent.username, agent.picture);
    }
    return map;
  });

  function pictureFor(
    sender: MessageSender | undefined,
  ): Attachment | undefined {
    if (!sender || sender.kind === "system") return undefined;
    if (sender.id) {
      const byId = byActorId.value.get(sender.id);
      if (byId) return byId;
    }
    return byUsername.value.get(sender.username);
  }

  async function loadActorPictures(): Promise<void> {
    await Promise.all([loadUsers(), loadAgents()]);
  }

  return { actorPictures: byActorId, loadActorPictures, pictureFor };
}
