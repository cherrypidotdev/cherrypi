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
.flex.flex-wrap.items-center.gap-1(v-if="badges.length")
  template(v-for="badge in shown" :key="`${badge.pluginId}:${badge.key}`")
    a.inline-flex.max-w-full.items-stretch.overflow-hidden.rounded-md.border.text-xs(
      v-if="badge.url"
      class="hover:brightness-105"
      rel="noopener noreferrer"
      :class="taskBadgeToneClass(badgeTone(badge))"
      :href="badge.url"
      :title="badgeTitle(badge)"
      target="_blank"
      data-testid="task-badge"
    )
      span.text-muted-foreground.inline-flex.items-center.truncate(
        class="bg-muted/60 px-1.5 py-0.5"
      ) {{ badge.label }}
      span.inline-flex.items-center.truncate.font-medium(
        v-if="badge.value"
        class="px-1.5 py-0.5"
      ) {{ badge.value }}
    span.inline-flex.max-w-full.items-stretch.overflow-hidden.rounded-md.border.text-xs(
      v-else
      :class="taskBadgeToneClass(badgeTone(badge))"
      :title="badgeTitle(badge)"
      data-testid="task-badge"
    )
      span.text-muted-foreground.inline-flex.items-center.truncate(
        class="bg-muted/60 px-1.5 py-0.5"
      ) {{ badge.label }}
      span.inline-flex.items-center.truncate.font-medium(
        v-if="badge.value"
        class="px-1.5 py-0.5"
      ) {{ badge.value }}
  span.text-muted-foreground.text-xs(v-if="badges.length > shown.length")
    | +{{ badges.length - shown.length }}
</template>

<script setup lang="ts">
import { taskBadgeToneClass } from "@/lib/task";
import type { TaskBadge } from "@cherrypi/sdk";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    badges: TaskBadge[];

    limit?: number;
  }>(),
  { limit: 0 },
);

const shown = computed(() =>
  props.limit > 0 ? props.badges.slice(0, props.limit) : props.badges,
);

function badgeTone(badge: TaskBadge): string {
  return badge.value ? badge.tone : "neutral";
}

function badgeTitle(badge: TaskBadge): string {
  const owner = badge.pluginName || badge.pluginId || "Plugin";
  const value = badge.value ? `${badge.label}: ${badge.value}` : badge.label;
  return `${owner} — ${value}`;
}
</script>
