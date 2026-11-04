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
      p.text-2xs.text-muted-foreground.font-bold.tracking-widest.uppercase Health
      h2.mt-1.text-base.font-semibold CPU, memory, disk &amp; devices
      p.text-muted-foreground.mt-1.max-w-xl.text-xs
        | Host samples pushed by the agent. The latest sample updates live; the
        | charts read the persisted history.
    .flex.flex-wrap.items-end.gap-3
      label.flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold Time range
        .bg-muted.flex.rounded-lg.p-1
          button.text-115.rounded-md.px-2.py-1.font-medium(
            v-for="option in HEALTH_RANGE_OPTIONS"
            :key="option.value"
            :class="range === option.value ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
            @click="range = option.value"
          ) {{ option.label }}
      .flex.flex-col.gap-1.text-right(v-if="current")
        span.text-2xs.text-muted-foreground.font-semibold Latest sample
        span.text-115.font-medium.tabular-nums {{ sampleAgeLabel(current.sampledAt) }}
        Badge.text-2xs(variant="outline" class="mt-0.5") {{ current.scope === "host" ? "host" : "container" }}

  p.text-destructive.text-sm(v-if="error") {{ error }}

  .flex.flex-col.items-center.justify-center.rounded-xl.border.border-dashed.py-16.text-center(
    v-if="!loading && !current"
  )
    span.text-sm.font-medium No health data yet
    p.text-muted-foreground.mt-1.max-w-md.text-xs
      | The agent reports its first sample shortly after it starts. A binary
      | older than the agency does not report health, so update it if this
      | persists.

  template(v-else)
    HealthKpis(:sample="current" :history="history")

    .grid.grid-cols-1.gap-4(class="lg:grid-cols-12")
      .min-w-0(class="lg:col-span-8")
        HealthTrendChart(:samples="history" :range="range")
      .min-w-0(class="lg:col-span-4")
        HealthDisks(:disks="current?.disks ?? []")
      .min-w-0(v-if="current?.gpus.length" class="lg:col-span-12")
        HealthGpus(:gpus="current.gpus")
      .min-w-0(class="lg:col-span-8")
        HealthNetworks(:networks="current?.networks ?? []")
      .min-w-0(class="lg:col-span-4")
        HealthHost(:sample="current")
</template>

<script setup lang="ts">
import HealthDisks from "@/components/health/HealthDisks.vue";
import HealthGpus from "@/components/health/HealthGpus.vue";
import HealthHost from "@/components/health/HealthHost.vue";
import HealthKpis from "@/components/health/HealthKpis.vue";
import HealthNetworks from "@/components/health/HealthNetworks.vue";
import HealthTrendChart from "@/components/health/HealthTrendChart.vue";
import { Badge } from "@/components/ui/badge";
import { useAgentHealth } from "@/composables/useAgentHealth";
import {
  HEALTH_RANGE_OPTIONS,
  sampleAgeLabel,
  type HealthRange,
} from "@/lib/health";
import { computed, ref } from "vue";

const props = defineProps<{ agentId: string }>();

const range = ref<HealthRange>("24h");
const agentId = computed(() => props.agentId);
const { current, history, loading, error } = useAgentHealth(agentId, range);
</script>
