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
.mx-auto.flex.w-full.max-w-4xl.flex-col.gap-6.p-6
  BackButton.-ml-2.self-start(
    :fallback="{ to: { name: 'users' }, label: 'Users' }"
  )
  .flex.items-center.gap-4
    span.relative.size-16.shrink-0.overflow-hidden.rounded-full
      img.absolute.inset-0.size-full.object-cover(
        v-if="user?.picture"
        :src="user.picture.url"
        :alt="displayName"
      )
      span.absolute.inset-0(v-else :innerHTML="fallbackAvatar")
    .flex.min-w-0.flex-col(class="gap-0.5")
      .flex.items-center.gap-2
        h1.truncate.text-lg.font-semibold {{ displayName }}
        Badge(v-if="user?.role === 'admin'" variant="secondary") Administrator
        Badge(v-if="user?.status === 'invited'" variant="secondary") Invited
      p.text-muted-foreground.text-sm @{{ user?.username }}

  p.text-muted-foreground.text-sm(v-if="loading") Loading…
  template(v-else-if="user")
    Card(v-if="user.description")
      CardContent.pt-6
        p.text-sm(v-text="user.description")

    Card
      CardHeader
        CardTitle Profile
      CardContent
        dl.grid.grid-cols-2.gap-4.text-sm
          div
            dt.text-muted-foreground Role
            dd.font-medium {{ user.role === "admin" ? "Administrator" : "Member" }}
          div
            dt.text-muted-foreground Status
            dd.font-medium.capitalize {{ user.status }}
          div
            dt.text-muted-foreground Member since
            dd.font-medium {{ formatDate(user.created_at) }}
          div
            dt.text-muted-foreground Last sign-in
            dd.font-medium {{ user.last_login_at ? formatDate(user.last_login_at) : "Never" }}

    Card(v-if="canEdit")
      CardHeader
        CardTitle Picture
        CardDescription Upload a picture shown in the sidebar and on this page.
      CardContent
        PictureUpload(
          :picture="user.picture"
          :user-id="String(user.id)"
          :fallback="fallbackAvatar"
          @changed="onPictureChanged"
        )

    Card
      CardHeader
        CardTitle Teams
        CardDescription(v-if="teams.length") Teams this user belongs to.
        CardDescription(v-else) Not a member of any team yet.
      CardContent(v-if="teams.length")
        .flex.flex-col(class="gap-0.5")
          RouterLink.flex.items-center.justify-between.gap-2.rounded-md.px-2.text-sm(
            v-for="team in teams"
            :key="team.id"
            class="hover:bg-accent py-1.5"
            :to="{ name: 'team', params: { id: String(team.id) } }"
          )
            span.truncate {{ team.name }}
            Badge(variant="secondary") {{ team.role }}
      CardContent(v-else)
        p.text-muted-foreground.text-sm Not a member of any team yet.

    Card
      CardHeader
        CardTitle Projects
        CardDescription Projects this user can read.
      CardContent
        Empty(v-if="!projects.length")
          EmptyTitle No projects
        .flex.flex-col(v-else class="gap-0.5")
          RouterLink.flex.items-center.justify-between.gap-2.rounded-md.px-2.text-sm(
            v-for="project in projects"
            :key="project.id"
            class="hover:bg-accent py-1.5"
            :to="{ name: 'project', params: { slug: project.slug } }"
          )
            span.truncate {{ project.name }}
            Badge(v-if="project.archived_at" variant="secondary") Archived

    Card
      CardHeader
        CardTitle Agents
        CardDescription Agents this user owns.
      CardContent
        Empty(v-if="!ownedAgents.length")
          EmptyTitle No owned agents
        .flex.flex-wrap.gap-2(v-else)
          RouterLink.flex.items-center.gap-2.rounded-lg.border.px-3.py-2.text-sm.transition-colors(
            v-for="agent in ownedAgents"
            :key="agent.id"
            class="hover:bg-muted/50"
            :to="{ name: 'agent', params: { username: agent.username } }"
          )
            img.size-6.shrink-0.rounded-full.object-cover(
              v-if="agent.picture"
              :src="agent.picture.url"
              :alt="agent.displayName || agent.username"
            )
            span.size-6.shrink-0(v-else :innerHTML="agentAvatar(agent.id)")
            span {{ agent.displayName || agent.username }}

    Card
      CardHeader
        CardTitle Lead tasks
        CardDescription Tasks this user is accountable for.
      CardContent
        TasksList(
          empty-text="No tasks list this user as a lead."
          :lead-actor-id="String(user.id)"
        )
  p.text-muted-foreground.text-sm(v-else) User not found.
</template>

<script setup lang="ts">
import PictureUpload from "@/components/attachments/PictureUpload.vue";
import BackButton from "@/components/BackButton.vue";
import TasksList from "@/components/tasks/TasksList.vue";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Empty, EmptyTitle } from "@/components/ui/empty";
import { useAgents } from "@/composables/useAgents";
import { setPageTitle } from "@/composables/usePageTitle";
import {
  accountApi,
  currentAccount,
  type AccountActor,
  type DirectoryProject,
  type DirectoryTeam,
} from "@/lib/account";
import type { Agent, Attachment } from "@cherrypi/sdk";
import { toSvg } from "jdenticon";
import { computed, ref, watch, watchEffect } from "vue";
import { RouterLink, useRoute } from "vue-router";

const route = useRoute();
const { agents, loadAgents } = useAgents();
const user = ref<AccountActor | null>(null);
const teams = ref<DirectoryTeam[]>([]);
const projects = ref<DirectoryProject[]>([]);
const ownedAgentActorIds = ref<string[]>([]);
const loading = ref(true);

const displayName = computed(
  () => user.value?.name || user.value?.username || "",
);
const fallbackAvatar = computed(() =>
  toSvg(user.value?.username || "user", 64),
);

watchEffect(() => setPageTitle(displayName.value || "User"));

const canEdit = computed(
  () =>
    !!user.value &&
    (currentAccount.value?.is_admin === true ||
      currentAccount.value?.id === user.value.id),
);

const ownedAgents = computed<Agent[]>(() => {
  const owned = new Set(ownedAgentActorIds.value);
  return agents.value.filter((agent) => owned.has(agent.actorId));
});

watch(() => route.params.username, load, { immediate: true });

async function load(): Promise<void> {
  const username = String(route.params.username ?? "");
  loading.value = true;
  user.value = null;
  teams.value = [];
  projects.value = [];
  ownedAgentActorIds.value = [];
  try {
    const payload = await accountApi.getDirectoryUser(username);
    void loadAgents();
    user.value = payload.user;
    teams.value = payload.teams ?? [];
    projects.value = payload.projects ?? [];
    ownedAgentActorIds.value = payload.owned_agent_actor_ids ?? [];
  } catch {
  } finally {
    loading.value = false;
  }
}

function agentAvatar(id: string): string {
  return toSvg(id, 24);
}

function onPictureChanged(picture: Attachment | undefined): void {
  if (user.value) user.value.picture = picture;

  if (user.value && currentAccount.value?.id === user.value.id) {
    currentAccount.value.picture = picture;
  }
}

function formatDate(value?: string): string {
  if (!value) return "Never";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? "—" : parsed.toLocaleString();
}
</script>
