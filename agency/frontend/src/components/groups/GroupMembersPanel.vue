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
  Card(v-if="group.canEdit")
    CardHeader
      CardTitle {{ t("groups.members.addTitle") }}
      CardDescription {{ t("groups.members.addHint") }}
    CardContent
      form.flex.flex-wrap.items-end.gap-3(@submit.prevent="add")
        .flex.min-w-60.flex-1.flex-col.gap-1
          label.text-sm.font-medium(for="group-add-agent")
            | {{ t("groups.members.agent") }}
          Select(v-model="selectedAgent")
            SelectTrigger#group-add-agent
              SelectValue(:placeholder="t('groups.members.selectAgent')")
            SelectContent
              SelectItem(
                v-for="agent in candidates"
                :key="agent.id"
                :value="agent.id"
              ) {{ agent.displayName || agent.username }}
        Button(type="submit" :disabled="!selectedAgent || adding")
          Spinner(v-if="adding" data-icon="inline-start")
          PlusIcon(v-else data-icon="inline-start")
          span {{ t("common.add") }}
      p.text-muted-foreground.text-xs(v-if="!candidates.length")
        | {{ t("groups.members.noCandidates") }}
      p.text-destructive.text-sm(v-if="addError") {{ addError }}

  Card
    CardHeader
      CardTitle {{ t("groups.members.title") }}
      CardDescription {{ t("groups.members.hint") }}
    CardContent
      Empty(v-if="!members.length")
        EmptyTitle {{ t("groups.members.empty") }}
        EmptyDescription {{ t("groups.members.emptyHint") }}
      Table(v-else)
        TableHeader
          TableRow
            TableHead {{ t("groups.members.agent") }}
            TableHead {{ t("common.status") }}
            TableHead {{ t("groups.members.addedBy") }}
            TableHead
        TableBody
          TableRow(v-for="member in members" :key="member.agentId")
            TableCell
              ActorBadge(
                kind="agent"
                :username="member.agent?.username"
                :name="member.agent?.name"
                :picture-url="member.agent?.picture?.url"
              )
            TableCell
              Badge(
                :variant="isOnline(member.agentId) ? 'secondary' : 'outline'"
              )
                | {{ isOnline(member.agentId) ? t("nav.online") : t("nav.offline") }}
            TableCell.text-muted-foreground
              ActorBadge(
                v-if="member.addedBy?.username"
                kind="user"
                :username="member.addedBy.username"
                :name="member.addedBy.name"
              )
              span(v-else) {{ t("common.unknown") }}
            TableCell
              .flex.justify-end
                Button(
                  v-if="canRemove"
                  variant="outline"
                  size="sm"
                  @click="remove(member)"
                  :disabled="busy === member.agentId"
                ) {{ t("common.remove") }}

  Card(v-if="group.canEdit || requests.length")
    CardHeader
      CardTitle {{ t("groups.requests.title") }}
      CardDescription {{ t("groups.requests.hint") }}
    CardContent
      Empty(v-if="!requests.length")
        EmptyTitle {{ t("groups.requests.empty") }}
        EmptyDescription {{ t("groups.requests.emptyHint") }}
      .flex.flex-col.divide-y(v-else)
        .flex.flex-col.gap-2.py-3(
          v-for="request in requests"
          :key="String(request.id)"
        )
          .flex.flex-wrap.items-center.gap-2
            ActorBadge(
              kind="agent"
              :username="request.agent?.username"
              :name="request.agent?.name"
            )
            Badge(variant="secondary") {{ request.status }}
            span.text-muted-foreground.text-xs
              | {{ t("groups.requests.requestedBy", { actor: request.requestedBy?.username || t("common.unknown") }) }}
            span.text-muted-foreground.text-xs
              | {{ t("groups.requests.expires", { at: formatTimestamp(request.expiresAt) }) }}
          .flex.gap-2(v-if="request.canRespond")
            Button(
              size="sm"
              @click="decide(request, true)"
              :disabled="busy === 'r' + String(request.id)"
            ) {{ t("groups.requests.approve") }}
            Button(
              size="sm"
              variant="outline"
              @click="decide(request, false)"
              :disabled="busy === 'r' + String(request.id)"
            ) {{ t("groups.requests.decline") }}
          p.text-muted-foreground.text-xs(v-else)
            | {{ t("groups.requests.waiting") }}
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
import { useAgentGroups } from "@/composables/useAgentGroups";
import { useAgents } from "@/composables/useAgents";
import { formatTimestamp } from "@/lib/format";
import { agency } from "@/lib/sdk";
import type { AgentGroupMember, AgentGroupMemberRequest } from "@cherrypi/sdk";
import { AgentStatus } from "@cherrypi/sdk";
import { PlusIcon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { toast } from "vue3-toastify";

const { t } = useI18n();
const route = useRoute();
const { groupBySlug, loadMembers, loadRequests, refresh } = useAgentGroups();
const { agents, loadAgents } = useAgents();

const members = ref<AgentGroupMember[]>([]);
const requests = ref<AgentGroupMemberRequest[]>([]);
const selectedAgent = ref("");
const adding = ref(false);
const addError = ref("");
const busy = ref("");

const slug = computed(() => String(route.params.slug ?? ""));
const group = computed(() => groupBySlug(slug.value));
const canRemove = computed(() => group.value?.canEdit === true);

const candidates = computed(() => {
  const memberIDs = new Set(members.value.map((member) => member.agentId));
  return agents.value.filter((agent) => !memberIDs.has(agent.id));
});

function isOnline(agentId: string): boolean {
  return (
    agents.value.find((agent) => agent.id === agentId)?.status ===
    AgentStatus.ONLINE
  );
}

async function load() {
  const current = group.value;
  if (!current) return;
  void loadAgents();
  [members.value, requests.value] = await Promise.all([
    loadMembers(current.id),
    loadRequests(current.id),
  ]);
}

watch(slug, () => void load());
onMounted(() => void load());

async function add() {
  const current = group.value;
  if (!current || !selectedAgent.value) return;
  adding.value = true;
  addError.value = "";
  try {
    const result = await agency.addAgentGroupMember(
      current.id,
      selectedAgent.value,
    );
    if (result.added) {
      toast.success(t("groups.members.toasts.added"));
    } else {
      toast.info(t("groups.members.toasts.requested"));
    }
    selectedAgent.value = "";
    await refresh();
    await load();
  } catch (err) {
    addError.value = err instanceof Error ? err.message : String(err);
  } finally {
    adding.value = false;
  }
}

async function remove(member: AgentGroupMember) {
  const current = group.value;
  if (!current) return;
  busy.value = member.agentId;
  try {
    await agency.removeAgentGroupMember(current.id, member.agentId);
    toast.success(t("groups.members.toasts.removed"));
    await refresh();
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function decide(request: AgentGroupMemberRequest, approve: boolean) {
  busy.value = "r" + String(request.id);
  try {
    await agency.decideAgentGroupMember(request.id, approve);
    toast.success(
      approve
        ? t("groups.requests.toasts.approved")
        : t("groups.requests.toasts.declined"),
    );
    await refresh();
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
    members:
      title: Members
      hint: An agent's projects and teams belong to the agent, not to the group.
      addTitle: Add a machine
      addHint: Adding an agent needs an owner of that agent's approval, unless you own both the group and the agent.
      agent: Agent
      selectAgent: Select an agent
      noCandidates: Every agent is already a member.
      addedBy: Added by
      empty: No members yet
      emptyHint: Add an agent to start applying the group's prompts.
      toasts:
        added: Agent added
        requested: Membership request filed for the agent's owners
        removed: Agent removed
    requests:
      title: Membership requests
      hint: Pending requests wait for an owner of the agent (or an administrator).
      empty: No pending requests
      emptyHint: A request appears here when a group owner adds an agent they do not own.
      requestedBy: "Requested by {actor}"
      expires: "Expires {at}"
      approve: Approve
      decline: Decline
      waiting: Waiting for an owner of this agent.
      toasts:
        approved: Membership request approved
        declined: Membership request declined
ru:
  groups:
    members:
      title: Участники
      hint: Проекты и команды агента принадлежат агенту, а не группе.
      addTitle: Добавить машину
      addHint: Для добавления агента нужно одобрение его владельца, если только вы не владеете и группой, и агентом.
      agent: Агент
      selectAgent: Выберите агента
      noCandidates: Все агенты уже участники.
      addedBy: Добавил
      empty: Участников пока нет
      emptyHint: Добавьте агента, чтобы промпты группы начали применяться.
      toasts:
        added: Агент добавлен
        requested: Запрос на участие отправлен владельцам агента
        removed: Агент удалён
    requests:
      title: Запросы на участие
      hint: Ожидающие запросы ждут владельца агента (или администратора).
      empty: Ожидающих запросов нет
      emptyHint: Запрос появится здесь, когда владелец группы добавит агента, которым не владеет.
      requestedBy: "Запросил {actor}"
      expires: "Истекает {at}"
      approve: Одобрить
      decline: Отклонить
      waiting: Ожидание владельца этого агента.
      toasts:
        approved: Запрос на участие одобрен
        declined: Запрос на участие отклонён
</i18n>
