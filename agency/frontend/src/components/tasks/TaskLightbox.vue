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
    DialogTitle.sr-only Task files
    DialogDescription.sr-only Browse every file attached to or produced by this task.

    header.flex.shrink-0.items-center.gap-3.px-4.py-3(class="bg-black/40")
      .flex.min-w-0.flex-col
        span.truncate.text-sm.font-medium {{ active?.filename ?? "Task files" }}
        span.text-xs(v-if="active" class="text-white/60")
          | {{ active.source }} · {{ activeIndex + 1 }} of {{ items.length }}
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
        v-if="items.length > 1"
        type="button"
        variant="ghost"
        size="icon-lg"
        title="Previous item"
        aria-label="Previous item"
        class="hover:bg-white/10 hover:text-white"
        @click="move(-1)"
      )
        ChevronLeftIcon.size-7

      template(v-if="active")
        .flex.h-full.w-full.flex-col.items-center.justify-center(
          v-if="active.diagram"
          ref="diagramHost"
          data-test="lightbox-diagram"
        )
        img.max-h-full.max-w-full.rounded-md.object-contain.select-none(
          v-else-if="active.image"
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
        v-if="items.length > 1"
        type="button"
        variant="ghost"
        size="icon-lg"
        title="Next item"
        aria-label="Next item"
        class="hover:bg-white/10 hover:text-white"
        @click="move(1)"
      )
        ChevronRightIcon.size-7

      p.absolute.right-4.bottom-2.left-4.truncate.text-center.text-xs(
        v-if="active"
        class="text-white/60"
      ) {{ active?.source }} · {{ active?.filename }}

    .flex.shrink-0.justify-center.gap-2.overflow-x-auto.px-4.py-3(
      v-if="items.length > 1"
      class="bg-black/40"
    )
      button.size-14.shrink-0.cursor-pointer.overflow-hidden.rounded-md.border-2.transition-colors(
        v-for="(item, index) in items"
        :key="item.key"
        type="button"
        :class="index === activeIndex ? 'border-white' : 'border-transparent hover:border-white/40'"
        :title="`${item.source} · ${item.filename}`"
        :aria-label="item.filename"
        @click="activeIndex = index"
      )
        img.size-full.object-cover(
          v-if="item.image"
          :src="item.url"
          draggable="false"
          :alt="item.filename"
        )
        span.flex.size-full.items-center.justify-center(
          v-else-if="item.diagram"
          class="text-white/60"
        )
          WorkflowIcon.size-6
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
import type { LightboxItem } from "@/lib/lightbox";
import { attachMermaidTools } from "@/lib/mermaidViewer";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  ExternalLinkIcon,
  FileIcon,
  WorkflowIcon,
  XIcon,
} from "@lucide/vue";
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  watchEffect,
} from "vue";

const props = withDefaults(
  defineProps<{
    open: boolean;

    items: LightboxItem[];

    startIndex?: number;
  }>(),
  { startIndex: 0 },
);
const emit = defineEmits<{ "update:open": [open: boolean] }>();

const activeIndex = ref(0);
const active = computed<LightboxItem | undefined>(
  () => props.items[activeIndex.value],
);

function clampIndex(index: number): number {
  if (!props.items.length) return 0;
  return Math.min(Math.max(index, 0), props.items.length - 1);
}

function move(delta: number) {
  if (props.items.length < 2) return;
  activeIndex.value =
    (activeIndex.value + delta + props.items.length) % props.items.length;
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
  () => [props.open, props.startIndex, props.items.length] as const,
  ([open, startIndex]) => {
    if (open) activeIndex.value = clampIndex(startIndex);
    else activeIndex.value = clampIndex(activeIndex.value);
  },
  { immediate: true },
);

const diagramHost = ref<HTMLElement>();

watchEffect((onCleanup) => {
  const host = diagramHost.value;
  const diagram = props.open ? active.value?.diagram : undefined;
  if (!host || !diagram) return;
  host.innerHTML = diagram.svg;
  const svg = host.querySelector("svg");
  if (!svg) return;

  svg.removeAttribute("style");

  host.classList.add("mermaid-always");
  const tools = attachMermaidTools(host, { code: diagram.code });
  onCleanup(() => {
    tools.dispose();
    host.replaceChildren();
    host.classList.remove("mermaid-always");
  });
});

onMounted(() => window.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => window.removeEventListener("keydown", onKeydown));
</script>
