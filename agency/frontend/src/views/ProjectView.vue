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
      ActorAvatar(
        :src="project?.picture?.url"
        :emoji="project?.emoji"
        :name="project?.name"
        :size="48"
        :fallback-html="logo"
      )
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
          Badge(v-if="project?.archivedAt" variant="secondary") {{ $t("domainLog.kind.archived") }}
        span.text-muted-foreground.truncate.text-sm(
          v-if="project?.description"
        )
          | {{ project.description }}
        span.text-muted-foreground.font-mono.text-xs {{ project?.slug }}
    Tabs(:model-value="activeTab" @update:model-value="onTabChange")
      TabsList
        TabsTrigger(
          v-for="tab in tabs"
          :key="tab.value"
          :value="tab.value"
          as-child
        )
          RouterLink(:to="tabTo(tab)") {{ $t(tab.labelKey) }}

  p.text-destructive.text-sm(v-if="page.error") {{ page.error }}

  ActorLinks.profile-links(:links="project?.links")

  RouterView(v-slot="{ Component }")
    component(:is="Component" :page="page")
</template>

<script setup lang="ts">
import ActorLinks from "@/components/actor/ActorLinks.vue";
import ActorAvatar from "@/components/ActorAvatar.vue";
import BackButton from "@/components/BackButton.vue";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAgents } from "@/composables/useAgents";
import type { BackFallback } from "@/composables/useBackNavigation";
import { setPageTitle } from "@/composables/usePageTitle";
import { useProjects } from "@/composables/useProjects";
import { projectAccessApi, type ProjectAccess } from "@/lib/projectAccess";
import { agency } from "@/lib/sdk";
import type { Project } from "@cherrypi/sdk";
import { ChevronRightIcon } from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router";

const { t } = useI18n();

const props = defineProps<{ slug: string }>();
const route = useRoute();
const router = useRouter();
const { loadAgents } = useAgents();
const { projects, loadProjects } = useProjects();

const tabs = [
  { value: "overview", name: "project", labelKey: "projectView.overview" },
  {
    value: "knowledge",
    name: "project-knowledge",
    labelKey: "projectView.knowledge",
  },
  { value: "usage", name: "project-usage", labelKey: "projectView.usage" },
  {
    value: "concurrency",
    name: "project-concurrency",
    labelKey: "agentConcurrency.title",
  },
  { value: "settings", name: "project-settings", labelKey: "common.settings" },
];

const project = ref<Project>();
const access = ref<ProjectAccess>();
const canPrompt = ref(false);
const canManageProject = ref(false);
const error = ref("");

const page = reactive({
  project,
  access,
  canPrompt,
  canManageProject,
  error,
  setProject: (updated: Project) => {
    project.value = updated;
  },
  reload: load,
  reloadAccess: loadAccess,
});

const activeTab = computed(() => {
  const name = String(route.name ?? "");
  if (name.startsWith("project-settings")) return "settings";
  if (name === "project-knowledge") return "knowledge";
  if (name === "project-usage") return "usage";
  if (name === "project-concurrency") return "concurrency";
  return "overview";
});

function tabTo(tab: (typeof tabs)[number]) {
  return { name: tab.name, params: { slug: props.slug } };
}

function onTabChange(value: unknown) {
  const tab = tabs.find((candidate) => candidate.value === value);
  if (!tab) return;
  void router.push(tabTo(tab));
}

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

watchEffect(() => {
  void route.name;
  setPageTitle(project.value?.name || t("admin.projects.columnProject"));
});

async function load() {
  error.value = "";
  try {
    project.value = await agency.getProject(props.slug);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
    return;
  }
  await loadAccess();
}

async function loadAccess() {
  if (!project.value) {
    access.value = undefined;
    canPrompt.value = false;
    canManageProject.value = false;
    return;
  }

  const projectID = project.value.id;
  try {
    const payload = await projectAccessApi.access(projectID);
    if (project.value?.id !== projectID) return;
    access.value = payload;
    canPrompt.value = payload.role !== "viewer";
    canManageProject.value = payload.can_manage_access;
  } catch {
    if (project.value?.id !== projectID) return;
    access.value = undefined;
    canPrompt.value = false;
    canManageProject.value = false;
  }
}

onMounted(async () => {
  await Promise.all([loadAgents(), loadProjects(), load()]);
});
</script>

<i18n lang="yaml">
en:
  projectView:
    overview: "Overview"
    knowledge: "Knowledge"
    usage: "Usage"
ru:
  projectView:
    overview: "Обзор"
    knowledge: "Знания"
    usage: "Использование"
</i18n>
