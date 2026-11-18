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
.d-chat.d-chat-start
  ChatAvatar(:sender="sender" :picture="picture")

  .d-chat-header.mb-1(v-if="sender")
    MessageSender(:sender="sender" :avatar="false")

  .d-chat-bubble
    MarkdownText.prose.prose-sm(
      class="dark:prose-invert prose-p:my-1 prose-pre:my-2"
      zoomable
      :text="item.text"
      @open-image="emit('openImage', $event)"
      @open-diagram="emit('openDiagram', $event)"
    )

  .d-chat-footer.mt-1.items-center.gap-2
    time.text-muted-foreground(
      v-if="item.at"
      :datetime="isoInstant(item.at)"
      :title="formatInstant(item.at)"
    ) {{ formatClock(item.at) }}
    RunUsageLabel(v-if="usage" :usage="usage")
    button.text-muted-foreground.rounded-md.p-1(
      type="button"
      class="hover:bg-muted hover:text-foreground"
      :aria-label="copied ? $t('taskConsole.copied') : $t('taskConsole.copyResponse')"
      :title="copied ? $t('taskConsole.copied') : $t('taskConsole.copyResponse')"
      @click="emit('copy')"
    )
      CheckIcon(v-if="copied" class="size-3.5")
      CopyIcon(v-else class="size-3.5")
</template>

<script setup lang="ts">
import MarkdownText from "@/components/MarkdownText.vue";
import MessageSender from "@/components/MessageSender.vue";
import ChatAvatar from "@/components/tasks/ChatAvatar.vue";
import RunUsageLabel from "@/components/tasks/RunUsageLabel.vue";
import type {
  MessageSender as ChatSender,
  Item,
  RunUsage,
} from "@/lib/console";
import { formatClock, formatInstant, isoInstant } from "@/lib/format";
import type { DiagramSource, InlineImage } from "@/lib/lightbox";
import type { Attachment } from "@cherrypi/sdk";
import { CheckIcon, CopyIcon } from "@lucide/vue";

withDefaults(
  defineProps<{
    item: Extract<Item, { kind: "text" }>;

    sender?: ChatSender;

    picture?: Attachment;

    usage?: RunUsage;

    copied?: boolean;
  }>(),
  {
    sender: undefined,
    picture: undefined,
    usage: undefined,
    copied: false,
  },
);

const emit = defineEmits<{
  copy: [];

  openImage: [image: InlineImage];

  openDiagram: [diagram: DiagramSource];
}>();
</script>

<i18n lang="yaml">
en:
  taskConsole:
    copyResponse: "Copy response"
ru:
  taskConsole:
    copyResponse: "Скопировать ответ"
</i18n>
