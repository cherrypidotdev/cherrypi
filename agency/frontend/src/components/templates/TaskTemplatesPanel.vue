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
  .flex.flex-wrap.items-center.justify-between.gap-2
    .flex.flex-col
      span.text-sm.font-medium Task templates
      span.text-muted-foreground.text-xs
        | Reusable Jinja prompts with arguments and task defaults. Members and
        | owners may edit them; viewers may apply them.
    Button(@click="openCreate" :disabled="!projectId")
      PlusIcon(data-icon="inline-start")
      span New template
  p.text-destructive.text-sm(v-if="error") {{ error }}
  p.text-muted-foreground.text-xs(v-if="loading") Loading…
  p.text-muted-foreground.text-xs(
    v-else-if="!active.length && !archived.length"
  )
    | No templates yet. Create one to reuse a prompt with fill-in arguments.

  .flex.flex-col.gap-2(v-if="active.length")
    .flex.items-start.justify-between.gap-3.rounded-md.border.px-3(
      v-for="template in active"
      :key="template.id"
      class="py-2.5"
    )
      .flex.min-w-0.flex-col
        .flex.items-center.gap-2
          span.font-medium {{ template.name }}
          Badge(variant="secondary") v{{ Number(template.latestVersion) }}
        span.text-muted-foreground.text-xs(v-if="template.description")
          | {{ template.description }}
        span.text-muted-foreground.text-xs
          | {{ template.latest?.content?.args?.length ?? 0 }} argument(s) ·
          | updated {{ formatRelativeTimestamp(template.updatedAt) }}
      .flex.shrink-0.items-center.gap-1
        Button(variant="outline" size="sm" @click="openEdit(template)")
          PencilIcon(data-icon="inline-start")
          span Edit
        Button(
          variant="ghost"
          size="icon-sm"
          title="Archive template"
          aria-label="Archive template"
          @click="archive(template)"
        )
          ArchiveIcon.size-3

  template(v-if="archived.length")
    Separator
    .flex.flex-col.gap-2
      span.text-muted-foreground.text-xs Archived
      .flex.items-center.justify-between.gap-3.rounded-md.border.px-3.py-2.opacity-70(
        v-for="template in archived"
        :key="template.id"
      )
        .flex.min-w-0.flex-col
          span.text-sm {{ template.name }}
          span.text-muted-foreground.text-xs
            | v{{ Number(template.latestVersion) }} ·
            | {{ template.latest?.content?.args?.length ?? 0 }} argument(s)
        Button(variant="outline" size="sm" @click="unarchive(template)")
          ArchiveRestoreIcon(data-icon="inline-start")
          span Restore

  TaskTemplateEditor(
    v-model:open="editorOpen"
    :project-id="projectId"
    :template="editing"
    @saved="onSaved"
  )
</template>

<script setup lang="ts">
import TaskTemplateEditor from "@/components/templates/TaskTemplateEditor.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { formatRelativeTimestamp } from "@/lib/format";
import { agency } from "@/lib/sdk";
import type { TaskTemplate } from "@cherrypi/sdk";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  PencilIcon,
  PlusIcon,
} from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{ projectId: string }>();

const templates = ref<TaskTemplate[]>([]);
const loading = ref(false);
const error = ref("");
const editorOpen = ref(false);
const editing = ref<TaskTemplate | null>(null);

const active = computed(() =>
  templates.value.filter((template) => !template.archivedAt),
);
const archived = computed(() =>
  templates.value.filter((template) => !!template.archivedAt),
);

async function load() {
  if (!props.projectId) {
    templates.value = [];
    return;
  }
  loading.value = true;
  try {
    templates.value = await agency.listTaskTemplates(props.projectId, true);
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editing.value = null;
  editorOpen.value = true;
}

function openEdit(template: TaskTemplate) {
  editing.value = template;
  editorOpen.value = true;
}

function onSaved(template: TaskTemplate) {
  editing.value = template;
  void load();
}

async function archive(template: TaskTemplate) {
  try {
    await agency.archiveTaskTemplate(template.id);
    toast.success("Template archived");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function unarchive(template: TaskTemplate) {
  try {
    await agency.unarchiveTaskTemplate(template.id);
    toast.success("Template restored");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

watch(() => props.projectId, load);
onMounted(load);
</script>
