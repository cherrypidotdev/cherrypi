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
.relative.flex.flex-wrap.items-center.gap-2(ref="root")
  Button(type="button" variant="outline" size="sm" @click="toggle")
    span.text-base.leading-none(v-if="modelValue") {{ modelValue }}
    SmileIcon(v-else data-icon="inline-start")
    span {{ modelValue ? $t("actor.emoji.change") : $t("actor.emoji.pick") }}
  Button(
    v-if="modelValue"
    type="button"
    variant="ghost"
    size="sm"
    @click="clear"
  )
    Trash2Icon(data-icon="inline-start")
    span {{ $t("common.remove") }}
  .bg-popover.absolute.top-full.left-0.z-50.mt-2.overflow-hidden.rounded-lg.border.shadow-lg(
    v-if="open"
  )
    div(ref="host")
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import data from "@emoji-mart/data";
import { SmileIcon, Trash2Icon } from "@lucide/vue";
import { onClickOutside } from "@vueuse/core";
import { Picker } from "emoji-mart";
import { nextTick, onBeforeUnmount, ref } from "vue";

const props = withDefaults(
  defineProps<{ modelValue?: string; disabled?: boolean }>(),
  { modelValue: "", disabled: false },
);
const emit = defineEmits<{ "update:modelValue": [value: string] }>();

const root = ref<HTMLElement>();
const host = ref<HTMLElement>();
const open = ref(false);
let picker: HTMLElement | null = null;

function toggle() {
  if (props.disabled) return;
  open.value = !open.value;
  if (open.value) void mountPicker();
}

async function mountPicker() {
  await nextTick();
  if (!host.value) return;
  if (picker) {
    if (picker.parentElement !== host.value) host.value.appendChild(picker);
    return;
  }
  const element = new Picker({
    data,
    theme:
      typeof document !== "undefined" &&
      document.documentElement.classList.contains("dark")
        ? "dark"
        : "light",
    previewPosition: "none",
    skinTonePosition: "search",
    onEmojiSelect: (selected: { native?: string }) => {
      if (!selected.native) return;
      emit("update:modelValue", selected.native);
      open.value = false;
    },
  });
  picker = element as unknown as HTMLElement;
  host.value.appendChild(picker);
}

function clear() {
  emit("update:modelValue", "");
}

onClickOutside(root, () => {
  open.value = false;
});

onBeforeUnmount(() => {
  picker?.remove();
  picker = null;
});
</script>

<i18n lang="yaml">
en:
  actor:
    emoji:
      label: "Emoji"
      hint: "A single emoji instead of an uploaded picture. Setting one clears the picture."
      pick: "Pick emoji"
      change: "Change emoji"
ru:
  actor:
    emoji:
      label: "Эмодзи"
      hint: "Один эмодзи вместо загруженного изображения. При выборе изображение удаляется."
      pick: "Выбрать эмодзи"
      change: "Сменить эмодзи"
</i18n>
