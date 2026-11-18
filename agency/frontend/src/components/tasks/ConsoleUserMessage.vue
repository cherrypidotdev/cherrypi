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
.d-chat.d-chat-end
  ChatAvatar(:sender="item.sender" :picture="picture")

  .d-chat-header.mb-1
    MessageSender(:sender="item.sender" :avatar="false")

  .d-chat-bubble.d-chat-bubble-primary(class="max-w-[80%]")
    MarkdownText.prose.prose-sm.max-w-none(
      class="prose-p:my-1 prose-pre:my-2"
      zoomable
      :style="INHERIT_PROSE"
      :text="item.text"
      @open-image="emit('openImage', $event)"
      @open-diagram="emit('openDiagram', $event)"
    )

  .d-chat-footer.mt-1.items-center.gap-2
    AttachmentChips(
      v-if="attachments.length"
      :attachments="attachments"
      @open="emit('openAttachment', $event)"
    )

    time.text-muted-foreground(
      v-if="item.at"
      :datetime="isoInstant(item.at)"
      :title="formatInstant(item.at)"
    ) {{ formatClock(item.at) }}

    button.text-muted-foreground.rounded-md.p-1(
      type="button"
      class="hover:bg-muted hover:text-foreground"
      :aria-label="copied ? $t('taskConsole.copied') : $t('taskConsole.copyMessage')"
      :title="copied ? $t('taskConsole.copied') : $t('taskConsole.copyMessage')"
      @click="emit('copy')"
    )
      CheckIcon(v-if="copied" class="size-3.5")
      CopyIcon(v-else class="size-3.5")
</template>

<script setup lang="ts">
import AttachmentChips from "@/components/attachments/AttachmentChips.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import MessageSender from "@/components/MessageSender.vue";
import ChatAvatar from "@/components/tasks/ChatAvatar.vue";
import type { Item } from "@/lib/console";
import { formatClock, formatInstant, isoInstant } from "@/lib/format";
import type { DiagramSource, InlineImage } from "@/lib/lightbox";
import { INHERIT_PROSE } from "@/lib/markdown";
import type { Attachment } from "@cherrypi/sdk";
import { CheckIcon, CopyIcon } from "@lucide/vue";

withDefaults(
  defineProps<{
    item: Extract<Item, { kind: "user" }>;

    picture?: Attachment;

    attachments?: Attachment[];

    copied?: boolean;
  }>(),
  {
    picture: undefined,
    attachments: () => [],
    copied: false,
  },
);

const emit = defineEmits<{
  copy: [];

  openAttachment: [attachment: Attachment];

  openImage: [image: InlineImage];

  openDiagram: [diagram: DiagramSource];
}>();
</script>

<i18n lang="yaml">
en:
  taskConsole:
    copied: "Copied"
    copyMessage: "Copy message"
ru:
  taskConsole:
    copied: "Скопировано"
    copyMessage: "Скопировать сообщение"
</i18n>
