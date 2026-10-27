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
    CardTitle {{ title }}
    CardDescription {{ subtitle }}
    CardAction
      span.text-11.text-muted-foreground {{ rows.length }} {{ unitLabel }}
  CardContent
    p.text-muted-foreground.py-10.text-center.text-sm(v-if="!rows.length")
      | No usage in this range.
    .flex.flex-col(v-else)
      .flex.items-center.justify-between.gap-3.border-b.py-2(
        v-for="row in rows"
        :key="row.id"
        class="last:border-0"
      )
        .flex.min-w-0.items-center.gap-2
          span.text-10.flex.size-6.shrink-0.items-center.justify-center.rounded-md.font-bold.text-white(
            :style="{ background: row.accent }"
          ) {{ row.name.slice(0, 1).toUpperCase() }}
          .min-w-0
            strong.text-115.block.truncate.font-semibold {{ row.name }}
            span.text-10.text-muted-foreground.block {{ row.tasks }} tasks · {{ row.runs }} runs
            .bg-muted.mt-1.h-1.w-full.overflow-hidden.rounded-full
              i.block.h-full.rounded-full(
                :style="{ width: barWidth(row.cost), background: row.accent }"
              )
        .shrink-0.text-right
          strong.text-115.block.tabular-nums {{ formatCost(row.cost) }}
          span.text-10.text-muted-foreground.block {{ formatTokens(row.tokens) }} tok
</template>

<script setup lang="ts">
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { UsageBreakdownRow } from "@/composables/useUsageAnalytics";
import { formatCost, formatTokens } from "@/lib/format";
import { computed } from "vue";

const props = defineProps<{
  rows: UsageBreakdownRow[];
  title: string;
  subtitle: string;
  unit: string;
}>();

const max = computed(() =>
  Math.max(...props.rows.map((row) => row.cost), 0.0001),
);
const unitLabel = computed(() =>
  props.rows.length === 1 ? props.unit.replace(/s$/, "") : props.unit,
);

function barWidth(cost: number): string {
  return `${Math.max(2, (cost / max.value) * 100)}%`;
}
</script>
