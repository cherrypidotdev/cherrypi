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
RouterLink.text-muted-foreground.flex.items-center.justify-center.rounded-md.py-2.transition-colors(
  v-if="collapsed && to"
  :to="to"
  :title="title"
  :aria-label="title"
  :class="[active ? 'text-primary' : 'hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground']"
)
  component.size-5.shrink-0(v-if="icon" :is="icon")
.flex.flex-col(v-else-if="collapsed" class="gap-0.5")
  slot
.flex.flex-col(v-else)
  button.text-muted-foreground.flex.w-full.items-center.rounded-md.px-3.py-2.text-xs.font-medium.transition-colors(
    type="button"
    class="hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground gap-1.5"
    :aria-expanded="open"
    @click="emit('toggle')"
  )
    component.size-4.shrink-0(v-if="icon" :is="icon")
    span.min-w-0.flex-1.truncate.text-left {{ title }}
    span.shrink-0(v-if="count" class="text-[10px]") {{ count }}
    component.size-3.shrink-0(:is="open ? ChevronDownIcon : ChevronRightIcon")
  .flex.flex-col(class="gap-0.5" v-show="open")
    slot
</template>

<script setup lang="ts">
import { ChevronDownIcon, ChevronRightIcon } from "@lucide/vue";
import type { Component } from "vue";
import { RouterLink, type RouteLocationRaw } from "vue-router";

withDefaults(
  defineProps<{
    title: string;
    open: boolean;

    collapsed?: boolean;

    count?: number;

    icon?: Component;

    to?: RouteLocationRaw;

    active?: boolean;
  }>(),
  {
    collapsed: false,
    count: undefined,
    icon: undefined,
    to: undefined,
    active: false,
  },
);

const emit = defineEmits<{ toggle: [] }>();
</script>
