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
.flex.items-center.gap-1(data-testid="task-parent-picker")
  DropdownMenu(v-model:open="menuOpen")
    DropdownMenuTrigger(as-child)
      Button(
        type="button"
        variant="outline"
        size="sm"
        :title="parent ? $t('taskParent.subTaskOf', { task: taskReferenceTitle(parent) }) : $t('taskParent.nest')"
        :disabled="disabled"
        data-testid="composer-parent"
      )
        ListTreeIcon(data-icon="inline-start")
        component.size-3.shrink-0(
          v-if="parent"
          :is="taskStateIcon(parent.state)"
        )
        span.max-w-40.truncate(v-if="parent") {{ parent.id }} · {{ taskName(parent) }}
        span(v-else) {{ $t("taskParent.none") }}
    DropdownMenuContent.w-96(align="start" @select.prevent)
      .p-2
        DropdownMenuFilter(
          v-model="search"
          :placeholder="$t('taskParent.searchPlaceholder')"
          auto-focus
          :aria-label="$t('taskParent.searchAria')"
          data-testid="parent-picker-search"
        )

      TaskPicker(
        embedded
        :search="search"
        :reference-project-id="projectId"
        @pick="pick"
      )
      .border-border.mt-1.border-t.pt-1(v-if="parent")
        Button.w-full(
          type="button"
          variant="ghost"
          size="sm"
          :title="$t('taskParent.clear')"
          @click="clear"
          data-testid="parent-picker-clear"
        )
          XIcon(data-icon="inline-start")
          span {{ $t("taskParent.clear") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuFilter,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { taskName, taskReferenceTitle, taskStateIcon } from "@/lib/task";
import type { Task } from "@cherrypi/sdk";
import { ListTreeIcon, XIcon } from "@lucide/vue";
import { ref, watch } from "vue";
import TaskPicker from "./TaskPicker.vue";

const props = withDefaults(
  defineProps<{
    modelValue?: Task | null;

    projectId?: string;

    disabled?: boolean;
  }>(),
  { modelValue: null, projectId: "", disabled: false },
);
const emit = defineEmits<{ "update:modelValue": [Task | null] }>();

const menuOpen = ref(false);
const search = ref("");
const parent = ref<Task | null>(props.modelValue);

watch(
  () => props.modelValue,
  (value) => {
    parent.value = value;
  },
);

watch(
  () => props.projectId,
  () => {
    search.value = "";
    menuOpen.value = false;
  },
);

watch(menuOpen, (open) => {
  if (!open) search.value = "";
});

function pick(candidate: Task) {
  parent.value = candidate;
  emit("update:modelValue", candidate);
  menuOpen.value = false;
}

function clear() {
  parent.value = null;
  emit("update:modelValue", null);
  menuOpen.value = false;
}
</script>

<i18n lang="yaml">
en:
  taskParent:
    none: "No parent"
    clear: "Clear parent"
    searchPlaceholder: "Search tasks…"
    searchAria: "Search tasks for a parent"
    subTaskOf: "Sub-task of {task}"
    nest: "Nest this task under a parent"
ru:
  taskParent:
    none: "Без родителя"
    clear: "Убрать родителя"
    searchPlaceholder: "Поиск задач…"
    searchAria: "Поиск задач для родителя"
    subTaskOf: "Подзадача {task}"
    nest: "Вложить задачу в родительскую"
</i18n>
