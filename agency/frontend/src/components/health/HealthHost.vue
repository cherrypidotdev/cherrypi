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
    CardTitle Host
    CardDescription {{ sample ? (sample.scope === "host" ? "Host-mounted /proc and /sys." : "Container view of /proc and /sys.") : "No sample yet." }}
  CardContent.flex.flex-col.gap-4
    dl.grid.grid-cols-1.gap-x-4.gap-y-2.text-sm(class="sm:grid-cols-2")
      .flex.min-w-0.flex-col
        dt.text-muted-foreground.text-2xs.font-semibold Hostname
        dd.truncate.font-mono.text-xs {{ sample?.hostname || "—" }}
      .flex.min-w-0.flex-col
        dt.text-muted-foreground.text-2xs.font-semibold Platform
        dd.truncate.text-xs {{ platform }}
      .flex.min-w-0.flex-col
        dt.text-muted-foreground.text-2xs.font-semibold Kernel
        dd.truncate.font-mono.text-xs {{ kernel }}
      .flex.min-w-0.flex-col
        dt.text-muted-foreground.text-2xs.font-semibold Uptime
        dd.text-xs {{ formatUptime(Number(sample?.uptimeSeconds ?? 0)) }}
      .flex.min-w-0.flex-col
        dt.text-muted-foreground.text-2xs.font-semibold Booted
        dd.text-xs {{ booted }}
      .flex.min-w-0.flex-col
        dt.text-muted-foreground.text-2xs.font-semibold Processes
        dd.text-xs.tabular-nums {{ formatCount(sample?.processCount ?? 0) }}
      .flex.min-w-0.flex-col
        dt.text-muted-foreground.text-2xs.font-semibold Agent RSS
        dd.text-xs.tabular-nums {{ formatBytes(sample?.agentRssBytes ?? 0) }}
      .flex.min-w-0.flex-col
        dt.text-muted-foreground.text-2xs.font-semibold Agent CPU
        dd.text-xs.tabular-nums {{ (sample?.agentCpuPercent ?? 0).toFixed(1) }}%
      .flex.min-w-0.flex-col
        dt.text-muted-foreground.text-2xs.font-semibold Go heap
        dd.text-xs.tabular-nums {{ formatBytes(sample?.goHeapBytes ?? 0) }}

    .flex.flex-col.gap-2(v-if="sample?.cpuCores.length")
      span.text-muted-foreground.text-2xs.font-semibold CPU cores
      .flex.flex-wrap.gap-1
        .bg-muted.flex.h-12.w-4.items-end.overflow-hidden.rounded-sm(
          v-for="(core, index) in sample.cpuCores"
          :key="index"
          :title="`Core ${index}: ${core.toFixed(0)}%`"
        )
          .w-full.transition-all(
            :class="healthToneBar(healthTone(core, 70, 90))"
            :style="{ height: `${Math.min(100, Math.max(2, core))}%` }"
          )
</template>

<script setup lang="ts">
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatBytes, formatCount } from "@/lib/format";
import { formatUptime, healthTone, healthToneBar } from "@/lib/health";
import type { AgentHealthSample } from "@cherrypi/sdk";
import { computed } from "vue";

const props = defineProps<{ sample?: AgentHealthSample }>();

const platform = computed(() => {
  const parts = [props.sample?.platform, props.sample?.os].filter(Boolean);
  return parts.length ? parts.join(" · ") : "—";
});

const kernel = computed(() => {
  const parts = [props.sample?.kernelVersion, props.sample?.kernelArch].filter(
    Boolean,
  );
  return parts.length ? parts.join(" · ") : "—";
});

const booted = computed(() => {
  const seconds = Number(props.sample?.bootTime ?? 0);
  if (!seconds) return "—";
  return new Date(seconds * 1000).toLocaleString();
});
</script>
