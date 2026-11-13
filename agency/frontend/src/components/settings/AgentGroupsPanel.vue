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
    CardTitle {{ t("agentGroups.title") }}
    CardDescription {{ t("agentGroups.hint") }}
  CardContent
    Empty(v-if="!loading && !groups.length")
      EmptyTitle {{ t("agentGroups.empty") }}
      EmptyDescription {{ t("agentGroups.emptyHint") }}
    Table(v-else)
      TableHeader
        TableRow
          TableHead {{ t("common.name") }}
          TableHead {{ t("groups.memberColumn") }}
          TableHead
      TableBody
        TableRowsSkeleton(
          v-if="loading && !groups.length"
          :widths="['w-40', 'w-16', 'w-16']"
        )
        TableRow(v-for="group in groups" :key="String(group.id)")
          TableCell
            RouterLink.font-medium(
              class="hover:text-primary"
              :to="{ name: 'group-overview', params: { slug: group.slug } }"
            ) {{ group.name }}
            p.text-muted-foreground.text-xs(v-if="group.description")
              | {{ group.description }}
          TableCell.text-muted-foreground {{ group.memberCount }}
          TableCell
            .flex.justify-end
              Button(
                variant="outline"
                size="sm"
                @click="leave(group)"
                :disabled="busy === String(group.id)"
              )
                LogOutIcon(data-icon="inline-start")
                span {{ t("agentGroups.leave") }}
    p.text-destructive.text-sm(v-if="error") {{ error }}
</template>

<script setup lang="ts">
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
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
import type { AgentGroup } from "@cherrypi/sdk";
import { LogOutIcon } from "@lucide/vue";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const props = defineProps<{ agentId: string }>();

const { t } = useI18n();
const { groupsForAgent, refresh } = useAgentGroups();

const groups = ref<AgentGroup[]>([]);
const loading = ref(true);
const error = ref("");
const busy = ref("");

async function load() {
  loading.value = true;
  try {
    groups.value = await groupsForAgent(props.agentId);
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => void load());

async function leave(group: AgentGroup) {
  busy.value = String(group.id);
  try {
    await agency.removeAgentGroupMember(group.id, props.agentId);
    toast.success(t("agentGroups.toasts.left", { group: group.name }));
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
  agentGroups:
    title: Agent groups
    hint: Groups this machine belongs to. Every one of them contributes its prompts to this agent's tasks, in group-id order.
    empty: Not in a group
    emptyHint: A group owner adds this machine; its own owner may remove it at any time.
    leave: Leave
    toasts:
      left: "Left the {group} group"
ru:
  agentGroups:
    title: Группы агентов
    hint: Группы, в которых состоит эта машина. Каждая добавляет свои промпты в задачи агента в порядке id группы.
    empty: Не входит в группы
    emptyHint: Машину добавляет владелец группы; её собственный владелец может убрать её в любой момент.
    leave: Покинуть
    toasts:
      left: "Машина покинула группу {group}"
</i18n>
