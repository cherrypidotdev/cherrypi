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
.grid.grid-cols-1.gap-3(
  class="sm:grid-cols-2 lg:grid-cols-4"
  :aria-label="$t('health.kpis.aria')"
)
  .bg-card.ring-border.relative.flex.flex-col.overflow-hidden.rounded-xl.bg-gradient-to-br.p-3.ring-1(
    class="from-chart-1/5"
  )
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold {{ $t("health.kpis.cpu") }}
      span.text-muted-foreground.text-10 {{ $t("health.kpis.loadValue", { value: loadLabel }) }}
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums(
      :class="healthToneText(cpuTone)"
    ) {{ percent(cpu) }}
    .text-2xs.text-muted-foreground.mt-1.flex.items-center.justify-between.gap-2
      span {{ coresLabel }}
      span(v-if="sample") {{ sample.scope === "host" ? "host" : "container" }}
    .mt-auto.h-7.pt-1
      HealthSparkline(:color="HEALTH_COLORS.cpu" :values="spark.cpu")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold {{ $t("health.kpis.memory") }}
      span.text-muted-foreground.text-10 {{ $t("health.kpis.ofTotal", { total: formatBytes(memoryTotal) }) }}
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums(
      :class="healthToneText(memoryTone)"
    ) {{ percent(memoryPercent) }}
    .text-2xs.text-muted-foreground.mt-1
      | {{ $t("health.kpis.used", { used: formatBytes(memoryUsed) }) }}
    .mt-auto.h-7.pt-1
      HealthSparkline(:color="HEALTH_COLORS.memory" :values="spark.memory")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold {{ $t("health.kpis.disk") }}
      span.text-muted-foreground.text-10 {{ disk?.label ?? "no target" }}
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums(
      :class="healthToneText(diskTone)"
    ) {{ disk ? percent(disk.usedPercent) : "—" }}
    .text-2xs.text-muted-foreground.mt-1(v-if="disk")
      | {{ $t("health.kpis.usedOfTotal", { used: formatBytes(disk.usedBytes), total: formatBytes(disk.totalBytes) }) }}
    .text-2xs.text-muted-foreground.mt-1(v-else) {{ $t("health.kpis.noFilesystem") }}
    .mt-auto.h-7.pt-1
      HealthSparkline(:color="HEALTH_COLORS.disk" :values="spark.disk")

  .bg-card.ring-border.flex.flex-col.rounded-xl.p-3.ring-1
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-2xs.font-semibold {{ $t("health.kpis.load") }}
      span.text-muted-foreground.text-10 1m / 5m / 15m
    .text-23.mt-1.leading-tight.font-semibold.tracking-tight.tabular-nums(
      :class="loadToneClass"
    ) {{ load.toFixed(2) }}
    .text-2xs.text-muted-foreground.mt-1.tabular-nums
      | {{ sample?.load5.toFixed(2) ?? "—" }} · {{ sample?.load15.toFixed(2) ?? "—" }}
    .mt-auto.h-7.pt-1
      HealthSparkline(:color="HEALTH_COLORS.load" :values="spark.load")
</template>

<script setup lang="ts">
import HealthSparkline from "@/components/health/HealthSparkline.vue";
import { formatBytes } from "@/lib/format";
import {
  fullestDisk,
  HEALTH_COLORS,
  healthTone,
  healthToneText,
  type HealthTone,
} from "@/lib/health";
import type { AgentHealthSample } from "@cherrypi/sdk";
import { computed } from "vue";

const props = defineProps<{
  sample?: AgentHealthSample;
  history: AgentHealthSample[];
}>();

function percent(value: number): string {
  return `${value.toFixed(0)}%`;
}

const cpu = computed(() => props.sample?.cpuPercent ?? 0);
const memoryPercent = computed(() => props.sample?.memory?.usedPercent ?? 0);
const memoryUsed = computed(() => props.sample?.memory?.usedBytes ?? 0n);
const memoryTotal = computed(() => props.sample?.memory?.totalBytes ?? 0n);
const disk = computed(() => fullestDisk(props.sample?.disks));

const cpuTone = computed<HealthTone>(() => healthTone(cpu.value, 70, 90));
const memoryTone = computed<HealthTone>(() =>
  healthTone(memoryPercent.value, 75, 90),
);
const diskTone = computed<HealthTone>(() =>
  healthTone(disk.value?.usedPercent ?? 0, 80, 90),
);

const load = computed(() => props.sample?.load1 ?? 0);
const coresLabel = computed(() => {
  const count = props.sample?.cpuCores.length ?? 0;
  return count ? `${count} cores` : "no cores reported";
});
const loadLabel = computed(() =>
  props.sample ? `${props.sample.load1.toFixed(2)}` : "—",
);
const loadToneClass = computed(() => {
  const cores = props.sample?.cpuCores.length ?? 0;
  if (!cores) return "";
  const ratio = load.value / cores;
  return healthToneText(healthTone(ratio * 100, 70, 100));
});

function historyValue(
  selector: (sample: AgentHealthSample) => number,
): number[] {
  return props.history.map(selector);
}

const spark = computed(() => ({
  cpu: historyValue((sample) => sample.cpuPercent),
  memory: historyValue((sample) => sample.memory?.usedPercent ?? 0),
  disk: historyValue((sample) => fullestDisk(sample.disks)?.usedPercent ?? 0),
  load: historyValue((sample) => sample.load1),
}));
</script>

<i18n lang="yaml">
en:
  health:
    kpis:
      cpu: "CPU"
      memory: "Memory"
      disk: "Disk"
      noFilesystem: "no filesystem configured"
      load: "Load"
      aria: "Agent health summary"
      loadValue: "load {value}"
      ofTotal: "of {total}"
      used: "{used} used"
      usedOfTotal: "{used} of {total}"
ru:
  health:
    kpis:
      cpu: "CPU"
      memory: "Память"
      disk: "Диск"
      noFilesystem: "файловая система не задана"
      load: "Нагрузка"
      aria: "Сводка здоровья агента"
      loadValue: "нагрузка {value}"
      ofTotal: "из {total}"
      used: "использовано {used}"
      usedOfTotal: "{used} из {total}"
</i18n>
