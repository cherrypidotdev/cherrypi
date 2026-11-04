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
  g(ref="plot")
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
import { computed, onBeforeUnmount, ref, watch } from "vue";

/**
 * A tiny live sparkline for the health cards.
 *
 * The x axis is a fixed ring of `SLOTS` positions anchored at the right edge,
 * so appending a value shifts the whole line left by exactly one slot. The line
 * is then moved back with a transform and transitioned home: every point
 * travels the same distance, so the sparkline slides instead of being redrawn
 * (a path interpolation would shear each point towards its neighbour).
 */
const SLOTS = 60;
const SLIDE_MS = 260;

const props = withDefaults(
  defineProps<{ values: number[]; color?: string }>(),
  { color: "var(--chart-1)" },
);

const plot = ref<SVGGElement>();
let slideTimer: number | undefined;

const slot = 100 / (SLOTS - 1);

const points = computed(() => {
  const values = props.values.slice(-SLOTS);
  const count = values.length;
  if (!count) return [] as (readonly [number, number])[];
  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  const span = max - min || 1;
  return values.map((value, index) => {
    const x = (SLOTS - count + index) * slot;
    const y = 26 - ((value - min) / span) * 24;
    return [x, y] as const;
  });
});

const line = computed(() =>
  points.value.map(([x, y]) => `${x.toFixed(2)},${y.toFixed(2)}`).join(" "),
);

const last = computed(() => points.value[points.value.length - 1] ?? [0, 26]);

function slide() {
  const node = plot.value;
  if (!node) return;
  if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) return;
  node.style.transition = "none";
  node.style.transform = `translateX(${slot}px)`;
  void node.getBoundingClientRect();
  requestAnimationFrame(() => {
    node.style.transition = `transform ${SLIDE_MS}ms cubic-bezier(0.22, 0.61, 0.36, 1)`;
    node.style.transform = "translateX(0)";
    window.clearTimeout(slideTimer);
    slideTimer = window.setTimeout(() => {
      node.style.transition = "";
      node.style.transform = "";
    }, SLIDE_MS + 80);
  });
}

watch(
  () => props.values,
  (next, previous) => {
    if (!previous?.length || next.length <= previous.length) return;
    slide();
  },
  { flush: "post" },
);

onBeforeUnmount(() => {
  window.clearTimeout(slideTimer);
});
</script>
