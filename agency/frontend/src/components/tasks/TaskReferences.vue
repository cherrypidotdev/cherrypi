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
.flex.flex-col.gap-2(ref="root")
  .flex.items-center.gap-2
    span.text-muted-foreground.text-xs References
    Spinner.size-3(v-if="busy")
  p.text-muted-foreground.text-xs(v-if="!rows.length")
    | —

  .flex.flex-col.gap-1(v-if="rows.length")
    .flex.items-center.rounded-md.border.px-2.py-1(
      v-for="entry in rows"
      :key="entry.key"
      class="gap-1.5"
      :class="taskReferenceKindClass(entry.kind)"
      :title="taskReferenceKindDescription(entry.kind)"
    )
      component.size-3.shrink-0(
        :class="taskReferenceKindTextClass(entry.kind)"
        :is="taskReferenceKindIcon(entry.kind)"
      )
      component.size-3.shrink-0(:is="taskStateIcon(entry.task.state)")
      RouterLink.inline-flex.min-w-0.flex-1.items-center.gap-1(
        :to="{ name: 'task', params: { id: entry.task.id } }"
        :title="taskReferenceTitle(entry.task)"
      )
        strong.shrink-0.text-xs.font-semibold {{ entry.task.id }}:
        span.truncate.text-xs(:class="taskNameClass(entry.task)") {{ taskReferenceName(entry.task) }}
      Select.ml-auto.w-36(
        :model-value="entry.kind"
        @update:model-value="requestKind(entry, $event)"
      )
        SelectTrigger(size="sm" :title="taskReferenceKindLabel(entry.kind)")
          SelectValue
        SelectContent
          SelectGroup
            SelectItem(
              v-for="option in TASK_REFERENCE_KINDS"
              :key="option.value"
              :value="option.value"
            ) {{ option.label }}
      Button.shrink-0(
        variant="ghost"
        size="icon-sm"
        title="Remove reference"
        @click="remove(entry)"
        :disabled="busy === entry.task.id"
      )
        XIcon

  p.text-muted-foreground.text-xs(
    v-if="rows.some((entry) => entry.kind === TaskReferenceKind.BLOCKED_BY)"
  )
    | This task waits for its blocking references to be completed before it can be queued.
  p.text-muted-foreground.text-xs(
    v-if="rows.some((entry) => entry.kind === TaskReferenceKind.BLOCKS)"
  )
    | The tasks this one blocks wait for it to be completed.

  .relative
    .relative
      SearchIcon.text-muted-foreground.pointer-events-none.absolute.size-4(
        class="top-1/2 left-2.5 -translate-y-1/2"
      )
      Input.w-full.pl-8(
        v-model="search"
        placeholder="Add reference…"
        autocomplete="off"
        @focus="open = true"
      )
    TaskPicker.absolute.top-full.right-0.left-0.z-50.mt-1(
      v-if="open"
      :search="search"
      :reference-project-id="task.projectId"
      :exclude-ids="excludeIds"
      @pick="addCandidate"
    )

  Dialog(:open="confirmOpen" @update:open="confirmOpen = $event")
    DialogContent(class="sm:max-w-md")
      DialogHeader
        DialogTitle.flex.items-center.gap-2
          TriangleAlertIcon.size-4.text-amber-500
          span Add blocking reference?
        DialogDescription
          | "{{ pending ? taskReferenceTitle(pending.task) : "" }}" must be completed
          | before this task can be queued again.
      p.text-muted-foreground.text-sm
        | This task is currently {{ taskStateLabel(task.state) }}. It is stopped
        | and parked as Waiting until the referenced task is completed. Its Pi
        | session and workspace are kept for a re-queue.
      DialogFooter
        Button(type="button" variant="outline" @click="confirmOpen = false")
          | Cancel
        Button(type="button" @click="confirmKind" :disabled="busy !== ''")
          LockIcon(data-icon="inline-start")
          span Add blocking reference
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import { agency } from "@/lib/sdk";
import {
  TASK_REFERENCE_KINDS,
  taskNameClass,
  taskReferenceKindClass,
  taskReferenceKindIcon,
  taskReferenceKindLabel,
  taskReferenceKindTextClass,
  taskReferenceName,
  taskReferencePerspectiveKind,
  taskReferenceTitle,
  taskStateIcon,
  taskStateLabel,
} from "@/lib/task";
import {
  TaskReferenceKind,
  TaskState,
  type Task,
  type TaskReference,
} from "@cherrypi/sdk";
import { LockIcon, SearchIcon, TriangleAlertIcon, XIcon } from "@lucide/vue";
import { onClickOutside, useEventListener } from "@vueuse/core";
import { computed, ref, watch } from "vue";
import { RouterLink } from "vue-router";
import { toast } from "vue3-toastify";
import TaskPicker from "./TaskPicker.vue";

interface ReferenceRow {
  key: string;
  task: TaskReference;

  kind: TaskReferenceKind;

  incoming: boolean;
}

const props = defineProps<{ task: Task }>();
const emit = defineEmits<{ updated: [Task] }>();

const busy = ref("");
const search = ref("");
const open = ref(false);
const root = ref<HTMLElement | null>(null);
const confirmOpen = ref(false);
const pending = ref<ReferenceRow | null>(null);

const rows = computed<ReferenceRow[]>(() => {
  const outgoing = (props.task.references ?? []).map((ref) => ({
    key: `out:${ref.id}`,
    task: ref,
    kind: taskReferencePerspectiveKind(ref.kind, false),
    incoming: false,
  }));
  const incoming = (props.task.referencedBy ?? []).map((ref) => ({
    key: `in:${ref.id}`,
    task: ref,
    kind: taskReferencePerspectiveKind(ref.kind, true),
    incoming: true,
  }));
  return [...outgoing, ...incoming].sort((a, b) =>
    a.task.id.localeCompare(b.task.id),
  );
});

const excludeIds = computed(() => [
  props.task.id,
  ...rows.value.map((entry) => entry.task.id),
]);

const activeTask = computed(() => isActiveState(props.task.state));

function isActiveState(state: TaskState | undefined): boolean {
  return (
    state === TaskState.QUEUED ||
    state === TaskState.RUNNING ||
    state === TaskState.INPUT
  );
}

function isIncomingKind(kind: TaskReferenceKind): boolean {
  return (
    kind === TaskReferenceKind.BLOCKS ||
    kind === TaskReferenceKind.SUPERSEDED_BY
  );
}

function taskReferenceKindDescription(kind: TaskReferenceKind): string {
  return (
    TASK_REFERENCE_KINDS.find((option) => option.value === kind)?.description ??
    ""
  );
}

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

async function add(candidate: Task) {
  busy.value = candidate.id;
  try {
    const updated = await agency.addTaskReference(
      props.task.id,
      candidate.id,
      TaskReferenceKind.INFORMATIONAL,
    );
    emit("updated", updated);
    return true;
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
    return false;
  } finally {
    busy.value = "";
  }
}

async function addCandidate(candidate: Task) {
  if (await add(candidate)) search.value = "";
}

async function applyKind(row: ReferenceRow, kind: TaskReferenceKind) {
  if (kind === row.kind) return true;
  busy.value = row.task.id;
  try {
    if (isIncomingKind(kind) === row.incoming) {
      const updated = await agency.addTaskReference(
        props.task.id,
        row.task.id,
        kind,
      );
      emit("updated", updated);
      return true;
    }
    await agency.removeTaskReference(props.task.id, row.task.id, {
      reverse: row.incoming,
    });
    try {
      const updated = await agency.addTaskReference(
        props.task.id,
        row.task.id,
        kind,
      );
      emit("updated", updated);
      return true;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
      const restored = await agency.addTaskReference(
        props.task.id,
        row.task.id,
        row.kind,
      );
      emit("updated", restored);
      return false;
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
    return false;
  } finally {
    busy.value = "";
  }
}

function requestKind(row: ReferenceRow, kind: unknown) {
  const next = kind as TaskReferenceKind;
  if (next === row.kind) return;
  if (next === TaskReferenceKind.BLOCKED_BY && activeTask.value) {
    pending.value = row;
    confirmOpen.value = true;
    return;
  }
  if (
    next === TaskReferenceKind.BLOCKS &&
    isActiveState(row.task.state) &&
    !window.confirm(
      `Block "${taskReferenceTitle(row.task)}"? That task stops until this one is completed.`,
    )
  ) {
    return;
  }
  if (
    next === TaskReferenceKind.SUPERSEDES &&
    !window.confirm(
      `Supersede "${taskReferenceTitle(row.task)}"? A backlog, queued, or failed task is cancelled.`,
    )
  ) {
    return;
  }
  if (
    next === TaskReferenceKind.SUPERSEDED_BY &&
    !window.confirm(
      `Mark this task superseded by "${taskReferenceTitle(row.task)}"? This task is cancelled when it is in backlog, queued, or failed.`,
    )
  ) {
    return;
  }
  void applyKind(row, next);
}

async function confirmKind() {
  const row = pending.value;
  if (!row) return;
  if (await applyKind(row, TaskReferenceKind.BLOCKED_BY)) {
    confirmOpen.value = false;
  }
}

async function remove(row: ReferenceRow) {
  busy.value = row.task.id;
  try {
    const updated = await agency.removeTaskReference(
      props.task.id,
      row.task.id,
      { reverse: row.incoming },
    );
    emit("updated", updated);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

watch(
  () => props.task.id,
  () => {
    confirmOpen.value = false;
    pending.value = null;
  },
);
</script>
