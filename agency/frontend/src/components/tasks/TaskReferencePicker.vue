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
  DropdownMenu(v-model:open="menuOpen")
    DropdownMenuTrigger(as-child)
      Button(
        type="button"
        variant="outline"
        size="sm"
        :title="triggerTitle"
        :disabled="disabled"
      )
        ListTreeIcon(data-icon="inline-start")
        span {{ $t("domainLog.kind.references") }}
        span.text-muted-foreground(v-if="selected.length") {{ selected.length }}
    DropdownMenuContent.w-96(align="start" @select.prevent)
      .p-2
        DropdownMenuFilter(
          v-model="search"
          :placeholder="$t('taskParent.searchPlaceholder')"
          auto-focus
          :aria-label="$t('taskRefs.searchAria')"
        )
      TaskPicker(
        embedded
        :search="search"
        :reference-project-id="projectId"
        :exclude-ids="excludeIds"
        @pick="add"
      )

  span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-2.text-xs(
    v-for="entry in selected"
    :key="entry.task.id"
    class="py-0.5"
    :class="taskReferenceKindClass(entry.kind)"
    :title="taskReferenceTitle(entry.task)"
  )
    component.size-3.shrink-0(
      :class="taskReferenceKindTextClass(entry.kind)"
      :is="taskReferenceKindIcon(entry.kind)"
    )
    strong.font-semibold {{ entry.task.id }}
    span.max-w-40.truncate(:class="taskNameClass(entry.task)") {{ taskReferenceName(entry.task) }}
    DropdownMenu
      DropdownMenuTrigger(as-child)
        button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
          type="button"
          class="p-0.5 hover:opacity-100"
          :title="$t('taskRefs.kindLabel')"
        )
          ChevronDownIcon.size-3
      DropdownMenuContent.w-40(align="end")
        DropdownMenuRadioGroup(
          :model-value="entry.kind"
          @update:model-value="setKind(entry, $event)"
        )
          DropdownMenuRadioItem(
            v-for="option in TASK_REFERENCE_KINDS"
            :key="option.value"
            :value="option.value"
          ) {{ $t(option.labelKey) }}
    button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
      type="button"
      class="p-0.5 hover:opacity-100"
      :title="$t('taskRefs.remove')"
      @click="remove(entry)"
    )
      XIcon.size-3
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuFilter,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  TASK_REFERENCE_KINDS,
  taskNameClass,
  taskReferenceKindClass,
  taskReferenceKindIcon,
  taskReferenceKindTextClass,
  taskReferenceName,
  taskReferenceTitle,
} from "@/lib/task";
import {
  TaskReferenceKind,
  type Task,
  type TaskReferenceInput,
} from "@cherrypi/sdk";
import { ChevronDownIcon, ListTreeIcon, XIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import TaskPicker from "./TaskPicker.vue";

const { t } = useI18n();

interface SelectedReference {
  task: Task;
  kind: TaskReferenceKind;
}

const props = withDefaults(
  defineProps<{
    modelValue: TaskReferenceInput[];

    projectId?: string;

    disabled?: boolean;

    knownTasks?: Task[];
  }>(),
  { projectId: "", disabled: false, knownTasks: () => [] },
);
const emit = defineEmits<{ "update:modelValue": [TaskReferenceInput[]] }>();

const selected = ref<SelectedReference[]>([]);

const search = ref("");
const menuOpen = ref(false);

const triggerTitle = computed(() =>
  props.projectId
    ? t("taskRefs.triggerTitle")
    : t("taskRefs.selectProjectFirst"),
);

const excludeIds = computed(() => selected.value.map((entry) => entry.task.id));

function emitValue() {
  emit(
    "update:modelValue",
    selected.value.map((entry) => ({
      taskId: entry.task.id,
      kind: entry.kind,
    })),
  );
}

function add(candidate: Task) {
  if (selected.value.some((entry) => entry.task.id === candidate.id)) return;
  selected.value = [
    ...selected.value,
    { task: candidate, kind: TaskReferenceKind.INFORMATIONAL },
  ];
  search.value = "";
  menuOpen.value = false;
  emitValue();
}

function remove(entry: SelectedReference) {
  selected.value = selected.value.filter(
    (item) => item.task.id !== entry.task.id,
  );
  emitValue();
}

function setKind(entry: SelectedReference, kind: unknown) {
  entry.kind = kind as TaskReferenceKind;
  emitValue();
}

watch(
  [() => props.modelValue, () => props.knownTasks],
  ([refs, known]) => {
    const knownById = new Map(known.map((task) => [task.id, task]));
    const next = selected.value.filter((entry) =>
      refs.some((ref) => ref.taskId === entry.task.id),
    );
    for (const entry of next) {
      const ref = refs.find((item) => item.taskId === entry.task.id);
      if (ref?.kind !== undefined) entry.kind = ref.kind;
    }
    for (const ref of refs) {
      if (next.some((entry) => entry.task.id === ref.taskId)) continue;
      const task = knownById.get(ref.taskId);
      if (!task) continue;
      next.push({
        task,
        kind: ref.kind ?? TaskReferenceKind.INFORMATIONAL,
      });
    }
    selected.value = next;
  },
  { deep: true },
);
</script>

<i18n lang="yaml">
en:
  taskRefs:
    searchAria: "Search tasks to reference"
    selectProjectFirst: "Select a project first"
    kindLabel: "Reference kind"
    triggerTitle: "References"
ru:
  taskRefs:
    searchAria: "Поиск задач для ссылки"
    selectProjectFirst: "Сначала выберите проект"
    kindLabel: "Тип ссылки"
    triggerTitle: "Связи"
</i18n>
