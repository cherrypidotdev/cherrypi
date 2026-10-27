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
UsageKpis(:summary="summary" :daily="daily" :range="RANGE")
</template>

<script setup lang="ts">
import UsageKpis from "@/components/usage/UsageKpis.vue";
import {
  useUsageAnalytics,
  type UsageContext,
} from "@/composables/useUsageAnalytics";
import type { UsageRange } from "@/lib/usage";
import { computed, ref } from "vue";

const props = defineProps<{
  context: UsageContext;
  agentId?: string;
  projectId?: string;
}>();

const RANGE: UsageRange = "30";
const includeArchived = ref(true);

const { summary, daily } = useUsageAnalytics({
  context: computed(() => props.context),
  agentId: computed(() => props.agentId ?? ""),
  projectId: computed(() => props.projectId ?? ""),
  range: computed(() => RANGE),
  includeArchived,
});
</script>
