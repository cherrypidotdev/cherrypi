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
.flex.min-w-0.flex-1.flex-col(v-show="active")
  .shrink-0.border-b.px-4.py-2(
    v-if="task && waitingReason"
    class="bg-amber-500/10"
    data-testid="task-waiting-reason"
  )
    .mx-auto.flex.max-w-3xl.items-start.gap-3
      InfoIcon.text-muted-foreground.size-4.shrink-0(class="mt-0.5")

      p.text-sm.font-medium.break-words
        template(v-for="(part, index) in waitingParts" :key="index")
          RouterLink.font-semibold.underline.underline-offset-2(
            v-if="part.kind === 'id'"
            class="hover:text-foreground/80"
            :to="{ name: 'task', params: { id: part.id } }"
          ) {{ part.id }}
          TaskHoldReason(
            v-else-if="part.kind === 'hold'"
            :plugin-name="part.pluginName"
            :reason="part.reason"
            :url="part.url"
            :icon-url="part.pluginIconUrl"
          )
          template(v-else) {{ part.text }}

  .shrink-0.border-b.px-4.py-2(
    v-if="task && task.state === TaskState.FAILED"
    class="bg-destructive/10"
  )
    .mx-auto.flex.max-w-3xl.items-start.gap-3
      TriangleAlertIcon.text-destructive.size-4.shrink-0(class="mt-0.5")
      .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
        p.text-destructive.text-sm.font-medium {{ $t("taskConsolePage.failed") }}
        p.text-muted-foreground.text-xs.break-words {{ task.error || "No error details." }}

        p.text-muted-foreground.text-xs.break-words(
          v-if="task.needsOperator"
          data-testid="task-needs-operator"
        )
          span.font-medium {{ $t("taskConsolePage.needsOperator") }}
          | {{ task.needsOperator }}
      Button.shrink-0(
        v-if="!sessionDeleted"
        variant="outline"
        size="sm"
        @click="$emit('reopen', true)"
      )
        PlayIcon(data-icon="inline-start")
        span {{ $t("taskConsolePage.requeue") }}

  TaskPrompt.flex-1.overflow-hidden(
    v-if="task && showPrompt"
    :hint="promptHint"
    :task="task"
    :editable="canEditPrompt"
    :agent-id="task.assigneeActorId ? agentIdForActor(task.assigneeActorId) : ''"
    @saved="$emit('draft-saved', $event)"
    @changed="$emit('attachment-uploaded')"
    @remove="$emit('remove-attachment', $event)"
    @open-attachment="$emit('open-attachment', $event)"
  )

  .flex.flex-1.items-center.justify-center.p-6(
    v-else-if="task && !isStarted && !canFollowUp"
  )
    p.text-muted-foreground.max-w-md.text-center.text-sm
      | {{ $t("taskConsolePage.noConsole") }}

  .relative.flex.min-h-0.flex-1.flex-col.overflow-hidden(
    v-else-if="task"
    @dragenter.prevent="$emit('drag-enter', $event)"
    @dragover.prevent
    @dragleave.prevent="$emit('drag-leave')"
    @drop.prevent="$emit('drop', $event)"
    @paste="$emit('paste', $event)"
    data-testid="console-dropzone"
  )
    .border-primary.pointer-events-none.absolute.inset-2.z-20.flex.items-center.justify-center.rounded-lg.border-2.border-dashed(
      v-if="draggingFiles && canCompose"
      class="bg-background/80"
    )
      .flex.flex-col.items-center.gap-1.text-sm.font-medium
        PaperclipIcon.size-5
        span {{ $t("taskPrompt.dropFiles") }}

    TaskConsole.min-h-0.flex-1(
      :key="task.sessionId"
      :id="id"
      :project-id="task?.projectId ?? ''"
      :busy="busy"
      :waiting-for-input="task?.runtime?.waitingForInput"
      :run-over="runOver"
      :attachments="task?.attachments ?? []"
      :actor-pictures="actorPictures"
      :actor-names="actorNames"
      :agent-id="task.assigneeActorId && !sessionDeleted ? agentIdForActor(task.assigneeActorId) : ''"
      :agent="assigneeAgent"
      :model-config="config"
      :model-name="config?.modelName ?? ''"
      :model-provider="config?.modelProvider ?? ''"
      :model-selection="modelSelection"
      :model-disabled="saving.model || !config"
      :usage-runs="taskUsage?.runs ?? []"
      :can-answer="canAnswer"
      :can-interrupt="steer"
      :state-changes="stateChanges"
      :unsent="unsent"
      @queue="$emit('queue', $event)"
      @cancel-unsent="$emit('cancel-unsent', $event)"
      @open-attachment="$emit('open-attachment', $event)"
      @open-image="$emit('open-image', $event)"
      @open-diagram="$emit('open-diagram', $event)"
      @model-change="$emit('model-change', $event)"
      @live-timing="$emit('live-timing', $event)"
    )

    //- Published plan: a compact status line above the composer; the full

    TaskProgressStrip(
      v-if="task"
      :progress="task.progress"
      :state="task.state"
      :size="task.size"
      :can-remind="canCompose"
      @open="$emit('open-plan')"
      @remind="$emit('remind-plan')"
    )

    TaskFollowUpComposer(
      v-if="task"
      :task="task"
      :prompt="prompt"
      :tab="tab"
      :can-compose="canCompose"
      :can-follow-up="canFollowUp"
      :awaiting-answers="awaitingAnswers"
      :steer="steer"
      :sending="sending"
      :stopping="stopping"
      :config="config"
      :model-selection="modelSelection"
      :model-saving="saving.model"
      :staged-attachments="stagedAttachments"
      :task-usage="taskUsage"
      @update:prompt="$emit('update:prompt', $event)"
      @update:tab="$emit('update:tab', $event)"
      @send="$emit('send')"
      @enter="$emit('enter')"
      @stop="$emit('stop')"
      @model-change="$emit('model-change', $event)"
      @staged-uploaded="$emit('staged-uploaded', $event)"
      @remove-staged="$emit('remove-staged', $event)"
      @open-attachment="$emit('open-attachment', $event)"
      @open-new-sketch="$emit('open-new-sketch')"
      @open-sketch="$emit('open-sketch', $event)"
    )
</template>

<script setup lang="ts">
import TaskConsole from "@/components/tasks/TaskConsole.vue";
import TaskFollowUpComposer from "@/components/tasks/TaskFollowUpComposer.vue";
import TaskHoldReason from "@/components/tasks/TaskHoldReason.vue";
import TaskProgressStrip from "@/components/tasks/TaskProgressStrip.vue";
import TaskPrompt from "@/components/tasks/TaskPrompt.vue";
import { Button } from "@/components/ui/button";
import { useActorPictures } from "@/composables/useActorPictures";
import { useAgents } from "@/composables/useAgents";
import { namedSender } from "@/lib/actors";
import type {
  StateChange,
  UnsentMessage,
  UserMessageParts,
} from "@/lib/console";
import type { DiagramSource, InlineImage } from "@/lib/lightbox";
import {
  taskWaitingReason,
  waitingReasonParts,
  type TaskModelSelection,
} from "@/lib/task";
import {
  TaskState,
  type Attachment,
  type Task,
  type TaskConfig,
  type TaskUsageWithRuns,
} from "@cherrypi/sdk";
import {
  InfoIcon,
  PaperclipIcon,
  PlayIcon,
  TriangleAlertIcon,
} from "@lucide/vue";
import { computed, onMounted } from "vue";

const props = defineProps<{
  active: boolean;

  id: string;
  task?: Task;
  taskUsage?: TaskUsageWithRuns;
  config?: TaskConfig;
  modelSelection: TaskModelSelection;
  saving: { model: boolean };
  canFollowUp: boolean;
  canCompose: boolean;

  canAnswer: boolean;
  prompt: string;
  tab: "write" | "preview";
  stagedAttachments: Attachment[];
  draggingFiles: boolean;
  taskQueue: (UserMessageParts & { id: number; queueId: string })[];

  cancellingUnsent: string;
  sending: boolean;
  stopping: boolean;

  stateChanges: StateChange[];
}>();

defineEmits<{
  (e: "update:prompt", value: string): void;
  (e: "update:tab", value: "write" | "preview"): void;
  (e: "send"): void;
  (e: "enter"): void;
  (e: "stop"): void;
  (e: "reopen", toQueue: boolean): void;
  (e: "remove-attachment", attachment: Attachment): void;
  (e: "attachment-uploaded"): void;
  (e: "draft-saved", task: Task): void;
  (e: "open-attachment", attachment: Attachment): void;
  (e: "open-image", image: InlineImage): void;

  (e: "open-diagram", diagram: DiagramSource): void;
  (e: "open-new-sketch"): void;
  (e: "open-sketch", attachment: Attachment): void;
  (e: "model-change", selection: TaskModelSelection): void;
  (
    e: "live-timing",
    timing: { toolRunning: boolean; modelWaiting: boolean },
  ): void;
  (e: "staged-uploaded", attachment: Attachment): void;
  (e: "remove-staged", attachment: Attachment): void;
  (e: "drag-enter", event: DragEvent): void;
  (e: "drag-leave"): void;
  (e: "drop", event: DragEvent): void;
  (e: "paste", event: ClipboardEvent): void;
  (e: "queue", steering: { id: string; text: string }[]): void;
  (e: "cancel-unsent", item: UnsentMessage): void;
  (e: "open-plan"): void;
  (e: "remind-plan"): void;
}>();

const { agents, loadAgents, agentIdForActor } = useAgents();
const { actorNames, actorPictures } = useActorPictures();

const assigneeAgent = computed(() => {
  const actorId = props.task?.assigneeActorId;
  return actorId
    ? agents.value.find((item) => item.actorId === actorId)
    : undefined;
});

onMounted(() => {
  void loadAgents();
});

const isBacklog = computed(() => props.task?.state === TaskState.BACKLOG);
const hasStarted = computed(() => Boolean(props.task?.startedAt));
const isPostponed = computed(() => isBacklog.value && hasStarted.value);

const isEditableBacklog = computed(() => isBacklog.value && !hasStarted.value);
const canEditPrompt = isEditableBacklog;
const isStarted = computed(() => Boolean(props.task?.sessionId));

const awaitingAnswers = computed(
  () =>
    props.task?.state === TaskState.INPUT ||
    props.task?.runtime?.waitingForInput === true,
);

const runOver = computed(() => {
  const state = props.task?.state;
  return (
    state === TaskState.SETTLED ||
    state === TaskState.FAILED ||
    state === TaskState.COMPLETED ||
    state === TaskState.CANCELLED
  );
});

const showPrompt = computed(() => !isStarted.value && !isPostponed.value);
const waitingReason = computed(() =>
  props.task ? taskWaitingReason(props.task) : "",
);
const waitingParts = computed(() =>
  waitingReasonParts(waitingReason.value, props.task?.holds ?? []),
);
const promptHint = computed(() => {
  if (isPostponed.value) return "Started · follow up below to queue it again";
  if (isBacklog.value) return "Backlog · click to edit";
  if (waitingReason.value) return waitingReason.value;
  return "";
});

const sessionDeleted = computed(() => Boolean(props.task?.sessionDeletedAt));

const showPendingPrompt = computed(
  () => !!props.task?.pendingPrompt && isStarted.value,
);

const stashCancellable = computed(() => {
  const state = props.task?.state;
  return state === TaskState.INPUT || state === TaskState.RUNNING;
});
const busy = computed(() => !props.task || props.task.runtime?.runInProgress);

const steer = computed(() => props.task?.runtime?.runInProgress === true);

const PENDING_PROMPT_KEY = "pending-prompt";

const unsent = computed<UnsentMessage[]>(() => {
  const rows: UnsentMessage[] = [];
  const task = props.task;
  if (task && showPendingPrompt.value) {
    rows.push({
      key: PENDING_PROMPT_KEY,
      queueId: "",
      text: task.pendingPrompt,
      attachmentIds: [],
      sender: pendingPromptSender(task),

      cancellable: stashCancellable.value,
      busy: props.cancellingUnsent === PENDING_PROMPT_KEY,
    });
  }
  for (const item of props.taskQueue) {
    const key = item.queueId || `queue-${item.id}`;
    rows.push({
      key,
      queueId: item.queueId,
      text: item.text,
      attachmentIds: item.attachmentIds,

      sender: namedSender(item.sender, actorNames.value),

      cancellable: Boolean(item.queueId),
      busy: props.cancellingUnsent === key,
    });
  }
  return rows;
});

function pendingPromptSender(task: Task): UnsentMessage["sender"] {
  const id = task.pendingPromptActorId;
  const name = task.pendingPromptActorName;
  if (!id && !name) return undefined;
  return namedSender(
    { id, username: "", name: name || id, kind: "user" },
    actorNames.value,
  );
}
</script>

<style lang="postcss" scoped></style>

<i18n lang="yaml">
en:
  taskConsolePage:
    failed: "The task failed."
    needsOperator: "Needs an operator:"
    requeue: "Re-queue"
    noConsole: "This task has no console yet. Its Pi session starts when the scheduler finds an eligible agent with a free slot."
ru:
  taskConsolePage:
    failed: "Задача завершилась ошибкой."
    needsOperator: "Требуется оператор:"
    requeue: "Поставить в очередь снова"
    noConsole: "У задачи пока нет консоли. Сеанс Pi запустится, когда планировщик найдёт подходящего агента со свободным слотом."
</i18n>
