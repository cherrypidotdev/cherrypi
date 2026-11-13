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
div(:class="embedded ? '' : 'bg-card overflow-hidden rounded-lg border'")
  .flex.items-start.gap-3(:class="embedded ? 'pb-2' : 'p-3'")
    .min-w-0.flex-1
      .flex.flex-wrap.items-center(class="gap-1.5")
        span.text-sm.font-medium {{ title }}
        Badge(variant="secondary") Jinja
      p.text-muted-foreground.mt-1.text-xs(v-if="description") {{ description }}
    .flex.items-center.gap-1
      Button(
        v-if="!matchesDefault"
        type="button"
        variant="ghost"
        size="sm"
        @click="resetToDefault"
      ) {{ $t("pluginTemplate.resetToDefault") }}
      Button(type="button" variant="ghost" size="sm" @click="expanded = true")
        MaximizeIcon(data-icon="inline-start")
        span {{ $t("pluginTemplate.expand") }}
  PluginTemplateVariablePicker.px-3.pb-2(
    :variables="variables"
    @insert="insertInto(inlineEditor, $event)"
  )
  .border-t
    .grid.grid-cols-1.grid-rows-2.overflow-hidden(
      ref="panes"
      class="md:grid-cols-2 md:grid-rows-1"
      :style="{ height: `${paneHeight}px` }"
      data-testid="plugin-template-panes"
    )
      .flex.min-h-0.flex-col.overflow-hidden
        .text-muted-foreground.flex.h-8.shrink-0.items-center.justify-between.border-b.px-2.text-xs.font-medium
          span {{ $t("pluginTemplate.editor") }}
          span.font-mono Jinja
        .min-h-0.flex-1(
          :class="hasErrors ? 'ring-1 ring-inset ring-destructive/40' : ''"
        )
          JinjaTemplateEditor(
            ref="inlineEditor"
            :label="$t('pluginTemplate.editorAria', { title })"
            :model-value="modelValue"
            :invalid="hasErrors"
            @update:model-value="onInput"
          )
      .flex.min-h-0.flex-col.overflow-hidden(class="md:border-l")
        .text-muted-foreground.flex.h-8.shrink-0.items-center.justify-between.border-b.px-2.text-xs.font-medium
          span {{ $t("pluginTemplate.preview") }}
          span {{ $t("pluginTemplate.withSampleData") }}
        pre.min-h-0.flex-1.overflow-auto.p-2.text-xs.leading-relaxed.break-words.whitespace-pre-wrap(
          :class="hasErrors ? 'bg-destructive/5 text-destructive' : ''"
        ) {{ previewText }}
  .flex.h-2.cursor-row-resize.items-center.justify-center.border-t(
    :title="$t('pluginTemplate.resizeHint')"
    @pointerdown="startResize"
    data-testid="plugin-template-resize"
  )
    .bg-border.h-1.w-10.rounded-full.transition-opacity(
      :class="dragging ? 'opacity-100' : 'opacity-40 hover:opacity-100'"
    )
  .flex.flex-wrap.items-center.gap-2.border-t.px-3.py-1.text-xs
    span.flex.items-center.gap-1.text-emerald-600(v-if="!hasErrors")
      CheckIcon.size-3
      span {{ statusLabel }} · {{ $t("pluginTemplate.variablesUsed", { count: usedCount }) }}
    span.text-destructive.flex.items-center.gap-1(v-else)
      TriangleAlertIcon.size-3
      span {{ firstError }}
    .flex-1
    span.text-muted-foreground {{ hint }}

Dialog(v-model:open="expanded")
  DialogContent.overflow-y-auto(class="max-h-[90vh] sm:max-w-5xl")
    DialogHeader
      DialogTitle {{ title }}
      DialogDescription {{ $t("pluginTemplate.largerEditor") }}
    PluginTemplateVariablePicker.pb-2(
      :variables="variables"
      @insert="insertInto(dialogEditor, $event)"
    )
    .grid.h-96.gap-3(class="md:grid-cols-2")
      .flex.min-h-0.flex-col.gap-1
        span.text-xs.font-medium {{ $t("pluginTemplate.editor") }}
        .min-h-0.flex-1.overflow-hidden.rounded-md.border(
          :class="hasErrors ? 'border-destructive' : ''"
        )
          JinjaTemplateEditor(
            ref="dialogEditor"
            :label="$t('pluginTemplate.editorAria', { title })"
            :model-value="modelValue"
            :invalid="hasErrors"
            @update:model-value="onInput"
          )
      .flex.min-h-0.flex-col.gap-1
        span.text-xs.font-medium {{ $t("pluginTemplate.preview") }}
        pre.h-full.min-h-0.overflow-auto.rounded-md.border.p-2.text-xs.leading-relaxed.break-words.whitespace-pre-wrap(
          class="bg-muted/20"
          :class="hasErrors ? 'text-destructive' : ''"
        ) {{ previewText }}
    p.text-destructive.text-xs(v-if="props.error") {{ props.error }}
    DialogFooter
      Button(type="button" variant="outline" @click="expanded = false")
        | {{ $t("common.close") }}
</template>

<script setup lang="ts">
import JinjaTemplateEditor from "@/components/plugins/JinjaTemplateEditor.vue";
import PluginTemplateVariablePicker from "@/components/plugins/PluginTemplateVariablePicker.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  EDITOR_MAX_HEIGHT,
  clampEditorHeightInRange,
} from "@/lib/editorHeight";
import {
  pluginTemplateSamples,
  type PluginTemplateMeta,
  type PluginTemplateVariable,
} from "@/lib/plugins";
import { agency } from "@/lib/sdk";
import type { PreviewPluginTemplateResponse } from "@cherrypi/sdk";
import { CheckIcon, MaximizeIcon, TriangleAlertIcon } from "@lucide/vue";
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  fieldKey: string;
  title: string;
  description?: string;
  defaultValue: string;
  template: PluginTemplateMeta | null;
  modelValue: string;
  error?: string;
  hint?: string;
  pluginId?: string;
  scope?: string;
  targetId?: string;
  sampleOverrides?: Record<string, unknown>;

  embedded?: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

interface TemplateEditorHandle {
  insert: (snippet: string) => void;
}

const inlineEditor = ref<TemplateEditorHandle | null>(null);
const dialogEditor = ref<TemplateEditorHandle | null>(null);
const expanded = ref(false);
const preview = ref<PreviewPluginTemplateResponse | null>(null);
const previewFailure = ref("");
let timer: number | undefined;
let previewRequest = 0;

const TEMPLATE_PANE_MIN_HEIGHT = 160;
const TEMPLATE_PANE_DEFAULT_HEIGHT = 256;
const paneHeight = ref(TEMPLATE_PANE_DEFAULT_HEIGHT);
const dragging = ref(false);
let dragStartY = 0;
let dragStartHeight = 0;

function clampPaneHeight(height: number): number {
  return clampEditorHeightInRange(
    height,
    TEMPLATE_PANE_MIN_HEIGHT,
    EDITOR_MAX_HEIGHT,
  );
}

const variables = computed<Record<string, PluginTemplateVariable>>(
  () => props.template?.variables ?? {},
);
const matchesDefault = computed(() => props.modelValue === props.defaultValue);
const explicitEmpty = computed(
  () => !matchesDefault.value && props.modelValue.trim() === "",
);
const statusLabel = computed(() => {
  if (matchesDefault.value) return t("pluginTemplate.usingDefault");
  if (explicitEmpty.value) return t("pluginTemplate.explicitEmpty");
  return t("pluginTemplate.validJinja");
});
const hasErrors = computed(
  () =>
    !!props.error ||
    !!previewFailure.value ||
    (preview.value?.errors.length ?? 0) > 0,
);
const firstError = computed(
  () =>
    props.error ||
    previewFailure.value ||
    preview.value?.errors[0] ||
    "Preview unavailable",
);
const previewText = computed(() => {
  if (previewFailure.value) return previewFailure.value;
  if (preview.value?.errors.length) return preview.value.errors.join("\n");
  if (preview.value) {
    return preview.value.rendered || "(renders an empty string)";
  }
  return "Rendering…";
});
const usedCount = computed(() => preview.value?.variables.length ?? 0);

function onInput(value: string) {
  emit("update:modelValue", value);
}

function resetToDefault() {
  emit("update:modelValue", props.defaultValue);
}

function variableSnippet(name: string): string {
  const variable = variables.value[name];
  if (variable?.type !== "array") return `{{ ${name} }}`;
  const itemName = name.split(".").pop()?.replace(/s$/, "") || "item";
  const properties = Object.keys(variable.items?.properties ?? {});
  if (properties.length) {
    return `{% for ${itemName} in ${name} %}\n- ${properties
      .map((property) => `{{ ${itemName}.${property} }}`)
      .join(": ")}\n{% endfor %}`;
  }
  return `{% for ${itemName} in ${name} %}\n{{ ${itemName} }}\n{% endfor %}`;
}

function insertInto(editor: TemplateEditorHandle | null, name: string) {
  editor?.insert(variableSnippet(name));
}

function startResize(event: PointerEvent) {
  if (event.button !== 0) return;
  event.preventDefault();
  dragging.value = true;
  dragStartY = event.clientY;
  dragStartHeight = paneHeight.value;
  document.body.style.cursor = "row-resize";
  document.body.style.userSelect = "none";
  window.addEventListener("pointermove", onResizeMove);
  window.addEventListener("pointerup", stopResize, { once: true });
}

function onResizeMove(event: PointerEvent) {
  paneHeight.value = clampPaneHeight(
    dragStartHeight + event.clientY - dragStartY,
  );
}

function stopResize() {
  dragging.value = false;
  document.body.style.cursor = "";
  document.body.style.userSelect = "";
  window.removeEventListener("pointermove", onResizeMove);
}

function schedulePreview() {
  window.clearTimeout(timer);
  timer = window.setTimeout(refreshPreview, 300);
}

async function refreshPreview() {
  if (!props.pluginId) {
    preview.value = null;
    previewFailure.value = "";
    return;
  }
  const request = ++previewRequest;
  previewFailure.value = "";
  try {
    const response = await agency.previewPluginTemplate(
      props.pluginId,
      props.scope ?? "org",
      props.targetId ?? "",
      props.fieldKey,
      props.modelValue.trim() ? props.modelValue : props.defaultValue,
      {
        ...pluginTemplateSamples(variables.value),
        ...(props.sampleOverrides ?? {}),
      },
    );
    if (request !== previewRequest) return;
    preview.value = response;
  } catch (error) {
    if (request !== previewRequest) return;
    preview.value = null;
    previewFailure.value =
      error instanceof Error ? error.message : String(error);
  }
}

onBeforeUnmount(() => {
  window.clearTimeout(timer);
  window.removeEventListener("pointermove", onResizeMove);
  previewRequest += 1;
});

watch(
  () => [
    props.modelValue,
    props.defaultValue,
    props.pluginId,
    props.scope,
    props.targetId,
    props.fieldKey,
    JSON.stringify(props.sampleOverrides ?? {}),
  ],
  schedulePreview,
  { immediate: true },
);
</script>

<i18n lang="yaml">
en:
  pluginTemplate:
    resetToDefault: "Reset to default"
    expand: "Expand"
    editor: "Editor"
    editorAria: "{title} template"
    preview: "Preview"
    withSampleData: "with sample data"
    variablesUsed: "{count} variable | {count} variables"
    largerEditor: "A larger editor for the same field. Changes apply immediately."
    usingDefault: "Using manifest default"
    explicitEmpty: "Explicit empty override"
    validJinja: "Valid Jinja"
    resizeHint: "Drag to resize both panes"
ru:
  pluginTemplate:
    resetToDefault: "Сбросить к значению по умолчанию"
    expand: "Развернуть"
    editor: "Редактор"
    editorAria: "Шаблон {title}"
    preview: "Просмотр"
    withSampleData: "с примером данных"
    variablesUsed: "{count} переменная | {count} переменные | {count} переменных"
    largerEditor: "Большой редактор того же поля. Изменения применяются сразу."
    usingDefault: "Используется значение манифеста"
    explicitEmpty: "Явно заданное пустое значение"
    validJinja: "Корректный Jinja"
    resizeHint: "Потяните, чтобы изменить высоту обеих панелей"
</i18n>
