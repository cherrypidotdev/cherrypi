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
button.text-muted-foreground.rounded-md.p-1(
  type="button"
  class="hover:bg-muted hover:text-foreground"
  :title="copied ? $t('copyButton.copied') : label"
  :aria-label="copied ? $t('copyButton.copied') : label"
  @click="copy"
)
  CheckIcon(v-if="copied" class="size-3.5")
  CopyIcon(v-else class="size-3.5")
</template>

<script setup lang="ts">
import { CheckIcon, CopyIcon } from "@lucide/vue";
import { onBeforeUnmount, ref } from "vue";

const COPIED_MS = 1500;

const props = withDefaults(
  defineProps<{
    text: string;

    label?: string;
  }>(),
  { label: "Copy" },
);

const copied = ref(false);
let timer: ReturnType<typeof setTimeout> | undefined;

async function copy() {
  try {
    await navigator.clipboard.writeText(props.text);
    copied.value = true;
    clearTimeout(timer);
    timer = setTimeout(() => {
      copied.value = false;
    }, COPIED_MS);
  } catch {}
}

onBeforeUnmount(() => clearTimeout(timer));
</script>

<i18n lang="yaml">
en:
  copyButton:
    copied: "Copied"
ru:
  copyButton:
    copied: "Скопировано"
</i18n>
