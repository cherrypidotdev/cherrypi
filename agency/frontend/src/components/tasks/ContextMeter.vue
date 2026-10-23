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
.flex.items-center.gap-2(
  v-if="total > 0"
  role="meter"
  aria-valuemin="0"
  aria-valuemax="100"
  :aria-valuenow="percent"
  :aria-label="label"
  :title="hint"
)
  .bg-muted.w-20.overflow-hidden.rounded-full(class="h-1.5")
    .h-full.rounded-full.transition-all(:class="barClass" :style="fillStyle")
  span.text-xs.tabular-nums(:class="textClass") {{ percent }}%
  span.text-muted-foreground.text-xs.tabular-nums {{ ratio }}
</template>

<script setup lang="ts">
import { formatCount, formatTokens } from "@/lib/format";
import { computed } from "vue";

const props = defineProps<{
  /** Tokens currently in the model's context. */
  tokens: number | bigint;
  /** The model's total context window. */
  window: number | bigint;
}>();

const total = computed(() => Number(props.window));
// Clamp so a transient overshoot (e.g. a request that exceeded the window)
// never renders a bar past its track.
const percent = computed(() =>
  total.value > 0
    ? Math.min(
        100,
        Math.max(0, Math.round((Number(props.tokens) / total.value) * 100)),
      )
    : 0,
);

// Warn as the window fills: amber near the compaction threshold, red when a
// compaction is imminent.
const barClass = computed(() => {
  if (percent.value >= 90) return "bg-destructive";
  if (percent.value >= 75) return "bg-amber-500";
  return "bg-primary";
});

const textClass = computed(() => {
  if (percent.value >= 90) return "text-destructive";
  if (percent.value >= 75) return "text-amber-600 dark:text-amber-400";
  return "text-muted-foreground";
});

const fillStyle = computed(() => ({ width: percent.value + "%" }));
const label = computed(() => "Context window " + percent.value + "% used");
const hint = computed(
  () =>
    formatCount(props.tokens) +
    " / " +
    formatCount(total.value) +
    " tokens in context",
);
const ratio = computed(
  () => formatTokens(props.tokens) + "/" + formatTokens(total.value),
);
</script>
