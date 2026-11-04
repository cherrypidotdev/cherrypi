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
  DialogContent.overflow-y-auto(class="max-h-[90vh] sm:max-w-md")
    DialogHeader
      DialogTitle {{ tag ? "Edit tag" : "Add tag" }}
      DialogDescription(v-if="tag")
        | Rename the tag or adjust its colour.
      DialogDescription(v-else)
        | Create a label for this project's tasks.
    form.flex.flex-col.gap-4(@submit.prevent="save")
      Field
        FieldLabel(for="tag-editor-name") Name
        Input#tag-editor-name(
          v-model="name"
          placeholder="bug"
          autocomplete="off"
          data-test="tag-name"
        )

      Field
        FieldLabel(for="tag-editor-description") Description
        Input#tag-editor-description(
          v-model="description"
          placeholder="Optional"
          data-test="tag-description"
        )

      Field
        FieldLabel Colour
        TagColorPicker(v-model="color" :disabled="saving")
        FieldDescription Pick a palette colour or enter a custom hex value.

      Field(v-if="!tag && presets.length")
        FieldLabel Suggested tags
        .flex.flex-wrap(class="gap-1.5")
          button.rounded-full.border.px-2.text-xs.transition-opacity(
            v-for="preset in presets"
            :key="preset.name"
            type="button"
            class="py-0.5 hover:opacity-80"
            :style="tagStyle(preset.color)"
            :title="preset.description"
            @click="applyPreset(preset)"
            data-test="tag-preset"
          )
            | {{ preset.name }}
        FieldDescription
          | One-click presets; nothing is created until you save.

      p.text-destructive.text-sm(v-if="error") {{ error }}

      DialogFooter
        DialogClose
          Button(type="button" variant="ghost") Cancel
        Button(type="submit" :disabled="saving || !name.trim() || !projectId")
          Spinner(v-if="saving" data-icon="inline-start")
          CheckIcon(v-else data-icon="inline-start")
          span {{ tag ? "Save" : "Add tag" }}
</template>

<script setup lang="ts">
import TagColorPicker from "@/components/tags/TagColorPicker.vue";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { withProgress } from "@/composables/useProgress";
import { useTags } from "@/composables/useTags";
import { suggestTagColor, TAG_PRESETS, type TagPreset } from "@/lib/tag";
import { tagStyle } from "@/lib/task";
import type { Tag } from "@cherrypi/sdk";
import { CheckIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{
  open: boolean;
  projectId: string;
  tag?: Tag | null;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  saved: [tag: Tag];
}>();

const { active, createTag, updateTag } = useTags(() => props.projectId);

const name = ref("");
const description = ref("");
const color = ref("");
const error = ref("");
const saving = ref(false);

const presets = computed(() => {
  const taken = new Set(
    active.value.map((tag) => tag.name.trim().toLowerCase()),
  );
  return TAG_PRESETS.filter((preset) => !taken.has(preset.name));
});

function reset() {
  const tag = props.tag;
  name.value = tag?.name ?? "";
  description.value = tag?.description ?? "";
  color.value = tag?.color || suggestTagColor(active.value);
  error.value = "";
}

watch(
  () => [props.open, props.tag?.id] as const,
  () => {
    if (props.open) reset();
  },
  { immediate: true },
);

function applyPreset(preset: TagPreset) {
  name.value = preset.name;
  description.value = preset.description;
  color.value = preset.color;
}

async function save() {
  if (saving.value || !name.value.trim() || !props.projectId) return;
  saving.value = true;
  error.value = "";
  try {
    const tag = props.tag;
    const saved = tag
      ? await withProgress(() =>
          updateTag(tag.id, {
            name: name.value.trim(),
            description: description.value.trim(),
            color: color.value,
            isDefault: tag.isDefault,
          }),
        )
      : await withProgress(() =>
          createTag({
            name: name.value.trim(),
            description: description.value.trim(),
            color: color.value,
          }),
        );
    toast.success(tag ? "Tag updated" : "Tag created");
    emit("saved", saved);
    emit("update:open", false);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}
</script>
