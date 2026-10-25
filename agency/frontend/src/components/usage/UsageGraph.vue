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
    CardTitle Cost per day
    CardDescription Accumulated settled cost, UTC.
    CardAction
      .flex.items-baseline.gap-1
        Skeleton.h-5.w-14(v-if="loading")
        template(v-else)
          span.text-lg.font-semibold {{ formatCost(usage?.cost ?? 0) }}
          span.text-muted-foreground.text-xs total
  CardContent
    p.text-destructive.text-sm(v-if="error") {{ error }}
    p.text-muted-foreground.py-6.text-center.text-sm(
      v-else-if="!dayPoints.length"
    )
      | No cost recorded yet.
    ChartContainer.chart.aspect-auto.w-full(
      v-else
      class="h-[160px]"
      :config="costConfig"
    )
      VisXYContainer(:data="dayPoints")
        VisArea(
          :color="costConfig.cost.color"
          :x="(d: DayPoint) => d.date"
          :y="(d: DayPoint) => d.cost"
          :opacity="0.35"
        )
        VisLine(
          :color="costConfig.cost.color"
          :x="(d: DayPoint) => d.date"
          :y="(d: DayPoint) => d.cost"
        )
        VisAxis(
          type="x"
          :tick-line="false"
          :domain-line="false"
          :grid-line="false"
          :tick-format="(d: number | Date) => new Date(d).toLocaleDateString()"
        )
        VisAxis(type="y" :tick-line="false" :domain-line="false")
        ChartTooltip
        ChartCrosshair(
          :template="componentToString(costConfig, ChartTooltipContent, { labelFormatter: (d: number | Date) => new Date(d).toLocaleDateString() })"
        )
</template>

<script setup lang="ts">
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
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
import { Skeleton } from "@/components/ui/skeleton";
import { useUsage } from "@/composables/useUsage";
import { formatCost } from "@/lib/format";
import { VisArea, VisAxis, VisLine, VisXYContainer } from "@unovis/vue";
import { computed } from "vue";

interface DayPoint {
  date: Date;
  cost: number;
}

const props = withDefaults(
  defineProps<{ agentId?: string; projectId?: string }>(),
  { agentId: "", projectId: "" },
);

const { usage, loading, error } = useUsage(
  computed(() => props.agentId),
  computed(() => props.projectId),
);

const dayPoints = computed<DayPoint[]>(() =>
  (usage.value?.days ?? []).map((day) => ({
    date: new Date(Number(day.day?.seconds ?? 0n) * 1000),
    cost: day.cost,
  })),
);

const costConfig = {
  cost: { label: "Cost", color: "var(--chart-3)", format: formatCost },
} satisfies ChartConfig;
</script>
