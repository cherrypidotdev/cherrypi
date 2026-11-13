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
  Card
    CardHeader
      CardTitle {{ t("groups.settings.identity") }}
      CardDescription {{ t("groups.settings.identityHint") }}
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="save")
        FieldGroup
          Field
            FieldLabel(for="group-settings-name") {{ t("common.name") }}
            Input#group-settings-name(v-model="name" maxlength="100")
          Field
            FieldLabel(for="group-settings-slug") {{ t("groups.slug") }}
            Input#group-settings-slug(v-model="slugInput" maxlength="100")
            p.text-muted-foreground.text-xs {{ t("groups.settings.slugHint") }}
          Field
            FieldLabel(for="group-settings-description") {{ t("common.description") }}
            Input#group-settings-description(v-model="description")
        p.text-destructive.text-sm(v-if="saveError") {{ saveError }}
        .flex.justify-end
          Button(type="submit" :disabled="saving || !name.trim()")
            Spinner(v-if="saving" data-icon="inline-start")
            CheckIcon(v-else data-icon="inline-start")
            span {{ t("common.save") }}

  Card
    CardHeader
      CardTitle {{ t("groups.settings.owners") }}
      CardDescription {{ t("groups.settings.ownersHint") }}
    CardContent
      .flex.flex-col.divide-y
        .flex.items-center.justify-between.gap-3.py-2(
          v-for="owner in owners"
          :key="owner.actor?.actorId"
        )
          ActorBadge(
            kind="user"
            :username="owner.actor?.username"
            :name="owner.actor?.name"
            :picture-url="owner.actor?.picture?.url"
          )
          Button(
            v-if="group.canEdit && owners.length > 1"
            variant="outline"
            size="sm"
            @click="removeOwner(owner)"
            :disabled="busy === 'o' + String(owner.actor?.actorId)"
          ) {{ t("common.remove") }}
      form.flex.flex-wrap.items-end.gap-3.pt-4(
        v-if="group.canEdit"
        @submit.prevent="addOwner"
      )
        .flex.min-w-60.flex-1.flex-col.gap-1
          label.text-sm.font-medium(for="group-add-owner")
            | {{ t("groups.settings.addOwner") }}
          ActorCombobox#group-add-owner(
            v-model="ownerUsername"
            :placeholder="t('groups.settings.ownerPlaceholder')"
            kinds="users"
          )
        Button(type="submit" :disabled="!ownerUsername || addingOwner")
          Spinner(v-if="addingOwner" data-icon="inline-start")
          PlusIcon(v-else data-icon="inline-start")
          span {{ t("common.add") }}
      p.text-destructive.text-sm(v-if="ownerError") {{ ownerError }}

  Card
    CardHeader
      CardTitle {{ t("groups.settings.archive") }}
      CardDescription
        | {{ group.archivedAt ? t("groups.settings.unarchiveHint") : t("groups.settings.archiveHint") }}
    CardContent
      .flex.items-center.justify-between.gap-3
        p.text-muted-foreground.text-sm
          | {{ group.archivedAt ? t("groups.settings.archivedState") : t("groups.settings.archiveWarning") }}
        Button(
          v-if="group.canEdit"
          :variant="group.archivedAt ? 'default' : 'destructive'"
          @click="toggleArchived"
          :disabled="archiving"
        )
          Spinner(v-if="archiving" data-icon="inline-start")
          ArchiveIcon(v-else data-icon="inline-start")
          span {{ group.archivedAt ? t("groups.settings.unarchive") : t("groups.settings.archiveAction") }}

  Card
    CardHeader
      CardTitle {{ t("groups.activity.title") }}
      CardDescription {{ t("groups.activity.hint") }}
    CardContent
      Empty(v-if="!logs.length")
        EmptyTitle {{ t("groups.activity.empty") }}
      .flex.flex-col.divide-y(v-else)
        .flex.items-start.justify-between.gap-3.py-2(
          v-for="entry in logs"
          :key="String(entry.id)"
        )
          .flex.min-w-0.flex-col
            span.text-sm {{ entry.message }}
            span.text-muted-foreground.text-xs
              | {{ entry.actor?.username || t("groups.activity.system") }} · {{ formatTimestamp(entry.at) }}
          Badge(v-if="entry.kind" variant="outline") {{ entry.kind }}
      .flex.justify-center.pt-3(v-if="hasMore")
        Button(
          variant="outline"
          size="sm"
          @click="loadMore"
          :disabled="loadingLogs"
        )
          Spinner(v-if="loadingLogs" data-icon="inline-start")
          span {{ t("common.loadMore") }}
      div(ref="sentinel")
</template>

<script setup lang="ts">
import ActorBadge from "@/components/ActorBadge.vue";
import ActorCombobox from "@/components/ActorCombobox.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Empty, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { useAgentGroups } from "@/composables/useAgentGroups";
import { useUsers } from "@/composables/useUsers";
import { formatTimestamp } from "@/lib/format";
import { agency } from "@/lib/sdk";
import type { AgentGroupLog, AgentGroupOwner } from "@cherrypi/sdk";
import { ArchiveIcon, CheckIcon, PlusIcon } from "@lucide/vue";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { groupBySlug, loadOwners, loadLogs, updateGroup, setArchived, refresh } =
  useAgentGroups();
const { users, loadUsers } = useUsers();

const owners = ref<AgentGroupOwner[]>([]);
const logs = ref<AgentGroupLog[]>([]);
const hasMore = ref(false);
const loadingLogs = ref(false);
const busy = ref("");
const archiving = ref(false);

const name = ref("");
const slugInput = ref("");
const description = ref("");
const saving = ref(false);
const saveError = ref("");

const ownerUsername = ref("");
const addingOwner = ref(false);
const ownerError = ref("");

const sentinel = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

const slug = computed(() => String(route.params.slug ?? ""));
const group = computed(() => groupBySlug(slug.value));

const PAGE = 25;

function syncForm() {
  const current = group.value;
  if (!current) return;
  name.value = current.name;
  slugInput.value = current.slug;
  description.value = current.description;
}

async function load() {
  const current = group.value;
  if (!current) return;
  syncForm();
  void loadUsers();
  logs.value = await loadLogs(current.id, 0, PAGE);
  hasMore.value = logs.value.length === PAGE;
  owners.value = await loadOwners(current.id);
}

watch(slug, () => void load());
onMounted(() => {
  void load();
  observer = new IntersectionObserver((entries) => {
    if (entries.some((entry) => entry.isIntersecting)) void loadMore();
  });
});

watch(
  group,
  async () => {
    await nextTick();
    if (sentinel.value) observer?.observe(sentinel.value);
  },
  { immediate: true },
);
onBeforeUnmount(() => observer?.disconnect());

async function loadMore() {
  const current = group.value;
  if (!current || !hasMore.value || loadingLogs.value) return;
  const last = logs.value[logs.value.length - 1];
  if (!last) return;
  loadingLogs.value = true;
  try {
    const page = await loadLogs(current.id, last.id, PAGE);
    logs.value = [...logs.value, ...page];
    hasMore.value = page.length === PAGE;
  } finally {
    loadingLogs.value = false;
  }
}

async function save() {
  const current = group.value;
  if (!current) return;
  saving.value = true;
  saveError.value = "";
  try {
    const updated = await updateGroup(current.id, {
      name: name.value.trim(),
      slug: slugInput.value.trim() || undefined,
      description: description.value.trim(),
    });
    toast.success(t("groups.settings.toasts.saved"));
    if (updated.slug !== current.slug) {
      await router.replace({
        name: "group-settings",
        params: { slug: updated.slug },
      });
    }
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

async function addOwner() {
  const current = group.value;
  if (!current || !ownerUsername.value) return;
  const actor = users.value.find(
    (user) => user.username === ownerUsername.value,
  );
  if (!actor) {
    ownerError.value = t("groups.settings.unknownUser");
    return;
  }
  addingOwner.value = true;
  ownerError.value = "";
  try {
    await agency.addAgentGroupOwner(current.id, String(actor.id));
    ownerUsername.value = "";
    owners.value = await loadOwners(current.id);
    toast.success(t("groups.settings.toasts.ownerAdded"));
  } catch (err) {
    ownerError.value = err instanceof Error ? err.message : String(err);
  } finally {
    addingOwner.value = false;
  }
}

async function removeOwner(owner: AgentGroupOwner) {
  const current = group.value;
  if (!current || !owner.actor) return;
  busy.value = "o" + owner.actor.actorId;
  try {
    await agency.removeAgentGroupOwner(current.id, owner.actor.actorId);
    owners.value = await loadOwners(current.id);
    toast.success(t("groups.settings.toasts.ownerRemoved"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function toggleArchived() {
  const current = group.value;
  if (!current) return;
  archiving.value = true;
  try {
    await setArchived(current.id, !current.archivedAt);
    await refresh();
    toast.success(
      current.archivedAt
        ? t("groups.settings.toasts.unarchived")
        : t("groups.settings.toasts.archived"),
    );
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    archiving.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  groups:
    settings:
      identity: Identity
      identityHint: The slug is the page address. Changing it makes the old address 404 by design; every API call uses the immutable id.
      slugHint: Lowercase letters, digits and dashes. Changing it breaks old links.
      owners: Owners
      ownersHint: Owners add other users directly, with no invitation. A group keeps at least one owner.
      addOwner: Add an owner
      ownerPlaceholder: Search a user
      unknownUser: Pick a user from the list.
      archive: Archive
      archiveHint: Archiving is a soft delete. It removes every membership; un-archiving restores the group empty.
      unarchiveHint: Un-archiving restores the group empty; members are not restored.
      archiveWarning: Archiving removes every membership at once, and un-archiving does not bring them back.
      archivedState: This group is archived and controls no machines.
      archiveAction: Archive group
      unarchive: Un-archive
      toasts:
        saved: Group saved
        ownerAdded: Owner added
        ownerRemoved: Owner removed
        archived: Group archived
        unarchived: Group un-archived
    activity:
      title: Activity
      hint: Every membership, ownership and prompt change is recorded with the acting actor.
      empty: No activity yet
      system: system
ru:
  groups:
    settings:
      identity: Идентичность
      identityHint: Слаг — это адрес страницы. Его изменение по задумке ломает старый адрес (404); все вызовы API используют неизменяемый id.
      slugHint: Строчные буквы, цифры и дефисы. Изменение ломает старые ссылки.
      owners: Владельцы
      ownersHint: Владельцы добавляют других пользователей напрямую, без приглашения. У группы всегда остаётся хотя бы один владелец.
      addOwner: Добавить владельца
      ownerPlaceholder: Найдите пользователя
      unknownUser: Выберите пользователя из списка.
      archive: Архив
      archiveHint: Архивирование — это мягкое удаление. Оно убирает всех участников; распаковка возвращает пустую группу.
      unarchiveHint: Распаковка возвращает пустую группу; участники не восстанавливаются.
      archiveWarning: Архивирование сразу убирает всех участников, и распаковка их не вернёт.
      archivedState: Группа в архиве и не управляет машинами.
      archiveAction: Архивировать группу
      unarchive: Распаковать
      toasts:
        saved: Группа сохранена
        ownerAdded: Владелец добавлен
        ownerRemoved: Владелец удалён
        archived: Группа архивирована
        unarchived: Группа распакована
    activity:
      title: Активность
      hint: Каждое изменение состава, владельцев и промптов записывается с действующим актором.
      empty: Активности пока нет
      system: система
</i18n>
