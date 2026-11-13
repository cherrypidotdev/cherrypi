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
.flex.flex-col.gap-3
  span.text-sm.font-medium {{ t("projectGroups.title") }}
  p.text-muted-foreground.text-xs {{ t("projectGroups.hint") }}
  Table
    TableHeader
      TableRow
        TableHead {{ t("nav.groups") }}
        TableHead {{ t("projectGroups.members") }}
        TableHead {{ t("admin.role") }}
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
        TableCell
          Badge(variant="secondary") {{ link.role }}
        TableCell
          .flex.justify-end(v-if="canManage")
            Button(
              variant="outline"
              size="sm"
              @click="remove(link)"
              :disabled="busy === 'g' + String(link.groupId)"
            )
              Trash2Icon(data-icon="inline-start")
              span {{ t("common.remove") }}
      TableRow(v-if="!access.links.length")
        TableCell.text-muted-foreground(colspan="4")
          | {{ t("projectGroups.noGroups") }}
  .flex.flex-wrap.items-center.gap-2(v-if="canManage")
    Select(v-model="selectedGroup")
      SelectTrigger.w-56
        SelectValue(:placeholder="t('projectGroups.invitePlaceholder')")
      SelectContent
        SelectItem(
          v-for="group in availableGroups"
          :key="String(group.id)"
          :value="String(group.id)"
        ) {{ group.name }}
    Select(v-model="selectedRole")
      SelectTrigger.w-36
        SelectValue
      SelectContent
        SelectItem(value="viewer") {{ t("projectGroups.viewer") }}
        SelectItem(value="member") {{ t("projectGroups.member") }}
    Button(@click="invite" :disabled="!selectedGroup || inviting")
      PlusIcon(data-icon="inline-start")
      span {{ t("projectGroups.invite") }}
  p.text-muted-foreground.text-xs(v-if="!availableGroups.length")
    | {{ t("projectGroups.noOtherGroups") }}

  template(v-if="access.inherited.length")
    Separator
    .flex.flex-col.gap-3
      span.text-sm.font-medium {{ t("projectGroups.inherited") }}
      p.text-muted-foreground.text-xs {{ t("projectGroups.inheritedHint") }}
      Table
        TableHeader
          TableRow
            TableHead {{ t("nav.groups") }}
            TableHead {{ t("projectGroups.members") }}
            TableHead {{ t("projectSettings.teams.effectiveRole") }}
            TableHead {{ t("projectSettings.teams.inheritedFrom") }}
        TableBody
          TableRow(
            v-for="link in access.inherited"
            :key="String(link.groupId)"
          )
            TableCell.font-medium {{ link.name }}
            TableCell.text-muted-foreground {{ link.memberCount }}
            TableCell
              Badge(variant="secondary") {{ link.role }}
            TableCell
              RouterLink.text-primary(
                :to="{ name: 'project', params: { slug: link.inheritedFromProjectSlug }, query: { tab: 'settings', section: 'access' } }"
              ) {{ link.inheritedFromProjectName }}

  template(v-if="access.invitations.length")
    Separator
    .flex.flex-col.gap-3
      span.text-sm.font-medium {{ t("projectGroups.invitations") }}
      Table
        TableHeader
          TableRow
            TableHead {{ t("nav.groups") }}
            TableHead {{ t("admin.role") }}
            TableHead {{ t("common.status") }}
            TableHead
        TableBody
          TableRow(
            v-for="invitation in access.invitations"
            :key="String(invitation.id)"
          )
            TableCell.font-medium {{ invitation.groupName }}
            TableCell {{ invitation.role }}
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
                )
                  Trash2Icon(data-icon="inline-start")
                  span {{ t("common.revoke") }}
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
  AgentGroupProjectGroup,
} from "@cherrypi/sdk";
import { PlusIcon, Trash2Icon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const props = defineProps<{ projectId: string; canManage: boolean }>();
const { t } = useI18n();
const { groups, loadAgentGroups } = useAgentGroups();

const access = ref<{
  links: AgentGroupProjectGroup[];
  inherited: AgentGroupProjectGroup[];
  invitations: AgentGroupAccessInvitation[];
}>({ links: [], inherited: [], invitations: [] });
const selectedGroup = ref("");
const selectedRole = ref("member");
const inviting = ref(false);
const busy = ref("");

const availableGroups = computed(() => {
  const linked = new Set<string>([
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
  access.value = await agency.listProjectAgentGroups(props.projectId);
}

watch(
  () => props.projectId,
  () => void load(),
);
onMounted(() => void load());

async function invite() {
  if (!selectedGroup.value) return;
  inviting.value = true;
  try {
    const result = await agency.inviteAgentGroupToProject(
      selectedGroup.value,
      props.projectId,
      selectedRole.value,
    );
    toast.success(
      result.attached
        ? t("projectGroups.toasts.attached")
        : t("projectGroups.toasts.invited"),
    );
    selectedGroup.value = "";
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    inviting.value = false;
  }
}

async function remove(link: AgentGroupProjectGroup) {
  busy.value = "g" + String(link.groupId);
  try {
    await agency.removeAgentGroupFromProject(link.groupId, props.projectId);
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
    await agency.revokeAgentGroupAccess("project", invitation.id);
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
  projectGroups:
    title: Groups
    hint: A group attached here makes every member agent a project member with the chosen role, including in descendant projects.
    members: Members
    invitePlaceholder: Invite a group
    invite: Invite
    viewer: Viewer
    member: Member
    noGroups: No groups have access to this project yet.
    noOtherGroups: Every group is already attached or invited.
    inherited: Inherited groups
    inheritedHint: These groups are attached to a parent project and inherit access here. Detach them on that project.
    invitations: Group invitations
    toasts:
      attached: Group attached
      invited: Group invited
ru:
  projectGroups:
    title: Группы
    hint: Прикреплённая группа делает каждого агента-участника участником проекта с выбранной ролью, включая дочерние проекты.
    members: Участников
    invitePlaceholder: Пригласить группу
    invite: Пригласить
    viewer: Наблюдатель
    member: Участник
    noGroups: Пока ни одна группа не имеет доступа к проекту.
    noOtherGroups: Все группы уже прикреплены или приглашены.
    inherited: Унаследованные группы
    inheritedHint: Эти группы прикреплены к родительскому проекту и наследуют доступ здесь. Открепляйте их в том проекте.
    invitations: Приглашения групп
    toasts:
      attached: Группа прикреплена
      invited: Группа приглашена
</i18n>
