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
.relative.flex.flex-col.gap-2(ref="root")
  .flex.items-center.gap-2
    ListTreeIcon.text-muted-foreground(class="size-3.5")
    span.text-muted-foreground.text-xs References
    Spinner.size-3(v-if="searching && !candidates.length")
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
  .relative
    SearchIcon.text-muted-foreground.pointer-events-none.absolute.size-4(
      class="top-1/2 left-2.5 -translate-y-1/2"
    )
    Input.w-full.pl-8(
      v-model="search"
      placeholder="Add reference…"
      autocomplete="off"
      @focus="open = true"
      @keydown.esc="open = false"
      :disabled="disabled"
    )
    .bg-popover.text-popover-foreground.absolute.z-50.mt-1.max-h-56.w-full.overflow-y-auto.rounded-md.border.p-1.shadow-md(
      v-if="open && !disabled"
    )
      p.text-muted-foreground.px-2.text-xs(v-if="searching") Searching…
      p.text-muted-foreground.px-2.text-xs(v-else-if="!candidates.length")
        | No available tasks.
      template(v-else)
        p.text-muted-foreground.px-2.text-xs(v-if="!search.trim()")
          | Recent tasks
        button.inline-flex.w-full.min-w-0.items-center.rounded-full.border.px-2.text-left.text-xs(
          v-for="candidate in candidates"
          :key="candidate.id"
          type="button"
          class="hover:bg-muted py-0.5"
          :class="taskStateClassFor(candidate)"
          :title="taskReferenceTitle(candidate)"
          @click="add(candidate)"
        )
          component.size-3.shrink-0(:is="taskStateIcon(candidate.state)")
          strong.shrink-0.font-semibold {{ candidate.id }}:
          span.truncate(:class="taskNameClass(candidate)") {{ taskReferenceName(candidate) }}
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
import { Spinner } from "@/components/ui/spinner";
import { referenceProjectIds } from "@/lib/project";
import { agency } from "@/lib/sdk";
import {
  TASK_REFERENCE_KINDS,
  taskNameClass,
  taskReferenceKindClass,
  taskReferenceKindIcon,
  taskReferenceKindTextClass,
  taskReferenceName,
  taskReferenceTitle,
  taskStateClassFor,
  taskStateIcon,
} from "@/lib/task";
import {
  TaskReferenceKind,
  type Task,
  type TaskReferenceInput,
} from "@cherrypi/sdk";
import { ListTreeIcon, SearchIcon, XIcon } from "@lucide/vue";
import { onClickOutside, refDebounced } from "@vueuse/core";
import { ref, watch } from "vue";
import { toast } from "vue3-toastify";

interface SelectedReference {
  task: Task;
  kind: TaskReferenceKind;
}

const props = withDefaults(
  defineProps<{
    modelValue: TaskReferenceInput[];

    projectId?: string;

    disabled?: boolean;
  }>(),
  { projectId: "", disabled: false },
);
const emit = defineEmits<{ "update:modelValue": [TaskReferenceInput[]] }>();

const root = ref<HTMLElement | null>(null);
const open = ref(false);
const searching = ref(false);
const search = ref("");
const debouncedSearch = refDebounced(search, 300);

const selected = ref<SelectedReference[]>([]);
const candidates = ref<Task[]>([]);

onClickOutside(root, () => (open.value = false));

function emitValue() {
  emit(
    "update:modelValue",
    selected.value.map((entry) => ({
      taskId: entry.task.id,
      kind: entry.kind,
    })),
  );
}

async function searchCandidates() {
  const query = debouncedSearch.value.trim();
  const projectId = props.projectId;
  searching.value = true;
  try {
    const picked = new Set(selected.value.map((entry) => entry.task.id));
    const projectIDs = projectId
      ? await referenceProjectIds(projectId)
      : [undefined];
    const results = await Promise.all(
      projectIDs.map((id) =>
        agency.listTasks({ search: query || undefined, projectId: id }),
      ),
    );

    if (props.projectId !== projectId || debouncedSearch.value.trim() !== query)
      return;
    const found = new Map<string, Task>();
    for (const task of results.flat()) found.set(task.id, task);
    const available = [...found.values()].filter(
      (task) => !picked.has(task.id),
    );
    candidates.value = query ? available.slice(0, 8) : available;
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    searching.value = false;
  }
}

function add(candidate: Task) {
  if (!selected.value.some((entry) => entry.task.id === candidate.id)) {
    selected.value = [
      ...selected.value,
      { task: candidate, kind: TaskReferenceKind.INFORMATIONAL },
    ];
    emitValue();
  }
  search.value = "";
  candidates.value = candidates.value.filter(
    (item) => item.id !== candidate.id,
  );
}

function remove(entry: SelectedReference) {
  selected.value = selected.value.filter(
    (item) => item.task.id !== entry.task.id,
  );
  emitValue();

  if (open.value) void searchCandidates();
}

function setKind(entry: SelectedReference, kind: unknown) {
  entry.kind = kind as TaskReferenceKind;
  emitValue();
}

watch(debouncedSearch, () => void searchCandidates());

watch(open, (isOpen) => {
  if (isOpen) void searchCandidates();
});
watch(search, (value) => {
  if (value.trim()) open.value = true;
});

watch(
  () => props.projectId,
  () => {
    search.value = "";
    candidates.value = [];
    if (open.value) void searchCandidates();
  },
);

watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) open.value = false;
  },
);

watch(
  () => props.modelValue,
  (refs) => {
    if (refs.length === 0 && selected.value.length) selected.value = [];
  },
);
</script>
