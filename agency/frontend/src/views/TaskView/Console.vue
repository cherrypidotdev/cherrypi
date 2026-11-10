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
          template(v-else) {{ part.text }}

  .shrink-0.border-b.px-4.py-2(
    v-if="task && task.state === TaskState.FAILED"
    class="bg-destructive/10"
  )
    .mx-auto.flex.max-w-3xl.items-start.gap-3
      TriangleAlertIcon.text-destructive.size-4.shrink-0(class="mt-0.5")
      .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
        p.text-destructive.text-sm.font-medium The task failed.
        p.text-muted-foreground.text-xs.break-words {{ task.error || "No error details." }}
      Button.shrink-0(
        v-if="!sessionDeleted"
        variant="outline"
        size="sm"
        @click="$emit('reopen', true)"
      )
        PlayIcon(data-icon="inline-start")
        span Re-queue

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
      | This task has no console yet. Its Pi session starts when the scheduler
      | finds an eligible agent with a free slot.

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
        span Drop files to attach

    TaskConsole.min-h-0.flex-1(
      :key="task.sessionId"
      :id="id"
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
      :state-changes="stateChanges"
      @queue="$emit('queue', $event)"
      @open-attachment="$emit('open-attachment', $event)"
      @open-image="$emit('open-image', $event)"
      @open-diagram="$emit('open-diagram', $event)"
      @model-change="$emit('model-change', $event)"
      @live-timing="$emit('live-timing', $event)"
    )

    .shrink-0.px-4.pt-3.pb-3(v-if="showPendingPrompt")
      .mx-auto.w-full.max-w-3xl
        TaskPrompt(
          field="pendingPrompt"
          compact
          :hint="pendingPromptHint"
          :task="task"
          :editable="canEditPrompt"
          :text="task.pendingPrompt"
          @saved="$emit('pending-prompt-saved')"
        )

    Collapsible.shrink-0.border-t.px-4.pt-3(
      v-if="taskQueue.length"
      v-model:open="queueOpen"
      class="group/queue"
      :class="queueOpen ? '' : 'pb-3'"
    )
      .mx-auto.flex.max-w-3xl.flex-col.gap-1
        .flex.items-center.gap-2
          CollapsibleTrigger.flex.flex-1.cursor-pointer.items-center.gap-2.text-left(
            data-testid="steering-queue-trigger"
          )
            ChevronRightIcon.text-muted-foreground.size-4.shrink-0.transition-transform(
              class="group-data-[state=open]/queue:rotate-90"
            )
            p.text-muted-foreground.text-xs
              | Queued steering ({{ taskQueue.length }}){{ queueOpen ? " — injected between blocks" : "" }}
          Button.ml-auto(
            variant="outline"
            size="sm"
            @click="$emit('clear-queue')"
            :disabled="clearingQueue"
          )
            Spinner(v-if="clearingQueue" data-icon="inline-start")
            Trash2Icon(v-else data-icon="inline-start")
            span Clear queue
        CollapsibleContent
          .flex.flex-col.gap-1
            .d-chat.d-chat-end(
              v-for="item in queuedMessages"
              :key="item.id"
              data-testid="steering-message"
            )
              ChatAvatar(
                v-if="item.sender && item.sender.kind !== 'system'"
                :sender="item.sender"
                :picture="pictureFor(item.sender)"
              )
              .d-chat-header(v-if="item.sender")
                MessageSender(:sender="item.sender" :avatar="false")
              .d-chat-bubble.border.border-dashed(class="bg-muted/60 max-w-[80%]")
                .text-sm.break-words.whitespace-pre-wrap(v-if="!item.text")
                MarkdownText.prose.prose-sm.max-w-none(
                  v-else
                  class="dark:prose-invert prose-p:my-1 prose-pre:my-2"
                  :text="item.text"
                )
              .d-chat-footer(v-if="item.attachmentIds.length")
                AttachmentChips(
                  :attachments="queueAttachments(item.attachmentIds)"
                  @open="$emit('open-attachment', $event)"
                )

    TaskProgressStrip(
      v-if="task"
      :progress="task.progress"
      :state="task.state"
      :size="task.size"
      :can-remind="canCompose"
      @open="$emit('open-plan')"
      @remind="$emit('remind-plan')"
    )

    form.relative.shrink-0.border-t.p-3(
      ref="composer"
      :class="canCompose ? '' : 'opacity-50'"
      @submit.prevent="$emit('send')"
    )
      .absolute.inset-x-0.-top-1.z-10.flex.h-2.cursor-row-resize.items-center.justify-center(
        v-if="canCompose && tab === 'write'"
        title="Drag to resize the composer"
        @pointerdown="startResize"
        data-testid="composer-resize-handle"
      )
        .bg-border.h-1.w-10.rounded-full.transition-opacity(
          :class="resizing ? 'opacity-100' : 'opacity-40 hover:opacity-100'"
        )

      .mx-auto.flex.max-w-3xl.flex-col.gap-3
        p.text-muted-foreground.inline-flex.items-start.gap-1.text-xs(
          v-if="sessionDeleted"
          data-testid="session-cleaned-up-hint"
        )
          CircleSlashIcon.size-3.shrink-0(class="mt-0.5")
          span
            | The agent-side Pi session and workspace are gone (cleaned up, or
            | the agent restarted with fresh data). Sending a message replays
            | the stored conversation in a new Pi session.

        p.text-muted-foreground.inline-flex.items-start.gap-1.text-xs(
          v-if="isInput"
          data-testid="composer-input-hint"
        )
          InfoIcon.size-3.shrink-0(class="mt-0.5")
          span
            | The agent is waiting for your answers. You can draft a follow-up
            | now; sending unlocks once the questionnaire is submitted.

        .flex.flex-col.gap-2
          .flex.flex-wrap.items-center.gap-2
            WritePreviewTabs(
              test-id="composer-tab"
              :model-value="tab"
              @update:model-value="$emit('update:tab', $event)"
            )

            .ml-auto.flex.flex-wrap.items-center.gap-2
              TaskModelPicker(
                v-if="task.assigneeActorId && !sessionDeleted"
                :agent-id="agentIdForActor(task.assigneeActorId)"
                :config="config"
                :selection="modelSelection"
                :use-defaults="false"
                @change="$emit('model-change', $event)"
                :disabled="saving.model || !config"
              )
              Button.shrink-0(
                v-if="steer"
                type="button"
                variant="outline"
                title="Stop the agent"
                @click="$emit('stop')"
                :disabled="stopping"
              )
                Spinner(v-if="stopping" data-icon="inline-start")
                SquareIcon(v-else data-icon="inline-start")
                span Stop
              Button.shrink-0(
                type="submit"
                :title="isInput ? 'Answer the questionnaire to send this follow-up' : steer ? 'Steer: injected between the run\u2019s blocks' : 'Send prompt'"
                :disabled="!canFollowUp || !prompt.trim() || sending"
              )
                SendIcon(data-icon="inline-start")
                span {{ steer ? "Steer" : "Send" }}

          MarkdownEditor(
            v-if="tab === 'write'"
            ref="promptInput"
            :placeholder="isInput ? 'You can draft a follow-up while the questionnaire is open…' : 'Send a follow-up prompt…'"
            submit-on-enter
            references
            test-id="composer-editor"
            aria-label="Follow-up prompt"
            title="Enter sends · Shift+Enter adds a line · Tab indents · Ctrl+B bold · Ctrl+I italic · Ctrl+E code · Ctrl+K deletes to the end of the line · @ mentions · # tasks"
            :model-value="prompt"
            :reference-project-id="task?.projectId ?? ''"
            :reference-task-id="task?.id ?? ''"
            :fill="textareaHeight !== null"
            :style="textareaStyle"
            height="6.5rem"
            @update:model-value="$emit('update:prompt', $event)"
            @submit="$emit('enter')"
            :disabled="!canCompose"
          )
          .text-foreground.border-border.w-full.rounded-lg.border.px-4.py-3.text-sm(
            v-else
            data-testid="composer-preview"
          )
            MarkdownText.prose.prose-sm.max-w-none(
              class="prose-p:my-1 prose-pre:my-2"
              :style="inheritProse"
              :text="prompt"
            )

          .flex.flex-col.gap-1(v-if="canCompose || contextWindow")
            .flex.min-h-7.flex-wrap.items-center.gap-2
              template(v-if="canCompose")
                AttachmentUploadButton(
                  label="Attach"
                  :task-id="task.id"
                  @uploaded="$emit('staged-uploaded', $event)"
                )
                Button(
                  type="button"
                  variant="outline"
                  size="sm"
                  title="Draw a sketch"
                  @click="$emit('open-new-sketch')"
                )
                  PencilIcon(data-icon="inline-start")
                  span Sketch
                span.text-muted-foreground.text-xs(
                  v-if="!stagedAttachments.length"
                ) Paste, drop, or pick files to send with this prompt
              ContextMeter.ml-auto(
                v-if="contextWindow"
                :tokens="taskUsage?.contextTokens ?? 0"
                :window="contextWindow"
              )
            AttachmentList(
              v-if="canCompose && stagedAttachments.length"
              removable
              sketchable
              compact
              :show-author="false"
              :attachments="stagedAttachments"
              @remove="$emit('remove-staged', $event)"
              @sketch="$emit('open-sketch', $event)"
              @open="$emit('open-attachment', $event)"
            )
          p.text-muted-foreground.text-xs(v-if="isPostponed")
            | Sending a follow-up queues this task immediately.
</template>

<script setup lang="ts">
import AttachmentChips from "@/components/attachments/AttachmentChips.vue";
import AttachmentList from "@/components/attachments/AttachmentList.vue";
import AttachmentUploadButton from "@/components/attachments/AttachmentUploadButton.vue";
import MarkdownEditor from "@/components/MarkdownEditor.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import MessageSender from "@/components/MessageSender.vue";
import ChatAvatar from "@/components/tasks/ChatAvatar.vue";
import ContextMeter from "@/components/tasks/ContextMeter.vue";
import TaskConsole from "@/components/tasks/TaskConsole.vue";
import TaskModelPicker from "@/components/tasks/TaskModelPicker.vue";
import TaskProgressStrip from "@/components/tasks/TaskProgressStrip.vue";
import TaskPrompt from "@/components/tasks/TaskPrompt.vue";
import WritePreviewTabs from "@/components/tasks/WritePreviewTabs.vue";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Spinner } from "@/components/ui/spinner";
import { useActorPictures } from "@/composables/useActorPictures";
import { useAgents } from "@/composables/useAgents";
import { namedSender } from "@/lib/actors";
import type { StateChange, UserMessageParts } from "@/lib/console";
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
  ChevronRightIcon,
  CircleSlashIcon,
  InfoIcon,
  PaperclipIcon,
  PencilIcon,
  PlayIcon,
  SendIcon,
  SquareIcon,
  Trash2Icon,
  TriangleAlertIcon,
} from "@lucide/vue";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

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
  taskQueue: (UserMessageParts & { id: number })[];
  clearingQueue: boolean;
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
  (e: "pending-prompt-saved"): void;
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
  (e: "queue", steering: string[]): void;
  (e: "clear-queue"): void;
  (e: "open-plan"): void;
  (e: "remind-plan"): void;
}>();

const { agents, loadAgents, agentIdForActor } = useAgents();
const { actorNames, actorPictures, pictureFor } = useActorPictures();

const assigneeAgent = computed(() => {
  const actorId = props.task?.assigneeActorId;
  return actorId
    ? agents.value.find((item) => item.actorId === actorId)
    : undefined;
});

onMounted(() => {
  void loadAgents();
});

const inheritProse = {
  "--tw-prose-body": "currentColor",
  "--tw-prose-headings": "currentColor",
  "--tw-prose-lead": "currentColor",
  "--tw-prose-links": "currentColor",
  "--tw-prose-bold": "currentColor",
  "--tw-prose-counters": "currentColor",
  "--tw-prose-bullets": "currentColor",
  "--tw-prose-hr": "currentColor",
  "--tw-prose-quotes": "currentColor",
  "--tw-prose-quote-borders": "currentColor",
  "--tw-prose-captions": "currentColor",
  "--tw-prose-code": "currentColor",
  "--tw-prose-pre-code": "currentColor",
  "--tw-prose-pre-bg": "transparent",
  "--tw-prose-th-borders": "currentColor",
  "--tw-prose-td-borders": "currentColor",
};

const isBacklog = computed(() => props.task?.state === TaskState.BACKLOG);
const hasStarted = computed(() => Boolean(props.task?.startedAt));
const isPostponed = computed(() => isBacklog.value && hasStarted.value);

const isEditableBacklog = computed(() => isBacklog.value && !hasStarted.value);
const canEditPrompt = isEditableBacklog;
const isStarted = computed(() => Boolean(props.task?.sessionId));
const isInput = computed(() => props.task?.state === TaskState.INPUT);

const runOver = computed(() => {
  const state = props.task?.state;
  return (
    state === TaskState.SETTLED ||
    state === TaskState.FAILED ||
    state === TaskState.COMPLETED ||
    state === TaskState.CANCELLED
  );
});

const sessionDeleted = computed(() => Boolean(props.task?.sessionDeletedAt));

const showPrompt = computed(() => !isStarted.value && !isPostponed.value);
const waitingReason = computed(() =>
  props.task ? taskWaitingReason(props.task) : "",
);
const waitingParts = computed(() => waitingReasonParts(waitingReason.value));
const promptHint = computed(() => {
  if (isPostponed.value) return "Started · follow up below to queue it again";
  if (isBacklog.value) return "Backlog · click to edit";
  if (waitingReason.value) return waitingReason.value;
  return "";
});

const showPendingPrompt = computed(
  () => !!props.task?.pendingPrompt && isStarted.value,
);
const isResting = computed(() => {
  const state = props.task?.state;
  return (
    state === TaskState.SETTLED ||
    state === TaskState.FAILED ||
    state === TaskState.COMPLETED ||
    state === TaskState.CANCELLED
  );
});
const pendingPromptHint = computed(() => {
  if (isPostponed.value) return "Started · follow up below to queue it again";
  if (isBacklog.value) return "Backlog · not queued yet";
  if (waitingReason.value) return waitingReason.value;
  if (isResting.value)
    return "Undelivered follow-up · the next run delivers it";
  return "";
});
const busy = computed(() => !props.task || props.task.runtime?.runInProgress);

const steer = computed(() => props.task?.runtime?.runInProgress === true);

const contextWindow = computed(() =>
  Number(props.taskUsage?.contextWindow ?? 0),
);

const composer = ref<HTMLElement | null>(null);
const promptInput = ref<InstanceType<typeof MarkdownEditor> | null>(null);
const textareaHeight = ref<number | null>(null);
const resizing = ref(false);

const queueOpen = ref(true);
watch(
  () => props.task?.id,
  () => {
    queueOpen.value = true;
  },
);
const textareaStyle = computed(() =>
  textareaHeight.value === null
    ? undefined
    : { height: `${textareaHeight.value}px` },
);

const MIN_TEXTAREA_HEIGHT = 72;
let resizeStartY = 0;
let resizeStartHeight = 0;

function startResize(event: PointerEvent) {
  if (event.button !== 0 || !props.canCompose) return;
  const el = promptInput.value?.$el;
  if (!el) return;
  event.preventDefault();
  resizing.value = true;
  resizeStartY = event.clientY;
  resizeStartHeight = el.offsetHeight;
  window.addEventListener("pointermove", onResizeMove);
  window.addEventListener("pointerup", stopResize, { once: true });
}

function onResizeMove(event: PointerEvent) {
  const column = composer.value?.parentElement;
  const max = column
    ? Math.max(MIN_TEXTAREA_HEIGHT, column.clientHeight - 120)
    : 480;
  const next = resizeStartHeight + (resizeStartY - event.clientY);
  textareaHeight.value = Math.min(Math.max(next, MIN_TEXTAREA_HEIGHT), max);
}

function stopResize() {
  resizing.value = false;
  window.removeEventListener("pointermove", onResizeMove);
}

onBeforeUnmount(() => {
  window.removeEventListener("pointermove", onResizeMove);
});

const queuedMessages = computed(() =>
  props.taskQueue.map((item) => ({
    ...item,
    sender: namedSender(item.sender, actorNames.value),
  })),
);

function queueAttachments(ids: string[]): Attachment[] {
  const byId = new Map(
    [...props.stagedAttachments, ...(props.task?.attachments ?? [])].map(
      (attachment) => [String(attachment.id), attachment],
    ),
  );
  return ids
    .map((id) => byId.get(id))
    .filter((attachment): attachment is Attachment => attachment !== undefined);
}
</script>

<style lang="postcss" scoped></style>
