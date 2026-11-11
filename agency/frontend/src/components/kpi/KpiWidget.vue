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
.bg-card.ring-border.group.relative.flex.flex-col.overflow-hidden.rounded-xl.p-3.ring-1(
  ref="card"
  :class="cardClass"
  :style="style"
  :draggable="editing && !resizing"
  :tabindex="editing ? 0 : undefined"
  :aria-label="editing ? editLabel : undefined"
  @click="onClick"
  @keydown="onKeydown"
  @dragstart="onDragStart"
  @dragend="$emit('dragend')"
  @dragover.prevent="editing && $emit('dragover', widget.id)"
  @dragleave="$emit('dragleave', widget.id)"
  @drop.prevent="editing && $emit('drop', widget.id)"
  :data-kpi-card="editing ? widget.id : undefined"
  :data-kpi-widget="widget.id"
  :data-kpi-kind="widget.kind"
)
  .flex.items-center.justify-between.gap-2
    span.text-muted-foreground.text-2xs.truncate.font-semibold {{ label }}
    .flex.shrink-0.items-center.gap-1
      span.text-muted-foreground.text-3xs.truncate(v-if="content?.pill") {{ content.pill }}
      span.text-muted-foreground.text-3xs.truncate(v-else-if="content?.hint") {{ content.hint }}

      Button.rounded-full(
        v-if="editing"
        type="button"
        variant="destructive"
        size="icon-xs"
        :title="$t('kpiWidget.remove')"
        :aria-label="$t('kpiWidget.remove')"
        @click.stop="$emit('remove', widget.id)"
        :data-kpi-remove="widget.id"
      )
        XIcon.size-3

      DropdownMenu(v-else-if="canEdit")
        DropdownMenuTrigger(as-child)
          Button.rounded-full(
            type="button"
            variant="ghost"
            size="icon-xs"
            :title="$t('kpiWidget.options')"
            :aria-label="$t('kpiWidget.options')"
            :data-kpi-menu="widget.id"
          )
            MoreHorizontalIcon.size-4
        DropdownMenuContent.w-48(align="end")
          DropdownMenuItem(@click="$emit('settings', widget.id)")
            SlidersHorizontalIcon
            span {{ $t("kpiWidget.settings") }}
          DropdownMenuSeparator
          DropdownMenuItem.text-destructive(
            @click="$emit('remove', widget.id)"
          )
            Trash2Icon
            span {{ $t("kpiWidget.removeShort") }}

  template(v-if="placeholder")
    .text-muted-foreground.mt-1.text-sm.leading-snug {{ placeholder }}
  template(v-else-if="error")
    p.text-destructive.mt-1.text-xs {{ error }}
  template(v-else-if="loading && !content")
    Skeleton.mt-2.h-7.w-16
    Skeleton.mt-3.w-full(class="h-1.5")
  template(v-else)
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums {{ content?.value ?? "—" }}
    .text-2xs.text-muted-foreground.mt-1 {{ content?.detail }}
    .mt-auto.h-7.pt-1(v-if="content?.spark.length")
      UsageSparkline(:color="sparkColor" :values="content.spark")

  template(v-if="editing && !narrow")
    .absolute.top-0.right-0.bottom-0.w-2.cursor-ew-resize.touch-none(
      aria-hidden="true"
      @pointerdown.stop.prevent="startResize($event, 'e')"
      data-kpi-resize="e"
    )
    .absolute.right-0.bottom-0.left-0.h-2.cursor-ns-resize.touch-none(
      aria-hidden="true"
      @pointerdown.stop.prevent="startResize($event, 's')"
      data-kpi-resize="s"
    )
    .absolute.right-0.bottom-0.size-4.cursor-nwse-resize.touch-none(
      aria-hidden="true"
      @pointerdown.stop.prevent="startResize($event, 'se')"
      data-kpi-resize="se"
    )
    .ring-primary.pointer-events-none.absolute.inset-0.rounded-xl.ring-2(
      :class="resizing ? 'opacity-100' : 'opacity-0 group-hover:opacity-60'"
    )

  .ring-primary.pointer-events-none.absolute.inset-0.rounded-xl.opacity-0.ring-2(
    v-if="editing"
    class="group-focus-visible:opacity-100"
  )
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import UsageSparkline from "@/components/usage/UsageSparkline.vue";
import {
  kpiWidgetDefinition,
  kpiWidgetFits,
  type KpiPlacement,
  type KpiWidgetContent,
  type KpiWidgetState,
} from "@/lib/kpiWidgets";
import {
  MoreHorizontalIcon,
  SlidersHorizontalIcon,
  Trash2Icon,
  XIcon,
} from "@lucide/vue";
import { computed, onBeforeUnmount, ref } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const GRID_GAP = 12;

const props = defineProps<{
  widget: KpiWidgetState;
  placement: KpiPlacement;
  content?: KpiWidgetContent;

  placeholder?: string;
  loading: boolean;
  error: string;

  editing: boolean;

  canEdit: boolean;

  widgets: KpiWidgetState[];

  index: number;

  narrow: boolean;
}>();

const emit = defineEmits<{
  settings: [id: string];
  remove: [id: string];
  resize: [id: string, size: { width: number; height: number }];
  move: [id: string, delta: number];
  dragstart: [id: string];
  dragend: [];
  dragover: [id: string];
  dragleave: [id: string];
  drop: [id: string];
}>();

const card = ref<HTMLElement>();
const resizing = ref(false);
const suppressClick = ref(false);

const style = computed(() =>
  props.narrow
    ? { gridColumn: "1 / -1", gridRow: "auto" }
    : {
        gridColumn: `${props.placement.x + 1} / span ${props.widget.width}`,
        gridRow: `${props.placement.y + 1} / span ${props.widget.height}`,

        animationDelay: props.editing
          ? `${(props.index % 4) * 70}ms`
          : undefined,
      },
);

const cardClass = computed(() => {
  if (!props.editing) return "transition-shadow hover:shadow-sm";
  return [
    "kpi-wiggle cursor-grab select-none outline-none active:cursor-grabbing",
    resizing.value ? "kpi-resizing cursor-nwse-resize" : "",
  ]
    .filter(Boolean)
    .join(" ");
});

const label = computed(
  () =>
    props.content?.label ??
    (kpiWidgetDefinition(props.widget.kind) &&
      t(kpiWidgetDefinition(props.widget.kind)!.labelKey)) ??
    "KPI",
);

const editLabel = computed(() =>
  t("kpiWidget.editLabel", {
    label: label.value,
    width: props.widget.width,
    height: props.widget.height,
  }),
);

const sparkColor = computed(
  () => kpiWidgetDefinition(props.widget.kind)?.color ?? "var(--chart-1)",
);

function onClick() {
  if (!props.editing || resizing.value || suppressClick.value) return;
  emit("settings", props.widget.id);
}

function onKeydown(event: KeyboardEvent) {
  if (!props.editing || resizing.value) return;
  if (event.key === "Enter" || event.key === " ") {
    event.preventDefault();
    emit("settings", props.widget.id);
    return;
  }
  const resize = event.shiftKey;
  if (event.key === "ArrowUp" && resize) return resizeBy(0, -1, event);
  if (event.key === "ArrowDown" && resize) return resizeBy(0, 1, event);
  if (event.key === "ArrowLeft" && resize) return resizeBy(-1, 0, event);
  if (event.key === "ArrowRight" && resize) return resizeBy(1, 0, event);
  if (event.key === "ArrowUp" || event.key === "ArrowLeft") {
    event.preventDefault();
    emit("move", props.widget.id, -1);
    return;
  }
  if (event.key === "ArrowDown" || event.key === "ArrowRight") {
    event.preventDefault();
    emit("move", props.widget.id, 1);
  }
}

function resizeBy(
  deltaWidth: number,
  deltaHeight: number,
  event: KeyboardEvent,
) {
  event.preventDefault();

  if (props.narrow) return;
  const width = clampSize(props.widget.width + deltaWidth, 1, 4);
  const height = clampSize(props.widget.height + deltaHeight, 1, 2);
  if (width === props.widget.width && height === props.widget.height) return;
  if (!kpiWidgetFits(props.widgets, props.widget.id, width, height)) return;
  emit("resize", props.widget.id, { width, height });
}

function onDragStart(event: DragEvent) {
  if (!props.editing || resizing.value) {
    event.preventDefault();
    return;
  }
  event.dataTransfer?.setData("text/plain", props.widget.id);
  if (event.dataTransfer) event.dataTransfer.effectAllowed = "move";
  emit("dragstart", props.widget.id);
}

let stopResize: (() => void) | undefined;

onBeforeUnmount(() => stopResize?.());

function startResize(event: PointerEvent, edge: "e" | "s" | "se") {
  const element = card.value;
  if (!element) return;
  const rect = element.getBoundingClientRect();
  const cellWidth = (rect.width + GRID_GAP) / props.widget.width;
  const cellHeight = (rect.height + GRID_GAP) / props.widget.height;
  const startX = event.clientX;
  const startY = event.clientY;
  const startWidth = props.widget.width;
  const startHeight = props.widget.height;
  resizing.value = true;
  suppressClick.value = true;
  (event.target as HTMLElement).setPointerCapture?.(event.pointerId);

  function move(moveEvent: PointerEvent) {
    const byWidth =
      edge === "e" || edge === "se"
        ? Math.round((moveEvent.clientX - startX) / cellWidth)
        : 0;
    const byHeight =
      edge === "s" || edge === "se"
        ? Math.round((moveEvent.clientY - startY) / cellHeight)
        : 0;
    const width = clampSize(startWidth + byWidth, 1, 4);
    const height = clampSize(startHeight + byHeight, 1, 2);
    if (width === props.widget.width && height === props.widget.height) return;
    if (!kpiWidgetFits(props.widgets, props.widget.id, width, height)) return;
    emit("resize", props.widget.id, { width, height });
  }

  function finish() {
    resizing.value = false;

    window.setTimeout(() => (suppressClick.value = false), 0);
    window.removeEventListener("pointermove", move);
    window.removeEventListener("pointerup", finish);
    window.removeEventListener("pointercancel", finish);
    stopResize = undefined;
  }

  window.addEventListener("pointermove", move);
  window.addEventListener("pointerup", finish);
  window.addEventListener("pointercancel", finish);
  stopResize = finish;
}

function clampSize(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
</script>

<style scoped>
.kpi-wiggle {
  animation: kpi-wiggle 420ms ease-in-out infinite alternate;
}

.kpi-wiggle:hover,
.kpi-resizing {
  animation: none;
}

@keyframes kpi-wiggle {
  from {
    transform: rotate(-0.35deg) scale(0.995);
  }
  to {
    transform: rotate(0.35deg) scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .kpi-wiggle {
    animation: none;
  }
}
</style>

<i18n lang="yaml">
en:
  kpiWidget:
    totalCost: "Total cost"
    tokens: "Tokens"
    responseRounds: "Response rounds"
    thinkingTime: "Thinking time"
    activeTime: "Active time"
    ttcTotal: "TTC total"
    completedTasks: "Completed tasks"
    costPerTask: "Cost per task"
    tokensPerTask: "Tokens per task"
    roundsPerTask: "Rounds per task"
    thinkingPerTask: "Thinking per task"
    activePerTask: "Active per task"
    ttcPerTask: "TTC per task"
    slotsHeld: "Slots held"
    waiting: "Waiting"
    freeSlots: "Free slots"
    capsFull: "Caps full"
    unknown: "Unknown widget"
    unknownHint: "This build does not know this widget kind."
    remove: "Remove this widget"
    options: "Widget options"
    editLabel: "{label} widget, {width} by {height} cells. Arrow keys move it, Shift with the arrow keys resizes it, Enter opens its settings."
    settings: "Settings…"
    removeShort: "Remove"
  kpiHint:
    period: "period"
    inOut: "in / out"
    agentTurns: "agent turns"
    modelWork: "model work"
    agentWork: "agent work"
    outsideBacklog: "outside backlog"
    inclArchived: "incl. archived"
    average: "average"
    thisAgent: "this agent"
    pinned: "pinned"
    available: "available"
    dimensions: "dimensions"
  kpiMenu:
    usageTotals: "Usage totals"
    usagePerTask: "Usage per task"
    concurrency: "Concurrency"
ru:
  kpiWidget:
    totalCost: "Общая стоимость"
    tokens: "Токены"
    responseRounds: "Раунды ответов"
    thinkingTime: "Время размышления"
    activeTime: "Активное время"
    ttcTotal: "TTC всего"
    completedTasks: "Завершённые задачи"
    costPerTask: "Стоимость на задачу"
    tokensPerTask: "Токены на задачу"
    roundsPerTask: "Раунды на задачу"
    thinkingPerTask: "Размышление на задачу"
    activePerTask: "Активное время на задачу"
    ttcPerTask: "TTC на задачу"
    slotsHeld: "Занято слотов"
    waiting: "В очереди"
    freeSlots: "Свободные слоты"
    capsFull: "Лимиты исчерпаны"
    unknown: "Неизвестный виджет"
    unknownHint: "Эта сборка не знает такой вид виджета."
    remove: "Убрать этот виджет"
    options: "Параметры виджета"
    editLabel: "Виджет «{label}», {width}×{height} ячеек. Стрелки перемещают, Shift со стрелками меняет размер, Enter открывает настройки."
    settings: "Настройки…"
    removeShort: "Убрать"
  kpiHint:
    period: "период"
    inOut: "вход / выход"
    agentTurns: "ходов агента"
    modelWork: "работа модели"
    agentWork: "работа агента"
    outsideBacklog: "вне бэклога"
    inclArchived: "вкл. архивные"
    average: "в среднем"
    thisAgent: "этот агент"
    pinned: "закреплено"
    available: "доступно"
    dimensions: "измерений"
  kpiMenu:
    usageTotals: "Итоги использования"
    usagePerTask: "Использование на задачу"
    concurrency: "Параллельность"
</i18n>
