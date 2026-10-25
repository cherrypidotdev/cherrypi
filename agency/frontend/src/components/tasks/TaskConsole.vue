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
  button.bg-background.w-full.shrink-0.border-b.px-4.py-2.text-left(
    v-if="pinnedUser"
    type="button"
    aria-label="Scroll to this message"
    @click="scrollToUser(pinnedUser.id)"
  )
    .mx-auto.flex.max-w-3xl.justify-end
      .bg-primary.text-primary-foreground.rounded-lg.px-3.py-2.text-sm(
        class="max-w-[80%]"
      )
        span.block.truncate(:title="pinnedUser.text") {{ pinnedUser.text }}
  .relative.min-h-0.flex-1.overflow-auto.p-4(
    ref="scroller"
    @scroll.passive="onScroll"
  )
    .mx-auto.flex.max-w-3xl.flex-col.gap-3(ref="content")
      .flex.justify-center.py-2(v-if="loadingOlder")
        Spinner.text-muted-foreground.size-4
      p.text-muted-foreground.py-2.text-center.text-xs(
        v-else-if="historyLoaded && !nextBefore && items.length"
      ) Beginning of conversation

      template(v-for="group in groups" :key="group.id")
        template(v-if="group.kind === 'single'")
          .flex.flex-col.items-end.gap-1(
            v-if="group.item.kind === 'user'"
            class="group/user"
            :data-item-id="group.id"
          )
            MessageSender(
              v-if="group.item.sender"
              :sender="group.item.sender"
              :picture="pictureFor(group.item.sender)"
            )
            .bg-primary.text-primary-foreground.rounded-lg.px-3.py-2.text-sm(
              class="max-w-[80%]"
            )
              MarkdownText.prose.prose-sm.max-w-none(
                class="prose-p:my-1 prose-pre:my-2"
                :style="inheritProse"
                :text="group.item.text"
              )
            AttachmentChips(
              v-if="group.item.attachmentIds.length"
              :attachments="attachmentsFor(group.item.attachmentIds)"
              @open="emit('openAttachment', $event)"
            )

            button.text-muted-foreground.rounded-md.p-1.opacity-0.transition-opacity(
              type="button"
              class="hover:bg-muted hover:text-foreground group-hover/user:opacity-100 focus-visible:opacity-100"
              :aria-label="copiedId === group.id ? 'Copied' : 'Copy message'"
              :title="copiedId === group.id ? 'Copied' : 'Copy message'"
              @click="copyResponse(group.item.text, group.id)"
            )
              CheckIcon(v-if="copiedId === group.id" class="size-3.5")
              CopyIcon(v-else class="size-3.5")
          .flex.flex-col.text-sm(
            v-else-if="group.item.kind === 'text'"
            class="group/message"
            :data-item-id="group.id"
          )
            MarkdownText.prose.prose-sm.max-w-none(
              class="dark:prose-invert prose-p:my-1 prose-pre:my-2"
              :text="group.item.text"
            )

            button.text-muted-foreground.self-end.rounded-md.p-1.opacity-0.transition-opacity(
              type="button"
              class="hover:bg-muted hover:text-foreground group-hover/message:opacity-100 focus-visible:opacity-100"
              :aria-label="copiedId === group.id ? 'Copied' : 'Copy response'"
              :title="copiedId === group.id ? 'Copied' : 'Copy response'"
              @click="copyResponse(group.item.text, group.id)"
            )
              CheckIcon(v-if="copiedId === group.id" class="size-3.5")
              CopyIcon(v-else class="size-3.5")
          p.text-muted-foreground.text-xs.whitespace-pre-wrap(
            v-else
            :data-item-id="group.id"
          ) {{ group.item.text }}
        details.group.bg-card.rounded-lg.border(
          v-else-if="group.kind === 'activity'"
          @toggle="onActivityToggle(group.id, $event)"
          :data-item-id="group.id"
          :data-activity-id="group.id"
        )
          summary.flex.cursor-pointer.items-center.gap-2.px-3.py-2.text-xs.select-none
            ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
              class="group-open:rotate-90"
            )
            span.activity-title(v-if="group.running") {{ group.title }}
            span.text-muted-foreground(v-else) {{ group.title }}
            span.text-muted-foreground(v-if="group.running") running…
            span.text-muted-foreground · {{ group.items.length }} {{ group.items.length === 1 ? "step" : "steps" }}
            span.text-muted-foreground(v-if="groupDuration(group)") · {{ groupDuration(group) }}
          .flex.flex-col.gap-2.px-3.pb-3
            template(v-for="child in group.items" :key="child.id")
              p.text-muted-foreground.text-xs.whitespace-pre-wrap(
                v-if="child.kind === 'thinking'"
              )
                | {{ child.text }}
                span.ml-1(v-if="childDuration(child)") · {{ childDuration(child) }}
              details.rounded-md.border(v-else class="group/tool")
                summary.flex.cursor-pointer.items-center.gap-2.px-2.py-1.text-xs.select-none
                  ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
                    class="group-open/tool:rotate-90"
                  )
                  span.font-mono {{ child.name }}
                  span.text-muted-foreground(
                    v-if="child.phase !== ToolPhase.FINISHED"
                  ) running…
                  span.text-destructive(v-if="child.isError") error
                  span.text-muted-foreground(v-if="childDuration(child)") · {{ childDuration(child) }}
                .flex.flex-col.gap-2.px-2.pb-2
                  pre.overflow-auto.text-xs(v-if="child.args") {{ pretty(child.args) }}
                  pre.overflow-auto.text-xs(
                    v-if="child.result"
                    :class="child.isError ? 'text-destructive' : 'text-muted-foreground'"
                  ) {{ pretty(child.result) }}
        QuestionnaireWizard(
          v-else
          :key="`review-${group.id}`"
          readonly
          :questions="group.questions"
          :review-answers="questionnaireAnswersFromReview(group.questions, group.answers)"
          :cancelled="group.cancelled"
          :unavailable="group.unavailable"
          :submitting="false"
          :data-item-id="group.id"
          data-test="questionnaire-review"
        )

      QuestionnaireWizard(
        :key="wizardKey"
        v-if="questionnaire && showWizard"
        :questions="questionnaire.questions"
        :submitting="wizardSubmitting"
        :errors="wizardErrors"
        :storage-key="wizardStorageKey"
        @submit="submitWizard"
        @decline="declineWizard"
      )

      template(v-if="!showWizard")
        ExtensionUiPrompt(
          v-for="dialog in dialogs"
          :key="dialog.requestId"
          :request="dialog"
          :busy="resolving[dialog.requestId]"
          @respond="(response) => answer(dialog, response)"
        )

      .flex.flex-col.items-center.gap-3.py-10(v-if="!items.length && !error")
        Spinner.text-muted-foreground.size-5(v-if="busy")
        p.text-muted-foreground.text-sm {{ busy ? "The agent is starting…" : "Waiting for events…" }}

  .absolute.right-4.bottom-4.flex.flex-col.items-center.gap-2
    Button(
      v-if="openActivities.size"
      variant="outline"
      size="icon-sm"
      title="Collapse reasoning"
      @click="collapseActivities"
    )
      ChevronsDownUpIcon
    Button(
      variant="outline"
      size="icon-sm"
      title="Previous item"
      @click="step(-1)"
    )
      ChevronUpIcon
    Button(variant="outline" size="icon-sm" title="Next item" @click="step(1)")
      ChevronDownIcon
    Button(
      variant="outline"
      size="icon-sm"
      title="Scroll to bottom"
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
import ExtensionUiPrompt from "@/components/tasks/ExtensionUiPrompt.vue";
import QuestionnaireWizard from "@/components/tasks/QuestionnaireWizard.vue";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { pickActivityTitle } from "@/lib/activity";
import {
  activeQuestionnaire,
  findTool,
  groupElapsed,
  groupItems,
  itemElapsed,
  mergeTool,
  newItem,
  pinnedUserIndex,
  prependEvents,
  timestampMs,
  userMessageParts,
  type ActivityItem,
  type Group,
  type Item,
  type NewItem,
} from "@/lib/console";
import { formatDuration } from "@/lib/format";
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
  ToolPhase,
  type Attachment,
  type ExtensionUiRequest,
  type ExtensionUiResponse,
  type QuestionnaireAnswerInput,
  type TaskEvent,
} from "@cherrypi/sdk";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  ArrowDownIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  ChevronUpIcon,
  ChevronsDownUpIcon,
  CopyIcon,
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
import { toast } from "vue3-toastify";

const props = withDefaults(
  defineProps<{
    id: string;
    busy?: boolean;

    waitingForInput?: boolean;

    attachments?: Attachment[];

    actorPictures?: Map<string, Attachment>;
  }>(),
  {
    waitingForInput: undefined,
    attachments: () => [],
    actorPictures: () => new Map<string, Attachment>(),
  },
);

type Sender = NonNullable<Extract<Item, { kind: "user" }>["sender"]>;

function pictureFor(sender: Sender | undefined): Attachment | undefined {
  return sender?.id ? props.actorPictures.get(sender.id) : undefined;
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
  queue: [steering: string[]];

  openAttachment: [attachment: Attachment];
}>();

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

const items = ref<Item[]>([]);

const titles = new Map<number, string>();
function titleFor(id: number): string {
  let title = titles.get(id);
  if (!title) {
    title = pickActivityTitle();
    titles.set(id, title);
  }
  return title;
}

interface RenderActivity extends Extract<Group, { kind: "activity" }> {
  running: boolean;
}
type RenderGroup =
  | Extract<Group, { kind: "single" }>
  | Extract<Group, { kind: "questionnaire" }>
  | RenderActivity;

const groups = computed<RenderGroup[]>(() => {
  const raw = groupItems(items.value, titleFor);
  return raw.map((group, index) =>
    group.kind === "activity"
      ? { ...group, running: isRunning(group, index === raw.length - 1) }
      : group,
  );
});

function isRunning(
  group: Extract<Group, { kind: "activity" }>,
  isLast: boolean,
): boolean {
  return (
    group.items.some(
      (item) => item.kind === "tool" && item.phase !== ToolPhase.FINISHED,
    ) ||
    (isLast && props.busy === true)
  );
}

function elapsedLabel(ms: number | undefined): string {
  return ms !== undefined && ms >= 1000 ? formatDuration(ms) : "";
}

function groupDuration(group: RenderActivity): string {
  return elapsedLabel(groupElapsed(group.items));
}

function childDuration(item: ActivityItem): string {
  return elapsedLabel(itemElapsed(item));
}

const openActivities = reactive(new Set<number>());

const dialogs = ref<ExtensionUiRequest[]>([]);
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

const questionnaire = computed(() => activeQuestionnaire(items.value));
const wizardAnswers = ref<QuestionnaireAnswer[] | null>(null);
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
const showWizard = computed(
  () =>
    wizardActive.value &&
    (props.waitingForInput !== false || wizardEngaged.value),
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
    if (!resumed) {
      wizardKey.value += 1;
      wizardAnswers.value = null;
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
const error = ref("");
const scroller = ref<HTMLDivElement | null>(null);
const content = ref<HTMLDivElement | null>(null);
let resizeObserver: ResizeObserver | undefined;

const stickToBottom = ref(true);

const pinnedUser = ref<Extract<Item, { kind: "user" }> | undefined>();

function updatePinnedUser() {
  const el = scroller.value;
  const users = items.value.filter(
    (item): item is Extract<Item, { kind: "user" }> => item.kind === "user",
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
  nextBefore.value = "";
  historyLoaded.value = false;
  dialogs.value = [];
  startStream();
  void loadHistory();
}

function push(item: NewItem) {
  items.value.push({ ...item, id: nextId++ } as Item);
  if (stickToBottom.value) {
    void scrollToBottom();
  }
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
  } else if (kind === "thinking") {
    push({
      kind: "thinking",
      text: delta,
      startedAt,
      endedAt: endedAt ?? startedAt,
    });
    return;
  } else {
    push({ kind: "text", text: delta });
    return;
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
}

async function scrollToBottom() {
  stickToBottom.value = true;
  await nextTick();
  pinToBottom();
  updatePinnedUser();
}

function pinToBottom() {
  const el = scroller.value;
  if (el) {
    el.scrollTop = el.scrollHeight;
  }
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

function onActivityToggle(id: number, event: Event) {
  if ((event.target as HTMLDetailsElement).open) {
    openActivities.add(id);
  } else {
    openActivities.delete(id);
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
    items.value = prependEvents(items.value, page.events, () => nextId++);
    nextBefore.value = page.nextBefore;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    historyLoaded.value = true;
    for (const event of liveQueue.splice(0)) {
      applyEvent(event);
    }
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
    items.value = prependEvents(items.value, page.events, () => nextId++);
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

  if (payload.case !== "text" && payload.case !== "thinking") {
    flushDeltas();
  }
  switch (payload.case) {
    case "text":
      queueDelta("text", payload.value.delta);
      break;
    case "userMessage": {
      push({ kind: "user", ...userMessageParts(payload.value) });
      break;
    }
    case "pendingMessages":
      emit("queue", payload.value.steering);
      break;
    case "thinking":
      queueDelta("thinking", payload.value.delta, timestampMs(event.at));
      break;
    case "tool": {
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
    case "log":
      push({ kind: "log", text: payload.value.line });
      break;
    case "extensionUi": {
      const request = payload.value;
      if (isDialogMethod(request.method)) {
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
        await respondWizardBatch(request, active.questions);
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
      await respondWizard(request, question);
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
}

function isBatchDialog(request: ExtensionUiRequest): boolean {
  return parseQuestionnaireDialog(request) !== null;
}

async function respondWizardBatch(
  request: ExtensionUiRequest,
  questions: QuestionnaireQuestion[],
) {
  const answers = wizardAnswers.value;
  if (!answers) {
    return;
  }
  try {
    await agent.respondToExtensionUi(
      props.id,
      request.requestId,
      { value: questionnaireResponseValue(questions, answers) },
      submittedQuestionnaire(),
    );
  } catch (err) {
    wizardErrors.value = err instanceof Error ? err.message : String(err);
  }
}

async function respondWizard(
  request: ExtensionUiRequest,
  question: QuestionnaireQuestion,
) {
  const answer = wizardAnswers.value?.[wizardStep.value];
  if (!answer) {
    return;
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
      return;
    }
    if (question.multiSelect) {
      await agent.respondToExtensionUi(
        props.id,
        request.requestId,
        { value: multiSelectValue(answer) },
        resumeAnswers,
      );
      wizardStep.value += 1;
      return;
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
      return;
    }
    await agent.respondToExtensionUi(
      props.id,
      request.requestId,
      { value: options[answer.selected[0] ?? 0] ?? "" },
      resumeAnswers,
    );
    wizardStep.value += 1;
  } catch (err) {
    wizardErrors.value = err instanceof Error ? err.message : String(err);
  }
}

function submittedQuestionnaire(): QuestionnaireAnswerInput[] | undefined {
  const active = questionnaire.value;
  const answers = wizardAnswers.value;
  if (!active || !answers) {
    return undefined;
  }
  return active.questions.map((question, index) => {
    const answer = answers[index];
    return {
      question: question.question,
      answer: answer ? answerLabel(question, answer) : "",
    };
  });
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

function submitWizard(answers: QuestionnaireAnswer[]) {
  wizardErrors.value = "";
  wizardAnswers.value = answers;
  wizardStep.value = 0;
  wizardAwaitingCustom.value = false;
  wizardEngaged.value = true;
  void drainWizard();
}

function declineWizard() {
  wizardDeclined.value = true;
  wizardAnswers.value = null;
  wizardAwaitingCustom.value = false;
  const pending = wizardQueue.shift();
  if (pending) {
    void cancelWizardDialog(pending);
  }
}

function pretty(json: string): string {
  try {
    return JSON.stringify(JSON.parse(json), null, 2);
  } catch {
    return json;
  }
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
});
</script>
