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
.bg-background.text-foreground.flex.h-screen
  aside#agency-sidebar.bg-sidebar.text-sidebar-foreground.flex.shrink-0.flex-col.border-r.duration-200.ease-out(
    class="transition-[width] motion-reduce:transition-none"
    :class="collapsed ? 'w-14' : 'w-56'"
  )
    header.flex.shrink-0.items-center.gap-1.border-b(
      :class="collapsed ? 'flex-col gap-1 px-1.5 py-2' : 'h-14 px-2'"
    )
      Button(
        variant="ghost"
        size="icon-sm"
        aria-controls="agency-sidebar"
        :title="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
        :aria-label="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
        :aria-expanded="!collapsed"
        @click="toggle()"
      )
        PanelLeftOpenIcon(v-if="collapsed")
        PanelLeftCloseIcon(v-else)
      RouterLink.text-primary.flex.min-w-0.flex-1.items-center.gap-2.rounded-md.font-semibold(
        title="Tasks"
        :to="{ name: 'agency-tasks' }"
        :class="collapsed ? 'justify-center py-1' : 'px-1'"
      )
        img.block.size-5.shrink-0.rounded-sm.object-contain(
          :src="logoUrl"
          :alt="agencyName"
        )
        span.text-base(:class="collapsed ? 'sr-only' : 'truncate'") {{ agencyName }}
      Button(
        size="icon-sm"
        variant="default"
        title="New task"
        aria-label="New task"
        @click="newTaskOpen = true"
      )
        PlusIcon

    .flex.flex-1.flex-col.overflow-auto(
      :class="collapsed ? 'px-1.5 py-2' : 'p-2'"
      data-testid="sidebar-directory"
    )
      SidebarSection(
        :icon="FolderIcon"
        title="Projects"
        :to="{ name: 'projects' }"
        :active="isProjectsSectionActive()"
        :open="isSectionOpen('projects')"
        :collapsed="collapsed"
        @toggle="toggleSection('projects')"
      )
        ProjectsNav(:collapsed="collapsed")
        RouterLink.text-muted-foreground.flex.items-center.gap-2.rounded-md.px-3.py-2.text-xs.font-medium(
          v-if="!collapsed"
          :to="{ name: 'projects' }"
        )
          span All projects
      .mt-2.border-t.pt-2
        SidebarSection(
          :icon="BotIcon"
          title="Agents"
          :to="{ name: 'agents' }"
          :active="isAgentsSectionActive()"
          :open="isSectionOpen('agents')"
          :collapsed="collapsed"
          :count="sidebarAgents.length"
          @toggle="toggleSection('agents')"
        )
          p.text-muted-foreground.px-3.py-1.text-xs(
            v-if="!sidebarAgents.length && !collapsed"
          )
            | No agents yet
          RouterLink.flex.items-center.gap-2.rounded-md.px-3.py-2.text-sm.font-medium.transition-colors(
            v-for="item in sidebarAgents"
            :key="item.id"
            :to="{ name: 'agent', params: { username: item.username } }"
            :title="collapsed ? item.displayName || item.username : undefined"
            :class="[isAgentActive(item.username) ? 'text-primary' : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground', collapsed ? 'justify-center px-0' : '']"
          )
            span.relative.shrink-0
              img.block.size-5.rounded-full.object-cover(
                v-if="item.picture"
                :src="item.picture.url"
                :alt="item.displayName || item.username"
              )
              span.block.size-5(v-else :innerHTML="avatar(item)")
              span.border-sidebar.absolute.-right-0.-bottom-0.size-2.rounded-full.border(
                :class="item.status === AgentStatus.ONLINE ? 'bg-emerald-500' : 'bg-muted-foreground/40'"
                :title="item.status === AgentStatus.ONLINE ? 'Online' : 'Offline'"
              )
            span(:class="collapsed ? 'sr-only' : 'min-w-0 truncate'") {{ item.displayName || item.username }}
            span.ml-auto.inline-flex.shrink-0.items-center(
              v-if="!collapsed && fleetStat(item.id)"
              class="gap-1.5"
              :title="fleetTitle(item.id)"
            )
              span.text-3xs.min-w-3.text-right.font-mono.text-amber-600(
                v-if="fleetStat(item.id)?.waiting"
                class="dark:text-amber-400"
              ) {{ fleetStat(item.id)?.waiting }}
              ConcurrencyTrack.w-10(
                :label="item.displayName || item.username"
                size="sm"
                :used="fleetStat(item.id)?.used ?? 0"
                :cap="fleetStat(item.id)?.cap ?? 1"
                :waiting="0"
                :show-count="false"
                :show-waiting="false"
              )
          RouterLink.text-muted-foreground.flex.items-center.gap-2.rounded-md.px-3.py-2.text-xs.font-medium(
            v-if="!collapsed"
            :to="{ name: 'agents' }"
          )
            span Manage agents
      .mt-2.border-t.pt-2
        SidebarSection(
          :icon="UsersIcon"
          title="Teams"
          :to="{ name: 'teams' }"
          :active="isTeamsSectionActive()"
          :open="isSectionOpen('teams')"
          :collapsed="collapsed"
          :count="sidebarTeams.length"
          @toggle="toggleSection('teams')"
        )
          p.text-muted-foreground.px-3.py-1.text-xs(
            v-if="!sidebarTeams.length && !collapsed"
          )
            | Not in a team yet
          RouterLink.flex.items-center.gap-2.rounded-md.px-3.py-2.text-sm.font-medium.transition-colors(
            v-for="team in sidebarTeams"
            :key="team.id"
            :to="{ name: 'team', params: { id: String(team.id) } }"
            :title="collapsed ? team.name : undefined"
            :class="[isTeamActive(team.id) ? 'text-primary' : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground', collapsed ? 'justify-center px-0' : '']"
          )
            span.size-5.shrink-0(:innerHTML="teamLogo(team)")
            span(:class="collapsed ? 'sr-only' : 'min-w-0 truncate'") {{ team.name }}
          RouterLink.text-muted-foreground.flex.items-center.gap-2.rounded-md.px-3.py-2.text-xs.font-medium(
            v-if="!collapsed"
            :to="{ name: 'teams' }"
          )
            span All teams
      .mt-2.border-t.pt-2
        SidebarSection(
          :icon="UserIcon"
          title="Users"
          :to="{ name: 'users' }"
          :active="isUsersSectionActive()"
          :open="isSectionOpen('users')"
          :collapsed="collapsed"
          :count="sidebarUsers.length"
          @toggle="toggleSection('users')"
        )
          p.text-muted-foreground.px-3.py-1.text-xs(
            v-if="!sidebarUsers.length && !collapsed"
          )
            | No teammates yet
          RouterLink.flex.items-center.gap-2.rounded-md.px-3.py-2.text-sm.font-medium.transition-colors(
            v-for="person in sidebarUsers"
            :key="person.key"
            :to="{ name: 'user', params: { username: person.username } }"
            :title="collapsed ? person.name || person.username : undefined"
            :class="[isUserActive(person.username) ? 'text-primary' : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground', collapsed ? 'justify-center px-0' : '']"
          )
            span.relative.shrink-0
              img.block.size-5.rounded-full.object-cover(
                v-if="person.pictureUrl"
                :src="person.pictureUrl"
                :alt="person.name || person.username"
              )
              span.block.size-5(v-else :innerHTML="person.avatar")
            span(:class="collapsed ? 'sr-only' : 'min-w-0 truncate'") {{ person.name || person.username }}

    .flex.shrink-0.flex-col.border-t(
      :class="collapsed ? 'items-center gap-1 px-1.5 py-2' : 'gap-2 p-2'"
    )
      .flex.w-full.items-center.gap-2(:class="collapsed ? 'flex-col' : ''")
        img.block.size-8.shrink-0.rounded-full.object-cover(
          v-if="currentAccount?.picture"
          :src="currentAccount.picture.url"
          :alt="displayName"
          :title="collapsed ? displayName : undefined"
        )
        span.block.size-8.shrink-0(
          v-else
          :innerHTML="userAvatar"
          :title="collapsed ? displayName : undefined"
        )
        .flex.min-w-0.flex-1.flex-col(v-if="!collapsed")
          span.truncate.text-sm.font-medium {{ displayName }}
          span.text-muted-foreground.truncate.text-xs @{{ username || "…" }}
        Button(
          as-child
          variant="ghost"
          size="icon-sm"
          title="Inbox"
          aria-label="Inbox"
        )
          RouterLink.relative.flex.items-center.justify-center(
            :to="{ name: 'inbox' }"
          )
            InboxIcon
            span.bg-primary.text-primary-foreground.absolute.flex.size-4.items-center.justify-center.rounded-full.font-semibold(
              v-if="unread"
              class="-top-1.5 -right-1.5 text-[9px]"
            ) {{ unread > 9 ? "9+" : unread }}
        Button(
          as-child
          variant="ghost"
          size="icon-sm"
          title="User settings"
          aria-label="User settings"
        )
          RouterLink.flex.items-center.justify-center(
            :to="{ name: 'account' }"
          )
            SettingsIcon
    footer.flex.shrink-0.border-t(
      :class="collapsed ? 'flex-col items-center gap-1 px-1.5 py-2' : 'items-center gap-1 px-2 py-2'"
    )
      span.text-muted-foreground.truncate.font-mono(
        title="App version"
        :class="collapsed ? 'sr-only' : 'flex-1 text-[10px]'"
      ) {{ appVersion }}
      Button(
        v-if="isAdmin"
        as-child
        variant="ghost"
        size="icon-sm"
        title="Organization settings"
        aria-label="Organization settings"
      )
        RouterLink.flex.items-center.justify-center(:to="{ name: 'settings' }")
          SlidersHorizontalIcon
      Button(
        as-child
        variant="ghost"
        size="icon-sm"
        title="Documentation"
        aria-label="Documentation"
      )
        a.flex.items-center.justify-center(
          href="/docs/"
          rel="noopener"
          target="_blank"
        )
          BookOpenIcon
      Button(
        variant="ghost"
        size="icon-sm"
        :title="isDark ? 'Switch to light theme' : 'Switch to dark theme'"
        :aria-label="isDark ? 'Switch to light theme' : 'Switch to dark theme'"
        @click="toggleDark()"
      )
        MoonIcon(v-if="isDark")
        SunIcon(v-else)
  .flex.flex-1.flex-col.overflow-hidden
    main.flex-1.overflow-auto
      RouterView(v-slot="{ Component }")
        component(:key="route.fullPath" :is="Component")
  TaskComposerDialog(v-model:open="newTaskOpen")
</template>

<script setup lang="ts">
import ConcurrencyTrack from "@/components/concurrency/ConcurrencyTrack.vue";
import ProjectsNav from "@/components/ProjectsNav.vue";
import SidebarSection from "@/components/SidebarSection.vue";
import TaskComposerDialog from "@/components/tasks/TaskComposerDialog.vue";
import { Button } from "@/components/ui/button";
import { useAgents } from "@/composables/useAgents";
import { attentionPreference } from "@/composables/useAttentionPreferences";
import { useDirectory } from "@/composables/useDirectory";
import { useFleetConcurrency } from "@/composables/useFleetConcurrency";
import { useNotifications } from "@/composables/useNotifications";
import { usePageTitle } from "@/composables/usePageTitle";
import {
  startRealtime,
  stopRealtime,
  useRealtime,
} from "@/composables/useRealtime";
import { useSettings } from "@/composables/useSettings";
import { useSidebar } from "@/composables/useSidebar";
import { useUsers } from "@/composables/useUsers";
import { currentAccount, type AccountActor } from "@/lib/account";
import { applyFavicon, armAttentionPermission } from "@/lib/attention";
import { ATTENTION_EVENTS } from "@/lib/attentionEvents";
import type { Team } from "@/lib/team";
import { currentUser } from "@/lib/user";
import { AgentStatus, type Agent } from "@cherrypi/sdk";
import {
  BookOpenIcon,
  BotIcon,
  FolderIcon,
  InboxIcon,
  MoonIcon,
  PanelLeftCloseIcon,
  PanelLeftOpenIcon,
  PlusIcon,
  SettingsIcon,
  SlidersHorizontalIcon,
  SunIcon,
  UserIcon,
  UsersIcon,
} from "@lucide/vue";
import { useDark, useToggle } from "@vueuse/core";
import { toSvg } from "jdenticon";
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  watchEffect,
} from "vue";
import { RouterLink, RouterView, useRoute } from "vue-router";

// vite.config.ts.
const appVersion = __APP_VERSION__;

const isDark = useDark();
const toggleDark = useToggle(isDark);

const { settings, loadSettings } = useSettings();
const { collapsed, toggle, isSectionOpen, toggleSection } = useSidebar();

const newTaskOpen = ref(false);
const { agents, loadAgents } = useAgents();
const { users, loadUsers } = useUsers();
const {
  teams,
  ownedAgentActorIds,
  teammateUsers,
  teammateAgentActorIds,
  loadDirectory,
} = useDirectory();

const isAdmin = computed(() => currentAccount.value?.is_admin === true);
const {
  notifications,
  unread,
  start: startNotifications,
  stop: stopNotifications,
  markTaskRead,
} = useNotifications();
const agencyName = computed(
  () => settings.value?.agencyName?.trim() || "CherryPi",
);

const DEFAULT_LOGO = "/logo.png";
const logoUrl = computed(() => settings.value?.logo?.url || DEFAULT_LOGO);

const configuredIcon = computed(
  () => settings.value?.favicon?.url || settings.value?.logo?.url || "",
);

interface SidebarUser {
  key: string;
  username: string;
  name: string;
  pictureUrl?: string;
  avatar: string;
}

const username = ref("");
const displayName = computed(
  () => currentAccount.value?.name || username.value || "User",
);
const userAvatar = computed(() =>
  toSvg(currentAccount.value?.username || username.value || "user", 32),
);

const sidebarUsers = computed(() => {
  if (isAdmin.value) {
    // Invited accounts are not people yet; the directory endpoint already

    return users.value
      .filter((person) => person.status !== "invited")
      .map((person) => sidebarUser(person));
  }
  const byUsername = new Map<string, SidebarUser>(
    teammateUsers.value.map((person) => [
      person.username,
      {
        key: person.actor_id,
        username: person.username,
        name: person.name,
        pictureUrl: undefined,
        avatar: toSvg(person.username || person.actor_id, 20),
      },
    ]),
  );
  if (currentAccount.value) {
    byUsername.set(
      currentAccount.value.username,
      sidebarUser(currentAccount.value),
    );
  }
  return [...byUsername.values()].sort((a, b) =>
    (a.name || a.username).localeCompare(b.name || b.username),
  );
});

function sidebarUser(person: AccountActor): SidebarUser {
  return {
    key: String(person.id),
    username: person.username,
    name: person.name,
    pictureUrl: person.picture?.url,
    avatar: toSvg(person.username || String(person.id), 20),
  };
}

const sidebarAgents = computed(() => {
  const owned = new Set(ownedAgentActorIds.value);
  const shared = teammateAgentActorIds.value;
  return agents.value.filter(
    (agent) => owned.has(agent.actorId) || shared.has(agent.actorId),
  );
});

const sidebarTeams = computed(() => teams.value);

const { stats: fleetStats } = useFleetConcurrency(() => sidebarAgents.value);

function fleetStat(agentId: string) {
  return fleetStats.value[agentId];
}

function fleetTitle(agentId: string): string {
  const stat = fleetStat(agentId);
  if (!stat) return "";
  return `${stat.used}/${stat.cap} slots held${stat.waiting ? ` · ${stat.waiting} waiting` : ""}`;
}

const { section } = usePageTitle();
watchEffect(() => {
  const prefix = unread.value > 0 ? `(${unread.value}) ` : "";
  document.title = section.value
    ? `${prefix}${agencyName.value} - ${section.value}`
    : `${prefix}${agencyName.value}`;
  applyFavicon(unread.value, configuredIcon.value);
});

onMounted(async () => {
  void loadSettings();
  void loadAgents();
  void loadDirectory();
  startNotifications();
  startRealtime();

  if (
    ATTENTION_EVENTS.some((event) => attentionPreference(event.kind).browser)
  ) {
    armAttentionPermission();
  }
  username.value = currentAccount.value?.username ?? (await currentUser());
});

watch(currentAccount, (actor) => {
  if (actor) username.value = actor.username;
});

watch(
  isAdmin,
  (admin) => {
    if (admin) void loadUsers();
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  stopNotifications();
  stopRealtime();
});

useRealtime(["agents"], () => {
  void loadAgents(true);
  void loadDirectory(true);
});

function avatar(item: Agent): string {
  return toSvg(item.id, 20);
}

function teamLogo(team: Team): string {
  return toSvg(String(team.id), 20);
}

const route = useRoute();

const currentTaskId = computed(() =>
  route.name === "task" ? String(route.params.id ?? "") : "",
);

watch(
  [currentTaskId, notifications],
  ([id]) => {
    if (!id) return;
    const unseen = notifications.value.some(
      (item) => item.taskId === id && !item.readAt,
    );
    if (unseen) void markTaskRead(id);
  },
  { immediate: true },
);

function isAgentActive(username: string): boolean {
  return route.name === "agent" && route.params.username === username;
}

function isUserActive(username: string): boolean {
  return route.name === "user" && route.params.username === username;
}

function isTeamActive(id: number): boolean {
  return route.name === "team" && String(route.params.id) === String(id);
}

function isProjectsSectionActive(): boolean {
  return route.name === "projects" || route.name === "project";
}

function isAgentsSectionActive(): boolean {
  return route.name === "agents" || route.name === "agent";
}

function isTeamsSectionActive(): boolean {
  return route.name === "teams" || route.name === "team";
}

function isUsersSectionActive(): boolean {
  return route.name === "users" || route.name === "user";
}
</script>
