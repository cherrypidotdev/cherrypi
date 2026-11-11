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
.flex.flex-col.gap-3
  p.text-destructive.text-xs(v-if="error") {{ error }}
  .grid.grid-cols-1.gap-3(
    v-else-if="loading && !summary"
    class="sm:grid-cols-2 lg:grid-cols-4"
    :aria-label="$t('concurrencyKpis.title')"
  )
    .bg-card.ring-border.rounded-xl.p-3.ring-1(v-for="index in 4" :key="index")
      Skeleton.h-3.w-20
      Skeleton.mt-2.h-7.w-16
      Skeleton.mt-3.w-full(class="h-1.5")
  .grid.grid-cols-1.gap-3(
    v-else-if="summary"
    class="sm:grid-cols-2 lg:grid-cols-4"
    :aria-label="$t('concurrencyKpis.title')"
  )
    .bg-card.ring-border.to-card.relative.flex.flex-col.overflow-hidden.rounded-xl.bg-gradient-to-br.p-3.ring-1(
      class="from-chart-1/5"
      :title="summary.kind === 'project' ? $t('concurrencyKpis.projectCapsHint') : undefined"
    )
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-2xs.font-semibold {{ summary.heldLabel }}
        span.text-muted-foreground.text-3xs {{ summary.heldHint }}
      .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums
        | {{ summary.held }}
        span.text-muted-foreground.text-sm /{{ summary.cap }}
      .mt-auto.pt-2
        ConcurrencyTrack(
          :label="summary.heldLabel"
          size="sm"
          :used="summary.held"
          :cap="summary.cap"
          :show-count="false"
          :show-waiting="false"
        )
      .text-2xs.text-muted-foreground.mt-1 {{ summary.heldDetail }}

    .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-2xs.font-semibold {{ $t("concurrencyKpis.waiting") }}
        span.text-muted-foreground.text-3xs {{ summary.waitingHint }}
      .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums(
        :class="summary.waiting ? 'text-amber-600 dark:text-amber-400' : ''"
      ) {{ summary.waiting }}
      .text-2xs.text-muted-foreground.mt-1 {{ summary.waitingDetail }}

    .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-2xs.font-semibold {{ summary.thirdLabel }}
        span.text-muted-foreground.text-3xs {{ summary.thirdHint }}
      .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ summary.thirdValue }}
      .text-2xs.text-muted-foreground.mt-1 {{ summary.thirdDetail }}

    .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-2xs.font-semibold {{ $t("concurrencyKpis.capsFull") }}
        span.text-muted-foreground.text-3xs {{ summary.capsHint }}
      .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums(
        :class="summary.caps ? 'text-amber-600 dark:text-amber-400' : ''"
      ) {{ summary.caps }}
      .text-2xs.text-muted-foreground.mt-1 {{ summary.capsDetail }}
</template>

<script setup lang="ts">
import ConcurrencyTrack from "@/components/concurrency/ConcurrencyTrack.vue";
import { Skeleton } from "@/components/ui/skeleton";
import {
  useAgentConcurrency,
  useProjectConcurrency,
} from "@/composables/useConcurrency";
import {
  agentConcurrencyKpis,
  projectConcurrencyKpis,
} from "@/lib/concurrencyKpis";
import type { Project } from "@cherrypi/sdk";
import { computed } from "vue";

const props = defineProps<{
  context: "agent" | "project";
  agentId?: string;
  project?: Project;
}>();

const agentScope = useAgentConcurrency(() =>
  props.context === "agent" ? (props.agentId ?? "") : "",
);
const projectScope = useProjectConcurrency(() =>
  props.context === "project" ? props.project : undefined,
);

const board = computed(() =>
  props.context === "agent" ? agentScope.board.value : projectScope.board.value,
);
const loading = computed(() =>
  props.context === "agent"
    ? agentScope.loading.value
    : projectScope.loading.value,
);
const error = computed(() =>
  props.context === "agent" ? agentScope.error.value : projectScope.error.value,
);

const summary = computed(() => {
  const value = board.value;
  if (!value) return undefined;
  return value.kind === "agent"
    ? agentConcurrencyKpis(value)
    : projectConcurrencyKpis(value);
});
</script>

<i18n lang="yaml">
en:
  concurrencyKpis:
    title: "Concurrency summary"
    projectCapsHint: "Sum of the member agents project caps"
    slotsHeld: "Slots held"
    free: "Free"
    memberAgents: "Member agents"
    thisAgent: "this agent"
    ofAgentCap: "{percent}% of the agent cap"
    pinned: "pinned"
    pinnedToAgent: "pinned to this agent"
    nothingPinned: "nothing pinned"
    slots: "slots"
    availableNow: "available now"
    noFreeSlots: "no free slots"
    dimensions: "dimensions"
    noCapFull: "no cap is full"
    fullProjects: "{count} project | {count} projects"
    fullTags: "{count} tag | {count} tags"
    fullProjectTags: "{count} project × tag"
    projectSlots: "project slots"
    sumOfCaps: "sum of {count} project cap | sum of {count} project caps"
    queued: "queued"
    pinnedAndAuto: "{pinned} pinned · {auto} auto"
    pinnedCount: "{count} pinned"
    nothingWaiting: "nothing waiting"
    serving: "serving"
    onlineCount: "{count} online"
    agentsHint: "agents"
    agentsFull: "{full} of {total} agents have a full cap"
    waiting: "Waiting"
    capsFull: "Caps full"
ru:
  concurrencyKpis:
    title: "Сводка параллельности"
    projectCapsHint: "Сумма лимитов проекта у агентов-участников"
    slotsHeld: "Занято слотов"
    free: "Свободно"
    memberAgents: "Агенты-участники"
    thisAgent: "этот агент"
    ofAgentCap: "{percent}% лимита агента"
    pinned: "закреплено"
    pinnedToAgent: "закреплено за этим агентом"
    nothingPinned: "ничего не закреплено"
    slots: "слотов"
    availableNow: "доступно сейчас"
    noFreeSlots: "свободных слотов нет"
    dimensions: "измерений"
    noCapFull: "ни один лимит не исчерпан"
    fullProjects: "{count} проект | {count} проекта | {count} проектов"
    fullTags: "{count} тег | {count} тега | {count} тегов"
    fullProjectTags: "{count} проект × тег"
    projectSlots: "слоты проекта"
    sumOfCaps: "сумма {count} лимита проектов | сумма {count} лимитов проектов"
    queued: "в очереди"
    pinnedAndAuto: "закреплено {pinned} · авто {auto}"
    pinnedCount: "закреплено {count}"
    nothingWaiting: "ничего не ожидает"
    serving: "обслуживают"
    onlineCount: "в сети: {count}"
    agentsHint: "агентов"
    agentsFull: "у {full} из {total} агентов лимит исчерпан"
    waiting: "В очереди"
    capsFull: "Лимиты исчерпаны"
</i18n>
