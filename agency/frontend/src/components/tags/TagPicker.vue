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
.flex.flex-wrap.items-center(class="gap-1.5")
  DropdownMenu
    DropdownMenuTrigger(as-child)
      Button(
        type="button"
        variant="outline"
        size="sm"
        :title="triggerTitle"
        :disabled="disabled || !projectId"
      )
        TagIcon(data-icon="inline-start")
        span {{ $t("domainLog.kind.tags") }}
    DropdownMenuContent.w-56(align="start")
      DropdownMenuLabel {{ $t("domainLog.kind.tags") }}
      DropdownMenuSeparator
      p.text-muted-foreground.px-2.py-1.text-xs(v-if="!tags.length")
        | {{ $t("tagPicker.empty") }}
      DropdownMenuCheckboxItem(
        v-for="tag in tags"
        :key="tag.id"
        :model-value="selectedIds.has(tag.id)"
        @update:model-value="toggle(tag.id)"
        :disabled="disabled"
      )
        span.flex.items-center.gap-2
          span.size-2.shrink-0.rounded-full(
            :style="{ backgroundColor: tagColor(tag.color) }"
          )
          span {{ tag.name }}
          LockIcon.size-3.shrink-0.text-amber-500(
            v-if="tag.isRequired"
            :title="$t('tasksTable.requiredTagHint')"
          )
      DropdownMenuSeparator
      DropdownMenuItem(@click="openEditor" :disabled="disabled || !projectId")
        PlusIcon
        span {{ $t("tags.add") }}

  span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-2.text-xs(
    v-for="tag in selectedTags"
    :key="tag.id"
    class="py-0.5"
    :style="tagStyle(tag.color)"
  )
    span.rounded-full.bg-current(class="size-1.5")
    span {{ tag.name }}
    LockIcon.size-3.shrink-0.text-amber-500(
      v-if="tag.isRequired"
      :title="$t('tasksTable.requiredTagHint')"
    )
    button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
      type="button"
      class="p-0.5 hover:opacity-100"
      :title="$t('taskDetails.removeTag')"
      @click="toggle(tag.id)"
      :disabled="disabled"
    )
      XIcon.size-3

TagEditorDialog(
  v-model:open="editorOpen"
  :project-id="projectId"
  @saved="onSaved"
)
</template>

<script setup lang="ts">
import TagEditorDialog from "@/components/tags/TagEditorDialog.vue";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { tagColor, tagStyle } from "@/lib/task";
import type { Tag } from "@cherrypi/sdk";
import { LockIcon, PlusIcon, TagIcon, XIcon } from "@lucide/vue";
import { computed, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  projectId: string;
  tags: readonly Tag[];
  modelValue: readonly string[];
  disabled?: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string[]];
  created: [tag: Tag];
}>();

const editorOpen = ref(false);

const selectedIds = computed(() => new Set(props.modelValue));
const selectedTags = computed(() =>
  props.tags.filter((tag) => selectedIds.value.has(tag.id)),
);
const triggerTitle = computed(() =>
  props.projectId ? "Tags" : t("express.selectProjectFirst"),
);

function toggle(id: string) {
  const next = new Set(props.modelValue);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  emit("update:modelValue", [...next]);
}

function openEditor() {
  void nextTick(() => {
    editorOpen.value = true;
  });
}

function onSaved(tag: Tag) {
  emit("created", tag);
  if (!selectedIds.value.has(tag.id)) {
    emit("update:modelValue", [...props.modelValue, tag.id]);
  }
}

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  tagPicker:
    empty: "No tags yet"
ru:
  tagPicker:
    empty: "Тегов пока нет"
</i18n>
