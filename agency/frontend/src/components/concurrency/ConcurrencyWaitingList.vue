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
  CardHeader.pb-2
    .flex.items-center.justify-between.gap-2
      CardTitle {{ title }}
      span.text-3xs.rounded-full.font-medium(
        :class="waiting.length ? 'bg-amber-500/12 px-2 py-0.5 text-amber-600 dark:text-amber-400' : 'text-muted-foreground'"
      ) {{ waiting.length }}
    CardDescription {{ description }}
  CardContent.p-2.pt-0
    p.text-muted-foreground.px-2.text-xs(v-if="!waiting.length") {{ empty }}
    .flex.flex-col(v-else class="gap-0.5")
      RouterLink.flex.flex-col.gap-1.rounded-md.px-2.py-2.transition-colors(
        v-for="item in waiting"
        :key="item.task.id"
        class="hover:bg-muted/60"
        :to="{ name: 'task', params: { id: item.task.id } }"
      )
        .flex.flex-wrap.items-center(class="gap-1.5")
          span.text-3xs.text-muted-foreground.font-mono {{ item.task.id }}
          span.text-3xs.text-muted-foreground(v-if="showAgent && item.agent") @{{ item.agent.username }}
          span.text-3xs.text-muted-foreground(
            v-else-if="showAgent && item.reason"
          ) unpinned
          span.text-3xs.text-muted-foreground.ml-auto {{ queuedFor(item.task) }}
        span.truncate.text-sm.font-medium {{ taskName(item.task) }}
        .flex.flex-wrap.items-center(class="gap-1.5")
          span.text-3xs.text-muted-foreground {{ projectName(item.task.projectId) }}
          span.text-3xs.rounded-full.border(
            v-for="tag in item.task.tags"
            :key="tag.id"
            class="px-1.5"
            :style="tagStyle(tag.color)"
          ) {{ tag.name }}
        .flex.flex-wrap.items-center.gap-1
          template(v-if="item.blockers.length")
            span.text-3xs.rounded-full.border(
              v-for="blocker in item.blockers"
              :key="blocker.kind + blocker.label"
              class="px-1.5 py-0.5"
              :class="blockerClass(blocker.kind)"
              :title="blocker.detail"
            ) {{ blocker.label }}
          span.text-3xs.text-muted-foreground(v-else)
            | {{ item.reason || taskWaitingReason(item.task) || "Waiting for free agent slot" }}
</template>

<script setup lang="ts">
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { ConcurrencyBlocker, WaitingTask } from "@/lib/concurrency";
import { formatRelativeTimestamp } from "@/lib/format";
import { tagStyle, taskName, taskWaitingReason } from "@/lib/task";
import type { Project } from "@cherrypi/sdk";
import { computed } from "vue";
import { RouterLink } from "vue-router";

const props = defineProps<{
  waiting: WaitingTask[];
  projects: Project[];
  title: string;
  description: string;
  empty: string;
  showAgent?: boolean;
}>();

const projectNames = computed(
  () => new Map(props.projects.map((project) => [project.id, project.name])),
);

function projectName(id: string): string {
  return projectNames.value.get(id) ?? id;
}

function queuedFor(task: WaitingTask["task"]): string {
  return formatRelativeTimestamp(task.queuedAt ?? task.createdAt);
}

function blockerClass(kind: ConcurrencyBlocker["kind"]): string {
  switch (kind) {
    case "offline":
      return "border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400";
    case "tag":
    case "project-tag":
      return "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400";
    case "reference":
      return "border-indigo-500/30 bg-indigo-500/10 text-indigo-600 dark:text-indigo-400";
    default:
      return "border-border bg-muted text-muted-foreground";
  }
}
</script>
