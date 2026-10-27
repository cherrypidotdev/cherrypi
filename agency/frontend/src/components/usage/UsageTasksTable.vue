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
        CardTitle Task breakdown
        CardDescription Range-scoped totals. Click a row to open the task.
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
          placeholder="Search tasks"
        )
  CardContent.p-0
    Table
      TableHeader
        TableRow
          TableHead Task
          TableHead {{ context === "project" ? "Agent" : "Project" }}
          TableHead Tags
          TableHead State
          TableHead.cursor-pointer.text-right.select-none(
            @click="sortBy('runs')"
          ) Runs
          TableHead.cursor-pointer.text-right.select-none(
            @click="sortBy('tokens')"
          ) Tokens
          TableHead.cursor-pointer.text-right.select-none(
            @click="sortBy('active')"
          ) Active
          TableHead.cursor-pointer.text-right.select-none(
            @click="sortBy('cost')"
          ) Cost
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
                  | archived
              span.text-3xs.text-muted-foreground.font-mono
                | {{ row.task?.id ?? row.taskId }}
          TableCell
            .flex.items-center.gap-1
              span.bg-muted.text-8.flex.size-4.items-center.justify-center.rounded.font-bold(
                :style="{ color: usageAccent(row.agentId || row.projectId) }"
              ) {{ (context === "project" ? row.agentName : row.projectName).slice(0, 1) }}
              span {{ context === "project" ? row.agentName : row.projectName }}
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
              | {{ taskStateLabel(row.state) }}
          TableCell.text-right.tabular-nums {{ row.runs || "—" }}
          TableCell.text-right.tabular-nums {{ formatTokens(row.inputTokens + row.outputTokens) }}
          TableCell.text-right.tabular-nums {{ formatDuration(row.activeMs) }}
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
          TableCell.text-muted-foreground.py-8.text-center(colspan="9")
            | No tasks match these filters.
  CardFooter.justify-between.gap-3
    span.text-2xs.text-muted-foreground Showing {{ visible.length }} of {{ filtered.length }} tasks
    span.text-2xs.text-muted-foreground Task costs are clamped to the selected time range.
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
import { taskStateClass, taskStateLabel } from "@/lib/task";
import { usageAccent } from "@/lib/usage";
import { TaskState } from "@cherrypi/sdk";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";

const props = defineProps<{
  context: UsageContext;
  rows: UsageTaskRow[];
}>();

type StateFilter = "all" | "completed" | "active";
const STATE_FILTERS: { value: StateFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "completed", label: "Completed" },
  { value: "active", label: "Active" },
];

const router = useRouter();
const stateFilter = ref<StateFilter>("all");
const search = ref("");
const sortKey = ref<"runs" | "tokens" | "active" | "cost">("runs");
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
    const value = (row: UsageTaskRow) =>
      sortKey.value === "runs"
        ? row.runs
        : sortKey.value === "tokens"
          ? row.inputTokens + row.outputTokens
          : sortKey.value === "active"
            ? row.activeMs
            : row.cost;
    const delta = value(a) - value(b);
    return sortDir.value === "desc" ? -delta : delta;
  });
});

const visible = computed(() => filtered.value.slice(0, 20));

function sortBy(key: "runs" | "tokens" | "active" | "cost") {
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
