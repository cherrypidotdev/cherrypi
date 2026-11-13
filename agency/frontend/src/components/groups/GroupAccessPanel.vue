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
.flex.flex-col.gap-6(v-if="group")
  Card(v-if="access.invitations.length")
    CardHeader
      CardTitle {{ t("groups.access.invitationsTitle") }}
      CardDescription {{ t("groups.access.invitationsHint") }}
    CardContent
      Table
        TableHeader
          TableRow
            TableHead {{ t("groups.access.target") }}
            TableHead {{ t("admin.role") }}
            TableHead {{ t("common.status") }}
            TableHead {{ t("groups.access.invitedBy") }}
            TableHead
        TableBody
          TableRow(
            v-for="invitation in access.invitations"
            :key="invitation.targetKind + String(invitation.id)"
          )
            TableCell.font-medium
              | {{ targetName(invitation) }}
            TableCell
              Badge(variant="secondary") {{ invitation.role }}
            TableCell.text-muted-foreground {{ invitation.status }}
            TableCell.text-muted-foreground
              ActorBadge(
                v-if="invitation.invitedBy?.username"
                :kind="invitation.invitedBy.kind || 'user'"
                :username="invitation.invitedBy.username"
                :name="invitation.invitedBy.name"
              )
              span(v-else) {{ t("common.unknown") }}
            TableCell
              .flex.justify-end.gap-2(v-if="invitation.canRespond")
                Button(
                  size="sm"
                  @click="decide(invitation, true)"
                  :disabled="busy === invitationKey(invitation)"
                ) {{ t("groups.access.accept") }}
                Button(
                  size="sm"
                  variant="outline"
                  @click="decide(invitation, false)"
                  :disabled="busy === invitationKey(invitation)"
                ) {{ t("groups.access.decline") }}
                Button(
                  size="sm"
                  variant="outline"
                  @click="revoke(invitation)"
                  :disabled="busy === invitationKey(invitation)"
                ) {{ t("common.revoke") }}

  Card
    CardHeader
      CardTitle {{ t("groups.access.projectsTitle") }}
      CardDescription {{ t("groups.access.projectsHint") }}
    CardContent
      Empty(v-if="!access.projects.length")
        EmptyTitle {{ t("groups.access.noProjects") }}
        EmptyDescription {{ t("groups.access.noProjectsHint") }}
      Table(v-else)
        TableHeader
          TableRow
            TableHead {{ t("nav.projects") }}
            TableHead {{ t("admin.role") }}
            TableHead {{ t("groups.access.grantedBy") }}
            TableHead
        TableBody
          TableRow(v-for="link in access.projects" :key="link.projectId")
            TableCell
              RouterLink.text-primary(
                :to="{ name: 'project', params: { slug: link.slug } }"
              ) {{ link.name }}
            TableCell
              Badge(variant="secondary") {{ link.role }}
            TableCell.text-muted-foreground
              ActorBadge(
                v-if="link.grantedBy?.username"
                :kind="link.grantedBy.kind || 'user'"
                :username="link.grantedBy.username"
                :name="link.grantedBy.name"
              )
              span(v-else) {{ t("common.unknown") }}
            TableCell
              .flex.justify-end(v-if="group.isOwner")
                Button(
                  size="sm"
                  variant="outline"
                  @click="leaveProject(link)"
                  :disabled="busy === 'project' + link.projectId"
                ) {{ t("groups.access.leave") }}

  Card
    CardHeader
      CardTitle {{ t("groups.access.teamsTitle") }}
      CardDescription {{ t("groups.access.teamsHint") }}
    CardContent
      Empty(v-if="!access.teams.length")
        EmptyTitle {{ t("groups.access.noTeams") }}
        EmptyDescription {{ t("groups.access.noTeamsHint") }}
      Table(v-else)
        TableHeader
          TableRow
            TableHead {{ t("nav.teams") }}
            TableHead {{ t("admin.role") }}
            TableHead {{ t("groups.access.members") }}
            TableHead {{ t("groups.access.grantedBy") }}
            TableHead
        TableBody
          TableRow(v-for="link in access.teams" :key="String(link.teamId)")
            TableCell
              RouterLink.text-primary(
                :to="{ name: 'team', params: { id: String(link.teamId) } }"
              ) {{ link.name }}
            TableCell
              Badge(variant="secondary") {{ link.role }}
            TableCell.text-muted-foreground {{ link.memberCount }}
            TableCell.text-muted-foreground
              ActorBadge(
                v-if="link.grantedBy?.username"
                :kind="link.grantedBy.kind || 'user'"
                :username="link.grantedBy.username"
                :name="link.grantedBy.name"
              )
              span(v-else) {{ t("common.unknown") }}
            TableCell
              .flex.justify-end(v-if="group.isOwner")
                Button(
                  size="sm"
                  variant="outline"
                  @click="leaveTeam(link)"
                  :disabled="busy === 'team' + String(link.teamId)"
                ) {{ t("groups.access.leave") }}

  p.text-muted-foreground.text-sm(v-if="group.isOwner")
    | {{ t("groups.access.initiateHint") }}
</template>

<script setup lang="ts">
import ActorBadge from "@/components/ActorBadge.vue";
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
import { useAgentGroups } from "@/composables/useAgentGroups";
import { agency } from "@/lib/sdk";
import type {
  AgentGroupAccess,
  AgentGroupAccessInvitation,
  AgentGroupProjectAccess,
  AgentGroupTeamAccess,
} from "@cherrypi/sdk";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { toast } from "vue3-toastify";

const { t } = useI18n();
const route = useRoute();
const { groupBySlug } = useAgentGroups();

const access = ref<AgentGroupAccess>({
  $typeName: "agency.v1.AgentGroupAccess",
  projects: [],
  teams: [],
  invitations: [],
});
const busy = ref("");

const slug = computed(() => String(route.params.slug ?? ""));
const group = computed(() => groupBySlug(slug.value));

function targetName(invitation: AgentGroupAccessInvitation): string {
  return invitation.targetKind === "project"
    ? invitation.projectName
    : invitation.teamName;
}

function invitationKey(invitation: AgentGroupAccessInvitation): string {
  return invitation.targetKind + String(invitation.id);
}

async function load() {
  const current = group.value;
  if (!current) return;
  access.value = await agency.listAgentGroupAccess(current.id);
}

watch(slug, () => void load());
onMounted(() => void load());

async function decide(
  invitation: AgentGroupAccessInvitation,
  approve: boolean,
) {
  busy.value = invitationKey(invitation);
  try {
    await agency.decideAgentGroupAccess(
      invitation.targetKind as "project" | "team",
      invitation.id,
      approve,
    );
    toast.success(
      approve
        ? t("groups.access.toasts.accepted")
        : t("groups.access.toasts.declined"),
    );
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function revoke(invitation: AgentGroupAccessInvitation) {
  busy.value = invitationKey(invitation);
  try {
    await agency.revokeAgentGroupAccess(
      invitation.targetKind as "project" | "team",
      invitation.id,
    );
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function leaveProject(link: AgentGroupProjectAccess) {
  const current = group.value;
  if (!current) return;
  busy.value = "project" + link.projectId;
  try {
    await agency.removeAgentGroupFromProject(current.id, link.projectId);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function leaveTeam(link: AgentGroupTeamAccess) {
  const current = group.value;
  if (!current) return;
  busy.value = "team" + String(link.teamId);
  try {
    await agency.removeAgentGroupFromTeam(current.id, link.teamId);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}
</script>

<i18n lang="yaml">
en:
  groups:
    access:
      invitationsTitle: Invitations
      invitationsHint: A project owner or team member invited this group. Accepting makes every member of the group a project/team member.
      target: Project or team
      invitedBy: Invited by
      accept: Accept
      decline: Decline
      projectsTitle: Projects
      projectsHint: The projects this group is attached to. Its members inherit the role shown here, including descendant projects.
      noProjects: No project access
      noProjectsHint: A project owner can invite this group from the project's Access settings.
      teamsTitle: Teams
      teamsHint: The teams this group is a member of. Its members are treated as team members everywhere, at the member role.
      noTeams: No team access
      noTeamsHint: A team member can invite this group from the team's Groups tab.
      members: Members
      grantedBy: Added by
      leave: Leave
      initiateHint: Access is started from the project or team side. Once attached, a group owner may leave here.
      toasts:
        accepted: Invitation accepted
        declined: Invitation declined
ru:
  groups:
    access:
      invitationsTitle: Приглашения
      invitationsHint: Владелец проекта или участник команды пригласил эту группу. Принятие делает каждого участника группы участником проекта/команды.
      target: Проект или команда
      invitedBy: Пригласил
      accept: Принять
      decline: Отклонить
      projectsTitle: Проекты
      projectsHint: Проекты, к которым прикреплена группа. Её участники получают указанную роль, включая дочерние проекты.
      noProjects: Нет доступа к проектам
      noProjectsHint: Владелец проекта может пригласить группу на вкладке доступа проекта.
      teamsTitle: Команды
      teamsHint: Команды, в которых состоит группа. Её участники считаются участниками команды с ролью member.
      noTeams: Нет доступа к командам
      noTeamsHint: Участник команды может пригласить группу на вкладке групп команды.
      members: Участников
      grantedBy: Добавил
      leave: Покинуть
      initiateHint: Доступ выдаётся со стороны проекта или команды. После прикрепления владелец группы может выйти здесь.
      toasts:
        accepted: Приглашение принято
        declined: Приглашение отклонено
</i18n>
