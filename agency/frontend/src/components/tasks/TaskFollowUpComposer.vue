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
form.relative.shrink-0.border-t.p-3(
  ref="composer"
  :class="canCompose ? '' : 'opacity-50'"
  @submit.prevent="emit('send')"
)
  .absolute.inset-x-0.-top-1.z-10.flex.h-2.cursor-row-resize.items-center.justify-center(
    v-if="canCompose && tab === 'write'"
    :title="$t('taskConsolePage.resizeHint')"
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
        | {{ $t("taskConsolePage.sessionGone") }}

    p.text-muted-foreground.inline-flex.items-start.gap-1.text-xs(
      v-if="awaitingAnswers"
      data-testid="composer-input-hint"
    )
      InfoIcon.size-3.shrink-0(class="mt-0.5")
      span
        | {{ $t("taskConsolePage.awaitingAnswers") }}

    p.text-muted-foreground.inline-flex.items-start.gap-1.text-xs(
      v-if="promptLocked"
      data-testid="composer-locked-hint"
    )
      LockIcon.size-3.shrink-0(class="mt-0.5")
      span
        | {{ $t("taskConsolePage.promptLocked") }}

    .flex.flex-col.gap-2
      .flex.flex-wrap.items-center.gap-2
        WritePreviewTabs(
          test-id="composer-tab"
          :model-value="tab"
          @update:model-value="emit('update:tab', $event)"
        )

        .ml-auto.flex.flex-wrap.items-center.gap-2
          TaskModelPicker(
            v-if="task.assigneeActorId && !sessionDeleted"
            :agent-id="agentIdForActor(task.assigneeActorId)"
            :config="config"
            :selection="modelSelection"
            :use-defaults="false"
            @change="emit('model-change', $event)"
            :disabled="modelSaving || !config || promptLocked"
          )
          Button.shrink-0(
            v-if="steer"
            type="button"
            variant="outline"
            :title="$t('taskConsolePage.stopHint')"
            @click="emit('stop')"
            :disabled="stopping"
          )
            Spinner(v-if="stopping" data-icon="inline-start")
            SquareIcon(v-else data-icon="inline-start")
            span {{ $t("settings.sounds.stop") }}
          Button.shrink-0(
            type="submit"
            :title="sendTitle"
            :disabled="!canFollowUp || !prompt.trim() || sending"
          )
            SendIcon(data-icon="inline-start")
            span {{ steer ? "Steer" : "Send" }}

      MarkdownEditor(
        v-if="tab === 'write'"
        ref="promptInput"
        :placeholder="awaitingAnswers ? $t('taskConsolePage.draftPlaceholder') : $t('taskConsolePage.sendPlaceholder')"
        submit-on-enter
        references
        test-id="composer-editor"
        :title="$t('taskConsolePage.editorHint')"
        :aria-label="$t('releaseHold.followUp')"
        :model-value="prompt"
        :reference-project-id="task?.projectId ?? ''"
        :reference-task-id="task?.id ?? ''"
        :fill="textareaHeight !== null"
        :style="textareaStyle"
        height="6.5rem"
        @update:model-value="emit('update:prompt', $event)"
        @submit="emit('enter')"
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
              :label="$t('taskPrompt.attach')"
              :task-id="task.id"
              @uploaded="emit('staged-uploaded', $event)"
            )
            Button(
              type="button"
              variant="outline"
              size="sm"
              :title="$t('taskPrompt.sketchAria')"
              @click="emit('open-new-sketch')"
            )
              PencilIcon(data-icon="inline-start")
              span {{ $t("taskPrompt.sketch") }}
            span.text-muted-foreground.text-xs(
              v-if="!stagedAttachments.length"
            ) {{ $t("taskConsolePage.attachmentsHint") }}
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
          @remove="emit('remove-staged', $event)"
          @sketch="emit('open-sketch', $event)"
          @open="emit('open-attachment', $event)"
        )
      p.text-muted-foreground.text-xs(v-if="isPostponed")
        | {{ $t("taskConsolePage.queuesImmediately") }}
</template>

<script setup lang="ts">
import AttachmentList from "@/components/attachments/AttachmentList.vue";
import AttachmentUploadButton from "@/components/attachments/AttachmentUploadButton.vue";
import MarkdownEditor from "@/components/MarkdownEditor.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import ContextMeter from "@/components/tasks/ContextMeter.vue";
import TaskModelPicker from "@/components/tasks/TaskModelPicker.vue";
import WritePreviewTabs from "@/components/tasks/WritePreviewTabs.vue";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useAgents } from "@/composables/useAgents";
import type { TaskModelSelection } from "@/lib/task";
import type {
  Attachment,
  Task,
  TaskConfig,
  TaskState as TaskStateValue,
  TaskUsageWithRuns,
} from "@cherrypi/sdk";
import { TaskState } from "@cherrypi/sdk";
import {
  CircleSlashIcon,
  InfoIcon,
  LockIcon,
  PencilIcon,
  SendIcon,
  SquareIcon,
} from "@lucide/vue";
import { computed, onBeforeUnmount, ref } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    task: Task;

    prompt: string;

    tab: "write" | "preview";

    canCompose: boolean;

    canFollowUp: boolean;

    promptLocked?: boolean;

    awaitingAnswers: boolean;

    steer: boolean;
    sending: boolean;
    stopping: boolean;

    config?: TaskConfig;

    modelSelection: TaskModelSelection;

    modelSaving?: boolean;

    stagedAttachments?: Attachment[];

    taskUsage?: TaskUsageWithRuns;
  }>(),
  {
    config: undefined,
    modelSaving: false,
    stagedAttachments: () => [],
    taskUsage: undefined,
    promptLocked: false,
  },
);

const emit = defineEmits<{
  (e: "update:prompt", value: string): void;
  (e: "update:tab", value: "write" | "preview"): void;
  (e: "send"): void;
  (e: "enter"): void;
  (e: "stop"): void;
  (e: "model-change", selection: TaskModelSelection): void;
  (e: "open-new-sketch"): void;
  (e: "open-sketch", attachment: Attachment): void;
  (e: "open-attachment", attachment: Attachment): void;
  (e: "staged-uploaded", attachment: Attachment): void;
  (e: "remove-staged", attachment: Attachment): void;
}>();

const { agentIdForActor } = useAgents();

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

const sessionDeleted = computed(() => Boolean(props.task.sessionDeletedAt));

const isPostponed = computed(
  () =>
    props.task.state === (TaskState.BACKLOG as TaskStateValue) &&
    props.task.startedAt !== undefined,
);

const sendTitle = computed(() =>
  props.awaitingAnswers
    ? t("taskConsolePage.sendTitleBlocked")
    : props.steer
      ? t("taskConsolePage.sendTitleSteer")
      : t("taskConsolePage.sendTitle"),
);

const contextWindow = computed(() =>
  Number(props.taskUsage?.contextWindow ?? 0),
);

const composer = ref<HTMLElement | null>(null);
const promptInput = ref<InstanceType<typeof MarkdownEditor> | null>(null);
const textareaHeight = ref<number | null>(null);
const resizing = ref(false);
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
</script>

<i18n lang="yaml">
en:
  taskConsolePage:
    sessionGone: "The agent-side Pi session and workspace are gone (cleaned up, or the agent restarted with fresh data). Sending a message replays the stored conversation in a new Pi session."
    awaitingAnswers: "The agent is waiting for your answers. You can draft a follow-up now; sending unlocks once the questionnaire is submitted."
    promptLocked: "Only this task’s owning teams, its leader, or an administrator can send follow-ups."
    attachmentsHint: "Paste, drop, or pick files to send with this prompt"
    queuesImmediately: "Sending a follow-up queues this task immediately."
    resizeHint: "Drag to resize the composer"
    stopHint: "Stop the agent"
    sendTitle: "Send prompt"
    sendTitleSteer: "Steer: injected between the run’s blocks"
    sendTitleBlocked: "Answer the questionnaire to send this follow-up"
    draftPlaceholder: "You can draft a follow-up while the questionnaire is open…"
    sendPlaceholder: "Send a follow-up prompt…"
    editorHint: "Enter sends · Shift+Enter adds a line · Tab indents · Ctrl+B bold · Ctrl+I italic · Ctrl+E code · Ctrl+K deletes to the end of the line · {'@'} mentions · # tasks"
ru:
  taskConsolePage:
    sessionGone: "Сеанс Pi и рабочее пространство на стороне агента отсутствуют (очищены или агент перезапущен с новыми данными). Отправка сообщения воспроизведёт сохранённую переписку в новом сеансе Pi."
    awaitingAnswers: "Агент ждёт ваших ответов. Уточнение можно написать сейчас; отправка станет доступна после отправки опросника."
    promptLocked: "Отправлять уточнения могут только владеющие команды задачи, её лидер или администратор."
    attachmentsHint: "Вставьте, перетащите или выберите файлы, чтобы отправить их с этим промптом"
    queuesImmediately: "Отправка уточнения сразу ставит задачу в очередь."
    resizeHint: "Потяните, чтобы изменить размер композера"
    stopHint: "Остановить агента"
    sendTitle: "Отправить промпт"
    sendTitleSteer: "Указание: вставляется между блоками запуска"
    sendTitleBlocked: "Ответьте на опросник, чтобы отправить уточнение"
    draftPlaceholder: "Пока опросник открыт, можно написать уточнение…"
    sendPlaceholder: "Отправить уточняющий промпт…"
    editorHint: "Enter — отправить · Shift+Enter — новая строка · Tab — отступ · Ctrl+B — жирный · Ctrl+I — курсив · Ctrl+E — код · Ctrl+K — удалить до конца строки · {'@'} — упоминания · # — задачи"
</i18n>
