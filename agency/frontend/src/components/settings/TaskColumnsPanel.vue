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
Card
  CardHeader
    CardTitle Task columns
    CardDescription
      | Reorder and show or hide the columns in every task table. Saved in
      | this browser.
    CardAction
      Button(variant="outline" size="sm" @click="reset")
        RotateCcwIcon(data-icon="inline-start")
        span Reset
  CardContent.flex.flex-col.gap-2
    p.text-muted-foreground.text-xs
      | Drag a column to reorder it, or use the arrows.
    ul.flex.flex-col.gap-1
      li.flex.items-center.gap-2.rounded-md.border.px-2(
        v-for="(column, index) in columns"
        :key="column.key"
        class="py-1.5"
        :class="dropIndex === index ? 'border-ring' : ''"
        @dragover.prevent="onDragOver(index)"
        @drop.prevent="onDrop(index)"
      )
        span.text-muted-foreground.cursor-grab(
          draggable="true"
          :title="`Drag to reorder ${column.label}`"
          @dragstart="onDragStart(index)"
          @dragend="onDragEnd"
        )
          GripVerticalIcon.size-4
        span.flex-1.truncate.text-sm {{ column.label }}
        .flex.items-center(class="gap-0.5")
          Button(
            variant="ghost"
            size="icon-sm"
            :title="`Move ${column.label} up`"
            :aria-label="`Move ${column.label} up`"
            @click="moveBy(column.key, -1)"
            :disabled="index === 0"
          )
            ChevronUpIcon.size-4
          Button(
            variant="ghost"
            size="icon-sm"
            :title="`Move ${column.label} down`"
            :aria-label="`Move ${column.label} down`"
            @click="moveBy(column.key, 1)"
            :disabled="index === columns.length - 1"
          )
            ChevronDownIcon.size-4
        Switch(
          :model-value="column.visible"
          :aria-label="`Show ${column.label} column`"
          @update:model-value="(value: boolean) => setVisible(column.key, value)"
        )
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { useTaskColumns } from "@/composables/useTaskColumns";
import {
  ChevronDownIcon,
  ChevronUpIcon,
  GripVerticalIcon,
  RotateCcwIcon,
} from "@lucide/vue";
import { ref } from "vue";

const { columns, move, moveBy, setVisible, reset } = useTaskColumns();

// Native HTML5 drag-and-drop: the handle is the draggable element and the row
// is the drop target, so a drop lands the dragged column at that row.
const dragIndex = ref<number | null>(null);
const dropIndex = ref<number | null>(null);

function onDragStart(index: number) {
  dragIndex.value = index;
}

function onDragOver(index: number) {
  dropIndex.value = index;
}

function onDrop(index: number) {
  if (dragIndex.value !== null) move(dragIndex.value, index);
  onDragEnd();
}

function onDragEnd() {
  dragIndex.value = null;
  dropIndex.value = null;
}
</script>
