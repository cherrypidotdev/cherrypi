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
  .flex.flex-wrap.items-center.gap-1
    .group.flex.max-w-full.items-center.gap-2.rounded-md.border.pr-1.pl-2.text-xs(
      v-for="item in items"
      :key="item.id"
      class="py-0.5"
    )
      img.size-6.shrink-0.rounded.object-cover(
        v-if="item.url"
        :src="item.url"
        :alt="item.filename"
      )
      FileIcon.text-muted-foreground.shrink-0(v-else class="size-3.5")
      span.max-w-40.min-w-0.truncate(:title="item.filename") {{ item.filename }}
      span.text-muted-foreground.shrink-0(v-if="item.kind === 'sketch'")
        | {{ $t("stagedAttachments.sketch") }}
      span.text-muted-foreground.shrink-0 {{ size(item) }}
      .flex.shrink-0.items-center.opacity-60.transition-opacity(
        class="gap-0.5 group-hover:opacity-100"
      )
        Button.shrink-0(
          v-if="item.kind === 'sketch'"
          type="button"
          variant="ghost"
          size="icon-xs"
          :title="$t('stagedAttachments.editSketch')"
          :aria-label="$t('attachments.editNamed', { filename: item.filename })"
          @click="emit('edit', item)"
        )
          PencilIcon.size-3
        Button.shrink-0(
          v-if="item.url"
          as-child
          variant="ghost"
          size="icon-xs"
          :title="$t('taskFiles.openInNewTab')"
          :aria-label="$t('attachments.openInNewTab', { filename: item.filename })"
        )
          a(rel="noopener noreferrer" :href="item.url" target="_blank")
            ExternalLinkIcon.size-3
        Button.shrink-0(
          type="button"
          variant="ghost"
          size="icon-xs"
          :title="$t('attachments.delete')"
          :aria-label="$t('attachments.deleteNamed', { filename: item.filename })"
          @click="emit('remove', item.id)"
        )
          Trash2Icon.size-3
  span.text-muted-foreground.text-xs
    | {{ $t("stagedAttachments.uploadedOnCreate") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import type { StagedAttachment } from "@/composables/useStagedAttachments";
import { formatFileSize } from "@/lib/attachments";
import {
  ExternalLinkIcon,
  FileIcon,
  PencilIcon,
  Trash2Icon,
} from "@lucide/vue";

defineProps<{ items: StagedAttachment[] }>();
const emit = defineEmits<{
  remove: [id: string];
  edit: [item: StagedAttachment];
}>();

function size(item: StagedAttachment): string {
  return formatFileSize(item.kind === "sketch" ? item.png.size : item.size);
}
</script>

<i18n lang="yaml">
en:
  stagedAttachments:
    sketch: "sketch"
    uploadedOnCreate: "Uploaded when the task is created."
    editSketch: "Edit sketch"
  attachments:
    editNamed: "Edit {filename}"
ru:
  stagedAttachments:
    sketch: "набросок"
    uploadedOnCreate: "Загружается при создании задачи."
    editSketch: "Изменить набросок"
  attachments:
    editNamed: "Изменить {filename}"
</i18n>
