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
  Tabs(:model-value="activeTab" @update:model-value="onTabChange")
    TabsList
      TabsTrigger(
        v-for="tab in tabs"
        :key="tab.value"
        :value="tab.value"
        as-child
      )
        RouterLink(:to="tabTo(tab)") {{ $t(tab.labelKey) }}

  RouterView(v-slot="{ Component }")
    component(:is="Component" :page="page" :settings="settings")
</template>

<script setup lang="ts">
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { projectAccessApi, type ProjectHierarchy } from "@/lib/projectAccess";
import type { ProjectPageState } from "@/lib/projectPage";
import { computed, reactive, ref, watch } from "vue";
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router";

const props = defineProps<{ page: ProjectPageState }>();
const route = useRoute();
const router = useRouter();

const tabs = [
  {
    value: "general",
    name: "project-settings",
    labelKey: "settings.tabs.general",
  },
  {
    value: "tasks",
    name: "project-settings-tasks",
    labelKey: "nav.tasks",
  },
  {
    value: "models",
    name: "project-settings-models",
    labelKey: "projectSettings.tabs.models",
  },
  {
    value: "templates",
    name: "project-settings-templates",
    labelKey: "projectSettings.tabs.templates",
  },
  {
    value: "plugins",
    name: "project-settings-plugins",
    labelKey: "account.tabs.plugins",
  },
  {
    value: "variables",
    name: "project-settings-variables",
    labelKey: "variables.title",
  },
  {
    value: "access",
    name: "project-settings-access",
    labelKey: "domainLog.kind.access",
  },
  {
    value: "danger",
    name: "project-settings-danger",
    labelKey: "projectSettings.dangerZone",
  },
];

const hierarchy = ref<ProjectHierarchy>();
const settings = reactive({
  hierarchy,
  reloadHierarchy,
});

const activeTab = computed(() => {
  const name = String(route.name ?? "");
  switch (name) {
    case "project-settings-tasks":
      return "tasks";
    case "project-settings-models":
      return "models";
    case "project-settings-templates":
      return "templates";
    case "project-settings-plugins":
    case "project-settings-plugin":
      return "plugins";
    case "project-settings-variables":
      return "variables";
    case "project-settings-access":
      return "access";
    case "project-settings-danger":
      return "danger";
    default:
      return "general";
  }
});

const slug = computed(() => String(route.params.slug ?? ""));

function tabTo(tab: (typeof tabs)[number]) {
  return { name: tab.name, params: { slug: slug.value } };
}

function onTabChange(value: unknown) {
  const tab = tabs.find((candidate) => candidate.value === value);
  if (!tab) return;
  void router.push(tabTo(tab));
}

async function reloadHierarchy() {
  const project = props.page.project;
  if (!project) {
    hierarchy.value = undefined;
    return;
  }
  try {
    hierarchy.value = await projectAccessApi.hierarchy(project.id);
  } catch {
    hierarchy.value = undefined;
  }
}

watch(() => props.page.project?.id, reloadHierarchy, { immediate: true });
</script>
