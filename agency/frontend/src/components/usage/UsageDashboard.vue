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
      p.text-2xs.text-muted-foreground.font-bold.tracking-widest.uppercase Usage
      h2.mt-1.text-base.font-semibold Cost, tokens &amp; task executions
      p.text-muted-foreground.mt-1.max-w-xl.text-xs {{ subtitle }}
    .flex.flex-wrap.items-end.gap-3
      label.flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold Time range
        .bg-muted.flex.rounded-lg.p-1
          button.text-115.rounded-md.px-2.py-1.font-medium(
            v-for="option in USAGE_RANGE_OPTIONS"
            :key="option.value"
            :class="range === option.value ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
            @click="range = option.value"
          ) {{ option.label }}
      label.flex.flex-col.gap-1(v-if="context !== 'agent'")
        span.text-2xs.text-muted-foreground.font-semibold Agent
        Select(v-model="agentScope")
          SelectTrigger.w-44(title="Filter by agent")
            SelectValue(placeholder="All agents")
          SelectContent
            SelectItem(:value="ALL") All agents
            SelectItem(
              v-for="option in availableAgents"
              :key="option.id"
              :value="option.id"
            ) {{ option.name }}
      label.flex.flex-col.gap-1(v-if="context !== 'project'")
        span.text-2xs.text-muted-foreground.font-semibold Project
        Combobox.w-44(
          v-model="projectScope"
          placeholder="All projects"
          empty-text="No matching project."
          title="Filter by project"
          aria-label="Filter by project"
          :options="projectScopeOptions"
        )
      label.flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold Model
        Select(v-model="modelScope")
          SelectTrigger.w-44(title="Filter by model")
            SelectValue(placeholder="All models")
          SelectContent
            SelectItem(:value="ALL") All models
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
  usageRangeLabel,
  type UsageRange,
} from "@/lib/usage";
import { UNKNOWN_USAGE_MODEL } from "@cherrypi/sdk";
import { computed, ref, watch } from "vue";

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

const rangeLabel = computed(() => usageRangeLabel(range.value));
const projectScopeOptions = computed<ComboboxOption[]>(() => [
  { value: ALL, label: "All projects" },
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
    return "All agents working in this project. The range and model filter apply to every chart and breakdown below.";
  }
  if (props.context === "agent") {
    return "Everything this agent worked on. Filter by project, model or time; the charts reuse the project components.";
  }
  return "Every agent and project in the agency. Filter by scope, model or time; the charts reuse the project components.";
});

const breakdownTitle = computed(() =>
  props.context === "project" ? "By agent" : "By project",
);
const breakdownSubtitle = computed(() =>
  props.context === "project"
    ? "Average per completed task, by agent. Follows the insights tag filter."
    : "Average per completed task, by project. Follows the insights tag filter.",
);
const breakdownUnit = computed(() =>
  props.context === "project" ? "agents" : "projects",
);
const modelSubtitle = computed(() => {
  if (props.context === "project") {
    return "Cost and tokens per model used in this project. The model filter narrows the cards above, not this table.";
  }
  if (props.context === "agent") {
    return "Cost and tokens per model this agent used. The model filter narrows the cards above, not this table.";
  }
  return "Cost and tokens per model across the agency. The model filter narrows the cards above, not this table.";
});
</script>
