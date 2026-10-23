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
    button.w-full.rounded-md.border.p-2.text-left.transition-colors(
      v-for="(option, optionIndex) in question.options"
      :key="optionIndex"
      type="button"
      :class="rowClass(isSelected(optionIndex), option.recommended === true)"
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
      pre.text-muted-foreground.bg-muted.mt-2.overflow-auto.rounded.p-2.text-xs(
        v-if="option.preview"
      ) {{ option.preview }}

    //- The free-text escape is a first-class option: choosing it clears any
    //- picked options and turns the single free-text field below into the
    //- custom answer (the field relabels to "Custom answer" and blinks once).
    //- An empty custom answer still counts. In review it only appears when chosen.
    button.w-full.rounded-md.border.p-2.text-left.transition-colors(
      v-if="!readonly || answer.customSelected"
      type="button"
      :class="rowClass(answer.customSelected)"
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
  /** Review-only mode: show the answers but do not let them change. */
  readonly?: boolean;
}>();
const emit = defineEmits<{
  toggle: [optionIndex: number];
  notes: [value: string];
  "toggle-custom": [];
}>();

// The single free-text field doubles as the custom answer: selecting the
// "Type something." row focuses it and blinks it once so the user knows to type
// their answer there. The wizard turns whatever it holds into the answer.
const blinking = ref(false);

async function focusNotes() {
  await nextTick();
  const field = document.getElementById(`note-${props.index}`);
  if (field instanceof HTMLTextAreaElement) {
    field.focus();
  }
  // Toggle the class off first so re-selecting the row replays the animation.
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

function rowClass(selected: boolean, recommended = false): string {
  // Review rows keep full contrast and drop the hover affordance.
  const base = props.readonly
    ? "cursor-default"
    : props.disabled
      ? "opacity-60"
      : "hover:bg-muted/50";
  if (selected) {
    return `${base} border-primary`;
  }
  if (recommended) {
    return `${base} border-primary/50 bg-primary/5`;
  }
  return `${base} border-border`;
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

// Space activates a focused option. Prevent the native button activation so
// the row toggles exactly once (and the page does not scroll); key repeat is
// ignored so holding the key does not flicker the selection.
function onSpace(event: KeyboardEvent, action: () => void) {
  if (event.repeat || props.disabled || props.readonly) {
    return;
  }
  event.preventDefault();
  action();
}
</script>

<style scoped>
/* A one-shot attention cue; the label change and focus are the static cues. */
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
