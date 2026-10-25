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
.mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
  Card(v-if="pending.length")
    CardHeader
      CardTitle Pending invitations
      CardDescription
        | You have been invited to these teams. Accepting adds you as a member.
    CardContent
      .flex.flex-col.gap-2
        .flex.items-center.justify-between.gap-3.rounded-lg.border.p-3(
          v-for="invitation in pending"
          :key="invitation.id"
        )
          .flex.min-w-0.flex-col
            span.text-sm.font-medium {{ invitation.team_name }}
            span.text-muted-foreground.truncate.text-xs
              | Invited by {{ invitation.invited_by_username || "an administrator" }}
              | as {{ invitation.role }}
          .flex.shrink-0.gap-2
            Button(
              size="sm"
              @click="respond(invitation.id, true)"
              :disabled="responding === invitation.id"
            ) Approve
            Button(
              size="sm"
              variant="outline"
              @click="respond(invitation.id, false)"
              :disabled="responding === invitation.id"
            ) Decline

  Card(v-if="projectInvitations.length")
    CardHeader
      CardTitle Project invitations
      CardDescription
        | Teams you own have been invited to these projects. Approving attaches
        | the team with the invited role.
    CardContent
      .flex.flex-col.gap-2
        .flex.items-center.justify-between.gap-3.rounded-lg.border.p-3(
          v-for="invitation in projectInvitations"
          :key="invitation.id"
        )
          .flex.min-w-0.flex-col
            span.text-sm.font-medium
              | {{ invitation.team_name }} → {{ invitation.project_name }}
            span.text-muted-foreground.truncate.text-xs
              | Invited by {{ invitation.invited_by_username || "an administrator" }}
              | as {{ PROJECT_ROLE_LABEL[invitation.role] }}
          .flex.shrink-0.gap-2
            Button(
              size="sm"
              @click="respondProject(invitation.id, true)"
              :disabled="respondingProject === invitation.id"
            ) Approve
            Button(
              size="sm"
              variant="outline"
              @click="respondProject(invitation.id, false)"
              :disabled="respondingProject === invitation.id"
            ) Decline

  Card
    CardHeader
      CardTitle Teams
      CardDescription Teams you belong to (administrators see every team).
    CardContent
      p.text-destructive.text-sm(v-if="error") {{ error }}
      Empty(v-if="!loading && !teams.length")
        EmptyTitle No teams yet
        EmptyDescription Create one above to get started.
      Table(v-else)
        TableHeader
          TableRow
            TableHead Team
            TableHead Members
            TableHead Role
            TableHead
        TableBody
          TableRowsSkeleton(
            v-if="loading && !teams.length"
            :widths="['w-40', 'w-16', 'w-16', 'w-12']"
          )
          TableRow(v-for="team in teams" :key="team.id")
            TableCell
              RouterLink.font-medium(
                class="hover:text-primary"
                :to="{ name: 'team', params: { id: String(team.id) } }"
              ) {{ team.name }}
              p.text-muted-foreground.text-xs(v-if="team.description") {{ team.description }}
            TableCell.text-muted-foreground {{ team.member_count }}
            TableCell
              Badge(v-if="team.role" variant="secondary") {{ team.role }}
              span.text-muted-foreground(v-else) admin
            TableCell
              .flex.justify-end
                RouterLink(
                  :to="{ name: 'team', params: { id: String(team.id) } }"
                )
                  Button(variant="outline" size="sm") Open

  Card
    CardHeader
      CardTitle Create team
      CardDescription
        | Teams group users and agents and are the primary way project access is
        | granted. The creator becomes the team's first owner.
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="create")
        FieldGroup
          Field
            FieldLabel(for="team-name") Name
            Input#team-name(
              v-model="name"
              placeholder="Platform"
              autocomplete="off"
              maxlength="100"
            )
          Field
            FieldLabel(for="team-description") Description
            Input#team-description(
              v-model="description"
              placeholder="Optional"
              autocomplete="off"
            )
        p.text-destructive.text-sm(v-if="createError") {{ createError }}
        .flex.justify-end
          Button(type="submit" :disabled="creating || !name.trim()")
            Spinner(v-if="creating" data-icon="inline-start")
            PlusIcon(v-else data-icon="inline-start")
            span Create
</template>

<script setup lang="ts">
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { withProgress } from "@/composables/useProgress";
import {
  PROJECT_ROLE_LABEL,
  projectAccessApi,
  type ProjectTeamInvitation,
} from "@/lib/projectAccess";
import { teamApi, type Team, type TeamInvitation } from "@/lib/team";
import { PlusIcon } from "@lucide/vue";
import { onMounted, ref } from "vue";
import { RouterLink, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const router = useRouter();
const teams = ref<Team[]>([]);
const pending = ref<TeamInvitation[]>([]);
const projectInvitations = ref<ProjectTeamInvitation[]>([]);
const loading = ref(true);
const error = ref("");

const name = ref("");
const description = ref("");
const creating = ref(false);
const createError = ref("");
const responding = ref(0);
const respondingProject = ref(0);

async function load() {
  loading.value = true;
  try {
    const [teamList, pendingList, projectList] = await Promise.all([
      teamApi.list(),
      teamApi.pendingInvitations(),
      projectAccessApi.pendingInvitations(),
    ]);
    teams.value = teamList.teams;
    pending.value = pendingList.invitations;
    projectInvitations.value = projectList.invitations;
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function create() {
  creating.value = true;
  createError.value = "";
  try {
    const result = await withProgress(() =>
      teamApi.create(name.value.trim(), description.value.trim()),
    );
    toast.success(`Team "${result.team.name}" created`);
    await router.push({
      name: "team",
      params: { id: String(result.team.id) },
    });
  } catch (err) {
    createError.value = err instanceof Error ? err.message : String(err);
  } finally {
    creating.value = false;
  }
}

async function respond(invitationId: number, approve: boolean) {
  responding.value = invitationId;
  try {
    await withProgress(() => teamApi.respond(invitationId, approve));
    toast.success(approve ? "Joined team" : "Invitation declined");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    responding.value = 0;
  }
}

async function respondProject(invitationId: number, approve: boolean) {
  respondingProject.value = invitationId;
  try {
    await withProgress(() => projectAccessApi.respond(invitationId, approve));
    toast.success(approve ? "Project access granted" : "Invitation declined");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    respondingProject.value = 0;
  }
}

onMounted(load);
</script>
