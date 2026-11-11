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
.mx-auto.flex.w-full.max-w-6xl.flex-col.gap-5
  .flex.flex-wrap.items-end.justify-between.gap-4
    div
      p.text-2xs.text-muted-foreground.font-bold.tracking-widest.uppercase {{ $t("projectView.usage") }}
      h2.mt-1.text-base.font-semibold {{ $t("usage.dashboard.title3") }}
      p.text-muted-foreground.mt-1.max-w-xl.text-xs {{ subtitle }}
    .flex.flex-wrap.items-end.gap-3
      label.flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold {{ $t("health.panel.range") }}
        .bg-muted.flex.rounded-lg.p-1
          button.text-115.rounded-md.px-2.py-1.font-medium(
            v-for="option in USAGE_RANGE_OPTIONS"
            :key="option.value"
            :class="range === option.value ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
            @click="range = option.value"
          ) {{ option.label }}
      label.flex.flex-col.gap-1(v-if="context !== 'agent'")
        span.text-2xs.text-muted-foreground.font-semibold {{ $t("admin.agents.columnAgent") }}
        Select(v-model="agentScope")
          SelectTrigger.w-44(:title="$t('usage.dashboard.agentTitle')")
            SelectValue(:placeholder="$t('usage.dashboard.allAgents')")
          SelectContent
            SelectItem(:value="ALL") {{ $t("usage.dashboard.allAgents") }}
            SelectItem(
              v-for="option in availableAgents"
              :key="option.id"
              :value="option.id"
            ) {{ option.name }}
      label.flex.flex-col.gap-1(v-if="context !== 'project'")
        span.text-2xs.text-muted-foreground.font-semibold {{ $t("admin.projects.columnProject") }}
        Combobox.w-44(
          v-model="projectScope"
          :placeholder="$t('nav.allProjects')"
          empty-text="No matching project."
          :title="$t('usage.dashboard.projectTitle')"
          :aria-label="$t('usage.dashboard.projectTitle')"
          :options="projectScopeOptions"
        )
      label.flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold {{ $t("domainLog.kind.model") }}
        Select(v-model="modelScope")
          SelectTrigger.w-44(:title="$t('usage.dashboard.modelTitle')")
            SelectValue(:placeholder="$t('usage.dashboard.allModels')")
          SelectContent
            SelectItem(:value="ALL") {{ $t("usage.dashboard.allModels") }}
            SelectItem(
              v-for="option in modelScopeOptions"
              :key="option.value"
              :value="option.value"
            ) {{ option.label }}

  p.text-destructive.text-sm(v-if="error") {{ error }}

  UsageKpis(
    :summary="summary"
    :daily="daily"
    :insight-daily="insightDaily"
    :range="range"
    :context="context"
  )

  UsageInsights(
    v-model:tag-ids="insightTagIds"
    :insights="insights"
    :daily="insightDaily"
    :range="range"
    :tags="insightTags"
  )

  .grid.grid-cols-1.gap-4(class="lg:grid-cols-12")
    .min-w-0(class="lg:col-span-8")
      UsageTrendChart(:points="daily" :range-label="rangeLabel")
    .min-w-0(class="lg:col-span-4")
      UsageRunsChart(
        :points="daily"
        :range-label="rangeLabel"
        :completed="summary.completed"
      )
    .min-w-0(class="lg:col-span-5")
      UsageTagChart(
        :series="tagSeries"
        :completed="summary.completed"
        :range-label="rangeLabel"
        :include-archived="includeArchived"
        @update:include-archived="includeArchived = $event"
      )
    .min-w-0(class="lg:col-span-7")
      UsageBreakdown(
        :rows="breakdown"
        :title="breakdownTitle"
        :subtitle="breakdownSubtitle"
        :unit="breakdownUnit"
      )
    .min-w-0(class="lg:col-span-12")
      UsageModelTable(:rows="models" :subtitle="modelSubtitle")
    .min-w-0(class="lg:col-span-12")
      UsageTasksTable(:context="context" :rows="rows")
</template>

<script setup lang="ts">
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import UsageBreakdown from "@/components/usage/UsageBreakdown.vue";
import UsageInsights from "@/components/usage/UsageInsights.vue";
import UsageKpis from "@/components/usage/UsageKpis.vue";
import UsageModelTable from "@/components/usage/UsageModelTable.vue";
import UsageRunsChart from "@/components/usage/UsageRunsChart.vue";
import UsageTagChart from "@/components/usage/UsageTagChart.vue";
import UsageTasksTable from "@/components/usage/UsageTasksTable.vue";
import UsageTrendChart from "@/components/usage/UsageTrendChart.vue";
import {
  useUsageAnalytics,
  type UsageContext,
} from "@/composables/useUsageAnalytics";
import {
  USAGE_RANGE_OPTIONS,
  usageRangeKey,
  type UsageRange,
} from "@/lib/usage";
import { UNKNOWN_USAGE_MODEL } from "@cherrypi/sdk";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  context: UsageContext;
  agentId?: string;
  projectId?: string;
}>();

const ALL = "__all__";
const agentScope = ref(ALL);
const projectScope = ref(ALL);
const modelScope = ref(ALL);
const range = ref<UsageRange>("30");
const includeArchived = ref(true);
const insightTagIds = ref<string[]>([]);

const effectiveAgentId = computed(() =>
  props.context === "agent"
    ? (props.agentId ?? "")
    : agentScope.value === ALL
      ? ""
      : agentScope.value,
);
const effectiveProjectId = computed(() =>
  props.context === "project"
    ? (props.projectId ?? "")
    : projectScope.value === ALL
      ? ""
      : projectScope.value,
);

const effectiveModel = computed(() =>
  modelScope.value === ALL ? "" : modelScope.value,
);

watch([effectiveAgentId, effectiveProjectId], () => {
  modelScope.value = ALL;
});

const {
  summary,
  daily,
  rows,
  tagSeries,
  insights,
  insightDaily,
  insightTags,
  breakdown,
  models,
  availableModels,
  availableAgents,
  availableProjects,
  error,
} = useUsageAnalytics({
  context: computed(() => props.context),
  agentId: effectiveAgentId,
  projectId: effectiveProjectId,
  range,
  includeArchived,
  insightTagIds,
  model: effectiveModel,
});

const rangeLabel = computed(() =>
  t(usageRangeKey(range.value), { days: range.value }),
);
const projectScopeOptions = computed<ComboboxOption[]>(() => [
  { value: ALL, label: t("nav.allProjects") },
  ...availableProjects.value.map((option) => ({
    value: option.id,
    label: option.name,
  })),
]);

const modelScopeOptions = computed(() => {
  const seen = new Set<string>();
  const options: { value: string; label: string }[] = [];
  for (const model of availableModels.value) {
    const value = model || UNKNOWN_USAGE_MODEL;
    if (seen.has(value)) continue;
    seen.add(value);
    options.push({ value, label: model || "Unknown" });
  }

  if (modelScope.value !== ALL && !seen.has(modelScope.value)) {
    options.push({
      value: modelScope.value,
      label:
        modelScope.value === UNKNOWN_USAGE_MODEL ? "Unknown" : modelScope.value,
    });
  }
  return options;
});
const subtitle = computed(() => {
  if (props.context === "project") {
    return t("usage.dashboard.projectHint");
  }
  if (props.context === "agent") {
    return t("usage.dashboard.agentHint");
  }
  return t("usage.dashboard.agencyHint");
});

const breakdownTitle = computed(() =>
  props.context === "project"
    ? t("usage.dashboard.byAgent")
    : t("usage.dashboard.byProject"),
);
const breakdownSubtitle = computed(() =>
  props.context === "project"
    ? t("usage.dashboard.perTaskByAgent")
    : t("usage.dashboard.perTaskByProject"),
);
const breakdownUnit = computed(() =>
  props.context === "project" ? "agents" : "projects",
);
const modelSubtitle = computed(() => {
  if (props.context === "project") {
    return t("usage.dashboard.modelsProject");
  }
  if (props.context === "agent") {
    return t("usage.dashboard.modelsAgent");
  }
  return t("usage.dashboard.modelsAgency");
});

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  usage:
    dashboard:
      allAgents: "All agents"
      allModels: "All models"
      agentTitle: "Filter by agent"
      projectTitle: "Filter by project"
      modelTitle: "Filter by model"
      modelsAgency: "Cost and tokens per model across the agency. The model filter narrows the cards above, not this table."
      modelsAgent: "Cost and tokens per model this agent used. The model filter narrows the cards above, not this table."
      modelsProject: "Cost and tokens per model used in this project. The model filter narrows the cards above, not this table."
      perTaskByProject: "Average per completed task, by project. Follows the insights tag filter."
      perTaskByAgent: "Average per completed task, by agent. Follows the insights tag filter."
      byProject: "By project"
      byAgent: "By agent"
      agencyHint: "Every agent and project in the agency. Filter by scope, model or time; the charts reuse the project components."
      agentHint: "Everything this agent worked on. Filter by project, model or time; the charts reuse the project components."
      projectHint: "All agents working in this project. The range and model filter apply to every chart and breakdown below."
      title3: "Cost, tokens & task executions"
ru:
  usage:
    dashboard:
      allAgents: "Все агенты"
      allModels: "Все модели"
      agentTitle: "Фильтр по агенту"
      projectTitle: "Фильтр по проекту"
      modelTitle: "Фильтр по модели"
      modelsAgency: "Стоимость и токены по моделям всего агентства. Фильтр по модели сужает карточки выше, но не эту таблицу."
      modelsAgent: "Стоимость и токены по моделям этого агента. Фильтр по модели сужает карточки выше, но не эту таблицу."
      modelsProject: "Стоимость и токены по моделям в этом проекте. Фильтр по модели сужает карточки выше, но не эту таблицу."
      perTaskByProject: "Средние значения на завершённую задачу по проектам. Учитывает фильтр тегов из аналитики."
      perTaskByAgent: "Средние значения на завершённую задачу по агентам. Учитывает фильтр тегов из аналитики."
      byProject: "По проектам"
      byAgent: "По агентам"
      agencyHint: "Все агенты и проекты агентства. Фильтруйте по области, модели или времени; графики используют компоненты проекта."
      agentHint: "Всё, над чем работал этот агент. Фильтруйте по проекту, модели или времени; графики используют компоненты проекта."
      projectHint: "Все агенты, работающие в этом проекте. Диапазон и фильтр по модели применяются ко всем графикам и разбивкам ниже."
      title3: "Стоимость, токены и выполнения задач"
</i18n>
