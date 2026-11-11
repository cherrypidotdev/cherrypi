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
  DialogContent.overflow-y-auto(class="max-h-[90vh] sm:max-w-2xl")
    DialogHeader
      DialogTitle {{ $t("pluginSamples.title") }}
      DialogDescription
        | {{ $t("pluginSamples.description") }}
    .flex.flex-col.gap-4
      .flex.flex-col.gap-1(v-for="name in names" :key="name")
        .flex.flex-wrap.items-center.gap-2
          code.font-mono.text-xs.text-violet-700 {{ variableLabel(name) }}
          Badge(v-if="variables[name]?.type" variant="outline") {{ variables[name]?.type }}
        p.text-muted-foreground.text-xs(v-if="variables[name]?.description") {{ variables[name]?.description }}
        Switch(
          v-if="variables[name]?.type === 'boolean'"
          :model-value="Boolean(effective(name))"
          @update:model-value="setValue(name, $event)"
        )
        Input(
          v-else-if="variables[name]?.type === 'number' || variables[name]?.type === 'integer'"
          type="number"
          :model-value="String(effective(name) ?? '')"
          @update:model-value="setValue(name, $event === '' ? 0 : Number($event))"
        )
        Textarea.font-mono.text-xs(
          v-else-if="variables[name]?.type === 'array' || variables[name]?.type === 'object'"
          rows="4"
          :model-value="jsonDraft(name)"
          @update:model-value="setJSON(name, String($event))"
        )
        Input(
          v-else
          :model-value="String(effective(name) ?? '')"
          @update:model-value="setValue(name, $event)"
        )
        p.text-destructive.text-xs(v-if="jsonErrors[name]") {{ jsonErrors[name] }}
    DialogFooter
      Button(type="button" variant="ghost" @click="reset")
        | {{ $t("pluginSamples.reset") }}
      Button(
        type="button"
        variant="outline"
        @click="emit('update:open', false)"
      ) {{ $t("common.done") }}
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
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import type { PluginTemplateVariable } from "@/lib/plugins";
import { computed, reactive, watch } from "vue";

const props = defineProps<{
  open: boolean;
  variables: Record<string, PluginTemplateVariable>;
  modelValue: Record<string, unknown>;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  "update:modelValue": [value: Record<string, unknown>];
}>();

const jsonDrafts = reactive<Record<string, string>>({});
const jsonErrors = reactive<Record<string, string>>({});
const names = computed(() => Object.keys(props.variables).sort());

function variableLabel(name: string): string {
  return `{{ ${name} }}`;
}

function zero(variable: PluginTemplateVariable | undefined): unknown {
  switch (variable?.type) {
    case "array":
      return [];
    case "object":
      return {};
    case "number":
    case "integer":
      return 0;
    case "boolean":
      return false;
    default:
      return "";
  }
}

function effective(name: string): unknown {
  if (Object.prototype.hasOwnProperty.call(props.modelValue, name)) {
    return props.modelValue[name];
  }
  const variable = props.variables[name];
  return variable?.sample !== undefined ? variable.sample : zero(variable);
}

function jsonDraft(name: string): string {
  return jsonDrafts[name] ?? JSON.stringify(effective(name) ?? null, null, 2);
}

function setValue(name: string, value: unknown) {
  emit("update:modelValue", { ...props.modelValue, [name]: value });
}

function setJSON(name: string, raw: string) {
  jsonDrafts[name] = raw;
  try {
    setValue(name, JSON.parse(raw || "null"));
    delete jsonErrors[name];
  } catch {
    jsonErrors[name] = "Must be valid JSON";
  }
}

function reset() {
  emit("update:modelValue", {});
  for (const name of Object.keys(jsonDrafts)) delete jsonDrafts[name];
  for (const name of Object.keys(jsonErrors)) delete jsonErrors[name];
}

watch(
  () => props.open,
  (open) => {
    if (!open) return;
    for (const name of Object.keys(jsonDrafts)) delete jsonDrafts[name];
    for (const name of Object.keys(jsonErrors)) delete jsonErrors[name];
    for (const name of names.value) {
      const type = props.variables[name]?.type;
      if (type === "array" || type === "object") {
        jsonDrafts[name] = JSON.stringify(effective(name) ?? null, null, 2);
      }
    }
  },
);
</script>

<i18n lang="yaml">
en:
  pluginSamples:
    title: "Sample data"
    description: "The values the preview renders with. They are editor-only and never stored; the plugin uses each incident's real values at runtime."
    reset: "Reset samples"
ru:
  pluginSamples:
    title: "Пример данных"
    description: "Значения, с которыми формируется предпросмотр. Они нужны только редактору и не сохраняются; во время работы плагин использует реальные значения каждого случая."
    reset: "Сбросить примеры"
</i18n>
