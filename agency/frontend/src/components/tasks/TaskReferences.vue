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
  p.text-muted-foreground.text-xs(
    v-if="!references.length && !referencedBy.length"
  )
    | —

  .flex.flex-col.gap-1(v-if="references.length")
    .flex.items-center.rounded-md.border.px-2.py-1(
      v-for="ref in references"
      :key="ref.id"
      class="gap-1.5"
      :class="taskReferenceKindClass(ref.kind)"
    )
      component.size-3.shrink-0(:is="taskStateIcon(ref.state)")
      RouterLink.inline-flex.min-w-0.flex-1.items-center.gap-1(
        :to="{ name: 'task', params: { id: ref.id } }"
        :title="taskReferenceTitle(ref)"
      )
        strong.shrink-0.text-xs.font-semibold {{ ref.id }}:
        span.truncate.text-xs(:class="taskNameClass(ref)") {{ taskReferenceName(ref) }}
      .border-border.divide-border.inline-flex.shrink-0.items-center.divide-x.overflow-hidden.rounded-md.border
        button(
          v-for="option in TASK_REFERENCE_KINDS"
          :key="option.value"
          type="button"
          class="px-1.5 py-0.5"
          :class="ref.kind === option.value ? 'bg-muted text-foreground' : 'text-muted-foreground hover:bg-muted/60'"
          :title="option.label"
          :aria-label="option.label"
          @click="requestKind(ref, option.value)"
          :disabled="busy === ref.id"
        )
          component.size-3(
            :class="ref.kind === option.value ? taskReferenceKindTextClass(option.value) : ''"
            :is="taskReferenceKindIcon(option.value)"
          )
      Button.shrink-0(
        variant="ghost"
        size="icon-sm"
        title="Remove reference"
        @click="remove(ref)"
        :disabled="busy === ref.id"
      )
        XIcon

  .flex.flex-col.gap-1(v-if="referencedBy.length")
    span.text-muted-foreground.text-xs Referenced by
    .flex.items-center(
      v-for="ref in referencedBy"
      :key="ref.id"
      class="gap-1.5 py-0.5"
    )
      component.size-3.shrink-0(
        :class="taskReferenceKindTextClass(ref.kind)"
        :is="taskReferenceKindIcon(ref.kind)"
      )
      strong.text-xs.font-semibold {{ ref.id }}:
      RouterLink.max-w-40.truncate.text-xs(
        :to="{ name: 'task', params: { id: ref.id } }"
        :title="taskReferenceTitle(ref)"
        :class="taskNameClass(ref)"
      ) {{ taskReferenceName(ref) }}
      span.text-muted-foreground.ml-auto.text-xs
        | {{ taskReferenceKindLabel(ref.kind) }}
    span.text-muted-foreground.text-xs
      | Incoming references are managed from the referencing task.

  p.text-muted-foreground.text-xs(
    v-if="references.some((ref) => ref.kind === TaskReferenceKind.BLOCKING)"
  )
    | A blocking reference holds this task back until the referenced task is completed.

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
          | "{{ pending ? taskReferenceTitle(pending) : "" }}" must be completed
          | before this task can be queued again.
      p.text-muted-foreground.text-sm
        | This task is currently {{ taskStateLabel(task.state) }}. It is stopped
        | and parked as Blocked until the referenced task is completed. Its Pi
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

const props = defineProps<{ task: Task }>();
const emit = defineEmits<{ updated: [Task] }>();

const busy = ref("");
const search = ref("");
const open = ref(false);
const root = ref<HTMLElement | null>(null);
const confirmOpen = ref(false);
const pending = ref<TaskReference | null>(null);
const pendingKind = ref<TaskReferenceKind>(TaskReferenceKind.BLOCKING);

const references = computed(() => props.task.references ?? []);
const excludeIds = computed(() => [
  props.task.id,
  ...references.value.map((ref) => ref.id),
]);
const referencedBy = computed(() => props.task.referencedBy ?? []);

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

const activeTask = computed(
  () =>
    props.task.state === TaskState.QUEUED ||
    props.task.state === TaskState.RUNNING ||
    props.task.state === TaskState.INPUT,
);

async function apply(refId: string, kind: TaskReferenceKind) {
  busy.value = refId;
  try {
    const updated = await agency.addTaskReference(props.task.id, refId, kind);
    emit("updated", updated);
    return updated;
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
    return null;
  } finally {
    busy.value = "";
  }
}

async function addCandidate(candidate: Task) {
  await apply(candidate.id, TaskReferenceKind.INFORMATIONAL);
  search.value = "";
}

function requestKind(ref: TaskReference, kind: unknown) {
  const next = kind as TaskReferenceKind;
  if (next === TaskReferenceKind.BLOCKING && activeTask.value) {
    pending.value = ref;
    pendingKind.value = TaskReferenceKind.BLOCKING;
    confirmOpen.value = true;
    return;
  }
  if (next === TaskReferenceKind.SUPERSEDES) {
    if (
      !window.confirm(
        `Supersede "${taskReferenceTitle(ref)}"? A draft, backlog, queued, or failed task is cancelled.`,
      )
    ) {
      return;
    }
  }
  void apply(ref.id, next);
}

async function confirmKind() {
  const ref = pending.value;
  if (!ref) return;
  if (await apply(ref.id, pendingKind.value)) confirmOpen.value = false;
}

async function remove(ref: TaskReference) {
  busy.value = ref.id;
  try {
    const updated = await agency.removeTaskReference(props.task.id, ref.id);
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
