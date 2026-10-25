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
    CardTitle Owners
    CardDescription
      | Owners may edit this agent, approve its team invitations and manage other
      | owners. An agent with no owner is administrator-only.
  CardContent.flex.flex-col.gap-4
    Table
      TableHeader
        TableRow
          TableHead Owner
          TableHead Kind
          TableHead Added
          TableHead
      TableBody
        TableRowsSkeleton(
          v-if="loading && !owners.length"
          :widths="['w-32', 'w-16', 'w-28', 'w-16']"
        )
        TableRow(v-for="owner in owners" :key="owner.actor_id")
          TableCell.font-medium {{ owner.username }}
          TableCell.text-muted-foreground {{ owner.kind }}
          TableCell.text-muted-foreground {{ formatDate(owner.added_at) }}
          TableCell
            .flex.justify-end
              Button(
                variant="outline"
                size="sm"
                @click="remove(owner)"
                :disabled="busy === owner.actor_id"
              )
                Trash2Icon(data-icon="inline-start")
                span Remove
        TableRow(v-if="!owners.length && !loading")
          TableCell.text-muted-foreground(:colspan="4")
            | No owners. Only administrators may manage this agent.
    form.flex.items-end.gap-2(@submit.prevent="add")
      Field.flex-1
        FieldLabel(for="new-owner") Add owner
        Input#new-owner(
          v-model="newOwner"
          placeholder="username"
          autocomplete="off"
        )
      Button(type="submit" :disabled="adding || !newOwner.trim()")
        Spinner(v-if="adding" data-icon="inline-start")
        PlusIcon(v-else data-icon="inline-start")
        span Add
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
import { Field, FieldLabel } from "@/components/ui/field";
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
import { agentOwnerApi, type AgentOwner } from "@/lib/team";
import { PlusIcon, Trash2Icon } from "@lucide/vue";
import { onMounted, ref } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{ agentId: string }>();

const owners = ref<AgentOwner[]>([]);
const loading = ref(true);
const error = ref("");
const newOwner = ref("");
const adding = ref(false);
const busy = ref("");

function formatDate(value: string): string {
  if (!value) return "—";
  return new Date(value).toLocaleString();
}

async function load() {
  loading.value = true;
  try {
    owners.value = (await agentOwnerApi.list(props.agentId)).owners;
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function add() {
  adding.value = true;
  try {
    await withProgress(() =>
      agentOwnerApi.add(props.agentId, newOwner.value.trim()),
    );
    toast.success("Owner added");
    newOwner.value = "";
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    adding.value = false;
  }
}

async function remove(owner: AgentOwner) {
  if (!window.confirm(`Remove ${owner.username} as an owner?`)) return;
  busy.value = owner.actor_id;
  try {
    await withProgress(() =>
      agentOwnerApi.remove(props.agentId, owner.actor_id),
    );
    toast.success("Owner removed");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

onMounted(load);
</script>
