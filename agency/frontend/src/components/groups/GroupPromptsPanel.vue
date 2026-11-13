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
.flex.flex-col.gap-6(v-if="group")
  Card
    CardHeader
      CardTitle {{ t("groups.prompts.groupTitle") }}
      CardDescription {{ t("groups.prompts.groupHint") }}
    CardContent
      PromptEditor(
        :placeholder="t('groups.prompts.groupPlaceholder')"
        :scope="PromptScope.GROUP"
        :group-id="String(group.id)"
      )

  Card
    CardHeader
      CardTitle {{ t("groups.prompts.projectTitle") }}
      CardDescription {{ t("groups.prompts.projectHint") }}
    CardContent
      Empty(v-if="!projects.length")
        EmptyTitle {{ t("groups.prompts.noProjects") }}
        EmptyDescription {{ t("groups.prompts.noProjectsHint") }}
      .flex.flex-col.gap-3(v-else)
        Collapsible.rounded-lg.border(
          v-for="row in projects"
          :key="row.project?.id"
          class="group/collapsible"
        )
          .flex.flex-wrap.items-center.gap-3.p-4
            CollapsibleTrigger.flex.flex-1.cursor-pointer.items-center.gap-2.text-left
              ChevronRightIcon.text-muted-foreground.size-4.shrink-0.transition-transform(
                class="group-data-[state=open]/collapsible:rotate-90"
              )
              span.text-sm.font-medium {{ row.project?.name }}
              Badge(v-if="row.prompt" variant="secondary")
                | {{ t("groups.prompts.version", { version: row.prompt.version }) }}
              Badge(v-else variant="outline") {{ t("groups.prompts.noPrompt") }}
            Badge(v-if="row.disabled" variant="outline")
              | {{ t("groups.disabled") }}
            span.text-muted-foreground.text-xs(v-if="row.disabled")
              | {{ row.disabledReason }}
          CollapsibleContent
            .px-4.pb-4
              PromptEditor(
                :placeholder="t('groups.prompts.projectPlaceholder', { project: row.project?.name ?? '' })"
                :scope="PromptScope.GROUP_PROJECT"
                :group-id="String(group.id)"
                :project-id="row.project?.id ?? ''"
              )

  Card
    CardHeader
      CardTitle {{ t("groups.prompts.teamTitle") }}
      CardDescription {{ t("groups.prompts.teamHint") }}
    CardContent
      Empty(v-if="!teams.length")
        EmptyTitle {{ t("groups.prompts.noTeams") }}
        EmptyDescription {{ t("groups.prompts.noTeamsHint") }}
      .flex.flex-col.gap-3(v-else)
        Collapsible.rounded-lg.border(
          v-for="row in teams"
          :key="String(row.teamId)"
          class="group/collapsible"
        )
          .flex.flex-wrap.items-center.gap-3.p-4
            CollapsibleTrigger.flex.flex-1.cursor-pointer.items-center.gap-2.text-left
              ChevronRightIcon.text-muted-foreground.size-4.shrink-0.transition-transform(
                class="group-data-[state=open]/collapsible:rotate-90"
              )
              span.text-sm.font-medium {{ row.name }}
              Badge(v-if="row.prompt" variant="secondary")
                | {{ t("groups.prompts.version", { version: row.prompt.version }) }}
              Badge(v-else variant="outline") {{ t("groups.prompts.noPrompt") }}
            span.text-muted-foreground.text-xs
              | {{ t("groups.prompts.attachedTo", { count: row.projectIds.length }) }}
          CollapsibleContent
            .px-4.pb-4
              PromptEditor(
                :placeholder="t('groups.prompts.teamPlaceholder', { team: row.name })"
                :scope="PromptScope.GROUP_TEAM"
                :group-id="String(group.id)"
                :team-id="String(row.teamId)"
              )

  p.text-muted-foreground.text-sm {{ t("groups.prompts.disabledHint") }}
</template>

<script setup lang="ts">
import PromptEditor from "@/components/PromptEditor.vue";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
import { useAgentGroups } from "@/composables/useAgentGroups";
import type { AgentGroupProject, AgentGroupTeam } from "@cherrypi/sdk";
import { PromptScope } from "@cherrypi/sdk";
import { ChevronRightIcon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";

const { t } = useI18n();
const route = useRoute();
const { groupBySlug, loadProjects, loadTeams } = useAgentGroups();

const projects = ref<AgentGroupProject[]>([]);
const teams = ref<AgentGroupTeam[]>([]);

const slug = computed(() => String(route.params.slug ?? ""));
const group = computed(() => groupBySlug(slug.value));

async function load() {
  const current = group.value;
  if (!current) return;
  [projects.value, teams.value] = await Promise.all([
    loadProjects(current.id),
    loadTeams(current.id),
  ]);
}

watch(slug, () => void load());
onMounted(() => void load());
</script>

<i18n lang="yaml">
en:
  groups:
    prompts:
      groupTitle: Group prompt
      groupHint: Injected into every task a member agent runs, before the agent's own prompt.
      groupPlaceholder: What every machine in this group should know.
      projectTitle: Per-project prompts
      projectHint: A group × project prompt applies to tasks in that project. It is disabled while no member can reach the project.
      projectPlaceholder: "Extra instructions for tasks in {project}."
      noProjects: No project prompts
      noProjectsHint: A project appears here once a member can reach it, or when a prompt was written for it earlier.
      teamTitle: Per-team prompts
      teamHint: A group × team prompt applies when a team that OWNS the task matches, even if the running machine is not in that team.
      teamPlaceholder: "Extra instructions for tasks owned by {team}."
      noTeams: No team prompts
      noTeamsHint: A team appears here when it is attached to a project a member can reach, or when a prompt was written for it earlier.
      version: "v{version}"
      noPrompt: No prompt
      attachedTo: "Attached to {count} project | Attached to {count} projects"
      disabledHint: A disabled prompt is never deleted. It keeps its content, versions and proposals and simply does not apply to any task right now.
ru:
  groups:
    prompts:
      groupTitle: Промпт группы
      groupHint: Добавляется в каждую задачу, которую выполняет агент-участник, перед собственным промптом агента.
      groupPlaceholder: Что должна знать каждая машина этой группы.
      projectTitle: Промпты по проектам
      projectHint: Промпт группы × проекта применяется к задачам этого проекта. Он отключён, пока ни один участник не имеет доступа к проекту.
      projectPlaceholder: "Дополнительные инструкции для задач в проекте {project}."
      noProjects: Промптов по проектам нет
      noProjectsHint: Проект появится здесь, когда участник получит к нему доступ, или если для него уже был написан промпт.
      teamTitle: Промпты по командам
      teamHint: Промпт группы × команды применяется, когда совпадает команда, ВЛАДЕЮЩАЯ задачей, даже если выполняющая машина не входит в эту команду.
      teamPlaceholder: "Дополнительные инструкции для задач команды {team}."
      noTeams: Промптов по командам нет
      noTeamsHint: Команда появится здесь, когда она привязана к проекту, доступному участнику, или если для неё уже был написан промпт.
      version: "v{version}"
      noPrompt: Промпта нет
      attachedTo: "Привязана к {count} проекту | Привязана к {count} проектам | Привязана к {count} проектам"
      disabledHint: Отключённый промпт никогда не удаляется. Он сохраняет содержимое, версии и предложения и просто не применяется ни к одной задаче сейчас.
</i18n>
