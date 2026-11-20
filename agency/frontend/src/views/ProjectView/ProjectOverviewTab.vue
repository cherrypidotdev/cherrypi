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
.flex.flex-col.gap-6
  KpiOverview(
    :page="page.project ? { kind: 'project', id: page.project.id } : undefined"
    :page-context="kpiPageContext"
  )

  ExpressTaskForm(
    v-if="page.project && page.canPrompt"
    :project-id="page.project.id"
  )
  p.text-muted-foreground.text-sm(v-else-if="page.project")
    | {{ $t("projectView.viewOnly") }}

  Card.overflow-visible
    CardHeader
      CardTitle {{ $t("nav.tasks") }}
      CardDescription {{ $t("projectView.tasksDescription") }}
    CardContent
      TasksList(
        v-if="page.project"
        empty-text="No tasks in this project."
        hide-project
        :project-id="page.project.id"
      )
</template>

<script setup lang="ts">
import KpiOverview from "@/components/kpi/KpiOverview.vue";
import ExpressTaskForm from "@/components/tasks/ExpressTaskForm.vue";
import TasksList from "@/components/tasks/TasksList.vue";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { KpiPageContext } from "@/lib/kpiWidgets";
import type { ProjectPageState } from "@/lib/projectPage";
import { computed } from "vue";

const props = defineProps<{ page: ProjectPageState }>();

const kpiPageContext = computed<KpiPageContext>(() => ({
  kind: "project",
  projectId: props.page.project?.id ?? "",
}));
</script>

<i18n lang="yaml">
en:
  projectView:
    viewOnly: "You have view-only access to this project."
    tasksDescription: "Tasks in this project, across agents."
ru:
  projectView:
    viewOnly: "У вас доступ к этому проекту только для чтения."
    tasksDescription: "Задачи этого проекта по всем агентам."
</i18n>
