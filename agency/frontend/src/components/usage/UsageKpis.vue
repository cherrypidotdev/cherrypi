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
.grid.grid-cols-1.gap-3(
  class="sm:grid-cols-2 lg:grid-cols-5"
  aria-label="Usage summary"
)
  .bg-card.ring-border.to-card.relative.flex.flex-col.overflow-hidden.rounded-xl.bg-gradient-to-br.p-3.ring-1(
    class="from-chart-1/5"
  )
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold Total cost
      span.bg-muted.text-3xs.rounded-full.px-2.py-1 {{ rangePill }}
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ money(summary.cost) }}
    .text-2xs.text-muted-foreground.mt-1.flex.items-center.justify-between.gap-2
      span {{ summary.tasks }} tasks in scope
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-1)" :values="spark.cost")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold Tokens
      span.text-muted-foreground.text-10 in / out
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatTokens(summary.inputTokens + summary.outputTokens) }}
    .text-2xs.text-muted-foreground.mt-1
      | {{ formatTokens(summary.inputTokens) }} in · {{ formatTokens(summary.outputTokens) }} out
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-2)" :values="spark.tokens")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold Response rounds
      span.text-muted-foreground.text-10 agent turns
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatCount(summary.runs) }}
    .text-2xs.text-muted-foreground.mt-1
      | {{ summary.completed }} completed · {{ summary.tasks }} tasks
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-3)" :values="spark.runs")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold Completed tasks
      span.text-muted-foreground.text-10 incl. archived
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatCount(summary.completed) }}
    .text-2xs.text-muted-foreground.mt-1
      | {{ summary.archived }} archived · {{ summary.allTimeCompleted }} all time
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-3)" :values="spark.completed")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold Active time
      span.text-muted-foreground.text-10 agent work
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatDuration(summary.activeMs) }}
    .text-2xs.text-muted-foreground.mt-1
      template(v-if="summary.runs")
        | {{ formatDuration(summary.activeMs / summary.runs) }} avg / round
      template(v-else) no rounds
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-4)" :values="spark.active")
</template>

<script setup lang="ts">
import UsageSparkline from "@/components/usage/UsageSparkline.vue";
import type { UsageSeriesPoint } from "@/composables/useUsageAnalytics";
import { formatCount, formatDuration, formatTokens } from "@/lib/format";
import {
  usageRangePill,
  type UsageRange,
  type UsageSummary,
} from "@/lib/usage";
import { computed } from "vue";

const props = defineProps<{
  summary: UsageSummary;
  daily: UsageSeriesPoint[];
  range: UsageRange;
}>();

const rangePill = computed(() => usageRangePill(props.range));

function money(value: number): string {
  if (value >= 1) return `$${value.toFixed(2)}`;
  return `$${value.toFixed(4)}`;
}

function last(values: number[]): number[] {
  return values.slice(-18);
}

const spark = computed(() => ({
  cost: last(props.daily.map((point) => point.cost)),
  tokens: last(props.daily.map((point) => point.input + point.output)),
  runs: last(props.daily.map((point) => point.runs)),
  completed: last(props.daily.map((point) => point.completed)),
  active: last(props.daily.map((point) => point.runs)),
}));
</script>
