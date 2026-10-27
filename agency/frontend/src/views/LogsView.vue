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
.flex.flex-col.gap-4
  p.text-muted-foreground.text-sm
    | Operational records this agent pushed to the agency — task/session
    | lifecycle, connectivity and the terminal lifecycle events the agency
    | observed — in one feed.

  .flex.flex-wrap.items-center.gap-2
    .relative.min-w-48.flex-1
      SearchIcon.text-muted-foreground.absolute.size-4(
        class="top-1/2 left-2.5 -translate-y-1/2"
      )
      Input.pl-8(
        v-model="search"
        placeholder="Filter log lines…"
        aria-label="Filter log lines"
      )
    Select(v-model="levelFilter")
      SelectTrigger.w-40
        SelectValue
      SelectContent
        SelectItem(
          v-for="opt in levelOptions"
          :key="opt.value"
          :value="opt.value"
        ) {{ opt.label }}
    Button(
      variant="outline"
      size="sm"
      :class="filtersOpen ? 'bg-muted text-foreground' : ''"
      @click="filtersOpen = !filtersOpen"
    )
      SlidersHorizontalIcon(data-icon="inline-start")
      span Filters
    Button(
      variant="outline"
      size="sm"
      @click="refreshAgentLogs"
      :disabled="logsLoading"
    )
      RefreshCwIcon(data-icon="inline-start")
      span Refresh

  .rounded-lg.border.p-3(v-if="filtersOpen" class="bg-muted/30")
    .flex.flex-wrap.items-center.gap-3
      label.flex.cursor-pointer.items-center.gap-2.text-sm
        Switch(v-model="hideLifecycle" aria-label="Hide routine lifecycle")
        span Hide routine lifecycle
      .min-w-64.flex-1
        Input(
          v-model="hiddenText"
          placeholder="pi started, pi exited, pi stopped"
          aria-label="Hidden message patterns"
        )
    p.text-muted-foreground.mt-2.text-xs
      | Comma-separated message substrings to hide while the toggle is on.
      | {{ hiddenCount ? `${hiddenCount} hidden now.` : "" }}

  Card
    CardContent.p-0
      p.text-destructive.px-4.py-2.text-sm(v-if="logsError")
        | {{ logsError }}
      p.text-muted-foreground.px-4.py-2.text-sm(v-if="terminalsError")
        | Terminal events unavailable: {{ terminalsError }}
      TableRowsSkeleton(
        v-if="(logsLoading || terminalsLoading) && !rows.length"
        :widths="['h-5 w-32', 'w-16', 'w-64']"
      )
      p.text-muted-foreground.px-4.py-6.text-sm(
        v-if="!rows.length && !logsLoading && !terminalsLoading"
      )
        | {{ hasFilters ? "No log lines match the filters." : "No log lines yet." }}
      .divide-border.divide-y
        template(v-for="row in rows" :key="rowKey(row)")
          .flex.items-start.gap-3.px-4.py-2(v-if="row.kind === 'agent'")
            span.text-muted-foreground.shrink-0.font-mono.text-xs.whitespace-nowrap(
              class="pt-0.5"
            ) {{ formatTimestamp(row.log.at) }}
            span.inline-flex.shrink-0.items-center.rounded-full.border.px-2.text-xs.font-medium(
              class="py-0.5"
              :class="levelClass(row.log.level)"
            ) {{ levelLabel(row.log.level) }}
            .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
              span.text-sm.break-words {{ row.log.message }}
              .flex.flex-wrap.items-center.gap-x-3(
                v-if="row.log.taskId || row.log.sessionId || attrs(row.log).length"
                class="gap-y-0.5"
              )
                RouterLink.text-xs(
                  v-if="row.log.taskId"
                  class="hover:underline"
                  :to="{ name: 'task', params: { id: row.log.taskId } }"
                ) {{ row.log.taskId }}
                span.text-muted-foreground.truncate.font-mono.text-xs(
                  v-if="row.log.sessionId"
                  :title="row.log.sessionId"
                ) session {{ row.log.sessionId.slice(0, 8) }}
                span.text-muted-foreground.font-mono.text-xs(
                  v-for="attr in attrs(row.log)"
                  :key="attr.key"
                  :title="attr.value"
                ) {{ attr.key }}={{ attr.value }}
          .flex.items-start.gap-3.px-4.py-2(v-else)
            span.text-muted-foreground.shrink-0.font-mono.text-xs.whitespace-nowrap(
              class="pt-0.5"
            ) {{ formatTimestamp(row.log.at) }}
            span.inline-flex.shrink-0.items-center.rounded-full.border.px-2.text-xs.font-medium(
              class="py-0.5"
              :class="eventClass(row.log.event)"
            ) {{ row.log.event }}
            .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
              span.text-sm.break-words {{ row.log.reason || "—" }}
              span.text-muted-foreground.truncate.font-mono.text-xs(
                v-if="row.log.detail"
                :title="row.log.detail"
              ) {{ row.log.detail }}
            .flex.shrink-0.items-center
              Button(
                v-if="row.log.event === 'interrupted'"
                variant="outline"
                size="sm"
                @click="restore(row.log)"
                :disabled="restoring === row.log.id"
              )
                RotateCcwIcon(data-icon="inline-start")
                span Restore
</template>

<script setup lang="ts">
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { useRealtime } from "@/composables/useRealtime";
import {
  DEFAULT_HIDDEN_LOG_PATTERNS,
  isHiddenAgentLog,
  mergeLogRows,
  type MergedLogRow,
} from "@/lib/agentLogs";
import { formatTimestamp } from "@/lib/format";
import { agent } from "@/lib/sdk";
import { AgentLogLevel, type AgentLog, type TerminalLog } from "@cherrypi/sdk";
import {
  RefreshCwIcon,
  RotateCcwIcon,
  SearchIcon,
  SlidersHorizontalIcon,
} from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const HIDDEN_PATTERNS_KEY = "cherrypi.agent-logs.hidden-patterns";
const HIDE_LIFECYCLE_KEY = "cherrypi.agent-logs.hide-lifecycle";

function storedString(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function store(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {}
}

const logs = ref<AgentLog[]>([]);
const logsLoading = ref(true);
const logsError = ref("");
const levelFilter = ref<"all" | "info" | "warn" | "error">("all");

const terminalLogs = ref<TerminalLog[]>([]);
const terminalsLoading = ref(true);
const terminalsError = ref("");
const restoring = ref<bigint | null>(null);

const search = ref("");
const filtersOpen = ref(false);
const hiddenText = ref(
  storedString(HIDDEN_PATTERNS_KEY) ?? DEFAULT_HIDDEN_LOG_PATTERNS.join(", "),
);
const hideLifecycle = ref(storedString(HIDE_LIFECYCLE_KEY) !== "0");

const levelOptions = [
  { value: "all", label: "All levels" },
  { value: "info", label: "Info and up" },
  { value: "warn", label: "Warnings and up" },
  { value: "error", label: "Errors only" },
] as const;

function minLevel(): AgentLogLevel {
  switch (levelFilter.value) {
    case "info":
      return AgentLogLevel.INFO;
    case "warn":
      return AgentLogLevel.WARN;
    case "error":
      return AgentLogLevel.ERROR;
    default:
      return AgentLogLevel.UNSPECIFIED;
  }
}

const hiddenPatterns = computed(() =>
  hiddenText.value
    .split(",")
    .map((pattern) => pattern.trim())
    .filter(Boolean),
);

const rows = computed(() =>
  mergeLogRows(logs.value, terminalLogs.value, {
    search: search.value,
    hiddenPatterns: hideLifecycle.value ? hiddenPatterns.value : [],
  }),
);

const hiddenCount = computed(() =>
  hideLifecycle.value
    ? logs.value.filter((log) => isHiddenAgentLog(log, hiddenPatterns.value))
        .length
    : 0,
);

const hasFilters = computed(
  () =>
    search.value.trim() !== "" ||
    (hideLifecycle.value && hiddenPatterns.value.length > 0),
);

watch(hiddenText, (value) => store(HIDDEN_PATTERNS_KEY, value));
watch(hideLifecycle, (value) => store(HIDE_LIFECYCLE_KEY, value ? "1" : "0"));

const LEVEL_LABELS: Record<number, string> = {
  [AgentLogLevel.DEBUG]: "debug",
  [AgentLogLevel.INFO]: "info",
  [AgentLogLevel.WARN]: "warn",
  [AgentLogLevel.ERROR]: "error",
};

function levelLabel(level: AgentLogLevel): string {
  return LEVEL_LABELS[level] ?? "info";
}

function levelClass(level: AgentLogLevel): string {
  switch (level) {
    case AgentLogLevel.ERROR:
      return "border-destructive/40 bg-destructive/10 text-destructive";
    case AgentLogLevel.WARN:
      return "border-amber-500/35 bg-amber-500/12 text-amber-600 dark:text-amber-400";
    case AgentLogLevel.DEBUG:
      return "border-border bg-muted/60 text-muted-foreground";
    default:
      return "border-sky-500/30 bg-sky-500/10 text-sky-600 dark:text-sky-400";
  }
}

function attrs(log: AgentLog): { key: string; value: string }[] {
  if (!log.attrsJson) return [];
  try {
    const obj = JSON.parse(log.attrsJson) as Record<string, unknown>;
    return Object.entries(obj).map(([key, value]) => ({
      key,
      value: typeof value === "string" ? value : JSON.stringify(value),
    }));
  } catch {
    return [];
  }
}

async function refreshAgentLogs() {
  try {
    logs.value = await agent.listAgentLogs({
      minLevel: minLevel(),
      limit: 200,
    });
    logsError.value = "";
  } catch (err) {
    logsError.value = err instanceof Error ? err.message : String(err);
  } finally {
    logsLoading.value = false;
  }
}

async function refreshTerminals() {
  try {
    terminalLogs.value = await agent.listTerminalLogs(200);
    terminalsError.value = "";
  } catch (err) {
    terminalsError.value = err instanceof Error ? err.message : String(err);
  } finally {
    terminalsLoading.value = false;
  }
}

function eventClass(event: string): string {
  switch (event) {
    case "interrupted":
      return "border-amber-500/35 bg-amber-500/12 text-amber-600 dark:text-amber-400";
    case "created":
      return "border-emerald-500/32 bg-emerald-500/12 text-emerald-600 dark:text-emerald-400";
    default:
      return "border-border bg-muted text-muted-foreground";
  }
}

function rowKey(row: MergedLogRow): string {
  return `${row.kind}-${row.log.id}`;
}

async function restore(log: TerminalLog) {
  restoring.value = log.id;
  try {
    await agent.createTerminal(log.kind);
    toast.success("Terminal restored");
    await refreshTerminals();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    restoring.value = null;
  }
}

watch(levelFilter, () => void refreshAgentLogs());

onMounted(() => {
  void refreshAgentLogs();
  void refreshTerminals();
});

useRealtime(["agent-logs", "terminals"], () => {
  void refreshAgentLogs();
  void refreshTerminals();
});
</script>
