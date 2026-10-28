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
    .flex.min-w-0.items-center.gap-4
      img.size-12.shrink-0.rounded-full.object-cover(
        v-if="project?.picture"
        :src="project.picture.url"
        :alt="project?.name"
      )
      span.size-12.shrink-0(v-else :innerHTML="logo")
      .flex.min-w-0.flex-col
        BackButton.-ml-2.self-start(:fallback="backFallback")
        nav.text-muted-foreground.flex.flex-wrap.items-center.gap-1.text-xs(
          v-if="ancestors.length"
        )
          template(v-for="ancestor in ancestors" :key="ancestor.id")
            RouterLink(
              class="hover:text-foreground"
              :to="{ name: 'project', params: { slug: ancestor.slug } }"
            ) {{ ancestor.name }}
            ChevronRightIcon.size-3.shrink-0
        .flex.items-center.gap-2
          h1.truncate.text-xl.font-semibold {{ project?.name ?? "Project" }}
          Badge(v-if="project?.archivedAt" variant="secondary") Archived
        span.text-muted-foreground.truncate.text-sm(
          v-if="project?.description"
        )
          | {{ project.description }}
        span.text-muted-foreground.font-mono.text-xs {{ project?.slug }}
    Tabs(v-model="tab")
      TabsList
        TabsTrigger(value="overview") Overview
        TabsTrigger(value="usage") Usage
        TabsTrigger(value="settings") Settings

  p.text-destructive.text-sm(v-if="error") {{ error }}

  Tabs(v-model="tab")
    TabsContent(value="overview")
      .flex.flex-col.gap-6
        UsageOverview(
          v-if="project"
          context="project"
          :project-id="project.id"
        )

        ExpressTaskForm(v-if="project && canPrompt" :project-id="project.id")
        p.text-muted-foreground.text-sm(v-else-if="project")
          | You have view-only access to this project.

        Card.overflow-visible
          CardHeader
            CardTitle Tasks
            CardDescription Tasks in this project, across agents.
          CardContent
            TasksList(
              v-if="project"
              empty-text="No tasks in this project."
              hide-project
              :project-id="project.id"
            )

    TabsContent(value="usage")
      UsageDashboard(v-if="project" context="project" :project-id="project.id")

    TabsContent(value="settings")
      ProjectSettingsView(
        v-if="project"
        :slug="slug"
        @changed="onProjectChanged"
      )
</template>

<script setup lang="ts">
import BackButton from "@/components/BackButton.vue";
import ExpressTaskForm from "@/components/tasks/ExpressTaskForm.vue";
import TasksList from "@/components/tasks/TasksList.vue";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import UsageDashboard from "@/components/usage/UsageDashboard.vue";
import UsageOverview from "@/components/usage/UsageOverview.vue";
import { useAgents } from "@/composables/useAgents";
import type { BackFallback } from "@/composables/useBackNavigation";
import { setPageTitle } from "@/composables/usePageTitle";
import { useProjects } from "@/composables/useProjects";
import { projectAccessApi } from "@/lib/projectAccess";
import { agency } from "@/lib/sdk";
import type { Project } from "@cherrypi/sdk";
import { ChevronRightIcon } from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref, watch, watchEffect } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import ProjectSettingsView from "./ProjectSettingsView.vue";

const props = defineProps<{ slug: string }>();
const route = useRoute();
const router = useRouter();
const { loadAgents } = useAgents();
const { projects, loadProjects } = useProjects();

const TABS = ["overview", "usage", "settings"];
const tab = ref(
  typeof route.query.tab === "string" && TABS.includes(route.query.tab)
    ? route.query.tab
    : "overview",
);
const project = ref<Project>();
const canPrompt = ref(false);
const error = ref("");

const ancestors = computed(() => {
  const byID = new Map(projects.value.map((item) => [item.id, item]));
  const chain: Project[] = [];
  let parentID = project.value?.parentProjectId;
  const seen = new Set<string>();
  while (parentID && !seen.has(parentID)) {
    seen.add(parentID);
    const parent = byID.get(parentID);
    if (!parent) break;
    chain.unshift(parent);
    parentID = parent.parentProjectId;
  }
  return chain;
});

const backFallback = computed<BackFallback>(() => {
  const parent = ancestors.value.at(-1);
  if (parent) {
    return {
      to: { name: "project", params: { slug: parent.slug } },
      label: parent.name,
    };
  }
  return { to: { name: "projects" }, label: "Projects" };
});

const logo = computed(() => toSvg(project.value?.id ?? "project", 48));

watchEffect(() => setPageTitle(project.value?.name || "Project"));

function onProjectChanged(updated: Project) {
  project.value = updated;
}

async function load() {
  try {
    project.value = await agency.getProject(props.slug);
    try {
      const access = await projectAccessApi.access(project.value.id);
      canPrompt.value = access.role !== "viewer";
    } catch {
      canPrompt.value = false;
    }
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

watch(tab, (value) => {
  const next = { ...route.query };
  if (value === "overview") delete next.tab;
  else next.tab = value;
  void router.replace({ query: next });
});

onMounted(async () => {
  await Promise.all([loadAgents(), loadProjects(), load()]);
});
</script>
