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
.flex.flex-col.gap-3
  p.text-muted-foreground.text-xs(v-if="!args.length")
    | This template takes no arguments.
  Field(v-for="arg in args" :key="arg.key")
    FieldLabel(:for="`template-arg-${arg.key}`")
      | {{ arg.label || arg.key }}
      span.text-destructive(v-if="arg.required" title="Required" class="ml-0.5") *
    Textarea(
      v-if="arg.type === TaskTemplateArgType.MULTILINE"
      :placeholder="arg.defaultValue || arg.description || arg.key"
      rows="3"
      :id="`template-arg-${arg.key}`"
      :model-value="stringValue(arg.key)"
      @update:model-value="setValue(arg.key, $event)"
    )
    Select(
      v-else-if="arg.type === TaskTemplateArgType.SELECT"
      :model-value="stringValue(arg.key)"
      @update:model-value="setValue(arg.key, $event)"
    )
      SelectTrigger.w-full(:id="`template-arg-${arg.key}`")
        SelectValue(:placeholder="arg.description || 'Choose an option'")
      SelectContent
        SelectItem(
          v-for="option in arg.options"
          :key="option.value"
          :value="option.value"
        ) {{ option.label || option.value }}
    label.flex.items-center.gap-2(
      v-else-if="arg.type === TaskTemplateArgType.BOOLEAN"
    )
      Switch(
        :id="`template-arg-${arg.key}`"
        :model-value="booleanValue(arg.key)"
        @update:model-value="setValue(arg.key, $event)"
      )
      span.text-sm {{ booleanValue(arg.key) ? "Yes" : "No" }}
    Input(
      v-else
      :type="arg.type === TaskTemplateArgType.NUMBER ? 'number' : 'text'"
      :placeholder="arg.defaultValue || arg.description || arg.key"
      :id="`template-arg-${arg.key}`"
      :model-value="stringValue(arg.key)"
      @update:model-value="setValue(arg.key, $event)"
    )
    FieldDescription(v-if="arg.description") {{ arg.description }}
</template>

<script setup lang="ts">
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { TaskTemplateArgType, type TaskTemplateArgInput } from "@cherrypi/sdk";

const props = defineProps<{
  args: TaskTemplateArgInput[];
  modelValue: Record<string, unknown>;
}>();
const emit = defineEmits<{ "update:modelValue": [Record<string, unknown>] }>();

function stringValue(key: string): string {
  const value = props.modelValue[key];
  if (value === undefined || value === null) return "";
  return String(value);
}

function booleanValue(key: string): boolean {
  const value = props.modelValue[key];
  if (typeof value === "boolean") return value;
  if (typeof value === "string") {
    return ["true", "1", "yes", "on"].includes(value.toLowerCase());
  }
  return Boolean(value);
}

function setValue(key: string, value: unknown) {
  emit("update:modelValue", { ...props.modelValue, [key]: value });
}
</script>
