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
section.d-card.bg-base-100.border-base-300.w-full.min-w-0.border
  .d-card-body.gap-3
    header.flex.flex-col
      h2.d-card-title.text-sm {{ title }}

      p.text-muted-foreground.flex.flex-wrap.items-center.gap-1.text-xs(
        v-if="readonly && answeredBy"
        data-test="questionnaire-answered-by"
      )
        span {{ $t("questionnaire.answeredBy") }}

        span.d-inline-flex.border-base-300.items-center.rounded-full.border.px-2(
          class="bg-base-200/60 py-0.5"
        )
          MessageSender(
            :sender="answeredBySender"
            :picture="answeredByPicture"
          )
        template(v-if="answeredAt !== undefined")
          span {{ $t("questionnaire.at") }}
          time.tabular-nums(
            :datetime="isoInstant(answeredAt)"
            :title="formatInstant(answeredAt)"
            data-test="questionnaire-answered-at"
          ) {{ formatClock(answeredAt) }}
      p.text-muted-foreground.text-sm(v-if="readonly && unavailable")
        | {{ $t("questionnaire.nonInteractive") }}
      p.text-muted-foreground.text-sm(v-else-if="readonly && cancelled")
        | {{ $t("questionnaire.declined") }}
      p.text-muted-foreground.text-sm(v-else-if="blockedReason")
        | {{ blockedReason }}
      p.text-muted-foreground.text-sm(v-else-if="readonly")
        | {{ $t("questionnaire.review") }}
      p.text-muted-foreground.text-sm(v-else-if="submitting")
        | {{ $t("questionnaire.sending") }}
      p.text-muted-foreground.text-sm(v-else-if="multiple")
        | {{ $t("questionnaire.answerMany") }}
      p.text-muted-foreground.text-sm(v-else)
        | {{ $t("questionnaire.answerOne") }}

    .flex.items-center.gap-3(v-if="multiple && !locked" data-test="progress")
      progress.d-progress.d-progress-primary.flex-1(
        :value="answeredCount"
        :aria-label="$t('questionnaire.answered')"
        :max="questions.length"
      )
      span.text-muted-foreground.text-xs.tabular-nums
        | {{ $t("questionnaire.answeredCount", { answered: answeredCount, total: questions.length }) }}

    template(v-if="multiple")
      .w-full.overflow-x-auto.overflow-y-hidden.overscroll-x-none(
        ref="tabStrip"
      )
        .d-tabs.d-tabs-box.d-tabs-sm.w-max(role="tablist")
          button.d-tab.shrink-0(
            v-for="(question, index) in questions"
            :key="index"
            type="button"
            class="gap-1.5"
            role="tab"
            :aria-selected="active === String(index + 1)"
            :tabindex="active === String(index + 1) ? 0 : -1"
            @click="selectTab(index)"
            @keydown="onTabKeydown($event, index)"
          )
            span.truncate
              MarkdownText(
                inline
                :allow-html="false"
                :text="label(question, index)"
              )
            CheckIcon.size-3(v-if="answered(index)" data-test="tab-answered")
            HourglassIcon.size-3.opacity-50(v-else data-test="tab-unanswered")

      template(
        v-for="(question, index) in questions"
        :key="`question-${index}`"
      )
        QuestionEditor(
          v-if="active === String(index + 1)"
          :question="question"
          :answer="answerAt(index)"
          :readonly="locked"
          @toggle="toggleOption(question, index, $event)"
          @toggle-custom="toggleCustom(index)"
          @notes="setNotes(index, $event)"
          :disabled="disabled"
        )

    QuestionEditor(
      v-else-if="single"
      :question="single"
      :answer="answerAt(0)"
      :readonly="locked"
      @toggle="toggleOption(single, 0, $event)"
      @toggle-custom="toggleCustom(0)"
      @notes="setNotes(0, $event)"
      :disabled="disabled"
    )

    .d-alert.d-alert-error.d-alert-soft.py-2.text-xs(
      v-if="errors"
      role="alert"
    )
      CircleAlertIcon(:size="14")
      span {{ errors }}

  template(v-if="!locked")
    .d-collapse.d-collapse-arrow.rounded-none.border-t(
      :class="{ 'd-collapse-open': finalNoteOpen }"
    )
      button.d-collapse-title.flex.items-center.gap-2.text-left(
        type="button"
        aria-controls="questionnaire-final-note"
        :aria-expanded="finalNoteOpen"
        @click="finalNoteOpen = !finalNoteOpen"
        data-test="final-note-toggle"
      )
        span.text-sm.font-medium {{ $t("questionnaire.finalNoteOptional") }}
        span.d-badge.d-badge-sm.d-badge-soft(
          v-if="finalNote.trim() && !finalNoteOpen"
        ) {{ $t("questionnaire.added") }}

      .d-collapse-content
        .flex.flex-col.gap-1(v-if="finalNoteOpen")
          #questionnaire-final-note
            MarkdownEditor(
              v-model="finalNote"
              v-model:resize-height="finalNoteHeight"
              :placeholder="$t('questionnaire.finalNotePlaceholder')"
              resizable
              test-id="final-note"
              :aria-label="$t('questionnaire.finalNote')"
              :disabled="submitting"
            )

    .flex.flex-wrap.items-center.justify-between.gap-2.border-t.p-3
      button.d-btn.d-btn-ghost.d-btn-sm(
        type="button"
        @click="emit('decline')"
        :disabled="submitting"
      )
        XIcon(:size="16")
        span {{ $t("questionnaire.decline") }}

      .flex.items-center.gap-2
        TaskModelPicker(
          v-if="agentId"
          :agent-id="agentId"
          :config="modelConfig"
          :selection="modelSelection"
          :use-defaults="false"
          @change="emit('modelChange', $event)"
          :disabled="modelDisabled || submitting"
        )
        button.d-btn.d-btn-primary(
          type="button"
          @click="submit"
          :disabled="disabled || remaining > 0"
          data-test="questionnaire-submit"
        )
          span.d-loading.d-loading-spinner.d-loading-xs(v-if="submitting")
          SendIcon(v-else :size="16")
          span {{ submitting ? "Submitting…" : "Submit" }}

  .border-t.p-4(v-else-if="finalNote.trim()" data-test="final-note-review")
    p.text-muted-foreground.text-xs {{ $t("questionnaire.finalNote") }}
    MarkdownText.prose.prose-sm.max-w-none(
      class="prose-p:my-1 prose-pre:my-2"
      :style="INHERIT_PROSE"
      :allow-html="false"
      :text="finalNote"
    )
</template>

<script setup lang="ts">
import MarkdownEditor from "@/components/MarkdownEditor.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import MessageSender from "@/components/MessageSender.vue";
import TaskModelPicker from "@/components/tasks/TaskModelPicker.vue";
import { useSessionEditorHeight } from "@/composables/useEditorHeight";
import type { MessageSender as ChatSender } from "@/lib/console";
import {
  COMPACT_EDITOR_HEIGHT,
  QUESTIONNAIRE_FINAL_NOTE_HEIGHT_KEY,
} from "@/lib/editorHeight";
import { formatClock, formatInstant, isoInstant } from "@/lib/format";
import { INHERIT_PROSE } from "@/lib/markdown";
import {
  emptyAnswer,
  isAnswered,
  type QuestionnaireAnswer,
  type QuestionnaireAnsweredBy,
  type QuestionnaireQuestion,
} from "@/lib/questionnaire";
import {
  EMPTY_TASK_MODEL_SELECTION,
  type TaskModelSelection,
} from "@/lib/task";
import type { Attachment, TaskConfig } from "@cherrypi/sdk";
import {
  CheckIcon,
  CircleAlertIcon,
  HourglassIcon,
  SendIcon,
  XIcon,
} from "@lucide/vue";
import { computed, nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import QuestionEditor from "./QuestionEditor.vue";

const props = withDefaults(
  defineProps<{
    questions: QuestionnaireQuestion[];
    submitting: boolean;
    errors?: string;

    storageKey?: string;

    readonly?: boolean;

    reviewAnswers?: QuestionnaireAnswer[];

    reviewFinalNote?: string;

    cancelled?: boolean;

    answeredBy?: QuestionnaireAnsweredBy;

    answeredByPicture?: Attachment;

    answeredAt?: number;

    blockedReason?: string;

    unavailable?: boolean;

    agentId?: string;

    modelConfig?: TaskConfig;

    modelSelection?: TaskModelSelection;

    modelDisabled?: boolean;
  }>(),
  {
    agentId: "",
    answeredBy: undefined,
    answeredByPicture: undefined,
    answeredAt: undefined,
    blockedReason: undefined,
    modelConfig: undefined,
    modelSelection: () => ({ ...EMPTY_TASK_MODEL_SELECTION }),
    modelDisabled: false,
  },
);
const emit = defineEmits<{
  submit: [answers: QuestionnaireAnswer[], finalNote: string];
  decline: [];
  modelChange: [selection: TaskModelSelection];
}>();

const readonlyTitle = computed(() => {
  if (props.unavailable) {
    return t("questionnaire.notShown");
  }
  return props.cancelled
    ? t("questionnaire.declinedTitle")
    : t("questionnaire.answered");
});

const locked = computed(
  () => props.readonly || props.blockedReason !== undefined,
);

const title = computed(() => {
  if (props.readonly) {
    return readonlyTitle.value;
  }
  return props.blockedReason === undefined
    ? t("questionnaire.needInput")
    : t("questionnaire.title");
});

const answeredBySender = computed<ChatSender>(() => ({
  id: props.answeredBy?.id,
  username: props.answeredBy?.username ?? "",
  name: props.answeredBy?.name ?? "",

  kind: props.answeredBy?.kind ?? "user",
}));

function draftKey(): string | null {
  return props.storageKey
    ? `cherrypi.questionnaire.draft.${props.storageKey}`
    : null;
}

function toAnswer(value: unknown): QuestionnaireAnswer {
  const raw = (value ?? {}) as Record<string, unknown>;
  return {
    selected: Array.isArray(raw.selected)
      ? raw.selected.filter((n): n is number => typeof n === "number" && n >= 0)
      : [],
    customSelected: raw.customSelected === true,
    custom: typeof raw.custom === "string" ? raw.custom : "",
    notes: typeof raw.notes === "string" ? raw.notes : "",
  };
}

interface WizardDraft {
  answers: QuestionnaireAnswer[];
  finalNote: string;
}

function loadDraft(): WizardDraft | null {
  const key = draftKey();
  if (!key) {
    return null;
  }
  try {
    const raw = sessionStorage.getItem(key);
    if (!raw) {
      return null;
    }
    const parsed: unknown = JSON.parse(raw);

    const answersRaw = Array.isArray(parsed)
      ? parsed
      : (parsed as { answers?: unknown } | null)?.answers;
    if (
      !Array.isArray(answersRaw) ||
      answersRaw.length !== props.questions.length
    ) {
      return null;
    }
    const storedNote = Array.isArray(parsed)
      ? ""
      : (parsed as { finalNote?: unknown }).finalNote;
    return {
      answers: answersRaw.map(toAnswer),
      finalNote: typeof storedNote === "string" ? storedNote : "",
    };
  } catch {
    return null;
  }
}

function clearDraft() {
  const key = draftKey();
  if (!key) {
    return;
  }
  try {
    sessionStorage.removeItem(key);
  } catch {}
}

const initialDraft = locked.value ? null : loadDraft();
const answers = ref<QuestionnaireAnswer[]>(
  locked.value
    ? (props.reviewAnswers?.map((answer) => ({
        ...answer,
        selected: [...answer.selected],
      })) ?? props.questions.map(() => emptyAnswer()))
    : (initialDraft?.answers ?? props.questions.map(() => emptyAnswer())),
);
const finalNote = ref(
  locked.value
    ? (props.reviewFinalNote ?? "")
    : (initialDraft?.finalNote ?? ""),
);
const finalNoteOpen = ref(finalNote.value.trim().length > 0);

const finalNoteHeight = useSessionEditorHeight(
  QUESTIONNAIRE_FINAL_NOTE_HEIGHT_KEY,
  COMPACT_EDITOR_HEIGHT,
);
const active = ref("1");
const tabStrip = ref<HTMLElement | null>(null);
const multiple = computed(() => props.questions.length > 1);
const single = computed(() => props.questions[0] ?? null);

let questionsSignature = "";
const currentSignature = () => JSON.stringify(props.questions);

function focusUnanswered() {
  const index = props.questions.findIndex(
    (_, i) => !isAnswered(answers.value[i]!),
  );
  active.value = String((index >= 0 ? index : 0) + 1);
}

watch(
  () => props.questions,
  () => {
    if (locked.value) {
      return;
    }
    const signature = currentSignature();
    if (signature === questionsSignature) {
      return;
    }
    questionsSignature = signature;
    const draft = loadDraft();
    answers.value = draft?.answers ?? props.questions.map(() => emptyAnswer());
    finalNote.value = draft?.finalNote ?? "";
    finalNoteOpen.value = finalNote.value.trim().length > 0;
    focusUnanswered();
  },
);

function saveDraft() {
  const key = draftKey();
  if (locked.value || !key || props.submitting) {
    return;
  }
  try {
    sessionStorage.setItem(
      key,
      JSON.stringify({ answers: answers.value, finalNote: finalNote.value }),
    );
  } catch {}
}
watch(answers, saveDraft, { deep: true });
watch(finalNote, saveDraft);

watch(finalNote, (value) => {
  if (value.trim().length > 0) {
    finalNoteOpen.value = true;
  }
});
watch(
  () => props.submitting,
  (submitting) => {
    if (submitting) {
      clearDraft();
    }
  },
);

const remaining = computed(
  () => props.questions.filter((_, i) => !isAnswered(answers.value[i]!)).length,
);
const answeredCount = computed(() => props.questions.length - remaining.value);
const disabled = computed(() => props.submitting);

function label(question: QuestionnaireQuestion, index: number): string {
  return question.header || `Question ${index + 1}`;
}

function answerAt(index: number): QuestionnaireAnswer {
  return answers.value[index] ?? emptyAnswer();
}

function answered(index: number): boolean {
  const answer = answers.value[index];
  return answer ? isAnswered(answer) : false;
}

function selectTab(index: number) {
  active.value = String(index + 1);
}

function onTabKeydown(event: KeyboardEvent, index: number) {
  const last = props.questions.length - 1;
  let target: number;
  switch (event.key) {
    case "ArrowRight":
      target = index === last ? 0 : index + 1;
      break;
    case "ArrowLeft":
      target = index === 0 ? last : index - 1;
      break;
    case "Home":
      target = 0;
      break;
    case "End":
      target = last;
      break;
    default:
      return;
  }
  event.preventDefault();
  selectTab(target);
  void nextTick(() => {
    tabStrip.value
      ?.querySelectorAll<HTMLElement>('[role="tab"]')
      .item(target)
      ?.focus();
  });
}

function setNotes(index: number, value: string) {
  const answer = answers.value[index];
  if (answer) {
    answer.notes = value;
  }
}

function toggleCustom(index: number) {
  if (disabled.value) {
    return;
  }
  const answer = answers.value[index];
  if (!answer) {
    return;
  }
  answer.customSelected = !answer.customSelected;
  if (answer.customSelected) {
    answer.selected = [];
  }
}

function toggleOption(
  question: QuestionnaireQuestion,
  questionIndex: number,
  optionIndex: number,
) {
  if (disabled.value) {
    return;
  }
  const answer = answers.value[questionIndex]!;
  answer.customSelected = false;
  if (question.multiSelect) {
    answer.selected = answer.selected.includes(optionIndex)
      ? answer.selected.filter((i) => i !== optionIndex)
      : [...answer.selected, optionIndex];
    return;
  }

  answer.selected = [optionIndex];
}

function submit() {
  if (remaining.value > 0 || disabled.value) {
    return;
  }
  emit(
    "submit",
    answers.value.map((answer) => {
      const text = answer.notes.trim();
      return {
        selected: [...answer.selected],
        customSelected: answer.customSelected,
        custom: answer.customSelected ? text : "",
        notes: answer.customSelected ? "" : text,
      };
    }),
    finalNote.value.trim(),
  );
}

watch(active, () => {
  void nextTick(() => {
    const strip = tabStrip.value;
    const element = strip?.querySelector<HTMLElement>(
      '[role="tab"][aria-selected="true"]',
    );
    if (!strip || !element) {
      return;
    }
    const stripRect = strip.getBoundingClientRect();
    const elementRect = element.getBoundingClientRect();
    if (elementRect.left < stripRect.left) {
      strip.scrollLeft += elementRect.left - stripRect.left;
    } else if (elementRect.right > stripRect.right) {
      strip.scrollLeft += elementRect.right - stripRect.right;
    }
  });
});

questionsSignature = currentSignature();
if (locked.value) {
  active.value = "1";
} else {
  focusUnanswered();
}

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  questionnaire:
    answeredBy: "Answered by"
    at: "at"
    nonInteractive: "This task is non-interactive, so the questionnaire was not shown. The agent was told to ask in plain text."
    declined: "The questionnaire was declined."
    review: "Review the answers that were sent; they can no longer be changed."
    sending: "Sending your answers…"
    answerMany: "Answer the questions below, then submit them together."
    answerOne: "Answer the question below, then submit."
    finalNoteOptional: "Final note (optional)"
    added: "added"
    finalNote: "Final note"
    answered: "Questions answered"
    finalNotePlaceholder: "Anything not tied to a single answer… Markdown supported."
    decline: "Decline"
    notShown: "Questionnaire not shown"
    declinedTitle: "Questionnaire declined"
    needInput: "Need your input"
    title: "Questionnaire"
    answeredCount: "{answered} / {total} answered"
ru:
  questionnaire:
    answeredBy: "Ответил"
    at: "в"
    nonInteractive: "Эта задача неинтерактивная, поэтому опросник не показывался. Агенту сообщили, что нужно спрашивать обычным текстом."
    declined: "Опросник отклонён."
    review: "Проверьте отправленные ответы; изменить их уже нельзя."
    sending: "Отправляем ответы…"
    answerMany: "Ответьте на вопросы ниже и отправьте их вместе."
    answerOne: "Ответьте на вопрос ниже и отправьте."
    finalNoteOptional: "Итоговая заметка (необязательно)"
    added: "добавлено"
    finalNote: "Итоговая заметка"
    answered: "Ответы отправлены"
    finalNotePlaceholder: "Всё, что не относится к одному ответу… Поддерживается Markdown."
    decline: "Отклонить"
    notShown: "Опросник не показан"
    declinedTitle: "Опросник отклонён"
    needInput: "Нужен ваш ответ"
    title: "Опросник"
    answeredCount: "отвечено {answered} / {total}"
</i18n>
