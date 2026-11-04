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
    CardTitle Filesystems
    CardDescription Measured inside the agent's visible filesystem.
  CardContent.flex.flex-col.gap-4
    p.text-muted-foreground.text-sm(v-if="!disks.length")
      | No filesystem targets configured.
    .flex.flex-col(v-for="disk in disks" :key="disk.path" class="gap-1.5")
      .flex.items-baseline.justify-between.gap-3
        .flex.min-w-0.flex-col
          span.truncate.text-sm.font-medium {{ disk.label }}
          span.text-muted-foreground.text-2xs.truncate.font-mono {{ disk.path }}
        .flex.items-baseline.gap-2.tabular-nums
          span.text-sm.font-semibold(
            :class="healthToneText(healthTone(disk.usedPercent, 80, 90))"
          ) {{ disk.usedPercent.toFixed(0) }}%
          span.text-muted-foreground.text-2xs
            | {{ formatBytes(disk.usedBytes) }} / {{ formatBytes(disk.totalBytes) }}
      .bg-muted.w-full.overflow-hidden.rounded-full(class="h-1.5")
        .h-full.rounded-full.transition-all(
          :class="healthToneBar(healthTone(disk.usedPercent, 80, 90))"
          :style="{ width: `${Math.min(100, Math.max(0, disk.usedPercent))}%` }"
        )
      span.text-muted-foreground.text-2xs {{ formatBytes(disk.freeBytes) }} free
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
import type { AgentDiskUsage } from "@cherrypi/sdk";

defineProps<{ disks: AgentDiskUsage[] }>();
</script>
