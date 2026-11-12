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

details.rounded-md.border(
  class="group/thinking"
  :open="open"
  @toggle="onToggle"
  data-testid="thinking-block"
)
  summary.flex.cursor-pointer.items-center.gap-2.px-2.py-1.text-xs.select-none
    ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
      class="group-open/thinking:rotate-90"
    )
    BrainIcon.text-muted-foreground.shrink-0(class="size-3.5")
    span {{ $t("thinking.title") }}
    Spinner.size-3.shrink-0(v-if="running" data-testid="thinking-running")
    //- The metrics sit at the right edge so they line up with the tool rows'

    span.text-muted-foreground.ml-auto.flex.shrink-0.items-center.gap-1(
      :title="estimated ? $t('thinking.estimated') : undefined"
      data-testid="thinking-metrics"
      :data-estimated="estimated ? 'true' : undefined"
    )
      template(v-for="(metric, index) in metrics" :key="index")
        span(v-if="index > 0") ·
        span {{ metric }}
      template(v-if="running && start !== undefined")
        span(v-if="metrics.length") ·
        LiveSpan(:since="start")
      template(v-else-if="elapsed !== undefined")
        span(v-if="metrics.length") ·
        span.tabular-nums {{ elapsed }}

  p.text-muted-foreground.max-h-60.overflow-auto.border-t.px-2.py-2.text-xs.whitespace-pre-wrap(
    v-if="text"
    ref="body"
    @scroll.passive="onBodyScroll"
    data-testid="thinking-text"
  ) {{ text }}
</template>

<script setup lang="ts">
import LiveSpan from "@/components/tasks/LiveSpan.vue";
import { Spinner } from "@/components/ui/spinner";
import { formatCharRate, formatChars, formatElapsed } from "@/lib/format";
import { BrainIcon, ChevronRightIcon } from "@lucide/vue";
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    text: string;

    startedAt?: number;

    endedAt?: number;

    durationMs?: number;

    estimated?: boolean;

    running?: boolean;
  }>(),
  {
    startedAt: undefined,
    endedAt: undefined,
    durationMs: undefined,
    estimated: false,
    running: false,
  },
);

const chars = computed(() => Array.from(props.text).length);

const start = computed(() => props.startedAt ?? props.endedAt);

const settledAt = ref<number | undefined>(
  props.running ? undefined : Date.now(),
);

const now = ref(Date.now());
watch(
  () => props.running,
  (running) => {
    if (running) settledAt.value = undefined;
    else settledAt.value ??= Date.now();
  },
);

const measured = computed(() => {
  if (props.durationMs !== undefined) return true;
  if (props.startedAt === undefined) return false;
  if (props.endedAt === undefined) return true;
  return props.endedAt > props.startedAt;
});

const elapsedMs = computed(() => {
  if (props.durationMs !== undefined) return props.durationMs;
  if (!measured.value || start.value === undefined) return undefined;
  const end = props.running
    ? now.value
    : (props.endedAt ?? settledAt.value ?? start.value);
  return Math.max(0, end - start.value);
});

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

/** The metrics that follow the span: the characters the model wrote, and the
 *  rate it wrote them at, so the header reads as one line of figures. */
const metrics = computed(() => {
  const out: string[] = [];
  if (chars.value > 0) out.push(formatChars(chars.value));
  if (rate.value !== undefined) out.push(formatCharRate(rate.value));
  return out;
});

const elapsed = computed(() => {
  const ms = elapsedMs.value;
  if (ms === undefined) return undefined;
  const label = formatElapsed(ms);
  return props.estimated ? `~${label}` : label;
});

const rate = computed(() => {
  const ms = elapsedMs.value;
  if (ms === undefined || ms <= 0 || chars.value === 0) return undefined;
  return chars.value / (ms / 1000);
});

/**
 * Whether the body is showing. A block opens itself while it streams, so the
 * reasoning is readable as the model writes it, and folds back to one line when
 * it stops. A reader who toggles it by hand keeps their choice until the block
 * settles; the fold on settle is the only state the component forces.
 */
const open = ref(props.running);

function onToggle(event: Event) {
  open.value = (event.target as HTMLDetailsElement).open;
}

watch(
  () => props.running,
  (running) => {
    open.value = running;
  },
);

const body = ref<HTMLElement | null>(null);
const bodyStuck = ref(true);

const STICK_TOLERANCE = 24;

function onBodyScroll() {
  const el = body.value;
  if (!el) return;
  bodyStuck.value =
    el.scrollHeight - el.scrollTop - el.clientHeight < STICK_TOLERANCE;
}

watch(
  () => props.text,
  async () => {
    await nextTick();
    const el = body.value;
    if (!el || !open.value || !bodyStuck.value) return;
    el.scrollTop = el.scrollHeight;
  },
);

watch(open, async (isOpen) => {
  if (!isOpen) return;
  await nextTick();
  const el = body.value;
  if (!el) return;
  bodyStuck.value = true;
  el.scrollTop = el.scrollHeight;
});
</script>

<i18n lang="yaml">
en:
  thinking:
    title: "Thinking"
    estimated: "Estimated from the assistant message: this block was not measured while it streamed"
ru:
  thinking:
    title: "Размышление"
    estimated: "Оценка по сообщению ассистента: блок не измерялся во время стриминга"
</i18n>
