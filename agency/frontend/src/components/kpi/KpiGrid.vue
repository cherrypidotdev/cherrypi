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
.flex.flex-col.gap-3
  .grid.gap-3(
    class="auto-rows-[minmax(6.75rem,auto)]"
    :aria-label="$t('kpiGrid.title')"
    :class="editing ? 'rounded-xl bg-muted/30 p-2 ring-1 ring-border' : ''"
    :style="gridStyle"
  )
    template(v-if="!placed.length")
      .text-muted-foreground.text-3xs.flex.items-center.justify-center.rounded-lg.border.border-dashed(
        v-for="slot in emptySlots"
        :key="`slot-${slot}`"
        class="border-muted-foreground/30"
      )
        template(v-if="slot === 1") {{ $t("kpiGrid.empty") }}

    template(v-for="group in groups" :key="group.key")
      KpiUsageScope(
        v-if="group.kind === 'usage'"
        v-slot="{ data }"
        :agent-id="group.agentId"
        :project-id="group.projectId"
        :range="group.range"
      )
        KpiWidget(
          v-for="entry in group.widgets"
          :key="entry.widget.id"
          :widget="entry.widget"
          :widgets="widgets"
          :placement="entry.placement"
          :content="usageContent(entry.widget, data)"
          :error="data.error"
          :editing="editing"
          :can-edit="canEdit"
          :index="indexOf(entry.widget.id)"
          :narrow="narrow"
          :class="dropClass(entry.widget.id)"
          @settings="openSettings"
          @remove="remove"
          @resize="resize"
          @move="move"
          @dragstart="draggingId = $event"
          @dragend="draggingId = ''"
          @dragover="dropTarget = $event"
          @dragleave="clearDropTarget"
          @drop="dropOn($event)"
          :loading="data.loading"
        )
      KpiConcurrencyScope(
        v-else-if="group.boardKind && group.boardId"
        v-slot="{ data }"
        :board-kind="group.boardKind"
        :board-id="group.boardId"
      )
        KpiWidget(
          v-for="entry in group.widgets"
          :key="entry.widget.id"
          :widget="entry.widget"
          :widgets="widgets"
          :placement="entry.placement"
          :content="concurrencyContent(entry.widget, data)"
          :error="data.error"
          :editing="editing"
          :can-edit="canEdit"
          :index="indexOf(entry.widget.id)"
          :narrow="narrow"
          :class="dropClass(entry.widget.id)"
          @settings="openSettings"
          @remove="remove"
          @resize="resize"
          @move="move"
          @dragstart="draggingId = $event"
          @dragend="draggingId = ''"
          @dragover="dropTarget = $event"
          @dragleave="clearDropTarget"
          @drop="dropOn($event)"
          :loading="data.loading"
        )
      template(v-else)
        KpiWidget(
          v-for="entry in group.widgets"
          :key="entry.widget.id"
          :placeholder="$t('kpiGrid.pickScope')"
          :widget="entry.widget"
          :widgets="widgets"
          :placement="entry.placement"
          :error="''"
          :editing="editing"
          :can-edit="canEdit"
          :index="indexOf(entry.widget.id)"
          :narrow="narrow"
          :class="dropClass(entry.widget.id)"
          @settings="openSettings"
          @remove="remove"
          @resize="resize"
          @move="move"
          @dragstart="draggingId = $event"
          @dragend="draggingId = ''"
          @dragover="dropTarget = $event"
          @dragleave="clearDropTarget"
          @drop="dropOn($event)"
          :loading="false"
        )

  template(v-if="editing && !hideActions")
    p.text-muted-foreground.text-3xs.flex.items-center.gap-1
      MoveIcon.size-3.shrink-0
      | {{ $t("kpiGrid.dragHint") }}
    p.text-muted-foreground.text-3xs
      | {{ $t("kpiGrid.keyboardHint") }}

    .flex.flex-wrap.items-center.gap-2
      DropdownMenu
        DropdownMenuTrigger(as-child)
          Button(type="button" variant="outline" size="sm")
            PlusIcon(data-icon="inline-start")
            span {{ $t("kpiGrid.addWidget") }}
        DropdownMenuContent.w-64(align="start")
          template(v-for="section in KPI_WIDGET_MENU" :key="section.labelKey")
            DropdownMenuLabel {{ $t(section.labelKey) }}
            DropdownMenuItem(
              v-for="definition in section.widgets"
              :key="definition.kind"
              @click="add(definition.kind)"
              :disabled="!canAdd(definition.kind)"
            )
              .flex.min-w-0.flex-col
                span {{ $t(definition.labelKey) }}
                span.text-muted-foreground.text-3xs {{ $t(definition.hintKey) }}
            DropdownMenuSeparator
      Button(
        v-if="!inherited"
        type="button"
        variant="outline"
        size="sm"
        :title="$t('kpiGrid.resetOrg')"
        @click="$emit('reset')"
      )
        RotateCcwIcon(data-icon="inline-start")
        span {{ $t("kpiGrid.useDefault") }}
      .flex-1
      Button(type="button" variant="ghost" size="sm" @click="$emit('cancel')")
        | {{ $t("common.cancel") }}
      Button(
        type="button"
        size="sm"
        :title="fits ? $t('kpiGrid.saveHint') : $t('kpiGrid.noFitHint')"
        @click="$emit('save')"
        :disabled="saving || !fits"
      )
        Spinner(v-if="saving" data-icon="inline-start")
        CheckIcon(v-else data-icon="inline-start")
        span {{ $t("kpiGrid.saveLayout") }}

  Dialog(
    :open="Boolean(selected)"
    @update:open="(value) => !value && (settingsId = '')"
  )
    DialogContent(class="sm:max-w-md")
      DialogHeader
        DialogTitle {{ selectedDefinition ? $t(selectedDefinition.labelKey) : $t("kpiGrid.widgetSettings") }}
        DialogDescription
          | {{ $t("kpiGrid.resizeHint") }}
      .flex.flex-col.gap-4(v-if="selected")
        label.flex.flex-col.gap-1(v-if="selectedDefinition?.ranged")
          span.text-sm.font-medium {{ $t("kpiGrid.timeFrame") }}
          Select(
            :model-value="rangeValue(selected.widget)"
            @update:model-value="setRange"
          )
            SelectTrigger.w-full(:title="$t('kpiGrid.frame')")
              SelectValue
            SelectContent
              SelectItem(:value="PAGE_RANGE") {{ $t("kpiGrid.pageDefault") }}
              SelectItem(
                v-for="option in USAGE_RANGE_OPTIONS"
                :key="option.value"
                :value="option.value"
              ) {{ option.label }}
          span.text-muted-foreground.text-xs
            | {{ $t("kpiGrid.pageDefaultHint") }}
        label.flex.flex-col.gap-1(
          v-if="selectedDefinition?.scoped && scopeOptions.length > 1"
        )
          span.text-sm.font-medium {{ $t("kpiGrid.scopeLabel") }}
          Select(
            :model-value="scopeValue(selected.widget)"
            @update:model-value="setScope"
          )
            SelectTrigger.w-full(:title="$t('kpiGrid.scope')")
              SelectValue
            SelectContent
              SelectItem(
                v-for="option in scopeOptions"
                :key="option.value"
                :value="option.value"
              ) {{ option.label }}
          span.text-muted-foreground.text-xs
            | {{ $t("kpiGrid.followPageHint") }}
        .bg-muted.text-muted-foreground.flex.items-center.justify-between.rounded-lg.px-3.py-2.text-xs
          span {{ $t("kpiGrid.size") }}
          span.font-mono {{ selected.widget.width }} × {{ selected.widget.height }}
      DialogFooter
        Button(type="button" @click="settingsId = ''")
          | {{ $t("common.done") }}
</template>

<script setup lang="ts">
import KpiConcurrencyScope from "@/components/kpi/KpiConcurrencyScope.vue";
import KpiUsageScope from "@/components/kpi/KpiUsageScope.vue";
import KpiWidget from "@/components/kpi/KpiWidget.vue";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import {
  addKpiWidget,
  KPI_GRID_COLUMNS,
  KPI_MAX_CELLS,
  KPI_WIDGET_MENU,
  kpiBoardScope,
  kpiConcurrencyContent,
  kpiLayoutFits,
  kpiUsageContent,
  kpiUsageScopeKey,
  kpiWidgetDefinition,
  kpiWidgetRange,
  kpiWidgetScope,
  moveKpiWidget,
  placeKpiWidgets,
  removeKpiWidget,
  updateKpiWidget,
  type KpiConcurrencyData,
  type KpiPageContext,
  type KpiPlacement,
  type KpiUsageData,
  type KpiWidgetState,
} from "@/lib/kpiWidgets";
import { USAGE_RANGE_OPTIONS, type UsageRange } from "@/lib/usage";
import { CheckIcon, MoveIcon, PlusIcon, RotateCcwIcon } from "@lucide/vue";
import { useMediaQuery } from "@vueuse/core";
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  widgets: KpiWidgetState[];
  page: KpiPageContext;

  range: UsageRange;

  editing?: boolean;
  saving?: boolean;

  canEdit?: boolean;

  inherited?: boolean;

  hideActions?: boolean;
  agentOptions: { id: string; name: string }[];
  projectOptions: { id: string; name: string }[];
}>();

const emit = defineEmits<{
  "update:widgets": [widgets: KpiWidgetState[]];
  save: [];
  cancel: [];
  reset: [];
}>();

const PAGE_RANGE = "__page__";
const PAGE_SCOPE = "__page__";

const settingsId = ref("");
const draggingId = ref("");
const dropTarget = ref("");

const narrow = useMediaQuery("(max-width: 1023px)");
const emptySlots = computed(() => (narrow.value ? 1 : KPI_MAX_CELLS));

const gridStyle = computed(() => ({
  gridTemplateColumns: narrow.value
    ? "minmax(0, 1fr)"
    : `repeat(${KPI_GRID_COLUMNS}, minmax(0, 1fr))`,
}));

const placed = computed(() => placeKpiWidgets(props.widgets));

const selected = computed(() =>
  placed.value.find((entry) => entry.widget.id === settingsId.value),
);
const selectedDefinition = computed(() =>
  selected.value ? kpiWidgetDefinition(selected.value.widget.kind) : undefined,
);

watch(
  () => props.widgets,
  (widgets) => {
    if (!widgets.some((widget) => widget.id === settingsId.value)) {
      settingsId.value = "";
    }
  },
);

interface WidgetGroup {
  key: string;
  kind: "usage" | "concurrency";
  agentId: string;
  projectId: string;
  range: UsageRange;
  boardKind?: "agent" | "project";
  boardId?: string;
  widgets: { widget: KpiWidgetState; placement: KpiPlacement }[];
}

const groups = computed<WidgetGroup[]>(() => {
  const out: WidgetGroup[] = [];
  const byKey = new Map<string, WidgetGroup>();
  for (const entry of placed.value) {
    const definition = kpiWidgetDefinition(entry.widget.kind);
    const concurrency = definition?.group === "concurrency";
    const range = kpiWidgetRange(entry.widget, props.range);
    const scope = kpiWidgetScope(entry.widget, props.page);
    const board = concurrency
      ? kpiBoardScope(entry.widget, props.page)
      : undefined;
    const key = concurrency
      ? `board|${board ? `${board.kind}:${board.id}` : "none"}`
      : kpiUsageScopeKey(scope, range);
    let group = byKey.get(key);
    if (!group) {
      group = {
        key,
        kind: concurrency ? "concurrency" : "usage",
        agentId: scope.agentId,
        projectId: scope.projectId,
        range,
        boardKind: board?.kind,
        boardId: board?.id,
        widgets: [],
      };
      byKey.set(key, group);
      out.push(group);
    }
    group.widgets.push(entry);
  }
  return out;
});

const scopeOptions = computed(() => {
  const options = [{ value: PAGE_SCOPE, label: t("kpiGrid.followPage") }];
  if (props.page.kind !== "agent") {
    for (const agent of props.agentOptions) {
      options.push({
        value: `agent:${agent.id}`,
        label: t("kpiGrid.agentScope", { name: agent.name }),
      });
    }
  }
  if (props.page.kind !== "project") {
    for (const project of props.projectOptions) {
      options.push({
        value: `project:${project.id}`,
        label: t("kpiGrid.projectScope", { name: project.name }),
      });
    }
  }
  return options;
});

function usageContent(widget: KpiWidgetState, data: KpiUsageData) {
  return kpiUsageContent(widget, data, kpiWidgetRange(widget, props.range));
}

function concurrencyContent(widget: KpiWidgetState, data: KpiConcurrencyData) {
  return kpiConcurrencyContent(widget, data);
}

function indexOf(id: string): number {
  return props.widgets.findIndex((widget) => widget.id === id);
}

function dropClass(id: string): string {
  if (!draggingId.value) return "";
  if (id === draggingId.value) return "opacity-40";
  if (id === dropTarget.value) return "ring-2 ring-primary/50";
  return "";
}

function clearDropTarget(id: string) {
  if (dropTarget.value === id) dropTarget.value = "";
}

function openSettings(id: string) {
  settingsId.value = id;
}

function rangeValue(widget: KpiWidgetState): string {
  return widget.range || PAGE_RANGE;
}

function setRange(value: unknown) {
  if (!selected.value) return;
  const range = String(value) === PAGE_RANGE ? "" : String(value);
  emit(
    "update:widgets",
    updateKpiWidget(props.widgets, selected.value.widget.id, { range }),
  );
}

function scopeValue(widget: KpiWidgetState): string {
  return widget.scope || PAGE_SCOPE;
}

function setScope(value: unknown) {
  if (!selected.value) return;
  const scope = String(value) === PAGE_SCOPE ? "" : String(value);
  emit(
    "update:widgets",
    updateKpiWidget(props.widgets, selected.value.widget.id, { scope }),
  );
}

function resize(id: string, size: { width: number; height: number }) {
  emit("update:widgets", updateKpiWidget(props.widgets, id, size));
}

function move(id: string, delta: number) {
  emit("update:widgets", moveKpiWidget(props.widgets, id, indexOf(id) + delta));
}

function canAdd(kind: string): boolean {
  return addKpiWidget(props.widgets, kind) !== props.widgets;
}

function add(kind: string) {
  emit("update:widgets", addKpiWidget(props.widgets, kind));
}

function remove(id: string) {
  emit("update:widgets", removeKpiWidget(props.widgets, id));
}

function dropOn(targetId: string) {
  const from = draggingId.value;
  draggingId.value = "";
  dropTarget.value = "";
  if (!from || from === targetId) return;
  emit("update:widgets", moveKpiWidget(props.widgets, from, indexOf(targetId)));
}

function onKeydown(event: KeyboardEvent) {
  if (event.key !== "Escape" || !props.editing) return;
  if (settingsId.value) return;
  emit("cancel");
}

onBeforeUnmount(() => {
  if (typeof window !== "undefined") {
    window.removeEventListener("keydown", onKeydown);
  }
});

watch(
  () => props.editing,
  (editing) => {
    draggingId.value = "";
    dropTarget.value = "";
    if (!editing) settingsId.value = "";
    if (typeof window === "undefined") return;
    if (editing) window.addEventListener("keydown", onKeydown);
    else window.removeEventListener("keydown", onKeydown);
  },
  { immediate: true },
);

const fits = computed(() => kpiLayoutFits(props.widgets));
</script>

<i18n lang="yaml">
en:
  kpiGrid:
    dragHint: "Drag a widget to reorder it, drag its right or bottom edge to resize it, or click it to change its time frame and scope."
    keyboardHint: "With the keyboard: Tab to a widget, arrow keys to move it, Shift + arrow keys to resize it, Enter for its settings."
    widgetSettings: "Widget settings"
    pageDefault: "Page default"
    pageDefaultHint: "“Page default” follows the page's own time frame."
    followPageHint: "“Follow the page” uses whatever this page is about."
    followPage: "Follow the page"
    agentScope: "Agent · {name}"
    projectScope: "Project · {name}"
    addWidget: "Add widget"
    useDefault: "Use the default"
    saveLayout: "Save layout"
    saveHint: "Save the layout of this page"
    noFitHint: "The widgets do not fit the grid"
    timeFrame: "Time frame"
    scopeLabel: "Scope"
    size: "Size"
    title: "KPI widgets"
    empty: "No widgets yet"
    pickScope: "Pick an agent or project above to see concurrency."
    resetOrg: "Drop this page's own layout and follow the organization default"
    resizeHint: "What this widget shows. Its size is set by dragging the card's edge."
    frame: "Widget time frame"
    scope: "Widget scope"
ru:
  kpiGrid:
    dragHint: "Перетащите виджет, чтобы изменить порядок, потяните за правый или нижний край, чтобы изменить размер, или нажмите на него, чтобы сменить период и область."
    keyboardHint: "С клавиатуры: Tab — к виджету, стрелки — перемещение, Shift + стрелки — размер, Enter — настройки."
    widgetSettings: "Настройки виджета"
    pageDefault: "Как на странице"
    pageDefaultHint: "«Как на странице» использует период самой страницы."
    followPageHint: "«Как на странице» использует то, чему посвящена страница."
    followPage: "Как на странице"
    agentScope: "Агент · {name}"
    projectScope: "Проект · {name}"
    addWidget: "Добавить виджет"
    useDefault: "Использовать значение по умолчанию"
    saveLayout: "Сохранить макет"
    saveHint: "Сохранить макет этой страницы"
    noFitHint: "Виджеты не помещаются в сетку"
    timeFrame: "Период"
    scopeLabel: "Область"
    size: "Размер"
    title: "Виджеты KPI"
    empty: "Виджетов пока нет"
    pickScope: "Выберите агента или проект выше, чтобы увидеть параллельность."
    resetOrg: "Сбросить собственный макет страницы и следовать значению организации"
    resizeHint: "Что показывает виджет. Размер задаётся перетаскиванием края карточки."
    frame: "Период виджета"
    scope: "Область виджета"
</i18n>
