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
.relative.flex.min-h-0.flex-1.flex-col
  //- one-line chat row that keeps its author, avatar, and timestamp.
  .bg-background.flex.shrink-0.items-center.gap-2.border-b.px-4.py-2(
    v-if="pinnedUser"
    data-testid="pinned-prompt"
  )
    .mx-auto.flex.w-full.max-w-3xl.min-w-0.items-center.justify-end.gap-2
      button.d-chat-bubble.d-chat-bubble-primary.min-w-0.px-3.text-sm(
        type="button"
        class="max-w-[80%] py-1.5 before:hidden"
        :aria-label="$t('taskConsole.scrollToMessage')"
        @click="scrollToUser(pinnedUser.id)"
      )
        span.flex.min-w-0.items-baseline.gap-2
          span.shrink-0.font-medium(v-if="pinnedSender") {{ senderName(pinnedSender) }}
          time.shrink-0.opacity-70(
            v-if="pinnedUser.at"
            :datetime="isoAt(pinnedUser.at)"
            :title="fullAt(pinnedUser.at)"
          ) {{ clockAt(pinnedUser.at) }}
          span.truncate(:title="pinnedUser.text") {{ pinnedUser.text }}
      ChatAvatar.self-center(
        v-if="pinnedSender && pinnedSender.kind !== 'system'"
        :sender="pinnedSender"
        :picture="pictureFor(pinnedSender)"
        :size="20"
      )

  .relative.min-h-0.flex-1.overflow-auto.p-4(
    ref="scroller"
    @scroll.passive="onScroll"
  )
    .mx-auto.flex.max-w-3xl.flex-col.gap-3(ref="content")
      .flex.justify-center.py-2(v-if="loadingOlder")
        Spinner.text-muted-foreground.size-4

      p.text-muted-foreground.py-2.text-center.text-xs(
        v-else-if="historyLoaded && !nextBefore && items.length"
      ) {{ $t("taskConsole.beginning") }}

      template(v-for="group in groups" :key="rowKey(group)")
        template(v-if="group.kind === 'single'")
          .d-chat.d-chat-start(
            v-if="group.item.kind === 'user' && group.item.system === true"
            :data-item-id="group.id"
            :data-message-id="group.item.messageId"
            data-testid="system-turn"
          )
            ChatAvatar(
              :sender="group.item.sender"
              :picture="pictureFor(group.item.sender)"
            )

            .d-chat-header.mb-1
              MessageSender(:sender="group.item.sender" :avatar="false")

            .d-chat-bubble(class="max-w-[80%]")
              MarkdownText.prose.prose-sm.max-w-none(
                class="prose-p:my-1 prose-pre:my-2"
                zoomable
                :style="INHERIT_PROSE"
                :text="group.item.text"
                @open-image="emit('openImage', $event)"
                @open-diagram="emit('openDiagram', $event)"
              )

            .d-chat-footer.mt-1.items-center.gap-2
              AttachmentChips(
                v-if="group.item.attachmentIds.length"
                :attachments="attachmentsFor(group.item.attachmentIds)"
                @open="emit('openAttachment', $event)"
              )

              time.text-muted-foreground(
                v-if="group.item.at"
                :datetime="isoAt(group.item.at)"
                :title="fullAt(group.item.at)"
              ) {{ clockAt(group.item.at) }}

              button.text-muted-foreground.rounded-md.p-1(
                type="button"
                class="hover:bg-muted hover:text-foreground"
                :aria-label="copiedId === group.id ? $t('taskConsole.copied') : $t('taskConsole.copyMessage')"
                :title="copiedId === group.id ? $t('taskConsole.copied') : $t('taskConsole.copyMessage')"
                @click="copyResponse(group.item.text, group.id)"
              )
                CheckIcon(v-if="copiedId === group.id" class="size-3.5")
                CopyIcon(v-else class="size-3.5")

          ConsoleUserMessage(
            v-else-if="group.item.kind === 'user'"
            :item="group.item"
            :picture="pictureFor(group.item.sender)"
            :attachments="attachmentsFor(group.item.attachmentIds)"
            :copied="copiedId === group.id"
            :pending="group.unsent !== undefined"
            @copy="copyResponse(group.item.text, group.id)"
            @open-attachment="emit('openAttachment', $event)"
            @open-image="emit('openImage', $event)"
            @open-diagram="emit('openDiagram', $event)"
            :data-item-id="group.unsent ? undefined : group.id"
            :data-message-id="group.unsent ? undefined : group.item.messageId"
            :data-unsent-key="group.unsent?.key"
            :data-row-key="rowKey(group)"
            :data-testid="group.unsent ? 'unsent-message' : undefined"
          )
            template(#pending-actions)
              Button(
                v-if="group.unsent?.cancellable"
                variant="ghost"
                size="xs"
                :aria-label="$t('taskConsole.cancelUnsentTitle')"
                :title="$t('taskConsole.cancelUnsentTitle')"
                @click="group.unsent && emit('cancelUnsent', group.unsent)"
                :disabled="group.unsent.busy"
                data-testid="unsent-cancel"
              )
                Spinner(v-if="group.unsent.busy" data-icon="inline-start")
                XIcon(v-else data-icon="inline-start")
                span {{ $t("taskConsole.cancelUnsent") }}

          ConsoleAgentMessage(
            v-else-if="group.item.kind === 'text'"
            :item="group.item"
            :sender="agentSender"
            :picture="agentPicture"
            :usage="runUsageByItem.get(group.item.id)"
            :copied="copiedId === group.id"
            :streaming="isStreamingText(group.item)"
            @copy="copyResponse(group.item.text, group.id)"
            @open-image="emit('openImage', $event)"
            @open-diagram="emit('openDiagram', $event)"
            :data-item-id="group.id"
          )

          p.text-muted-foreground.pl-11.text-xs.whitespace-pre-wrap(
            v-else
            :data-item-id="group.id"
          ) {{ group.item.text }}

        CompactionLine(
          v-else-if="group.kind === 'compaction'"
          :before-tokens="group.item.contextTokensBefore"
          :after-tokens="group.item.contextTokensAfter"
          :duration-ms="group.item.durationMs"
          :reason="group.item.reason"
          :at="group.item.endedAt"
          :data-item-id="group.id"
        )

        ConsoleActivity(
          v-else-if="group.kind === 'activity'"
          :group="group"
          :run="runState"
          :now="activityClock"
          :sender="agentSender"
          :picture="agentPicture"
          :model-name="modelName"
          :model-provider="modelProvider"
          :can-interrupt="canInterrupt"
          :interrupting="interrupting"
          @toggle="onActivityToggle(group.id, $event)"
          @interrupt="interruptToolCall($event)"
          :data-item-id="group.id"
        )

        .flex.w-full(
          v-else-if="group.kind === 'state'"
          :data-item-id="group.id"
          :data-state-key="group.change.key"
        )
          ConsoleNotice(
            :label="stateChangeLabel(group.change)"
            :icon="stateChangeIcon(group.change)"
            :hint="stateChangeTitle(group.change)"
            :reason="stateChangeReason(group.change)"
            :at="group.change.at"
            data-testid="state-change"
          )
            template(#actor)
              MessageSender(
                v-if="stateChangeActor(group.change)"
                :sender="stateChangeActor(group.change) ?? SYSTEM_SENDER"
                :picture="stateChangeActorPicture(group.change)"
                :avatar="false"
                data-testid="state-change-actor"
              )
              Badge.font-normal(
                v-else-if="stateChangeKindLabel(group.change)"
                variant="outline"
                class="h-4.5 px-1.5 text-[10px]"
                data-testid="state-change-actor"
              ) {{ stateChangeKindLabel(group.change) }}

        .d-chat.d-chat-start(
          v-else-if="group.kind === 'questionnaire-live'"
          data-testid="questionnaire-live"
        )
          ChatAvatar(:sender="agentSender" :picture="agentPicture")

          QuestionnaireWizard.w-full(
            :key="`closed-${wizardKey}`"
            v-if="questionnaire && questionnaireRetired"
            readonly
            cancelled
            :questions="questionnaire.questions"
            :submitting="false"
            data-test="questionnaire-closed"
          )

          QuestionnaireWizard.w-full(
            :key="wizardKey"
            v-else-if="questionnaire"
            :questions="questionnaire.questions"
            :submitting="wizardSubmitting"
            :errors="wizardErrors"
            :storage-key="wizardStorageKey"
            :blocked-reason="blockedReason"
            :agent-id="agentId"
            :model-config="modelConfig"
            :model-selection="modelSelection"
            :model-disabled="modelDisabled"
            :task-id="id"
            :project-id="projectId"
            @submit="submitWizard"
            @decline="declineWizard"
            @model-change="emit('modelChange', $event)"
          )

        .d-chat.d-chat-start(
          v-else-if="group.kind === 'dialog'"
          :data-request-id="group.request.requestId"
        )
          ChatAvatar(:sender="agentSender" :picture="agentPicture")

          ExtensionUiPrompt.w-full(
            :request="group.request"
            :busy="resolving[group.request.requestId]"
            :blocked-reason="blockedReason"
            @respond="(response) => answer(group.request, response)"
          )

        //- so it stays collapsed to one line until the reader opens it; the

        .d-chat.d-chat-start(
          v-else
          :data-item-id="group.id"
          :data-request-id="group.requestId"
        )
          ChatAvatar(:sender="agentSender" :picture="agentPicture")

          details.group.w-full.overflow-hidden.rounded-lg.border(
            data-test="questionnaire-review"
          )
            summary.flex.cursor-pointer.items-center.gap-2.px-3.py-2.text-xs.select-none
              ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
                class="group-open:rotate-90"
              )
              CheckIcon.size-3.shrink-0.text-emerald-600(
                v-if="!group.cancelled && !group.unavailable"
                class="dark:text-emerald-400"
              )
              XIcon.text-destructive.size-3.shrink-0(
                v-else-if="group.cancelled"
              )
              InfoIcon.text-muted-foreground.size-3.shrink-0(v-else)
              span.font-medium {{ reviewTitle(group) }}
              span.text-muted-foreground {{ reviewDetail(group) }}

            .border-t.p-2
              QuestionnaireWizard.w-full(
                :key="`review-${group.id}`"
                readonly
                :focus-question-id="focus?.questionId"
                :questions="group.questions"
                :review-answers="questionnaireAnswersFromReview(group.questions, group.answers)"
                :review-final-note="group.finalNote"
                :answered-by="group.answeredBy"
                :answered-by-picture="pictureForActor(group.answeredBy?.id)"
                :answered-at="group.answeredAt"
                :cancelled="group.cancelled"
                :unavailable="group.unavailable"
                :submitting="false"
              )

      .flex.flex-col.items-center.gap-3.py-10(
        v-if="!items.length && !groups.length && !unsent.length && !error"
      )
        Spinner.text-muted-foreground.size-5(v-if="busy")
        p.text-muted-foreground.text-sm {{ busy ? "The agent is starting…" : "Waiting for events…" }}

  .absolute.right-4.bottom-4.flex.flex-col.items-center.gap-2
    Button(
      v-if="openActivities.size"
      variant="outline"
      size="icon-sm"
      :title="$t('taskConsole.collapseReasoning')"
      @click="collapseActivities"
    )
      ChevronsDownUpIcon
    Button(
      variant="outline"
      size="icon-sm"
      :title="$t('taskConsole.previousItem')"
      @click="step(-1)"
    )
      ChevronUpIcon
    Button(
      variant="outline"
      size="icon-sm"
      :title="$t('taskConsole.nextItem')"
      @click="step(1)"
    )
      ChevronDownIcon
    Button(
      variant="outline"
      size="icon-sm"
      :title="$t('taskConsole.scrollToBottom')"
      @click="scrollToBottom"
      :disabled="stickToBottom"
    )
      ArrowDownIcon

  p.text-destructive.border-t.px-4.py-2.text-xs(v-if="error") {{ error }}
</template>

<script setup lang="ts">
import AttachmentChips from "@/components/attachments/AttachmentChips.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import MessageSender from "@/components/MessageSender.vue";
import ChatAvatar from "@/components/tasks/ChatAvatar.vue";
import CompactionLine from "@/components/tasks/CompactionLine.vue";
import ConsoleActivity from "@/components/tasks/ConsoleActivity.vue";
import ConsoleAgentMessage from "@/components/tasks/ConsoleAgentMessage.vue";
import ConsoleNotice from "@/components/tasks/ConsoleNotice.vue";
import ConsoleUserMessage from "@/components/tasks/ConsoleUserMessage.vue";
import ExtensionUiPrompt from "@/components/tasks/ExtensionUiPrompt.vue";
import QuestionnaireWizard from "@/components/tasks/QuestionnaireWizard.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { namedSender } from "@/lib/actors";
import {
  activeQuestionnaire,
  applyThinkingSpan,
  blockWaiting,
  findTool,
  groupItems,
  isBlankDelta,
  isThinkingStreaming,
  isToolInFlight,
  mergeStateChanges,
  mergeTool,
  newItem,
  pinnedUserIndex,
  prependEvents,
  runUsageByMessage,
  SYSTEM_SENDER,
  thinkingSpan,
  timestampMs,
  WAIT_RENDER_MS,
  type ActivityItem,
  type MessageSender as ChatSender,
  type Group,
  type Item,
  type NewItem,
  type PendingStep,
  type RenderActivity,
  type RunState,
  type RunUsage,
  type StateChange,
  type UnsentMessage,
} from "@/lib/console";
import { formatClock, formatInstant, isoInstant } from "@/lib/format";
import type { DiagramSource, InlineImage } from "@/lib/lightbox";
import { INHERIT_PROSE } from "@/lib/markdown";
import {
  multiSelectValue,
  parseQuestionnaireDialog,
  questionnaireAnswersFromReview,
  questionnaireResponseValue,
  type QuestionnaireAnswer,
  type QuestionnaireQuestion,
} from "@/lib/questionnaire";
import { agent } from "@/lib/sdk";
import {
  EMPTY_TASK_MODEL_SELECTION,
  taskStateFromStored,
  taskStateIcon,
  taskStateKey,
  type TaskModelSelection,
} from "@/lib/task";
import {
  PendingActivity_Kind,
  TaskStatus,
  ToolPhase,
  type Agent,
  type AgentUsageRun,
  type Attachment,
  type ExtensionUiRequest,
  type ExtensionUiResponse,
  type QuestionnaireAnswerInput,
  type TaskConfig,
  type TaskEvent,
} from "@cherrypi/sdk";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  ArrowDownIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  ChevronsDownUpIcon,
  ChevronUpIcon,
  CopyIcon,
  InfoIcon,
  XIcon,
} from "@lucide/vue";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    id: string;

    projectId?: string;
    busy?: boolean;

    waitingForInput?: boolean;

    attachments?: Attachment[];

    actorPictures?: Map<string, Attachment>;

    actorNames?: Map<string, string>;

    agentId?: string;

    agent?: Agent;

    modelConfig?: TaskConfig;

    modelName?: string;

    modelProvider?: string;

    modelSelection?: TaskModelSelection;

    modelDisabled?: boolean;

    usageRuns?: AgentUsageRun[];

    stateChanges?: StateChange[];

    runOver?: boolean;

    canAnswer?: boolean;

    canInterrupt?: boolean;

    unsent?: UnsentMessage[];

    history?: TaskEvent[];

    focus?: {
      messageId?: string;
      requestId?: string;
      questionId?: string;
    };
  }>(),
  {
    projectId: "",
    waitingForInput: undefined,
    attachments: () => [],
    actorPictures: () => new Map<string, Attachment>(),
    actorNames: () => new Map<string, string>(),
    agentId: "",
    agent: undefined,
    modelConfig: undefined,
    modelName: "",
    modelProvider: "",
    modelSelection: () => ({ ...EMPTY_TASK_MODEL_SELECTION }),
    modelDisabled: false,
    usageRuns: () => [],
    stateChanges: () => [],
    runOver: false,
    canAnswer: true,
    unsent: () => [],
    history: undefined,
  },
);

type Sender = NonNullable<Extract<Item, { kind: "user" }>["sender"]>;

function pictureFor(sender: Sender | undefined): Attachment | undefined {
  return sender?.id ? props.actorPictures.get(sender.id) : undefined;
}

function pictureForActor(actorId: string | undefined): Attachment | undefined {
  return actorId ? props.actorPictures.get(actorId) : undefined;
}

const blockedReason = computed(() => {
  if (props.canAnswer) {
    return undefined;
  }
  return (
    "Answering takes a lead of this task or an administrator. " +
    "You can read the questions here."
  );
});

function resolveSender(sender: Sender): Sender {
  return namedSender(sender, props.actorNames) ?? sender;
}

const agentSender = computed<ChatSender | undefined>(() => {
  const agent = props.agent;
  if (!agent) return undefined;
  return {
    id: agent.actorId || undefined,
    username: agent.username,
    name: agent.displayName || agent.username,
    kind: "agent",
  };
});

const agentPicture = computed<Attachment | undefined>(
  () => props.agent?.picture,
);

function senderName(sender: Sender | undefined): string {
  return sender?.name || sender?.username || "Unknown";
}

function clockAt(at: number | undefined): string {
  return formatClock(at);
}

function fullAt(at: number | undefined): string {
  return formatInstant(at);
}

function isoAt(at: number | undefined): string {
  return isoInstant(at);
}

const attachmentsById = computed(() => {
  const map = new Map<string, Attachment>();
  for (const attachment of props.attachments) {
    map.set(attachment.id.toString(), attachment);
  }
  return map;
});

function attachmentsFor(ids: string[]): Attachment[] {
  return ids
    .map((id) => attachmentsById.value.get(id))
    .filter((attachment): attachment is Attachment => attachment !== undefined);
}

const emit = defineEmits<{
  queue: [steering: { id: string; text: string }[]];

  cancelUnsent: [item: UnsentMessage];

  openAttachment: [attachment: Attachment];

  openImage: [image: InlineImage];

  openDiagram: [diagram: DiagramSource];

  modelChange: [selection: TaskModelSelection];

  liveTiming: [timing: LiveTiming];
}>();

export interface LiveTiming {
  toolRunning: boolean;

  modelWaiting: boolean;
}

const items = ref<Item[]>([]);

const questionnaire = computed(() => activeQuestionnaire(items.value));
const wizardAnswers = ref<QuestionnaireAnswer[] | null>(null);

const wizardFinalNote = ref("");
const wizardStep = ref(0);
const wizardAwaitingCustom = ref(false);
const wizardDeclined = ref(false);
const wizardErrors = ref("");
const wizardSubmitting = computed(() => wizardAnswers.value !== null);

const wizardDone = ref(false);

const wizardEngaged = ref(false);

const wizardActive = computed(
  () => questionnaire.value !== null && !wizardDone.value,
);

const retiredCallId = ref<string | undefined>(undefined);
const questionnaireRetired = computed(() => {
  if (props.runOver === true || wizardDeclined.value) return true;
  const active = questionnaire.value;
  return active !== null && retiredCallId.value === active.callId;
});
const showWizard = computed(
  () =>
    wizardActive.value &&
    !questionnaireRetired.value &&
    (props.waitingForInput === true ||
      (props.busy === true && wizardEngaged.value)),
);
const wizardQueue: ExtensionUiRequest[] = [];
const wizardSeen = new Set<string>();
let wizardWorking = false;

const wizardKey = ref(0);

let wizardCallId: string | undefined;
let wizardQuestionsJSON = "";

const wizardStorageKey = computed(() =>
  questionnaire.value ? `${props.id}:${questionnaire.value.callId}` : "",
);

async function dismissClosedDialog(request: ExtensionUiRequest) {
  try {
    await agent.respondToExtensionUi(props.id, request.requestId, {
      cancelled: true,
    });
  } catch {}
}

watch(
  () => props.runOver,
  (over) => {
    if (over !== true) return;
    retiredCallId.value = questionnaire.value?.callId;
    const pending = [...wizardQueue, ...dialogs.value];
    wizardQueue.length = 0;
    dialogs.value = [];
    wizardAnswers.value = null;
    wizardFinalNote.value = "";
    wizardErrors.value = "";
    wizardAwaitingCustom.value = false;
    wizardDeclined.value = true;
    wizardDone.value = false;
    wizardEngaged.value = false;
    for (const request of pending) dismissClosedDialog(request);
  },
);

watch(
  () => questionnaire.value?.callId,
  (callId) => {
    const active = questionnaire.value;
    const signature = active ? JSON.stringify(active.questions) : "";
    const previous = wizardCallId
      ? findTool(items.value, wizardCallId)
      : undefined;

    const resumed =
      callId !== undefined &&
      wizardAnswers.value !== null &&
      previous !== undefined &&
      previous.phase !== ToolPhase.FINISHED &&
      signature === wizardQuestionsJSON;

    wizardCallId = callId;
    wizardQuestionsJSON = signature;

    if (active && props.runOver === true) {
      retiredCallId.value = active.callId;
    }
    if (!resumed) {
      wizardKey.value += 1;
      wizardAnswers.value = null;
      wizardFinalNote.value = "";
      wizardStep.value = 0;
      wizardAwaitingCustom.value = false;
      wizardDeclined.value = false;
      wizardErrors.value = "";
      wizardDone.value = false;
      wizardEngaged.value = false;
      wizardQueue.length = 0;
      wizardSeen.clear();
    }

    if (active && dialogs.value.length) {
      const pending = dialogs.value.splice(0);
      for (const dialog of pending) {
        enqueueWizardDialog(dialog);
      }
    }
  },
  { flush: "sync" },
);

interface RenderQuestionnaire {
  id: number;
  kind: "questionnaire-live";
}

interface RenderSingle extends Extract<Group, { kind: "single" }> {
  unsent?: UnsentMessage;

  pendingKey?: string | number;
}

type RenderGroup =
  | RenderSingle
  | Extract<Group, { kind: "questionnaire" }>
  | Extract<Group, { kind: "compaction" }>
  | Extract<Group, { kind: "state" }>
  | RenderActivity
  | RenderQuestionnaire;

interface RenderDialog {
  id: number;
  kind: "dialog";
  request: ExtensionUiRequest;
}

type RenderRow = RenderGroup | RenderDialog;

const UNSENT_ROW_ID_BASE = -1_000_000;
const DIALOG_ROW_ID_BASE = -2_000_000;

const QUESTIONNAIRE_SLOT_ID = -1;

function pendingSlotId(): number {
  reservedId ??= allocateId();
  return reservedId;
}

function isStreamingText(item: Item): boolean {
  const last = items.value[items.value.length - 1];
  return (
    live.value &&
    !waitingForInput.value &&
    liveTextId.value === item.id &&
    last?.kind === "text" &&
    last.id === item.id
  );
}

function groupHoldsCall(group: RenderGroup, callId: string): boolean {
  return (
    group.kind === "activity" &&
    group.items.some((item) => item.kind === "tool" && item.callId === callId)
  );
}

const duplicatedMessageIds = computed(() => {
  const seen = new Set<string>();
  const duplicated = new Set<string>();
  for (const item of items.value) {
    if (item.kind !== "user" || !item.messageId) continue;
    if (seen.has(item.messageId)) duplicated.add(item.messageId);
    else seen.add(item.messageId);
  }
  return duplicated;
});

const unsentMatch = computed(() => {
  const shadowed = new Set<number>();
  const delivered = items.value.filter(
    (item): item is Extract<Item, { kind: "user" }> => item.kind === "user",
  );
  for (const row of props.unsent) {
    if (!row.messageId) continue;
    for (let i = delivered.length - 1; i >= 0; i--) {
      const candidate = delivered[i]!;
      if (candidate.messageId !== row.messageId) continue;
      if (shadowed.has(candidate.id)) continue;
      shadowed.add(candidate.id);
      break;
    }
  }
  return { shadowed };
});

const visibleItems = computed(() =>
  unsentMatch.value.shadowed.size
    ? items.value.filter((item) => !unsentMatch.value.shadowed.has(item.id))
    : items.value,
);

const dialogs = ref<ExtensionUiRequest[]>([]);

const groups = computed<RenderRow[]>(() => {
  const raw = groupItems(visibleItems.value);
  const out: RenderGroup[] = raw.flatMap((group, index): RenderGroup[] => {
    if (group.kind === "activity") {
      const items = group.items.filter(
        (item) => item.kind !== "wait" || item.durationMs >= WAIT_RENDER_MS,
      );
      return [
        {
          ...group,
          items,
          span: group.items,
          waiting: blockWaiting(group.items),
          running: isRunning(group, index === raw.length - 1),
        },
      ];
    }
    if (group.kind === "single" && group.item.kind === "user") {
      return [
        {
          ...group,
          item: { ...group.item, sender: resolveSender(group.item.sender) },
        },
      ];
    }
    return [group];
  });
  const callId = questionnaire.value?.callId;
  if (callId && (showWizard.value || questionnaireRetired.value)) {
    const slot: RenderQuestionnaire = {
      id: QUESTIONNAIRE_SLOT_ID,
      kind: "questionnaire-live",
    };
    const index = out.findIndex((group) => groupHoldsCall(group, callId));
    if (index >= 0) {
      out.splice(index + 1, 0, slot);
    } else {
      out.push(slot);
    }
  }

  const step = pending.value;
  if (showPending.value && step) {
    const last = out[out.length - 1];
    if (last && last.kind === "activity") {
      out[out.length - 1] = { ...last, pending: step, running: true };
    } else {
      out.push({
        id: pendingSlotId(),
        kind: "activity",
        items: [],
        span: [],
        waiting: { ms: 0, requests: 0, retries: 0, compactions: 0 },
        pending: step,
        running: true,
      });
    }
  }

  const drawn = out.filter(
    (group) =>
      group.kind !== "activity" || group.items.length > 0 || group.pending,
  );
  const rows: RenderRow[] = [
    ...mergeStateChanges(drawn, props.stateChanges, items.value),
  ];

  if (!showWizard.value && !questionnaireRetired.value) {
    dialogs.value.forEach((request, index) => {
      rows.push({ id: DIALOG_ROW_ID_BASE - index, kind: "dialog", request });
    });
  }
  props.unsent.forEach((row, index) => {
    rows.push({
      id: UNSENT_ROW_ID_BASE - index,
      kind: "single",
      item: {
        id: UNSENT_ROW_ID_BASE - index,
        kind: "user",
        text: row.text,
        attachmentIds: row.attachmentIds,
        sender: row.sender ?? SYSTEM_SENDER,
        system: false,
        messageId: row.messageId,
        at: undefined,
      },
      unsent: row,

      pendingKey:
        row.messageId && !duplicatedMessageIds.value.has(row.messageId)
          ? `msg:${row.messageId}`
          : `unsent:${row.key}`,
    });
  });
  return rows;
});

// each run reads as one line without duplicating its numbers across the blocks

const runs = computed<RunUsage[]>(() =>
  props.usageRuns.map((run) => ({
    provider: run.provider,
    model: run.model,
    inputTokens: Number(run.inputTokens),
    outputTokens: Number(run.outputTokens),
    cacheReadTokens: Number(run.cacheReadTokens),
    cacheWriteTokens: Number(run.cacheWriteTokens),
    cost: run.cost,
    activeMs: Number(run.activeMs),
    thinkingMs: Number(run.thinkingMs),
    modelWaitMs: Number(run.modelWaitMs),
    at: timestampMs(run.at),
  })),
);
const runUsageByItem = computed(() =>
  runUsageByMessage(items.value, runs.value),
);

const live = computed(
  () => props.busy === true || props.waitingForInput === true,
);

const waitingForInput = computed(() => props.waitingForInput === true);

const waitStartedAt = ref<number | undefined>(undefined);
const waitStartedFromMount = ref(false);
watch(
  () => props.waitingForInput,
  (waiting) => {
    if (waiting === true) {
      if (waitStartedAt.value === undefined) {
        waitStartedAt.value = Date.now();
        waitStartedFromMount.value = true;
      }
      return;
    }
    waitStartedAt.value = undefined;
    waitStartedFromMount.value = false;
  },
  { immediate: true },
);
const waitingCutoff = computed(() => {
  if (!waitingForInput.value) return undefined;
  if (!waitStartedFromMount.value) return waitStartedAt.value;

  const last = items.value[items.value.length - 1];
  return lastItemAt(last) ?? waitStartedAt.value;
});

const runState = computed<RunState>(() => ({
  live: live.value,
  waitingForInput: waitingForInput.value,
  pending: pending.value !== null,
  newestId: items.value[items.value.length - 1]?.id,
}));

function isInFlight(item: Item): boolean {
  return isToolInFlight(item, runState.value);
}

const interrupting = reactive(new Set<string>());

async function interruptToolCall(item: ActivityItem) {
  if (item.kind !== "tool" || interrupting.has(item.callId)) return;
  interrupting.add(item.callId);
  try {
    await agent.interruptTaskToolCall(props.id, item.callId);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    interrupting.delete(item.callId);
  }
}

function isThinkingRunning(item: ActivityItem): boolean {
  return isThinkingStreaming(item, runState.value);
}

function interruptOpenTools(at?: number) {
  const boundary = at ?? Date.now();
  for (const item of items.value) {
    if (
      item.kind === "tool" &&
      item.phase !== ToolPhase.FINISHED &&
      item.interrupted !== true
    ) {
      item.interrupted = true;
      item.endedAt ??= boundary;
    }
  }
}

function retireCallsBefore(stepSince: number) {
  for (const item of items.value) {
    if (
      item.kind !== "tool" ||
      item.phase === ToolPhase.FINISHED ||
      item.interrupted === true
    ) {
      continue;
    }
    if (item.startedAt === undefined || item.startedAt >= stepSince) continue;
    item.interrupted = true;
    item.endedAt ??= stepSince;
  }
}

function isRunning(
  group: Extract<Group, { kind: "activity" }>,
  isLast: boolean,
): boolean {
  return (
    group.items.some((item) => isInFlight(item) || isThinkingRunning(item)) ||
    (isLast && props.busy === true && props.waitingForInput !== true)
  );
}

const hasRunningTool = computed(() => items.value.some(isInFlight));

const hasRunningThinking = computed(() =>
  groups.value.some(
    (group) => group.kind === "activity" && group.items.some(isThinkingRunning),
  ),
);

const clock = ref(Date.now());
let clockTimer: ReturnType<typeof setInterval> | undefined;

const activityClock = computed(() => {
  if (waitingForInput.value) return waitingCutoff.value;
  return live.value ? clock.value : undefined;
});

const pending = ref<PendingStep | null>(null);

watch(pending, (step) => {
  if (step === null) reservedId = undefined;
});

const pendingFromStream = ref(false);

const PENDING_REVEAL_MS = 100;

const pendingRevealed = ref(false);
let pendingRevealTimer: ReturnType<typeof setTimeout> | undefined;
watch(
  pending,
  (step) => {
    if (pendingRevealTimer !== undefined) {
      clearTimeout(pendingRevealTimer);
      pendingRevealTimer = undefined;
    }
    pendingRevealed.value = false;
    if (!step) return;
    pendingRevealTimer = setTimeout(() => {
      pendingRevealed.value = true;
      pendingRevealTimer = undefined;
    }, PENDING_REVEAL_MS);
  },
  { immediate: true },
);

const showPending = computed(
  () =>
    pending.value !== null &&
    pendingRevealed.value &&
    props.busy === true &&
    props.waitingForInput !== true &&
    !hasRunningTool.value,
);

function groupKey(group: RenderGroup): string | number {
  if (group.kind === "state") return `state:${group.change.key}`;
  if (group.kind === "single" && group.item.kind === "user") {
    const id = group.item.messageId;
    if (id && !duplicatedMessageIds.value.has(id)) return `msg:${id}`;

    if (id) return `msg:${id}#${group.id}`;
  }
  return group.id;
}

function rowKey(row: RenderRow): string | number {
  if (row.kind === "single" && row.unsent) return row.pendingKey ?? row.id;
  if (row.kind === "dialog") return `dialog:${row.request.requestId}`;
  return groupKey(row);
}

function stateChangeLabel(change: StateChange): string {
  const from = taskStateFromStored(change.from);
  const to = taskStateFromStored(change.to);
  const label = (state: typeof from, raw: string) =>
    state === undefined
      ? raw || t("taskState.unknown")
      : t(taskStateKey(state));
  return `${label(from, change.from)} → ${label(to, change.to)}`;
}

function stateChangeIcon(change: StateChange) {
  return taskStateIcon(taskStateFromStored(change.to));
}

function stateChangeReason(change: StateChange): string {
  if (change.trigger?.kind === "stop") return t("taskConsole.stoppedBy");
  return change.reason;
}

function stateChangeActor(change: StateChange): ChatSender | undefined {
  const actor = change.trigger?.actor ?? change.actor;
  const id = change.trigger?.actorId || change.actorId;
  if (!actor || actor === "system") return undefined;

  const directoryName = id ? props.actorNames.get(id) : undefined;
  if (!directoryName) return undefined;
  return {
    id: id || undefined,
    username: "",
    name: directoryName,
    kind: actor,
  };
}

function stateChangeActorPicture(change: StateChange): Attachment | undefined {
  return pictureForActor(
    change.trigger?.actorId || change.actorId || undefined,
  );
}

function stateChangeKindLabel(change: StateChange): string {
  const actor = change.trigger?.actor ?? change.actor;
  switch (actor) {
    case "user":
      return t("taskConsole.user");
    case "agent":
      return t("admin.agents.columnAgent");
    case "plugin":
      return t("taskConsole.plugin");
    default:
      return t("taskConsole.system");
  }
}

function stateChangeTitle(change: StateChange): string {
  const parts = [stateChangeLabel(change)];
  const cause = stateChangeReason(change);
  const actor = change.trigger?.actor || change.actor;
  if (change.trigger) {
    const name = stateChangeActor(change)?.name;
    parts.push(name ? `${cause} ${name}` : cause);
  } else {
    if (cause) parts.push(cause);
    if (actor) parts.push(`by ${actor}`);
  }
  if (change.at) parts.push(formatInstant(change.at));
  return parts.filter(Boolean).join(" · ");
}

/** The one-line outcome of an answered questionnaire. */
function reviewTitle(group: Extract<Group, { kind: "questionnaire" }>): string {
  if (group.unavailable) return "Questionnaire not shown";
  if (group.cancelled) return "Questionnaire declined";
  return "Questionnaire answered";
}

function reviewDetail(
  group: Extract<Group, { kind: "questionnaire" }>,
): string {
  const count = group.questions.length;
  return `· ${count} question${count === 1 ? "" : "s"}`;
}

function clearPending() {
  pending.value = null;
}

function lastItemAt(item: Item | undefined): number | undefined {
  if (!item) return undefined;
  if (
    item.kind === "thinking" ||
    item.kind === "tool" ||
    item.kind === "wait"
  ) {
    return item.endedAt ?? item.startedAt;
  }
  return item.at;
}

function derivePending() {
  if (pendingFromStream.value || pending.value || props.busy !== true) return;
  const last = items.value[items.value.length - 1];

  if (!last) return;
  if (last.kind === "tool" && last.phase !== ToolPhase.FINISHED) return;
  pending.value = {
    kind: PendingActivity_Kind.MODEL_REQUEST,
    since: lastItemAt(last) ?? Date.now(),
    model: "",
    provider: "",
    error: "",
    attempt: 0,
    maxAttempts: 0,
    reason: "",
  };
}

const clockNeeded = computed(
  () =>
    hasRunningTool.value || hasRunningThinking.value || pending.value !== null,
);

watch(
  clockNeeded,
  (needed) => {
    if (needed && clockTimer === undefined) {
      clock.value = Date.now();
      clockTimer = setInterval(() => {
        clock.value = Date.now();
      }, 1000);
    } else if (!needed && clockTimer !== undefined) {
      clearInterval(clockTimer);
      clockTimer = undefined;
    }
  },
  { immediate: true },
);

const liveTiming = computed<LiveTiming>(() => ({
  toolRunning: hasRunningTool.value,
  modelWaiting: showPending.value,
}));
watch(liveTiming, (timing) => emit("liveTiming", timing), {
  immediate: true,
  deep: true,
});

watch(
  () => props.busy,
  (busy) => {
    if (busy !== true) clearPending();
  },
);

const openActivities = reactive(new Set<number>());

const autoOpened = new Set<number>();
watch(
  () => groups.value,
  async (rendered) => {
    const fresh = rendered.filter(
      (group): group is RenderActivity =>
        group.kind === "activity" &&
        group.pending !== undefined &&
        group.items.length === 0 &&
        !autoOpened.has(group.id),
    );
    if (!fresh.length) return;
    await nextTick();
    const el = scroller.value;
    if (!el) return;
    for (const group of fresh) {
      autoOpened.add(group.id);
      const node = el.querySelector<HTMLDetailsElement>(
        `details[data-activity-id="${group.id}"]`,
      );
      if (!node) continue;
      node.open = true;
      openActivities.add(group.id);
    }
  },
  { immediate: true },
);

const resolving = reactive<Record<string, boolean>>({});

const copiedId = ref<number | null>(null);
let copiedTimer: ReturnType<typeof setTimeout> | undefined;

async function copyResponse(text: string, id: number) {
  try {
    await navigator.clipboard.writeText(text);
    copiedId.value = id;
    clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copiedId.value = null;
    }, 1500);
  } catch {
    toast.error("Copy failed");
  }
}

const error = ref("");
const scroller = ref<HTMLDivElement | null>(null);
const content = ref<HTMLDivElement | null>(null);
let resizeObserver: ResizeObserver | undefined;

const stickToBottom = ref(true);

const pinnedUser = ref<Extract<Item, { kind: "user" }> | undefined>();

const pinnedSender = computed(() =>
  pinnedUser.value ? resolveSender(pinnedUser.value.sender) : undefined,
);

function updatePinnedUser() {
  const el = scroller.value;

  const users = items.value.filter(
    (item): item is Extract<Item, { kind: "user" }> =>
      item.kind === "user" && item.system !== true,
  );
  if (!el || !users.length) {
    pinnedUser.value = undefined;
    return;
  }
  const offsets = users.map((user) => {
    const node = el.querySelector<HTMLElement>(`[data-item-id="${user.id}"]`);
    return node?.offsetTop ?? 0;
  });
  pinnedUser.value = users[pinnedUserIndex(offsets, el.scrollTop)];
}

const historyLoaded = ref(false);
const loadingOlder = ref(false);
const nextBefore = ref("");
const liveQueue: TaskEvent[] = [];

let nextId = 0;

const liveTextId = ref<number | undefined>(undefined);

let reservedId: number | undefined;

function allocateId(adoptReserved = false): number {
  if (adoptReserved && reservedId !== undefined) {
    const id = reservedId;
    reservedId = undefined;
    return id;
  }
  return nextId++;
}

let streamController: AbortController | undefined;
let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
let reconnectAttempt = 0;
let stopped = false;

const RECONNECT_CAP_MS = 30_000;

function startStream() {
  streamController?.abort();
  const stream = new AbortController();
  streamController = stream;
  void agent
    .watchTask(props.id, handleEvent, { signal: stream.signal })
    .catch((err) => {
      if (stream.signal.aborted) return;
      if (isPermanentStreamError(err)) {
        error.value = err.message;
        return;
      }
      scheduleReconnect(err);
    });
}

function isPermanentStreamError(err: unknown): err is ConnectError {
  return (
    err instanceof ConnectError &&
    (err.code === Code.FailedPrecondition ||
      err.code === Code.NotFound ||
      err.code === Code.PermissionDenied)
  );
}

function scheduleReconnect(err: unknown) {
  if (stopped || reconnectTimer) {
    return;
  }
  console.warn("[console] task stream interrupted", err);
  reconnectAttempt += 1;
  error.value = "Connection lost — reconnecting…";
  const delay = Math.min(1000 * 2 ** (reconnectAttempt - 1), RECONNECT_CAP_MS);
  reconnectTimer = setTimeout(resync, delay);
}

function resync() {
  reconnectTimer = undefined;
  if (stopped) {
    return;
  }
  error.value = "";
  dropDeltas();
  items.value = [];
  liveQueue.length = 0;
  nextId = 0;

  reservedId = undefined;
  autoOpened.clear();
  nextBefore.value = "";
  historyLoaded.value = false;
  dialogs.value = [];

  pending.value = null;
  pendingFromStream.value = false;
  startStream();
  void loadHistory();
}

function push(item: NewItem) {
  items.value.push({ ...item, id: allocateId(isActivityItem(item)) } as Item);
  if (stickToBottom.value) {
    void scrollToBottom();
  }
}

function isActivityItem(item: NewItem): boolean {
  return (
    item.kind === "thinking" || item.kind === "tool" || item.kind === "wait"
  );
}

function pushWait(item: NewItem) {
  if (item.kind !== "wait" || item.startedAt === undefined) {
    push(item);
    return;
  }

  const last = items.value[items.value.length - 1];
  const lastStart =
    last?.kind === "text"
      ? last.at
      : last?.kind === "thinking"
        ? last.startedAt
        : undefined;
  if (
    last &&
    (last.kind === "text" || last.kind === "thinking") &&
    lastStart !== undefined &&
    lastStart >= item.startedAt
  ) {
    items.value.splice(items.value.length - 1, 0, {
      ...item,
      id: allocateId(),
    } as Item);
    return;
  }
  push(item);
}

function appendText(
  kind: "text" | "thinking",
  delta: string,
  startedAt?: number,
  endedAt?: number,
) {
  const last = items.value[items.value.length - 1];
  if (last && last.kind === kind) {
    last.text += delta;
    if (last.kind === "thinking") {
      last.startedAt ??= startedAt;
      last.endedAt = endedAt ?? last.endedAt ?? startedAt;
    }
  } else if (isBlankDelta(delta)) {
    return;
  } else if (kind === "thinking") {
    push({
      kind: "thinking",
      text: delta,
      startedAt,
      endedAt: endedAt ?? startedAt,
    });
    return;
  } else {
    push({ kind: "text", text: delta, at: startedAt });
  }
  if (kind === "text") {
    const appended = items.value[items.value.length - 1];
    liveTextId.value = appended?.kind === "text" ? appended.id : undefined;
  }
  if (stickToBottom.value) {
    void scrollToBottom();
  }
}

const DELTA_FLUSH_MS = 50;

interface PendingDelta {
  kind: "text" | "thinking";
  text: string;

  startedAt?: number;
  endedAt?: number;
}

let pendingDeltas: PendingDelta[] = [];
let deltaTimer: ReturnType<typeof setTimeout> | undefined;

function flushDeltas() {
  if (deltaTimer !== undefined) {
    clearTimeout(deltaTimer);
    deltaTimer = undefined;
  }
  if (!pendingDeltas.length) {
    return;
  }
  const pending = pendingDeltas;
  pendingDeltas = [];
  for (const delta of pending) {
    appendText(delta.kind, delta.text, delta.startedAt, delta.endedAt);
  }

  clearPending();
}

function queueDelta(kind: "text" | "thinking", text: string, at?: number) {
  const last = pendingDeltas[pendingDeltas.length - 1];
  if (last && last.kind === kind) {
    last.text += text;
    last.endedAt = at ?? last.endedAt;
  } else {
    pendingDeltas.push({ kind, text, startedAt: at, endedAt: at });
  }
  if (deltaTimer === undefined) {
    deltaTimer = setTimeout(flushDeltas, DELTA_FLUSH_MS);
  }
}

function dropDeltas() {
  if (deltaTimer !== undefined) {
    clearTimeout(deltaTimer);
    deltaTimer = undefined;
  }
  pendingDeltas = [];
  liveTextId.value = undefined;
}

async function scrollToBottom() {
  stickToBottom.value = true;
  await nextTick();
  pinToBottom(true);
  updatePinnedUser();
}

function pinToBottom(force = false) {
  const el = scroller.value;
  if (!el) return;
  if (!force && wizardActive.value) {
    el.scrollTop = questionnaireTop(el);
    return;
  }
  el.scrollTop = el.scrollHeight;
}

function questionnaireTop(el: HTMLElement): number {
  const node = el.querySelector<HTMLElement>(
    '[data-testid="questionnaire-live"]',
  );
  if (!node) return el.scrollHeight;
  const atBottom = node.offsetTop + node.offsetHeight - el.clientHeight;
  return node.offsetHeight <= el.clientHeight ? atBottom : node.offsetTop - 8;
}

function onScroll() {
  const el = scroller.value;
  if (!el) return;
  stickToBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
  updatePinnedUser();
  if (el.scrollTop < 80) {
    void loadOlder();
  }
}

function scrollToUser(id: number) {
  const el = scroller.value;
  const node = el?.querySelector<HTMLElement>(`[data-item-id="${id}"]`);
  if (el && node) {
    el.scrollTo({ top: node.offsetTop - 8, behavior: "smooth" });
  }
}

const focusedKey = ref("");
async function focusSource() {
  const focus = props.focus;
  if (!focus || (!focus.messageId && !focus.requestId)) return;
  const key = [focus.messageId, focus.requestId, focus.questionId].join("|");
  if (focusedKey.value === key) return;
  await nextTick();
  const el = scroller.value;
  if (!el) return;
  let node: HTMLElement | null = null;
  if (focus.messageId) {
    node = el.querySelector<HTMLElement>(
      `[data-message-id="${focus.messageId}"]`,
    );
  }
  if (!node && focus.requestId) {
    node = el.querySelector<HTMLElement>(
      `[data-request-id="${focus.requestId}"]`,
    );
    if (node && focus.questionId) {
      const details = node.querySelector<HTMLDetailsElement>("details");
      if (details) details.open = true;
      await nextTick();
      const question = node.querySelector<HTMLElement>(
        `[data-question-id="${focus.questionId}"]`,
      );
      if (question) node = question;
    }
  }
  if (!node) return;
  focusedKey.value = key;
  node.scrollIntoView({ block: "center", behavior: "smooth" });
  const highlight = [
    "ring-2",
    "ring-primary",
    "ring-offset-2",
    "ring-offset-background",
    "rounded-lg",
  ];
  node.classList.add(...highlight);
  const target = node;
  setTimeout(() => target.classList.remove(...highlight), 2500);
}

watch(
  () => props.focus,
  () => {
    focusedKey.value = "";
    void focusSource();
  },
  { deep: true },
);

watch(
  () => [historyLoaded.value, items.value.length],
  () => {
    if (historyLoaded.value && props.focus) void focusSource();
  },
);

onMounted(() => {
  void focusSource();
});

function onActivityToggle(id: number, event: Event) {
  const activity = event.target as HTMLDetailsElement;
  if (activity.open) {
    openActivities.add(id);

    activity
      .querySelectorAll<HTMLDetailsElement>(
        'details[data-testid="thinking-block"]',
      )
      .forEach((block) => {
        block.open = block.dataset.thinkingRunning === "true";
      });
  } else {
    openActivities.delete(id);

    activity
      .querySelectorAll<HTMLDetailsElement>(
        'details[data-testid="thinking-block"]',
      )
      .forEach((block) => {
        block.open = false;
      });
  }
}

function collapseActivities() {
  scroller.value
    ?.querySelectorAll<HTMLDetailsElement>("details[data-activity-id][open]")
    .forEach((node) => {
      node.open = false;
    });
  openActivities.clear();
}

function step(direction: 1 | -1) {
  const el = scroller.value;
  if (!el) return;
  const els = Array.from(el.querySelectorAll<HTMLElement>("[data-item-id]"));
  if (!els.length) return;

  let current = els.findIndex((node) => node.offsetTop >= el.scrollTop + 1);
  if (current === -1) {
    current = els.length - 1;
  }
  const target = Math.min(Math.max(current + direction, 0), els.length - 1);
  const node = els[target];
  if (node) {
    el.scrollTo({ top: node.offsetTop - 8, behavior: "smooth" });
  }
}

async function loadHistory() {
  try {
    const page = await agent.getTaskHistory(props.id);
    items.value = prependEvents(items.value, page.events, () => allocateId());
    nextBefore.value = page.nextBefore;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    historyLoaded.value = true;
    for (const event of liveQueue.splice(0)) {
      applyEvent(event);
    }

    if (!live.value)
      interruptOpenTools(lastItemAt(items.value[items.value.length - 1]));

    derivePending();
    await scrollToBottom();
    if ((scroller.value?.scrollTop ?? 0) < 80) {
      void loadOlder();
    }
  }
}

async function loadOlder() {
  const el = scroller.value;
  if (!el || loadingOlder.value || !nextBefore.value || !historyLoaded.value) {
    return;
  }
  loadingOlder.value = true;
  const previousHeight = el.scrollHeight;
  try {
    const page = await agent.getTaskHistory(props.id, {
      before: nextBefore.value,
    });
    items.value = prependEvents(items.value, page.events, () => allocateId());
    nextBefore.value = page.nextBefore;
    await nextTick();
    el.scrollTop += el.scrollHeight - previousHeight;
    updatePinnedUser();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loadingOlder.value = false;

    if (el.scrollTop < 80 && nextBefore.value) {
      void loadOlder();
    }
  }
}

function applyEvent(event: TaskEvent) {
  const payload = event.event;

  if (
    payload.case !== "text" &&
    payload.case !== "thinking" &&
    payload.case !== "modelWait"
  ) {
    flushDeltas();
  }

  if (payload.case !== "text") {
    liveTextId.value = undefined;
  }
  switch (payload.case) {
    case "text":
      clearPending();
      queueDelta("text", payload.value.delta, timestampMs(event.at));
      break;
    case "userMessage": {
      const item = newItem(event);
      if (item) push(item);
      break;
    }
    case "pendingMessages":
      emit(
        "queue",
        payload.value.queuedSteering.length
          ? payload.value.queuedSteering.map((item) => ({
              id: item.id,
              text: item.text,
            }))
          : payload.value.steering.map((text) => ({ id: "", text })),
      );
      break;
    case "thinking": {
      const span = thinkingSpan(event);
      if (span) {
        flushDeltas();
        applyThinkingSpan(items.value, span);
        break;
      }

      queueDelta("thinking", payload.value.delta, timestampMs(event.at));
      break;
    }
    case "tool": {
      clearPending();
      const item = newItem(event);
      if (item?.kind !== "tool") break;
      const existing = findTool(items.value, item.callId);
      if (existing) {
        mergeTool(existing, item);
      } else {
        push(item);
      }
      break;
    }
    case "log": {
      const item = newItem(event);
      if (item) push(item);
      break;
    }
    case "modelWait": {
      const item = newItem(event);
      if (item?.kind === "wait") {
        clearPending();
        pushWait(item);
      }
      break;
    }
    case "runStarted":
      interruptOpenTools(timestampMs(event.at));
      clearPending();
      break;
    case "runSettled":
      interruptOpenTools(timestampMs(event.at));
      clearPending();
      break;
    case "statusChanged": {
      const status = payload.value.status;
      if (status === TaskStatus.FAILED || status === TaskStatus.DONE) {
        interruptOpenTools(timestampMs(event.at));
        clearPending();
      }
      break;
    }
    case "pendingActivity": {
      pendingFromStream.value = true;
      const state = payload.value;
      if (state.kind === PendingActivity_Kind.UNSPECIFIED) {
        pending.value = null;
        break;
      }

      const since = timestampMs(event.at) ?? Date.now();
      if (state.kind === PendingActivity_Kind.MODEL_REQUEST) {
        retireCallsBefore(since);
      }
      pending.value = {
        kind: state.kind,
        since,
        model: state.model,
        provider: state.provider,
        error: state.error,
        attempt: state.attempt,
        maxAttempts: state.maxAttempts,
        reason: state.reason,
      };
      break;
    }
    case "extensionUi": {
      const request = payload.value;
      if (isDialogMethod(request.method)) {
        if (props.runOver === true) {
          dismissClosedDialog(request);
          break;
        }
        if (wizardActive.value) {
          enqueueWizardDialog(request);
          void scrollToBottom();
        } else if (
          !dialogs.value.some((d) => d.requestId === request.requestId)
        ) {
          dialogs.value.push(request);
          void scrollToBottom();
        }
      } else if (request.method === "notify") {
        toast(request.message);
      }
      break;
    }
  }
}

function handleEvent(event: TaskEvent) {
  if (reconnectAttempt > 0) {
    reconnectAttempt = 0;
    error.value = "";
  }
  if (!historyLoaded.value) {
    liveQueue.push(event);
    return;
  }
  applyEvent(event);
}

function isDialogMethod(method: string): boolean {
  return ["select", "confirm", "input", "editor"].includes(method);
}

async function answer(
  request: ExtensionUiRequest,
  response: ExtensionUiResponse,
) {
  resolving[request.requestId] = true;
  try {
    await agent.respondToExtensionUi(props.id, request.requestId, response);
    dialogs.value = dialogs.value.filter(
      (d) => d.requestId !== request.requestId,
    );
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    delete resolving[request.requestId];
  }
}

function enqueueWizardDialog(request: ExtensionUiRequest) {
  if (wizardSeen.has(request.requestId)) {
    return;
  }
  wizardSeen.add(request.requestId);
  wizardEngaged.value = true;
  wizardQueue.push(request);
  if (wizardDeclined.value) {
    void cancelWizardDialog(request);
  } else if (wizardAnswers.value) {
    void drainWizard();
  }
}

async function cancelWizardDialog(request: ExtensionUiRequest) {
  try {
    await agent.respondToExtensionUi(props.id, request.requestId, {
      cancelled: true,
    });
  } catch (err) {
    wizardErrors.value = err instanceof Error ? err.message : String(err);
  }
}

async function drainWizard() {
  if (wizardWorking) {
    return;
  }
  wizardWorking = true;
  try {
    while (
      !wizardDeclined.value &&
      wizardAnswers.value &&
      wizardQueue.length > 0
    ) {
      const active = questionnaire.value;
      const request = wizardQueue[0];
      if (!active || !request) {
        break;
      }
      if (isBatchDialog(request)) {
        if (!(await respondWizardBatch(request, active.questions))) break;
        wizardQueue.shift();
        wizardDone.value = true;
        break;
      }
      if (wizardStep.value >= active.questions.length) {
        break;
      }
      const question = active.questions[wizardStep.value];
      if (!question) {
        break;
      }
      if (!(await respondWizard(request, question))) break;
      wizardQueue.shift();
    }
    const active = questionnaire.value;
    if (
      active &&
      wizardAnswers.value &&
      !wizardAwaitingCustom.value &&
      wizardStep.value >= active.questions.length
    ) {
      wizardDone.value = true;
    }
  } finally {
    wizardWorking = false;
  }

  if (!wizardDone.value && wizardAnswers.value !== null && wizardErrors.value) {
    wizardAnswers.value = null;
  }
}

function isBatchDialog(request: ExtensionUiRequest): boolean {
  return parseQuestionnaireDialog(request) !== null;
}

async function respondWizardBatch(
  request: ExtensionUiRequest,
  questions: QuestionnaireQuestion[],
): Promise<boolean> {
  const answers = wizardAnswers.value;
  if (!answers) {
    return false;
  }
  try {
    await agent.respondToExtensionUi(
      props.id,
      request.requestId,
      {
        value: questionnaireResponseValue(
          questions,
          answers,
          wizardFinalNote.value,
        ),
      },
      submittedQuestionnaire(),
    );
    return true;
  } catch (err) {
    wizardErrors.value = err instanceof Error ? err.message : String(err);
    return false;
  }
}

async function respondWizard(
  request: ExtensionUiRequest,
  question: QuestionnaireQuestion,
): Promise<boolean> {
  const answer = wizardAnswers.value?.[wizardStep.value];
  if (!answer) {
    return false;
  }

  const resumeAnswers =
    wizardStep.value === 0 ? submittedQuestionnaire() : undefined;
  try {
    if (wizardAwaitingCustom.value) {
      await agent.respondToExtensionUi(
        props.id,
        request.requestId,
        { value: answer.custom },
        resumeAnswers,
      );
      wizardAwaitingCustom.value = false;
      wizardStep.value += 1;
      return true;
    }
    if (question.multiSelect) {
      await agent.respondToExtensionUi(
        props.id,
        request.requestId,
        { value: multiSelectValue(answer) },
        resumeAnswers,
      );
      wizardStep.value += 1;
      return true;
    }
    const options = request.options;
    if (answer.customSelected) {
      if (options.length > question.options.length) {
        await agent.respondToExtensionUi(
          props.id,
          request.requestId,
          { value: options[options.length - 1] ?? "" },
          resumeAnswers,
        );
        wizardAwaitingCustom.value = true;
      } else {
        await agent.respondToExtensionUi(
          props.id,
          request.requestId,
          { value: answer.custom },
          resumeAnswers,
        );
        wizardStep.value += 1;
      }
      return true;
    }
    await agent.respondToExtensionUi(
      props.id,
      request.requestId,
      { value: options[answer.selected[0] ?? 0] ?? "" },
      resumeAnswers,
    );
    wizardStep.value += 1;
    return true;
  } catch (err) {
    wizardErrors.value = err instanceof Error ? err.message : String(err);
    return false;
  }
}

function submittedQuestionnaire(): QuestionnaireAnswerInput[] | undefined {
  const active = questionnaire.value;
  const answers = wizardAnswers.value;
  if (!active || !answers) {
    return undefined;
  }
  const pairs = active.questions.map((question, index) => {
    const answer = answers[index];
    return {
      question: question.question,
      answer: answer ? answerLabel(question, answer) : "",
    };
  });

  const finalNote = wizardFinalNote.value.trim();
  if (finalNote) {
    pairs.push({ question: "Final note", answer: finalNote });
  }
  return pairs;
}

function answerLabel(
  question: QuestionnaireQuestion,
  answer: QuestionnaireAnswer,
): string {
  const base = answer.customSelected
    ? answer.custom
    : answer.selected
        .map((index) => question.options[index]?.label ?? "")
        .filter(Boolean)
        .join(", ");
  const notes = answer.notes.trim();
  return notes.length > 0 ? `${base} (note: ${notes})` : base;
}

function submitWizard(answers: QuestionnaireAnswer[], finalNote: string) {
  wizardErrors.value = "";

  if (wizardQueue.length === 0) {
    wizardAnswers.value = null;
    wizardErrors.value =
      "This questionnaire is no longer waiting for an answer. Send a follow-up to continue the task.";
    return;
  }
  wizardAnswers.value = answers;
  wizardFinalNote.value = finalNote;
  wizardEngaged.value = true;
  void drainWizard();
}

function declineWizard() {
  wizardDeclined.value = true;
  wizardAnswers.value = null;
  wizardFinalNote.value = "";
  wizardAwaitingCustom.value = false;
  const pending = wizardQueue.shift();
  if (pending) {
    void cancelWizardDialog(pending);
  }
}

async function seedHistory(events: TaskEvent[]) {
  historyLoaded.value = true;
  for (const event of events) {
    applyEvent(event);
  }
  flushDeltas();
  if (!live.value) {
    interruptOpenTools(lastItemAt(items.value[items.value.length - 1]));
  }
  derivePending();
  await scrollToBottom();
}

onMounted(() => {
  resizeObserver = new ResizeObserver(() => {
    if (stickToBottom.value) {
      pinToBottom();
    }
    updatePinnedUser();
  });
  if (content.value) {
    resizeObserver.observe(content.value);
  }
  if (props.history) {
    void seedHistory(props.history);
    return;
  }

  startStream();
  void loadHistory();
});

onBeforeUnmount(() => {
  stopped = true;
  dropDeltas();
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
  }
  streamController?.abort();
  resizeObserver?.disconnect();
  clearTimeout(copiedTimer);
  if (clockTimer !== undefined) clearInterval(clockTimer);
  if (pendingRevealTimer !== undefined) clearTimeout(pendingRevealTimer);
});
</script>

<i18n lang="yaml">
en:
  taskConsole:
    stoppedBy: "stopped by"
    user: "User"
    plugin: "Plugin"
    system: "System"
    beginning: "Beginning of conversation"
    scrollToMessage: "Scroll to this message"
    collapseReasoning: "Collapse reasoning"
    previousItem: "Previous item"
    nextItem: "Next item"
    cancelUnsent: "Cancel"
    cancelUnsentTitle: "Cancel this message: the agent has not read it yet"
    scrollToBottom: "Scroll to bottom"
ru:
  taskConsole:
    stoppedBy: "остановлено пользователем"
    user: "Пользователь"
    plugin: "Плагин"
    system: "Система"
    beginning: "Начало переписки"
    scrollToMessage: "Перейти к этому сообщению"
    collapseReasoning: "Свернуть рассуждение"
    previousItem: "Предыдущий элемент"
    nextItem: "Следующий элемент"
    cancelUnsent: "Отменить"
    cancelUnsentTitle: "Отменить это сообщение: агент его ещё не прочитал"
    scrollToBottom: "Прокрутить вниз"
</i18n>
