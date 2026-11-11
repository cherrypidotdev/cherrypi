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
span.tabular-nums {{ label }}
</template>

<script setup lang="ts">
import { formatElapsed } from "@/lib/format";
import { computed, onBeforeUnmount, ref, watch } from "vue";

const props = defineProps<{
  since: number;

  until?: number;
}>();

const FINE_TICK_MS = 100;
const COARSE_TICK_MS = 1_000;

const now = ref(Date.now());
let timer: ReturnType<typeof setInterval> | undefined;
let fine = false;

function stop() {
  if (timer !== undefined) {
    clearInterval(timer);
    timer = undefined;
  }
}

function tick() {
  now.value = Date.now();
  if (fine && now.value - props.since >= COARSE_TICK_MS) {
    fine = false;
    stop();
    timer = setInterval(tick, COARSE_TICK_MS);
  }
}

watch(
  () => props.until,
  (until) => {
    stop();
    if (until !== undefined) return;
    now.value = Date.now();
    fine = now.value - props.since < COARSE_TICK_MS;
    timer = setInterval(tick, fine ? FINE_TICK_MS : COARSE_TICK_MS);
  },
  { immediate: true },
);

onBeforeUnmount(stop);

const label = computed(() =>
  formatElapsed(Math.max(0, (props.until ?? now.value) - props.since)),
);
</script>
