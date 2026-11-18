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
//- A single-line text that the row clips at its end and, while `active`, drifts

span.marquee(ref="root")
  span.marquee-track(
    ref="track"
    :class="{ 'marquee-track--scrolling': drifting }"
    :style="trackStyle"
  )
    slot
</template>

<script setup lang="ts">
import {
  marqueeShift,
  marqueeStartSkipSeconds,
  marqueeTimeline,
} from "@/lib/marquee";
import { usePreferredReducedMotion } from "@vueuse/core";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    active?: boolean;

    delay?: number;

    speed?: number;
  }>(),
  { active: false, delay: 3000, speed: 40 },
);

const root = ref<HTMLElement>();
const track = ref<HTMLElement>();

const overflow = ref(0);

const drifting = ref(false);
const motion = usePreferredReducedMotion();
let restTimer: ReturnType<typeof setTimeout> | undefined;
let observer: ResizeObserver | null = null;

function measure() {
  const box = root.value;
  const text = track.value;
  if (!box || !text) return;
  const width = text.getBoundingClientRect().width;
  overflow.value = Math.max(0, Math.ceil(width - box.clientWidth));
}

function stopRest() {
  if (restTimer !== undefined) clearTimeout(restTimer);
  restTimer = undefined;
}

function schedule() {
  stopRest();
  const shift = marqueeShift(overflow.value);
  if (!props.active || motion.value === "reduce" || shift === 0) {
    drifting.value = false;
    return;
  }
  if (drifting.value) return;
  restTimer = setTimeout(() => {
    restTimer = undefined;
    drifting.value = true;
  }, props.delay);
}

const trackStyle = computed(() => {
  if (!drifting.value) return {};

  const { shift, duration } = marqueeTimeline(
    marqueeShift(overflow.value),
    props.speed,
  );
  return {
    "--marquee-shift": `${shift}px`,
    "--marquee-duration": `${duration.toFixed(2)}s`,

    "--marquee-start-skip": `${marqueeStartSkipSeconds(duration)}s`,
  };
});

watch([() => props.active, overflow, motion], schedule, { flush: "post" });

onMounted(() => {
  measure();
  if (typeof ResizeObserver !== "undefined" && root.value) {
    observer = new ResizeObserver(() => {
      const wasDrifting = drifting.value;
      measure();

      if (wasDrifting && marqueeShift(overflow.value) === 0)
        drifting.value = false;
      else schedule();
    });
    observer.observe(root.value);
  }
  schedule();
});

onBeforeUnmount(() => {
  stopRest();
  observer?.disconnect();
  observer = null;
});
</script>

<style scoped>
.marquee {
  display: block;
  min-width: 0;
  overflow: hidden;
}

.marquee-track {
  display: inline-block;
  white-space: nowrap;
}

@media (prefers-reduced-motion: no-preference) {
  .marquee-track--scrolling {
    animation: marquee-title var(--marquee-duration, 6s) linear
      var(--marquee-start-skip, 0s) infinite;
    will-change: transform;
  }
}

@keyframes marquee-title {
  0% {
    transform: translateX(0);
  }
  10% {
    transform: translateX(0);
  }
  45% {
    transform: translateX(calc(-1 * var(--marquee-shift, 0px)));
  }
  55% {
    transform: translateX(calc(-1 * var(--marquee-shift, 0px)));
  }
  90% {
    transform: translateX(0);
  }
  100% {
    transform: translateX(0);
  }
}
</style>
