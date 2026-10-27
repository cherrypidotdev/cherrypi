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
    CardTitle Completed tasks by tag
    CardDescription Cumulative all-time count · one line per tag · multi-tag tasks count in each line.
    CardAction
      .flex.flex-wrap.items-center.justify-end.gap-2
        span.text-11.text-muted-foreground.tabular-nums {{ completed }} completed in range
        label.text-2xs.text-muted-foreground.flex.items-center.gap-1
          Switch(
            size="sm"
            :model-value="includeArchived"
            @update:model-value="emit('update:includeArchived', $event)"
          )
          | Include archived
  CardContent
    p.text-muted-foreground.py-12.text-center.text-sm(v-if="!series.length")
      | No completed tasks yet.
    ChartContainer.chart.w-full(v-else class="h-[232px]" :config="config")
      VisXYContainer(:data="rows")
        VisLine(
          v-for="line in series"
          :key="line.id"
          :color="line.color"
          :x="x"
          :y="yFor(line.id)"
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
      span.flex.items-center.gap-1(v-for="line in series" :key="line.id")
        i.inline-block.size-2.rounded-sm(:style="{ background: line.color }")
        | {{ line.name }}
    .text-2xs.text-muted-foreground.tabular-nums {{ rangeLabel }} · all-time cumulative
  CardContent.pt-0
    .text-10.text-muted-foreground.flex.items-start.gap-1
      span.bg-muted.text-8.flex.size-3.shrink-0.items-center.justify-center.rounded-full.font-bold i
      span Tag lines overlap by design. The first point carries the count completed before the selected range.
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
import { Switch } from "@/components/ui/switch";
import type { UsageTagSeries } from "@/composables/useUsageAnalytics";
import { VisAxis, VisLine, VisXYContainer } from "@unovis/vue";
import { computed } from "vue";

const props = defineProps<{
  series: UsageTagSeries[];
  completed: number;
  rangeLabel: string;
  includeArchived: boolean;
}>();

const emit = defineEmits<{
  (e: "update:includeArchived", value: boolean): void;
}>();

interface TagRow {
  date: Date;
  [key: string]: number | Date;
}

const x = (row: TagRow) => row.date.getTime();

const rows = computed<TagRow[]>(() => {
  const first = props.series[0];
  if (!first) return [];
  return first.points.map((point, index) => {
    const row: TagRow = { date: point.date };
    for (const line of props.series) {
      row[line.id] = line.points[index]?.value ?? 0;
    }
    return row;
  });
});

function yFor(id: string) {
  return (row: TagRow) => Number(row[id] ?? 0);
}

const config = computed<ChartConfig>(() => {
  const result: ChartConfig = {};
  for (const line of props.series) {
    result[line.id] = {
      label: line.name,
      color: line.color,
      format: (value: number) => String(Math.round(value)),
    };
  }
  return result;
});
</script>
