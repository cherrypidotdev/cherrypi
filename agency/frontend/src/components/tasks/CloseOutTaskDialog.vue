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
      DialogTitle {{ $t("closeOut.title") }}
      DialogDescription
        | {{ $t("closeOut.description", { name: task ? taskName(task) : "" }) }}
    .grid.gap-2
      button.flex.items-center.gap-2.rounded-lg.border.p-3.text-left.text-sm.transition-colors(
        type="button"
        :class="choice === TaskState.COMPLETED ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'hover:bg-muted/50'"
        @click="choice = TaskState.COMPLETED"
      )
        CheckIcon.size-4.shrink-0
        .flex.flex-col
          span.font-medium {{ $t("usage.state.completed") }}
          span.text-muted-foreground.text-xs {{ $t("closeOut.completed") }}
      button.flex.items-center.gap-2.rounded-lg.border.p-3.text-left.text-sm.transition-colors(
        type="button"
        :class="choice === TaskState.CANCELLED ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'hover:bg-muted/50'"
        @click="choice = TaskState.CANCELLED"
      )
        BanIcon.size-4.shrink-0
        .flex.flex-col
          span.font-medium {{ $t("tasks.state.cancelled") }}
          span.text-muted-foreground.text-xs {{ $t("closeOut.cancelled") }}
    DialogFooter
      Button(
        type="button"
        variant="outline"
        @click="emit('update:open', false)"
      ) {{ $t("common.cancel") }}
      Button(type="button" @click="closeOut" :disabled="busy || !task")
        Spinner(v-if="busy" data-icon="inline-start")
        CheckIcon(v-else data-icon="inline-start")
        span {{ $t("closeOut.submit") }}
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
import { useI18n } from "vue-i18n";
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

watch(
  () => props.open,
  (open) => {
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
        ? t("attention.event.cancelled")
        : t("attention.event.completed"),
    );
    emit("update:open", false);
    emit("closed");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = false;
  }
}

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  closeOut:
    title: "Close out task"
    completed: "The work is done."
    cancelled: "The work is abandoned."
    submit: "Close out"
    description: 'Choose how to close out "{name}". It releases the agent''s slot and stays in the task list under the chosen state.'
    blockedHint: "Stop the run or let it settle before closing out"
ru:
  closeOut:
    title: "Закрыть задачу"
    completed: "Работа выполнена."
    cancelled: "Работа заброшена."
    submit: "Закрыть"
    description: "Выберите, как закрыть «{name}». Слот агента освобождается, а задача остаётся в списке в выбранном состоянии."
    blockedHint: "Остановите запуск или дождитесь завершения, прежде чем закрывать"
</i18n>
