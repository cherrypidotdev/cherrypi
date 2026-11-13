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
Card
  CardHeader
    CardTitle {{ t("teamGroups.title") }}
    CardDescription {{ t("teamGroups.hint") }}
  CardContent.flex.flex-col.gap-4
    Table
      TableHeader
        TableRow
          TableHead {{ t("nav.groups") }}
          TableHead {{ t("teamGroups.members") }}
          TableHead {{ t("teamGroups.addedBy") }}
          TableHead
      TableBody
        TableRow(v-for="link in access.links" :key="String(link.groupId)")
          TableCell
            .flex.flex-col
              RouterLink.text-primary.font-medium(
                :to="{ name: 'group-overview', params: { slug: link.slug } }"
              ) {{ link.name }}
              span.text-muted-foreground.text-xs(v-if="link.description")
                | {{ link.description }}
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
            .flex.justify-end(v-if="canManage")
              Button(
                variant="outline"
                size="sm"
                @click="remove(link)"
                :disabled="busy === String(link.groupId)"
              ) {{ t("common.remove") }}
        TableRow(v-if="!access.links.length")
          TableCell.text-muted-foreground(colspan="4")
            | {{ t("teamGroups.noGroups") }}

    .flex.flex-wrap.items-center.gap-2(v-if="canInvite")
      Select(v-model="selectedGroup")
        SelectTrigger.w-56
          SelectValue(:placeholder="t('teamGroups.invitePlaceholder')")
        SelectContent
          SelectItem(
            v-for="group in availableGroups"
            :key="String(group.id)"
            :value="String(group.id)"
          ) {{ group.name }}
      Button(@click="invite" :disabled="!selectedGroup || inviting")
        PlusIcon(data-icon="inline-start")
        span {{ t("teamGroups.invite") }}
    p.text-muted-foreground.text-xs(
      v-if="canInvite && !availableGroups.length"
    )
      | {{ t("teamGroups.noOtherGroups") }}

    template(v-if="access.invitations.length")
      Separator
      span.text-sm.font-medium {{ t("teamGroups.invitations") }}
      Table
        TableHeader
          TableRow
            TableHead {{ t("nav.groups") }}
            TableHead {{ t("common.status") }}
            TableHead
        TableBody
          TableRow(
            v-for="invitation in access.invitations"
            :key="String(invitation.id)"
          )
            TableCell.font-medium {{ invitation.groupName }}
            TableCell.text-muted-foreground {{ invitation.status }}
            TableCell
              .flex.justify-end(
                v-if="canManage && invitation.status === 'pending'"
              )
                Button(
                  variant="outline"
                  size="sm"
                  @click="revoke(invitation)"
                  :disabled="busy === 'i' + String(invitation.id)"
                ) {{ t("common.revoke") }}
</template>

<script setup lang="ts">
import ActorBadge from "@/components/ActorBadge.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
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
  AgentGroupAccessInvitation,
  AgentGroupTeamGroup,
} from "@cherrypi/sdk";
import { PlusIcon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const props = defineProps<{
  teamId: string | number;
  canInvite: boolean;
  canManage: boolean;
}>();
const { t } = useI18n();
const { groups, loadAgentGroups } = useAgentGroups();

const access = ref<{
  links: AgentGroupTeamGroup[];
  invitations: AgentGroupAccessInvitation[];
}>({ links: [], invitations: [] });
const selectedGroup = ref("");
const inviting = ref(false);
const busy = ref("");

const availableGroups = computed(() => {
  const linked = new Set([
    ...access.value.links.map((link) => String(link.groupId)),
    ...access.value.invitations
      .filter((invitation) => invitation.status === "pending")
      .map((invitation) => String(invitation.groupId)),
  ]);
  return groups.value.filter(
    (group) => !group.archivedAt && !linked.has(String(group.id)),
  );
});

async function load() {
  await loadAgentGroups();
  access.value = await agency.listTeamAgentGroups(props.teamId);
}

watch(
  () => props.teamId,
  () => void load(),
);
onMounted(() => void load());

async function invite() {
  if (!selectedGroup.value) return;
  inviting.value = true;
  try {
    const result = await agency.inviteAgentGroupToTeam(
      selectedGroup.value,
      props.teamId,
    );
    toast.success(
      result.attached
        ? t("teamGroups.toasts.attached")
        : t("teamGroups.toasts.invited"),
    );
    selectedGroup.value = "";
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    inviting.value = false;
  }
}

async function remove(link: AgentGroupTeamGroup) {
  busy.value = String(link.groupId);
  try {
    await agency.removeAgentGroupFromTeam(link.groupId, props.teamId);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function revoke(invitation: AgentGroupAccessInvitation) {
  busy.value = "i" + String(invitation.id);
  try {
    await agency.revokeAgentGroupAccess("team", invitation.id);
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
  teamGroups:
    title: Groups
    hint: Every agent in an attached group is a team member at the member role. A group owner must accept the invitation.
    members: Members
    addedBy: Added by
    invitePlaceholder: Invite a group
    invite: Invite
    noGroups: No groups are attached to this team.
    noOtherGroups: Every group is already attached or invited.
    invitations: Group invitations
    toasts:
      attached: Group attached
      invited: Group invited
ru:
  teamGroups:
    title: Группы
    hint: Каждый агент прикреплённой группы является участником команды с ролью member. Владелец группы должен принять приглашение.
    members: Участников
    addedBy: Добавил
    invitePlaceholder: Пригласить группу
    invite: Пригласить
    noGroups: К этой команде не прикреплено ни одной группы.
    noOtherGroups: Все группы уже прикреплены или приглашены.
    invitations: Приглашения групп
    toasts:
      attached: Группа прикреплена
      invited: Группа приглашена
</i18n>
