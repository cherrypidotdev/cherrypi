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
Select(
  :model-value="modelValue"
  @update:model-value="onUpdate"
  :disabled="disabled"
)
  SelectTrigger.w-full(:aria-label="$t('pluginModelSelect.label')")
    SelectValue(:placeholder="placeholder || $t('pluginModelSelect.empty')")
      ModelName(v-if="selectedModel" :model="selectedModel")
  SelectContent
    SelectItem(value="__none__") {{ emptyLabel }}
    SelectItem(
      v-for="model in options"
      :key="keyOf(model)"
      :value="keyOf(model)"
    )
      ModelName(:model="model")
</template>

<script setup lang="ts">
import ModelName from "@/components/ModelName.vue";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { RoutingModelOption } from "@cherrypi/sdk";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    modelValue: string;
    models: RoutingModelOption[];
    placeholder?: string;
    emptyLabel?: string;
    disabled?: boolean;
  }>(),
  {
    placeholder: "",
    emptyLabel: "",
    disabled: false,
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

const options = computed<RoutingModelOption[]>(() => {
  const list = [...props.models];
  if (
    props.modelValue &&
    !list.some((model) => keyOf(model) === props.modelValue)
  ) {
    const slash = props.modelValue.indexOf("/");
    list.unshift({
      provider: slash > 0 ? props.modelValue.slice(0, slash) : props.modelValue,
      modelId: slash > 0 ? props.modelValue.slice(slash + 1) : "",
      name: props.modelValue,
      thinkingLevels: [],
    });
  }
  return list;
});

const selectedModel = computed(() =>
  options.value.find((entry) => keyOf(entry) === props.modelValue),
);

function keyOf(model: RoutingModelOption): string {
  return `${model.provider}/${model.modelId}`;
}

function onUpdate(value: unknown): void {
  if (value === "__none__") {
    emit("update:modelValue", "");
    return;
  }
  emit("update:modelValue", typeof value === "string" ? value : "");
}
</script>

<i18n lang="yaml">
en:
  pluginModelSelect:
    label: "Model"
    empty: "Select an agency model"
    useRouting: "Use model routing"
    inherit: "Inherit organization plugin model"
ru:
  pluginModelSelect:
    label: "Модель"
    empty: "Выберите модель агентства"
    useRouting: "Использовать маршрутизацию"
    inherit: "Наследовать модель плагина организации"
</i18n>
