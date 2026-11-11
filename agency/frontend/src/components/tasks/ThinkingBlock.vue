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
//- A thinking block reads as a tool call without a body: one header line with

//- stays one line while it streams.
details.group.rounded-md.border(data-testid="thinking-block")
  summary.flex.cursor-pointer.items-center.gap-2.px-2.py-1.text-xs.select-none
    ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
      class="group-open:rotate-90"
    )
    BrainIcon.text-muted-foreground.shrink-0(class="size-3.5")
    span {{ $t("thinking.title") }}
    Spinner.size-3.shrink-0(v-if="running" data-testid="thinking-running")
    //- The metrics sit at the right edge so they line up with the tool rows'

    span.text-muted-foreground.ml-auto.flex.shrink-0.items-center.gap-1
      LiveSpan(:since="start" :until="stoppedAt")
      template(v-if="chars > 0")
        span ·
        span {{ formatChars(chars) }}
      template(v-if="rate !== undefined")
        span ·
        span {{ formatCharRate(rate) }}

  p.text-muted-foreground.max-h-60.overflow-auto.border-t.px-2.py-2.text-xs.whitespace-pre-wrap(
    v-if="text"
    data-testid="thinking-text"
  ) {{ text }}
</template>

<script setup lang="ts">
import LiveSpan from "@/components/tasks/LiveSpan.vue";
import { Spinner } from "@/components/ui/spinner";
import { formatCharRate, formatChars } from "@/lib/format";
import { BrainIcon, ChevronRightIcon } from "@lucide/vue";
import { computed, onBeforeUnmount, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    text: string;

    startedAt?: number;

    endedAt?: number;

    running?: boolean;
  }>(),
  { startedAt: undefined, endedAt: undefined, running: false },
);

const chars = computed(() => Array.from(props.text).length);

const start = computed(() => props.startedAt ?? props.endedAt ?? Date.now());

const settledAt = ref<number | undefined>(
  props.running ? undefined : Date.now(),
);
watch(
  () => props.running,
  (running) => {
    if (running) settledAt.value = undefined;
    else settledAt.value ??= Date.now();
  },
);

const stoppedAt = computed(() =>
  props.running ? undefined : (props.endedAt ?? settledAt.value ?? start.value),
);

const now = ref(Date.now());
let timer: ReturnType<typeof setInterval> | undefined;

function stop() {
  if (timer !== undefined) {
    clearInterval(timer);
    timer = undefined;
  }
}

watch(
  () => props.running,
  (running) => {
    stop();
    if (!running) return;
    now.value = Date.now();
    timer = setInterval(() => {
      now.value = Date.now();
    }, 100);
  },
  { immediate: true },
);
onBeforeUnmount(stop);

const rate = computed(() => {
  const end = props.running ? now.value : (props.endedAt ?? now.value);
  const seconds = Math.max(0, end - start.value) / 1000;
  if (seconds <= 0 || chars.value === 0) return undefined;
  return chars.value / seconds;
});
</script>

<i18n lang="yaml">
en:
  thinking:
    title: "Thinking"
ru:
  thinking:
    title: "Размышление"
</i18n>
