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
  .flex.min-w-0.flex-col
    h1.truncate.text-lg.font-semibold Tasks
    p.text-muted-foreground.text-sm
      | Every agency task, newest first. The scheduler assigns queued work
      | subject to each agent's concurrency limits.

  UsageGraph(
    :project-id="filterProject === ALL ? undefined : filterProject"
    :agent-id="filterAgent === ALL ? undefined : filterAgent"
  )

  ExpressTaskForm

  TasksList(
    :agent-id="filterAgent === ALL ? undefined : filterAgent"
    :project-id="filterProject === ALL ? undefined : filterProject"
  )
    template(#actions)
      .flex.items-center.gap-2
        Select(v-model="filterProject")
          SelectTrigger.w-44(title="Filter by project")
            SelectValue(placeholder="All projects")
          SelectContent
            SelectItem(:value="ALL") All projects
            SelectItem(
              v-for="project in projectOptions"
              :key="project.id"
              :value="project.id"
            ) {{ project.name }}
        Select(v-model="filterAgent")
          SelectTrigger.w-44(title="Filter by agent")
            SelectValue(placeholder="All agents")
          SelectContent
            SelectItem(:value="ALL") All agents
            SelectItem(v-for="item in agents" :key="item.id" :value="item.id") {{ item.displayName || item.username }}
</template>

<script setup lang="ts">
import ExpressTaskForm from "@/components/tasks/ExpressTaskForm.vue";
import TasksList from "@/components/tasks/TasksList.vue";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import UsageGraph from "@/components/usage/UsageGraph.vue";
import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { sortProjectsByName } from "@/lib/project";
import { computed, onMounted, ref } from "vue";

const { agents, loadAgents } = useAgents();
const { active: projects, loadProjects } = useProjects();
const projectOptions = computed(() => sortProjectsByName(projects.value));

const ALL = "__all__";
const filterAgent = ref(ALL);
const filterProject = ref(ALL);

onMounted(() => {
  void loadAgents();
  void loadProjects();
});
</script>
