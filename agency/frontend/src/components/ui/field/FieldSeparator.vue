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
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import type { HTMLAttributes } from "vue";

const props = defineProps<{
  class?: HTMLAttributes["class"];
}>();
</script>

<template>
  <div
    data-slot="field-separator"
    :data-content="!!$slots.default"
    :class="
      cn(
        '-my-2 h-5 text-sm group-data-[variant=outline]/field-group:-mb-2 relative',
        props.class,
      )
    "
  >
    <Separator class="absolute inset-0 top-1/2" />
    <span
      v-if="$slots.default"
      class="text-muted-foreground px-2 relative mx-auto block w-fit bg-background"
      data-slot="field-separator-content"
    >
      <slot />
    </span>
  </div>
</template>
