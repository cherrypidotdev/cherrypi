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
import type {
  DropdownMenuCheckboxItemEmits,
  DropdownMenuCheckboxItemProps,
} from "reka-ui";

import { cn } from "@/lib/utils";
import { CheckIcon } from "@lucide/vue";
import { reactiveOmit } from "@vueuse/core";
import {
  DropdownMenuCheckboxItem,
  DropdownMenuItemIndicator,
  useForwardPropsEmits,
} from "reka-ui";
import type { HTMLAttributes } from "vue";

const props = defineProps<
  DropdownMenuCheckboxItemProps & { class?: HTMLAttributes["class"] }
>();
const emits = defineEmits<DropdownMenuCheckboxItemEmits>();

const delegatedProps = reactiveOmit(props, "class");

const forwarded = useForwardPropsEmits(delegatedProps, emits);
</script>

<template>
  <DropdownMenuCheckboxItem
    data-slot="dropdown-menu-checkbox-item"
    v-bind="forwarded"
    :class="
      cn(
        'focus:bg-accent focus:text-accent-foreground focus:**:text-accent-foreground gap-1.5 rounded-md py-1 pr-8 pl-1.5 text-sm data-inset:pl-7 [&_svg:not([class*=size-])]:size-4 relative flex cursor-default items-center outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0',
        props.class,
      )
    "
  >
    <span
      class="absolute right-2 flex items-center justify-center pointer-events-none"
      data-slot="dropdown-menu-checkbox-item-indicator"
    >
      <DropdownMenuItemIndicator>
        <slot name="indicator-icon">
          <CheckIcon />
        </slot>
      </DropdownMenuItemIndicator>
    </span>
    <slot />
  </DropdownMenuCheckboxItem>
</template>
