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
.flex.h-full.flex-col.overflow-hidden
  header.border-b.px-6.py-3
    .mx-auto.flex.w-full.max-w-6xl.flex-wrap.items-center.justify-between.gap-3
      .flex.min-w-0.items-center.gap-3
        span.relative.shrink-0
          img.block.size-10.rounded-full.object-cover(
            v-if="item?.picture"
            :src="item.picture.url"
            :alt="name"
          )
          span.block.size-10(v-else :innerHTML="avatar")
          span.border-background.absolute.-right-0.-bottom-0.size-3.rounded-full.border(
            :class="online ? 'bg-emerald-500' : 'bg-muted-foreground/40'"
            :title="online ? 'Online' : 'Offline'"
          )
        .flex.min-w-0.flex-col
          h1.truncate.text-lg.font-semibold {{ name }}
          span.text-muted-foreground.truncate.font-mono.text-xs @{{ item?.username ?? currentAgentUsername }}
      Tabs(v-model="tab")
        TabsList
          TabsTrigger(value="overview") Overview
          TabsTrigger(value="usage") Usage
          TabsTrigger(value="terminals" :disabled="!item || !canUseTerminals") Terminals
          TabsTrigger(value="rdp") RDP
          TabsTrigger(value="settings") Settings
  Tabs.flex.min-h-0.flex-1.flex-col.overflow-hidden(v-model="tab")
    TabsContent.flex-1.overflow-auto(value="overview")
      .mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
        .flex.flex-col.gap-3
          p.text-muted-foreground.max-w-2xl.text-sm(v-if="item?.description")
            | {{ item.description }}
          .flex.flex-wrap.items-center.gap-2(v-if="item")
            span.text-muted-foreground.text-xs.font-medium Binary
            Badge(variant="outline" :class="agentStatusClass(item.status)") {{ agentStatusLabel(item.status) }}
            span.font-mono.text-xs(
              v-if="item.version"
              :title="agentVersionTitle(item)"
            ) {{ item.version }}
            Badge(
              v-if="agentVersionNeedsAttention(item.versionStatus)"
              variant="outline"
              :class="agentVersionStatusClass(item.versionStatus)"
              :title="agentVersionTitle(item)"
            ) {{ agentVersionStatusLabel(item.versionStatus) }}
            span.text-muted-foreground.text-xs(v-if="!item.version") No version reported

          p.max-w-2xl.text-xs.text-amber-600(
            v-if="item?.versionStatus === AgentVersionStatus.OUTDATED"
            class="dark:text-amber-400"
          )
            | This agent is running a build older than the agency's. Rebuild and redeploy the agent image to update it.
          p.max-w-2xl.text-xs.text-red-600(
            v-if="item?.versionStatus === AgentVersionStatus.INCOMPATIBLE"
            class="dark:text-red-400"
          )
            | This agent's protocol is incompatible with the agency, so its tunnel is refused. Rebuild and redeploy the agent image.

          .flex.flex-col.gap-2(class="sm:flex-row sm:items-start sm:gap-6")
            .flex.flex-wrap.items-center.gap-2
              span.text-muted-foreground.text-xs.font-medium Teams
              RouterLink.flex.items-center.rounded-full.border.px-2.py-1.text-xs.transition-colors(
                v-for="team in teams"
                :key="team.id"
                class="hover:bg-muted/50 gap-1.5"
                :to="{ name: 'team', params: { id: String(team.id) } }"
              )
                span.size-4.shrink-0(:innerHTML="teamLogo(team)")
                span {{ team.name }}
              span.text-muted-foreground.text-xs(v-if="!teams.length")
                | No teams
            .flex.flex-wrap.items-center.gap-2
              span.text-muted-foreground.text-xs.font-medium Projects
              RouterLink.flex.items-center.rounded-full.border.px-2.py-1.text-xs.transition-colors(
                v-for="project in projects"
                :key="project.id"
                class="hover:bg-muted/50 gap-1.5"
                :to="{ name: 'project', params: { slug: project.slug } }"
              )
                span.size-4.shrink-0(:innerHTML="logo(project)")
                span {{ project.name }}
              span.text-muted-foreground.text-xs(v-if="!projects.length")
                | No projects

        UsageGraph(:agent-id="currentAgentId")

        ExpressTaskForm(:agent-id="currentAgentId")

        TasksList(
          empty-text="No tasks for this agent yet."
          hide-assignee
          :agent-id="currentAgentId"
          :from-agent="currentAgentUsername"
        )

    TabsContent.flex-1.overflow-auto(value="usage")
      .mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
        UsageGraph(:agent-id="currentAgentId")
        Card
          CardHeader
            CardTitle Advanced usage
            CardDescription
              | Per-project and per-task cost, token and run breakdowns are being
              | built as a follow-up (CHPI-175), reusing the project page's charts.
              | The summary above stays live until then.
          CardContent
            p.text-muted-foreground.text-sm
              | This tab will host filterable charts for cost, tokens and active
              | time with per-task drill-down.

    TabsContent.flex-1.overflow-hidden(value="terminals")
      RootTerminalsView(embedded)

    TabsContent.flex-1.overflow-hidden(value="rdp")
      DesktopView(embedded)

    TabsContent.flex-1.overflow-auto(value="settings")
      .mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
        AgentSettingsView
</template>

<script setup lang="ts">
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
import UsageGraph from "@/components/usage/UsageGraph.vue";
import { useAgentAccess } from "@/composables/useAgentAccess";
import { useAgents } from "@/composables/useAgents";
import { setPageTitle } from "@/composables/usePageTitle";
import {
  agentStatusClass,
  agentStatusLabel,
  agentVersionNeedsAttention,
  agentVersionStatusClass,
  agentVersionStatusLabel,
  agentVersionTitle,
} from "@/lib/agent";
import { agency, currentAgentId, currentAgentUsername } from "@/lib/sdk";
import { agentTeamApi, type Team } from "@/lib/team";
import { AgentStatus, AgentVersionStatus, type Project } from "@cherrypi/sdk";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref, watchEffect } from "vue";
import { RouterLink, useRoute } from "vue-router";
import AgentSettingsView from "./AgentSettingsView.vue";
import DesktopView from "./DesktopView.vue";
import RootTerminalsView from "./RootTerminalsView.vue";

const route = useRoute();
const { agents, loadAgents } = useAgents();
const { canManageAgent } = useAgentAccess();

const TABS = ["overview", "usage", "terminals", "rdp", "settings"];
const tab = ref(
  typeof route.query.tab === "string" && TABS.includes(route.query.tab)
    ? route.query.tab
    : "overview",
);

const projects = ref<Project[]>([]);
const teams = ref<Team[]>([]);

const item = computed(() =>
  agents.value.find((a) => a.id === currentAgentId.value),
);
const name = computed(
  () => item.value?.displayName || item.value?.username || "Agent",
);
const online = computed(() => item.value?.status === AgentStatus.ONLINE);
const avatar = computed(() => toSvg(currentAgentId.value || "agent", 40));
// Root terminals are owner-only. The server enforces it too; this keeps a
// non-owner from landing on a tab whose calls would all fail.
const canUseTerminals = computed(
  () => !!item.value && canManageAgent(item.value.id),
);

watchEffect(() => setPageTitle(name.value));
// Wait for the agent list before deciding: `item` is undefined on first
// render, and query-driven deep links must not be reset prematurely.
watchEffect(() => {
  if (tab.value === "terminals" && item.value && !canUseTerminals.value) {
    tab.value = "overview";
  }
});

function logo(project: Project): string {
  return toSvg(project.id, 16);
}

function teamLogo(team: Team): string {
  return toSvg(String(team.id), 16);
}

onMounted(async () => {
  await loadAgents();
  try {
    const [projectList, teamList] = await Promise.all([
      agency.listAgentProjects(currentAgentId.value),
      agentTeamApi.list(currentAgentId.value),
    ]);
    projects.value = projectList;
    teams.value = teamList.teams;
  } catch {
    projects.value = [];
    teams.value = [];
  }
});
</script>
