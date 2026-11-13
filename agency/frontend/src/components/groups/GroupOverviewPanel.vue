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
  .grid.grid-cols-1.gap-6(class="lg:grid-cols-2")
    Card
      CardHeader
        CardTitle {{ t("groups.overview.machines") }}
        CardDescription {{ t("groups.overview.machinesHint") }}
      CardContent
        Empty(v-if="!members.length")
          EmptyTitle {{ t("groups.overview.noMachines") }}
          EmptyDescription {{ t("groups.overview.noMachinesHint") }}
        Table(v-else)
          TableHeader
            TableRow
              TableHead {{ t("groups.memberColumn") }}
              TableHead {{ t("common.status") }}
              TableHead {{ t("groups.overview.lastSeen") }}
          TableBody
            TableRow(v-for="member in members" :key="member.agentId")
              TableCell
                ActorBadge(
                  kind="agent"
                  :username="member.agent?.username"
                  :name="member.agent?.name"
                  :picture-url="member.agent?.picture?.url"
                )
              TableCell
                Badge(
                  :variant="isOnline(member.agentId) ? 'secondary' : 'outline'"
                )
                  | {{ isOnline(member.agentId) ? t("nav.online") : t("nav.offline") }}
              TableCell.text-muted-foreground
                | {{ formatTimestamp(member.lastSeenAt) }}

    Card
      CardHeader
        CardTitle {{ t("groups.overview.projects") }}
        CardDescription {{ t("groups.overview.projectsHint") }}
      CardContent
        Empty(v-if="!projects.length")
          EmptyTitle {{ t("groups.overview.noProjects") }}
          EmptyDescription {{ t("groups.overview.noProjectsHint") }}
        .flex.flex-col.divide-y(v-else)
          .flex.items-start.justify-between.gap-3.py-2(
            v-for="row in projects"
            :key="row.project?.id"
          )
            .flex.min-w-0.flex-col
              RouterLink.text-sm.font-medium(
                v-if="row.project"
                class="hover:text-primary"
                :to="{ name: 'project', params: { slug: row.project.slug } }"
              ) {{ row.project.name }}
              span.text-muted-foreground.text-xs
                | {{ t("groups.overview.memberAccess", { count: row.memberCount }) }}
              span.text-muted-foreground.text-xs(v-if="row.prompt")
                | {{ t("groups.overview.projectPromptVersion", { version: row.prompt.version }) }}
              span.text-muted-foreground.text-xs(v-else)
                | {{ t("groups.overview.noProjectPrompt") }}
            Badge(v-if="row.disabled" variant="outline")
              | {{ t("groups.disabled") }}
            Badge(v-else variant="secondary")
              | {{ t("groups.active") }}

  p.text-muted-foreground.text-sm {{ t("groups.overview.partialHint") }}
</template>

<script setup lang="ts">
import ActorBadge from "@/components/ActorBadge.vue";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAgentGroups } from "@/composables/useAgentGroups";
import { useAgents } from "@/composables/useAgents";
import { formatTimestamp } from "@/lib/format";
import type { AgentGroupMember, AgentGroupProject } from "@cherrypi/sdk";
import { AgentStatus } from "@cherrypi/sdk";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";

const { t } = useI18n();
const route = useRoute();
const { groupBySlug, loadMembers, loadProjects } = useAgentGroups();
const { agents, loadAgents } = useAgents();

const members = ref<AgentGroupMember[]>([]);
const projects = ref<AgentGroupProject[]>([]);

const slug = computed(() => String(route.params.slug ?? ""));
const group = computed(() => groupBySlug(slug.value));

function isOnline(agentId: string): boolean {
  return (
    agents.value.find((agent) => agent.id === agentId)?.status ===
    AgentStatus.ONLINE
  );
}

async function load() {
  const current = group.value;
  if (!current) return;
  void loadAgents();
  [members.value, projects.value] = await Promise.all([
    loadMembers(current.id),
    loadProjects(current.id),
  ]);
}

watch(slug, () => void load());
onMounted(() => void load());
</script>

<i18n lang="yaml">
en:
  groups:
    disabled: Disabled
    overview:
      machines: Machines
      machinesHint: Membership is the only thing the group owns about an agent.
      noMachines: No machines yet
      noMachinesHint: Add an agent on the Members tab.
      lastSeen: Last seen
      projects: Derived projects
      projectsHint: Computed from the members' own access. Never granted, never stored.
      noProjects: No projects yet
      noProjectsHint: A project appears once a member can reach it.
      memberAccess: "{count} member can reach it | {count} members can reach it"
      projectPromptVersion: "Group × project prompt v{version}"
      noProjectPrompt: No group × project prompt
      partialHint: Partial membership is normal. A project where only some members have access is still listed; its group × project prompt applies to the tasks those members run. When the count reaches zero the prompt is disabled, not deleted.
ru:
  groups:
    disabled: Отключён
    overview:
      machines: Машины
      machinesHint: Участие — единственное, чем группа владеет в отношении агента.
      noMachines: Машин пока нет
      noMachinesHint: Добавьте агента на вкладке «Участники».
      lastSeen: Последняя активность
      projects: Производные проекты
      projectsHint: Вычисляются по собственному доступу участников. Никогда не выдаются и не хранятся.
      noProjects: Проектов пока нет
      noProjectsHint: Проект появится, когда участник получит к нему доступ.
      memberAccess: "Доступно {count} участнику | Доступно {count} участникам | Доступно {count} участникам"
      projectPromptVersion: "Промпт группы × проекта v{version}"
      noProjectPrompt: Промпта группы × проекта нет
      partialHint: Частичное участие — это норма. Проект, к которому есть доступ только у части участников, всё равно в списке; промпт группы × проекта применяется к задачам, которые выполняют эти участники. Когда счётчик достигает нуля, промпт отключается, а не удаляется.
</i18n>
