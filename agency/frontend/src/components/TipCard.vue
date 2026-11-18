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
//- inverted, single-line pill sized for a label, while these tips carry a name,

TooltipPortal
  RekaTooltipContent(
    v-bind="$attrs"
    :align="align"
    :side="side"
    :side-offset="sideOffset"
    :class="cardClass"
  )
    slot
</template>

<script setup lang="ts">
import { TooltipContent as RekaTooltipContent, TooltipPortal } from "reka-ui";
import { computed } from "vue";

defineOptions({ inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    side?: "top" | "right" | "bottom" | "left";

    align?: "start" | "center" | "end";

    sideOffset?: number;

    width?: string;
  }>(),
  { side: "top", align: "start", sideOffset: 6, width: "w-72 max-w-72" },
);

const cardClass = computed(() =>
  [
    props.width,
    "z-50 origin-(--reka-tooltip-content-transform-origin)",
    "rounded-lg border border-border-color bg-popover p-2.5",
    "text-popover-foreground shadow-md outline-hidden",
    "data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95",
    "data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95",
  ].join(" "),
);
</script>
