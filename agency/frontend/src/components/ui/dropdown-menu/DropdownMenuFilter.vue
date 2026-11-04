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
import type { DropdownMenuFilterEmits, DropdownMenuFilterProps } from "reka-ui";

import { cn } from "@/lib/utils";
import { reactiveOmit } from "@vueuse/core";
import { DropdownMenuFilter, useForwardPropsEmits } from "reka-ui";
import { onBeforeUnmount, type HTMLAttributes } from "vue";

const props = defineProps<
  DropdownMenuFilterProps & { class?: HTMLAttributes["class"] }
>();
const emits = defineEmits<DropdownMenuFilterEmits>();

const delegatedProps = reactiveOmit(props, "class");

const forwarded = useForwardPropsEmits(delegatedProps, emits);

onBeforeUnmount(() => emits("update:modelValue", ""));
</script>

<template>
  <DropdownMenuFilter
    data-slot="dropdown-menu-filter"
    v-bind="forwarded"
    :class="
      cn(
        'border-input placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 mb-1 h-7 w-full rounded-md border bg-transparent px-2 text-sm outline-none focus-visible:ring-3',
        props.class,
      )
    "
  />
</template>
