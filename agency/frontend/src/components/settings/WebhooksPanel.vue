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
      CardTitle {{ $t("webhooks.add") }}
      CardDescription {{ $t("webhooks.description") }}
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="create")
        FieldGroup
          Field
            FieldLabel(for="url") {{ $t("webhooks.url") }}
            Input#url(
              v-model="url"
              :placeholder="$t('webhooks.urlPlaceholder')"
            )
          FieldSet
            FieldLegend {{ $t("webhooks.events") }}
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
                FieldLabel(:for="`create-${option.key}`") {{ $t(option.labelKey) }}
          Field
            FieldLabel(for="secret") {{ $t("webhooks.secretOptional") }}
            Input#secret(
              v-model="secret"
              :placeholder="$t('webhooks.secretHint')"
            )
        .flex.justify-end
          Button(
            type="submit"
            :disabled="creating || !url.trim() || !anyCreateEvent"
          )
            Spinner(v-if="creating" data-icon="inline-start")
            PlusIcon(v-else data-icon="inline-start")
            span {{ $t("common.add") }}

  Card
    CardHeader
      CardTitle {{ $t("settings.webhooks.title") }}
      CardDescription {{ $t("webhooks.configured", { count: webhooks.length }) }}
    CardContent
      p.text-destructive.text-sm(v-if="error") {{ error }}
      Table
        TableHeader
          TableRow
            TableHead {{ $t("webhooks.url") }}
            TableHead {{ $t("webhooks.events") }}
            TableHead {{ $t("webhooks.secret") }}
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
              Badge(v-if="hook.hasSecret" variant="outline") {{ $t("webhooks.secretSet") }}
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
                      span {{ $t("webhooks.try") }}
                  DropdownMenuContent.w-56(align="end")
                    DropdownMenuLabel {{ $t("webhooks.sendTest") }}
                    DropdownMenuGroup
                      DropdownMenuItem(
                        v-for="option in EVENT_OPTIONS"
                        :key="option.key"
                        @select="tryHook(hook, option)"
                      ) {{ $t(option.labelKey) }}
                Button(variant="outline" size="sm" @click="openEdit(hook)")
                  PencilIcon(data-icon="inline-start")
                  span {{ $t("common.edit") }}
                Button(variant="outline" size="sm" @click="remove(hook.id)")
                  Trash2Icon(data-icon="inline-start")
                  span {{ $t("common.delete") }}
          TableRow(v-if="!webhooks.length && !loading")
            TableCell.text-muted-foreground(:colspan="4") {{ $t("webhooks.empty") }}

  Dialog(v-model:open="editingOpen")
    DialogContent
      DialogHeader
        DialogTitle {{ $t("webhooks.edit") }}
      form.flex.flex-col.gap-4(@submit.prevent="saveEdit")
        FieldGroup
          Field
            FieldLabel(for="edit-url") {{ $t("webhooks.url") }}
            Input#edit-url(v-model="editUrl")
          FieldSet
            FieldLegend {{ $t("webhooks.events") }}
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
                FieldLabel(:for="`edit-${option.key}`") {{ $t(option.labelKey) }}
          Field
            FieldLabel(for="edit-secret") {{ $t("webhooks.newSecret") }}
            Input#edit-secret(
              v-model="editSecret"
              :placeholder="$t('webhooks.keepSecret')"
            )
          Field(orientation="horizontal")
            Switch#clear-secret(v-model="editClearSecret")
            FieldContent
              FieldLabel(for="clear-secret") {{ $t("webhooks.clearSecret") }}
        DialogFooter
          Button(type="button" variant="outline" @click="editingOpen = false")
            XIcon(data-icon="inline-start")
            span {{ $t("common.cancel") }}
          Button(
            type="submit"
            :disabled="saving || !editUrl.trim() || !anyEditEvent"
          )
            Spinner(v-if="saving" data-icon="inline-start")
            CheckIcon(v-else data-icon="inline-start")
            span {{ $t("common.save") }}
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
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

type EventKey = "user_input_requested" | "agent_state_changed";

const EVENT_OPTIONS: {
  key: EventKey;
  labelKey: string;
  value: AgencyWebhookEvent;
}[] = [
  {
    key: "user_input_requested",
    labelKey: "webhooks.eventInputRequested",
    value: AgencyWebhookEvent.USER_INPUT_REQUESTED,
  },
  {
    key: "agent_state_changed",
    labelKey: "webhooks.eventAgentState",
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
    toast(t("webhooks.toasts.deleted"));
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
  tryResults[key] = { ok: true, text: t("webhooks.sending") };
  try {
    const res = await withProgress(() =>
      agent.testWebhook(hook.id, option.value),
    );
    const text = res.error
      ? `${t(option.labelKey)}: ${res.error}`
      : `${t(option.labelKey)}: HTTP ${res.statusCode}`;
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

<i18n lang="yaml">
en:
  webhooks:
    events: "Events"
    add: "Add webhook"
    description: "Outbound JSON notifications on task events."
    url: "URL"
    secretOptional: "Secret (optional)"
    secret: "Secret"
    secretSet: "set"
    try: "Try"
    sendTest: "Send a test payload"
    empty: "No webhooks configured."
    edit: "Edit webhook"
    newSecret: "New secret"
    clearSecret: "Clear the current secret"
    urlPlaceholder: "https://example.com/cherrypi"
    secretHint: "Sent as X-CherryPi-Secret"
    keepSecret: "Leave blank to keep the current secret"
    sending: "Sending…"
    eventInputRequested: "User input requested"
    eventAgentState: "Agent state changed"
    toasts:
      deleted: "Webhook deleted"
    configured: "{count} configured."
ru:
  webhooks:
    events: "События"
    add: "Добавить вебхук"
    description: "Исходящие JSON-уведомления о событиях задач."
    url: "URL"
    secretOptional: "Секрет (необязательно)"
    secret: "Секрет"
    secretSet: "задан"
    try: "Проверить"
    sendTest: "Отправить тестовое уведомление"
    empty: "Вебхуки не настроены."
    edit: "Изменить вебхук"
    newSecret: "Новый секрет"
    clearSecret: "Удалить текущий секрет"
    urlPlaceholder: "https://example.com/cherrypi"
    secretHint: "Отправляется как X-CherryPi-Secret"
    keepSecret: "Оставьте пустым, чтобы сохранить текущий секрет"
    sending: "Отправка…"
    eventInputRequested: "Запрошен ввод пользователя"
    eventAgentState: "Состояние агента изменилось"
    toasts:
      deleted: "Вебхук удалён"
    configured: "настроено: {count}."
</i18n>
