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
        CardTitle {{ $t("usage.insights.title") }}
        CardDescription
          | {{ $t("usage.insights.hint") }}
      .flex.items-center.gap-2
        span.text-11.text-muted-foreground {{ tasksLabel }}
        DropdownMenu
          DropdownMenuTrigger(as-child)
            Button(
              variant="outline"
              size="sm"
              :title="tags.length ? $t('usage.insights.filterByTag') : $t('usage.insights.noTags')"
              :disabled="!tags.length"
            )
              TagIcon(data-icon="inline-start")
              span {{ tagLabel }}
          DropdownMenuContent.max-h-72.w-56.overflow-y-auto(align="end")
            DropdownMenuLabel {{ $t("domainLog.kind.tags") }}
            DropdownMenuSeparator
            p.text-muted-foreground.px-2.py-1.text-xs(v-if="!tags.length")
              | {{ $t("usage.insights.noTags") }}
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
              DropdownMenuItem(@select="clearTags") {{ $t("usage.insights.clearFilter") }}
  CardContent
    .grid.grid-cols-2.gap-3(class="sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7")
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
      template(v-if="selectedTagIds.length") {{ $t("usage.insights.filtered") }}
      template(v-else) {{ $t("usage.insights.footnote") }}
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
import { formatDuration, formatRate, formatTokens } from "@/lib/format";
import {
  usageRangeKey,
  type UsageInsightDay,
  type UsageInsights,
  type UsageRange,
} from "@/lib/usage";
import { TagIcon } from "@lucide/vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  insights: UsageInsights;
  daily: UsageInsightDay[];
  range: UsageRange;
  tags: UsageInsightTag[];
}>();

const selectedTagIds = defineModel<string[]>("tagIds", {
  default: () => [],
});

const rangeLabel = computed(() =>
  t(usageRangeKey(props.range), { days: props.range }),
);

const tasksLabel = computed(() => {
  const count = props.insights.tasks;
  if (!count) return t("usage.insights.noCompleted");
  return t("usage.insights.completedTasks", { count });
});

const tagLabel = computed(() => {
  const count = selectedTagIds.value.length;
  if (!count) return t("usage.insights.allTags");
  if (count === 1) {
    return (
      props.tags.find((tag) => tag.id === selectedTagIds.value[0])?.name ??
      t("usage.insights.oneTag")
    );
  }
  return t("usage.insights.tagsCount", { count });
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
      ? t("usage.kpis.outsideBacklog")
      : t("usage.kpis.timed", {
          timed: insights.ttcSamples,
          completed: insights.tasks,
        });

  return [
    {
      key: "ttc",
      label: t("usage.breakdown.ttc"),
      hint: t("usage.insights.avgPerTask"),
      value: showsTtc ? formatDuration(insights.ttcMs) : "—",
      sub: ttcSub,
      color: "var(--chart-1)",
      values: last(props.daily.map((day) => day.ttcMs)),
    },
    {
      key: "active",
      label: t("usage.insights.ttcActive"),
      hint: t("usage.kpis.agentWork"),
      value: hasTasks ? formatDuration(insights.activeMs) : "—",
      sub: t("usage.insights.agentWorkPerTask"),
      color: "var(--chart-2)",
      values: last(props.daily.map((day) => day.activeMs)),
    },
    {
      key: "thinking",
      label: t("usage.insights.thinking"),
      hint: t("usage.kpis.modelWork"),
      value: hasTasks ? formatDuration(insights.thinkingMs) : "—",
      sub: insights.modelWaitMs
        ? t("usage.insights.waiting", {
            value: formatDuration(insights.modelWaitMs),
          })
        : t("usage.insights.perTask"),
      color: "var(--chart-3)",
      values: last(props.daily.map((day) => day.thinkingMs)),
    },
    {
      key: "rate",
      label: t("usage.insights.speed"),
      hint: t("usage.insights.outputTokPerSec"),
      value:
        hasTasks && insights.tokensPerSecond
          ? formatRate(insights.tokensPerSecond)
          : "—",
      sub: t("usage.insights.generationOnly"),
      color: "var(--chart-5)",
      values: last(props.daily.map((day) => day.tokensPerSecond)),
    },
    {
      key: "rounds",
      label: t("usage.insights.rounds"),
      hint: t("usage.kpis.agentTurns"),
      value: hasTasks ? insights.rounds.toFixed(1) : "—",
      sub: t("usage.insights.perTask"),
      color: "var(--chart-3)",
      values: last(props.daily.map((day) => day.rounds)),
    },
    {
      key: "tokens",
      label: t("account.tabs.tokens"),
      hint: t("usage.insights.inOut"),
      value: hasTasks ? formatTokens(insights.tokens) : "—",
      sub: t("usage.insights.perTask"),
      color: "var(--chart-2)",
      values: last(props.daily.map((day) => day.tokens)),
    },
    {
      key: "spend",
      label: t("usage.insights.spend"),
      hint: t("usage.insights.usd"),
      value: hasTasks ? money(insights.cost) : "—",
      sub: t("usage.insights.perTask"),
      color: "var(--chart-1)",
      values: last(props.daily.map((day) => day.cost)),
    },
  ];
});
</script>

<i18n lang="yaml">
en:
  usage:
    insights:
      title: "Averages per task"
      hint: "Completed tasks in range. TTC is the time outside backlog since the first queue, active is the agent's work time, and thinking is the model's own work inside it."
      noTags: "No tags on completed tasks"
      clearFilter: "Clear tag filter"
      filtered: "Tag-filtered · any selected tag"
      footnote: "Outside backlog · active excludes idle waits · thinking is model work"
      inOut: "in + out"
      speed: "Speed"
      thinking: "Thinking"
      ttcActive: "TTC active"
      allTags: "All tags"
      noCompleted: "No completed tasks"
      rounds: "Rounds"
      spend: "Spend"
      completedTasks: "{count} completed task | {count} completed tasks"
      oneTag: "1 tag"
      tagsCount: "{count} tag | {count} tags"
      avgPerTask: "avg / task"
      agentWorkPerTask: "agent work / task"
      perTask: "per task"
      waiting: "{value} waiting"
      outputTokPerSec: "output tok/s"
      generationOnly: "generation only"
      usd: "USD"
      filterByTag: "Filter the insights by tag"
ru:
  usage:
    insights:
      title: "Средние значения на задачу"
      hint: "Завершённые задачи в диапазоне. TTC — время вне бэклога с первой постановки в очередь, активное — рабочее время агента, размышление — работа модели внутри него."
      noTags: "У завершённых задач нет тегов"
      clearFilter: "Сбросить фильтр тегов"
      filtered: "Фильтр по тегам · любой выбранный тег"
      footnote: "Вне бэклога · активное время без простоев · размышление — работа модели"
      inOut: "вход + выход"
      speed: "Скорость"
      thinking: "Размышление"
      ttcActive: "TTC активное"
      allTags: "Все теги"
      noCompleted: "Нет завершённых задач"
      rounds: "Раунды"
      spend: "Расход"
      completedTasks: "{count} завершённая задача | {count} завершённые задачи | {count} завершённых задач"
      oneTag: "1 тег"
      tagsCount: "{count} тег | {count} тега | {count} тегов"
      avgPerTask: "в среднем / задача"
      agentWorkPerTask: "работа агента / задача"
      perTask: "на задачу"
      waiting: "{value} в ожидании"
      outputTokPerSec: "выходных ток./с"
      generationOnly: "только генерация"
      usd: "USD"
      filterByTag: "Фильтр аналитики по тегу"
</i18n>
