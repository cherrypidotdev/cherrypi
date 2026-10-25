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
    TabsTrigger(value="users") Users
    TabsTrigger(value="teams") Teams
    TabsTrigger(value="projects") Projects
    TabsTrigger(value="agents") Agents
    TabsTrigger(value="activity") Activity

  TabsContent(value="users")
    .flex.flex-col.gap-6
      Card
        CardHeader
          CardTitle Invite a user
          CardDescription
            | Reserve a username and hand out a one-time password. The invitee
            | must change it on first sign-in.
        CardContent
          form.flex.flex-wrap.items-end.gap-2(@submit.prevent="inviteUser")
            Field.w-56
              FieldLabel(for="admin-invite-username") Username
              Input#admin-invite-username(
                v-model="inviteUsername"
                placeholder="jane"
                autocomplete="off"
              )
            Field.w-40
              FieldLabel Role
              Select(
                :model-value="inviteRole"
                @update:model-value="inviteRole = String($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="member") Member
                  SelectItem(value="admin") Administrator
            Button(
              type="submit"
              :disabled="inviting || !inviteUsername.trim()"
            )
              Spinner(v-if="inviting" data-icon="inline-start")
              SendIcon(v-else data-icon="inline-start")
              span Invite
          Alert.mt-4(v-if="invited")
            AlertTitle Invitation created
            AlertDescription
              | Share this one-time password with @{{ invited.username }}.
            code.bg-muted.mt-2.block.rounded-md.px-2.py-1.font-mono.text-sm.break-all {{ invited.password }}
      Card
        CardHeader
          CardTitle Users
          CardDescription Accounts, roles, and status.
        CardContent
          p.text-destructive.text-sm(v-if="usersError") {{ usersError }}
          Table
            TableHeader
              TableRow
                TableHead Username
                TableHead Role
                TableHead Status
                TableHead Last login
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
                    Badge(v-if="user.invitation" variant="secondary") invited
                TableCell
                  Select(
                    :model-value="user.role"
                    @update:model-value="setRole(user, String($event))"
                  )
                    SelectTrigger
                      SelectValue
                    SelectContent
                      SelectItem(value="member") Member
                      SelectItem(value="admin") Admin
                TableCell
                  Select(
                    :model-value="user.status"
                    @update:model-value="setStatus(user, String($event))"
                  )
                    SelectTrigger
                      SelectValue
                    SelectContent
                      SelectItem(value="invited") Invited
                      SelectItem(value="active") Active
                      SelectItem(value="disabled") Disabled
                TableCell.text-muted-foreground {{ formatDate(user.last_login_at) }}
                TableCell
                  .flex.justify-end
                    Button(
                      variant="destructive"
                      size="sm"
                      @click="removeUser(user)"
                    )
                      Trash2Icon(data-icon="inline-start")
                      span Delete
              TableRow(v-if="!users.length && !usersLoading")
                TableCell.text-muted-foreground(:colspan="5") No users.
      AdminActivityCard(
        scope="user"
        title="User activity"
        description="Authentication and account mutations from the user log."
      )

  TabsContent(value="teams")
    .flex.flex-col.gap-6
      Card
        CardHeader
          CardTitle Create a team
          CardDescription Teams carry access to projects and own tasks.
        CardContent
          form.flex.flex-wrap.items-end.gap-2(@submit.prevent="createTeam")
            Field.w-56
              FieldLabel(for="admin-team-name") Name
              Input#admin-team-name(v-model="teamName" placeholder="Platform")
            Field.flex-1
              FieldLabel(for="admin-team-description") Description
              Input#admin-team-description(v-model="teamDescription")
            Button(type="submit" :disabled="creatingTeam || !teamName.trim()")
              Spinner(v-if="creatingTeam" data-icon="inline-start")
              PlusIcon(v-else data-icon="inline-start")
              span Create
      Card
        CardHeader
          CardTitle Teams
          CardDescription Every team in the agency.
        CardContent
          p.text-destructive.text-sm(v-if="teamsError") {{ teamsError }}
          Table
            TableHeader
              TableRow
                TableHead Team
                TableHead Members
                TableHead Created
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
                      span Delete
              TableRow(v-if="!teams.length && !teamsLoading")
                TableCell.text-muted-foreground(:colspan="4") No teams.
      AdminActivityCard(
        scope="team"
        title="Team activity"
        description="Membership and settings changes from the team log."
      )

  TabsContent(value="projects")
    .flex.flex-col.gap-6
      Card
        CardHeader
          CardTitle Projects
          CardDescription Every project in the agency.
          CardAction
            Button(as-child variant="outline" size="sm")
              RouterLink(:to="{ name: 'projects' }") New project
        CardContent
          p.text-destructive.text-sm(v-if="projectsError") {{ projectsError }}
          Table
            TableHeader
              TableRow
                TableHead Project
                TableHead Slug
                TableHead Status
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
                      span Delete
              TableRow(v-if="!projects.length && !projectsLoading")
                TableCell.text-muted-foreground(:colspan="4") No projects.
      AdminActivityCard(
        scope="project"
        title="Project activity"
        description="Access, prompt, and archive changes from the project log."
      )

  TabsContent(value="agents")
    .flex.flex-col.gap-6
      Card
        CardHeader
          CardTitle Register an agent
          CardDescription A tunnel credential is generated and shown once.
        CardContent
          form.flex.flex-wrap.items-end.gap-2(@submit.prevent="createAgent")
            Field.w-56
              FieldLabel(for="admin-agent-username") Username
              Input#admin-agent-username(
                v-model="agentUsername"
                placeholder="builder-1"
                autocomplete="off"
              )
            Field.w-56
              FieldLabel(for="admin-agent-name") Display name
              Input#admin-agent-name(v-model="agentDisplayName")
            Button(
              type="submit"
              :disabled="creatingAgent || !agentUsername.trim()"
            )
              Spinner(v-if="creatingAgent" data-icon="inline-start")
              PlusIcon(v-else data-icon="inline-start")
              span Register
          Alert.mt-4(v-if="agentCredential")
            AlertTitle Credential generated
            AlertDescription Copy the password now — it is shown only once.
            code.bg-muted.mt-2.block.rounded-md.px-2.py-1.font-mono.text-sm.break-all {{ agentCredential }}
      Card
        CardHeader
          CardTitle Agents
          CardDescription Presence is live while an agent's tunnel is connected.
        CardContent
          p.text-destructive.text-sm(v-if="agentsError") {{ agentsError }}
          Table
            TableHeader
              TableRow
                TableHead Agent
                TableHead Username
                TableHead Status
                TableHead Last seen
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
                      span Delete
              TableRow(v-if="!agents.length && !agentsLoading")
                TableCell.text-muted-foreground(:colspan="5") No agents.
      AdminActivityCard(
        scope="agent"
        title="Agent activity"
        description="Ownership, credential, and settings changes from the agent log."
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
import { RouterLink } from "vue-router";
import { toast } from "vue3-toastify";

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
const agentCredential = ref("");

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
    toast.success("User deleted");
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
    toast.success("Team created");
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
    toast.success("Team deleted");
    await loadTeams();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function toggleArchive(project: Project) {
  try {
    if (project.archivedAt) {
      await withProgress(() => unarchiveProject(project.id));
      toast.success("Project unarchived");
    } else {
      await withProgress(() => archiveProject(project.id));
      toast.success("Project archived");
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
    toast.success("Project deleted");
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
    agentCredential.value = `${result.credentialUsername}:${result.credentialPassword}`;
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
    toast.success("Agent deleted");
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
