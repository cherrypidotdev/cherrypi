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
.flex.flex-wrap.items-center.gap-1(v-if="attachments.length")
  a.inline-flex.max-w-full.items-center.gap-1.rounded-full.border.px-2.text-xs(
    v-for="attachment in attachments"
    :key="attachment.url"
    class="hover:bg-muted/60 py-0.5"
    :href="attachment.url"
    :title="`Preview ${attachment.filename}`"
    @click.prevent="emit('open', attachment)"
  )
    img.size-4.shrink-0.rounded.object-cover(
      v-if="isImageAttachment(attachment)"
      :src="attachment.url"
      :alt="attachment.filename"
    )
    FileIcon.text-muted-foreground.size-3.shrink-0(v-else)
    span.truncate {{ attachment.filename }}
</template>

<script setup lang="ts">
import { isImageAttachment } from "@/lib/attachments";
import type { Attachment } from "@cherrypi/sdk";
import { FileIcon } from "@lucide/vue";

defineProps<{ attachments: Attachment[] }>();
const emit = defineEmits<{ open: [attachment: Attachment] }>();
</script>
