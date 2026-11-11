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
.flex.flex-col.gap-2(data-testid="task-tool-calls")
  Input.w-full(
    v-model="search"
    type="search"
    :placeholder="$t('toolCalls.searchPlaceholder')"
    :aria-label="$t('toolCalls.searchPlaceholder')"
    data-testid="tool-calls-search"
  )
  .flex.items-center.gap-2
    Select.flex-1(
      :model-value="status"
      @update:model-value="setStatus($event)"
    )
      SelectTrigger.w-full(size="sm" :title="$t('toolCalls.filterByResult')")
        SelectValue
      SelectContent
        SelectGroup
          SelectItem(value="all") {{ $t("toolCalls.allResults") }}
          SelectItem(value="error") {{ $t("toolCalls.errors") }}
          SelectItem(value="interrupted") {{ $t("toolCalls.interrupted") }}
    DropdownMenu
      DropdownMenuTrigger(as-child)
        Button(
          variant="outline"
          size="sm"
          :title="namesLabel"
          :disabled="!names.length"
          data-testid="tool-calls-names"
        )
          FilterIcon(data-icon="inline-start")
          span {{ namesLabel }}
      DropdownMenuContent.max-h-72.w-56.overflow-y-auto(align="end")
        DropdownMenuLabel {{ $t("plugins.tools") }}
        DropdownMenuSeparator
        DropdownMenuCheckboxItem(
          v-for="name in names"
          :key="name"
          :model-value="selectedNames.includes(name)"
          @update:model-value="toggleName(name)"
          @select.prevent
        ) {{ name }}
        template(v-if="selectedNames.length")
          DropdownMenuSeparator
          DropdownMenuItem(@select="clearNames") {{ $t("toolCalls.clearFilter") }}

  p.text-muted-foreground.text-xs(v-if="error") {{ error }}
  p.text-muted-foreground.text-xs(v-else-if="loading && !calls.length")
    | {{ $t("toolCalls.loading") }}
  p.text-muted-foreground.text-xs(v-else-if="!calls.length")
    | {{ total ? "No tool calls match these filters." : "No tool calls recorded yet." }}

  //- The drawer is narrow, so a call is a two-line row rather than a table:

  .flex.flex-col.overflow-hidden.rounded-md.border(v-if="calls.length")
    .flex.flex-col.border-b(
      v-for="call in calls"
      :key="call.callId"
      class="last:border-b-0"
      data-testid="tool-call-row"
    )
      button.flex.w-full.flex-col.gap-1.px-2.text-left(
        type="button"
        class="hover:bg-muted/40 py-1.5"
        :aria-expanded="isOpen(call.callId)"
        :title="isOpen(call.callId) ? $t('toolCalls.collapse') : $t('toolCalls.expand')"
        @click="toggle(call)"
        data-testid="tool-call-toggle"
      )
        .flex.min-w-0.items-center.gap-2
          ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
            :class="isOpen(call.callId) ? 'rotate-90' : ''"
          )
          component.text-muted-foreground.shrink-0(class="size-3.5" :is="icon(call)")
          span.shrink-0.font-mono.text-xs {{ call.name || "tool" }}
          span.text-muted-foreground.min-w-0.truncate.text-xs(
            v-if="target(call)"
            :title="target(call)"
          ) {{ target(call) }}
        .text-muted-foreground.flex.flex-wrap.items-center.gap-x-1.pl-5.text-xs
          span.inline-flex.items-center.rounded-full.border.px-2(
            class="py-0.5"
            :class="statusClass(call)"
          ) {{ statusLabel(call) }}
          span ·
          span.tabular-nums(
            v-if="call.interrupted"
            :title="$t('toolCall.noResult')"
          ) {{ $t("toolCalls.noDuration") }}
          span.tabular-nums(v-else) {{ durationLabel(call) }}
          template(v-if="call.startedAt")
            span ·
            span.tabular-nums(
              :title="formatInstant(timestampMs(call.startedAt))"
            ) {{ formatClockPrecise(timestampMs(call.startedAt)) }}
          template(v-if="call.resultBytes > 0n")
            span ·
            span.tabular-nums {{ $t("toolCalls.resultSize", { size: formatBytes(call.resultBytes) }) }}

      .flex.flex-col.border-t(v-if="isOpen(call.callId)")
        p.text-muted-foreground.px-2.py-1.text-xs(
          v-if="detailError(call.callId)"
        )
          | {{ detailError(call.callId) }}
        p.text-muted-foreground.px-2.py-1.text-xs(
          v-else-if="!detail(call.callId)"
        )
          | {{ $t("toolCalls.loadingDetail") }}
        template(v-else)
          ToolCallBody(:item="toolItem(call)")
          .flex.items-center.gap-2.border-t.px-2.py-1
            span.text-muted-foreground.text-xs(v-if="truncated(call.callId)")
              | {{ $t("toolCalls.truncatedHint") }}
            Button.ml-auto(
              type="button"
              variant="ghost"
              size="sm"
              :title="$t('toolCalls.downloadTitle')"
              @click="download(call)"
              :disabled="downloading.has(call.callId)"
              data-testid="tool-call-download"
            )
              DownloadIcon(data-icon="inline-start")
              span {{ downloading.has(call.callId) ? $t("toolCalls.downloading") : $t("toolCalls.download") }}

  .flex.items-center.justify-between.gap-2
    span.text-muted-foreground.text-xs
      | {{ shownLabel }}
    Button(
      v-if="calls.length < total"
      variant="outline"
      size="sm"
      @click="loadMore"
      :disabled="loading"
    ) {{ loading ? "Loading…" : "Load more" }}
</template>

<script setup lang="ts">
import ToolCallBody from "@/components/tasks/ToolCallBody.vue";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { timestampMs, type ToolItem } from "@/lib/console";
import {
  formatBytes,
  formatClockPrecise,
  formatElapsed,
  formatInstant,
  isoInstant,
} from "@/lib/format";
import { agency } from "@/lib/sdk";
import { toolView } from "@/lib/toolPresentation";
import type { TaskToolCall, TaskToolCallDetail } from "@cherrypi/sdk";
import { ToolPhase } from "@cherrypi/sdk";
import {
  ChevronRightIcon,
  DownloadIcon,
  FilePlusIcon,
  FileTextIcon,
  FilterIcon,
  PencilIcon,
  SquareTerminalIcon,
  WrenchIcon,
  type LucideIcon,
} from "@lucide/vue";
import {
  computed,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from "vue";

const props = defineProps<{
  taskId: string;

  names?: string[];
}>();

const emit = defineEmits<{
  (e: "update:names", value: string[]): void;
}>();

const PAGE_SIZE = 50;

const MAX_RETAINED_DETAILS = 20;

const calls = ref<TaskToolCall[]>([]);
const names = ref<string[]>([]);
const total = ref(0);
const loading = ref(false);
const error = ref("");

const search = ref("");
const status = ref<"all" | "error" | "interrupted">("all");

const selectedNames = computed({
  get: () => props.names ?? [],
  set: (value: string[]) => emit("update:names", value),
});

const open = reactive(new Set<string>());
const details = reactive(new Map<string, TaskToolCallDetail>());
const detailErrors = reactive(new Map<string, string>());
const downloading = reactive(new Set<string>());

let generation = 0;

const namesLabel = computed(() => {
  const count = selectedNames.value.length;
  if (!count) return "All tools";
  if (count === 1) return selectedNames.value[0] ?? "1 tool";
  return `${count} tools`;
});

const shownLabel = computed(() => {
  if (!total.value) return "";
  return `${calls.value.length} of ${total.value} call${total.value === 1 ? "" : "s"}`;
});

function setStatus(value: unknown) {
  status.value = value === "error" || value === "interrupted" ? value : "all";
}

function toggleName(name: string) {
  selectedNames.value = selectedNames.value.includes(name)
    ? selectedNames.value.filter((item) => item !== name)
    : [...selectedNames.value, name];
}

function clearNames() {
  selectedNames.value = [];
}

async function fetchCalls(offset: number) {
  const gen = ++generation;
  loading.value = true;
  error.value = "";
  try {
    const result = await agency.listTaskToolCalls(props.taskId, {
      toolNames: selectedNames.value,
      errorsOnly: status.value === "error",
      interruptedOnly: status.value === "interrupted",
      search: search.value.trim(),
      limit: PAGE_SIZE,
      offset,
    });
    if (gen !== generation) return;
    calls.value =
      offset === 0 ? result.calls : [...calls.value, ...result.calls];
    names.value = result.names;
    total.value = result.total;
  } catch (err) {
    if (gen !== generation) return;
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    if (gen === generation) loading.value = false;
  }
}

function loadMore() {
  void fetchCalls(calls.value.length);
}

function isOpen(callId: string): boolean {
  return open.has(callId);
}

function toggle(call: TaskToolCall) {
  if (open.has(call.callId)) {
    open.delete(call.callId);
    return;
  }
  open.add(call.callId);
  void loadDetail(call.callId);
}

async function loadDetail(callId: string) {
  if (details.has(callId) || detailErrors.has(callId)) return;
  try {
    rememberDetail(callId, await agency.getTaskToolCall(props.taskId, callId));
  } catch (err) {
    detailErrors.set(callId, err instanceof Error ? err.message : String(err));
  }
}

function rememberDetail(callId: string, item: TaskToolCallDetail) {
  details.set(callId, item);
  while (details.size > MAX_RETAINED_DETAILS) {
    const stale = [...details.keys()].find((key) => !open.has(key));
    if (stale === undefined) return;
    details.delete(stale);
  }
}

function detail(callId: string): TaskToolCallDetail | undefined {
  return details.get(callId);
}

function detailError(callId: string): string {
  return detailErrors.get(callId) ?? "";
}

function truncated(callId: string): boolean {
  const item = details.get(callId);
  return Boolean(
    item?.argsTruncated || item?.resultTruncated || item?.detailsTruncated,
  );
}

function toolItem(call: TaskToolCall): ToolItem {
  const item = detail(call.callId)!;
  return {
    id: 0,
    kind: "tool",
    callId: item.callId || call.callId,
    name: item.name || call.name,
    phase: ToolPhase.FINISHED,
    args: item.argsJson,
    result: item.resultJson,
    details: item.detailsJson,
    isError: item.isError,
    interrupted: item.interrupted,
    startedAt: timestampMs(item.startedAt) ?? timestampMs(call.startedAt),
    endedAt: timestampMs(item.endedAt) ?? timestampMs(call.endedAt),
  };
}

/** A call's one-line target, derived from the arguments preview. A generic
 * tool falls back to its own name, which the row already shows, so that case
 * renders no target at all. */
function target(call: TaskToolCall): string {
  const view = toolView({
    name: call.name,
    args: call.argsJson,
    result: "",
    details: "",
  });
  return view.target === call.name ? "" : view.target;
}

const ICONS: Record<string, LucideIcon> = {
  bash: SquareTerminalIcon,
  read: FileTextIcon,
  write: FilePlusIcon,
  edit: PencilIcon,
  generic: WrenchIcon,
};

function icon(call: TaskToolCall): LucideIcon {
  return ICONS[
    toolView({
      name: call.name,
      args: call.argsJson,
      result: "",
      details: "",
    }).icon
  ]!;
}

function durationLabel(call: TaskToolCall): string {
  return formatElapsed(call.durationMs);
}

function statusLabel(call: TaskToolCall): string {
  if (call.interrupted) return "interrupted";
  if (call.isError) return "error";
  return "ok";
}

function statusClass(call: TaskToolCall): string {
  if (call.interrupted)
    return "border-amber-500/40 text-amber-600 dark:text-amber-400";
  if (call.isError) return "border-destructive/40 text-destructive";
  return "text-muted-foreground";
}

async function download(call: TaskToolCall) {
  if (downloading.has(call.callId)) return;
  downloading.add(call.callId);
  try {
    const item = await agency.getTaskToolCall(props.taskId, call.callId, {
      full: true,
    });
    const payload = {
      call_id: item.callId,
      name: item.name,
      is_error: item.isError,
      interrupted: item.interrupted,
      started_at: isoInstant(timestampMs(item.startedAt)) || null,
      ended_at: isoInstant(timestampMs(item.endedAt)) || null,
      duration_ms: Number(item.durationMs),
      args: parseJSON(item.argsJson),
      result: item.resultJson,
      details: parseJSON(item.detailsJson),
    };
    const blob = new Blob([JSON.stringify(payload, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${item.name || "tool"}-${item.callId}.json`;
    link.click();

    setTimeout(() => URL.revokeObjectURL(url), 0);
  } catch (err) {
    detailErrors.set(
      call.callId,
      err instanceof Error ? err.message : String(err),
    );
  } finally {
    downloading.delete(call.callId);
  }
}

function parseJSON(raw: string): unknown {
  if (!raw.trim()) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return raw;
  }
}

let searchTimer: ReturnType<typeof setTimeout> | undefined;
watch(search, () => {
  if (searchTimer !== undefined) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => void fetchCalls(0), 250);
});
watch([status, selectedNames], () => void fetchCalls(0));

onMounted(() => void fetchCalls(0));
onBeforeUnmount(() => {
  generation += 1;
  if (searchTimer !== undefined) clearTimeout(searchTimer);
});
</script>

<i18n lang="yaml">
en:
  toolCalls:
    allResults: "All results"
    errors: "Errors"
    interrupted: "Interrupted"
    clearFilter: "Clear filter"
    loading: "Loading tool calls…"
    noDuration: "no duration"
    searchPlaceholder: "Search tool calls"
    filterByResult: "Filter by result"
    statusInterrupted: "interrupted"
    statusError: "error"
    collapse: "Collapse this call"
    expand: "Show this call’s input and output"
    loadingDetail: "Loading input and output…"
    truncatedHint: "The agency cut the longest bodies short; download the call for the whole input and output."
    downloadTitle: "Download this call as JSON, with the whole input and output"
    download: "Download"
    downloading: "Downloading…"
    resultSize: "{size} result"
ru:
  toolCalls:
    allResults: "Все результаты"
    errors: "Ошибки"
    interrupted: "Прервано"
    clearFilter: "Сбросить фильтр"
    loading: "Загружаем вызовы инструментов…"
    noDuration: "без длительности"
    searchPlaceholder: "Поиск вызовов инструментов"
    filterByResult: "Фильтр по результату"
    statusInterrupted: "прервано"
    statusError: "ошибка"
    collapse: "Свернуть вызов"
    expand: "Показать входные и выходные данные вызова"
    loadingDetail: "Загружаем входные и выходные данные…"
    truncatedHint: "Агентство сократило самые длинные тела вызова; скачайте вызов, чтобы увидеть полные входные и выходные данные."
    downloadTitle: "Скачать вызов в JSON с полными входными и выходными данными"
    download: "Скачать"
    downloading: "Скачивание…"
    resultSize: "результат {size}"
</i18n>
