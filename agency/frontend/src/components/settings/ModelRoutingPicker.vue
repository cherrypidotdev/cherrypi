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
.flex.flex-wrap.items-center.gap-2
  Select(
    :model-value="modelValue.provider && modelValue.modelId ? keyOf(modelValue) : ''"
    @update:model-value="onModel"
    :disabled="disabled"
  )
    SelectTrigger.flex-1(:aria-label="$t('modelRouting.model')")
      SelectValue(:placeholder="placeholder")
        ModelName(v-if="selectedModel" :model="selectedModel")
    SelectContent
      SelectItem(v-if="allowInherit" value="__inherit__") {{ inheritLabel }}
      SelectItem(
        v-for="model in options"
        :key="keyOf(model)"
        :value="keyOf(model)"
      )
        ModelName(:model="model")
  Select(
    :model-value="modelValue.thinkingLevel || ''"
    @update:model-value="onThinking"
    :disabled="disabled || !modelValue.provider || !modelValue.modelId || !thinkingLevels.length"
  )
    SelectTrigger.w-28(:aria-label="$t('modelRouting.thinking')")
      SelectValue(:placeholder="$t('modelRouting.thinking')")
        | {{ thinkingLabel }}
    SelectContent
      SelectItem(value="__none__") {{ $t("modelRouting.modelDefault") }}
      SelectItem(v-for="level in thinkingLevels" :key="level" :value="level")
        | {{ level }}
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
import { sortThinkingLevels } from "@/lib/model";
import type { ModelSelectionInput, RoutingModelOption } from "@cherrypi/sdk";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    modelValue: ModelSelectionInput;
    models: RoutingModelOption[];
    allowInherit?: boolean;
    inheritLabel?: string;
    placeholder?: string;
    disabled?: boolean;
  }>(),
  {
    allowInherit: false,
    inheritLabel: "",
    placeholder: "",
    disabled: false,
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: ModelSelectionInput): void;
}>();

const options = computed<RoutingModelOption[]>(() => {
  const list = [...props.models];
  const selected = props.modelValue;
  if (
    selected.provider &&
    selected.modelId &&
    !list.some(
      (model) =>
        model.provider === selected.provider &&
        model.modelId === selected.modelId,
    )
  ) {
    list.unshift({
      provider: selected.provider,
      modelId: selected.modelId,
      name: selected.modelName || selected.modelId,
      thinkingLevels: [],
    });
  }
  return list;
});

const selectedModel = computed<RoutingModelOption | undefined>(() => {
  const selected = props.modelValue;
  if (!selected.provider || !selected.modelId) return undefined;
  return options.value.find(
    (model) =>
      model.provider === selected.provider &&
      model.modelId === selected.modelId,
  );
});

const thinkingLevels = computed(() =>
  sortThinkingLevels(selectedModel.value?.thinkingLevels ?? []),
);

function levelFor(model: RoutingModelOption): string {
  const current = props.modelValue.thinkingLevel ?? "";
  if (!current || !model.thinkingLevels.length) return current;
  return model.thinkingLevels.includes(current) ? current : "";
}

const thinkingLabel = computed(() => {
  const level = props.modelValue.thinkingLevel;
  if (!level) return t("modelRouting.modelDefault");
  return level;
});

function keyOf(value: { provider: string; modelId: string }): string {
  return `${value.provider}/${value.modelId}`;
}

function onModel(value: unknown): void {
  const key = typeof value === "string" ? value : "";
  if (key === "__inherit__") {
    emit("update:modelValue", {
      provider: "",
      modelId: "",
      modelName: "",
      thinkingLevel: "",
    });
    return;
  }
  const model = options.value.find((entry) => keyOf(entry) === key);
  if (!model) return;
  emit("update:modelValue", {
    provider: model.provider,
    modelId: model.modelId,
    modelName: model.name ?? "",
    thinkingLevel: levelFor(model),
  });
}

function onThinking(value: unknown): void {
  emit("update:modelValue", {
    ...props.modelValue,
    thinkingLevel:
      typeof value === "string" && value !== "__none__" ? value : "",
  });
}
</script>

<i18n lang="yaml">
en:
  modelRouting:
    model: "Model"
    thinking: "Thinking"
    modelDefault: "Agent default"
ru:
  modelRouting:
    model: "Модель"
    thinking: "Рассуждение"
    modelDefault: "Рассуждение агента"
</i18n>
