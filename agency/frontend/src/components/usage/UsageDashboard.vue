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
        Select(v-model="projectScope")
          SelectTrigger.w-44(title="Filter by project")
            SelectValue(placeholder="All projects")
          SelectContent
            SelectItem(:value="ALL") All projects
            SelectItem(
              v-for="option in availableProjects"
              :key="option.id"
              :value="option.id"
            ) {{ option.name }}

  p.text-destructive.text-sm(v-if="error") {{ error }}

  UsageKpis(:summary="summary" :daily="daily" :range="range")

  .grid.grid-cols-1.gap-4(class="lg:grid-cols-12")
    .min-w-0(class="lg:col-span-8")
      UsageTrendChart(:points="daily" :range-label="rangeLabel")
    .min-w-0(class="lg:col-span-4")
      UsageRunsChart(
        :points="daily"
        :range-label="rangeLabel"
        :completed="summary.completed"
      )
    .min-w-0(class="lg:col-span-6")
      UsageTagChart(
        :series="tagSeries"
        :completed="summary.completed"
        :range-label="rangeLabel"
        :include-archived="includeArchived"
        @update:include-archived="includeArchived = $event"
      )
    .min-w-0(class="lg:col-span-6")
      UsageBreakdown(
        :rows="breakdown"
        :title="breakdownTitle"
        :subtitle="breakdownSubtitle"
        :unit="breakdownUnit"
      )
    .min-w-0(class="lg:col-span-12")
      UsageTasksTable(:context="context" :rows="rows")
</template>

<script setup lang="ts">
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import UsageBreakdown from "@/components/usage/UsageBreakdown.vue";
import UsageKpis from "@/components/usage/UsageKpis.vue";
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
import { computed, ref } from "vue";

const props = defineProps<{
  context: UsageContext;
  agentId?: string;
  projectId?: string;
}>();

const ALL = "__all__";
const agentScope = ref(ALL);
const projectScope = ref(ALL);
const range = ref<UsageRange>("30");
const includeArchived = ref(true);

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

const {
  summary,
  daily,
  rows,
  tagSeries,
  breakdown,
  availableAgents,
  availableProjects,
  error,
} = useUsageAnalytics({
  context: computed(() => props.context),
  agentId: effectiveAgentId,
  projectId: effectiveProjectId,
  range,
  includeArchived,
});

const rangeLabel = computed(() => usageRangeLabel(range.value));
const subtitle = computed(() => {
  if (props.context === "project") {
    return "All agents working in this project. The range applies to every chart and breakdown below.";
  }
  if (props.context === "agent") {
    return "Everything this agent worked on. Filter by project or time; the charts reuse the project components.";
  }
  return "Every agent and project in the agency. Filter by either scope or time; the charts reuse the project components.";
});

const breakdownTitle = computed(() =>
  props.context === "project" ? "By agent" : "By project",
);
const breakdownSubtitle = computed(() => {
  if (props.context === "project")
    return "Cost and response rounds in this range.";
  if (props.context === "agent") return "Where this agent spent its usage.";
  return "Where the agency spent its usage in this range.";
});
const breakdownUnit = computed(() =>
  props.context === "project" ? "agents" : "projects",
);
</script>
