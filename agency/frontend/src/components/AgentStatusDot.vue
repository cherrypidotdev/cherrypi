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
span.absolute.-right-0.-bottom-0.rounded-full.border(
  role="img"
  :class="[sizeClass, borderClass, dotClass]"
  :title="title"
  :aria-label="title"
)
</template>

<script setup lang="ts">
import { t } from "@/i18n/translate";
import { AgentStatus } from "@cherrypi/sdk";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    status?: AgentStatus;

    running?: number;
    size?: "sm" | "md";

    borderClass?: string;
  }>(),
  {
    running: 0,
    size: "sm",
    borderClass: "border-background",
  },
);

const sizeClass = computed(() => (props.size === "md" ? "size-3" : "size-2"));

const dotClass = computed(() => {
  if (props.running > 0) {
    return "bg-blue-500 animate-pulse motion-reduce:animate-none";
  }
  return props.status === AgentStatus.ONLINE
    ? "bg-emerald-500"
    : "bg-muted-foreground/40";
});

const title = computed(() => {
  if (props.running > 0) {
    return t("agentStatusDot.running", { count: props.running });
  }
  return props.status === AgentStatus.ONLINE
    ? t("agentStatusDot.online")
    : t("agentStatusDot.offline");
});
</script>

<i18n lang="yaml">
en:
  agentStatusDot:
    online: Online
    offline: Offline
    running: "{count} running"
ru:
  agentStatusDot:
    online: В сети
    offline: Не в сети
    running: "Выполняется задач: {count}"
</i18n>
