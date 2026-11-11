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
Dialog(:open="open" @update:open="emit('update:open', $event)")
  DialogContent.overflow-y-auto.overscroll-contain(class="max-h-[92vh] sm:max-w-3xl")
    DialogHeader
      DialogTitle {{ $t("templateApply.title") }}
      DialogDescription
        | {{ $t("templateApply.description") }}
    .flex.flex-col.gap-4
      .flex.flex-col.gap-1
        span.text-sm.font-medium {{ $t("domainLog.kind.template") }}
        Combobox(
          v-model="templateId"
          :placeholder="$t('templateApply.choosePlaceholder')"
          empty-text="No matching template."
          :aria-label="$t('templateApply.templateAria')"
          :options="templateOptions"
        )
        p.text-muted-foreground.text-xs(v-if="loading") {{ $t("templateApply.loading") }}
        p.text-muted-foreground.text-xs(v-else-if="!templates.length")
          | {{ $t("templateApply.empty") }}
      template(v-if="selected")
        .flex.flex-wrap.items-center.gap-2.text-xs
          Badge(variant="secondary") v{{ Number(selected.latestVersion) }}
          span.text-muted-foreground {{ selected.description }}
        TaskTemplateArgsForm(
          v-model="args"
          :args="selected.latest?.content?.args ?? []"
        )
        .flex.flex-col.gap-1
          span.text-xs.font-medium {{ $t("settings.sounds.preview") }}
          p.text-destructive.text-xs(v-if="previewError") {{ previewError }}
          template(v-else-if="preview")
            span.text-sm.font-medium(v-if="preview.title")
              | {{ preview.title }}
            MarkdownText.prose.prose-sm.max-w-none(
              class="prose-p:my-1 prose-pre:my-2"
              :text="preview.prompt"
            )
          p.text-muted-foreground.text-xs(v-else) {{ $t("templateApply.rendering") }}
    DialogFooter
      Button(
        type="button"
        variant="outline"
        @click="emit('update:open', false)"
      )
        | {{ $t("common.cancel") }}
      Button(
        type="button"
        variant="outline"
        @click="apply"
        :disabled="!canApply"
      )
        PencilIcon(data-icon="inline-start")
        span {{ $t("templateApply.useInComposer") }}
      DropdownMenu
        DropdownMenuTrigger(as-child)
          Button(type="button" :disabled="!canApply")
            PlayIcon(data-icon="inline-start")
            span {{ $t("templateApply.createNow") }}
            ChevronDownIcon.size-3
        DropdownMenuContent.w-52(align="end")
          DropdownMenuItem(@click="createBacklog" :disabled="!canApply")
            InboxIcon.size-3
            span {{ $t("express.toBacklog") }}
          DropdownMenuItem(@click="createQueued" :disabled="!canApply")
            PlayIcon.size-3
            span {{ $t("tasks.action.queue") }}
</template>

<script setup lang="ts">
import MarkdownText from "@/components/MarkdownText.vue";
import TaskTemplateArgsForm from "@/components/templates/TaskTemplateArgsForm.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { agency } from "@/lib/sdk";
import { defaultTemplateArgs } from "@/lib/taskTemplates";
import {
  TaskPriority,
  TaskSize,
  TaskState,
  type TaskTemplate,
} from "@cherrypi/sdk";
import { ChevronDownIcon, InboxIcon, PencilIcon, PlayIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";

export interface AppliedTemplate {
  templateId: string;
  templateName: string;
  version: number;
  args: Record<string, unknown>;
  prompt: string;
  title: string;
  priority: TaskPriority;
  size: TaskSize;
  tagIds: string[];
  teamIds: number[];
  leadActorIds: string[];
}

const props = defineProps<{ open: boolean; projectId: string }>();
const emit = defineEmits<{
  "update:open": [boolean];
  apply: [AppliedTemplate];
  create: [AppliedTemplate, TaskState];
}>();

const templates = ref<TaskTemplate[]>([]);
const loading = ref(false);
const templateId = ref("");
const args = ref<Record<string, unknown>>({});
const preview = ref<{
  prompt: string;
  title: string;
  priority: TaskPriority;
  size: TaskSize;
  tagIds: string[];
  teamIds: number[];
  leadActorIds: string[];
} | null>(null);
const previewError = ref("");
let previewTimer: ReturnType<typeof setTimeout> | undefined;

const templateOptions = computed<ComboboxOption[]>(() =>
  templates.value.map((template) => ({
    value: template.id,
    label: template.name,
    description: template.description || undefined,
  })),
);
const selected = computed(() =>
  templates.value.find((template) => template.id === templateId.value),
);
const canApply = computed(() => !!preview.value && !previewError.value);

watch(
  () => props.open,
  (open) => {
    if (!open) return;
    templateId.value = "";
    args.value = {};
    preview.value = null;
    previewError.value = "";
    void loadTemplates();
  },
);

watch(
  () => props.projectId,
  () => {
    if (props.open) void loadTemplates();
  },
);

watch(templateId, () => {
  const template = selected.value;
  args.value = template?.latest
    ? defaultTemplateArgs({
        body: template.latest.content?.body ?? "",
        args: (template.latest.content?.args ?? []).map((arg) => ({
          key: arg.key,
          label: arg.label,
          description: arg.description,
          type: arg.type,
          required: arg.required,
          defaultValue: arg.defaultValue,
          options: arg.options.map((option) => ({
            value: option.value,
            label: option.label,
          })),
        })),
      })
    : {};
  schedulePreview();
});

watch(
  () => JSON.stringify(args.value),
  () => schedulePreview(),
);

async function loadTemplates() {
  if (!props.projectId) {
    templates.value = [];
    return;
  }
  loading.value = true;
  try {
    templates.value = await agency.listTaskTemplates(props.projectId, false);
  } catch {
    templates.value = [];
  } finally {
    loading.value = false;
  }
}

function schedulePreview() {
  clearTimeout(previewTimer);
  previewTimer = setTimeout(() => void renderPreview(), 250);
}

async function renderPreview() {
  const template = selected.value;
  if (!template) {
    preview.value = null;
    previewError.value = "";
    return;
  }
  try {
    const rendered = await agency.renderTaskTemplate({
      templateId: template.id,
      version: Number(template.latestVersion),
      args: args.value,
    });
    preview.value = {
      prompt: rendered.prompt,
      title: rendered.title,
      priority: rendered.priority,
      size: rendered.size,
      tagIds: rendered.tagIds,
      teamIds: rendered.teamIds,
      leadActorIds: rendered.leadActorIds,
    };
    previewError.value = "";
  } catch (err) {
    preview.value = null;
    previewError.value = err instanceof Error ? err.message : String(err);
  }
}

function payload(): AppliedTemplate | null {
  const template = selected.value;
  if (!template || !preview.value) return null;
  return {
    templateId: template.id,
    templateName: template.name,
    version: Number(template.latestVersion),
    args: args.value,
    ...preview.value,
  };
}

function apply() {
  const value = payload();
  if (!value) return;
  emit("apply", value);
  emit("update:open", false);
}

function createBacklog() {
  create(TaskState.BACKLOG);
}

function createQueued() {
  create(TaskState.QUEUED);
}

function create(state: TaskState) {
  const value = payload();
  if (!value) return;
  emit("create", value, state);
  emit("update:open", false);
}
</script>

<i18n lang="yaml">
en:
  templateApply:
    title: "Use a task template"
    description: "Pick a template, fill in its arguments, and either load the rendered prompt into the composer or create the task right away."
    loading: "Loading templates…"
    empty: "This project has no templates yet. Create one in the project's Settings → Templates tab."
    rendering: "Rendering…"
    useInComposer: "Use in composer"
    createNow: "Create now"
    choosePlaceholder: "Choose a template"
    templateAria: "Task template"
ru:
  templateApply:
    title: "Использовать шаблон задачи"
    description: "Выберите шаблон, заполните аргументы и загрузите готовый промпт в композер или сразу создайте задачу."
    loading: "Загружаем шаблоны…"
    empty: "В этом проекте пока нет шаблонов. Создайте его в настройках проекта → вкладка «Шаблоны»."
    rendering: "Формируем…"
    useInComposer: "В композер"
    createNow: "Создать сейчас"
    choosePlaceholder: "Выберите шаблон"
    templateAria: "Шаблон задачи"
</i18n>
