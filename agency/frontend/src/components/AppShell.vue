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
        :title="collapsed ? $t('nav.expandSidebar') : $t('nav.collapseSidebar')"
        :aria-label="collapsed ? $t('nav.expandSidebar') : $t('nav.collapseSidebar')"
        :aria-expanded="!collapsed"
        @click="toggle()"
      )
        PanelLeftOpenIcon(v-if="collapsed")
        PanelLeftCloseIcon(v-else)
      RouterLink.text-primary.flex.min-w-0.flex-1.items-center.gap-2.rounded-md.font-semibold(
        :to="{ name: 'agency-tasks' }"
        :title="$t('nav.tasks')"
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
        :title="$t('nav.newTask')"
        :aria-label="$t('nav.newTask')"
        @click="openNewTask"
      )
        PlusIcon

    .flex.flex-1.flex-col.overflow-auto(
      :class="collapsed ? 'px-1.5 py-2' : 'p-2'"
      data-testid="sidebar-directory"
    )
      SidebarSection(
        :icon="FolderIcon"
        :to="{ name: 'projects' }"
        :title="$t('nav.projects')"
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
          span {{ $t("nav.allProjects") }}
      .mt-2.border-t.pt-2
        SidebarSection(
          :icon="GlobeIcon"
          :to="{ name: 'sites' }"
          :title="$t('nav.sites')"
          :active="route.name === 'sites' || route.name === 'site'"
          :open="isSectionOpen('sites')"
          :collapsed="collapsed"
          :count="sidebarSites.length"
          @toggle="toggleSection('sites')"
        )
          p.text-muted-foreground.px-3.py-1.text-xs(
            v-if="!sidebarSites.length && !collapsed"
          )
            | {{ $t("nav.noAgencySites") }}
          RouterLink.flex.items-center.gap-2.rounded-md.px-3.py-2.text-sm.font-medium.transition-colors(
            v-for="site in sidebarSites"
            :key="site.id"
            :to="{ name: 'site', params: { id: site.id } }"
            :title="collapsed ? site.title || site.id : undefined"
            :class="[route.name === 'site' && route.params.id === site.id ? 'text-primary' : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground', collapsed ? 'justify-center px-0' : '']"
          )
            GlobeIcon.size-4.shrink-0
            span(:class="collapsed ? 'sr-only' : 'min-w-0 truncate'")
              | {{ site.title || site.id }}
          RouterLink.text-muted-foreground.flex.items-center.gap-2.rounded-md.px-3.py-2.text-xs.font-medium(
            v-if="!collapsed"
            :to="{ name: 'sites' }"
          )
            span {{ $t("nav.allSites") }}
      .mt-2.border-t.pt-2
        SidebarSection(
          :icon="BotIcon"
          :to="{ name: 'agents' }"
          :title="$t('nav.agents')"
          :active="isAgentsSectionActive()"
          :open="isSectionOpen('agents')"
          :collapsed="collapsed"
          :count="sidebarAgents.length"
          @toggle="toggleSection('agents')"
        )
          p.text-muted-foreground.px-3.py-1.text-xs(
            v-if="!sidebarAgents.length && !collapsed"
          )
            | {{ $t("nav.noAgents") }}
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
                :title="item.status === AgentStatus.ONLINE ? $t('nav.online') : $t('nav.offline')"
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
            span {{ $t("nav.manageAgents") }}
      .mt-2.border-t.pt-2
        SidebarSection(
          :icon="UsersIcon"
          :to="{ name: 'teams' }"
          :title="$t('nav.teams')"
          :active="isTeamsSectionActive()"
          :open="isSectionOpen('teams')"
          :collapsed="collapsed"
          :count="sidebarTeams.length"
          @toggle="toggleSection('teams')"
        )
          p.text-muted-foreground.px-3.py-1.text-xs(
            v-if="!sidebarTeams.length && !collapsed"
          )
            | {{ $t("nav.notInTeam") }}
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
            span {{ $t("nav.allTeams") }}
      .mt-2.border-t.pt-2
        SidebarSection(
          :icon="UserIcon"
          :to="{ name: 'users' }"
          :title="$t('nav.users')"
          :active="isUsersSectionActive()"
          :open="isSectionOpen('users')"
          :collapsed="collapsed"
          :count="sidebarUsers.length"
          @toggle="toggleSection('users')"
        )
          p.text-muted-foreground.px-3.py-1.text-xs(
            v-if="!sidebarUsers.length && !collapsed"
          )
            | {{ $t("nav.noTeammates") }}
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
          :title="$t('nav.inbox')"
          :aria-label="$t('nav.inbox')"
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
          :title="$t('nav.userSettings')"
          :aria-label="$t('nav.userSettings')"
        )
          RouterLink.flex.items-center.justify-center(
            :to="{ name: 'account' }"
          )
            SettingsIcon
    footer.flex.shrink-0.border-t(
      :class="collapsed ? 'flex-col items-center gap-1 px-1.5 py-2' : 'items-center gap-1 px-2 py-2'"
    )
      span.text-muted-foreground.truncate.font-mono(
        :title="$t('nav.appVersion')"
        :class="collapsed ? 'sr-only' : 'flex-1 text-[10px]'"
      ) {{ appVersion }}
      Button(
        v-if="isAdmin"
        as-child
        variant="ghost"
        size="icon-sm"
        :title="$t('nav.orgSettings')"
        :aria-label="$t('nav.orgSettings')"
      )
        RouterLink.flex.items-center.justify-center(:to="{ name: 'settings' }")
          SlidersHorizontalIcon
      Button(
        as-child
        variant="ghost"
        size="icon-sm"
        :title="$t('nav.documentation')"
        :aria-label="$t('nav.documentation')"
      )
        a.flex.items-center.justify-center(
          href="/docs/"
          rel="noopener"
          target="_blank"
        )
          BookOpenIcon
      LanguageSwitcher
      Button(
        variant="ghost"
        size="icon-sm"
        :title="isDark ? $t('nav.lightTheme') : $t('nav.darkTheme')"
        :aria-label="isDark ? $t('nav.lightTheme') : $t('nav.darkTheme')"
        @click="toggleDark()"
      )
        MoonIcon(v-if="isDark")
        SunIcon(v-else)
  .flex.flex-1.flex-col.overflow-hidden
    main.flex-1.overflow-auto
      RouterView(v-slot="{ Component }")
        component(:key="route.fullPath" :is="Component")
  TaskComposerDialog(
    v-model:open="newTaskOpen"
    :default-project-id="pageProjectId"
  )
</template>

<script setup lang="ts">
import ConcurrencyTrack from "@/components/concurrency/ConcurrencyTrack.vue";
import LanguageSwitcher from "@/components/LanguageSwitcher.vue";
import ProjectsNav from "@/components/ProjectsNav.vue";
import SidebarSection from "@/components/SidebarSection.vue";
import TaskComposerDialog from "@/components/tasks/TaskComposerDialog.vue";
import { Button } from "@/components/ui/button";
import { useAgents } from "@/composables/useAgents";
import { attentionPreference } from "@/composables/useAttentionPreferences";
import { useDirectory } from "@/composables/useDirectory";
import { useFleetConcurrency } from "@/composables/useFleetConcurrency";
import { useNotifications } from "@/composables/useNotifications";
import { usePageProject } from "@/composables/usePageProject";
import { usePageTitle } from "@/composables/usePageTitle";
import {
  startRealtime,
  stopRealtime,
  useRealtime,
} from "@/composables/useRealtime";
import { useSettings } from "@/composables/useSettings";
import { useSidebar } from "@/composables/useSidebar";
import { useSites } from "@/composables/useSites";
import { useTheme } from "@/composables/useTheme";
import { useUsers } from "@/composables/useUsers";
import { currentAccount, type AccountActor } from "@/lib/account";
import {
  applyFavicon,
  armAttentionAudio,
  armAttentionPermission,
} from "@/lib/attention";
import { ATTENTION_EVENTS } from "@/lib/attentionEvents";
import { bindAttentionNavigation } from "@/lib/attentionToast";
import type { Team } from "@/lib/team";
import { currentUser } from "@/lib/user";
import { AgentStatus, type Agent } from "@cherrypi/sdk";
import {
  BookOpenIcon,
  BotIcon,
  FolderIcon,
  GlobeIcon,
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
import { toSvg } from "jdenticon";
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  watchEffect,
} from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router";

const { t } = useI18n();

// vite.config.ts.
const appVersion = __APP_VERSION__;

const { isDark, toggleDark } = useTheme();

const router = useRouter();
bindAttentionNavigation((to) => void router.push(to));

const { settings, loadSettings } = useSettings();
const { collapsed, toggle, isSectionOpen, toggleSection } = useSidebar();

const newTaskOpen = ref(false);
const { projectId: pageProjectId, resolve: resolvePageProject } =
  usePageProject();

function openNewTask() {
  void resolvePageProject();
  newTaskOpen.value = true;
}
const { agents, loadAgents } = useAgents();
const { users, loadUsers } = useUsers();

const { sites: sidebarSites, load: loadSites } = useSites();
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
  () => currentAccount.value?.name || username.value || t("common.user"),
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
  return stat.waiting
    ? t("nav.fleetBusy", {
        used: stat.used,
        cap: stat.cap,
        waiting: stat.waiting,
      })
    : t("nav.fleet", { used: stat.used, cap: stat.cap });
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
  void loadSites({ agencyWideOnly: true });
  void loadDirectory();
  startNotifications();
  startRealtime();

  armAttentionAudio();

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

useRealtime(["sites"], () => {
  void loadSites({ agencyWideOnly: true });
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

<i18n lang="yaml">
en:
  common:
    user: "User"
    save: "Save"
    saveChanges: "Save changes"
    cancel: "Cancel"
    close: "Close"
    delete: "Delete"
    edit: "Edit"
    create: "Create"
    add: "Add"
    remove: "Remove"
    search: "Search"
    loading: "Loading…"
    saving: "Saving…"
    none: "None"
    copy: "Copy"
    copied: "Copied"
    refresh: "Refresh"
    confirm: "Confirm"
    enabled: "Enabled"
    disabled: "Disabled"
    optional: "Optional"
    name: "Name"
    description: "Description"
    title: "Title"
    status: "Status"
    actions: "Actions"
    settings: "Settings"
    apply: "Apply"
    reset: "Reset"
    clear: "Clear"
    all: "All"
    dismiss: "Dismiss"
    open: "Open"
    unknown: "Unknown"
    never: "Never"
    default: "Default"
    username: "Username"
    password: "Password"
    email: "E-mail"
    newPassword: "New password"
    confirmPassword: "Confirm password"
    loadMore: "Load more"
    enable: "Enable"
    revoke: "Revoke"
    verify: "Verify"
    disable: "Disable"
    unlink: "Unlink"
    expiresIn: "Expires in"
    neverExpires: "Never expires"
    kind: "Kind"
    administrator: "an administrator"
  nav:
    tasks: "Tasks"
    projects: "Projects"
    allProjects: "All projects"
    sites: "Sites"
    allSites: "All sites"
    noAgencySites: "No agency-wide sites"
    agents: "Agents"
    noAgents: "No agents yet"
    manageAgents: "Manage agents"
    teams: "Teams"
    notInTeam: "Not in a team yet"
    allTeams: "All teams"
    users: "Users"
    noTeammates: "No teammates yet"
    inbox: "Inbox"
    userSettings: "User settings"
    orgSettings: "Organization settings"
    documentation: "Documentation"
    newTask: "New task"
    expandSidebar: "Expand sidebar"
    collapseSidebar: "Collapse sidebar"
    appVersion: "App version"
    online: "Online"
    offline: "Offline"
    lightTheme: "Switch to light theme"
    darkTheme: "Switch to dark theme"
    fleet: "{used}/{cap} slots held"
    fleetBusy: "{used}/{cap} slots held · {waiting} waiting"
  taskState:
    backlog: "Backlog"
    waiting: "Waiting"
    queued: "Queued"
    running: "Running"
    input: "Input"
    settled: "Settled"
    failed: "Failed"
    completed: "Completed"
    cancelled: "Cancelled"
    unknown: "Unknown"
  projectStatus:
    input: "Input required"
    running: "Running"
    settled: "Settled"
    waiting: "Waiting"
    queued: "Queued"
  taskColumns:
    dragColumn: "Drag to reorder {column}"
    moveUp: "Move {column} up"
    moveDown: "Move {column} down"
    showColumn: "Show {column} column"
  taskColumn:
    state: "State"
    project: "Project"
    title: "Title"
    progress: "Progress"
    badges: "Badges"
    links: "Links"
    tags: "Tags"
    references: "References"
    priority: "Priority"
    size: "Size"
    assignee: "Assignee"
    cost: "Cost"
    progressHint: "The agent's reported plan: completed steps out of total"
    badgesHint: "Status an installed plugin reports for the task (the pull request, CI)"
    costHint: "Cost across settled and in-flight runs"
  taskPriority:
    low: "Low"
    normal: "Normal"
    high: "High"
    critical: "Critical"
    lowWithDimension: "Low priority"
    normalWithDimension: "Normal priority"
    highWithDimension: "High priority"
    criticalWithDimension: "Critical priority"
  taskSize:
    small: "Small"
    medium: "Medium"
    big: "Big"
    epic: "Epic"
    smallWithDimension: "Small size"
    mediumWithDimension: "Medium size"
    bigWithDimension: "Big size"
    epicWithDimension: "Epic size"
  projectRole:
    owner: "Owner"
    member: "Member"
    viewer: "Viewer"
    ownerHint: "Manage access, prompt and edit"
    memberHint: "Prompt and edit"
    viewerHint: "View and comment only"
  usage:
    range:
      allTime: "All time"
      allTimeLower: "all time"
      lastDays: "last {days} days"
      lastDaysPill: "Last {days} days"
    unknownAgent: "Unknown agent"
    unknownProject: "Unknown project"
    agentFallback: "Agent {id}"
    projectFallback: "Project {id}"
  promptHistory:
    version: "Version {version}"
  soundSet:
    classic: "Classic"
    classicHint: "The original CherryPi chime, note for note."
    cherry: "Cherry"
    cherryHint: "Warm, rounded chimes. The default."
    wood: "Wood"
    woodHint: "Marimba and woodblock: a short knock with an organic body."
    glass: "Glass"
    glassHint: "Bell and glass partials with a short room around them."
    arcade: "Arcade"
    arcadeHint: "Dry chiptune blips: square waves, fast and bright."
  agentVersion:
    current: "Up to date"
    outdated: "Binary should be updated"
    incompatible: "Binary incompatible"
    none: "No version reported"
  attentionToast:
    openTask: "Open task"
    openTeams: "Open teams"
    openPrompt: "Open prompt"
  concurrency:
    agentNotFound: "Agent not found"
  sketch:
    new: "New sketch"
  lightbox:
    taskAttachment: "Task attachment"
    artifactFile: "Artifact file"
    artifactSource: "Artifact · {name}"
    mermaidDiagram: "Mermaid diagram"
    taskTranscript: "Task transcript"
    unsentAttachment: "Unsent attachment"
    inlineImage: "Inline image"
    inlineImageHost: "Inline image · {host}"
  mermaid:
    controls: "Diagram controls"
    panLeft: "Pan left"
    panUp: "Pan up"
    panDown: "Pan down"
    panRight: "Pan right"
    zoomOut: "Zoom out"
    zoomIn: "Zoom in"
    resetZoom: "Reset zoom"
    copySource: "Copy diagram source"
    openEnlarged: "Open diagram enlarged"
  wait:
    blocker:
      reservedLabel: "Reserved for #{tag}"
      reservedDetail: "{slots} slot(s) are kept for #{tag} tasks ({used} held); a task without #{tag} needs more room than that leaves."
      offlineLabel: "Agent offline"
      offlineDetail: "The agent has no tunnel connection."
      overallLabel: "Overall {used}/{cap}"
      overallDetail: "Every agent slot is held."
      projectLabel: "Project {used}/{cap}"
      projectDetail: "The project cap is full."
      tagLabel: "#{tag} {used}/{cap}"
      tagDetail: "The global #{tag} cap is full."
      projectTagLabel: "#{tag} in project {used}/{cap}"
      projectTagDetail: "The project × tag cap for #{tag} is full."
      referenceLabelOne: "Waiting for {task}"
      referenceLabelMany: "Waiting for {count} tasks"
      aTask: "a task"
    row:
      allProjects: "All projects"
      agentCap: "agent cap {cap}"
      globalNameCap: "global name cap {cap}"
      projectTag: "{project} × #{tag}"
      projectTagCap: "project × tag cap {cap}"
    reason:
      blockersPrefix: "Waiting for tasks to become completed:"
      blockingTasks: "Waiting for blocking tasks to become completed"
      freeSlot: "Waiting for free agent slot"
  tagPreset:
    feature: "New functionality"
    bug: "Defect or regression"
    chore: "Maintenance and housekeeping"
    docs: "Documentation"
    refactor: "Restructuring without behaviour change"
    test: "Test coverage"
    perf: "Performance work"
    design: "Design and UX"
    infra: "Build, CI and infrastructure"
    question: "Needs an answer or decision"
  taskProgress:
    allComplete: "All steps complete"
  pluginHandler:
    http: "HTTP requests"
    tool: "Agent tools"
    event: "Agency events"
    tick: "Scheduled ticks"
    taskTools: "Task tool filter"
  pluginPermission:
    network: "Outbound network"
    networkAnyHost: "Outbound network: any host"
    routes: "Public HTTP routes"
    events: "Agency events"
    tools: "Agent tools"
    schedule: "Scheduled ticks"
    scheduleDetail: "Calls handle_tick every {seconds}s while the plugin is enabled."
    settings: "Settings surfaces"
    tasks: "Task creation"
    tasksDetail: "May create tasks in projects where it is installed, capped by each project's switches."
  pluginScope:
    org: "Organization"
    project: "Project"
    agent: "Agent"
    user: "User"
  speech:
    errors:
      denied: "Microphone access was denied. Allow it in the browser settings, then try again."
      network: "Browser speech recognition needs a network connection."
      noMicrophone: "No microphone was found. Check the system input device and try again."
      language: "Browser speech recognition does not support this language."
      failed: "Speech recognition failed: {error}"
  terminalShortcuts:
    closeTab: "Close active tab"
    nextTab: "Next tab"
    prevTab: "Previous tab"
    showShortcuts: "Show these shortcuts"
    switchTabN: "Switch to tab N"
  pageTitle:
    designStudio: "Design studio"
    signIn: "Sign in"
    signingIn: "Signing in"
    resetPassword: "Reset password"
    acceptInvitation: "Accept invitation"
    orgSettings: "Organization settings"
    agents: "Agents"
    tasks: "Tasks"
    task: "Task"
    sites: "Sites"
    site: "Site"
    usage: "Usage"
    inbox: "Inbox"
    projects: "Projects"
    teams: "Teams"
    team: "Team"
    project: "Project"
    account: "Account"
    agent: "Agent"
    users: "Users"
    user: "User"
  argType:
    text: "Text"
    multiline: "Multiline text"
    number: "Number"
    boolean: "Boolean"
    select: "Select"
  desktop:
    fullscreenHint: "Enter fullscreen and send every shortcut (Ctrl+W, Ctrl+T, …) to the desktop. Esc exits."
    fullscreenHintNoLock: "Enter fullscreen. This browser cannot capture shortcuts it reserves for itself."
ru:
  common:
    user: "Пользователь"
    save: "Сохранить"
    saveChanges: "Сохранить изменения"
    cancel: "Отмена"
    close: "Закрыть"
    delete: "Удалить"
    edit: "Редактировать"
    create: "Создать"
    add: "Добавить"
    remove: "Убрать"
    search: "Поиск"
    loading: "Загрузка…"
    saving: "Сохранение…"
    none: "Нет"
    copy: "Скопировать"
    copied: "Скопировано"
    refresh: "Обновить"
    confirm: "Подтвердить"
    enabled: "Включено"
    disabled: "Отключено"
    optional: "Необязательно"
    name: "Название"
    description: "Описание"
    title: "Заголовок"
    status: "Статус"
    actions: "Действия"
    settings: "Настройки"
    apply: "Применить"
    reset: "Сбросить"
    clear: "Очистить"
    all: "Все"
    dismiss: "Скрыть"
    open: "Открыть"
    unknown: "Неизвестно"
    never: "Никогда"
    default: "По умолчанию"
    username: "Имя пользователя"
    password: "Пароль"
    email: "E-mail"
    newPassword: "Новый пароль"
    confirmPassword: "Подтвердите пароль"
    loadMore: "Показать ещё"
    enable: "Включить"
    revoke: "Отозвать"
    verify: "Подтвердить"
    disable: "Отключить"
    unlink: "Отвязать"
    expiresIn: "Срок действия"
    neverExpires: "Бессрочно"
    kind: "Тип"
    administrator: "администратор"
  nav:
    tasks: "Задачи"
    projects: "Проекты"
    allProjects: "Все проекты"
    sites: "Сайты"
    allSites: "Все сайты"
    noAgencySites: "Нет сайтов агентства"
    agents: "Агенты"
    noAgents: "Агентов пока нет"
    manageAgents: "Управление агентами"
    teams: "Команды"
    notInTeam: "Пока нет команды"
    allTeams: "Все команды"
    users: "Пользователи"
    noTeammates: "Коллег пока нет"
    inbox: "Уведомления"
    userSettings: "Настройки пользователя"
    orgSettings: "Настройки организации"
    documentation: "Документация"
    newTask: "Новая задача"
    expandSidebar: "Развернуть панель"
    collapseSidebar: "Свернуть панель"
    appVersion: "Версия приложения"
    online: "В сети"
    offline: "Не в сети"
    lightTheme: "Переключить на светлую тему"
    darkTheme: "Переключить на тёмную тему"
    fleet: "Занято слотов: {used}/{cap}"
    fleetBusy: "Занято слотов: {used}/{cap} · в очереди: {waiting}"
  taskState:
    backlog: "Бэклог"
    waiting: "Ожидает"
    queued: "В очереди"
    running: "Выполняется"
    input: "Нужен ввод"
    settled: "Завершена"
    failed: "Ошибка"
    completed: "Выполнена"
    cancelled: "Отменена"
    unknown: "Неизвестно"
  projectStatus:
    input: "Нужен ввод"
    running: "Выполняется"
    settled: "Завершён"
    waiting: "Ожидает"
    queued: "В очереди"
  taskColumns:
    dragColumn: "Потяните, чтобы изменить порядок: {column}"
    moveUp: "Переместить {column} выше"
    moveDown: "Переместить {column} ниже"
    showColumn: "Показывать колонку {column}"
  taskColumn:
    state: "Состояние"
    project: "Проект"
    title: "Название"
    progress: "Прогресс"
    badges: "Значки"
    links: "Ссылки"
    tags: "Теги"
    references: "Связи"
    priority: "Приоритет"
    size: "Размер"
    assignee: "Исполнитель"
    cost: "Стоимость"
    progressHint: "План, опубликованный агентом: выполненные шаги из общего числа"
    badgesHint: "Статус, который установленный плагин сообщает для задачи (pull request, CI)"
    costHint: "Стоимость по завершённым и текущим запускам"
  taskPriority:
    low: "Низкий"
    normal: "Обычный"
    high: "Высокий"
    critical: "Критический"
    lowWithDimension: "Низкий приоритет"
    normalWithDimension: "Обычный приоритет"
    highWithDimension: "Высокий приоритет"
    criticalWithDimension: "Критический приоритет"
  taskSize:
    small: "Малый"
    medium: "Средний"
    big: "Большой"
    epic: "Огромный"
    smallWithDimension: "Малый размер"
    mediumWithDimension: "Средний размер"
    bigWithDimension: "Большой размер"
    epicWithDimension: "Огромный размер"
  projectRole:
    owner: "Владелец"
    member: "Участник"
    viewer: "Наблюдатель"
    ownerHint: "Управляет доступом, отправляет промпты и редактирует"
    memberHint: "Отправляет промпты и редактирует"
    viewerHint: "Только просмотр и комментарии"
  usage:
    range:
      allTime: "За всё время"
      allTimeLower: "за всё время"
      lastDays: "последние {days} дн."
      lastDaysPill: "Последние {days} дн."
    unknownAgent: "Неизвестный агент"
    unknownProject: "Неизвестный проект"
    agentFallback: "Агент {id}"
    projectFallback: "Проект {id}"
  promptHistory:
    version: "Версия {version}"
  soundSet:
    classic: "Классика"
    classicHint: "Оригинальный звон CherryPi, нота в ноту."
    cherry: "Вишня"
    cherryHint: "Тёплые, мягкие звоны. По умолчанию."
    wood: "Дерево"
    woodHint: "Маримба и деревянный блок: короткий стук с живым телом."
    glass: "Стекло"
    glassHint: "Колокол и стеклянные обертоны с коротким залом."
    arcade: "Аркада"
    arcadeHint: "Сухие чиптюн-сигналы: прямоугольные волны, быстро и ярко."
  agentVersion:
    current: "Актуально"
    outdated: "Сборку нужно обновить"
    incompatible: "Сборка несовместима"
    none: "Версия не сообщена"
  attentionToast:
    openTask: "Открыть задачу"
    openTeams: "Открыть команды"
    openPrompt: "Открыть промпт"
  concurrency:
    agentNotFound: "Агент не найден"
  sketch:
    new: "Новый набросок"
  lightbox:
    taskAttachment: "Вложение задачи"
    artifactFile: "Файл артефакта"
    artifactSource: "Артефакт · {name}"
    mermaidDiagram: "Диаграмма Mermaid"
    taskTranscript: "Переписка задачи"
    unsentAttachment: "Неотправленное вложение"
    inlineImage: "Встроенное изображение"
    inlineImageHost: "Встроенное изображение · {host}"
  mermaid:
    controls: "Управление диаграммой"
    panLeft: "Сдвинуть влево"
    panUp: "Сдвинуть вверх"
    panDown: "Сдвинуть вниз"
    panRight: "Сдвинуть вправо"
    zoomOut: "Уменьшить"
    zoomIn: "Увеличить"
    resetZoom: "Сбросить масштаб"
    copySource: "Скопировать исходник диаграммы"
    openEnlarged: "Открыть диаграмму крупнее"
  wait:
    blocker:
      reservedLabel: "Зарезервировано для #{tag}"
      reservedDetail: "Для задач #{tag} оставлено слотов: {slots} (занято {used}); задаче без #{tag} нужно больше места, чем остаётся."
      offlineLabel: "Агент не в сети"
      offlineDetail: "У агента нет соединения через туннель."
      overallLabel: "Общий {used}/{cap}"
      overallDetail: "Все слоты агента заняты."
      projectLabel: "Проект {used}/{cap}"
      projectDetail: "Лимит проекта исчерпан."
      tagLabel: "#{tag} {used}/{cap}"
      tagDetail: "Глобальный лимит #{tag} исчерпан."
      projectTagLabel: "#{tag} в проекте {used}/{cap}"
      projectTagDetail: "Лимит проект × тег для #{tag} исчерпан."
      referenceLabelOne: "Ожидает {task}"
      referenceLabelMany: "Ожидает задач: {count}"
      aTask: "задачу"
    row:
      allProjects: "Все проекты"
      agentCap: "лимит агента {cap}"
      globalNameCap: "глобальный лимит тега {cap}"
      projectTag: "{project} × #{tag}"
      projectTagCap: "лимит проект × тег {cap}"
    reason:
      blockersPrefix: "Ожидает завершения задач:"
      blockingTasks: "Ожидает завершения блокирующих задач"
      freeSlot: "Ожидает свободного слота агента"
  tagPreset:
    feature: "Новая функциональность"
    bug: "Дефект или регрессия"
    chore: "Обслуживание и уборка"
    docs: "Документация"
    refactor: "Реструктуризация без изменения поведения"
    test: "Тестовое покрытие"
    perf: "Работа над производительностью"
    design: "Дизайн и UX"
    infra: "Сборка, CI и инфраструктура"
    question: "Нужен ответ или решение"
  taskProgress:
    allComplete: "Все шаги выполнены"
  pluginHandler:
    http: "HTTP-запросы"
    tool: "Инструменты агента"
    event: "События агентства"
    tick: "Плановые тики"
    taskTools: "Фильтр инструментов задачи"
  pluginPermission:
    network: "Исходящая сеть"
    networkAnyHost: "Исходящая сеть: любой хост"
    routes: "Публичные HTTP-маршруты"
    events: "События агентства"
    tools: "Инструменты агента"
    schedule: "Плановые тики"
    scheduleDetail: "Вызывает handle_tick каждые {seconds} с, пока плагин включён."
    settings: "Поверхности настроек"
    tasks: "Создание задач"
    tasksDetail: "Может создавать задачи в проектах, где установлен, в пределах переключателей каждого проекта."
  pluginScope:
    org: "Организация"
    project: "Проект"
    agent: "Агент"
    user: "Пользователь"
  speech:
    errors:
      denied: "Доступ к микрофону запрещён. Разрешите его в настройках браузера и повторите попытку."
      network: "Распознаванию речи в браузере нужно сетевое подключение."
      noMicrophone: "Микрофон не найден. Проверьте устройство ввода в системе и повторите попытку."
      language: "Распознавание речи в браузере не поддерживает этот язык."
      failed: "Распознавание речи не удалось: {error}"
  terminalShortcuts:
    closeTab: "Закрыть активную вкладку"
    nextTab: "Следующая вкладка"
    prevTab: "Предыдущая вкладка"
    showShortcuts: "Показать эти сочетания"
    switchTabN: "Перейти к вкладке N"
  pageTitle:
    designStudio: "Студия дизайна"
    signIn: "Вход"
    signingIn: "Вход"
    resetPassword: "Сброс пароля"
    acceptInvitation: "Принятие приглашения"
    orgSettings: "Настройки организации"
    agents: "Агенты"
    tasks: "Задачи"
    task: "Задача"
    sites: "Сайты"
    site: "Сайт"
    usage: "Использование"
    inbox: "Уведомления"
    projects: "Проекты"
    teams: "Команды"
    team: "Команда"
    project: "Проект"
    account: "Аккаунт"
    agent: "Агент"
    users: "Пользователи"
    user: "Пользователь"
  argType:
    text: "Текст"
    multiline: "Многострочный текст"
    number: "Число"
    boolean: "Логическое"
    select: "Список"
  desktop:
    fullscreenHint: "Войдите в полноэкранный режим, чтобы отправлять на рабочий стол все сочетания (Ctrl+W, Ctrl+T, …). Esc — выход."
    fullscreenHintNoLock: "Войдите в полноэкранный режим. Этот браузер не отдаёт сочетания, закреплённые за собой."
</i18n>
