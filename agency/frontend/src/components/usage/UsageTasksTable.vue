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
Card
  CardHeader
    .flex.flex-wrap.items-start.justify-between.gap-3
      div
        CardTitle {{ $t("usage.tasks.title") }}
        CardDescription {{ $t("usage.tasks.hint") }}
      .flex.flex-wrap.items-center.gap-2
        .bg-muted.flex.rounded-lg.p-1
          button.text-11.rounded-md.px-2.py-1.font-medium(
            v-for="option in STATE_FILTERS"
            :key="option.value"
            :class="stateFilter === option.value ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
            @click="stateFilter = option.value"
          ) {{ option.label }}
        Input.h-8.w-44(
          v-model="search"
          type="search"
          :placeholder="$t('usage.tasks.searchPlaceholder')"
        )
  CardContent.p-0
    Table
      TableHeader
        TableRow
          TableHead {{ $t("taskView.task") }}
          TableHead(v-if="context !== 'agent'") {{ $t("admin.agents.columnAgent") }}
          TableHead(v-if="context !== 'project'") {{ $t("admin.projects.columnProject") }}
          TableHead {{ $t("domainLog.kind.tags") }}
          TableHead {{ $t("domainLog.kind.state") }}
          TableHead.cursor-pointer.text-right.select-none(
            @click="sortBy('runs')"
          ) {{ $t("usage.tasks.runs") }}
          TableHead.cursor-pointer.text-right.select-none(
            @click="sortBy('tokens')"
          ) {{ $t("account.tabs.tokens") }}
          TableHead.cursor-pointer.text-right.select-none(
            :title="$t('usage.tasks.ttcTitle')"
            @click="sortBy('ttc')"
          ) {{ $t("usage.breakdown.ttc") }}
          TableHead.cursor-pointer.text-right.select-none(
            @click="sortBy('active')"
          ) {{ $t("admin.users.statusActive") }}
          TableHead.cursor-pointer.text-right.select-none(
            :title="$t('usage.tasks.thinkingTitle')"
            @click="sortBy('thinking')"
          ) {{ $t("usage.insights.thinking") }}
          TableHead.cursor-pointer.text-right.select-none(
            @click="sortBy('cost')"
          ) {{ $t("usage.graph.cost") }}
          TableHead
      TableBody
        TableRow(
          v-for="row in visible"
          :key="row.taskId"
          :class="row.task ? 'cursor-pointer' : ''"
          @click="open(row)"
        )
          TableCell
            .flex.flex-col.gap-1
              .flex.items-center.gap-1
                span.font-medium {{ row.name }}
                span.text-4xs.text-muted-foreground.rounded.border.px-1(
                  v-if="row.archived"
                )
                  | {{ $t("usage.tasks.archived") }}
              span.text-3xs.text-muted-foreground.font-mono
                | {{ row.task?.id ?? row.taskId }}
          TableCell(v-if="context !== 'agent'")
            .flex.items-center.gap-1
              span.bg-muted.text-8.flex.size-4.items-center.justify-center.rounded.font-bold(
                :style="{ color: usageAccent(row.agentId) }"
              ) {{ row.agentName.slice(0, 1) }}
              span {{ row.agentName }}
          TableCell(v-if="context !== 'project'")
            .flex.items-center.gap-1
              span.bg-muted.text-8.flex.size-4.items-center.justify-center.rounded.font-bold(
                :style="{ color: usageAccent(row.projectId) }"
              ) {{ row.projectName.slice(0, 1) }}
              span {{ row.projectName }}
          TableCell
            .flex.flex-wrap.gap-1
              span.text-3xs.rounded.px-1.font-medium(
                v-for="tag in row.tags.slice(0, 2)"
                :key="tag.id"
                :style="{ background: tag.color + '1a', color: tag.color }"
              ) {{ tag.name }}
              span.text-3xs.text-muted-foreground(v-if="row.tags.length > 2")
                | +{{ row.tags.length - 2 }}
              span.text-10.text-muted-foreground(v-if="!row.tags.length") —
          TableCell
            Badge(variant="outline" :class="taskStateClass(row.state)")
              | {{ $t(taskStateKey(row.state)) }}
          TableCell.text-right.tabular-nums {{ row.runs || "—" }}
          TableCell.text-right.tabular-nums {{ formatTokens(row.inputTokens + row.outputTokens) }}
          TableCell.text-right.tabular-nums {{ row.task ? formatDuration(ttcOf(row)) : "—" }}
          TableCell.text-right.tabular-nums {{ formatDuration(row.activeMs) }}
          TableCell.text-right.tabular-nums {{ formatDuration(row.thinkingMs) }}
          TableCell.text-right.font-semibold.tabular-nums {{ formatCost(row.cost) }}
          TableCell
            .flex.justify-end
              span.text-muted-foreground.flex.size-5.items-center.justify-center.rounded(
                v-if="row.task"
              )
                svg.size-3(
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                )
                  path(d="M7 17 17 7M9 7h8v8")
        TableRow(v-if="!visible.length")
          TableCell.text-muted-foreground.py-8.text-center(
            :colspan="columnCount"
          )
            | {{ $t("usage.tasks.noMatch") }}
  CardFooter.justify-between.gap-3
    span.text-2xs.text-muted-foreground {{ $t("usage.tasks.showing", { visible: visible.length, total: filtered.length }) }}
    span.text-2xs.text-muted-foreground {{ $t("usage.tasks.clamped") }}
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type {
  UsageContext,
  UsageTaskRow,
} from "@/composables/useUsageAnalytics";
import { formatCost, formatDuration, formatTokens } from "@/lib/format";
import { taskStateClass, taskStateKey } from "@/lib/task";
import { usageAccent } from "@/lib/usage";
import { TaskState } from "@cherrypi/sdk";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

const { t } = useI18n();

const props = defineProps<{
  context: UsageContext;
  rows: UsageTaskRow[];
}>();

type StateFilter = "all" | "completed" | "active";
const STATE_FILTERS = computed<{ value: StateFilter; label: string }[]>(() => [
  { value: "all", label: t("common.all") },
  { value: "completed", label: t("usage.state.completed") },
  { value: "active", label: t("usage.state.active") },
]);

const router = useRouter();
const columnCount = computed(() => 11 + (props.context === "agency" ? 1 : 0));
const stateFilter = ref<StateFilter>("all");
const search = ref("");
const sortKey = ref<"runs" | "tokens" | "active" | "thinking" | "ttc" | "cost">(
  "runs",
);
const sortDir = ref<"asc" | "desc">("desc");

const filtered = computed(() => {
  let rows = props.rows;
  if (stateFilter.value === "completed") {
    rows = rows.filter((row) => row.state === TaskState.COMPLETED);
  } else if (stateFilter.value === "active") {
    rows = rows.filter(
      (row) =>
        row.state === TaskState.RUNNING ||
        row.state === TaskState.QUEUED ||
        row.state === TaskState.INPUT ||
        row.state === TaskState.SETTLED,
    );
  }
  const query = search.value.trim().toLowerCase();
  if (query) {
    rows = rows.filter(
      (row) =>
        row.name.toLowerCase().includes(query) ||
        row.task?.id.toLowerCase().includes(query) ||
        row.agentName.toLowerCase().includes(query) ||
        row.projectName.toLowerCase().includes(query) ||
        row.tags.some((tag) => tag.name.toLowerCase().includes(query)),
    );
  }
  return [...rows].sort((a, b) => {
    const value = (row: UsageTaskRow) => {
      switch (sortKey.value) {
        case "runs":
          return row.runs;
        case "tokens":
          return row.inputTokens + row.outputTokens;
        case "active":
          return row.activeMs;
        case "thinking":
          return row.thinkingMs;
        case "ttc":
          return ttcOf(row);
        default:
          return row.cost;
      }
    };
    const delta = value(a) - value(b);
    return sortDir.value === "desc" ? -delta : delta;
  });
});

const visible = computed(() => filtered.value.slice(0, 20));

function ttcOf(row: UsageTaskRow): number {
  return Number(row.task?.ttcMs ?? 0);
}

function sortBy(
  key: "runs" | "tokens" | "active" | "thinking" | "ttc" | "cost",
) {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === "desc" ? "asc" : "desc";
  } else {
    sortKey.value = key;
    sortDir.value = "desc";
  }
}

function open(row: UsageTaskRow) {
  if (!row.task) return;
  void router.push({ name: "task", params: { id: row.task.id } });
}
</script>

<i18n lang="yaml">
en:
  usage:
    tasks:
      title: "Task breakdown"
      hint: "Range-scoped totals. Click a row to open the task."
      runs: "Runs"
      archived: "archived"
      noMatch: "No tasks match these filters."
      clamped: "Task costs are clamped to the selected time range."
      searchPlaceholder: "Search tasks"
      ttcTitle: "Time from the first queue to completion"
      thinkingTitle: "The model's own working time: active time minus tool calls"
      showing: "Showing {visible} of {total} tasks"
    state:
      completed: "Completed"
      active: "Active"
ru:
  usage:
    tasks:
      title: "Разбивка по задачам"
      hint: "Итоги за диапазон. Нажмите на строку, чтобы открыть задачу."
      runs: "Запуски"
      archived: "архив"
      noMatch: "Нет задач, подходящих под фильтры."
      clamped: "Стоимость задач ограничена выбранным диапазоном."
      searchPlaceholder: "Поиск задач"
      ttcTitle: "Время от первой постановки в очередь до завершения"
      thinkingTitle: "Собственное рабочее время модели: активное время минус вызовы инструментов"
      showing: "Показано {visible} из {total} задач"
    state:
      completed: "Завершённые"
      active: "Активные"
</i18n>
