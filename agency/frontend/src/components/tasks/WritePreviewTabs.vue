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
.bg-muted.inline-flex.w-fit.items-center.rounded-lg.border(class="p-0.5")
  button.rounded-md.px-2.text-xs.font-medium.transition-colors(
    v-for="option in TABS"
    :key="option.value"
    type="button"
    class="py-0.5"
    :class="props.modelValue === option.value ? 'bg-card text-foreground shadow-sm ring-1 ring-border' : 'text-muted-foreground hover:text-foreground'"
    :aria-pressed="props.modelValue === option.value"
    @click="emit('update:modelValue', option.value)"
    :data-testid="testIdFor(option.value)"
  ) {{ option.label }}
</template>

<script setup lang="ts">
// The Markdown editor's Write/Preview switch. Shared by the draft prompt
// editor, the task console composer and the quick-create form so all three
// read as the same control.
type MarkdownTab = "write" | "preview";

const props = defineProps<{
  modelValue: MarkdownTab;
  /** Prefix for the two buttons' `data-testid`s (`<prefix>-write`/`-preview`). */
  testId?: string;
}>();

const emit = defineEmits<{ "update:modelValue": [value: MarkdownTab] }>();

const TABS = [
  { value: "write", label: "Write" },
  { value: "preview", label: "Preview" },
] as const;

function testIdFor(value: MarkdownTab): string | undefined {
  return props.testId ? `${props.testId}-${value}` : undefined;
}
</script>
