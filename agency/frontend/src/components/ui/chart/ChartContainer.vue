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

<script lang="ts">
import { cn } from "@/lib/utils";
import { useId } from "reka-ui";
import type { HTMLAttributes } from "vue";
import { computed, toRefs } from "vue";
import type { ChartConfig } from ".";
import { provideChartContext } from ".";
import ChartStyle from "./ChartStyle.vue";
</script>

<script setup lang="ts">
const props = defineProps<{
  id?: HTMLAttributes["id"];
  class?: HTMLAttributes["class"];
  config: ChartConfig;
  cursor?: boolean;
}>();

defineSlots<{
  default: {
    id: string;
    config: ChartConfig;
  };
}>();

const { config } = toRefs(props);
const uniqueId = useId();
const chartId = computed(
  () => `chart-${props.id || uniqueId.replace(/:/g, "")}`,
);

provideChartContext({
  id: uniqueId,
  config,
});
</script>

<template>
  <div
    data-slot="chart"
    :data-chart="chartId"
    :class="
      cn(
        `cn-chart [&_.tick_text]:!fill-muted-foreground [&_.tick_line]:!stroke-border/50 [&_.recharts-curve.recharts-tooltip-cursor]:stroke-border [&_.recharts-polar-grid_[stroke='#ccc']]:stroke-border [&_.recharts-radial-bar-background-sector]:fill-muted [&_.recharts-rectangle.recharts-tooltip-cursor]:fill-muted [&_.recharts-reference-line_[stroke='#ccc']]:stroke-border flex flex-col aspect-video justify-center text-xs [&_.recharts-dot[stroke='#fff']]:stroke-transparent [&_.recharts-layer]:outline-hidden [&_.recharts-sector]:outline-hidden [&_.recharts-sector[stroke='#fff']]:stroke-transparent [&_.recharts-surface]:outline-hidden [&_[data-vis-xy-container]]:h-full [&_[data-vis-single-container]]:h-full h-full [&_[data-vis-xy-container]]:w-full [&_[data-vis-single-container]]:w-full w-full `,
        props.class,
      )
    "
    :style="{
      '--vis-tooltip-padding': '0px',
      '--vis-tooltip-background-color': 'transparent',
      '--vis-tooltip-border-color': 'transparent',
      '--vis-tooltip-text-color': 'none',
      '--vis-tooltip-shadow-color': 'none',
      '--vis-tooltip-backdrop-filter': 'none',
      '--vis-crosshair-circle-stroke-color': '#0000',
      '--vis-crosshair-line-stroke-width': cursor ? '1px' : '0px',
      '--vis-font-family': 'var(--font-sans)',
    }"
  >
    <slot :id="uniqueId" :config="config" />
    <ChartStyle :id="chartId" />
  </div>
</template>
