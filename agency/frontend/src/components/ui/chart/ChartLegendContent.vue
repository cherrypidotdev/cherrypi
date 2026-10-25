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
import { cn } from "@/lib/utils";
import type { HTMLAttributes } from "vue";
import { computed, onMounted, ref } from "vue";
import { useChart } from ".";

const props = withDefaults(
  defineProps<{
    hideIcon?: boolean;
    nameKey?: string;
    verticalAlign?: "bottom" | "top";

    class?: HTMLAttributes["class"];
  }>(),
  {
    verticalAlign: "bottom",
  },
);

const { id, config } = useChart();

const payload = computed(() =>
  Object.entries(config.value).map(([key]) => {
    return {
      key: props.nameKey || key,
      itemConfig: config.value[key],
    };
  }),
);

const containerSelector = ref("");
onMounted(() => {
  containerSelector.value = `[data-chart="chart-${id}"]>[data-vis-xy-container]`;
});
</script>

<template>
  <div
    v-if="containerSelector"
    :class="
      cn(
        'flex items-center justify-center gap-4',
        verticalAlign === 'top' ? 'pb-3' : 'pt-3',
        props.class,
      )
    "
  >
    <div
      v-for="{ key, itemConfig } in payload"
      :key="key"
      :class="
        cn(
          '[&>svg]:text-muted-foreground flex items-center gap-1.5 [&>svg]:h-3 [&>svg]:w-3',
        )
      "
    >
      <component :is="itemConfig?.icon" v-if="itemConfig?.icon" />
      <div
        v-else
        class="h-2 w-2 shrink-0 rounded-xs"
        :style="{
          backgroundColor: itemConfig?.color,
        }"
      />

      {{ itemConfig?.label }}
    </div>
  </div>
</template>
