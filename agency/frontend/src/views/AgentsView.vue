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
  Alert(v-if="revealed")
    AlertTitle Credential generated
    AlertDescription
      | Copy the password now — it is shown only once.
    .mt-3.flex.flex-col.gap-1.font-mono.text-sm
      span Username: {{ revealed.username }}
      span Password: {{ revealed.password }}
    .mt-3.flex.gap-2
      Button(size="sm" variant="outline" @click="copyCredential")
        CopyIcon(data-icon="inline-start")
        span Copy
      Button(size="sm" variant="ghost" @click="revealed = undefined") Dismiss

  Card
    CardHeader
      CardTitle Agents
      CardDescription Presence is live while an agent's tunnel is connected.
    CardContent
      p.text-destructive.text-sm(v-if="error") {{ error }}
      Table
        TableHeader
          TableRow
            TableHead Agent
            TableHead Username
            TableHead Status
            TableHead Version
            TableHead Address
            TableHead Last seen
            TableHead
        TableBody
          TableRowsSkeleton(
            v-if="loading && !agents.length"
            :widths="['w-40', 'w-24', 'w-20', 'w-32', 'w-28', 'w-24']"
          )
          TableRow(v-for="item in agents" :key="item.id")
            TableCell
              RouterLink.flex.items-center.gap-2.font-medium(
                class="hover:text-primary"
                :to="{ name: 'agent', params: { username: item.username } }"
              )
                img.size-6.shrink-0.rounded-full.object-cover(
                  v-if="item.picture"
                  :src="item.picture.url"
                  :alt="item.displayName || item.username"
                )
                span.size-6.shrink-0(v-else :innerHTML="avatar(item)")
                span {{ item.displayName || item.username }}
            TableCell.text-muted-foreground.font-mono {{ item.username }}
            TableCell
              Badge(variant="outline" :class="agentStatusClass(item.status)") {{ agentStatusLabel(item.status) }}
            TableCell
              .flex.items-center.gap-2
                span.font-mono.text-xs(
                  v-if="item.version"
                  :title="agentVersionTitle(item)"
                ) {{ item.version }}
                Badge(
                  v-if="agentVersionNeedsAttention(item.versionStatus)"
                  variant="outline"
                  :class="agentVersionStatusClass(item.versionStatus)"
                  :title="agentVersionTitle(item)"
                ) {{ agentVersionStatusLabel(item.versionStatus) }}
                span.text-muted-foreground(
                  v-if="!item.version && !agentVersionNeedsAttention(item.versionStatus)"
                ) —
            TableCell.text-muted-foreground.font-mono.text-xs {{ item.remoteAddr || "—" }}
            TableCell.text-muted-foreground {{ lastSeen(item) }}
            TableCell
              .flex.justify-end.gap-2
                Button(variant="outline" size="sm" @click="edit(item)")
                  PencilIcon(data-icon="inline-start")
                  span Edit
                Button(
                  variant="outline"
                  size="sm"
                  @click="remove(item)"
                  :disabled="busy === item.id"
                )
                  Trash2Icon(data-icon="inline-start")
                  span Delete
          TableRow(v-if="!agents.length && !loading")
            TableCell.text-muted-foreground(:colspan="7") No agents yet.

  //- Registering an agent is a rarer, secondary action, so the form sits after
  //- the list it adds to.
  Card
    CardHeader
      CardTitle Register agent
      CardDescription
        | Create an agent before it connects. A tunnel credential is generated
        | and shown once.
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="create")
        FieldGroup
          Field
            FieldLabel(for="agent-username") Username
            Input#agent-username(
              v-model="username"
              placeholder="builder-1"
              autocomplete="off"
            )
          Field
            FieldLabel(for="agent-display-name") Display name
            Input#agent-display-name(
              v-model="displayName"
              placeholder="Builder"
            )
          Field
            FieldLabel(for="agent-prompt") Prompt
            Textarea#agent-prompt(
              v-model="prompt"
              placeholder="Optional instructions for this agent's tasks"
            )
            FieldDescription Injected into new tasks on this agent, after the organization prompt.
        p.text-destructive.text-sm(v-if="createError") {{ createError }}
        .flex.justify-end
          Button(type="submit" :disabled="creating || !username")
            Spinner(v-if="creating" data-icon="inline-start")
            PlusIcon(v-else data-icon="inline-start")
            span Register
</template>

<script setup lang="ts">
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
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
import { Textarea } from "@/components/ui/textarea";
import { useAgents } from "@/composables/useAgents";
import { withProgress } from "@/composables/useProgress";
import {
  agentStatusClass,
  agentStatusLabel,
  agentVersionNeedsAttention,
  agentVersionStatusClass,
  agentVersionStatusLabel,
  agentVersionTitle,
} from "@/lib/agent";
import { formatTimestamp } from "@/lib/format";
import { agency } from "@/lib/sdk";
import { type Agent } from "@cherrypi/sdk";
import { CopyIcon, PencilIcon, PlusIcon, Trash2Icon } from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const { agents, loading, error, loadAgents, refresh } = useAgents();
const router = useRouter();

const username = ref("");
const displayName = ref("");
const prompt = ref("");
const creating = ref(false);
const createError = ref("");
const busy = ref("");
const revealed = ref<{ username: string; password: string } | undefined>();

const revealText = computed(() =>
  revealed.value ? `${revealed.value.username}:${revealed.value.password}` : "",
);

onMounted(() => void loadAgents());

function avatar(item: Agent): string {
  return toSvg(item.id, 24);
}

function lastSeen(item: Agent): string {
  return item.lastSeenAt ? formatTimestamp(item.lastSeenAt) : "Never";
}

async function create() {
  creating.value = true;
  createError.value = "";
  try {
    const result = await withProgress(() =>
      agency.createAgent({
        username: username.value.trim(),
        displayName: displayName.value.trim(),
        prompt: prompt.value.trim(),
      }),
    );
    revealed.value = {
      username: result.credentialUsername,
      password: result.credentialPassword,
    };
    username.value = "";
    displayName.value = "";
    prompt.value = "";
    await refresh();
  } catch (err) {
    createError.value = err instanceof Error ? err.message : String(err);
  } finally {
    creating.value = false;
  }
}

function edit(item: Agent) {
  void router.push({
    name: "agent",
    params: { username: item.username },
    query: { tab: "settings" },
  });
}

async function remove(item: Agent) {
  if (!window.confirm(`Delete agent "${item.displayName || item.username}"?`)) {
    return;
  }
  busy.value = item.id;
  try {
    await withProgress(() => agency.deleteAgent(item.id));
    toast.success("Agent deleted");
    await refresh();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function copyCredential() {
  try {
    await navigator.clipboard.writeText(revealText.value);
    toast.success("Copied");
  } catch {
    toast.error("Copy failed");
  }
}
</script>
