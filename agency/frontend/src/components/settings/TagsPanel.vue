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
      CardTitle Task tags
      CardDescription
        | Labels for this project's tasks, used for organization and scheduling
        | limits. A default tag is pre-selected in the task composer; it can
        | still be removed for any one task.
    CardContent.flex.flex-col.gap-4
      form.flex.flex-wrap.items-end.gap-3(@submit.prevent="create")
        Field.w-40
          FieldLabel(for="tag-name") Name
          Input#tag-name(v-model="name" placeholder="bug")
        Field.w-40
          FieldLabel(for="tag-description") Description
          Input#tag-description(v-model="description" placeholder="Optional")
        Field.w-24
          FieldLabel(for="tag-color") Color
          .flex.items-center.gap-2
            input#tag-color.border-input.size-8.cursor-pointer.rounded-md.border.bg-transparent(
              type="color"
              :value="color || '#a1a1aa'"
              @input="onPickColor"
            )
        Button(type="submit" :disabled="creating || !name.trim()")
          Spinner(v-if="creating" data-icon="inline-start")
          PlusIcon(v-else data-icon="inline-start")
          span Add tag

      p.text-destructive.text-sm(v-if="error || loadError")
        | {{ error || loadError }}

      Table
        TableHeader
          TableRow
            TableHead Tag
            TableHead Color
            TableHead Description
            TableHead.w-24 Default
            TableHead.w-20 Tasks
            TableHead.w-40
        TableBody
          TableRow(v-for="tag in active" :key="tag.id")
            TableCell
              .flex.items-center.gap-2
                span.size-3.rounded(
                  :style="{ background: tagColor(tag.color) }"
                )
                span.font-medium {{ tag.name }}
            TableCell.text-muted-foreground.font-mono.text-xs
              | {{ tag.color || tagColor(tag.color) }}
            TableCell.text-muted-foreground.text-sm {{ tag.description || "—" }}
            TableCell
              Switch(
                title="Pre-select this tag in the task composer"
                :model-value="tag.isDefault"
                @update:model-value="setDefault(tag, $event)"
                :disabled="busy === tag.id"
              )
            TableCell.text-muted-foreground {{ tag.taskCount }}
            TableCell
              .flex.justify-end.gap-2
                Button(variant="outline" size="sm" @click="openEdit(tag)")
                  PencilIcon(data-icon="inline-start")
                  span Edit
                Button(variant="outline" size="sm" @click="archive(tag)")
                  ArchiveIcon(data-icon="inline-start")
                  span Archive
          TableRow(v-if="!active.length")
            TableCell.text-muted-foreground(:colspan="6") No tags yet.

  Card(v-if="archived.length")
    CardHeader
      CardTitle Archived tags
      CardDescription
        | Soft-deleted tags still render on tasks that carry them.
    CardContent
      Table
        TableHeader
          TableRow
            TableHead Tag
            TableHead.w-20 Tasks
            TableHead.w-32
        TableBody
          TableRow(v-for="tag in archived" :key="tag.id")
            TableCell
              .flex.items-center.gap-2.opacity-60
                span.size-3.rounded(
                  :style="{ background: tagColor(tag.color) }"
                )
                span {{ tag.name }}
            TableCell.text-muted-foreground {{ tag.taskCount }}
            TableCell
              .flex.justify-end
                Button(variant="outline" size="sm" @click="restore(tag.id)")
                  ArchiveRestoreIcon(data-icon="inline-start")
                  span Restore

  Dialog(v-model:open="editingOpen")
    DialogContent
      DialogHeader
        DialogTitle Edit tag
      .flex.flex-col.gap-3
        Field
          FieldLabel(for="edit-tag-name") Name
          Input#edit-tag-name(v-model="editName" placeholder="bug")
        Field
          FieldLabel(for="edit-tag-description") Description
          Input#edit-tag-description(v-model="editDescription")
        Field
          FieldLabel(for="edit-tag-color") Color
          .flex.items-center.gap-2
            input#edit-tag-color.border-input.size-8.cursor-pointer.rounded-md.border.bg-transparent(
              type="color"
              :value="editColor || '#a1a1aa'"
              @input="onEditColor"
            )
            Input.max-w-32.font-mono.text-sm(v-model="editColor")
      DialogFooter
        DialogClose
          Button(variant="ghost") Cancel
        Button(@click="saveEdit" :disabled="savingEdit || !editName.trim()")
          Spinner(v-if="savingEdit" data-icon="inline-start")
          CheckIcon(v-else data-icon="inline-start")
          span Save
</template>

<script setup lang="ts">
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
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldLabel } from "@/components/ui/field";
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
import { useTags } from "@/composables/useTags";
import { tagColor } from "@/lib/task";
import type { Tag } from "@cherrypi/sdk";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  CheckIcon,
  PencilIcon,
  PlusIcon,
} from "@lucide/vue";
import { ref } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{ projectId: string }>();

const {
  active,
  archived,
  error: loadError,
  createTag,
  updateTag,
  archiveTag,
  restoreTag,
} = useTags(() => props.projectId);

const creating = ref(false);
const name = ref("");
const description = ref("");
const color = ref("");
const error = ref("");

const editingOpen = ref(false);
const editId = ref("");
const editName = ref("");
const editDescription = ref("");
const editColor = ref("");
const savingEdit = ref(false);
const busy = ref("");

function onPickColor(event: Event) {
  color.value = (event.target as HTMLInputElement).value;
}

function onEditColor(event: Event) {
  editColor.value = (event.target as HTMLInputElement).value;
}

async function create() {
  if (!name.value.trim()) return;
  creating.value = true;
  error.value = "";
  try {
    await withProgress(() =>
      createTag({
        name: name.value.trim(),
        description: description.value.trim(),
        color: color.value,
      }),
    );
    name.value = "";
    description.value = "";
    color.value = "";
    toast.success("Tag created");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    creating.value = false;
  }
}

function openEdit(tag: Tag) {
  editId.value = tag.id;
  editName.value = tag.name;
  editDescription.value = tag.description;
  editColor.value = tag.color;
  editingOpen.value = true;
}

async function saveEdit() {
  if (!editName.value.trim()) return;
  savingEdit.value = true;
  try {
    await withProgress(() =>
      updateTag(editId.value, {
        name: editName.value.trim(),
        description: editDescription.value.trim(),
        color: editColor.value,
      }),
    );
    editingOpen.value = false;
    toast.success("Tag updated");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    savingEdit.value = false;
  }
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
      }),
    );
    toast.success(value ? "Tag is a composer default" : "Tag default removed");
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
    toast.success("Tag archived");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function restore(id: string) {
  try {
    await withProgress(() => restoreTag(id));
    toast.success("Tag restored");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}
</script>
