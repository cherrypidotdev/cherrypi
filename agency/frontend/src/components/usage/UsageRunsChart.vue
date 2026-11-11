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
Card
  CardHeader
    CardTitle {{ $t("usage.runs.title") }}
    CardDescription {{ $t("usage.runs.hint") }}
    CardAction
      span.text-11.text-muted-foreground.tabular-nums {{ $t("usage.runs.roundsCount", { count: total }) }}
  CardContent
    p.text-muted-foreground.py-10.text-center.text-sm(v-if="!points.length")
      | {{ $t("usage.runs.empty") }}
    ChartContainer.chart.w-full(v-else class="h-[224px]" :config="config")
      VisXYContainer(:data="points")
        VisGroupedBar(
          :color="() => COLORS.runs"
          :x="x"
          :y="[(d: UsageSeriesPoint) => d.runs]"
          :rounded-corners="2"
        )
        VisLine(
          :color="COLORS.completed"
          :x="x"
          :y="(d: UsageSeriesPoint) => d.completed"
        )
        VisAxis(
          type="x"
          :tick-line="false"
          :domain-line="false"
          :grid-line="false"
          :tick-format="(d: number) => new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })"
        )
        VisAxis(type="y" :tick-line="false" :domain-line="false")
        ChartTooltip
        ChartCrosshair(
          :template="componentToString(config, ChartTooltipContent, { labelFormatter: (d: number | Date) => new Date(Number(d)).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) })"
        )
  CardFooter.justify-between.gap-3
    .text-2xs.text-muted-foreground.flex.w-full.flex-wrap.items-center.justify-between.gap-3
      span.flex.items-center.gap-1
        i.inline-block.size-2.rounded-sm(
          :style="{ background: USAGE_COLORS.runs }"
        )
        | {{ $t("usage.kpis.rounds") }}
      span.flex.items-center.gap-1
        i.inline-block.h-1.w-2(:style="{ background: USAGE_COLORS.completed }")
        | {{ $t("usage.kpis.completed") }}
    .text-2xs.text-muted-foreground.tabular-nums {{ $t("usage.runs.completedRange", { count: completed, range: rangeLabel }) }}
</template>

<script setup lang="ts">
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartContainer,
  ChartCrosshair,
  ChartTooltip,
  ChartTooltipContent,
  componentToString,
  type ChartConfig,
} from "@/components/ui/chart";
import type { UsageSeriesPoint } from "@/composables/useUsageAnalytics";
import { USAGE_COLORS } from "@/lib/usage";
import { VisAxis, VisGroupedBar, VisLine, VisXYContainer } from "@unovis/vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  points: UsageSeriesPoint[];
  rangeLabel: string;
  completed: number;
}>();

const COLORS = USAGE_COLORS;

const x = (d: UsageSeriesPoint) => d.date.getTime();

const config = computed<ChartConfig>(() => ({
  runs: {
    label: t("usage.breakdown.rounds"),
    color: USAGE_COLORS.runs,
    format: (value: number) => String(Math.round(value)),
  },
  completed: {
    label: t("usage.runs.completed"),
    color: USAGE_COLORS.completed,
    format: (value: number) => String(Math.round(value)),
  },
}));

const total = computed(() =>
  props.points.reduce((sum, point) => sum + point.runs, 0),
);
</script>

<i18n lang="yaml">
en:
  usage:
    runs:
      title: "Runs per day"
      hint: "One bar per agent response round: the start, then each follow-up settle."
      empty: "No response rounds in this range."
      completed: "Completed tasks"
      roundsCount: "{count} rounds"
      completedRange: "{count} completed · {range}"
ru:
  usage:
    runs:
      title: "Запуски по дням"
      hint: "Один столбец на раунд ответа агента: старт, затем каждое завершение уточнения."
      empty: "В этом диапазоне нет раундов ответов."
      completed: "Завершённые задачи"
      roundsCount: "раундов: {count}"
      completedRange: "завершено {count} · {range}"
</i18n>
