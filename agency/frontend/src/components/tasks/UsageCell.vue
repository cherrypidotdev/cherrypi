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
.flex.flex-col.text-xs(v-if="usage")
  span
    | {{ $t("usageCell.in", { value: formatTokens(usage.inputTokens) }) }}
    span.text-muted-foreground ·
    | {{ $t("usageCell.out", { value: formatTokens(usage.outputTokens) }) }}
  span.text-muted-foreground
    | {{ formatCost(usage.cost) }}
    template(v-if="usage.live") {{ $t("usageCell.live") }}
span.text-muted-foreground.text-xs(v-else) —
</template>

<script setup lang="ts">
import { type UsageBucket } from "@/composables/useUsage";
import { formatCost, formatTokens } from "@/lib/format";

defineProps<{ usage?: UsageBucket }>();
</script>

<i18n lang="yaml">
en:
  usageCell:
    live: "· live"
    in: "{value} in"
    out: "{value} out"
ru:
  usageCell:
    live: "· идёт"
    in: "{value} вх."
    out: "{value} исх."
</i18n>
