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
.flex.items-center.gap-3
  span.relative.size-12.shrink-0.overflow-hidden.border(
    :class="shape === 'square' ? 'rounded-lg' : 'rounded-full'"
  )
    img.absolute.inset-0.size-full.object-cover(
      v-if="preview"
      :src="preview"
      :alt="alt"
      data-testid="picture-picker-preview"
    )
    span.absolute.inset-0(v-else-if="fallback" :innerHTML="fallback")
  .flex.flex-col.gap-1
    .flex.items-center.gap-2
      Button(type="button" variant="outline" size="sm" @click="pick")
        UploadIcon(data-icon="inline-start")
        span {{ picture ? $t("picture.replace") : $t("picture.upload") }}
      Button(
        v-if="picture"
        type="button"
        variant="ghost"
        size="sm"
        @click="clear"
      )
        Trash2Icon(data-icon="inline-start")
        span {{ $t("common.remove") }}
    input.hidden(
      ref="input"
      type="file"
      accept="image/png,image/jpeg,image/webp,image/gif"
      @change="onChange"
    )
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Trash2Icon, UploadIcon } from "@lucide/vue";
import { onBeforeUnmount, ref, watch } from "vue";

withDefaults(
  defineProps<{
    fallback?: string;
    alt?: string;

    shape?: "circle" | "square";
  }>(),
  { fallback: "", alt: "", shape: "circle" },
);

const picture = defineModel<File | undefined>();
const input = ref<HTMLInputElement>();
const preview = ref("");
let previewUrl = "";

function release() {
  if (previewUrl && typeof URL.revokeObjectURL === "function") {
    URL.revokeObjectURL(previewUrl);
  }
  previewUrl = "";
}

watch(
  picture,
  (file) => {
    release();
    previewUrl =
      file && typeof URL.createObjectURL === "function"
        ? URL.createObjectURL(file)
        : "";
    preview.value = previewUrl;
  },
  { immediate: true },
);
onBeforeUnmount(release);

function pick() {
  input.value?.click();
}

function onChange(event: Event) {
  const element = event.target as HTMLInputElement;
  const file = element.files?.[0];
  element.value = "";
  if (file) picture.value = file;
}

function clear() {
  picture.value = undefined;
}
</script>
