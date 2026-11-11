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
    CardTitle {{ $t("health.net.title") }}
    CardDescription {{ $t("health.net.hint") }}
  CardContent
    p.text-muted-foreground.text-sm(v-if="!networks.length")
      | {{ $t("health.net.empty") }}
    .overflow-x-auto(v-else)
      table.w-full.text-sm
        thead.text-muted-foreground.text-2xs
          tr.border-b
            th.pb-2.text-left.font-semibold {{ $t("health.net.interface") }}
            th.pb-2.text-right.font-semibold {{ $t("settings.processLogs.download") }}
            th.pb-2.text-right.font-semibold {{ $t("health.net.upload") }}
            th.pb-2.text-right.font-semibold {{ $t("health.net.total") }}
        tbody
          tr.border-b(
            v-for="network in networks"
            :key="network.interface"
            class="last:border-0"
          )
            td.py-2.font-mono.text-xs {{ network.interface }}
            td.py-2.text-right.tabular-nums {{ formatBytesPerSecond(network.rxBytesPerSecond) }}
            td.py-2.text-right.tabular-nums {{ formatBytesPerSecond(network.txBytesPerSecond) }}
            td.text-muted-foreground.py-2.text-right.tabular-nums
              | {{ formatBytes(network.rxBytes) }} / {{ formatBytes(network.txBytes) }}
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
import { formatBytesPerSecond } from "@/lib/health";
import type { AgentNetworkUsage } from "@cherrypi/sdk";

defineProps<{ networks: AgentNetworkUsage[] }>();
</script>

<i18n lang="yaml">
en:
  health:
    net:
      title: "Network"
      hint: "Throughput since the agent's previous sample."
      empty: "No network interfaces reported."
      interface: "Interface"
      upload: "Upload"
      total: "Total in / out"
ru:
  health:
    net:
      title: "Сеть"
      hint: "Пропускная способность с момента предыдущего замера агента."
      empty: "Сетевые интерфейсы не сообщены."
      interface: "Интерфейс"
      upload: "Отдача"
      total: "Всего вход / выход"
</i18n>
