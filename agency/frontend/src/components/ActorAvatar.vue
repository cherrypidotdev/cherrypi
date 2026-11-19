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
.d-avatar.shrink-0(
  v-if="src"
  :style="{ width: `${size}px`, height: `${size}px` }"
)
  .h-full.w-full.rounded-full
    img(:src="src" :alt="label")
.d-avatar.shrink-0(
  v-else-if="emoji"
  :title="decorative ? undefined : label"
  :aria-label="decorative ? undefined : label"
  :aria-hidden="decorative ? 'true' : undefined"
  :style="{ width: `${size}px`, height: `${size}px` }"
)
  .bg-muted.text-muted-foreground.flex.h-full.w-full.items-center.justify-center.rounded-full(
    :style="{ fontSize: `${Math.round(size * 0.58)}px` }"
  ) {{ emoji }}
.d-avatar.d-avatar-placeholder.shrink-0(
  v-else-if="fallbackHtml"
  :title="decorative ? undefined : label"
  :aria-label="decorative ? undefined : label"
  :aria-hidden="decorative ? 'true' : undefined"
  :style="{ width: `${size}px`, height: `${size}px` }"
)
  span.h-full.w-full.rounded-full(:innerHTML="fallbackHtml")
.d-avatar.d-avatar-placeholder.shrink-0(
  v-else
  :title="decorative ? undefined : label"
  :aria-label="decorative ? undefined : label"
  :aria-hidden="decorative ? 'true' : undefined"
  :style="{ width: `${size}px`, height: `${size}px` }"
)
  .bg-muted.text-muted-foreground.h-full.w-full.rounded-full.font-medium(
    :style="{ fontSize: `${Math.round(size * 0.42)}px` }"
  ) {{ initials }}
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    src?: string;

    emoji?: string;

    name?: string;

    username?: string;

    size?: number;

    decorative?: boolean;

    fallbackHtml?: string;
  }>(),
  {
    src: "",
    emoji: "",
    name: "",
    username: "",
    size: 20,
    decorative: false,
    fallbackHtml: "",
  },
);

const label = computed(() => props.name || props.username || "?");
const initials = computed(() => {
  const parts = label.value.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase();
  return (parts[0]![0]! + parts[parts.length - 1]![0]!).toUpperCase();
});
</script>
