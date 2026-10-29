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
    CardTitle {{ title }}
    CardDescription {{ description }}
  CardContent.p-2.pt-0
    p.text-muted-foreground.px-2.text-xs(v-if="!tasks.length") {{ empty }}
    .flex.flex-col(v-else class="gap-0.5")
      RouterLink.flex.items-start.gap-2.rounded-md.px-2.py-2.transition-colors(
        v-for="task in tasks"
        :key="task.id"
        class="hover:bg-muted/60"
        :to="{ name: 'task', params: { id: task.id } }"
      )
        component.size-4.shrink-0(
          :is="taskStateIcon(task.state)"
          :class="[taskStateTextClass(task.state), 'mt-0.5']"
        )
        .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
          .flex.flex-wrap.items-center(class="gap-1.5")
            span.text-3xs.text-muted-foreground.font-mono {{ task.id }}
            span.text-3xs.rounded-full.border(
              class="px-1.5"
              :class="taskStateClass(task.state)"
            ) {{ taskStateLabel(task.state) }}
          span.truncate.text-sm.font-medium {{ taskName(task) }}
          .flex.flex-wrap.items-center(class="gap-1.5")
            span.text-3xs.text-muted-foreground {{ projectName(task.projectId) }}
            span.text-3xs.rounded-full.border(
              v-for="tag in task.tags"
              :key="tag.id"
              class="px-1.5"
              :style="tagStyle(tag.color)"
            ) {{ tag.name }}
        span.text-3xs.text-muted-foreground.whitespace-nowrap {{ heldFor(task) }}
</template>

<script setup lang="ts">
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatRelativeTimestamp } from "@/lib/format";
import {
  tagStyle,
  taskName,
  taskStateClass,
  taskStateIcon,
  taskStateLabel,
  taskStateTextClass,
} from "@/lib/task";
import type { Project, Task } from "@cherrypi/sdk";
import { computed } from "vue";
import { RouterLink } from "vue-router";

const props = defineProps<{
  tasks: Task[];
  projects: Project[];
  title: string;
  description: string;
  empty: string;
}>();

const projectNames = computed(
  () => new Map(props.projects.map((project) => [project.id, project.name])),
);

function projectName(id: string): string {
  return projectNames.value.get(id) ?? id;
}

function heldFor(task: Task): string {
  return formatRelativeTimestamp(task.startedAt ?? task.settledAt);
}
</script>
