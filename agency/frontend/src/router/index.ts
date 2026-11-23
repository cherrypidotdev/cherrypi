// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import AppShell from "@/components/AppShell.vue";
import GroupAccessPanel from "@/components/groups/GroupAccessPanel.vue";
import GroupMembersPanel from "@/components/groups/GroupMembersPanel.vue";
import GroupOverviewPanel from "@/components/groups/GroupOverviewPanel.vue";
import GroupPromptsPanel from "@/components/groups/GroupPromptsPanel.vue";
import GroupSettingsPanel from "@/components/groups/GroupSettingsPanel.vue";
import GroupVariablesPanel from "@/components/groups/GroupVariablesPanel.vue";
import PluginDetailPanel from "@/components/plugins/PluginDetailPanel.vue";
import PluginsPanel from "@/components/settings/PluginsPanel.vue";
import { useAgents } from "@/composables/useAgents";
import { loadLocalePreference } from "@/composables/useLocale";
import { setPageTitle } from "@/composables/usePageTitle";
import { t } from "@/i18n/translate";
import { ensureSession } from "@/lib/account";
import { currentAgentId, currentAgentUsername } from "@/lib/sdk";
import AccountView from "@/views/AccountView.vue";
import AgencyTasksView from "@/views/AgencyTasksView.vue";
import AgencyUsageView from "@/views/AgencyUsageView.vue";
import AgentView from "@/views/AgentView.vue";
import AgentsView from "@/views/AgentsView.vue";
import GroupView from "@/views/GroupView.vue";
import GroupsView from "@/views/GroupsView.vue";
import InboxView from "@/views/InboxView.vue";
import InviteView from "@/views/InviteView.vue";
import KnowledgeView from "@/views/KnowledgeView.vue";
import LibraryView from "@/views/LibraryView.vue";
import LoginView from "@/views/LoginView.vue";
import MagicLinkView from "@/views/MagicLinkView.vue";
import ProjectSettingsView from "@/views/ProjectSettingsView.vue";
import ProjectAccessSettingsTab from "@/views/ProjectSettingsView/AccessSettingsTab.vue";
import ProjectDangerSettingsTab from "@/views/ProjectSettingsView/DangerSettingsTab.vue";
import ProjectGeneralSettingsTab from "@/views/ProjectSettingsView/GeneralSettingsTab.vue";
import ProjectModelsSettingsTab from "@/views/ProjectSettingsView/ModelsSettingsTab.vue";
import ProjectPluginSettingsTab from "@/views/ProjectSettingsView/PluginSettingsTab.vue";
import ProjectPluginsSettingsTab from "@/views/ProjectSettingsView/PluginsSettingsTab.vue";
import ProjectTasksSettingsTab from "@/views/ProjectSettingsView/TasksSettingsTab.vue";
import ProjectTemplatesSettingsTab from "@/views/ProjectSettingsView/TemplatesSettingsTab.vue";
import ProjectVariablesSettingsTab from "@/views/ProjectSettingsView/VariablesSettingsTab.vue";
import ProjectView from "@/views/ProjectView.vue";
import ProjectConcurrencyTab from "@/views/ProjectView/ProjectConcurrencyTab.vue";
import ProjectKnowledgeTab from "@/views/ProjectView/ProjectKnowledgeTab.vue";
import ProjectOverviewTab from "@/views/ProjectView/ProjectOverviewTab.vue";
import ProjectUsageTab from "@/views/ProjectView/ProjectUsageTab.vue";
import ProjectsView from "@/views/ProjectsView.vue";
import ProviderEditorView from "@/views/ProviderEditorView.vue";
import ResetPasswordView from "@/views/ResetPasswordView.vue";
import SettingsView from "@/views/SettingsView.vue";
import SettingsAdminTab from "@/views/SettingsView/AdminSettingsTab.vue";
import SettingsAuthTab from "@/views/SettingsView/AuthSettingsTab.vue";
import SettingsGeneralTab from "@/views/SettingsView/GeneralSettingsTab.vue";
import SettingsIntegrationsTab from "@/views/SettingsView/IntegrationsSettingsTab.vue";
import SettingsLogsTab from "@/views/SettingsView/LogsSettingsTab.vue";
import SettingsModelsTab from "@/views/SettingsView/ModelsSettingsTab.vue";
import SettingsVariablesTab from "@/views/SettingsView/VariablesSettingsTab.vue";
import SiteView from "@/views/SiteView.vue";
import SitesView from "@/views/SitesView.vue";
import TaskView from "@/views/TaskView.vue";
import TeamView from "@/views/TeamView.vue";
import TeamsView from "@/views/TeamsView.vue";
import UserView from "@/views/UserView.vue";
import UsersView from "@/views/UsersView.vue";
import {
  createRouter,
  createWebHistory,
  type RouteRecordRaw,
} from "vue-router";

const publicRoutes = new Set([
  "login",
  "magic-link",
  "reset-password",
  "invite",
  ...(import.meta.env.DEV ? ["dev-studio"] : []),
]);

// branch is dead in a production build (Vite replaces the flag with `false`),

const devRoutes: RouteRecordRaw[] = import.meta.env.DEV
  ? [
      {
        path: "/dev-studio",
        name: "dev-studio",
        component: () => import("@/views/DevStudioView.vue"),
        meta: { titleKey: "pageTitle.designStudio" },
      },
    ]
  : [];

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: LoginView,
      meta: { titleKey: "pageTitle.signIn" },
    },
    {
      path: "/magic-link",
      name: "magic-link",
      component: MagicLinkView,
      meta: { titleKey: "pageTitle.signingIn" },
    },
    {
      path: "/reset-password",
      name: "reset-password",
      component: ResetPasswordView,
      meta: { titleKey: "pageTitle.resetPassword" },
    },
    {
      path: "/invite",
      name: "invite",
      component: InviteView,
      meta: { titleKey: "pageTitle.acceptInvitation" },
    },
    ...devRoutes,
    {
      path: "/",
      component: AppShell,
      children: [
        { path: "", redirect: "/tasks" },
        {
          path: "agents",
          name: "agents",
          component: AgentsView,
          meta: { titleKey: "pageTitle.agents" },
        },

        {
          path: "tasks",
          name: "agency-tasks",
          component: AgencyTasksView,
          meta: { titleKey: "pageTitle.tasks" },
        },
        {
          path: "tasks/:id",
          name: "task",
          component: TaskView,
          props: true,
          meta: { titleKey: "pageTitle.task" },
        },
        {
          path: "knowledge",
          name: "knowledge",
          component: KnowledgeView,
          meta: { titleKey: "pageTitle.knowledge" },
        },
        {
          path: "sites",
          name: "sites",
          component: SitesView,
          meta: { titleKey: "pageTitle.sites" },
        },
        {
          path: "sites/:id",
          name: "site",
          component: SiteView,
          props: true,
          meta: { titleKey: "pageTitle.site" },
        },
        {
          path: "usage",
          name: "agency-usage",
          component: AgencyUsageView,
          meta: { titleKey: "pageTitle.usage" },
        },
        {
          path: "inbox",
          name: "inbox",
          component: InboxView,
          meta: { titleKey: "pageTitle.inbox" },
        },
        {
          path: "projects",
          name: "projects",
          component: ProjectsView,
          meta: { titleKey: "pageTitle.projects" },
        },
        {
          path: "groups",
          name: "groups",
          component: GroupsView,
          meta: { titleKey: "pageTitle.groups" },
        },
        {
          path: "groups/:slug",
          component: GroupView,
          children: [
            {
              path: "",
              name: "group-overview",
              component: GroupOverviewPanel,
              meta: { titleKey: "pageTitle.group" },
            },
            {
              path: "members",
              name: "group-members",
              component: GroupMembersPanel,
              meta: { titleKey: "pageTitle.group" },
            },
            {
              path: "prompts",
              name: "group-prompts",
              component: GroupPromptsPanel,
              meta: { titleKey: "pageTitle.group" },
            },
            {
              path: "access",
              name: "group-access",
              component: GroupAccessPanel,
              meta: { titleKey: "pageTitle.group" },
            },
            {
              path: "variables",
              name: "group-variables",
              component: GroupVariablesPanel,
              meta: { titleKey: "pageTitle.group" },
            },
            {
              path: "settings",
              name: "group-settings",
              component: GroupSettingsPanel,
              meta: { titleKey: "pageTitle.group" },
            },
          ],
        },
        {
          path: "teams",
          name: "teams",
          component: TeamsView,
          meta: { titleKey: "pageTitle.teams" },
        },
        {
          path: "teams/:id",
          name: "team",
          component: TeamView,
          props: true,
          meta: { titleKey: "pageTitle.team" },
        },
        {
          path: "projects/:slug",
          component: ProjectView,
          props: true,
          children: [
            {
              path: "",
              name: "project",
              component: ProjectOverviewTab,
              meta: { titleKey: "pageTitle.project" },
            },
            {
              path: "knowledge",
              name: "project-knowledge",
              component: ProjectKnowledgeTab,
              meta: { titleKey: "pageTitle.project" },
            },

            { path: "wiki", redirect: { name: "project-knowledge" } },
            { path: "decisions", redirect: { name: "project-knowledge" } },
            {
              path: "usage",
              name: "project-usage",
              component: ProjectUsageTab,
              meta: { titleKey: "pageTitle.project" },
            },
            {
              path: "concurrency",
              name: "project-concurrency",
              component: ProjectConcurrencyTab,
              meta: { titleKey: "pageTitle.project" },
            },
            {
              path: "settings",
              component: ProjectSettingsView,
              children: [
                {
                  path: "",
                  name: "project-settings",
                  component: ProjectGeneralSettingsTab,
                  meta: { titleKey: "pageTitle.project" },
                },
                {
                  path: "tasks",
                  name: "project-settings-tasks",
                  component: ProjectTasksSettingsTab,
                  meta: { titleKey: "pageTitle.project" },
                },
                {
                  path: "models",
                  name: "project-settings-models",
                  component: ProjectModelsSettingsTab,
                  meta: { titleKey: "pageTitle.project" },
                },
                {
                  path: "templates",
                  name: "project-settings-templates",
                  component: ProjectTemplatesSettingsTab,
                  meta: { titleKey: "pageTitle.project" },
                },
                {
                  path: "plugins",
                  name: "project-settings-plugins",
                  component: ProjectPluginsSettingsTab,
                  meta: { titleKey: "pageTitle.project" },
                },
                {
                  path: "plugins/:pluginId",
                  name: "project-settings-plugin",
                  component: ProjectPluginSettingsTab,
                  meta: { titleKey: "pageTitle.project" },
                },
                {
                  path: "variables",
                  name: "project-settings-variables",
                  component: ProjectVariablesSettingsTab,
                  meta: { titleKey: "pageTitle.project" },
                },
                {
                  path: "access",
                  name: "project-settings-access",
                  component: ProjectAccessSettingsTab,
                  meta: { titleKey: "pageTitle.project" },
                },
                {
                  path: "danger",
                  name: "project-settings-danger",
                  component: ProjectDangerSettingsTab,
                  meta: { titleKey: "pageTitle.project" },
                },
              ],
            },
          ],
        },
        {
          path: "library",
          name: "library",
          component: LibraryView,
          meta: { titleKey: "pageTitle.library" },
        },
        {
          path: "settings",
          component: SettingsView,
          meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
          children: [
            {
              path: "",
              name: "settings",
              component: SettingsGeneralTab,
              meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
            },
            {
              path: "models",
              name: "settings-models",
              component: SettingsModelsTab,
              meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
            },
            {
              path: "plugins",
              name: "settings-plugins",
              component: PluginsPanel,
              meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
            },
            {
              path: "plugins/:pluginId",
              name: "settings-plugin",
              component: PluginDetailPanel,
              meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
            },
            {
              path: "auth",
              name: "settings-auth",
              component: SettingsAuthTab,
              meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
            },
            {
              path: "integrations",
              name: "settings-integrations",
              component: SettingsIntegrationsTab,
              meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
            },
            {
              path: "variables",
              name: "settings-variables",
              component: SettingsVariablesTab,
              meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
            },
            {
              path: "admin",
              name: "settings-admin",
              component: SettingsAdminTab,
              meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
            },
            {
              path: "logs",
              name: "settings-logs",
              component: SettingsLogsTab,
              meta: { titleKey: "pageTitle.orgSettings", adminOnly: true },
            },
          ],
        },
        {
          path: "settings/providers/create",
          name: "provider-create",
          component: ProviderEditorView,
          meta: { titleKey: "pageTitle.provider", adminOnly: true },
        },
        {
          path: "settings/providers/:slug/edit",
          name: "provider-edit",
          component: ProviderEditorView,
          meta: { titleKey: "pageTitle.provider", adminOnly: true },
        },
        {
          path: "account",
          name: "account",
          component: AccountView,
          meta: { titleKey: "pageTitle.account" },
        },

        {
          path: "agents/:username",
          name: "agent",
          component: AgentView,
          meta: { titleKey: "pageTitle.agent" },
        },

        {
          path: "users",
          name: "users",
          component: UsersView,
          meta: { titleKey: "pageTitle.users" },
        },
        {
          path: "users/:username",
          name: "user",
          component: UserView,
          meta: { titleKey: "pageTitle.user" },
        },

        {
          path: "agents/:username/tasks/:id",
          redirect: (to) => ({ name: "task", params: { id: to.params.id } }),
        },
        {
          path: "agents/:username/root-terminals",
          redirect: (to) => ({
            name: "agent",
            params: { username: to.params.username },
            query: { tab: "terminals" },
          }),
        },
        {
          path: "agents/:username/desktop",
          redirect: (to) => ({
            name: "agent",
            params: { username: to.params.username },
            query: { tab: "rdp" },
          }),
        },
        {
          path: "agents/:username/logs",
          redirect: (to) => ({
            name: "agent",
            params: { username: to.params.username },
            query: { tab: "logs" },
          }),
        },
        {
          path: "agents/:username/settings",
          redirect: (to) => ({
            name: "agent",
            params: { username: to.params.username },
            query: { tab: "settings" },
          }),
        },
      ],
    },
  ],
});

router.beforeEach(async (to) => {
  if (publicRoutes.has(String(to.name))) return true;

  const actor = await ensureSession();
  if (!actor) {
    return { name: "login", query: { redirect: to.fullPath } };
  }

  await loadLocalePreference();
  if (to.meta.adminOnly && !actor.is_admin) {
    return { name: "agency-tasks" };
  }

  // A username invite forces a password reset before anything else.
  if (actor.must_change_password && to.name !== "account") {
    return { name: "account", query: { must_change_password: "1" } };
  }

  const username = to.params.username;
  if (
    !to.path.startsWith("/agents/") ||
    typeof username !== "string" ||
    !username
  ) {
    currentAgentId.value = "";
    currentAgentUsername.value = "";
    return true;
  }

  const { agents, loadAgents } = useAgents();
  await loadAgents();
  const found = agents.value.find((agent) => agent.username === username);
  if (!found) return { name: "agents" };

  currentAgentId.value = found.id;
  currentAgentUsername.value = username;
  return true;
});

router.afterEach((to) => {
  const key = to.meta.titleKey;
  setPageTitle(typeof key === "string" ? t(key) : "");
});

export default router;
