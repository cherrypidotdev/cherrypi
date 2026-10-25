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
.flex.flex-col.gap-1(v-if="visibleAttachments.length")
  .group.flex.items-start.gap-2.rounded-md.border.px-2.py-1.text-xs(
    v-for="attachment in visibleAttachments"
    :key="attachment.url"
  )
    button.size-8.shrink-0.cursor-pointer.rounded.transition-opacity(
      v-if="isImageAttachment(attachment)"
      type="button"
      class="hover:opacity-75"
      :title="`Preview ${attachment.filename}`"
      :aria-label="`Preview ${attachment.filename}`"
      @click="emit('open', attachment)"
    )
      img.size-8.rounded.object-cover(
        :src="attachment.url"
        :alt="attachment.filename"
      )
    FileIcon.text-muted-foreground.size-4.shrink-0(v-else class="mt-0.5")
    .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
      a.truncate(
        class="hover:underline"
        :href="attachment.url"
        :title="`Preview ${attachment.filename}`"
        @click.prevent="emit('open', attachment)"
      ) {{ attachment.filename }}
      .flex.flex-wrap.items-center.gap-1.text-xs
        span.text-muted-foreground {{ formatFileSize(attachment.size) }}
        template(v-if="attachment.createdBy")
          span.text-muted-foreground ·
          AttachmentAuthor(:actor="attachment.createdBy")
    .flex.shrink-0.items-center(class="gap-0.5")
      Button.shrink-0(
        v-if="sketchable && isImageAttachment(attachment)"
        type="button"
        variant="ghost"
        size="icon-sm"
        :title="sketchTitle(attachment)"
        @click="emit('sketch', attachment)"
        :data-testid="sketchTestId(attachment)"
      )
        PencilIcon.size-3
      Button.shrink-0(
        as-child
        variant="ghost"
        size="icon-sm"
        title="Open in new tab"
        :aria-label="`Open ${attachment.filename} in a new tab`"
      )
        a(rel="noopener noreferrer" :href="attachment.url" target="_blank")
          ExternalLinkIcon.size-3
      Button.shrink-0(
        v-if="removable"
        type="button"
        variant="ghost"
        size="icon-sm"
        title="Delete attachment"
        :aria-label="`Delete ${attachment.filename}`"
        @click="emit('remove', attachment)"
      )
        Trash2Icon.size-3
span.text-muted-foreground.text-sm(v-else) {{ emptyText }}
</template>

<script setup lang="ts">
import AttachmentAuthor from "@/components/attachments/AttachmentAuthor.vue";
import { Button } from "@/components/ui/button";
import { formatFileSize, isImageAttachment } from "@/lib/attachments";
import { isSketchScene, sketchSceneFor } from "@/lib/sketches";
import type { Attachment } from "@cherrypi/sdk";
import {
  ExternalLinkIcon,
  FileIcon,
  PencilIcon,
  Trash2Icon,
} from "@lucide/vue";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    attachments: Attachment[];
    removable?: boolean;
    emptyText?: string;

    sketchable?: boolean;
  }>(),
  { removable: false, emptyText: "—", sketchable: false },
);
const emit = defineEmits<{
  remove: [attachment: Attachment];
  sketch: [attachment: Attachment];
  open: [attachment: Attachment];
}>();

const visibleAttachments = computed(() =>
  props.attachments.filter((attachment) => !isSketchScene(attachment)),
);

function hasScene(attachment: Attachment): boolean {
  return sketchSceneFor(props.attachments, attachment) !== undefined;
}

function sketchTitle(attachment: Attachment): string {
  return hasScene(attachment) ? "Edit sketch" : "Annotate image";
}

function sketchTestId(attachment: Attachment): string {
  return hasScene(attachment)
    ? `edit-sketch-${attachment.id}`
    : `annotate-image-${attachment.id}`;
}
</script>
