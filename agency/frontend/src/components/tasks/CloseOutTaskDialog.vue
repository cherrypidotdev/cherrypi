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
  DialogContent(class="sm:max-w-md")
    DialogHeader
      DialogTitle Close out task
      DialogDescription
        | Choose how to close out "{{ task ? taskName(task) : "" }}". It
        | releases the agent's slot and stays in the task list under the chosen
        | state.
    .grid.gap-2
      button.flex.items-center.gap-2.rounded-lg.border.p-3.text-left.text-sm.transition-colors(
        type="button"
        :class="choice === TaskState.COMPLETED ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'hover:bg-muted/50'"
        @click="choice = TaskState.COMPLETED"
      )
        CheckIcon.size-4.shrink-0
        .flex.flex-col
          span.font-medium Completed
          span.text-muted-foreground.text-xs The work is done.
      button.flex.items-center.gap-2.rounded-lg.border.p-3.text-left.text-sm.transition-colors(
        type="button"
        :class="choice === TaskState.CANCELLED ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'hover:bg-muted/50'"
        @click="choice = TaskState.CANCELLED"
      )
        BanIcon.size-4.shrink-0
        .flex.flex-col
          span.font-medium Cancelled
          span.text-muted-foreground.text-xs The work is abandoned.
    DialogFooter
      Button(
        type="button"
        variant="outline"
        @click="emit('update:open', false)"
      ) Cancel
      Button(type="button" @click="closeOut" :disabled="busy || !task")
        Spinner(v-if="busy" data-icon="inline-start")
        CheckIcon(v-else data-icon="inline-start")
        span Close out
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
import { Spinner } from "@/components/ui/spinner";
import { agent } from "@/lib/sdk";
import { defaultCloseOutOutcome, taskName } from "@/lib/task";
import { TaskState, type Task } from "@cherrypi/sdk";
import { BanIcon, CheckIcon } from "@lucide/vue";
import { ref, watch } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{
  open: boolean;
  task: Task | null;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  closed: [];
}>();

const busy = ref(false);
const choice = ref(TaskState.COMPLETED);

// A failed task is usually abandoned, so default it to Cancelled; everything
// else defaults to Completed. Reset each time the dialog opens.
watch(
  () => [props.open, props.task?.id] as const,
  ([open]) => {
    if (!open || !props.task) return;
    choice.value = defaultCloseOutOutcome(props.task.state);
  },
);

async function closeOut() {
  const task = props.task;
  if (!task) return;
  busy.value = true;
  try {
    await agent.closeOutTask(task.id, choice.value);
    toast.success(
      choice.value === TaskState.CANCELLED
        ? "Task cancelled"
        : "Task completed",
    );
    emit("update:open", false);
    emit("closed");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = false;
  }
}
</script>
