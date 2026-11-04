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
    CardTitle GPUs
    CardDescription NVIDIA utilization, memory, temperature and power.
  CardContent.grid.grid-cols-1.gap-4(class="md:grid-cols-2")
    .ring-border.flex.flex-col.gap-3.rounded-lg.p-3.ring-1(
      v-for="gpu in gpus"
      :key="gpu.index"
    )
      .flex.items-start.justify-between.gap-3
        .flex.min-w-0.flex-col
          span.truncate.text-sm.font-medium {{ gpu.name }}
          span.text-muted-foreground.text-2xs.font-mono GPU {{ gpu.index }}
        .flex.items-baseline.gap-2.tabular-nums
          span.text-lg.font-semibold(
            :class="healthToneText(healthTone(gpu.utilizationPercent, 80, 95))"
          ) {{ gpu.utilizationPercent.toFixed(0) }}%
          span.text-muted-foreground.text-2xs util
      .flex.flex-col.gap-1
        .text-2xs.flex.justify-between
          span.text-muted-foreground VRAM
          span.tabular-nums
            | {{ formatBytes(gpu.memoryUsedBytes) }} / {{ formatBytes(gpu.memoryTotalBytes) }}
        .bg-muted.w-full.overflow-hidden.rounded-full(class="h-1.5")
          .h-full.rounded-full(
            :class="healthToneBar(healthTone(memoryPercent(gpu), 80, 95))"
            :style="{ width: `${Math.min(100, Math.max(0, memoryPercent(gpu)))}%` }"
          )
      .text-2xs.text-muted-foreground.flex.flex-wrap.items-center.gap-3
        span Temperature
          span.text-foreground.ml-1.font-medium.tabular-nums {{ gpu.temperatureCelsius.toFixed(0) }}°C
        span Power
          span.text-foreground.ml-1.font-medium.tabular-nums {{ gpu.powerWatts.toFixed(0) }} W
</template>

<script setup lang="ts">
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatBytes } from "@/lib/format";
import { healthTone, healthToneBar, healthToneText } from "@/lib/health";
import type { AgentGpuUsage } from "@cherrypi/sdk";

defineProps<{ gpus: AgentGpuUsage[] }>();

function memoryPercent(gpu: AgentGpuUsage): number {
  if (!gpu.memoryTotalBytes) return 0;
  return (Number(gpu.memoryUsedBytes) / Number(gpu.memoryTotalBytes)) * 100;
}
</script>
