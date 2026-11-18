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
TooltipProvider(:delay-duration="120")
  Tooltip
    TooltipTrigger(as-child)
      component.inline-flex.shrink-0.items-center.justify-center.rounded-full(
        v-bind="to ? { to, ...$attrs } : { tabindex: 0, ...$attrs }"
        :is="to ? RouterLink : 'span'"
        :aria-label="ariaLabel"
        :class="to ? 'transition-opacity hover:opacity-80' : ''"
        @click.stop
      )
        img.rounded-full.object-cover(
          v-if="src"
          :src="src"
          :alt="name"
          :style="box"
        )
        span(v-else-if="seed" :innerHTML="generated")
        ActorAvatar(v-else :name="name" :size="size")
    TipCard(:width="cardWidth" :data-slot="cardSlot")
      slot
        .flex.flex-col(class="gap-0.5")
          span.font-medium {{ name }}
          span.text-muted-foreground(v-if="meta") {{ meta }}
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import TipCard from "@/components/TipCard.vue";
import {
  Tooltip,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { toSvg } from "jdenticon";
import { computed } from "vue";
import { RouterLink, type RouteLocationRaw } from "vue-router";

defineOptions({ inheritAttrs: false });

// the name and an optional second line unless the caller supplies its own.
const props = withDefaults(
  defineProps<{
    name: string;

    to?: RouteLocationRaw | null;

    src?: string;

    seed?: string;

    meta?: string;

    size?: number;

    cardSlot?: string;

    cardWidth?: string;
  }>(),
  {
    to: null,
    src: "",
    seed: "",
    meta: "",
    size: 20,
    cardSlot: "avatar-tip",
    cardWidth: "w-56 max-w-56",
  },
);

const box = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`,
}));

const generated = computed(() =>
  props.seed ? toSvg(props.seed, props.size) : "",
);

const ariaLabel = computed(() =>
  props.meta ? `${props.name} (${props.meta})` : props.name,
);
</script>
