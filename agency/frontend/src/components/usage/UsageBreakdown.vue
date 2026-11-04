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
    .flex.flex-wrap.items-start.justify-between.gap-3
      div
        CardTitle {{ title }}
        CardDescription {{ subtitle }}
      CardAction
        span.text-11.text-muted-foreground {{ rows.length }} {{ unitLabel }}
  CardContent.p-0
    p.text-muted-foreground.py-10.text-center.text-sm(v-if="!rows.length")
      | No completed tasks in this range.
    Table(v-else)
      TableHeader
        TableRow
          TableHead {{ entityLabel }}
          TableHead.text-right Tasks
          TableHead.text-right TTC
          TableHead.text-right Active
          TableHead.text-right Rounds
          TableHead.text-right Tokens
          TableHead.text-right Spend
      TableBody
        TableRow(v-for="row in rows" :key="row.id")
          TableCell
            .flex.min-w-0.items-center.gap-2
              span.text-10.flex.size-6.shrink-0.items-center.justify-center.rounded-md.font-bold.text-white(
                :style="{ background: row.accent }"
              ) {{ row.name.slice(0, 1).toUpperCase() }}
              strong.text-115.block.truncate.font-semibold {{ row.name }}
          TableCell.text-right.tabular-nums {{ row.tasks }}
          TableCell.text-right.tabular-nums {{ ttc(row) }}
          TableCell.text-right.tabular-nums {{ formatDuration(row.activeMs) }}
          TableCell.text-right.tabular-nums {{ row.rounds.toFixed(1) }}
          TableCell.text-right.tabular-nums {{ formatTokens(row.tokens) }}
          TableCell.text-right.font-semibold.tabular-nums {{ formatCost(row.cost) }}
  CardFooter.justify-between.gap-3
    span.text-2xs.text-muted-foreground
      | Averages per completed task in range.
    span.text-2xs.text-muted-foreground
      | TTC queue → completion · active excludes idle waits
</template>

<script setup lang="ts">
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { UsageBreakdownRow } from "@/composables/useUsageAnalytics";
import { formatCost, formatDuration, formatTokens } from "@/lib/format";
import { computed } from "vue";

const props = defineProps<{
  rows: UsageBreakdownRow[];
  title: string;
  subtitle: string;
  unit: string;
}>();

const unitLabel = computed(() =>
  props.rows.length === 1 ? props.unit.replace(/s$/, "") : props.unit,
);
const entityLabel = computed(() =>
  props.unit === "agents" ? "Agent" : "Project",
);

function ttc(row: UsageBreakdownRow): string {
  return row.ttcSamples ? formatDuration(row.ttcMs) : "—";
}
</script>
