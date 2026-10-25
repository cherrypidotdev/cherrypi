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
img.shrink-0.rounded-full.object-cover(
  v-if="src"
  :src="src"
  :alt="label"
  :style="{ width: `${size}px`, height: `${size}px` }"
)
span.bg-muted.text-muted-foreground.flex.shrink-0.items-center.justify-center.rounded-full.font-medium(
  v-else
  :title="label"
  :aria-label="label"
  :style="{ width: `${size}px`, height: `${size}px`, fontSize: `${Math.round(size * 0.42)}px` }"
) {{ initials }}
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    src?: string;

    name?: string;

    username?: string;

    size?: number;
  }>(),
  { src: "", name: "", username: "", size: 20 },
);

const label = computed(() => props.name || props.username || "?");
const initials = computed(() => {
  const parts = label.value.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase();
  return (parts[0]![0]! + parts[parts.length - 1]![0]!).toUpperCase();
});
</script>
