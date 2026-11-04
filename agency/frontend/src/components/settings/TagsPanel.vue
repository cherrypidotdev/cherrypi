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
      CardAction
        Button(size="sm" @click="createOpen = true")
          PlusIcon(data-icon="inline-start")
          span Add tag
      CardDescription
        | Labels for this project's tasks, used for organization and scheduling
        | limits. A default tag is pre-selected in the task composer; it can
        | still be removed for any one task.
    CardContent.flex.flex-col.gap-4
      p.text-destructive.text-sm(v-if="loadError")
        | {{ loadError }}

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
            TableCell.text-muted-foreground(:colspan="6")
              | No tags yet. Add one to organize this project's tasks.

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
