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
.flex.flex-col.gap-2
  MarkdownText.prose.prose-sm.max-w-none(
    :style="INHERIT_PROSE"
    :allow-html="false"
    :text="question.question"
  )
  .flex.flex-col.gap-2
    button.w-full.rounded-md.border.p-2.text-left.transition-colors(
      v-for="(option, optionIndex) in question.options"
      :key="optionIndex"
      type="button"
      :class="[cardClass(isSelected(optionIndex), option.recommended === true), rowClass()]"
      @click="onToggle(optionIndex)"
      @keydown.space="onSpace($event, () => onToggle(optionIndex))"
      :disabled="disabled"
      data-test="option-card"
    )
      .flex.items-start.gap-2
        span.flex.size-4.shrink-0.items-center.justify-center.border(
          class="mt-0.5"
          :class="indicatorClass(isSelected(optionIndex))"
        )
          CheckIcon(v-if="isSelected(optionIndex)" class="size-2.5")
        .flex.min-w-0.flex-col(class="gap-0.5")
          .flex.items-center(class="gap-1.5")
            MarkdownText.text-sm.font-medium(
              inline
              :allow-html="false"
              :text="option.label"
              data-test="option-label"
            )
            span.text-primary.rounded-full.font-medium(
              v-if="option.recommended"
              class="bg-primary/10 px-1.5 py-0.5 text-[10px]"
              data-test="recommended"
            ) {{ $t("questionEditor.recommended") }}
          p.text-muted-foreground.text-xs(v-if="option.description")
            MarkdownText(inline :allow-html="false" :text="option.description")

    button.w-full.rounded-md.border.p-2.text-left.transition-colors(
      v-if="!readonly || answer.customSelected"
      type="button"
      :class="[cardClass(answer.customSelected), rowClass()]"
      @click="onToggleCustom()"
      @keydown.space="onSpace($event, onToggleCustom)"
      :disabled="disabled"
      data-test="custom-row"
    )
      .flex.items-start.gap-2
        span.flex.size-4.shrink-0.items-center.justify-center.border(
          class="mt-0.5"
          :class="indicatorClass(answer.customSelected)"
        )
          CheckIcon(v-if="answer.customSelected" class="size-2.5")
        .flex.min-w-0.flex-col(class="gap-0.5")
          span.text-sm.font-medium(data-test="custom-option") {{ $t("questionEditor.customOption") }}
          p.text-muted-foreground.text-xs(v-if="!readonly") {{ $t("questionEditor.answerInField") }}
          MarkdownText.prose.prose-sm.max-w-none(
            v-else
            class="prose-p:my-1 prose-pre:my-2"
            :style="INHERIT_PROSE"
            :allow-html="false"
            :text="answer.custom || '(empty)'"
            data-test="custom-review"
          )
  .flex.flex-col.gap-1(v-if="!readonly")
    span.text-muted-foreground.text-xs
      | {{ answer.customSelected ? $t("questionEditor.customAnswer") : $t("questionEditor.noteOptional") }}
    .rounded-lg(
      :class="{ 'questionnaire-custom-blink': blinking }"
      @animationend="blinking = false"
    )
      MarkdownEditor(
        ref="notesEditor"
        v-model:resize-height="noteHeight"
        :placeholder="answer.customSelected ? $t('questionEditor.customPlaceholder') : $t('questionEditor.notePlaceholder')"
        resizable
        test-id="note"
        :model-value="answer.notes"
        :aria-label="answer.customSelected ? $t('questionEditor.customAnswer') : $t('questionEditor.note')"
        @update:model-value="(value) => emit('notes', String(value))"
        :disabled="disabled"
      )

  .flex.flex-col(v-else-if="answer.notes")
    p.text-muted-foreground.text-xs {{ $t("questionEditor.note") }}
    MarkdownText.prose.prose-sm.max-w-none(
      :style="INHERIT_PROSE"
      :allow-html="false"
      :text="answer.notes"
      data-test="note-review"
    )
</template>

<script setup lang="ts">
import MarkdownEditor from "@/components/MarkdownEditor.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import { useSessionEditorHeight } from "@/composables/useEditorHeight";
import {
  COMPACT_EDITOR_HEIGHT,
  QUESTIONNAIRE_NOTE_HEIGHT_KEY,
} from "@/lib/editorHeight";
import { INHERIT_PROSE } from "@/lib/markdown";
import type {
  QuestionnaireAnswer,
  QuestionnaireQuestion,
} from "@/lib/questionnaire";
import { CheckIcon } from "@lucide/vue";
import { nextTick, ref, watch } from "vue";

const props = defineProps<{
  question: QuestionnaireQuestion;
  answer: QuestionnaireAnswer;
  disabled: boolean;

  readonly?: boolean;
}>();
const emit = defineEmits<{
  toggle: [optionIndex: number];
  notes: [value: string];
  "toggle-custom": [];
}>();

const blinking = ref(false);
const notesEditor = ref<InstanceType<typeof MarkdownEditor> | null>(null);

const noteHeight = useSessionEditorHeight(
  QUESTIONNAIRE_NOTE_HEIGHT_KEY,
  COMPACT_EDITOR_HEIGHT,
);

async function focusNotes() {
  await nextTick();
  notesEditor.value?.focus();

  blinking.value = false;
  await nextTick();
  blinking.value = true;
}

watch(
  () => props.answer.customSelected,
  (selected) => {
    if (selected && !props.readonly && !props.disabled) {
      void focusNotes();
    } else {
      blinking.value = false;
    }
  },
);

function isSelected(optionIndex: number): boolean {
  return props.answer.selected.includes(optionIndex);
}

function indicatorClass(selected: boolean): string {
  const shape = props.question.multiSelect ? "rounded" : "rounded-full";
  const state = selected
    ? "border-primary bg-primary text-primary-foreground"
    : "border-muted-foreground/40";
  return `${shape} ${state}`;
}

function cardClass(selected: boolean, recommended = false): string {
  const state = props.disabled ? "opacity-60" : "";
  if (selected) {
    return `${state} border-primary`;
  }
  if (recommended) {
    return `${state} border-primary/50 bg-primary/5`;
  }
  return `${state} border-border`;
}

function rowClass(): string {
  if (props.readonly) {
    return "cursor-default";
  }
  return props.disabled ? "" : "hover:bg-muted/50";
}

function onToggle(optionIndex: number) {
  if (props.readonly || props.disabled) {
    return;
  }
  emit("toggle", optionIndex);
}

function onToggleCustom() {
  if (props.readonly || props.disabled) {
    return;
  }
  emit("toggle-custom");
}

function onSpace(event: KeyboardEvent, action: () => void) {
  if (event.repeat || props.disabled || props.readonly) {
    return;
  }
  event.preventDefault();
  action();
}
</script>

<style scoped>
@media (prefers-reduced-motion: no-preference) {
  .questionnaire-custom-blink {
    animation: questionnaire-custom-blink 0.9s ease-in-out 1;
  }
}

@keyframes questionnaire-custom-blink {
  0%,
  100% {
    box-shadow: 0 0 0 0 transparent;
  }
  50% {
    box-shadow: 0 0 0 3px var(--ring);
  }
}
</style>

<i18n lang="yaml">
en:
  questionEditor:
    recommended: "Recommended"
    customOption: "Type something."
    answerInField: "Type your answer in the field below"
    note: "Note"
    noteOptional: "Note (optional)"
    customPlaceholder: "Write your own answer… Markdown supported."
    notePlaceholder: "Add context for this answer… Markdown supported."
    customAnswer: "Custom answer"
ru:
  questionEditor:
    recommended: "Рекомендуется"
    customOption: "Напишите свой вариант."
    answerInField: "Введите ответ в поле ниже"
    note: "Заметка"
    noteOptional: "Заметка (необязательно)"
    customPlaceholder: "Напишите свой ответ… Поддерживается Markdown."
    notePlaceholder: "Добавьте пояснение к ответу… Поддерживается Markdown."
    customAnswer: "Свой ответ"
</i18n>
