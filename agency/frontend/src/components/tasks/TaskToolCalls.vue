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
    placeholder="Search tool calls"
    aria-label="Search tool calls"
    data-testid="tool-calls-search"
  )
  .flex.items-center.gap-2
    Select.flex-1(
      :model-value="status"
      @update:model-value="setStatus($event)"
    )
      SelectTrigger.w-full(size="sm" title="Filter by result")
        SelectValue
      SelectContent
        SelectGroup
          SelectItem(value="all") All results
          SelectItem(value="error") Errors
          SelectItem(value="interrupted") Interrupted
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
        DropdownMenuLabel Tools
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
          DropdownMenuItem(@select="clearNames") Clear filter

  p.text-muted-foreground.text-xs(v-if="error") {{ error }}
  p.text-muted-foreground.text-xs(v-else-if="loading && !calls.length")
    | Loading tool calls…
  p.text-muted-foreground.text-xs(v-else-if="!calls.length")
    | {{ total ? "No tool calls match these filters." : "No tool calls recorded yet." }}

  .flex.flex-col.overflow-hidden.rounded-md.border(v-if="calls.length")
    .flex.flex-col.border-b.px-2(
      v-for="call in calls"
      :key="call.callId"
      class="py-1.5 last:border-b-0"
      data-testid="tool-call-row"
    )
      .flex.min-w-0.items-center.gap-2
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
          title="The call's run ended before it recorded a result"
        ) no duration
        span.tabular-nums(v-else) {{ formatDuration(call.durationMs) }}
        template(v-if="call.startedAt")
          span ·
          span.tabular-nums(
            :title="formatInstant(timestampMs(call.startedAt))"
          ) {{ formatClockPrecise(timestampMs(call.startedAt)) }}
        template(v-if="call.resultBytes > 0n")
          span ·
          span.tabular-nums {{ formatBytes(call.resultBytes) }} result

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
import { timestampMs } from "@/lib/console";
import {
  formatBytes,
  formatClockPrecise,
  formatDuration,
  formatInstant,
} from "@/lib/format";
import { agency } from "@/lib/sdk";
import { toolView } from "@/lib/toolPresentation";
import type { TaskToolCall } from "@cherrypi/sdk";
import {
  FilePlusIcon,
  FileTextIcon,
  FilterIcon,
  PencilIcon,
  SquareTerminalIcon,
  WrenchIcon,
  type LucideIcon,
} from "@lucide/vue";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

const props = defineProps<{ taskId: string }>();

const PAGE_SIZE = 50;

const calls = ref<TaskToolCall[]>([]);
const names = ref<string[]>([]);
const total = ref(0);
const loading = ref(false);
const error = ref("");

const search = ref("");
const status = ref<"all" | "error" | "interrupted">("all");
const selectedNames = ref<string[]>([]);

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
