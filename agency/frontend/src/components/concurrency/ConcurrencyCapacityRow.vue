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
.grid.grid-cols-1.gap-x-4.gap-y-1.py-2(
  class="sm:grid-cols-[minmax(140px,190px)_minmax(0,1fr)] sm:items-center"
)
  .flex.min-w-0.flex-col(class="gap-0.5")
    .flex.min-w-0.items-center.gap-2
      span.text-2xs.inline-flex.items-center.gap-1.rounded-full.border.px-2(
        v-if="row.kind === 'tag' && row.tag"
        class="py-0.5"
        :style="tagStyle(row.tag.color)"
      )
        span.rounded-full.bg-current(class="size-1.5")
        | {{ row.tag.name }}
      span.truncate.text-sm.font-medium(v-else) {{ row.label }}
      span.text-3xs.rounded-full.font-medium.text-amber-600(
        v-if="row.full"
        class="bg-amber-500/12 px-1.5 py-0.5 dark:text-amber-400"
      ) At cap
    span.text-3xs.text-muted-foreground(v-if="row.detail") {{ row.detail }}
  ConcurrencyTrack(
    :label="row.label"
    :used="row.used"
    :cap="row.cap"
    :waiting="row.waiting"
  )
</template>

<script setup lang="ts">
import ConcurrencyTrack from "@/components/concurrency/ConcurrencyTrack.vue";
import type { CapacityRow } from "@/lib/concurrency";
import { tagStyle } from "@/lib/task";

defineProps<{ row: CapacityRow }>();
</script>
