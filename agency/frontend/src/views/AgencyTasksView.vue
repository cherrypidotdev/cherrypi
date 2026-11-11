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
.mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
  header.flex.flex-wrap.items-start.justify-between.gap-4
    .flex.min-w-0.flex-col
      h1.truncate.text-lg.font-semibold {{ $t("nav.tasks") }}
      p.text-muted-foreground.text-sm
        | {{ $t("agencyTasks.description") }}
    AgencyViewTabs

  .flex.flex-wrap.items-center.gap-2
    Combobox.w-44(
      v-model="filterProject"
      :placeholder="$t('nav.allProjects')"
      empty-text="No matching project."
      :title="$t('usage.dashboard.projectTitle')"
      :aria-label="$t('usage.dashboard.projectTitle')"
      :options="projectFilterOptions"
    )
    Select(v-model="filterAgent")
      SelectTrigger.w-44(:title="$t('usage.dashboard.agentTitle')")
        SelectValue(:placeholder="$t('usage.dashboard.allAgents')")
      SelectContent
        SelectItem(:value="ALL") {{ $t("usage.dashboard.allAgents") }}
        SelectItem(v-for="item in agents" :key="item.id" :value="item.id") {{ item.displayName || item.username }}

  KpiOverview(:page="{ kind: 'agency' }" :page-context="kpiPageContext")

  ExpressTaskForm

  TasksList(
    :agent-id="filterAgent === ALL ? undefined : filterAgent"
    :project-id="filterProject === ALL ? undefined : filterProject"
  )
</template>

<script setup lang="ts">
import AgencyViewTabs from "@/components/AgencyViewTabs.vue";
import KpiOverview from "@/components/kpi/KpiOverview.vue";
import ExpressTaskForm from "@/components/tasks/ExpressTaskForm.vue";
import TasksList from "@/components/tasks/TasksList.vue";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import type { KpiPageContext } from "@/lib/kpiWidgets";
import { sortProjectsByName } from "@/lib/project";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

const { agents, loadAgents } = useAgents();
const { active: projects, loadProjects } = useProjects();
const projectOptions = computed(() => sortProjectsByName(projects.value));

const ALL = "__all__";
const projectFilterOptions = computed<ComboboxOption[]>(() => [
  { value: ALL, label: t("nav.allProjects") },
  ...projectOptions.value.map((project) => ({
    value: project.id,
    label: project.name,
    description: project.prefix ? `#${project.prefix}` : undefined,
  })),
]);
const filterAgent = ref(ALL);
const filterProject = ref(ALL);

const kpiPageContext = computed<KpiPageContext>(() => ({
  kind: "agency",
  filterAgentId: filterAgent.value === ALL ? "" : filterAgent.value,
  filterProjectId: filterProject.value === ALL ? "" : filterProject.value,
}));

onMounted(() => {
  void loadAgents();
  void loadProjects();
});

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  agencyTasks:
    description: "Every agency task, newest first. The scheduler assigns queued work subject to each agent's concurrency limits."
ru:
  agencyTasks:
    description: "Все задачи агентства, сначала новые. Планировщик распределяет задачи из очереди с учётом лимитов параллельности каждого агента."
</i18n>
