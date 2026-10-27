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
svg.h-full.w-full(
  viewBox="0 0 100 28"
  preserveAspectRatio="none"
  aria-hidden="true"
)
  polyline(
    fill="none"
    stroke-width="1.5"
    stroke-linejoin="round"
    stroke-linecap="round"
    vector-effect="non-scaling-stroke"
    :points="line"
    :stroke="color"
  )
  circle(r="1.8" :cx="last[0]" :cy="last[1]" :fill="color")
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{ values: number[]; color?: string }>(),
  { color: "var(--chart-1)" },
);

const points = computed(() => {
  const values = props.values.length ? props.values : [0];
  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  const span = max - min || 1;
  return values.map((value, index) => {
    const x = (index / Math.max(values.length - 1, 1)) * 100;
    const y = 26 - ((value - min) / span) * 24;
    return [x, y] as const;
  });
});

const line = computed(() =>
  points.value.map(([x, y]) => `${x.toFixed(2)},${y.toFixed(2)}`).join(" "),
);

const last = computed(() => points.value[points.value.length - 1] ?? [0, 26]);
</script>
