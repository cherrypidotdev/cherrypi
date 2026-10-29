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
    aria-label="Concurrency summary"
  )
    .bg-card.ring-border.rounded-xl.p-3.ring-1(v-for="index in 4" :key="index")
      Skeleton.h-3.w-20
      Skeleton.mt-2.h-7.w-16
      Skeleton.mt-3.w-full(class="h-1.5")
  .grid.grid-cols-1.gap-3(
    v-else-if="summary"
    class="sm:grid-cols-2 lg:grid-cols-4"
    aria-label="Concurrency summary"
  )
    .bg-card.ring-border.to-card.relative.flex.flex-col.overflow-hidden.rounded-xl.bg-gradient-to-br.p-3.ring-1(
      class="from-chart-1/5"
      :title="summary.kind === 'project' ? 'Sum of the member agents project caps' : undefined"
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
        span.text-muted-foreground.text-2xs.font-semibold Waiting
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
        span.text-muted-foreground.text-2xs.font-semibold Caps full
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
  if (value.kind === "agent") {
    const fullCapDetail = [
      value.fullProjectCount
        ? `${value.fullProjectCount} project${value.fullProjectCount === 1 ? "" : "s"}`
        : "",
      value.fullTagCount
        ? `${value.fullTagCount} tag${value.fullTagCount === 1 ? "" : "s"}`
        : "",
      value.fullProjectTagCount
        ? `${value.fullProjectTagCount} project × tag`
        : "",
    ]
      .filter(Boolean)
      .join(" · ");
    return {
      kind: "agent" as const,
      heldLabel: "Slots held",
      heldHint: "this agent",
      held: value.overall.used,
      cap: value.overall.cap,
      heldDetail: `${
        value.overall.cap > 0
          ? Math.round((value.overall.used / value.overall.cap) * 100)
          : 0
      }% of the agent cap`,
      waiting: value.waiting.length,
      waitingHint: "pinned",
      waitingDetail: value.waiting.length
        ? "pinned to this agent"
        : "nothing pinned",
      thirdLabel: "Free",
      thirdHint: "slots",
      thirdValue: value.free,
      thirdDetail: value.free ? "available now" : "no free slots",
      caps: value.fullCapCount,
      capsHint: "dimensions",
      capsDetail: fullCapDetail || "no cap is full",
    };
  }

  const pinned = value.waiting.filter((item) => item.agent).length;
  const auto = value.waiting.length - pinned;
  return {
    kind: "project" as const,
    heldLabel: "Held here",
    heldHint: "project slots",
    held: value.held,
    cap: value.capSum,
    heldDetail: `sum of ${value.memberCount} project cap${value.memberCount === 1 ? "" : "s"}`,
    waiting: value.waiting.length,
    waitingHint: "queued",
    waitingDetail: auto
      ? `${pinned} pinned · ${auto} auto`
      : pinned
        ? `${pinned} pinned`
        : "nothing waiting",
    thirdLabel: "Member agents",
    thirdHint: "serving",
    thirdValue: value.memberCount,
    thirdDetail: `${value.onlineCount} online`,
    caps: value.fullMemberCount,
    capsHint: "agents",
    capsDetail: `${value.fullMemberCount} of ${value.memberCount} agents have a full cap`,
  };
});
</script>
