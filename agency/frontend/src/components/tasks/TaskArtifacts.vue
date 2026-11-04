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
.flex.flex-col.gap-3(v-if="view.length")
  .flex.flex-col.gap-2.rounded-md.border.p-3(
    v-for="entry in view"
    :key="entry.artifact.artifactId"
  )
    .flex.items-start.justify-between.gap-2
      .flex.min-w-0.flex-col(class="gap-0.5")
        span.truncate.text-sm.font-medium {{ entry.artifact.name || entry.artifact.artifactId }}
        span.text-muted-foreground.text-xs.break-words(
          v-if="entry.artifact.description"
        ) {{ entry.artifact.description }}
      span.text-muted-foreground.inline-flex.shrink-0.items-center.gap-1.text-xs(
        v-if="entry.artifact.streaming"
        title="The agent is still adding parts"
      )
        Spinner.size-3
        | streaming
    .flex.flex-col.gap-2(v-if="entry.parts.length")
      template(v-for="(partView, index) in entry.parts" :key="index")
        p.text-xs.break-words.whitespace-pre-wrap(
          v-if="partView.part.content.case === 'text'"
        ) {{ partView.part.content.value }}
        pre.text-muted-foreground.font-mono.text-xs.break-words.whitespace-pre-wrap(
          v-else-if="partView.part.content.case === 'dataJson'"
        ) {{ formatData(partView.part.content.value) }}
        template(v-else-if="partView.item")
          button.group.relative.block.w-full.cursor-zoom-in.overflow-hidden.rounded-md.border(
            v-if="partView.item.image"
            type="button"
            :title="`Preview ${partView.item.filename}`"
            :aria-label="`Preview ${partView.item.filename}`"
            @click="emit('open', partView.item)"
          )
            img.max-h-48.w-full.object-contain(
              :src="partView.item.url"
              :alt="partView.item.filename"
            )
            span.pointer-events-none.absolute.top-2.right-2.flex.size-6.items-center.justify-center.rounded-md.text-white.opacity-0.transition-opacity(
              class="bg-black/60 group-hover:opacity-100"
            )
              Maximize2Icon(class="size-3.5")
          button.flex.w-full.items-center.gap-2.rounded-md.text-left.text-xs(
            v-else
            type="button"
            class="hover:underline"
            :title="`Preview ${partView.item.filename}`"
            @click="emit('open', partView.item)"
          )
            FileIcon.size-3.shrink-0
            span.truncate {{ partView.item.filename }}
    span.text-muted-foreground.text-xs(v-else) No parts.
span.text-muted-foreground.text-sm(v-else)
  | No artifacts yet. The agent publishes its outputs here.
</template>

<script setup lang="ts">
import { artifactPartItem, type LightboxItem } from "@/lib/lightbox";
import type { Artifact } from "@cherrypi/sdk";
import { computed } from "vue";

const props = defineProps<{ artifacts: Artifact[] }>();
const emit = defineEmits<{ open: [item: LightboxItem] }>();

const view = computed(() =>
  props.artifacts.map((artifact) => ({
    artifact,
    parts: artifact.parts.map((part, index) => ({
      part,
      item: artifactPartItem(artifact, part, index),
    })),
  })),
);

function formatData(value: string): string {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}
</script>
