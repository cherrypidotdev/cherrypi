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
        CardTitle Averages per task
        CardDescription
          | Completed tasks in range. TTC is the time outside backlog since the first
          | queue, active is the agent's work time, and thinking is the model's own
          | work inside it.
      .flex.items-center.gap-2
        span.text-11.text-muted-foreground {{ tasksLabel }}
        DropdownMenu
          DropdownMenuTrigger(as-child)
            Button(
              variant="outline"
              size="sm"
              :title="tags.length ? 'Filter the insights by tag' : 'No tags on completed tasks'"
              :disabled="!tags.length"
            )
              TagIcon(data-icon="inline-start")
              span {{ tagLabel }}
          DropdownMenuContent.max-h-72.w-56.overflow-y-auto(align="end")
            DropdownMenuLabel Tags
            DropdownMenuSeparator
            p.text-muted-foreground.px-2.py-1.text-xs(v-if="!tags.length")
              | No tags on completed tasks
            DropdownMenuCheckboxItem(
              v-for="tag in tags"
              :key="tag.id"
              :model-value="selectedTagIds.includes(tag.id)"
              @update:model-value="toggleTag(tag.id)"
              @select.prevent
            )
              span.size-2.shrink-0.rounded-full(
                :style="{ background: tag.color }"
              )
              span {{ tag.name }}
              span.text-muted-foreground(v-if="tag.scope") · {{ tag.scope }}
            template(v-if="selectedTagIds.length")
              DropdownMenuSeparator
              DropdownMenuItem(@select="clearTags") Clear tag filter
  CardContent
    .grid.grid-cols-2.gap-3(class="sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-8")
      .border-border.flex.flex-col.rounded-lg.border.p-3(
        v-for="tile in tiles"
        :key="tile.key"
        class="bg-muted/30"
      )
        .flex.items-center.justify-between.gap-2
          span.text-muted-foreground.text-2xs.font-semibold {{ tile.label }}
          span.text-10.text-muted-foreground {{ tile.hint }}
        .mt-1.text-xl.leading-tight.font-semibold.tracking-tight.tabular-nums {{ tile.value }}
        .text-10.text-muted-foreground(class="mt-0.5") {{ tile.sub }}
        .mt-2.h-7
          UsageSparkline(:color="tile.color" :values="tile.values")
  CardFooter.justify-between.gap-3
    span.text-2xs.text-muted-foreground
      | {{ tasksLabel }} · {{ rangeLabel }}
    span.text-2xs.text-muted-foreground
      template(v-if="selectedTagIds.length") Tag-filtered · any selected tag
      template(v-else) Outside backlog · active excludes idle waits · thinking is model work
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import UsageSparkline from "@/components/usage/UsageSparkline.vue";
import type { UsageInsightTag } from "@/composables/useUsageAnalytics";
import {
  formatDuration,
  formatDurationPrecise,
  formatRate,
  formatTokens,
} from "@/lib/format";
import {
  usageRangeLabel,
  type UsageInsightDay,
  type UsageInsights,
  type UsageRange,
} from "@/lib/usage";
import { TagIcon } from "@lucide/vue";
import { computed } from "vue";

const props = defineProps<{
  insights: UsageInsights;
  daily: UsageInsightDay[];
  range: UsageRange;
  tags: UsageInsightTag[];
}>();

const selectedTagIds = defineModel<string[]>("tagIds", {
  default: () => [],
});

const rangeLabel = computed(() => usageRangeLabel(props.range));

const tasksLabel = computed(() => {
  const count = props.insights.tasks;
  if (!count) return "No completed tasks";
  return `${count} completed task${count === 1 ? "" : "s"}`;
});

const tagLabel = computed(() => {
  const count = selectedTagIds.value.length;
  if (!count) return "All tags";
  if (count === 1) {
    return (
      props.tags.find((tag) => tag.id === selectedTagIds.value[0])?.name ??
      "1 tag"
    );
  }
  return `${count} tags`;
});

function toggleTag(id: string) {
  const next = new Set(selectedTagIds.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  selectedTagIds.value = [...next];
}

function clearTags() {
  selectedTagIds.value = [];
}

function money(value: number): string {
  if (value >= 1) return `$${value.toFixed(2)}`;
  return `$${value.toFixed(4)}`;
}

function last(values: number[]): number[] {
  return values.slice(-18);
}

const tiles = computed(() => {
  const insights = props.insights;
  const hasTasks = insights.tasks > 0;
  const showsTtc = hasTasks && insights.ttcSamples > 0;
  const ttcSub =
    insights.ttcSamples === insights.tasks
      ? "outside backlog"
      : `${insights.ttcSamples} of ${insights.tasks} timed`;

  return [
    {
      key: "ttc",
      label: "TTC",
      hint: "avg / task",
      value: showsTtc ? formatDuration(insights.ttcMs) : "—",
      sub: ttcSub,
      color: "var(--chart-1)",
      values: last(props.daily.map((day) => day.ttcMs)),
    },
    {
      key: "active",
      label: "TTC active",
      hint: "agent work",
      value: hasTasks ? formatDuration(insights.activeMs) : "—",
      sub: "agent work / task",
      color: "var(--chart-2)",
      values: last(props.daily.map((day) => day.activeMs)),
    },
    {
      key: "thinking",
      label: "Thinking",
      hint: "model work",
      value: hasTasks ? formatDuration(insights.thinkingMs) : "—",
      sub: insights.modelWaitMs
        ? `${formatDuration(insights.modelWaitMs)} waiting`
        : "per task",
      color: "var(--chart-3)",
      values: last(props.daily.map((day) => day.thinkingMs)),
    },
    {
      key: "ttft",
      label: "TTFT",
      hint: "first token",
      value:
        hasTasks && insights.ttftRuns
          ? formatDurationPrecise(insights.ttftPerRunMs)
          : "—",
      sub: insights.ttftRuns
        ? `avg / run · ${insights.ttftRuns} runs`
        : "no measured runs",
      color: "var(--chart-4)",
      values: last(props.daily.map((day) => day.ttftPerRunMs)),
    },
    {
      key: "rate",
      label: "Speed",
      hint: "output tok/s",
      value:
        hasTasks && insights.tokensPerSecond
          ? formatRate(insights.tokensPerSecond)
          : "—",
      sub: "generation only",
      color: "var(--chart-5)",
      values: last(props.daily.map((day) => day.tokensPerSecond)),
    },
    {
      key: "rounds",
      label: "Rounds",
      hint: "agent turns",
      value: hasTasks ? insights.rounds.toFixed(1) : "—",
      sub: "per task",
      color: "var(--chart-3)",
      values: last(props.daily.map((day) => day.rounds)),
    },
    {
      key: "tokens",
      label: "Tokens",
      hint: "in + out",
      value: hasTasks ? formatTokens(insights.tokens) : "—",
      sub: "per task",
      color: "var(--chart-2)",
      values: last(props.daily.map((day) => day.tokens)),
    },
    {
      key: "spend",
      label: "Spend",
      hint: "USD",
      value: hasTasks ? money(insights.cost) : "—",
      sub: "per task",
      color: "var(--chart-1)",
      values: last(props.daily.map((day) => day.cost)),
    },
  ];
});
</script>
