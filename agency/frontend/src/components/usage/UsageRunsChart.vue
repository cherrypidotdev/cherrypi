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
    CardTitle Runs per day
    CardDescription One bar per agent response round: the start, then each follow-up settle.
    CardAction
      span.text-11.text-muted-foreground.tabular-nums {{ total }} rounds
  CardContent
    p.text-muted-foreground.py-10.text-center.text-sm(v-if="!points.length")
      | No response rounds in this range.
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
        | Response rounds
      span.flex.items-center.gap-1
        i.inline-block.h-1.w-2(:style="{ background: USAGE_COLORS.completed }")
        | Completed tasks
    .text-2xs.text-muted-foreground.tabular-nums {{ completed }} completed · {{ rangeLabel }}
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

const props = defineProps<{
  points: UsageSeriesPoint[];
  rangeLabel: string;
  completed: number;
}>();

const COLORS = USAGE_COLORS;

const x = (d: UsageSeriesPoint) => d.date.getTime();

const config: ChartConfig = {
  runs: {
    label: "Rounds",
    color: USAGE_COLORS.runs,
    format: (value: number) => String(Math.round(value)),
  },
  completed: {
    label: "Completed tasks",
    color: USAGE_COLORS.completed,
    format: (value: number) => String(Math.round(value)),
  },
};

const total = computed(() =>
  props.points.reduce((sum, point) => sum + point.runs, 0),
);
</script>
