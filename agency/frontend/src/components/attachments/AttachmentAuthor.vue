<!--
CherryPi: A multi-agent-multi-human self-hosted agency
Copyright (C) 2026 CherryPi authors <cherrypi.dev>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<template lang="pug">
RouterLink.text-muted-foreground.inline-flex.max-w-full.items-center.gap-1.rounded-full.border.px-1.text-xs(
  v-if="author && author.kind !== 'unknown'"
  class="hover:bg-muted/60 hover:text-foreground py-0.5"
  :to="{ name: author.kind === 'agent' ? 'agent' : 'user', params: { username: author.username } }"
  :title="author.title"
)
  ActorAvatar(
    :src="author.pictureUrl"
    :name="author.name"
    :username="author.username"
    :size="16"
  )
  span.truncate {{ author.name }}
span.text-muted-foreground.inline-flex.max-w-full.items-center.gap-1.rounded-full.border.px-1.text-xs(
  v-else-if="author"
  class="py-0.5"
  :title="author.title"
)
  ActorAvatar(:name="author.name" :username="author.username" :size="16")
  span.truncate {{ author.name }}
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import { useAgents } from "@/composables/useAgents";
import { useUsers } from "@/composables/useUsers";
import type { Agent } from "@cherrypi/sdk";
import { computed } from "vue";

/**
 * Resolve an attachment's `createdBy` to its actor. Users arrive as their
 * username, while agents arrive as their actor id; both link to their directory
 * page and render the same avatar chip as the rest of the task surfaces.
 */
const props = defineProps<{ actor: string }>();

const { users } = useUsers();
const { agents } = useAgents();

const author = computed(() => {
  const raw = props.actor.trim();
  if (!raw) return undefined;

  // Agent attachments have historically been recorded by their agent id,
  // which can collide with a user's actor id. A numeric author is therefore
  // resolved as an agent first; user uploads carry their username.
  if (/^\d+$/.test(raw)) {
    const agent = agents.value.find(
      (item) => item.actorId === raw || item.id === raw,
    );
    if (agent) return agentAuthor(agent);
  }

  const user = users.value.find(
    (item) => item.username === raw || String(item.id) === raw,
  );
  if (user) {
    const name = user.name || user.username;
    return {
      kind: "user" as const,
      username: user.username,
      name,
      pictureUrl: user.picture?.url ?? "",
      title: `User ${name}`,
    };
  }

  const agent = agents.value.find(
    (item) => item.actorId === raw || item.username === raw || item.id === raw,
  );
  if (agent) return agentAuthor(agent);

  return {
    kind: "unknown" as const,
    username: raw,
    name: raw,
    pictureUrl: "",
    title: `Actor ${raw}`,
  };
});

function agentAuthor(agent: Agent) {
  const name = agent.displayName || agent.username;
  return {
    kind: "agent" as const,
    username: agent.username,
    name,
    pictureUrl: agent.picture?.url ?? "",
    title: `Agent ${name}`,
  };
}
</script>
