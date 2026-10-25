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
  DialogContent.flex.flex-col.gap-0.overflow-hidden.p-0(
    class="h-[80vh] sm:max-w-5xl"
    :show-close-button="false"
  )
    DialogHeader.flex.flex-col.gap-1.border-b.px-4.py-3
      DialogTitle {{ title }}
      DialogDescription(v-if="description") {{ description }}
    .relative.min-h-0.flex-1
      TldrawCanvas(
        v-if="open"
        ref="canvas"
        :snapshot="snapshot"
        :background-image="backgroundImage"
        @ready="ready = true"
      )
    DialogFooter.flex.items-center.gap-2.border-t.px-4.py-3(class="sm:justify-between")
      span.text-muted-foreground.text-xs
        template(v-if="saving") Saving…
        template(v-else-if="!ready") Loading editor…
        template(v-else) Draw, then save to attach the sketch.
      .flex.items-center.gap-2
        Button(
          type="button"
          variant="outline"
          @click="close"
          :disabled="saving"
        )
          | Cancel
        Button(type="button" @click="save" :disabled="!ready || saving")
          Spinner(v-if="saving" data-icon="inline-start")
          | Save sketch
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import type { TLEditorSnapshot } from "tldraw";
import { ref } from "vue";

import TldrawCanvas, { type SketchBackgroundImage } from "./TldrawCanvas.vue";

withDefaults(
  defineProps<{
    open: boolean;
    title?: string;
    description?: string;
    snapshot?: unknown;
    backgroundImage?: SketchBackgroundImage;
    saving?: boolean;
  }>(),
  {
    title: "Sketch",
    description: "Draw with the sketch tools above, then save to attach it.",
    snapshot: undefined,
    backgroundImage: undefined,
    saving: false,
  },
);

const emit = defineEmits<{
  "update:open": [open: boolean];
  save: [payload: { png: Blob; snapshot: TLEditorSnapshot }];
}>();

const canvas = ref<InstanceType<typeof TldrawCanvas>>();
const ready = ref(false);

function close() {
  emit("update:open", false);
}

async function save() {
  const captured = await canvas.value?.capture();
  if (!captured) return;
  emit("save", captured);
}
</script>
