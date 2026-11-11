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
.bg-card.overflow-hidden.rounded-lg.border
  .flex.items-start.gap-3.p-3
    .min-w-0.flex-1
      .flex.flex-wrap.items-center(class="gap-1.5")
        span.text-sm.font-medium {{ title }}
        Badge(variant="secondary") Jinja
        Badge.text-amber-700(
          v-if="customized"
          variant="outline"
          class="border-amber-500/40 bg-amber-500/10 dark:text-amber-300"
        ) {{ $t("pluginTemplate.customized") }}
      p.text-muted-foreground.mt-1.text-xs(v-if="description") {{ description }}
    .flex.items-center.gap-1
      Button(
        v-if="customized"
        type="button"
        variant="ghost"
        size="sm"
        @click="resetToDefault"
      ) {{ $t("pluginTemplate.resetToDefault") }}
      Button(type="button" variant="ghost" size="sm" @click="expanded = true")
        MaximizeIcon(data-icon="inline-start")
        span {{ $t("pluginTemplate.expand") }}
  .flex.flex-wrap.items-center.px-3.pb-2(class="gap-1.5")
    span.text-muted-foreground.text-xs.font-semibold.tracking-wide.uppercase
      | {{ $t("pluginTemplate.insert") }}
    Button.h-6.rounded.px-2.font-mono.text-xs(
      v-for="name in quickVariables"
      :key="name"
      type="button"
      variant="outline"
      size="sm"
      :title="variables[name]?.description"
      @click="insertVariable(name)"
    ) {{ name }}
    details.relative.ml-auto
      summary.text-muted-foreground.cursor-pointer.list-none.rounded.px-2.py-1.text-xs(
        class="hover:bg-muted"
      ) {{ $t("pluginTemplate.allVariables") }}
      .bg-popover.absolute.right-0.z-30.mt-1.max-h-72.w-80.overflow-auto.rounded-md.border.p-1.shadow-md
        template(v-for="name in variableNames" :key="name")
          button.w-full.rounded.px-2.py-1.text-left.text-xs(
            type="button"
            class="hover:bg-muted"
            @click="insertVariable(name)"
          )
            code.font-mono.text-xs.text-violet-700 {{ variableLabel(name) }}
            span.text-muted-foreground.block.text-xs {{ variables[name]?.description }}
  .grid.grid-cols-1.border-t(class="md:grid-cols-2")
    .min-w-0
      .text-muted-foreground.flex.h-8.items-center.justify-between.border-b.px-2.text-xs.font-medium
        span {{ $t("pluginTemplate.editor") }}
        span.font-mono Jinja
      textarea.min-h-40.w-full.resize-y.p-2.font-mono.text-xs.leading-relaxed.outline-none(
        ref="editor"
        :value="modelValue"
        spellcheck="false"
        :aria-label="$t('pluginTemplate.editorAria', { title })"
        :aria-invalid="hasErrors"
        @input="onInput"
      )
    .min-w-0(class="md:border-l")
      .text-muted-foreground.flex.h-8.items-center.justify-between.border-b.px-2.text-xs.font-medium
        span {{ $t("pluginTemplate.preview") }}
        span {{ $t("pluginTemplate.withSampleData") }}
      pre.max-h-64.min-h-40.overflow-auto.p-2.text-xs.leading-relaxed.break-words.whitespace-pre-wrap(
        :class="hasErrors ? 'bg-destructive/5 text-destructive' : ''"
      ) {{ previewText }}
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
    .grid.gap-3(class="md:grid-cols-2")
      .flex.flex-col.gap-1
        span.text-xs.font-medium {{ $t("pluginTemplate.editor") }}
        textarea.min-h-96.w-full.resize-y.rounded-md.border.p-2.font-mono.text-xs.leading-relaxed(
          :value="modelValue"
          spellcheck="false"
          @input="onInput"
        )
      .flex.flex-col.gap-1
        span.text-xs.font-medium {{ $t("pluginTemplate.preview") }}
        pre.max-h-96.min-h-96.overflow-auto.rounded-md.border.p-2.text-xs.leading-relaxed.break-words.whitespace-pre-wrap(
          class="bg-muted/20"
          :class="hasErrors ? 'text-destructive' : ''"
        ) {{ previewText }}
    p.text-destructive.text-xs(v-if="props.error") {{ props.error }}
    DialogFooter
      Button(type="button" variant="outline" @click="expanded = false")
        | {{ $t("common.close") }}
</template>

<script setup lang="ts">
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
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const editor = ref<HTMLTextAreaElement | null>(null);
const expanded = ref(false);
const preview = ref<PreviewPluginTemplateResponse | null>(null);
const previewFailure = ref("");
let timer: number | undefined;
let previewRequest = 0;

const variables = computed<Record<string, PluginTemplateVariable>>(
  () => props.template?.variables ?? {},
);
const variableNames = computed(() => Object.keys(variables.value));
const quickVariables = computed(() => variableNames.value.slice(0, 6));
const matchesDefault = computed(() => props.modelValue === props.defaultValue);
const explicitEmpty = computed(
  () => !matchesDefault.value && props.modelValue.trim() === "",
);
const statusLabel = computed(() => {
  if (matchesDefault.value) return t("pluginTemplate.usingDefault");
  if (explicitEmpty.value) return t("pluginTemplate.explicitEmpty");
  return t("pluginTemplate.validJinja");
});
const customized = computed(() => !matchesDefault.value);
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

function variableLabel(name: string): string {
  return `{{ ${name} }}`;
}

function onInput(event: Event) {
  emit("update:modelValue", (event.target as HTMLTextAreaElement).value);
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

function insertVariable(name: string) {
  const element = editor.value;
  if (!element) return;
  const snippet = variableSnippet(name);
  const start = element.selectionStart ?? element.value.length;
  const end = element.selectionEnd ?? start;
  const next =
    element.value.slice(0, start) + snippet + element.value.slice(end);
  emit("update:modelValue", next);
  requestAnimationFrame(() => {
    element.focus();
    element.selectionStart = element.selectionEnd = start + snippet.length;
  });
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
    customized: "Customized"
    resetToDefault: "Reset to default"
    expand: "Expand"
    allVariables: "All variables"
    editor: "Editor"
    editorAria: "{title} template"
    preview: "Preview"
    insert: "Insert"
    withSampleData: "with sample data"
    variablesUsed: "{count} variable | {count} variables"
    largerEditor: "A larger editor for the same field. Changes apply immediately."
    usingDefault: "Using manifest default"
    explicitEmpty: "Explicit empty override"
    validJinja: "Valid Jinja"
ru:
  pluginTemplate:
    customized: "Изменено"
    resetToDefault: "Сбросить к значению по умолчанию"
    expand: "Развернуть"
    allVariables: "Все переменные"
    editor: "Редактор"
    editorAria: "Шаблон {title}"
    preview: "Просмотр"
    insert: "Вставить"
    withSampleData: "с примером данных"
    variablesUsed: "{count} переменная | {count} переменные | {count} переменных"
    largerEditor: "Большой редактор того же поля. Изменения применяются сразу."
    usingDefault: "Используется значение манифеста"
    explicitEmpty: "Явно заданное пустое значение"
    validJinja: "Корректный Jinja"
</i18n>
