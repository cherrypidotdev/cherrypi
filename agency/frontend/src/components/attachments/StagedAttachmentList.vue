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
.flex.flex-col.gap-1(v-if="items.length")
  .group.flex.items-center.gap-2.rounded-md.border.px-2.py-1.text-xs(
    v-for="item in items"
    :key="item.id"
  )
    img.size-8.shrink-0.rounded.object-cover(
      v-if="item.url"
      :src="item.url"
      :alt="item.filename"
    )
    FileIcon.text-muted-foreground.size-4.shrink-0(v-else)
    span.min-w-0.flex-1.truncate(:title="item.filename") {{ item.filename }}
    span.text-muted-foreground.shrink-0(v-if="item.kind === 'sketch'")
      | sketch
    span.text-muted-foreground.shrink-0 {{ size(item) }}
    Button.shrink-0(
      v-if="item.url"
      as-child
      variant="ghost"
      size="icon-sm"
      title="Open in new tab"
      :aria-label="`Open ${item.filename} in a new tab`"
    )
      a(rel="noopener noreferrer" :href="item.url" target="_blank")
        ExternalLinkIcon.size-3
    Button.shrink-0(
      type="button"
      variant="ghost"
      size="icon-sm"
      title="Delete attachment"
      :aria-label="`Delete ${item.filename}`"
      @click="emit('remove', item.id)"
    )
      Trash2Icon.size-3
  span.text-muted-foreground.text-xs
    | Uploaded when the task is created.
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import type { StagedAttachment } from "@/composables/useStagedAttachments";
import { formatFileSize } from "@/lib/attachments";
import { ExternalLinkIcon, FileIcon, Trash2Icon } from "@lucide/vue";

defineProps<{ items: StagedAttachment[] }>();
const emit = defineEmits<{ remove: [id: string] }>();

function size(item: StagedAttachment): string {
  return formatFileSize(item.kind === "sketch" ? item.png.size : item.size);
}
</script>
