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
    CardTitle Usage over time
    CardDescription {{ subtitle }}
    CardAction
      .bg-muted.flex.rounded-lg.p-1
        button.text-11.rounded-md.px-2.py-1.font-medium(
          v-for="option in METRICS"
          :key="option.value"
          :class="metric === option.value ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
          @click="metric = option.value"
        ) {{ option.label }}
  CardContent
    p.text-muted-foreground.py-10.text-center.text-sm(v-if="!points.length")
      | No usage recorded in this range.
    ChartContainer.chart.w-full(v-else class="h-[238px]" :config="config")
      VisXYContainer(:data="points")
        template(v-if="metric === 'cost'")
          VisArea(
            :color="COLORS.cost"
            :x="x"
            :y="(d: UsageSeriesPoint) => d.cost"
            :opacity="0.35"
          )
          VisLine(
            :color="COLORS.cost"
            :x="x"
            :y="(d: UsageSeriesPoint) => d.cost"
          )
        template(v-else)
          VisLine(
            :color="COLORS.input"
            :x="x"
            :y="(d: UsageSeriesPoint) => d.input"
          )
          VisLine(
            :color="COLORS.output"
            :x="x"
            :y="(d: UsageSeriesPoint) => d.output"
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
      template(v-if="metric === 'cost'")
        span.flex.items-center.gap-1
          i.inline-block.size-2.rounded-sm(
            :style="{ background: USAGE_COLORS.cost }"
          )
          | Daily cost
      template(v-else)
        span.flex.items-center.gap-1
          i.inline-block.size-2.rounded-sm(
            :style="{ background: USAGE_COLORS.input }"
          )
          | Input
        span.flex.items-center.gap-1
          i.inline-block.size-2.rounded-sm(
            :style="{ background: USAGE_COLORS.output }"
          )
          | Output
    .text-2xs.text-muted-foreground.tabular-nums {{ footer }}
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
import { formatCost, formatTokens } from "@/lib/format";
import { USAGE_COLORS } from "@/lib/usage";
import { VisArea, VisAxis, VisLine, VisXYContainer } from "@unovis/vue";
import { computed, ref } from "vue";

type Metric = "cost" | "tokens";

const props = defineProps<{
  points: UsageSeriesPoint[];
  rangeLabel: string;
}>();

const METRICS: { value: Metric; label: string }[] = [
  { value: "cost", label: "Cost" },
  { value: "tokens", label: "Tokens" },
];
const metric = ref<Metric>("cost");

const COLORS = USAGE_COLORS;

const x = (d: UsageSeriesPoint) => d.date.getTime();

const config = computed<ChartConfig>(() => {
  if (metric.value === "cost") {
    const cost: ChartConfig = {
      cost: {
        label: "Cost",
        color: USAGE_COLORS.cost,
        format: (value: number) => formatCost(value),
      },
    };
    return cost;
  }
  const tokens: ChartConfig = {
    input: {
      label: "Input",
      color: USAGE_COLORS.input,
      format: (value: number) => formatTokens(value),
    },
    output: {
      label: "Output",
      color: USAGE_COLORS.output,
      format: (value: number) => formatTokens(value),
    },
  };
  return tokens;
});

const subtitle = computed(() =>
  metric.value === "cost"
    ? `Settled cost per UTC day · ${props.rangeLabel}.`
    : `Prompt and completion tokens per UTC day · ${props.rangeLabel}.`,
);

const footer = computed(() => {
  const total =
    metric.value === "cost"
      ? formatCost(props.points.reduce((sum, point) => sum + point.cost, 0))
      : `${formatTokens(props.points.reduce((sum, point) => sum + point.input + point.output, 0))} total`;
  return `${total} · ${props.rangeLabel}`;
});
</script>
