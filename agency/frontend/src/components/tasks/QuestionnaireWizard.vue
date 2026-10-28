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
Card.border-primary
  CardHeader
    CardTitle.text-sm
      | {{ readonly ? readonlyTitle : "Need your input" }}
    CardDescription(v-if="readonly && unavailable")
      | This task is non-interactive, so the questionnaire was not shown. The agent was told to ask in plain text.
    CardDescription(v-else-if="readonly && cancelled") The questionnaire was declined.
    CardDescription(v-else-if="readonly") Review the answers that were sent; they can no longer be changed.
    CardDescription(v-else-if="submitting") Sending your answers…
    CardDescription(v-else-if="multiple") Answer the questions below, then submit them together.
    CardDescription(v-else) Answer the question below, then submit.
  CardContent
    .flex.items-center.gap-3(v-if="multiple && !readonly" data-test="progress")
      .bg-muted.flex-1.overflow-hidden.rounded-full(class="h-1.5")
        .bg-primary.h-full.rounded-full.duration-150.ease-out(
          class="transition-[width]"
          role="progressbar"
          aria-valuemin="0"
          :style="{ width: `${percent}%` }"
          :aria-valuenow="answeredCount"
          :aria-valuemax="questions.length"
        )
      span.text-muted-foreground.text-xs.tabular-nums
        | {{ answeredCount }} / {{ questions.length }} answered

    Tabs.gap-3(v-if="multiple" v-model="active")
      .w-full.overflow-x-auto.overscroll-x-contain(
        ref="tabStrip"
        class="[&::-webkit-scrollbar-thumb]:bg-border [scrollbar-width:thin] [&::-webkit-scrollbar]:h-1.5 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-track]:bg-transparent"
      )
        TabsList.w-max.justify-start
          TabsTrigger.shrink-0(
            v-for="(question, index) in questions"
            :key="index"
            :value="String(index + 1)"
          )
            span.truncate
              MarkdownText(inline :text="label(question, index)")
            CheckIcon.size-3(v-if="answered(index)" data-test="tab-answered")
            HourglassIcon.size-3.opacity-50(v-else data-test="tab-unanswered")

      TabsContent(
        v-for="(question, index) in questions"
        :key="`question-${index}`"
        :value="String(index + 1)"
      )
        QuestionEditor(
          :question="question"
          :index="index"
          :answer="answerAt(index)"
          :readonly="readonly"
          @toggle="toggleOption(question, index, $event)"
          @toggle-custom="toggleCustom(index)"
          @notes="setNotes(index, $event)"
          :disabled="disabled"
        )

    template(v-else-if="single")
      QuestionEditor(
        :question="single"
        :index="0"
        :answer="answerAt(0)"
        :readonly="readonly"
        @toggle="toggleOption(single, 0, $event)"
        @toggle-custom="toggleCustom(0)"
        @notes="setNotes(0, $event)"
        :disabled="disabled"
      )
      p.text-destructive.mt-2.text-xs(v-if="errors") {{ errors }}

    p.text-destructive.mt-2.text-xs(v-if="multiple && errors") {{ errors }}

  CardFooter.flex-wrap.justify-between.border-t(v-if="!readonly")
    Button(
      variant="ghost"
      size="sm"
      @click="emit('decline')"
      :disabled="submitting"
    )
      XIcon(data-icon="inline-start")
      span Decline
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
      Button(@click="submit" :disabled="disabled || remaining > 0")
        Spinner(v-if="submitting" data-icon="inline-start")
        SendIcon(v-else data-icon="inline-start")
        span {{ submitting ? "Submitting…" : "Submit answers" }}
</template>

<script setup lang="ts">
import MarkdownText from "@/components/MarkdownText.vue";
import TaskModelPicker from "@/components/tasks/TaskModelPicker.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  emptyAnswer,
  isAnswered,
  type QuestionnaireAnswer,
  type QuestionnaireQuestion,
} from "@/lib/questionnaire";
import {
  EMPTY_TASK_MODEL_SELECTION,
  type TaskModelSelection,
} from "@/lib/task";
import type { TaskConfig } from "@cherrypi/sdk";
import { CheckIcon, HourglassIcon, SendIcon, XIcon } from "@lucide/vue";
import { computed, nextTick, ref, watch } from "vue";
import QuestionEditor from "./QuestionEditor.vue";

const props = withDefaults(
  defineProps<{
    questions: QuestionnaireQuestion[];
    submitting: boolean;
    errors?: string;

    storageKey?: string;

    readonly?: boolean;

    reviewAnswers?: QuestionnaireAnswer[];

    cancelled?: boolean;

    unavailable?: boolean;

    agentId?: string;

    modelConfig?: TaskConfig;

    modelSelection?: TaskModelSelection;

    modelDisabled?: boolean;
  }>(),
  {
    agentId: "",
    modelConfig: undefined,
    modelSelection: () => ({ ...EMPTY_TASK_MODEL_SELECTION }),
    modelDisabled: false,
  },
);
const emit = defineEmits<{
  submit: [answers: QuestionnaireAnswer[]];
  decline: [];
  modelChange: [selection: TaskModelSelection];
}>();

const readonlyTitle = computed(() => {
  if (props.unavailable) {
    return "Questionnaire not shown";
  }
  return props.cancelled ? "Questionnaire declined" : "Questions answered";
});

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

function loadDraft(): QuestionnaireAnswer[] | null {
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
    if (!Array.isArray(parsed) || parsed.length !== props.questions.length) {
      return null;
    }
    return parsed.map(toAnswer);
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

function initialAnswers(): QuestionnaireAnswer[] {
  if (props.readonly) {
    return (
      props.reviewAnswers?.map((answer) => ({
        ...answer,
        selected: [...answer.selected],
      })) ?? props.questions.map(() => emptyAnswer())
    );
  }
  return loadDraft() ?? props.questions.map(() => emptyAnswer());
}

const answers = ref<QuestionnaireAnswer[]>(initialAnswers());
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
    if (props.readonly) {
      return;
    }
    const signature = currentSignature();
    if (signature === questionsSignature) {
      return;
    }
    questionsSignature = signature;
    answers.value = loadDraft() ?? props.questions.map(() => emptyAnswer());
    focusUnanswered();
  },
);

watch(
  answers,
  (value) => {
    const key = draftKey();
    if (props.readonly || !key || props.submitting) {
      return;
    }
    try {
      sessionStorage.setItem(key, JSON.stringify(value));
    } catch {}
  },
  { deep: true },
);
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
const percent = computed(() =>
  props.questions.length === 0
    ? 0
    : Math.round((answeredCount.value / props.questions.length) * 100),
);
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
  );
}

watch(active, () => {
  void nextTick(() => {
    const strip = tabStrip.value;
    const element = strip?.querySelector<HTMLElement>('[data-state="active"]');
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
if (props.readonly) {
  active.value = "1";
} else {
  focusUnanswered();
}
</script>
