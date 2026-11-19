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

<template>
  <div v-if="items.length" class="flex flex-col gap-1.5">
    <a
      v-for="link in items"
      :key="link.url"
      :href="link.url"
      target="_blank"
      rel="noreferrer noopener"
      class="text-primary flex items-center gap-2 text-sm hover:underline"
    >
      <span class="truncate">{{ link.display }}</span>
      <span class="text-muted-foreground" aria-hidden="true">↗</span>
    </a>
  </div>
</template>

<script setup lang="ts">
import type { ActorLinkInput } from "@cherrypi/sdk";
import { computed } from "vue";

const props = withDefaults(defineProps<{ links?: ActorLinkInput[] }>(), {
  links: () => [],
});

const items = computed(() =>
  (props.links ?? []).map((link) => ({
    ...link,
    display: link.label || host(link.url),
  })),
);

function host(url: string): string {
  try {
    return new URL(url).host;
  } catch {
    return url;
  }
}
</script>
