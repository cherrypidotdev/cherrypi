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

<script setup lang="ts">
// The fence rule in .vitepress/config.mts replaces the code block with this

import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { useData } from "vitepress";
import { naturalSize, renderDiagram } from "../mermaid";

const props = defineProps<{ code: string }>();

const { isDark, lang } = useData();

function decode(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

const source = decode(props.code);

const ru = computed(() => lang.value.startsWith("ru"));
const labels = computed(() =>
  ru.value
    ? {
        expand: "Развернуть",
        showSource: "Исходник",
        hideSource: "Скрыть исходник",
        failed:
          "Не удалось отрисовать диаграмму, поэтому показан её исходный код.",
        canvas: "Диаграмма. Активируйте, чтобы открыть во весь экран.",
        title: "Открыть во весь экран",
        dialog: "Диаграмма",
        zoomOut: "Уменьшить",
        zoomIn: "Увеличить",
        reset: "Сброс",
        close: "Закрыть",
        hint: "Прокрутите, чтобы масштабировать, перетащите, чтобы двигать, двойной щелчок сбрасывает, Esc закрывает.",
      }
    : {
        expand: "Expand",
        showSource: "Source",
        hideSource: "Hide source",
        failed:
          "This diagram could not be rendered, so its source is shown instead.",
        canvas: "Diagram. Activate to open it full screen.",
        title: "Open full screen",
        dialog: "Diagram",
        zoomOut: "Zoom out",
        zoomIn: "Zoom in",
        reset: "Reset",
        close: "Close",
        hint: "Scroll to zoom, drag to pan, double-click to reset, Esc to close.",
      },
);

const svg = ref("");
const error = ref("");
const showSource = ref(false);

async function draw(): Promise<void> {
  try {
    svg.value = await renderDiagram(source, isDark.value);
    error.value = "";
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause);
    svg.value = "";
    console.error("[mermaid] could not render a diagram:", cause);
  }
}

onMounted(() => {
  void draw();
});

watch(isDark, () => {
  void draw();
  if (open.value) void drawOverlay();
});

const MIN_SCALE = 0.5;
const MAX_SCALE = 8;

const open = ref(false);
const overlaySvg = ref("");
const fit = ref({ width: 960, height: 540 });
const scale = ref(1);
const translateX = ref(0);
const translateY = ref(0);
const dragging = ref(false);

const stage = ref<HTMLElement | null>(null);
const closeButton = ref<HTMLButtonElement | null>(null);
let restoreFocus: HTMLElement | null = null;

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function computeFit(text: string): void {
  const { width, height } = naturalSize(text);

  const ratio = Math.min(
    (window.innerWidth * 0.92) / width,
    (window.innerHeight * 0.84) / height,
    3,
  );
  fit.value = {
    width: Math.round(width * ratio),
    height: Math.round(height * ratio),
  };
}

function resetView(): void {
  scale.value = 1;
  translateX.value = 0;
  translateY.value = 0;
}

async function drawOverlay(): Promise<void> {
  try {
    overlaySvg.value = await renderDiagram(source, isDark.value);
    computeFit(overlaySvg.value);
  } catch {
    overlaySvg.value = svg.value;
    if (overlaySvg.value) computeFit(overlaySvg.value);
  }
}

async function show(): Promise<void> {
  if (!svg.value) return;
  restoreFocus =
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;
  overlaySvg.value = svg.value;
  computeFit(overlaySvg.value);
  resetView();
  open.value = true;
  document.documentElement.classList.add("mermaid-diagram-locked");
  window.addEventListener("keydown", onKeydown);
  window.addEventListener("resize", onResize);
  await nextTick();
  closeButton.value?.focus();

  void drawOverlay();
}

function hide(): void {
  if (!open.value) return;
  open.value = false;
  overlaySvg.value = "";
  dragging.value = false;
  document.documentElement.classList.remove("mermaid-diagram-locked");
  window.removeEventListener("keydown", onKeydown);
  window.removeEventListener("resize", onResize);
  restoreFocus?.focus();
  restoreFocus = null;
}

function onResize(): void {
  if (overlaySvg.value) {
    computeFit(overlaySvg.value);
    resetView();
  }
}

function onKeydown(event: KeyboardEvent): void {
  switch (event.key) {
    case "Escape":
      event.preventDefault();
      hide();
      break;
    case "+":
    case "=":
      event.preventDefault();
      zoomBy(1.25);
      break;
    case "-":
    case "_":
      event.preventDefault();
      zoomBy(1 / 1.25);
      break;
    case "0":
      event.preventDefault();
      resetView();
      break;
  }
}

function zoomAtPoint(clientX: number, clientY: number, factor: number): void {
  const element = stage.value;
  if (!element) return;
  const next = clamp(scale.value * factor, MIN_SCALE, MAX_SCALE);
  if (next === scale.value) return;
  const rect = element.getBoundingClientRect();

  const pointX = clientX - (rect.left + rect.width / 2);
  const pointY = clientY - (rect.top + rect.height / 2);
  const ratio = next / scale.value;
  translateX.value = pointX - ratio * (pointX - translateX.value);
  translateY.value = pointY - ratio * (pointY - translateY.value);
  scale.value = next;
}

function zoomBy(factor: number): void {
  const element = stage.value;
  if (!element) return;
  const rect = element.getBoundingClientRect();
  zoomAtPoint(rect.left + rect.width / 2, rect.top + rect.height / 2, factor);
}

function onWheel(event: WheelEvent): void {
  zoomAtPoint(event.clientX, event.clientY, event.deltaY < 0 ? 1.12 : 1 / 1.12);
}

let drag = {
  x: 0,
  y: 0,
  translateX: 0,
  translateY: 0,
  moved: false,
  backdrop: false,
};

function onPointerDown(event: PointerEvent): void {
  if (event.button !== 0) return;
  dragging.value = true;
  drag = {
    x: event.clientX,
    y: event.clientY,
    translateX: translateX.value,
    translateY: translateY.value,
    moved: false,

    backdrop: event.target === event.currentTarget,
  };
  (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
}

function onPointerMove(event: PointerEvent): void {
  if (!dragging.value) return;
  const dx = event.clientX - drag.x;
  const dy = event.clientY - drag.y;
  if (Math.abs(dx) + Math.abs(dy) > 3) drag.moved = true;
  translateX.value = drag.translateX + dx;
  translateY.value = drag.translateY + dy;
}

function onPointerUp(event: PointerEvent): void {
  if (!dragging.value) return;
  dragging.value = false;
  const element = event.currentTarget as HTMLElement;
  if (element.hasPointerCapture(event.pointerId)) {
    element.releasePointerCapture(event.pointerId);
  }

  if (!drag.moved && drag.backdrop) hide();
}

onBeforeUnmount(() => {
  hide();
});
</script>

<template>
  <figure class="mermaid-diagram" :class="{ 'is-error': error }">
    <div
      v-if="svg"
      class="mermaid-diagram__canvas"
      role="button"
      tabindex="0"
      :aria-label="labels.canvas"
      :title="labels.title"
      @click="show"
      @keydown.enter.prevent="show"
      @keydown.space.prevent="show"
      v-html="svg"
    />
    <div v-else-if="!error" class="mermaid-diagram__canvas is-pending">
      <pre class="mermaid-diagram__code"><code>{{ source }}</code></pre>
    </div>

    <template v-if="error">
      <p class="mermaid-diagram__error">{{ labels.failed }}</p>
      <pre class="mermaid-diagram__code"><code>{{ source }}</code></pre>
    </template>

    <div v-else-if="svg" class="mermaid-diagram__bar">
      <button type="button" class="mermaid-diagram__button" @click="show">
        {{ labels.expand }}
      </button>
      <button
        type="button"
        class="mermaid-diagram__button"
        :aria-expanded="showSource"
        @click="showSource = !showSource"
      >
        {{ showSource ? labels.hideSource : labels.showSource }}
      </button>
    </div>

    <pre v-if="showSource && !error" class="mermaid-diagram__code"><code>{{
      source
    }}</code></pre>
  </figure>

  <Teleport to="body">
    <div
      v-if="open"
      class="mermaid-zoom"
      role="dialog"
      aria-modal="true"
      :aria-label="labels.dialog"
    >
      <div
        ref="stage"
        class="mermaid-zoom__stage"
        :class="{ 'is-dragging': dragging }"
        @wheel.prevent="onWheel"
        @pointerdown="onPointerDown"
        @pointermove="onPointerMove"
        @pointerup="onPointerUp"
        @pointercancel="onPointerUp"
        @dblclick="resetView"
      >
        <div
          class="mermaid-zoom__canvas"
          :style="{
            width: `${fit.width}px`,
            height: `${fit.height}px`,
            transform: `translate(${translateX}px, ${translateY}px) scale(${scale})`,
          }"
          v-html="overlaySvg"
        />
      </div>

      <div class="mermaid-zoom__bar">
        <button
          type="button"
          class="mermaid-zoom__button"
          :aria-label="labels.zoomOut"
          @click="zoomBy(1 / 1.25)"
        >
          −
        </button>
        <button
          type="button"
          class="mermaid-zoom__button"
          :aria-label="labels.zoomIn"
          @click="zoomBy(1.25)"
        >
          +
        </button>
        <button type="button" class="mermaid-zoom__button" @click="resetView">
          {{ labels.reset }}
        </button>
        <button
          ref="closeButton"
          type="button"
          class="mermaid-zoom__button is-primary"
          @click="hide"
        >
          {{ labels.close }}
        </button>
      </div>

      <p class="mermaid-zoom__hint">{{ labels.hint }}</p>
    </div>
  </Teleport>
</template>
