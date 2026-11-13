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
.flex.flex-wrap.items-center(class="gap-1.5")
  span.text-muted-foreground.text-xs.font-semibold.tracking-wide.uppercase
    | {{ $t("pluginTemplate.insert") }}
  Button.h-6.rounded.px-2.font-mono.text-xs(
    v-for="name in quickVariables"
    :key="name"
    type="button"
    variant="outline"
    size="sm"
    :title="variables[name]?.description"
    @click="emit('insert', name)"
  ) {{ name }}
  details.relative.ml-auto
    summary.text-muted-foreground.cursor-pointer.list-none.rounded.px-2.py-1.text-xs(
      class="hover:bg-muted"
    ) {{ $t("pluginTemplate.allVariables") }}
    .bg-popover.absolute.right-0.z-30.mt-1.max-h-72.w-80.overflow-auto.rounded-md.border.p-1.shadow-md
      template(v-for="name in variableNames" :key="name")
        button.w-full.rounded.px-2.py-1.text-left.text-xs(
          type="button"
          class="hover:bg-muted"
          @click="emit('insert', name)"
        )
          code.font-mono.text-xs.text-violet-700 {{ variableLabel(name) }}
          span.text-muted-foreground.block.text-xs {{ variables[name]?.description }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import type { PluginTemplateVariable } from "@/lib/plugins";
import { computed } from "vue";

const props = defineProps<{
  variables: Record<string, PluginTemplateVariable>;
}>();

const emit = defineEmits<{
  insert: [name: string];
}>();

const variableNames = computed(() => Object.keys(props.variables));
const quickVariables = computed(() => variableNames.value.slice(0, 6));

function variableLabel(name: string): string {
  return `{{ ${name} }}`;
}
</script>

<i18n lang="yaml">
en:
  pluginTemplate:
    insert: "Insert"
    allVariables: "All variables"
ru:
  pluginTemplate:
    insert: "Вставить"
    allVariables: "Все переменные"
</i18n>
