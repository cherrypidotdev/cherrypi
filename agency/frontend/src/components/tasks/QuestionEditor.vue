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
.flex.flex-col.gap-3
  MarkdownText.text-sm.font-medium(:text="question.question")
  .flex.flex-col.gap-2
    .overflow-hidden.rounded-md.border.transition-colors(
      v-for="(option, optionIndex) in question.options"
      :key="optionIndex"
      :class="cardClass(isSelected(optionIndex), option.recommended === true)"
      data-test="option-card"
    )
      button.w-full.p-2.text-left.transition-colors(
        type="button"
        :class="rowClass()"
        @click="onToggle(optionIndex)"
        @keydown.space="onSpace($event, () => onToggle(optionIndex))"
        :disabled="disabled"
        data-test="option-row"
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
                :text="option.label"
                data-test="option-label"
              )
              span.text-primary.rounded-full.font-medium(
                v-if="option.recommended"
                class="bg-primary/10 px-1.5 py-0.5 text-[10px]"
                data-test="recommended"
              ) Recommended
            p.text-muted-foreground.text-xs(v-if="option.description")
              MarkdownText(inline :text="option.description")
      .border-t.px-2.py-2(v-if="option.preview" data-test="option-preview")
        MarkdownText.prose.prose-sm.max-w-none(
          class="prose-p:my-1 prose-pre:my-2"
          :style="INHERIT_PROSE"
          :text="option.preview"
        )

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
          span.text-sm.font-medium(data-test="custom-option") Type something.
          p.text-muted-foreground.text-xs(v-if="!readonly") Type your answer in the field below
          p.text-sm(v-else) {{ answer.custom || "(empty)" }}
  .flex.flex-col.gap-1(v-if="!readonly")
    label.text-muted-foreground.text-xs(:for="`note-${index}`")
      | {{ answer.customSelected ? "Custom answer" : "Note (optional)" }}
    Textarea(
      :placeholder="answer.customSelected ? 'Write your own answer…' : 'Add context for this answer…'"
      rows="2"
      :id="`note-${index}`"
      :model-value="answer.notes"
      :class="{ 'questionnaire-custom-blink': blinking }"
      @update:model-value="(value) => emit('notes', String(value))"
      @animationend="blinking = false"
      :disabled="disabled"
      data-test="note"
    )
  p.text-muted-foreground.text-xs(v-else-if="answer.notes") Note: {{ answer.notes }}
</template>

<script setup lang="ts">
import MarkdownText from "@/components/MarkdownText.vue";
import { Textarea } from "@/components/ui/textarea";
import { INHERIT_PROSE } from "@/lib/markdown";
import type {
  QuestionnaireAnswer,
  QuestionnaireQuestion,
} from "@/lib/questionnaire";
import { CheckIcon } from "@lucide/vue";
import { nextTick, ref, watch } from "vue";

const props = defineProps<{
  question: QuestionnaireQuestion;
  index: number;
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

async function focusNotes() {
  await nextTick();
  const field = document.getElementById(`note-${props.index}`);
  if (field instanceof HTMLTextAreaElement) {
    field.focus();
  }

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
