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
import { useAgents } from "@/composables/useAgents";
import { setPageTitle } from "@/composables/usePageTitle";
import { ensureSession } from "@/lib/account";
import { currentAgentId, currentAgentUsername } from "@/lib/sdk";
import AccountView from "@/views/AccountView.vue";
import AgencyTasksView from "@/views/AgencyTasksView.vue";
import AgentView from "@/views/AgentView.vue";
import AgentsView from "@/views/AgentsView.vue";
import InboxView from "@/views/InboxView.vue";
import InviteView from "@/views/InviteView.vue";
import LoginView from "@/views/LoginView.vue";
import MagicLinkView from "@/views/MagicLinkView.vue";
import ProjectView from "@/views/ProjectView.vue";
import ProjectsView from "@/views/ProjectsView.vue";
import ResetPasswordView from "@/views/ResetPasswordView.vue";
import SettingsView from "@/views/SettingsView.vue";
import TaskView from "@/views/TaskView.vue";
import TeamView from "@/views/TeamView.vue";
import TeamsView from "@/views/TeamsView.vue";
import UserView from "@/views/UserView.vue";
import UsersView from "@/views/UsersView.vue";
import { createRouter, createWebHistory } from "vue-router";

const publicRoutes = new Set([
  "login",
  "magic-link",
  "reset-password",
  "invite",
]);

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: LoginView,
      meta: { title: "Sign in" },
    },
    {
      path: "/magic-link",
      name: "magic-link",
      component: MagicLinkView,
      meta: { title: "Signing in" },
    },
    {
      path: "/reset-password",
      name: "reset-password",
      component: ResetPasswordView,
      meta: { title: "Reset password" },
    },
    {
      path: "/invite",
      name: "invite",
      component: InviteView,
      meta: { title: "Accept invitation" },
    },
    {
      path: "/",
      component: AppShell,
      children: [
        { path: "", redirect: "/tasks" },
        {
          path: "agents",
          name: "agents",
          component: AgentsView,
          meta: { title: "Agents" },
        },

        {
          path: "tasks",
          name: "agency-tasks",
          component: AgencyTasksView,
          meta: { title: "Tasks" },
        },
        {
          path: "tasks/:id",
          name: "task",
          component: TaskView,
          props: true,
          meta: { title: "Task" },
        },
        {
          path: "inbox",
          name: "inbox",
          component: InboxView,
          meta: { title: "Inbox" },
        },
        {
          path: "projects",
          name: "projects",
          component: ProjectsView,
          meta: { title: "Projects" },
        },
        {
          path: "teams",
          name: "teams",
          component: TeamsView,
          meta: { title: "Teams" },
        },
        {
          path: "teams/:id",
          name: "team",
          component: TeamView,
          props: true,
          meta: { title: "Team" },
        },
        {
          path: "projects/:slug",
          name: "project",
          component: ProjectView,
          props: true,
          meta: { title: "Project" },
        },
        {
          path: "projects/:slug/settings",
          name: "project-settings",
          redirect: (to) => ({
            name: "project",
            params: { slug: to.params.slug },
            query: { ...to.query, tab: "settings" },
          }),
        },
        {
          path: "settings",
          name: "settings",
          component: SettingsView,
          meta: { title: "Organization settings", adminOnly: true },
        },
        {
          path: "account",
          name: "account",
          component: AccountView,
          meta: { title: "Account" },
        },

        {
          path: "agents/:username",
          name: "agent",
          component: AgentView,
          meta: { title: "Agent" },
        },

        {
          path: "users",
          name: "users",
          component: UsersView,
          meta: { title: "Users" },
        },
        {
          path: "users/:username",
          name: "user",
          component: UserView,
          meta: { title: "User" },
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
            query: { tab: "settings" },
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
  setPageTitle(typeof to.meta.title === "string" ? to.meta.title : "");
});

export default router;
