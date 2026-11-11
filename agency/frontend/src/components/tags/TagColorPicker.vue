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
  .grid.grid-cols-6(
    class="gap-1.5"
    role="radiogroup"
    :aria-label="$t('tagColor.label')"
  )
    button.relative.size-7.rounded-full.border.transition-transform(
      v-for="color in TAG_COLORS"
      :key="color.value"
      type="button"
      role="radio"
      class="focus-visible:outline-ring border-black/10 hover:scale-110 focus-visible:outline-2 focus-visible:outline-offset-2"
      :class="selected === color.value ? 'ring-2 ring-foreground ring-offset-2 ring-offset-background' : ''"
      :style="{ backgroundColor: color.value }"
      :title="color.name"
      :aria-label="color.name"
      :aria-checked="selected === color.value"
      @click="pick(color.value)"
      :disabled="disabled"
    )
      CheckIcon.size-4.text-white.drop-shadow(
        v-if="selected === color.value"
        aria-hidden="true"
      )
  .flex.flex-wrap.items-center.gap-2
    button.text-muted-foreground.text-xs.underline.underline-offset-2(
      type="button"
      :aria-expanded="customOpen"
      @click="toggleCustom"
      :disabled="disabled"
    )
      | {{ customOpen ? "Hide custom colour" : "Custom colour" }}
    Input.size-32.font-mono.text-xs(
      v-if="customOpen"
      v-model="custom"
      :placeholder="$t('tagColor.customPlaceholder')"
      maxlength="7"
      :aria-label="$t('tagColor.customLabel')"
      @input="onCustom"
      :disabled="disabled"
    )
  p.text-muted-foreground.text-xs(v-if="customOpen && custom && !customValid")
    | {{ $t("tagColor.invalid") }}
</template>

<script setup lang="ts">
import { Input } from "@/components/ui/input";
import { TAG_COLORS } from "@/lib/tag";
import { tagColor } from "@/lib/task";
import { CheckIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";

const props = defineProps<{
  modelValue: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{ "update:modelValue": [value: string] }>();

const HEX_RE = /^#[0-9a-fA-F]{6}$/;

const custom = ref("");
const customOpen = ref(false);

const selected = computed(() => tagColor(props.modelValue));
const customValid = computed(() => HEX_RE.test(custom.value.trim()));

const isPaletteColor = (value: string) =>
  TAG_COLORS.some((color) => color.value === value);

watch(
  () => props.modelValue,
  (value) => {
    const normalized = tagColor(value);
    if (value && !isPaletteColor(normalized)) {
      customOpen.value = true;
      custom.value = value;
    }
  },
  { immediate: true },
);

function pick(value: string) {
  if (props.disabled) return;
  customOpen.value = false;
  emit("update:modelValue", value);
}

function toggleCustom() {
  if (props.disabled) return;
  customOpen.value = !customOpen.value;
  if (customOpen.value && !custom.value) {
    custom.value = tagColor(props.modelValue);
  }
}

function onCustom(event: Event) {
  custom.value = (event.target as HTMLInputElement).value;
  if (customValid.value) {
    emit("update:modelValue", custom.value.trim().toLowerCase());
  }
}
</script>

<i18n lang="yaml">
en:
  tagColor:
    invalid: "Use a six-digit hex colour, e.g. #6366f1."
    label: "Tag color"
    customPlaceholder: "#rrggbb"
    customLabel: "Custom tag color"
ru:
  tagColor:
    invalid: "Используйте шестизначный hex-цвет, например #6366f1."
    label: "Цвет тега"
    customPlaceholder: "#rrggbb"
    customLabel: "Свой цвет тега"
</i18n>
