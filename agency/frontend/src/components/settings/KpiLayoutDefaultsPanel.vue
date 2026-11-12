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
.flex.flex-col.gap-4
  p.text-destructive.text-xs(v-if="error") {{ error }}

  .flex.flex-col.gap-2
    .flex.flex-col
      span.text-sm.font-medium {{ $t("kpiDefaults.projectLayout") }}
      span.text-muted-foreground.text-xs
        | {{ $t("kpiDefaults.projectHint") }}
    KpiGrid(
      editing
      hide-actions
      :widgets="projectDraft"
      :page="{ kind: 'project', projectId: '' }"
      :range="RANGE"
      :agent-options="agentOptions"
      :project-options="projectOptions"
      @update:widgets="projectDraft = $event"
    )

  .flex.flex-col.gap-2
    .flex.flex-col
      span.text-sm.font-medium {{ $t("kpiDefaults.agentLayout") }}
      span.text-muted-foreground.text-xs
        | {{ $t("kpiDefaults.agentHint") }}
    KpiGrid(
      editing
      hide-actions
      :widgets="agentDraft"
      :page="{ kind: 'agent', agentId: '' }"
      :range="RANGE"
      :agent-options="agentOptions"
      :project-options="projectOptions"
      @update:widgets="agentDraft = $event"
    )

  .flex.flex-wrap.items-center.gap-2
    Button(type="button" size="sm" @click="save" :disabled="saving")
      | {{ $t("kpiDefaults.save") }}
    span.text-muted-foreground.text-3xs
      | {{ $t("kpiDefaults.emptyHint") }}
</template>

<script setup lang="ts">
import KpiGrid from "@/components/kpi/KpiGrid.vue";
import { Button } from "@/components/ui/button";
import { useAgents } from "@/composables/useAgents";
import { useKpiLayoutDefaults } from "@/composables/useKpiLayout";
import { useProjects } from "@/composables/useProjects";
import type { KpiWidgetState } from "@/lib/kpiWidgets";
import { sortProjectsByName } from "@/lib/project";
import type { UsageRange } from "@/lib/usage";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const RANGE: UsageRange = "30";

const { agents, loadAgents } = useAgents();
const { active, loadProjects } = useProjects();
const {
  project,
  agent,
  saving,
  error,
  save: saveDefaults,
} = useKpiLayoutDefaults();

const projectDraft = ref<KpiWidgetState[]>([]);
const agentDraft = ref<KpiWidgetState[]>([]);

const agentOptions = computed(() =>
  agents.value.map((item) => ({
    id: item.id,
    name: item.displayName || item.username,
  })),
);
const projectOptions = computed(() =>
  sortProjectsByName(active.value).map((item) => ({
    id: item.id,
    name: item.name,
  })),
);

watch(project, (value) => {
  projectDraft.value = value.map((widget) => ({ ...widget }));
});
watch(agent, (value) => {
  agentDraft.value = value.map((widget) => ({ ...widget }));
});

async function save() {
  try {
    await saveDefaults({
      project: projectDraft.value,
      agent: agentDraft.value,
    });
    toast.success(t("kpiDefaults.saved"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

onMounted(() => {
  void loadAgents();
  void loadProjects();
});
</script>

<i18n lang="yaml">
en:
  kpiDefaults:
    projectHint: "Every project page that has no layout of its own renders this. Saving a project's own layout detaches it, and the default never changes it."
    agentHint: "Every agent page that has no layout of its own renders this. An agent's owner may replace it on the agent's own Overview."
    save: "Save default layouts"
    emptyHint: "An empty saved layout means new project and agent pages start with no widgets. Until you save, they use the four starter widgets."
    saved: "Default widget layouts saved"
    projectLayout: "Default Project layout"
    agentLayout: "Default Agent layout"
ru:
  kpiDefaults:
    projectHint: "Этот макет показывается на каждой странице проекта без собственного макета. Сохранение макета проекта отвязывает его, и значение по умолчанию его больше не меняет."
    agentHint: "Этот макет показывается на каждой странице агента без собственного макета. Владелец агента может заменить его в обзоре агента."
    save: "Сохранить макеты по умолчанию"
    emptyHint: "Пустой сохранённый макет означает, что новые страницы проектов и агентов начинаются без виджетов. До сохранения они используют четыре стартовых виджета."
    saved: "Макеты виджетов по умолчанию сохранены"
    projectLayout: "Макет проекта по умолчанию"
    agentLayout: "Макет агента по умолчанию"
</i18n>
