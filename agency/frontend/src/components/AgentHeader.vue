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
header.flex.h-14.shrink-0.items-center.gap-3.border-b.px-6
  RouterLink.-ml-2.flex.min-w-0.items-center.gap-2.rounded-md.px-2.py-1(
    class="hover:bg-muted/50"
    :to="{ name: 'agent', params: { username: currentAgentUsername } }"
  )
    span.relative.shrink-0
      img.block.size-6.rounded-full.object-cover(
        v-if="item?.picture"
        :src="item.picture.url"
        :alt="name"
      )
      span.block.size-6(v-else :innerHTML="avatar")
      span.border-background.absolute.-right-0.-bottom-0.size-2.rounded-full.border(
        :class="online ? 'bg-emerald-500' : 'bg-muted-foreground/40'"
        :title="online ? 'Online' : 'Offline'"
      )
    .flex.min-w-0.flex-col
      span.truncate.text-sm.font-semibold {{ name }}
      span.text-muted-foreground.truncate.font-mono.text-xs {{ item?.username ?? currentAgentUsername }}
  .ml-auto.flex.items-center.gap-1
    template(v-for="link in nav" :key="link.name")
      Button.text-muted-foreground(
        v-if="link.disabled"
        variant="ghost"
        size="sm"
        disabled
        :title="link.title"
      )
        component.size-4(:is="link.icon")
        span {{ link.label }}
      Button(
        v-else
        as-child
        variant="ghost"
        size="sm"
        :class="isActive(link.name) ? 'bg-muted text-foreground' : 'text-muted-foreground'"
      )
        RouterLink(
          :to="{ name: link.name, params: { username: currentAgentUsername } }"
        )
          component.size-4(:is="link.icon")
          span {{ link.label }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { useAgentAccess } from "@/composables/useAgentAccess";
import { useAgents } from "@/composables/useAgents";
import { useTerminals } from "@/composables/useTerminals";
import { currentAgentId, currentAgentUsername } from "@/lib/sdk";
import { AgentStatus } from "@cherrypi/sdk";
import { MonitorIcon, SettingsIcon, TerminalIcon } from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed } from "vue";
import { RouterLink, useRoute } from "vue-router";

const { agents } = useAgents();
const { canManageAgent } = useAgentAccess();
const { stats } = useTerminals();
const route = useRoute();

const item = computed(() =>
  agents.value.find((a) => a.id === currentAgentId.value),
);
const name = computed(
  () => item.value?.displayName || item.value?.username || "Agent",
);
const online = computed(() => item.value?.status === AgentStatus.ONLINE);
const avatar = computed(() => toSvg(currentAgentId.value || "agent", 24));

function isActive(page: string): boolean {
  return route.name === page;
}

const canUseTerminals = computed(() => canManageAgent(currentAgentId.value));

const nav = computed(() => [
  {
    name: "root-terminals",
    label: `Root terminals (${stats.value.root})`,
    icon: TerminalIcon,
    disabled: !canUseTerminals.value,
    title: canUseTerminals.value
      ? "Open root terminals"
      : "Only agent owners can open terminals",
  },
  { name: "desktop", label: "RDP", icon: MonitorIcon },
  { name: "agent-settings", label: "Settings", icon: SettingsIcon },
]);
</script>
