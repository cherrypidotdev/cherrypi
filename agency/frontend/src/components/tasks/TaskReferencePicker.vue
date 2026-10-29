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
.flex.flex-col.gap-2
  .flex.items-center.gap-2
    ListTreeIcon.text-muted-foreground(class="size-3.5")
    span.text-muted-foreground.text-xs References
  p.text-muted-foreground.text-xs(v-if="!selected.length")
    | None.
  .flex.flex-col.gap-1(v-if="selected.length")
    .flex.items-center.gap-2.rounded-md.border.px-2.py-1(
      v-for="entry in selected"
      :key="entry.task.id"
      :class="taskReferenceKindClass(entry.kind)"
    )
      component.size-3.shrink-0(
        :class="taskReferenceKindTextClass(entry.kind)"
        :is="taskReferenceKindIcon(entry.kind)"
      )
      component.size-3.shrink-0(:is="taskStateIcon(entry.task.state)")
      strong.text-xs.font-semibold {{ entry.task.id }}:
      span.truncate.text-xs(:class="taskNameClass(entry.task)") {{ taskReferenceName(entry.task) }}
      Select.ml-auto.w-36(
        :model-value="entry.kind"
        @update:model-value="setKind(entry, $event)"
      )
        SelectTrigger(size="sm" title="Reference kind")
          SelectValue
        SelectContent
          SelectGroup
            SelectItem(
              v-for="option in TASK_REFERENCE_KINDS"
              :key="option.value"
              :value="option.value"
            ) {{ option.label }}
      Button.shrink-0(
        type="button"
        variant="ghost"
        size="icon-sm"
        title="Remove reference"
        @click="remove(entry)"
      )
        XIcon
  .relative(ref="root")
    .relative
      SearchIcon.text-muted-foreground.pointer-events-none.absolute.size-4(
        class="top-1/2 left-2.5 -translate-y-1/2"
      )
      Input.w-full.pl-8(
        v-model="search"
        placeholder="Add reference…"
        autocomplete="off"
        @focus="open = true"
        :disabled="disabled"
      )
    TaskPicker.absolute.top-full.right-0.left-0.z-50.mt-1(
      v-if="open && !disabled"
      :search="search"
      :reference-project-id="projectId"
      :exclude-ids="excludeIds"
      @pick="add"
    )
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  TASK_REFERENCE_KINDS,
  taskNameClass,
  taskReferenceKindClass,
  taskReferenceKindIcon,
  taskReferenceKindTextClass,
  taskReferenceName,
  taskStateIcon,
} from "@/lib/task";
import {
  TaskReferenceKind,
  type Task,
  type TaskReferenceInput,
} from "@cherrypi/sdk";
import { ListTreeIcon, SearchIcon, XIcon } from "@lucide/vue";
import { onClickOutside, useEventListener } from "@vueuse/core";
import { computed, ref, watch } from "vue";
import TaskPicker from "./TaskPicker.vue";

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
const open = ref(false);
const root = ref<HTMLElement | null>(null);

onClickOutside(root, () => (open.value = false));

useEventListener(
  window,
  "keydown",
  (event: KeyboardEvent) => {
    if (!open.value || event.key !== "Escape") return;
    event.preventDefault();
    event.stopPropagation();
    open.value = false;
  },
  { capture: true },
);

watch(search, (value) => {
  if (value.trim()) open.value = true;
});
watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) open.value = false;
  },
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
