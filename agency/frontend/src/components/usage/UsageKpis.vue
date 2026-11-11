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
  class="sm:grid-cols-2 lg:grid-cols-4"
  :aria-label="$t('usage.kpis.aria')"
)
  .bg-card.ring-border.to-card.relative.flex.flex-col.overflow-hidden.rounded-xl.bg-gradient-to-br.p-3.ring-1(
    class="from-chart-1/5"
  )
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold {{ $t("taskDetails.usage.totalCost") }}
      span.bg-muted.text-3xs.rounded-full.px-2.py-1 {{ rangePill }}
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ money(summary.cost) }}
    .text-2xs.text-muted-foreground.mt-1.flex.items-center.justify-between.gap-2
      span {{ $t("usage.kpis.tasksInScope", { count: summary.tasks }) }}
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-1)" :values="spark.cost")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold {{ $t("account.tabs.tokens") }}
      span.text-muted-foreground.text-10 {{ $t("usage.kpis.inOut") }}
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatTokens(summary.inputTokens + summary.outputTokens) }}
    .text-2xs.text-muted-foreground.mt-1
      | {{ $t("usage.kpis.inOutCounts", { in: formatTokens(summary.inputTokens), out: formatTokens(summary.outputTokens) }) }}
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-2)" :values="spark.tokens")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold {{ $t("usage.kpis.rounds") }}
      span.text-muted-foreground.text-10 {{ $t("usage.kpis.agentTurns") }}
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatCount(summary.runs) }}
    .text-2xs.text-muted-foreground.mt-1
      | {{ $t("usage.kpis.completedOf", { completed: summary.completed, tasks: summary.tasks }) }}
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-3)" :values="spark.runs")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1(
    v-if="showWorkTiles"
  )
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold {{ $t("usage.kpis.ttcTotal") }}
      span.text-muted-foreground.text-10 {{ $t("usage.kpis.outsideBacklog") }}
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ summary.ttcSamples ? formatDuration(summary.ttcMs) : "—" }}
    .text-2xs.text-muted-foreground.mt-1
      template(v-if="summary.ttcSamples === summary.completed")
        | {{ $t("usage.kpis.acrossCompleted", { count: summary.completed }) }}
      template(v-else-if="summary.ttcSamples")
        | {{ $t("usage.kpis.timed", { timed: summary.ttcSamples, completed: summary.completed }) }}
      template(v-else) {{ $t("usage.kpis.noCompleted") }}
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-5)" :values="spark.ttc")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1(v-else)
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold {{ $t("usage.kpis.thinking") }}
      span.text-muted-foreground.text-10 {{ $t("usage.kpis.modelWork") }}
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatDuration(summary.thinkingMs) }}
    .text-2xs.text-muted-foreground.mt-1
      template(v-if="summary.runs")
        | {{ $t("usage.kpis.avgPerRound", { value: formatDuration(summary.thinkingMs / summary.runs) }) }}
      template(v-else) {{ $t("usage.kpis.noRounds") }}
    .mt-auto.h-7.pt-1
      UsageSparkline(color="var(--chart-2)" :values="spark.thinking")

  template(v-if="showWorkTiles")
    .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-2xs.font-semibold {{ $t("usage.kpis.completed") }}
        span.text-muted-foreground.text-10 {{ $t("usage.kpis.inclArchived") }}
      .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatCount(summary.completed) }}
      .text-2xs.text-muted-foreground.mt-1
        | {{ $t("usage.kpis.archivedAllTime", { archived: summary.archived, allTime: summary.allTimeCompleted }) }}
      .mt-auto.h-7.pt-1
        UsageSparkline(color="var(--chart-3)" :values="spark.completed")

    .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-2xs.font-semibold {{ $t("taskDetails.usage.thinking") }}
        span.text-muted-foreground.text-10 {{ $t("usage.kpis.modelWork") }}
      .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatDuration(summary.thinkingMs) }}
      .text-2xs.text-muted-foreground.mt-1
        template(v-if="summary.runs")
          | {{ $t("usage.kpis.avgPerRound", { value: formatDuration(summary.thinkingMs / summary.runs) }) }}
        template(v-else) {{ $t("usage.kpis.noRounds") }}
      .mt-auto.h-7.pt-1
        UsageSparkline(color="var(--chart-2)" :values="spark.thinking")

    .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-2xs.font-semibold {{ $t("taskDetails.usage.active") }}
        span.text-muted-foreground.text-10 {{ $t("usage.kpis.agentWork") }}
      .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ formatDuration(summary.activeMs) }}
      .text-2xs.text-muted-foreground.mt-1
        template(v-if="summary.runs")
          | {{ $t("usage.kpis.avgPerRound", { value: formatDuration(summary.activeMs / summary.runs) }) }}
        template(v-else) {{ $t("usage.kpis.noRounds") }}
      .mt-auto.h-7.pt-1
        UsageSparkline(color="var(--chart-4)" :values="spark.active")
</template>

<script setup lang="ts">
import UsageSparkline from "@/components/usage/UsageSparkline.vue";
import type {
  UsageContext,
  UsageSeriesPoint,
} from "@/composables/useUsageAnalytics";
import { formatCount, formatDuration, formatTokens } from "@/lib/format";
import {
  usageRangePillKey,
  type UsageInsightDay,
  type UsageRange,
  type UsageSummary,
} from "@/lib/usage";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  summary: UsageSummary;
  daily: UsageSeriesPoint[];

  insightDaily?: UsageInsightDay[];
  range: UsageRange;

  context: UsageContext;
}>();

const showWorkTiles = computed(() => props.context === "agency");

const rangePill = computed(() =>
  t(usageRangePillKey(props.range), { days: props.range }),
);

function money(value: number): string {
  if (value >= 1) return `$${value.toFixed(2)}`;
  return `$${value.toFixed(4)}`;
}

function last(values: number[]): number[] {
  return values.slice(-18);
}

const spark = computed(() => {
  const insights = props.insightDaily ?? [];
  return {
    cost: last(props.daily.map((point) => point.cost)),
    tokens: last(props.daily.map((point) => point.input + point.output)),
    runs: last(props.daily.map((point) => point.runs)),
    completed: last(props.daily.map((point) => point.completed)),
    active: last(insights.map((day) => day.activeMs)),
    ttc: last(insights.map((day) => day.ttcMs)),
    thinking: last(insights.map((day) => day.thinkingMs)),
  };
});
</script>

<i18n lang="yaml">
en:
  usage:
    kpis:
      agentTurns: "agent turns"
      modelWork: "model work"
      agentWork: "agent work"
      inOut: "in / out"
      rounds: "Response rounds"
      ttcTotal: "TTC total"
      outsideBacklog: "outside backlog"
      noCompleted: "no completed tasks in range"
      completed: "Completed tasks"
      inclArchived: "incl. archived"
      noRounds: "no rounds"
      aria: "Usage summary"
      tasksInScope: "{count} tasks in scope"
      inOutCounts: "{in} in · {out} out"
      completedOf: "{completed} completed · {tasks} tasks"
      acrossCompleted: "across {count} completed tasks"
      timed: "{timed} of {completed} timed"
      archivedAllTime: "{archived} archived · {allTime} all time"
      avgPerRound: "{value} avg / round"
      thinking: "Thinking time"
ru:
  usage:
    kpis:
      agentTurns: "ходов агента"
      modelWork: "работа модели"
      agentWork: "работа агента"
      inOut: "вход / выход"
      rounds: "Раунды ответов"
      ttcTotal: "TTC всего"
      outsideBacklog: "вне бэклога"
      noCompleted: "нет завершённых задач в диапазоне"
      completed: "Завершённые задачи"
      inclArchived: "вкл. архивные"
      noRounds: "нет раундов"
      aria: "Сводка использования"
      tasksInScope: "задач в области: {count}"
      inOutCounts: "{in} вх. · {out} исх."
      completedOf: "завершено {completed} · задач {tasks}"
      acrossCompleted: "по {count} завершённым задачам"
      timed: "измерено {timed} из {completed}"
      archivedAllTime: "архивных {archived} · всего {allTime}"
      avgPerRound: "{value} в среднем / раунд"
      thinking: "Время размышления"
</i18n>
