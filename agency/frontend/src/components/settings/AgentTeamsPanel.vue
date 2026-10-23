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
.flex.flex-col.gap-6
  Card
    CardHeader
      CardTitle Team invitations
      CardDescription
        | Teams this agent has been invited to. Approving joins the agent to the
        | team; a team member sends the invitation.
    CardContent
      Empty(v-if="!loading && !invitations.length")
        EmptyTitle No pending invitations
        EmptyDescription The agent has no team invitations to answer.
      .flex.flex-col.gap-2(v-else)
        .flex.items-center.justify-between.gap-3.rounded-lg.border.p-3(
          v-for="invitation in invitations"
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
              @click="respond(invitation, true)"
              :disabled="busy === 'i' + invitation.id"
            ) Approve
            Button(
              size="sm"
              variant="outline"
              @click="respond(invitation, false)"
              :disabled="busy === 'i' + invitation.id"
            ) Decline

  Card
    CardHeader
      CardTitle Teams
      CardDescription The teams this agent belongs to.
    CardContent
      Empty(v-if="!loading && !teams.length")
        EmptyTitle Not in any team
        EmptyDescription Invite the agent from a team's page.
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
            :widths="['w-40', 'w-16', 'w-16', 'w-16']"
          )
          TableRow(v-for="team in teams" :key="team.id")
            TableCell
              span.font-medium {{ team.name }}
              p.text-muted-foreground.text-xs(v-if="team.description")
                | {{ team.description }}
            TableCell.text-muted-foreground {{ team.member_count }}
            TableCell
              Badge(variant="secondary") {{ team.role || "member" }}
            TableCell
              .flex.justify-end
                Button(
                  variant="outline"
                  size="sm"
                  @click="leave(team)"
                  :disabled="busy === 't' + team.id"
                )
                  LogOutIcon(data-icon="inline-start")
                  span Leave
    p.text-destructive.text-sm(v-if="error") {{ error }}
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { withProgress } from "@/composables/useProgress";
import { agentTeamApi, type Team, type TeamInvitation } from "@/lib/team";
import { LogOutIcon } from "@lucide/vue";
import { onMounted, ref } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{ agentId: string }>();

const teams = ref<Team[]>([]);
const invitations = ref<TeamInvitation[]>([]);
const loading = ref(true);
const error = ref("");
const busy = ref("");

async function load() {
  loading.value = true;
  try {
    const [teamResult, invitationResult] = await Promise.all([
      agentTeamApi.list(props.agentId),
      agentTeamApi.invitations(props.agentId),
    ]);
    teams.value = teamResult.teams;
    invitations.value = invitationResult.invitations;
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function respond(invitation: TeamInvitation, approve: boolean) {
  busy.value = "i" + invitation.id;
  try {
    await withProgress(() =>
      agentTeamApi.respond(props.agentId, invitation.id, approve),
    );
    toast.success(approve ? "Invitation approved" : "Invitation declined");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function leave(team: Team) {
  if (!window.confirm(`Remove the agent from "${team.name}"?`)) return;
  busy.value = "t" + team.id;
  try {
    await withProgress(() => agentTeamApi.leave(props.agentId, team.id));
    toast.success("Agent removed from team");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

onMounted(load);
</script>
