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
    CardTitle Spend by model
    CardDescription {{ subtitle }}
    CardAction
      span.text-11.text-muted-foreground {{ rows.length }} {{ unitLabel }}
  CardContent.p-0
    p.text-muted-foreground.py-10.text-center.text-sm(v-if="!rows.length")
      | No model usage in this range.
    Table(v-else)
      TableHeader
        TableRow
          TableHead Model
          TableHead.text-right Tasks
          TableHead.text-right Rounds
          TableHead.text-right Tokens
          TableHead.text-right(
            title="Output tokens over the time the model spent generating them"
          ) Speed
          TableHead.text-right Spend
      TableBody
        TableRow(v-for="row in rows" :key="label(row)")
          TableCell
            .flex.min-w-0.items-center.gap-2
              span.text-10.flex.size-6.shrink-0.items-center.justify-center.rounded-md.font-bold.text-white(
                :style="{ background: accent(label(row)) }"
              ) {{ initial(label(row)) }}
              .min-w-0
                strong.text-115.block.truncate.font-semibold {{ label(row) }}
                span.text-10.text-muted-foreground.block(v-if="!row.model")
                  | Unknown
          TableCell.text-right.tabular-nums {{ formatCount(row.tasks) }}
          TableCell.text-right.tabular-nums {{ formatCount(row.runs) }}
          TableCell.text-right.tabular-nums {{ formatTokens(tokens(row)) }}
          TableCell.text-right.tabular-nums {{ rate(row) }}
          TableCell.text-right.font-semibold.tabular-nums {{ formatCost(row.cost) }}
  CardFooter.justify-between.gap-3
    span.text-2xs.text-muted-foreground
      | Settled runs only; a running task joins the table when it settles.
    span.text-2xs.text-muted-foreground
      | Unknown = Pi reported no model for the run.
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
import {
  formatCost,
  formatCount,
  formatRate,
  formatTokens,
} from "@/lib/format";
import { generationRate, thinkingMs, usageAccent } from "@/lib/usage";
import type { AgencyUsageModel } from "@cherrypi/sdk";
import { computed } from "vue";

const props = defineProps<{
  rows: AgencyUsageModel[];
  subtitle: string;
}>();

const unitLabel = computed(() =>
  props.rows.length === 1 ? "model" : "models",
);

function label(row: AgencyUsageModel): string {
  const model = row.model || "Unknown";
  return row.provider ? `${row.provider}/${model}` : model;
}

function initial(value: string): string {
  return (value || "?").slice(0, 1).toUpperCase();
}

function accent(value: string): string {
  return usageAccent(value || "unknown");
}

function tokens(row: AgencyUsageModel): number {
  return Number(row.inputTokens) + Number(row.outputTokens);
}

function rate(row: AgencyUsageModel): string {
  const timing = {
    activeMs: Number(row.activeMs),
    toolMs: Number(row.toolMs),
    modelWaitMs: Number(row.modelWaitMs),
  };
  const value = generationRate(
    Number(row.outputTokens),
    thinkingMs(timing),
    timing.modelWaitMs,
  );
  return value > 0 ? formatRate(value) : "—";
}
</script>
