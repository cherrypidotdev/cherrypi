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
      CardTitle Add webhook
      CardDescription Outbound JSON notifications on task events.
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="create")
        FieldGroup
          Field
            FieldLabel(for="url") URL
            Input#url(v-model="url" placeholder="https://example.com/cherrypi")
          FieldSet
            FieldLegend Events
            Field(
              v-for="option in EVENT_OPTIONS"
              :key="option.key"
              orientation="horizontal"
            )
              Switch(
                v-model="createEvents[option.key]"
                :id="`create-${option.key}`"
              )
              FieldContent
                FieldLabel(:for="`create-${option.key}`") {{ option.label }}
          Field
            FieldLabel(for="secret") Secret (optional)
            Input#secret(
              v-model="secret"
              placeholder="Sent as X-CherryPi-Secret"
            )
        .flex.justify-end
          Button(
            type="submit"
            :disabled="creating || !url.trim() || !anyCreateEvent"
          )
            Spinner(v-if="creating" data-icon="inline-start")
            PlusIcon(v-else data-icon="inline-start")
            span Add

  Card
    CardHeader
      CardTitle Webhooks
      CardDescription {{ webhooks.length }} configured.
    CardContent
      p.text-destructive.text-sm(v-if="error") {{ error }}
      Table
        TableHeader
          TableRow
            TableHead URL
            TableHead Events
            TableHead Secret
            TableHead
        TableBody
          TableRowsSkeleton(
            v-if="loading && !webhooks.length"
            :widths="['w-48', 'w-32', 'w-12', 'w-24']"
          )
          TableRow(v-for="hook in webhooks" :key="hook.id.toString()")
            TableCell.max-w-sm.truncate.font-mono.text-xs {{ hook.url }}
            TableCell
              .flex.flex-wrap.gap-1
                Badge(
                  v-for="event in hook.events"
                  :key="event"
                  variant="secondary"
                ) {{ eventLabel(event) }}
            TableCell
              Badge(v-if="hook.hasSecret" variant="outline") set
              span.text-muted-foreground(v-else) —
            TableCell
              .flex.items-center.justify-end.gap-2
                span.text-xs(
                  v-if="tryResults[hook.id.toString()]"
                  :class="tryResults[hook.id.toString()]?.ok ? 'text-muted-foreground' : 'text-destructive'"
                ) {{ tryResults[hook.id.toString()]?.text }}
                DropdownMenu
                  DropdownMenuTrigger(as-child)
                    Button(variant="outline" size="sm")
                      PlayIcon(data-icon="inline-start")
                      span Try
                  DropdownMenuContent.w-56(align="end")
                    DropdownMenuLabel Send a test payload
                    DropdownMenuGroup
                      DropdownMenuItem(
                        v-for="option in EVENT_OPTIONS"
                        :key="option.key"
                        @select="tryHook(hook, option)"
                      ) {{ option.label }}
                Button(variant="outline" size="sm" @click="openEdit(hook)")
                  PencilIcon(data-icon="inline-start")
                  span Edit
                Button(variant="outline" size="sm" @click="remove(hook.id)")
                  Trash2Icon(data-icon="inline-start")
                  span Delete
          TableRow(v-if="!webhooks.length && !loading")
            TableCell.text-muted-foreground(:colspan="4") No webhooks configured.

  Dialog(v-model:open="editingOpen")
    DialogContent
      DialogHeader
        DialogTitle Edit webhook
      form.flex.flex-col.gap-4(@submit.prevent="saveEdit")
        FieldGroup
          Field
            FieldLabel(for="edit-url") URL
            Input#edit-url(v-model="editUrl")
          FieldSet
            FieldLegend Events
            Field(
              v-for="option in EVENT_OPTIONS"
              :key="option.key"
              orientation="horizontal"
            )
              Switch(
                v-model="editEvents[option.key]"
                :id="`edit-${option.key}`"
              )
              FieldContent
                FieldLabel(:for="`edit-${option.key}`") {{ option.label }}
          Field
            FieldLabel(for="edit-secret") New secret
            Input#edit-secret(
              v-model="editSecret"
              placeholder="Leave blank to keep the current secret"
            )
          Field(orientation="horizontal")
            Switch#clear-secret(v-model="editClearSecret")
            FieldContent
              FieldLabel(for="clear-secret") Clear the current secret
        DialogFooter
          Button(type="button" variant="outline" @click="editingOpen = false")
            XIcon(data-icon="inline-start")
            span Cancel
          Button(
            type="submit"
            :disabled="saving || !editUrl.trim() || !anyEditEvent"
          )
            Spinner(v-if="saving" data-icon="inline-start")
            CheckIcon(v-else data-icon="inline-start")
            span Save
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
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Field,
  FieldContent,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { withProgress } from "@/composables/useProgress";
import { agent } from "@/lib/sdk";
import { AgencyWebhookEvent, type AgencyWebhook } from "@cherrypi/sdk";
import {
  CheckIcon,
  PencilIcon,
  PlayIcon,
  PlusIcon,
  Trash2Icon,
  XIcon,
} from "@lucide/vue";
import { computed, onMounted, reactive, ref } from "vue";
import { toast } from "vue3-toastify";

type EventKey = "user_input_requested" | "agent_state_changed";

const EVENT_OPTIONS: {
  key: EventKey;
  label: string;
  value: AgencyWebhookEvent;
}[] = [
  {
    key: "user_input_requested",
    label: "User input requested",
    value: AgencyWebhookEvent.USER_INPUT_REQUESTED,
  },
  {
    key: "agent_state_changed",
    label: "Agent state changed",
    value: AgencyWebhookEvent.AGENT_STATE_CHANGED,
  },
];

function emptyEvents(): Record<EventKey, boolean> {
  return { user_input_requested: true, agent_state_changed: false };
}

const webhooks = ref<AgencyWebhook[]>([]);
const url = ref("");
const secret = ref("");
const createEvents = reactive<Record<EventKey, boolean>>(emptyEvents());
const creating = ref(false);
const loading = ref(true);
const error = ref("");

const editingOpen = ref(false);
const editingId = ref<bigint | undefined>();
const editUrl = ref("");
const editSecret = ref("");
const editClearSecret = ref(false);
const editEvents = reactive<Record<EventKey, boolean>>(emptyEvents());
const saving = ref(false);

const tryResults = reactive<Record<string, { ok: boolean; text: string }>>({});

const anyCreateEvent = computed(() =>
  EVENT_OPTIONS.some((option) => createEvents[option.key]),
);
const anyEditEvent = computed(() =>
  EVENT_OPTIONS.some((option) => editEvents[option.key]),
);

function eventLabel(event: AgencyWebhookEvent): string {
  switch (event) {
    case AgencyWebhookEvent.USER_INPUT_REQUESTED:
      return "user_input_requested";
    case AgencyWebhookEvent.AGENT_STATE_CHANGED:
      return "agent_state_changed";
    default:
      return String(event);
  }
}

function selectedEvents(
  source: Record<EventKey, boolean>,
): AgencyWebhookEvent[] {
  return EVENT_OPTIONS.filter((option) => source[option.key]).map(
    (option) => option.value,
  );
}

async function refresh() {
  try {
    webhooks.value = await agent.listWebhooks();
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function create() {
  creating.value = true;
  try {
    await withProgress(() =>
      agent.createWebhook(
        url.value,
        selectedEvents(createEvents),
        secret.value || undefined,
      ),
    );
    url.value = "";
    secret.value = "";
    Object.assign(createEvents, emptyEvents());
    await refresh();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    creating.value = false;
  }
}

function openEdit(hook: AgencyWebhook) {
  editingId.value = hook.id;
  editUrl.value = hook.url;
  editSecret.value = "";
  editClearSecret.value = false;
  for (const option of EVENT_OPTIONS) {
    editEvents[option.key] = hook.events.includes(option.value);
  }
  editingOpen.value = true;
}

async function saveEdit() {
  const id = editingId.value;
  if (id === undefined) return;
  saving.value = true;
  try {
    const secret = editClearSecret.value ? "" : editSecret.value || undefined;
    await withProgress(() =>
      agent.updateWebhook(
        id,
        editUrl.value,
        selectedEvents(editEvents),
        secret,
      ),
    );
    editingOpen.value = false;
    await refresh();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

async function remove(id: bigint) {
  try {
    await withProgress(() => agent.deleteWebhook(id));
    toast("AgencyWebhook deleted");
    await refresh();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function tryHook(
  hook: AgencyWebhook,
  option: (typeof EVENT_OPTIONS)[number],
) {
  const key = hook.id.toString();
  tryResults[key] = { ok: true, text: "Sending…" };
  try {
    const res = await withProgress(() =>
      agent.testWebhook(hook.id, option.value),
    );
    const text = res.error
      ? `${option.label}: ${res.error}`
      : `${option.label}: HTTP ${res.statusCode}`;
    tryResults[key] = { ok: res.ok, text };
  } catch (err) {
    tryResults[key] = {
      ok: false,
      text: err instanceof Error ? err.message : String(err),
    };
  }
}

onMounted(refresh);
</script>
