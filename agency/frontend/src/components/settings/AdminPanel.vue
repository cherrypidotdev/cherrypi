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
Tabs(v-model="tab")
  TabsList
    TabsTrigger(value="users") {{ $t("nav.users") }}
    TabsTrigger(value="teams") {{ $t("nav.teams") }}
    TabsTrigger(value="projects") {{ $t("nav.projects") }}
    TabsTrigger(value="agents") {{ $t("nav.agents") }}
    TabsTrigger(value="activity") {{ $t("activity.title") }}

  TabsContent(value="users")
    .flex.flex-col.gap-6
      Card
        CardHeader
          CardTitle {{ $t("admin.invite.title") }}
          CardDescription
            | {{ $t("admin.invite.description") }}
        CardContent
          form.flex.flex-wrap.items-end.gap-2(@submit.prevent="inviteUser")
            Field.w-56
              FieldLabel(for="admin-invite-username") {{ $t("common.username") }}
              Input#admin-invite-username(
                v-model="inviteUsername"
                :placeholder="$t('admin.invite.usernamePlaceholder')"
                autocomplete="off"
              )
            Field.w-40
              FieldLabel {{ $t("admin.role") }}
              Select(
                :model-value="inviteRole"
                @update:model-value="inviteRole = String($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="member") {{ $t("admin.roleMember") }}
                  SelectItem(value="admin") {{ $t("admin.roleAdmin") }}
            Button(
              type="submit"
              :disabled="inviting || !inviteUsername.trim()"
            )
              Spinner(v-if="inviting" data-icon="inline-start")
              SendIcon(v-else data-icon="inline-start")
              span {{ $t("admin.invite.submit") }}
          Alert.mt-4(v-if="invited")
            AlertTitle {{ $t("admin.invite.created") }}
            AlertDescription
              | {{ $t("admin.invite.sharePassword", { username: invited.username }) }}
            code.bg-muted.mt-2.block.rounded-md.px-2.py-1.font-mono.text-sm.break-all {{ invited.password }}
      Card
        CardHeader
          CardTitle {{ $t("nav.users") }}
          CardDescription {{ $t("admin.users.description") }}
        CardContent
          p.text-destructive.text-sm(v-if="usersError") {{ usersError }}
          Table
            TableHeader
              TableRow
                TableHead {{ $t("common.username") }}
                TableHead {{ $t("admin.role") }}
                TableHead {{ $t("common.status") }}
                TableHead {{ $t("admin.users.lastLogin") }}
                TableHead
            TableBody
              TableRowsSkeleton(
                v-if="usersLoading && !users.length"
                :widths="['w-28', 'w-24', 'w-24', 'w-28', 'w-16']"
              )
              TableRow(v-for="user in users" :key="user.id")
                TableCell.font-medium
                  .flex.items-center.gap-2
                    span {{ user.username }}
                    Badge(v-if="user.invitation" variant="secondary") {{ $t("admin.users.statusInvited") }}
                TableCell
                  Select(
                    :model-value="user.role"
                    @update:model-value="setRole(user, String($event))"
                  )
                    SelectTrigger
                      SelectValue
                    SelectContent
                      SelectItem(value="member") {{ $t("admin.roleMember") }}
                      SelectItem(value="admin") {{ $t("settings.tabs.admin") }}
                TableCell
                  Select(
                    :model-value="user.status"
                    @update:model-value="setStatus(user, String($event))"
                  )
                    SelectTrigger
                      SelectValue
                    SelectContent
                      SelectItem(value="invited") {{ $t("admin.users.statusInvitedLabel") }}
                      SelectItem(value="active") {{ $t("admin.users.statusActive") }}
                      SelectItem(value="disabled") {{ $t("common.disabled") }}
                TableCell.text-muted-foreground {{ formatDate(user.last_login_at) }}
                TableCell
                  .flex.justify-end
                    Button(
                      variant="destructive"
                      size="sm"
                      @click="removeUser(user)"
                    )
                      Trash2Icon(data-icon="inline-start")
                      span {{ $t("common.delete") }}
              TableRow(v-if="!users.length && !usersLoading")
                TableCell.text-muted-foreground(:colspan="5") {{ $t("admin.users.empty") }}
      AdminActivityCard(
        scope="user"
        :title="$t('admin.users.activityTitle')"
        :description="$t('admin.users.activityDescription')"
      )

  TabsContent(value="teams")
    .flex.flex-col.gap-6
      Card
        CardHeader
          CardTitle {{ $t("admin.teams.create") }}
          CardDescription {{ $t("admin.teams.createHint") }}
        CardContent
          form.flex.flex-wrap.items-end.gap-2(@submit.prevent="createTeam")
            Field.w-56
              FieldLabel(for="admin-team-name") {{ $t("common.name") }}
              Input#admin-team-name(
                v-model="teamName"
                :placeholder="$t('admin.teams.platform')"
              )
            Field.flex-1
              FieldLabel(for="admin-team-description") {{ $t("common.description") }}
              Input#admin-team-description(v-model="teamDescription")
            Button(type="submit" :disabled="creatingTeam || !teamName.trim()")
              Spinner(v-if="creatingTeam" data-icon="inline-start")
              PlusIcon(v-else data-icon="inline-start")
              span {{ $t("common.create") }}
      Card
        CardHeader
          CardTitle {{ $t("nav.teams") }}
          CardDescription {{ $t("admin.teams.description") }}
        CardContent
          p.text-destructive.text-sm(v-if="teamsError") {{ teamsError }}
          Table
            TableHeader
              TableRow
                TableHead {{ $t("admin.teams.columnTeam") }}
                TableHead {{ $t("admin.teams.columnMembers") }}
                TableHead {{ $t("admin.teams.columnCreated") }}
                TableHead
            TableBody
              TableRowsSkeleton(
                v-if="teamsLoading && !teams.length"
                :widths="['w-32', 'w-16', 'w-28', 'w-24']"
              )
              TableRow(v-for="team in teams" :key="team.id")
                TableCell
                  RouterLink.font-medium(
                    :to="{ name: 'team', params: { id: String(team.id) } }"
                  ) {{ team.name }}
                TableCell.text-muted-foreground {{ team.member_count }}
                TableCell.text-muted-foreground {{ formatDate(team.created_at) }}
                TableCell
                  .flex.justify-end
                    Button(
                      variant="destructive"
                      size="sm"
                      @click="removeTeam(team)"
                    )
                      Trash2Icon(data-icon="inline-start")
                      span {{ $t("common.delete") }}
              TableRow(v-if="!teams.length && !teamsLoading")
                TableCell.text-muted-foreground(:colspan="4") {{ $t("admin.teams.empty") }}
      AdminActivityCard(
        scope="team"
        :title="$t('admin.teams.activityTitle')"
        :description="$t('admin.teams.activityDescription')"
      )

  TabsContent(value="projects")
    .flex.flex-col.gap-6
      Card
        CardHeader
          CardTitle {{ $t("nav.projects") }}
          CardDescription {{ $t("admin.projects.description") }}
          CardAction
            Button(as-child variant="outline" size="sm")
              RouterLink(:to="{ name: 'projects' }") {{ $t("admin.projects.create") }}
        CardContent
          p.text-destructive.text-sm(v-if="projectsError") {{ projectsError }}
          Table
            TableHeader
              TableRow
                TableHead {{ $t("admin.projects.columnProject") }}
                TableHead {{ $t("settings.auth.slug") }}
                TableHead {{ $t("common.status") }}
                TableHead
            TableBody
              TableRowsSkeleton(
                v-if="projectsLoading && !projects.length"
                :widths="['w-40', 'w-24', 'w-20', 'w-32']"
              )
              TableRow(v-for="project in projects" :key="project.id")
                TableCell
                  RouterLink.font-medium(
                    :to="{ name: 'project', params: { slug: project.slug } }"
                  ) {{ project.name }}
                TableCell.text-muted-foreground.font-mono.text-xs {{ project.slug }}
                TableCell
                  Badge(
                    :variant="project.archivedAt ? 'secondary' : 'outline'"
                  ) {{ project.archivedAt ? "Archived" : "Active" }}
                TableCell
                  .flex.justify-end.gap-2
                    Button(
                      variant="outline"
                      size="sm"
                      @click="toggleArchive(project)"
                    )
                      ArchiveRestoreIcon(
                        v-if="project.archivedAt"
                        data-icon="inline-start"
                      )
                      ArchiveIcon(v-else data-icon="inline-start")
                      span {{ project.archivedAt ? "Unarchive" : "Archive" }}
                    Button(
                      variant="destructive"
                      size="sm"
                      @click="removeProject(project)"
                    )
                      Trash2Icon(data-icon="inline-start")
                      span {{ $t("common.delete") }}
              TableRow(v-if="!projects.length && !projectsLoading")
                TableCell.text-muted-foreground(:colspan="4") {{ $t("admin.projects.empty") }}
      AdminActivityCard(
        scope="project"
        :title="$t('admin.projects.activityTitle')"
        :description="$t('admin.projects.activityDescription')"
      )

  TabsContent(value="agents")
    .flex.flex-col.gap-6
      Card
        CardHeader
          CardTitle {{ $t("admin.agents.register") }}
          CardDescription
            | {{ $t("admin.agents.registerHint") }}
        CardContent
          form.flex.flex-wrap.items-end.gap-2(@submit.prevent="createAgent")
            Field.w-56
              FieldLabel(for="admin-agent-username") {{ $t("common.username") }}
              Input#admin-agent-username(
                v-model="agentUsername"
                :placeholder="$t('admin.agents.usernamePlaceholder')"
                autocomplete="off"
              )
            Field.w-56
              FieldLabel(for="admin-agent-name") {{ $t("admin.agents.displayName") }}
              Input#admin-agent-name(v-model="agentDisplayName")
            Button(
              type="submit"
              :disabled="creatingAgent || !agentUsername.trim()"
            )
              Spinner(v-if="creatingAgent" data-icon="inline-start")
              PlusIcon(v-else data-icon="inline-start")
              span {{ $t("admin.agents.registerSubmit") }}
          Alert.mt-4(v-if="agentToken")
            AlertTitle {{ $t("admin.agents.tokenTitle") }}
            AlertDescription
              | {{ $t("admin.agents.tokenHint") }}
            code.bg-muted.mt-2.block.rounded-md.px-2.py-1.font-mono.text-sm.break-all {{ agentToken }}
      Card
        CardHeader
          CardTitle {{ $t("nav.agents") }}
          CardDescription {{ $t("admin.agents.description") }}
        CardContent
          p.text-destructive.text-sm(v-if="agentsError") {{ agentsError }}
          Table
            TableHeader
              TableRow
                TableHead {{ $t("admin.agents.columnAgent") }}
                TableHead {{ $t("common.username") }}
                TableHead {{ $t("common.status") }}
                TableHead {{ $t("admin.agents.lastSeen") }}
                TableHead
            TableBody
              TableRowsSkeleton(
                v-if="agentsLoading && !agents.length"
                :widths="['w-36', 'w-24', 'w-20', 'w-24', 'w-16']"
              )
              TableRow(v-for="agent in agents" :key="agent.id")
                TableCell
                  RouterLink.font-medium(
                    :to="{ name: 'agent', params: { username: agent.username } }"
                  ) {{ agent.displayName || agent.username }}
                TableCell.text-muted-foreground.font-mono {{ agent.username }}
                TableCell
                  Badge(
                    variant="outline"
                    :class="agentStatusClass(agent.status)"
                  ) {{ agentStatusLabel(agent.status) }}
                TableCell.text-muted-foreground
                  | {{ formatTimestamp(agent.lastSeenAt) }}
                TableCell
                  .flex.justify-end
                    Button(
                      variant="destructive"
                      size="sm"
                      @click="removeAgent(agent)"
                    )
                      Trash2Icon(data-icon="inline-start")
                      span {{ $t("common.delete") }}
              TableRow(v-if="!agents.length && !agentsLoading")
                TableCell.text-muted-foreground(:colspan="5") {{ $t("admin.agents.empty") }}
      AdminActivityCard(
        scope="agent"
        :title="$t('admin.agents.activityTitle')"
        :description="$t('admin.agents.activityDescription')"
      )

  TabsContent(value="activity")
    EventsPanel
</template>

<script setup lang="ts">
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import AdminActivityCard from "@/components/settings/AdminActivityCard.vue";
import EventsPanel from "@/components/settings/EventsPanel.vue";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAgents } from "@/composables/useAgents";
import { withProgress } from "@/composables/useProgress";
import { useProjects } from "@/composables/useProjects";
import { accountApi, type AdminUser } from "@/lib/account";
import { agentStatusClass, agentStatusLabel } from "@/lib/agent";
import { formatTimestamp } from "@/lib/format";
import { agency } from "@/lib/sdk";
import { teamApi, type Team } from "@/lib/team";
import type { Agent, Project } from "@cherrypi/sdk";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  PlusIcon,
  SendIcon,
  Trash2Icon,
} from "@lucide/vue";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const tab = ref("users");

const users = ref<AdminUser[]>([]);
const usersLoading = ref(true);
const usersError = ref("");
const inviteUsername = ref("");
const inviteRole = ref("member");
const inviting = ref(false);
const invited = ref<{ username: string; password: string }>();

const teams = ref<Team[]>([]);
const teamsLoading = ref(true);
const teamsError = ref("");
const teamName = ref("");
const teamDescription = ref("");
const creatingTeam = ref(false);

const {
  projects,
  loading: projectsLoading,
  error: projectsError,
  refresh: refreshProjects,
  archiveProject,
  unarchiveProject,
  deleteProject,
} = useProjects();

const {
  agents,
  loading: agentsLoading,
  error: agentsError,
  refresh: refreshAgents,
} = useAgents();
const agentUsername = ref("");
const agentDisplayName = ref("");
const creatingAgent = ref(false);
const agentToken = ref("");

async function loadUsers() {
  usersLoading.value = true;
  try {
    users.value = (await accountApi.listUsers()).users;
    usersError.value = "";
  } catch (err) {
    usersError.value = err instanceof Error ? err.message : String(err);
  } finally {
    usersLoading.value = false;
  }
}

async function loadTeams() {
  teamsLoading.value = true;
  try {
    teams.value = (await teamApi.list()).teams;
    teamsError.value = "";
  } catch (err) {
    teamsError.value = err instanceof Error ? err.message : String(err);
  } finally {
    teamsLoading.value = false;
  }
}

async function inviteUser() {
  inviting.value = true;
  invited.value = undefined;
  try {
    const result = await withProgress(() =>
      accountApi.createInvitation({
        kind: "username",
        username: inviteUsername.value.trim(),
        role: inviteRole.value,
      }),
    );
    invited.value = {
      username: result.invitation.username,
      password: result.password ?? "",
    };
    inviteUsername.value = "";
    await loadUsers();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    inviting.value = false;
  }
}

async function setRole(user: AdminUser, role: string) {
  try {
    await accountApi.updateUser(user.id, { role });
    await loadUsers();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function setStatus(user: AdminUser, status: string) {
  try {
    await accountApi.updateUser(user.id, { status });
    await loadUsers();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function removeUser(user: AdminUser) {
  if (!window.confirm(`Delete user "${user.username}"?`)) return;
  try {
    await withProgress(() => accountApi.deleteUser(user.id));
    toast.success(t("admin.toasts.userDeleted"));
    await loadUsers();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function createTeam() {
  creatingTeam.value = true;
  try {
    await withProgress(() =>
      teamApi.create(teamName.value.trim(), teamDescription.value.trim()),
    );
    teamName.value = "";
    teamDescription.value = "";
    toast.success(t("admin.toasts.teamCreated"));
    await loadTeams();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    creatingTeam.value = false;
  }
}

async function removeTeam(team: Team) {
  if (!window.confirm(`Delete team "${team.name}"?`)) return;
  try {
    await withProgress(() => teamApi.remove(team.id));
    toast.success(t("admin.toasts.teamDeleted"));
    await loadTeams();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function toggleArchive(project: Project) {
  try {
    if (project.archivedAt) {
      await withProgress(() => unarchiveProject(project.id));
      toast.success(t("admin.toasts.projectUnarchived"));
    } else {
      await withProgress(() => archiveProject(project.id));
      toast.success(t("admin.toasts.projectArchived"));
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function removeProject(project: Project) {
  if (
    !window.confirm(
      `Delete project "${project.name}" and everything in it? This cannot be undone.`,
    )
  )
    return;
  try {
    await withProgress(() => deleteProject(project.id));
    toast.success(t("admin.toasts.projectDeleted"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function createAgent() {
  creatingAgent.value = true;
  try {
    const result = await withProgress(() =>
      agency.createAgent({
        username: agentUsername.value.trim(),
        displayName: agentDisplayName.value.trim(),
      }),
    );
    agentToken.value = result.token;
    agentUsername.value = "";
    agentDisplayName.value = "";
    await refreshAgents();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    creatingAgent.value = false;
  }
}

async function removeAgent(agent: Agent) {
  if (
    !window.confirm(`Delete agent "${agent.displayName || agent.username}"?`)
  ) {
    return;
  }
  try {
    await withProgress(() => agency.deleteAgent(agent.id));
    toast.success(t("admin.toasts.agentDeleted"));
    await refreshAgents();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

function formatDate(value?: string): string {
  if (!value) return "Never";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? "—" : parsed.toLocaleString();
}

onMounted(async () => {
  await Promise.all([
    loadUsers(),
    loadTeams(),
    refreshProjects(),
    refreshAgents(),
  ]);
});
</script>

<i18n lang="yaml">
en:
  admin:
    agents:
      columnAgent: "Agent"
      register: "Register an agent"
      registerHint: "A personal access token and a legacy tunnel credential are shown once."
      displayName: "Display name"
      registerSubmit: "Register"
      tokenTitle: "Personal access token generated"
      tokenHint: "Copy the token now — it is shown only once. Set it as CHERRYPI_AGENT_TOKEN on the agent."
      description: "Presence is live while an agent's tunnel is connected."
      lastSeen: "Last seen"
      empty: "No agents."
      usernamePlaceholder: "builder-1"
      activityTitle: "Agent activity"
      activityDescription: "Ownership, credential, and settings changes from the agent log."
    users:
      statusInvited: "invited"
      statusInvitedLabel: "Invited"
      description: "Accounts, roles, and status."
      lastLogin: "Last login"
      statusActive: "Active"
      empty: "No users."
      activityTitle: "User activity"
      activityDescription: "Authentication and account mutations from the user log."
    toasts:
      userDeleted: "User deleted"
      agentDeleted: "Agent deleted"
      projectDeleted: "Project deleted"
      projectArchived: "Project archived"
      projectUnarchived: "Project unarchived"
      teamDeleted: "Team deleted"
      teamCreated: "Team created"
    roleMember: "Member"
    projects:
      columnProject: "Project"
      description: "Every project in the agency."
      create: "New project"
      empty: "No projects."
      activityTitle: "Project activity"
      activityDescription: "Access, prompt, and archive changes from the project log."
    teams:
      columnCreated: "Created"
      columnMembers: "Members"
      platform: "Platform"
      create: "Create a team"
      createHint: "Teams carry access to projects and own tasks."
      description: "Every team in the agency."
      columnTeam: "Team"
      empty: "No teams."
      activityTitle: "Team activity"
      activityDescription: "Membership and settings changes from the team log."
    invite:
      title: "Invite a user"
      description: "Reserve a username and hand out a one-time password. The invitee must change it on first sign-in."
      submit: "Invite"
      created: "Invitation created"
      usernamePlaceholder: "jane"
      sharePassword: "Share this one-time password with {'@'}{username}."
    role: "Role"
    roleAdmin: "Administrator"
ru:
  admin:
    agents:
      columnAgent: "Агент"
      register: "Зарегистрировать агента"
      registerHint: "Персональный токен доступа и устаревшие учётные данные туннеля показываются один раз."
      displayName: "Отображаемое имя"
      registerSubmit: "Зарегистрировать"
      tokenTitle: "Персональный токен доступа создан"
      tokenHint: "Скопируйте токен сейчас — он показывается только один раз. Задайте его как CHERRYPI_AGENT_TOKEN на агенте."
      description: "Присутствие активно, пока подключён туннель агента."
      lastSeen: "Последняя активность"
      empty: "Агентов нет."
      usernamePlaceholder: "builder-1"
      activityTitle: "Активность агентов"
      activityDescription: "Изменения владельцев, учётных данных и настроек из журнала агентов."
    users:
      statusInvited: "приглашён"
      statusInvitedLabel: "Приглашён"
      description: "Аккаунты, роли и статус."
      lastLogin: "Последний вход"
      statusActive: "Активен"
      empty: "Пользователей нет."
      activityTitle: "Активность пользователей"
      activityDescription: "Аутентификация и изменения аккаунтов из журнала пользователей."
    toasts:
      userDeleted: "Пользователь удалён"
      agentDeleted: "Агент удалён"
      projectDeleted: "Проект удалён"
      projectArchived: "Проект архивирован"
      projectUnarchived: "Проект разархивирован"
      teamDeleted: "Команда удалена"
      teamCreated: "Команда создана"
    roleMember: "Участник"
    projects:
      columnProject: "Проект"
      description: "Все проекты агентства."
      create: "Новый проект"
      empty: "Проектов нет."
      activityTitle: "Активность проектов"
      activityDescription: "Изменения доступа, промпта и архива из журнала проектов."
    teams:
      columnCreated: "Создана"
      columnMembers: "Участники"
      platform: "Платформа"
      create: "Создать команду"
      createHint: "Команды дают доступ к проектам и владеют задачами."
      description: "Все команды агентства."
      columnTeam: "Команда"
      empty: "Команд нет."
      activityTitle: "Активность команд"
      activityDescription: "Изменения состава и настроек из журнала команд."
    invite:
      title: "Пригласить пользователя"
      description: "Зарезервируйте имя пользователя и выдайте одноразовый пароль. Приглашённый должен сменить его при первом входе."
      submit: "Пригласить"
      created: "Приглашение создано"
      usernamePlaceholder: "jane"
      sharePassword: "Передайте этот одноразовый пароль пользователю {'@'}{username}."
    role: "Роль"
    roleAdmin: "Администратор"
</i18n>
