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

<script setup lang="ts">
import type { DialogContentEmits, DialogContentProps } from "reka-ui";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { XIcon } from "@lucide/vue";
import { reactiveOmit } from "@vueuse/core";
import {
  DialogClose,
  DialogContent,
  DialogPortal,
  useForwardPropsEmits,
} from "reka-ui";
import type { HTMLAttributes } from "vue";
import DialogOverlay from "./DialogOverlay.vue";

defineOptions({
  inheritAttrs: false,
});

const props = withDefaults(
  defineProps<
    DialogContentProps & {
      class?: HTMLAttributes["class"];
      showCloseButton?: boolean;
    }
  >(),
  {
    showCloseButton: true,
  },
);
const emits = defineEmits<DialogContentEmits>();

const delegatedProps = reactiveOmit(props, "class");

const forwarded = useForwardPropsEmits(delegatedProps, emits);
</script>

<template>
  <DialogPortal>
    <DialogOverlay />
    <DialogContent
      data-slot="dialog-content"
      v-bind="{ ...$attrs, ...forwarded }"
      :class="
        cn(
          'bg-popover text-popover-foreground data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95 ring-foreground/10 grid max-w-[calc(100%-2rem)] gap-4 rounded-xl p-4 text-sm ring-1 duration-100 sm:max-w-sm fixed top-1/2 left-1/2 z-50 w-full -translate-x-1/2 -translate-y-1/2 outline-none',
          props.class,
        )
      "
    >
      <slot />

      <DialogClose v-if="showCloseButton" data-slot="dialog-close" as-child>
        <Button variant="ghost" class="absolute top-2 right-2" size="icon-sm">
          <XIcon />
          <span class="sr-only">Close</span>
        </Button>
      </DialogClose>
    </DialogContent>
  </DialogPortal>
</template>
