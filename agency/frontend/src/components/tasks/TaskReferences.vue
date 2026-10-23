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
    SearchIcon.text-muted-foreground.pointer-events-none.absolute.size-4(
      class="top-1/2 left-2.5 -translate-y-1/2"
    )
    Input.w-full.pl-8(
      v-model="search"
      placeholder="Add reference…"
      autocomplete="off"
      @focus="open = true"
      @keydown.esc="open = false"
    )
    .bg-popover.text-popover-foreground.absolute.z-50.mt-1.max-h-56.w-full.overflow-y-auto.rounded-md.border.p-1.shadow-md(
      v-if="open"
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
          :disabled="busy === candidate.id"
        )
          component.size-3.shrink-0(:is="taskStateIcon(candidate.state)")
          strong.shrink-0.font-semibold {{ candidate.id }}:
          span.truncate(:class="taskNameClass(candidate)") {{ taskReferenceName(candidate) }}

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
import { referenceProjectIds } from "@/lib/project";
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
  taskStateClassFor,
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
import { onClickOutside, refDebounced } from "@vueuse/core";
import { computed, ref, watch } from "vue";
import { RouterLink } from "vue-router";
import { toast } from "vue3-toastify";

const props = defineProps<{ task: Task }>();
const emit = defineEmits<{ updated: [Task] }>();

const candidates = ref<Task[]>([]);
const searching = ref(false);
const busy = ref("");
const search = ref("");
const debouncedSearch = refDebounced(search, 300);
const root = ref<HTMLElement | null>(null);
const open = ref(false);
const confirmOpen = ref(false);
const pending = ref<TaskReference | null>(null);
const pendingKind = ref<TaskReferenceKind>(TaskReferenceKind.BLOCKING);

onClickOutside(root, () => (open.value = false));

const references = computed(() => props.task.references ?? []);
const referencedBy = computed(() => props.task.referencedBy ?? []);

// References are informational, so they are editable in any state. A blocking
// reference on a queued, running, or input task parks it, so it is confirmed
// first.
const activeTask = computed(
  () =>
    props.task.state === TaskState.QUEUED ||
    props.task.state === TaskState.RUNNING ||
    props.task.state === TaskState.INPUT,
);

// Candidates come from the task table's own server-side name/prompt/id search,
// scoped to the task's project; the task itself and its current references are
// filtered out. An empty query lists the project's recent tasks.
let searchSeq = 0;
async function searchCandidates() {
  const query = debouncedSearch.value.trim();
  const projectId = props.task.projectId || undefined;
  const seq = ++searchSeq;
  searching.value = true;
  try {
    const projectIDs = projectId
      ? await referenceProjectIds(projectId)
      : [undefined];
    const results = await Promise.all(
      projectIDs.map((id) =>
        agency.listTasks({ search: query || undefined, projectId: id }),
      ),
    );
    // A newer search superseded this response.
    if (seq !== searchSeq) return;
    const excluded = new Set([
      props.task.id,
      ...references.value.map((ref) => ref.id),
    ]);
    const found = new Map<string, Task>();
    for (const task of results.flat()) found.set(task.id, task);
    const available = [...found.values()].filter(
      (task) => !excluded.has(task.id),
    );
    candidates.value = query ? available.slice(0, 8) : available;
  } catch (err) {
    if (seq === searchSeq)
      toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    if (seq === searchSeq) searching.value = false;
  }
}

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

async function add(candidate: Task) {
  if (await apply(candidate.id, TaskReferenceKind.INFORMATIONAL)) {
    search.value = "";
    candidates.value = candidates.value.filter(
      (item) => item.id !== candidate.id,
    );
  }
}

// A blocking edge on a live task parks it, so it is confirmed first; a
// supersedes edge cancels the referenced task, so it warns too.
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
    // The removed task becomes an eligible candidate again.
    if (open.value) void searchCandidates();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
  }
}

watch(debouncedSearch, () => void searchCandidates());

// Opening the popover lists the project's recent tasks when the filter is
// empty; typing keeps it open.
watch(open, (isOpen) => {
  if (isOpen) void searchCandidates();
});
watch(search, (value) => {
  if (value.trim()) open.value = true;
});

// Reset the picker when the task changes.
watch(
  () => props.task.id,
  () => {
    search.value = "";
    candidates.value = [];
    open.value = false;
    confirmOpen.value = false;
    pending.value = null;
  },
);
</script>
