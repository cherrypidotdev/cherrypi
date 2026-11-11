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
.flex.flex-col.gap-4
  Card
    CardHeader
      CardTitle {{ $t("tags.title") }}
      CardAction
        Button(size="sm" @click="createOpen = true")
          PlusIcon(data-icon="inline-start")
          span {{ $t("tags.add") }}
      CardDescription
        | {{ $t("tags.description") }}
    CardContent.flex.flex-col.gap-4
      p.text-destructive.text-sm(v-if="loadError")
        | {{ loadError }}

      Table
        TableHeader
          TableRow
            TableHead {{ $t("tags.columnTag") }}
            TableHead {{ $t("tags.columnColor") }}
            TableHead {{ $t("common.description") }}
            TableHead.w-24 {{ $t("common.default") }}
            TableHead.w-28 {{ $t("tags.required") }}
            TableHead.w-20 {{ $t("nav.tasks") }}
            TableHead.w-40
        TableBody
          TableRow(v-for="tag in active" :key="tag.id")
            TableCell
              .flex.items-center.gap-2
                span.size-3.rounded(
                  :style="{ background: tagColor(tag.color) }"
                )
                span.font-medium {{ tag.name }}
                LockIcon.size-3.text-amber-500(
                  v-if="tag.isRequired"
                  :title="$t('tags.requiredHint')"
                )
            TableCell.text-muted-foreground.font-mono.text-xs
              | {{ tag.color || tagColor(tag.color) }}
            TableCell.text-muted-foreground.text-sm {{ tag.description || "—" }}
            TableCell
              Switch(
                :title="$t('tags.defaultHint')"
                :model-value="tag.isDefault"
                @update:model-value="setDefault(tag, $event)"
                :disabled="busy === tag.id"
              )
            TableCell
              Switch(
                :title="$t('tags.requiredToggleHint')"
                :model-value="tag.isRequired"
                @update:model-value="setRequired(tag, $event)"
                :disabled="busy === tag.id"
                data-test="tag-required-toggle"
              )
            TableCell.text-muted-foreground {{ tag.taskCount }}
            TableCell
              .flex.justify-end.gap-2
                Button(variant="outline" size="sm" @click="openEdit(tag)")
                  PencilIcon(data-icon="inline-start")
                  span {{ $t("common.edit") }}
                Button(variant="outline" size="sm" @click="archive(tag)")
                  ArchiveIcon(data-icon="inline-start")
                  span {{ $t("tags.archive") }}
          TableRow(v-if="!active.length")
            TableCell.text-muted-foreground(:colspan="7")
              | {{ $t("tags.empty") }}

  Card(v-if="archived.length")
    CardHeader
      CardTitle {{ $t("tags.archived.title") }}
      CardDescription
        | {{ $t("tags.archived.description") }}
    CardContent
      Table
        TableHeader
          TableRow
            TableHead {{ $t("tags.columnTag") }}
            TableHead.w-20 {{ $t("nav.tasks") }}
            TableHead.w-32
        TableBody
          TableRow(v-for="tag in archived" :key="tag.id")
            TableCell
              .flex.items-center.gap-2.opacity-60
                span.size-3.rounded(
                  :style="{ background: tagColor(tag.color) }"
                )
                span {{ tag.name }}
                LockIcon.size-3(
                  v-if="tag.isRequired"
                  :title="$t('tags.archived.requiredHint')"
                )
            TableCell.text-muted-foreground {{ tag.taskCount }}
            TableCell
              .flex.justify-end
                Button(variant="outline" size="sm" @click="restore(tag.id)")
                  ArchiveRestoreIcon(data-icon="inline-start")
                  span {{ $t("tags.restore") }}

TagEditorDialog(
  v-model:open="createOpen"
  :project-id="projectId"
  @saved="refresh"
)
TagEditorDialog(
  v-model:open="editingOpen"
  :project-id="projectId"
  :tag="editingTag"
  @saved="refresh"
)
</template>

<script setup lang="ts">
import TagEditorDialog from "@/components/tags/TagEditorDialog.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import { useTags } from "@/composables/useTags";
import { tagColor } from "@/lib/task";
import type { Tag } from "@cherrypi/sdk";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  LockIcon,
  PencilIcon,
  PlusIcon,
} from "@lucide/vue";
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const props = defineProps<{ projectId: string }>();

const {
  active,
  archived,
  error: loadError,
  refresh,
  updateTag,
  archiveTag,
  restoreTag,
} = useTags(() => props.projectId);

const createOpen = ref(false);
const editingOpen = ref(false);
const editingTag = ref<Tag | null>(null);
const busy = ref("");

function openEdit(tag: Tag) {
  editingTag.value = tag;
  editingOpen.value = true;
}

async function setDefault(tag: Tag, value: boolean) {
  busy.value = tag.id;
  try {
    await withProgress(() =>
      updateTag(tag.id, {
        name: tag.name,
        description: tag.description,
        color: tag.color,
        isDefault: value,
        isRequired: tag.isRequired,
      }),
    );
    toast.success(
      value ? t("tags.toasts.defaultSet") : t("tags.toasts.defaultRemoved"),
    );
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function setRequired(tag: Tag, value: boolean) {
  busy.value = tag.id;
  try {
    await withProgress(() =>
      updateTag(tag.id, {
        name: tag.name,
        description: tag.description,
        color: tag.color,
        isDefault: tag.isDefault,
        isRequired: value,
      }),
    );
    toast.success(
      value ? t("tags.toasts.requiredSet") : t("tags.toasts.requiredCleared"),
    );
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

async function archive(tag: Tag) {
  if (!window.confirm(`Archive tag "${tag.name}"?`)) return;
  try {
    await withProgress(() => archiveTag(tag.id));
    toast.success(t("tags.toasts.archived"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function restore(id: string) {
  try {
    await withProgress(() => restoreTag(id));
    toast.success(t("tags.toasts.restored"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}
</script>

<i18n lang="yaml">
en:
  tags:
    columnTag: "Tag"
    title: "Task tags"
    add: "Add tag"
    description: "Labels for this project's tasks, used for organization and scheduling limits. A default tag is pre-selected in the task composer; it can still be removed for any one task. A required tag only runs on agents whose concurrency limits declare it."
    columnColor: "Color"
    required: "Required"
    archive: "Archive"
    empty: "No tags yet. Add one to organize this project's tasks."
    archived:
      title: "Archived tags"
      description: "Soft-deleted tags still render on tasks that carry them."
      requiredHint: "Required by tasks that carry it"
    restore: "Restore"
    requiredHint: "Tasks with this tag only run on agents that declare it"
    defaultHint: "Pre-select this tag in the task composer"
    requiredToggleHint: "Only run tasks with this tag on agents that declare it"
    toasts:
      restored: "Tag restored"
      archived: "Tag archived"
      requiredCleared: "Tag no longer requires a declaring agent"
      requiredSet: "Tag now requires a declaring agent"
      defaultRemoved: "Tag default removed"
      defaultSet: "Tag is a composer default"
ru:
  tags:
    columnTag: "Тег"
    title: "Теги задач"
    add: "Добавить тег"
    description: "Метки для задач этого проекта, используемые для организации и лимитов планирования. Тег по умолчанию заранее выбран в композере задач; его всё равно можно убрать для отдельной задачи. Обязательный тег выполняется только на агентах, где он объявлен в лимитах параллельности."
    columnColor: "Цвет"
    required: "Обязательный"
    archive: "Архивировать"
    empty: "Тегов пока нет. Добавьте тег, чтобы упорядочить задачи проекта."
    archived:
      title: "Архивные теги"
      description: "Удалённые теги по-прежнему отображаются на задачах, которые их содержат."
      requiredHint: "Требуется задачами, которые его содержат"
    restore: "Восстановить"
    requiredHint: "Задачи с этим тегом выполняются только на агентах, которые его объявляют"
    defaultHint: "Заранее выбирать этот тег в композере задач"
    requiredToggleHint: "Выполнять задачи с этим тегом только на агентах, которые его объявляют"
    toasts:
      restored: "Тег восстановлен"
      archived: "Тег архивирован"
      requiredCleared: "Тег больше не требует агента, который его объявляет"
      requiredSet: "Теперь тег требует агента, который его объявляет"
      defaultRemoved: "Тег больше не выбирается по умолчанию"
      defaultSet: "Тег выбран по умолчанию в композере"
</i18n>
