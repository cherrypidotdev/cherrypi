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
  header.border-b.px-6.py-4
    .mx-auto.flex.w-full.max-w-6xl.items-start.justify-between.gap-4
      .flex.min-w-0.flex-col
        .flex.items-center.gap-2
          RouterLink.text-muted-foreground(
            class="hover:text-foreground"
            :to="{ name: 'teams' }"
          ) Teams
          span.text-muted-foreground /
          h1.truncate.text-lg.font-semibold {{ team?.name || "Team" }}
        p.text-muted-foreground.truncate.text-sm(v-if="team?.description")
          | {{ team.description }}
      .flex.shrink-0.items-center.gap-2
        Badge(v-if="team?.role" variant="secondary") {{ team.role }}
        Badge(v-else-if="isAdmin" variant="secondary") admin
  .flex-1.overflow-auto
    .mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
      p.text-destructive.text-sm(v-if="error") {{ error }}
      Tabs(v-model="tab")
        TabsList
          TabsTrigger(value="profile") Profile
          TabsTrigger(value="members") Members
          TabsTrigger(value="projects") Projects
          TabsTrigger(value="settings") Settings

        TabsContent(value="profile")
          Card
            CardHeader
              CardTitle Profile
              CardDescription The team's public identity.
            CardContent
              .flex.flex-col.gap-6
                .flex.items-center.gap-4
                  span.size-14.shrink-0(:innerHTML="teamLogo")
                  .flex.min-w-0.flex-col
                    h2.truncate.text-xl.font-semibold {{ team?.name || "Team" }}
                    p.text-muted-foreground.text-sm(v-if="team?.description")
                      | {{ team.description }}
                    p.text-muted-foreground.text-sm(v-else)
                      | No description yet.
                dl.grid.grid-cols-2.gap-4.text-sm(class="sm:grid-cols-4")
                  div
                    dt.text-muted-foreground Members
                    dd.font-medium {{ team?.member_count ?? 0 }}
                  div
                    dt.text-muted-foreground Your role
                    dd.font-medium.capitalize {{ team?.role || (isAdmin ? "admin" : "—") }}
                  div
                    dt.text-muted-foreground Projects
                    dd.font-medium {{ projects.length }}
                  div
                    dt.text-muted-foreground Created
                    dd.font-medium {{ formatDate(team?.created_at ?? "") }}

          Card
            CardHeader
              CardTitle Tasks
              CardDescription Tasks owned by this team.
            CardContent
              TasksList(
                empty-text="No tasks are owned by this team yet."
                :team-id="id"
              )

        TabsContent.flex.flex-col.gap-6(value="members")
          Card
            CardHeader
              CardTitle Members
              CardDescription Users and agents in this team.
            CardContent
              Table
                TableHeader
                  TableRow
                    TableHead Username
                    TableHead Kind
                    TableHead Role
                    TableHead Joined
                    TableHead
                TableBody
                  TableRowsSkeleton(
                    v-if="loading && !members.length"
                    :widths="['w-32', 'w-16', 'w-16', 'w-28', 'w-16']"
                  )
                  TableRow(v-for="member in members" :key="member.actor_id")
                    TableCell.font-medium {{ member.username }}
                    TableCell.text-muted-foreground {{ member.kind }}
                    TableCell
                      Badge(
                        :variant="member.role === 'owner' ? 'default' : 'secondary'"
                      )
                        | {{ member.role }}
                    TableCell.text-muted-foreground {{ formatDate(member.joined_at) }}
                    TableCell
                      .flex.justify-end
                        Button(
                          v-if="member.actor_id === ownActorId"
                          variant="outline"
                          size="sm"
                          @click="removeMember(member)"
                          :disabled="busy === member.actor_id"
                        ) Leave
                        Button(
                          v-else-if="canManage"
                          variant="outline"
                          size="sm"
                          @click="removeMember(member)"
                          :disabled="busy === member.actor_id"
                        ) Remove

          Card(v-if="canInvite")
            CardHeader
              CardTitle Invitations
              CardDescription
                | Users accept their own invitation; an agent's owner approves it.
            CardContent
              form.flex.flex-col.gap-4(@submit.prevent="invite")
                FieldGroup
                  .grid.grid-cols-2.gap-3
                    Field
                      FieldLabel(for="invite-username") Username
                      Input#invite-username(
                        v-model="inviteUsername"
                        placeholder="jane"
                        autocomplete="off"
                      )
                    Field
                      FieldLabel Role
                      Select(
                        :model-value="inviteRole"
                        @update:model-value="inviteRole = String($event)"
                      )
                        SelectTrigger
                          SelectValue
                        SelectContent
                          SelectItem(value="member") Member
                          SelectItem(value="owner") Owner
                  Field
                    FieldLabel(for="invite-note") Note
                    Input#invite-note(
                      v-model="inviteNote"
                      placeholder="Optional message"
                      autocomplete="off"
                    )
                p.text-destructive.text-sm(v-if="inviteError") {{ inviteError }}
                .flex.justify-end
                  Button(
                    type="submit"
                    :disabled="inviting || !inviteUsername.trim()"
                  )
                    Spinner(v-if="inviting" data-icon="inline-start")
                    SendIcon(v-else data-icon="inline-start")
                    span Send invitation
              .mt-6
                Empty(v-if="!loading && !invitations.length")
                  EmptyTitle No invitations
                  EmptyDescription Invite an actor above.
                Table(v-else)
                  TableHeader
                    TableRow
                      TableHead Username
                      TableHead Role
                      TableHead Status
                      TableHead Expires
                      TableHead
                  TableBody
                    TableRowsSkeleton(
                      v-if="loading && !invitations.length"
                      :widths="['w-32', 'w-16', 'w-20', 'w-28', 'w-16']"
                    )
                    TableRow(
                      v-for="invitation in invitations"
                      :key="invitation.id"
                    )
                      TableCell.font-medium {{ invitation.username }}
                      TableCell.text-muted-foreground {{ invitation.role }}
                      TableCell
                        Badge(
                          :variant="invitation.status === 'pending' ? 'secondary' : 'outline'"
                        )
                          | {{ invitation.status }}
                      TableCell.text-muted-foreground {{ formatDate(invitation.expires_at) }}
                      TableCell
                        .flex.justify-end.gap-2
                          Button(
                            v-if="invitation.status === 'pending'"
                            variant="outline"
                            size="sm"
                            @click="revoke(invitation)"
                            :disabled="busy === 'i' + invitation.id"
                          ) Revoke

        TabsContent.flex.flex-col.gap-6(value="projects")
          Card
            CardHeader
              CardTitle Team prompt
              CardDescription
                | Common instructions for every task this team owns. They are
                | added after the project prompt and before the agent's prompt.
            CardContent
              PromptEditor(
                placeholder="How should agents work for this team?"
                :scope="PromptScope.TEAM"
                :team-id="String(id)"
              )

          Card(v-if="projectInvitations.length")
            CardHeader
              CardTitle Project invitations
              CardDescription
                | Projects this team has been invited to. A team owner approves
                | the invitation.
            CardContent
              .flex.flex-col.gap-2
                .flex.items-center.justify-between.gap-3.rounded-lg.border.p-3(
                  v-for="invitation in projectInvitations"
                  :key="invitation.id"
                )
                  .flex.min-w-0.flex-col
                    span.text-sm.font-medium
                      | {{ invitation.project_name }}
                    span.text-muted-foreground.truncate.text-xs
                      | Invited by {{ invitation.invited_by_username || "an administrator" }}
                      | as {{ PROJECT_ROLE_LABEL[invitation.role] }}
                  .flex.shrink-0.gap-2
                    Button(
                      size="sm"
                      @click="respondProject(invitation, true)"
                      :disabled="respondingProject === invitation.id"
                    ) Approve
                    Button(
                      size="sm"
                      variant="outline"
                      @click="respondProject(invitation, false)"
                      :disabled="respondingProject === invitation.id"
                    ) Decline

          Card
            CardHeader
              CardTitle Projects
              CardDescription
                | Projects this team has been granted access to, each with a
                | project-specific prompt.
            CardContent
              Empty(v-if="!projects.length")
                EmptyTitle No projects
                EmptyDescription Attach this team to a project from the project's Access settings.
              .flex.flex-col.gap-4(v-else)
                Collapsible.rounded-lg.border(
                  v-for="project in projects"
                  :key="project.id"
                  class="group/collapsible"
                )
                  .flex.items-center.gap-3.p-4
                    CollapsibleTrigger.flex.flex-1.cursor-pointer.items-center.gap-2.text-left
                      ChevronRightIcon.text-muted-foreground.size-4.shrink-0.transition-transform(
                        class="group-data-[state=open]/collapsible:rotate-90"
                      )
                      span.size-5.shrink-0(
                        :innerHTML="projectLogo(project.id)"
                      )
                      span.text-sm.font-medium {{ project.name }}
                    Badge(variant="secondary") {{ roleLabel(project.role) }}
                    RouterLink.text-muted-foreground(
                      class="hover:text-foreground"
                      aria-label="Open project"
                      :to="{ name: 'project', params: { slug: project.slug } }"
                    )
                      ArrowUpRightIcon.size-4
                  CollapsibleContent
                    .flex.flex-col.gap-3.px-4.pb-4
                      p.text-muted-foreground.text-xs
                        | Applies to tasks this team owns in {{ project.name }}.
                      PromptEditor(
                        :placeholder="`How should agents work in ${project.name}?`"
                        :scope="PromptScope.TEAM_PROJECT"
                        :team-id="String(id)"
                        :project-id="project.id"
                      )

        TabsContent(value="tasks")
          Card
            CardHeader
              CardTitle Tasks
              CardDescription Tasks owned by this team.
            CardContent
              TasksList(
                empty-text="No tasks are owned by this team yet."
                :team-id="id"
              )

        TabsContent(value="settings")
          .flex.flex-col.gap-6
            Card
              CardHeader
                CardTitle Settings
                CardDescription Team name and description.
              CardContent
                form.flex.flex-col.gap-4(@submit.prevent="saveSettings")
                  FieldGroup
                    Field
                      FieldLabel(for="team-settings-name") Name
                      Input#team-settings-name(
                        v-model="settingsName"
                        maxlength="100"
                      )
                    Field
                      FieldLabel(for="team-settings-description") Description
                      Input#team-settings-description(
                        v-model="settingsDescription"
                      )
                  p.text-destructive.text-sm(v-if="settingsError") {{ settingsError }}
                  .flex.justify-end
                    Button(
                      type="submit"
                      :disabled="savingSettings || !settingsName.trim()"
                    )
                      Spinner(v-if="savingSettings" data-icon="inline-start")
                      CheckIcon(v-else data-icon="inline-start")
                      span Save

            Card
              CardHeader
                CardTitle Team log
                CardDescription Who changed what, newest first.
              CardContent
                Empty(v-if="!loading && !logs.length")
                  EmptyTitle No activity yet
                .flex.flex-col.divide-y(v-else)
                  .flex.items-start.justify-between.gap-3.py-2(
                    v-for="entry in logs"
                    :key="entry.id"
                  )
                    .flex.min-w-0.flex-col
                      span.text-sm {{ entry.message }}
                      span.text-muted-foreground.text-xs
                        | {{ entry.actor_username || "system" }} · {{ formatDate(entry.at) }}
                    Badge(v-if="entry.kind" variant="outline") {{ entry.kind }}

            Card.border-destructive(v-if="canDelete")
              CardHeader
                CardTitle Delete team
                CardDescription
                  | A team with members cannot be deleted. Remove every member first.
              CardContent
                .flex.items-center.justify-between.gap-3
                  p.text-muted-foreground.text-sm(v-if="team?.member_count")
                    | This team has {{ team.member_count }} member(s).
                  .flex.justify-end
                    Button(
                      variant="destructive"
                      @click="removeTeam"
                      :disabled="deleting || (team?.member_count ?? 0) > 0"
                    )
                      Spinner(v-if="deleting" data-icon="inline-start")
                      Trash2Icon(v-else data-icon="inline-start")
                      span Delete team
</template>

<script setup lang="ts">
import PromptEditor from "@/components/PromptEditor.vue";
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import TasksList from "@/components/tasks/TasksList.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
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
import { setPageTitle } from "@/composables/usePageTitle";
import { withProgress } from "@/composables/useProgress";
import { currentAccount } from "@/lib/account";
import {
  PROJECT_ROLE_LABEL,
  projectAccessApi,
  type ProjectTeamInvitation,
} from "@/lib/projectAccess";
import {
  teamApi,
  type Team,
  type TeamInvitation,
  type TeamLog,
  type TeamMember,
  type TeamProject,
} from "@/lib/team";
import { PromptScope } from "@cherrypi/sdk";
import {
  ArrowUpRightIcon,
  CheckIcon,
  ChevronRightIcon,
  SendIcon,
  Trash2Icon,
} from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref, watchEffect } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const route = useRoute();
const router = useRouter();
const id = Number(route.params.id);

const tab = ref("profile");
const team = ref<Team>();
const members = ref<TeamMember[]>([]);
const invitations = ref<TeamInvitation[]>([]);
const logs = ref<TeamLog[]>([]);
const projects = ref<TeamProject[]>([]);
const projectInvitations = ref<ProjectTeamInvitation[]>([]);
const loading = ref(true);
const error = ref("");

const inviteUsername = ref("");
const inviteRole = ref("member");
const inviteNote = ref("");
const inviting = ref(false);
const inviteError = ref("");

const settingsName = ref("");
const settingsDescription = ref("");
const savingSettings = ref(false);
const settingsError = ref("");
const deleting = ref(false);
const busy = ref("");
const respondingProject = ref(0);

const ownActorId = computed(() =>
  currentAccount.value ? String(currentAccount.value.id) : "",
);
const isAdmin = computed(() => currentAccount.value?.is_admin ?? false);
const canManage = computed(() => isAdmin.value || team.value?.role === "owner");
const canInvite = computed(() => isAdmin.value || !!team.value?.role);
const canDelete = computed(() => isAdmin.value || team.value?.role === "owner");

const teamLogo = computed(() => toSvg(String(id || "team"), 56));

watchEffect(() => setPageTitle(team.value?.name || "Team"));

function projectLogo(projectId: string): string {
  return toSvg(projectId, 24);
}

function roleLabel(role: string): string {
  return PROJECT_ROLE_LABEL[role as keyof typeof PROJECT_ROLE_LABEL] ?? role;
}

function formatDate(value: string): string {
  if (!value) return "—";
  return new Date(value).toLocaleString();
}

async function load() {
  loading.value = true;
  try {
    const [
      teamResult,
      memberResult,
      invitationResult,
      logResult,
      projectResult,
      projectInvitationResult,
    ] = await Promise.all([
      teamApi.get(id),
      teamApi.members(id),
      teamApi.invitations(id),
      teamApi.logs(id),
      teamApi.projects(id),
      projectAccessApi.pendingInvitations(),
    ]);
    team.value = teamResult.team;
    members.value = memberResult.members;
    invitations.value = invitationResult.invitations;
    logs.value = logResult.logs;
    projects.value = projectResult.projects;
    projectInvitations.value = projectInvitationResult.invitations.filter(
      (invitation) => invitation.team_id === id,
    );
    settingsName.value = teamResult.team.name;
    settingsDescription.value = teamResult.team.description;
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function invite() {
  inviting.value = true;
  inviteError.value = "";
  try {
    await withProgress(() =>
      teamApi.invite(
        id,
        inviteUsername.value.trim(),
        inviteRole.value === "owner" ? "owner" : "member",
        inviteNote.value.trim(),
      ),
    );
    toast.success("Invitation sent");
    inviteUsername.value = "";
    inviteNote.value = "";
    await load();
  } catch (err) {
    inviteError.value = err instanceof Error ? err.message : String(err);
  } finally {
    inviting.value = false;
  }
}

async function revoke(invitation: TeamInvitation) {
  busy.value = "i" + invitation.id;
  try {
    await withProgress(() => teamApi.revokeInvitation(id, invitation.id));
    toast.success("Invitation revoked");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function respondProject(
  invitation: ProjectTeamInvitation,
  approve: boolean,
) {
  respondingProject.value = invitation.id;
  try {
    await withProgress(() => projectAccessApi.respond(invitation.id, approve));
    toast.success(approve ? "Project access granted" : "Invitation declined");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    respondingProject.value = 0;
  }
}

async function removeMember(member: TeamMember) {
  const self = member.actor_id === ownActorId.value;
  if (
    !window.confirm(
      self ? "Leave this team?" : `Remove ${member.username} from the team?`,
    )
  ) {
    return;
  }
  busy.value = member.actor_id;
  try {
    await withProgress(() => teamApi.removeMember(id, member.actor_id));
    toast.success(self ? "Left team" : "Member removed");
    if (self) {
      await router.push({ name: "teams" });
      return;
    }
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function saveSettings() {
  savingSettings.value = true;
  settingsError.value = "";
  try {
    const result = await withProgress(() =>
      teamApi.update(
        id,
        settingsName.value.trim(),
        settingsDescription.value.trim(),
      ),
    );
    team.value = result.team;
    toast.success("Team updated");
  } catch (err) {
    settingsError.value = err instanceof Error ? err.message : String(err);
  } finally {
    savingSettings.value = false;
  }
}

async function removeTeam() {
  if (!window.confirm("Delete this team? This cannot be undone.")) return;
  deleting.value = true;
  try {
    await withProgress(() => teamApi.remove(id));
    toast.success("Team deleted");
    await router.push({ name: "teams" });
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    deleting.value = false;
  }
}

onMounted(load);
</script>
