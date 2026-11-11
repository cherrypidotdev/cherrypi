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
.flex.min-w-0.items-center.gap-2
  .bg-muted.relative.min-w-0.flex-1.overflow-hidden.rounded-full(
    role="progressbar"
    :class="size === 'lg' ? 'h-2.5' : size === 'sm' ? 'h-1.5' : 'h-2'"
    :aria-valuemin="0"
    :aria-valuemax="cap"
    :aria-valuenow="Math.min(used, cap)"
    :aria-label="ariaLabel"
    :title="title"
  )
    .absolute.inset-y-0.border-l(
      v-if="reservedPercent > 0"
      class="border-amber-500/40 bg-amber-500/20"
      :style="{ left: `${100 - reservedPercent}%`, width: `${reservedPercent}%` }"
      :title="reservedTitle"
    )
    .h-full.rounded-full.transition-all.duration-300(
      :class="fillClass"
      :style="{ width: `${percent}%` }"
    )
  template(v-if="showCount")
    span.text-2xs.font-mono.tabular-nums(:class="countClass" :title="title")
      | {{ used }}
      span.text-muted-foreground /{{ cap }}
  template(v-if="showWaiting && waiting > 0")
    span.text-3xs.rounded-full.font-medium.whitespace-nowrap(
      class="px-1.5 py-0.5"
      :class="waitingClass"
      :title="waitingTitle"
    )
      | {{ $t("agentConcurrency.waitingBadge", { count: waiting }) }}
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    used: number;
    cap: number;
    waiting?: number;

    reserved?: number;
    label?: string;
    size?: "sm" | "md" | "lg";
    showCount?: boolean;
    showWaiting?: boolean;
  }>(),
  {
    waiting: 0,
    reserved: 0,
    label: "Capacity",
    size: "md",
    showCount: true,
    showWaiting: true,
  },
);

const reservedPercent = computed(() => {
  if (props.cap <= 0 || props.reserved <= 0) return 0;
  return Math.max(0, Math.min(100, (props.reserved / props.cap) * 100));
});

const reservedTitle = computed(
  () => `${props.reserved} of ${props.cap} slot(s) reserved`,
);

const percent = computed(() => {
  if (props.cap <= 0) return props.used > 0 ? 100 : 0;
  return Math.max(0, Math.min(100, (props.used / props.cap) * 100));
});

const full = computed(() => props.cap > 0 && props.used >= props.cap);
const bottleneck = computed(() => full.value && props.waiting > 0);

const fillClass = computed(() => {
  if (bottleneck.value) return "bg-red-500 dark:bg-red-400";
  if (full.value) return "bg-amber-500 dark:bg-amber-400";
  return "bg-emerald-500 dark:bg-emerald-400";
});

const countClass = computed(() => {
  if (bottleneck.value) return "text-red-600 dark:text-red-400";
  if (full.value) return "text-amber-600 dark:text-amber-400";
  return "";
});

const waitingClass = computed(() =>
  bottleneck.value
    ? "bg-red-500/12 text-red-600 dark:text-red-400"
    : "bg-amber-500/12 text-amber-600 dark:text-amber-400",
);

const ariaLabel = computed(
  () =>
    `${props.label}: ${props.used} of ${props.cap} held${props.waiting ? `, ${props.waiting} waiting` : ""}`,
);

const title = computed(() => {
  if (props.cap <= 0) return `${props.used} held · no cap`;
  const free = Math.max(0, props.cap - props.used);
  return `${props.used} of ${props.cap} held · ${free} free${
    props.waiting ? ` · ${props.waiting} waiting` : ""
  }`;
});

const waitingTitle = computed(() => {
  if (bottleneck.value) {
    return `${props.waiting} task${props.waiting === 1 ? "" : "s"} waiting; this cap is full.`;
  }
  return `${props.waiting} task${props.waiting === 1 ? "" : "s"} waiting.`;
});
</script>

<i18n lang="yaml">
en:
  agentConcurrency:
    waitingBadge: "{count} waiting"
ru:
  agentConcurrency:
    waitingBadge: "в очереди: {count}"
</i18n>
