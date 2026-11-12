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
.flex.w-full.flex-col.gap-1.rounded-lg.border.border-dashed.px-3.py-2.text-xs(
  class="bg-muted/40"
  :title="hint"
  data-testid="console-notice"
)
  .flex.flex-wrap.items-center.gap-2
    component.text-muted-foreground.shrink-0(class="size-3.5" :is="icon")
    span.font-medium {{ label }}
    span.text-muted-foreground(v-if="reason") · {{ reason }}
    //- Who or what set the line off: a task state change names its actor, so a

    slot(name="actor")
    time.text-muted-foreground.ml-auto.tabular-nums(
      v-if="at"
      :datetime="isoInstant(at)"
      :title="formatInstant(at)"
    ) {{ formatClock(at) }}

  slot
</template>

<script setup lang="ts">
import { formatClock, formatInstant, isoInstant } from "@/lib/format";
import type { LucideIcon } from "@lucide/vue";

defineProps<{
  icon: LucideIcon;

  label: string;

  reason?: string;

  at?: number;

  hint?: string;
}>();
</script>
