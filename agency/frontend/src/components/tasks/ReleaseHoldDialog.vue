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
      DialogTitle {{ $t("releaseHold.title") }}
      DialogDescription
        | {{ $t("releaseHold.description", { hold: holdLabel, name: task ? taskName(task) : "" }) }}
    .grid.gap-2
      button.flex.items-center.gap-2.rounded-lg.border.p-3.text-left.text-sm.transition-colors(
        type="button"
        :class="choice === TaskHoldReleaseMode.SETTLE ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'hover:bg-muted/50'"
        @click="choice = TaskHoldReleaseMode.SETTLE"
      )
        CheckIcon.size-4.shrink-0
        .flex.flex-col
          span.font-medium {{ $t("releaseHold.letRest") }}
          span.text-muted-foreground.text-xs {{ $t("releaseHold.completedHint") }}
      button.flex.items-center.gap-2.rounded-lg.border.p-3.text-left.text-sm.transition-colors(
        type="button"
        :class="choice === TaskHoldReleaseMode.QUEUE ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'hover:bg-muted/50'"
        @click="choice = TaskHoldReleaseMode.QUEUE"
      )
        PlayIcon.size-4.shrink-0
        .flex.flex-col
          span.font-medium {{ $t("releaseHold.runAgain") }}
          span.text-muted-foreground.text-xs {{ $t("releaseHold.runAgainHint") }}
      button.flex.items-center.gap-2.rounded-lg.border.p-3.text-left.text-sm.transition-colors(
        type="button"
        :class="choice === TaskHoldReleaseMode.COMPLETE ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'hover:bg-muted/50'"
        @click="choice = TaskHoldReleaseMode.COMPLETE"
      )
        CheckCheckIcon.size-4.shrink-0
        .flex.flex-col
          span.font-medium {{ $t("releaseHold.closeOut") }}
          span.text-muted-foreground.text-xs {{ $t("releaseHold.closeOutHint") }}
    .flex.flex-col.gap-1(v-if="choice === TaskHoldReleaseMode.QUEUE")
      label.text-xs.font-medium(for="release-hold-prompt") {{ $t("releaseHold.followUp") }}
      Textarea#release-hold-prompt(
        v-model="prompt"
        :placeholder="$t('releaseHold.followUpPlaceholder')"
        rows="3"
        data-testid="release-hold-prompt"
      )
    DialogFooter
      Button(
        type="button"
        variant="outline"
        @click="emit('update:open', false)"
      ) {{ $t("common.cancel") }}
      Button(
        type="button"
        @click="release"
        :disabled="busy || !task || !hold || (choice === TaskHoldReleaseMode.QUEUE && !prompt.trim())"
      )
        Spinner(v-if="busy" data-icon="inline-start")
        LockOpenIcon(v-else data-icon="inline-start")
        span {{ $t("releaseHold.submit") }}
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
import { Textarea } from "@/components/ui/textarea";
import { agent } from "@/lib/sdk";
import { taskName } from "@/lib/task";
import { TaskHoldReleaseMode, type Task, type TaskHold } from "@cherrypi/sdk";
import { CheckCheckIcon, CheckIcon, LockOpenIcon, PlayIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{
  open: boolean;
  task: Task | null;
  hold: TaskHold | null;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  released: [task: Task];
}>();

const busy = ref(false);
const choice = ref(TaskHoldReleaseMode.SETTLE);
const prompt = ref("");

const holdLabel = computed(() => {
  const hold = props.hold;
  if (!hold) return "";
  const plugin = hold.pluginName || hold.plugin;
  return hold.reason ? `${plugin}: ${hold.reason}` : `${plugin}: ${hold.key}`;
});

watch(
  () => props.open,
  (open) => {
    if (!open) return;
    choice.value = TaskHoldReleaseMode.SETTLE;
    prompt.value = "";
  },
);

async function release() {
  const task = props.task;
  const hold = props.hold;
  if (!task || !hold) return;
  busy.value = true;
  try {
    const updated = await agent.releaseTaskHold(task.id, {
      plugin: hold.plugin,
      key: hold.key,
      mode: choice.value,
      prompt: prompt.value.trim(),
    });
    toast.success(`Released the ${hold.key} hold`);
    emit("released", updated);
    emit("update:open", false);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  releaseHold:
    title: "Release the hold"
    letRest: "Let the task rest"
    completedHint: "The work is done; the task waits for a close-out."
    runAgain: "Run the agent again"
    runAgainHint: "Queue the task with a follow-up prompt."
    closeOut: "Close the task out"
    closeOutHint: "The work is done and needs no close-out."
    followUp: "Follow-up prompt"
    submit: "Release"
    followUpPlaceholder: "What should the agent do now?"
    description: 'Release "{hold}" on "{name}" so the task can go on without it. The plugin that placed the hold is told; it decides whether to place it again.'
ru:
  releaseHold:
    title: "Снять удержание"
    letRest: "Оставить задачу в покое"
    completedHint: "Работа выполнена; задача ждёт закрытия."
    runAgain: "Запустить агента снова"
    runAgainHint: "Поставить задачу в очередь с уточняющим промптом."
    closeOut: "Закрыть задачу"
    closeOutHint: "Работа выполнена и не требует закрытия."
    followUp: "Уточняющий промпт"
    submit: "Снять"
    followUpPlaceholder: "Что агент должен сделать сейчас?"
    description: "Снимите «{hold}» с «{name}», чтобы задача могла продолжиться без него. Плагин, установивший удержание, получит уведомление и сам решит, ставить ли его снова."
</i18n>
