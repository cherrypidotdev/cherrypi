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
    CardTitle {{ $t("agentView.tabs.health") }}
    CardDescription {{ subtitle }}
    CardAction
      RouterLink.text-muted-foreground.text-xs.font-medium(
        class="hover:text-foreground"
        :to="{ name: 'agent', params: { username: currentAgentUsername }, query: { tab: 'health' } }"
      ) {{ $t("health.overview.view") }}
  CardContent
    p.text-destructive.text-sm(v-if="error") {{ error }}
    p.text-muted-foreground.text-sm(v-else-if="!current")
      | {{ $t("health.overview.empty") }}
    .grid.grid-cols-2.gap-4(v-else class="sm:grid-cols-4")
      .flex.flex-col.gap-1
        .flex.items-baseline.justify-between.gap-2
          span.text-muted-foreground.text-2xs.font-semibold {{ $t("health.kpis.cpu") }}
          span.text-lg.font-semibold.tabular-nums(
            :class="healthToneText(cpuTone)"
          ) {{ current.cpuPercent.toFixed(0) }}%
        .h-6
          HealthSparkline(:color="HEALTH_COLORS.cpu" :values="spark.cpu")
        span.text-muted-foreground.text-2xs.tabular-nums {{ $t("health.kpis.loadValue", { value: current.load1.toFixed(2) }) }}
      .flex.flex-col.gap-1
        .flex.items-baseline.justify-between.gap-2
          span.text-muted-foreground.text-2xs.font-semibold {{ $t("health.kpis.memory") }}
          span.text-lg.font-semibold.tabular-nums(
            :class="healthToneText(memoryTone)"
          ) {{ (current.memory?.usedPercent ?? 0).toFixed(0) }}%
        .h-6
          HealthSparkline(:color="HEALTH_COLORS.memory" :values="spark.memory")
        span.text-muted-foreground.text-2xs {{ $t("health.kpis.used", { used: formatBytes(current.memory?.usedBytes ?? 0) }) }}
      .flex.flex-col.gap-1
        .flex.items-baseline.justify-between.gap-2
          span.text-muted-foreground.text-2xs.font-semibold {{ $t("health.kpis.disk") }}
          span.text-lg.font-semibold.tabular-nums(
            :class="healthToneText(diskTone)"
          ) {{ disk ? `${disk.usedPercent.toFixed(0)}%` : "—" }}
        .h-6
          HealthSparkline(:color="HEALTH_COLORS.disk" :values="spark.disk")
        span.text-muted-foreground.text-2xs {{ disk?.label ?? "no target" }}
      .flex.flex-col.gap-1
        .flex.items-baseline.justify-between.gap-2
          span.text-muted-foreground.text-2xs.font-semibold {{ current.gpus.length ? "GPU" : "Uptime" }}
          span.text-lg.font-semibold.tabular-nums(
            :class="current.gpus.length ? healthToneText(gpuTone) : ''"
          ) {{ fourth }}
        .h-6
          HealthSparkline(
            :color="current.gpus.length ? HEALTH_COLORS.gpu : HEALTH_COLORS.load"
            :values="current.gpus.length ? spark.gpu : spark.load"
          )
        span.text-muted-foreground.text-2xs {{ fourthSub }}
</template>

<script setup lang="ts">
import HealthSparkline from "@/components/health/HealthSparkline.vue";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useAgentHealth } from "@/composables/useAgentHealth";
import { formatBytes } from "@/lib/format";
import {
  busiestGpu,
  formatUptime,
  fullestDisk,
  HEALTH_COLORS,
  healthTone,
  healthToneText,
  type HealthTone,
} from "@/lib/health";
import { currentAgentUsername } from "@/lib/sdk";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";

const props = defineProps<{ agentId: string }>();

const range = ref<"1h">("1h");
const agentId = computed(() => props.agentId);
const { current, history, error } = useAgentHealth(agentId, range);

const disk = computed(() => fullestDisk(current.value?.disks));
const gpu = computed(() => busiestGpu(current.value));

const cpuTone = computed<HealthTone>(() =>
  healthTone(current.value?.cpuPercent ?? 0, 70, 90),
);
const memoryTone = computed<HealthTone>(() =>
  healthTone(current.value?.memory?.usedPercent ?? 0, 75, 90),
);
const diskTone = computed<HealthTone>(() =>
  healthTone(disk.value?.usedPercent ?? 0, 80, 90),
);
const gpuTone = computed<HealthTone>(() =>
  healthTone(gpu.value?.utilizationPercent ?? 0, 80, 95),
);

const fourth = computed(() => {
  if (current.value?.gpus.length) {
    return `${(gpu.value?.utilizationPercent ?? 0).toFixed(0)}%`;
  }
  return formatUptime(Number(current.value?.uptimeSeconds ?? 0));
});

const fourthSub = computed(() => {
  if (current.value?.gpus.length) {
    return gpu.value?.name ?? "GPU";
  }
  return "host uptime";
});

const subtitle = computed(() => {
  if (!current.value) return t("health.overview.hint");
  return `Updated ${new Date(Number(current.value.sampledAt)).toLocaleTimeString()} · ${current.value.scope === "host" ? "host" : "container"}`;
});

const spark = computed(() => ({
  cpu: history.value.map((sample) => sample.cpuPercent),
  memory: history.value.map((sample) => sample.memory?.usedPercent ?? 0),
  disk: history.value.map(
    (sample) => fullestDisk(sample.disks)?.usedPercent ?? 0,
  ),
  gpu: history.value.map((sample) =>
    Math.max(0, ...sample.gpus.map((item) => item.utilizationPercent)),
  ),
  load: history.value.map((sample) => sample.load1),
}));

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  health:
    overview:
      view: "View health"
      empty: "No health data yet. The agent reports shortly after it starts."
      hint: "Latest host sample from this agent."
ru:
  health:
    overview:
      view: "Открыть здоровье"
      empty: "Данных о здоровье пока нет. Агент сообщит их вскоре после запуска."
      hint: "Последний замер хоста от этого агента."
</i18n>
