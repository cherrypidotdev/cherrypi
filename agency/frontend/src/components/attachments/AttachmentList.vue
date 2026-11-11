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
.flex(v-if="visibleAttachments.length" :class="trayClass")
  .group.flex.rounded-md.border.text-xs(
    v-for="attachment in visibleAttachments"
    :key="attachment.url"
    :class="entryClass"
  )
    button.shrink-0.cursor-pointer.overflow-hidden.rounded.transition-opacity(
      v-if="isImageAttachment(attachment)"
      type="button"
      class="hover:opacity-75"
      :class="thumbClass"
      :title="$t('attachments.preview', { filename: attachment.filename })"
      :aria-label="$t('attachments.preview', { filename: attachment.filename })"
      @click="emit('open', attachment)"
    )
      img.rounded.object-cover(
        :src="attachment.url"
        :alt="attachment.filename"
        :class="thumbClass"
      )
    FileIcon.text-muted-foreground.shrink-0(v-else :class="fileIconClass")
    .flex.min-w-0(:class="bodyClass")
      a.truncate(
        class="hover:underline"
        :class="nameClass"
        :href="attachment.url"
        :title="$t('attachments.preview', { filename: attachment.filename })"
        @click.prevent="emit('open', attachment)"
      ) {{ attachment.filename }}
      .flex.shrink-0.items-center.gap-1.text-xs(:class="metaClass")
        span.text-muted-foreground {{ formatFileSize(attachment.size) }}
        template(v-if="showAuthor && attachment.createdBy")
          span.text-muted-foreground ·
          AttachmentAuthor(:actor="attachment.createdBy")
    .flex.shrink-0.items-center.transition-opacity(:class="actionsClass")
      Button.shrink-0(
        v-if="sketchable && isImageAttachment(attachment)"
        type="button"
        variant="ghost"
        :size="actionSize"
        :title="sketchTitle(attachment)"
        @click="emit('sketch', attachment)"
        :data-testid="sketchTestId(attachment)"
      )
        PencilIcon.size-3
      Button.shrink-0(
        as-child
        variant="ghost"
        :title="$t('taskFiles.openInNewTab')"
        :size="actionSize"
        :aria-label="$t('attachments.openInNewTab', { filename: attachment.filename })"
      )
        a(rel="noopener noreferrer" :href="attachment.url" target="_blank")
          ExternalLinkIcon.size-3
      Button.shrink-0(
        v-if="removable"
        type="button"
        variant="ghost"
        :title="$t('attachments.delete')"
        :size="actionSize"
        :aria-label="$t('attachments.deleteNamed', { filename: attachment.filename })"
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
import { useI18n } from "vue-i18n";

const props = withDefaults(
  defineProps<{
    attachments: Attachment[];
    removable?: boolean;
    emptyText?: string;

    sketchable?: boolean;

    compact?: boolean;

    showAuthor?: boolean;
  }>(),
  {
    removable: false,
    emptyText: "—",
    sketchable: false,
    compact: false,
    showAuthor: true,
  },
);
const emit = defineEmits<{
  remove: [attachment: Attachment];
  sketch: [attachment: Attachment];
  open: [attachment: Attachment];
}>();

const trayClass = computed(() =>
  props.compact ? "flex-wrap items-center gap-1" : "flex-col gap-1",
);
const entryClass = computed(() =>
  props.compact
    ? "max-w-full items-center gap-2 py-0.5 pl-2 pr-1"
    : "items-start gap-2 px-2 py-1",
);
const thumbClass = computed(() => (props.compact ? "size-6" : "size-8"));
const fileIconClass = computed(() =>
  props.compact ? "size-3.5" : "mt-0.5 size-4",
);
const bodyClass = computed(() =>
  props.compact ? "items-center gap-1.5" : "flex-1 flex-col gap-0.5",
);
const nameClass = computed(() => (props.compact ? "max-w-40" : ""));
const metaClass = computed(() => (props.compact ? "" : "flex-wrap"));

const actionsClass = computed(() =>
  props.compact ? "gap-0.5 opacity-60 group-hover:opacity-100" : "gap-0.5",
);
const actionSize = computed<"icon-xs" | "icon-sm">(() =>
  props.compact ? "icon-xs" : "icon-sm",
);

const visibleAttachments = computed(() =>
  props.attachments.filter((attachment) => !isSketchScene(attachment)),
);

function hasScene(attachment: Attachment): boolean {
  return sketchSceneFor(props.attachments, attachment) !== undefined;
}

function sketchTitle(attachment: Attachment): string {
  return hasScene(attachment)
    ? t("stagedAttachments.editSketch")
    : t("attachments.annotateImage");
}

function sketchTestId(attachment: Attachment): string {
  return hasScene(attachment)
    ? `edit-sketch-${attachment.id}`
    : `annotate-image-${attachment.id}`;
}

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  attachments:
    delete: "Delete attachment"
    annotateImage: "Annotate image"
    openInNewTab: "Open {filename} in a new tab"
    deleteNamed: "Delete {filename}"
ru:
  attachments:
    delete: "Удалить вложение"
    annotateImage: "Разметить изображение"
    openInNewTab: "Открыть {filename} в новой вкладке"
    deleteNamed: "Удалить {filename}"
</i18n>
