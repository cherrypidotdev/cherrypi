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
  SelectTrigger.w-full(:id="id" :aria-label="label" :title="hint || undefined")
    SelectValue(:placeholder="$t('pluginThinkingSelect.placeholder')")
      | {{ displayLabel }}
  SelectContent
    SelectItem(value="__default__") {{ defaultLabel }}
    SelectItem(v-for="level in levels" :key="level" :value="level")
      | {{ labelFor(level) }}
</template>

<script setup lang="ts">
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { thinkingLevelLabel } from "@/lib/model";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    modelValue: string;

    levels: string[];

    labels?: Record<string, string>;

    defaultLabel: string;
    label?: string;
    hint?: string;
    id?: string;
    disabled?: boolean;
  }>(),
  {
    labels: () => ({}),
    label: "",
    hint: "",
    id: undefined,
    disabled: false,
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

const DEFAULT_ITEM = "__default__";

const displayLabel = computed(() =>
  props.modelValue ? labelFor(props.modelValue) : props.defaultLabel,
);

function labelFor(level: string): string {
  return props.labels[level] || thinkingLevelLabel(level);
}

function onUpdate(value: unknown): void {
  emit(
    "update:modelValue",
    value === DEFAULT_ITEM || !value ? "" : String(value),
  );
}
</script>

<i18n lang="yaml">
en:
  pluginThinkingSelect:
    placeholder: "Select a thinking level"
ru:
  pluginThinkingSelect:
    placeholder: "Выберите уровень рассуждения"
</i18n>
