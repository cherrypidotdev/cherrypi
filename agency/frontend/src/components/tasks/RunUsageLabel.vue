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
span.text-2xs.text-muted-foreground.flex.flex-wrap.items-center.gap-x-1(
  v-if="usage"
  :title="title"
  data-testid="run-usage"
)
  span.text-foreground.max-w-40.truncate.font-medium(:title="modelLabel") {{ modelLabel }}
  span ·
  span {{ tokensLabel }}
  span ·
  span {{ costLabel }}
  span ·
  span {{ durationLabel }}
  template(v-if="rateLabel")
    span ·
    span {{ rateLabel }}
</template>

<script setup lang="ts">
import type { RunUsage } from "@/lib/console";
import {
  formatCost,
  formatCount,
  formatDuration,
  formatDurationPrecise,
  formatRate,
  formatTokens,
} from "@/lib/format";
import { generationRate } from "@/lib/usage";
import { computed } from "vue";

const props = defineProps<{ usage?: RunUsage }>();

const modelLabel = computed(() => {
  const usage = props.usage;
  if (!usage?.model) return "Unknown model";
  return usage.provider ? `${usage.provider}/${usage.model}` : usage.model;
});
const tokens = computed(() => {
  const usage = props.usage;
  return usage ? usage.inputTokens + usage.outputTokens : 0;
});
const tokensLabel = computed(() => `${formatTokens(tokens.value)} tokens`);
const costLabel = computed(() => formatCost(props.usage?.cost ?? 0));
const durationLabel = computed(() =>
  formatDuration(props.usage?.activeMs ?? 0),
);

const rateLabel = computed(() => {
  const usage = props.usage;
  if (!usage) return "";
  const rate = generationRate(
    usage.outputTokens,
    usage.thinkingMs,
    usage.modelWaitMs,
  );
  return rate > 0 ? formatRate(rate) : "";
});
const title = computed(() => {
  const usage = props.usage;
  if (!usage) return "";
  const parts = [
    `${formatCount(usage.inputTokens)} in · ${formatCount(usage.outputTokens)} out`,
    modelLabel.value,
  ];
  parts.push(`thinking ${formatDuration(usage.thinkingMs)}`);
  if (usage.modelWaitMs > 0) {
    parts.push(`waiting ${formatDuration(usage.modelWaitMs)}`);
  }
  if (usage.ttftMs > 0) {
    parts.push(`TTFT ${formatDurationPrecise(usage.ttftMs)}`);
  }
  if (usage.at !== undefined) {
    parts.push(`settled ${new Date(usage.at).toLocaleString()}`);
  }
  return parts.join(" · ");
});
</script>
