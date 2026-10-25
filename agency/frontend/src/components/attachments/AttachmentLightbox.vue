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
Dialog(:open="open" @update:open="emit('update:open', $event)")
  DialogContent.flex.w-screen.max-w-none.flex-col.overflow-hidden.rounded-none.border-0.p-0.text-white(
    class="h-[100dvh] bg-black/95 sm:max-w-none"
    :show-close-button="false"
  )
    DialogTitle.sr-only Task attachments
    DialogDescription.sr-only Browse every file attached to this task.

    header.flex.shrink-0.items-center.gap-3.px-4.py-3(class="bg-black/40")
      .flex.min-w-0.flex-col
        span.truncate.text-sm.font-medium {{ active?.filename ?? "Task attachments" }}
        span.text-xs(v-if="active" class="text-white/60")
          | {{ activeIndex + 1 }} of {{ attachments.length }}
      .ml-auto.flex.shrink-0.items-center.gap-1
        Button.text-white(
          v-if="active"
          as-child
          variant="ghost"
          size="icon-sm"
          title="Open in new tab"
          aria-label="Open in new tab"
          class="hover:bg-white/10 hover:text-white"
        )
          a(rel="noopener noreferrer" :href="active.url" target="_blank")
            ExternalLinkIcon.size-4
        Button.text-white(
          type="button"
          variant="ghost"
          size="icon-sm"
          title="Close"
          aria-label="Close"
          class="hover:bg-white/10 hover:text-white"
          @click="close"
        )
          XIcon.size-5

    .relative.flex.min-h-0.flex-1.items-center.justify-center.overflow-hidden.p-4
      Button.absolute.left-3.z-10.text-white(
        v-if="attachments.length > 1"
        type="button"
        variant="ghost"
        size="icon-lg"
        title="Previous attachment"
        aria-label="Previous attachment"
        class="hover:bg-white/10 hover:text-white"
        @click="move(-1)"
      )
        ChevronLeftIcon.size-7

      template(v-if="active")
        img.max-h-full.max-w-full.rounded-md.object-contain.select-none(
          v-if="isRenderable(active)"
          :src="active.url"
          draggable="false"
          :alt="active.filename"
        )
        .flex.max-w-xl.flex-col.items-center.gap-3.text-center(v-else)
          FileIcon.size-16(class="text-white/50")
          .flex.flex-col.items-center.gap-1
            span.text-sm.font-medium.break-all {{ active.filename }}
            a.text-xs.break-all(
              class="text-white/60 hover:text-white hover:underline"
              rel="noopener noreferrer"
              :href="active.url"
              target="_blank"
            ) {{ active.url }}
          a.inline-flex.items-center.gap-1.rounded-md.border.px-3.text-xs(
            class="border-white/20 py-1.5 hover:bg-white/10"
            rel="noopener noreferrer"
            :href="active.url"
            target="_blank"
          )
            ExternalLinkIcon.size-3
            span Open file

      Button.absolute.right-3.z-10.text-white(
        v-if="attachments.length > 1"
        type="button"
        variant="ghost"
        size="icon-lg"
        title="Next attachment"
        aria-label="Next attachment"
        class="hover:bg-white/10 hover:text-white"
        @click="move(1)"
      )
        ChevronRightIcon.size-7

      p.absolute.right-4.bottom-2.left-4.truncate.text-center.text-xs(
        v-if="active"
        class="text-white/60"
      ) {{ active?.filename }}

    .flex.shrink-0.justify-center.gap-2.overflow-x-auto.px-4.py-3(
      v-if="attachments.length > 1"
      class="bg-black/40"
    )
      button.size-14.shrink-0.cursor-pointer.overflow-hidden.rounded-md.border-2.transition-colors(
        v-for="(attachment, index) in attachments"
        :key="attachment.id.toString()"
        type="button"
        :class="index === activeIndex ? 'border-white' : 'border-transparent hover:border-white/40'"
        :title="attachment.filename"
        :aria-label="attachment.filename"
        @click="activeIndex = index"
      )
        img.size-full.object-cover(
          v-if="isRenderable(attachment)"
          :src="attachment.url"
          draggable="false"
          :alt="attachment.filename"
        )
        span.flex.size-full.items-center.justify-center(v-else class="text-white/60")
          FileIcon.size-6
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { isImageAttachment } from "@/lib/attachments";
import type { Attachment } from "@cherrypi/sdk";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  ExternalLinkIcon,
  FileIcon,
  XIcon,
} from "@lucide/vue";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    open: boolean;

    attachments: Attachment[];

    startIndex?: number;
  }>(),
  { startIndex: 0 },
);
const emit = defineEmits<{ "update:open": [open: boolean] }>();

const activeIndex = ref(0);
const active = computed<Attachment | undefined>(
  () => props.attachments[activeIndex.value],
);

function isRenderable(attachment: Attachment): boolean {
  return isImageAttachment(attachment);
}

function clampIndex(index: number): number {
  if (!props.attachments.length) return 0;
  return Math.min(Math.max(index, 0), props.attachments.length - 1);
}

function move(delta: number) {
  if (props.attachments.length < 2) return;
  activeIndex.value =
    (activeIndex.value + delta + props.attachments.length) %
    props.attachments.length;
}

function close() {
  emit("update:open", false);
}

function onKeydown(event: KeyboardEvent) {
  if (!props.open) return;
  if (event.key === "ArrowLeft") {
    event.preventDefault();
    move(-1);
  } else if (event.key === "ArrowRight") {
    event.preventDefault();
    move(1);
  }
}

watch(
  () => [props.open, props.startIndex, props.attachments.length] as const,
  ([open, startIndex]) => {
    if (open) activeIndex.value = clampIndex(startIndex);
    else activeIndex.value = clampIndex(activeIndex.value);
  },
  { immediate: true },
);

onMounted(() => window.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => window.removeEventListener("keydown", onKeydown));
</script>
