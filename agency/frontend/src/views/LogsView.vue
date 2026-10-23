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
.flex.h-full.flex-col.overflow-hidden
  AgentHeader(v-if="!embedded")
  .flex-1.overflow-auto
    .mx-auto.flex.w-full.max-w-6xl.flex-col.gap-4.p-6
      p.text-muted-foreground.text-sm
        | Operational records this agent pushed to the agency — task/session
        | lifecycle and connectivity — plus the terminal lifecycle events the
        | agency observed.

      Tabs(v-model="tab")
        .flex.flex-wrap.items-center.justify-between.gap-2
          TabsList
            TabsTrigger(value="agent") Agent
            TabsTrigger(value="terminals") Terminals
          .flex.items-center.gap-2(v-if="tab === 'agent'")
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
              @click="refreshAgentLogs"
              :disabled="logsLoading"
            )
              RefreshCwIcon(data-icon="inline-start")
              span Refresh

        TabsContent(value="agent")
          Card
            CardContent.p-0
              p.text-destructive.px-4.py-2.text-sm(v-if="logsError")
                | {{ logsError }}
              p.text-muted-foreground.px-4.py-6.text-sm(
                v-if="!logs.length && !logsLoading"
              )
                | No agent log lines yet.
              .divide-border.divide-y
                .flex.items-start.gap-3.px-4.py-2(
                  v-for="log in logs"
                  :key="String(log.id)"
                )
                  span.text-muted-foreground.shrink-0.font-mono.text-xs.whitespace-nowrap(
                    class="pt-0.5"
                  ) {{ formatTimestamp(log.at) }}
                  span.inline-flex.shrink-0.items-center.rounded-full.border.px-2.text-xs.font-medium(
                    class="py-0.5"
                    :class="levelClass(log.level)"
                  ) {{ levelLabel(log.level) }}
                  .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
                    span.text-sm.break-words {{ log.message }}
                    .flex.flex-wrap.items-center.gap-x-3(
                      v-if="log.taskId || log.sessionId || attrs(log).length"
                      class="gap-y-0.5"
                    )
                      RouterLink.text-xs(
                        v-if="log.taskId"
                        class="hover:underline"
                        :to="{ name: 'task', params: { id: log.taskId } }"
                      ) {{ log.taskId }}
                      span.text-muted-foreground.truncate.font-mono.text-xs(
                        v-if="log.sessionId"
                        :title="log.sessionId"
                      ) session {{ log.sessionId.slice(0, 8) }}
                      span.text-muted-foreground.font-mono.text-xs(
                        v-for="attr in attrs(log)"
                        :key="attr.key"
                        :title="attr.value"
                      ) {{ attr.key }}={{ attr.value }}

        TabsContent(value="terminals")
          Card
            CardContent.p-0
              p.text-destructive.px-4.py-2.text-sm(v-if="terminalsError")
                | {{ terminalsError }}
              Table
                TableHeader
                  TableRow
                    TableHead.w-44 Time
                    TableHead.w-28 Event
                    TableHead Reason / working directory
                    TableHead.w-24
                TableBody
                  TableRowsSkeleton(
                    v-if="terminalsLoading && !terminalLogs.length"
                    :widths="['h-5 w-32', 'w-20', 'w-56', 'w-12']"
                  )
                  TableRow(v-for="log in terminalLogs" :key="String(log.id)")
                    TableCell.text-muted-foreground.text-xs
                      | {{ formatTimestamp(log.at) }}
                    TableCell
                      span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
                        class="py-0.5"
                        :class="eventClass(log.event)"
                      ) {{ log.event }}
                    TableCell
                      .flex.min-w-0.flex-col(class="gap-0.5")
                        span.text-xs {{ log.reason || "—" }}
                        span.text-muted-foreground.truncate.font-mono.text-xs(
                          v-if="log.detail"
                          :title="log.detail"
                        ) {{ log.detail }}
                    TableCell
                      .flex.justify-end
                        Button(
                          v-if="log.event === 'interrupted'"
                          variant="outline"
                          size="sm"
                          @click="restore(log)"
                          :disabled="restoring === log.id"
                        )
                          RotateCcwIcon(data-icon="inline-start")
                          span Restore
                  TableRow(v-if="!terminalLogs.length && !terminalsLoading")
                    TableCell.text-muted-foreground(:colspan="4")
                      | No terminal events yet.
</template>

<script setup lang="ts">
import AgentHeader from "@/components/AgentHeader.vue";
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { setPageTitle } from "@/composables/usePageTitle";
import { formatTimestamp } from "@/lib/format";
import { agent } from "@/lib/sdk";
import { AgentLogLevel, type AgentLog, type TerminalLog } from "@cherrypi/sdk";
import { RefreshCwIcon, RotateCcwIcon } from "@lucide/vue";
import { onBeforeUnmount, onMounted, ref, watch, watchEffect } from "vue";
import { toast } from "vue3-toastify";

const props = withDefaults(defineProps<{ embedded?: boolean }>(), {
  embedded: false,
});

const tab = ref<"agent" | "terminals">("agent");

const logs = ref<AgentLog[]>([]);
const logsLoading = ref(true);
const logsError = ref("");
const levelFilter = ref<"all" | "info" | "warn" | "error">("all");

const terminalLogs = ref<TerminalLog[]>([]);
const terminalsLoading = ref(true);
const terminalsError = ref("");
const restoring = ref<bigint | null>(null);

let timer: number | undefined;

watchEffect(() => {
  if (!props.embedded) setPageTitle("Logs");
});

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

const LEVEL_LABELS: Record<number, string> = {
  [AgentLogLevel.DEBUG]: "debug",
  [AgentLogLevel.INFO]: "info",
  [AgentLogLevel.WARN]: "warn",
  [AgentLogLevel.ERROR]: "error",
};

function levelLabel(level: AgentLogLevel): string {
  return LEVEL_LABELS[level] ?? "info";
}

// A loss (error) is prominent; warn is amber; info is calm; debug recedes.
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

// The structured fields an agent attached to a record (task_id/session_id and
// the level/message are already columns).
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

// Event pill hues: a loss (interrupted) is prominent, an exit/close is muted.
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

// A terminal lost to an agent restart is offered back as a fresh root shell.
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
  timer = window.setInterval(() => {
    void refreshAgentLogs();
    void refreshTerminals();
  }, 5000);
});

onBeforeUnmount(() => {
  if (timer !== undefined) window.clearInterval(timer);
});
</script>
