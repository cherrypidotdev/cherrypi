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
.flex.w-full.items-center.gap-2.rounded-md.border.border-dashed.px-3.text-xs(
  class="bg-muted/30 py-1.5"
  :title="titleHint"
  data-testid="compaction-line"
)
  span.leading-none(aria-hidden="true") 🏁
  span.font-medium {{ lineText }}
  time.text-muted-foreground.ml-auto.whitespace-nowrap.tabular-nums(
    v-if="at"
    :datetime="isoInstant(at)"
    :title="formatInstant(at)"
  ) {{ formatClock(at) }}
</template>

<script setup lang="ts">
import {
  formatClock,
  formatInstant,
  formatTokens,
  isoInstant,
} from "@/lib/format";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  beforeTokens: number | bigint;

  afterTokens: number | bigint;

  durationMs: number | bigint;

  reason?: string;

  at?: number;
}>();

const { t } = useI18n();

const seconds = computed(() =>
  (Math.max(0, Number(props.durationMs)) / 1000).toFixed(1),
);

// measurement: keep the duration line rather than claiming `0 → 0 tokens`.
const hasMeasurement = computed(
  () => Number(props.beforeTokens) > 0 && Number(props.afterTokens) > 0,
);
const lineText = computed(() =>
  hasMeasurement.value
    ? t("compaction.line", {
        before: formatTokens(props.beforeTokens),
        after: formatTokens(props.afterTokens),
        seconds: seconds.value,
      })
    : t("compaction.lineLegacy", { seconds: seconds.value }),
);
const titleHint = computed(() =>
  [lineText.value, reasonText.value].filter(Boolean).join(" · "),
);
const reasonText = computed(() => {
  switch (props.reason) {
    case "threshold":
      return t("compaction.reason.threshold");
    case "overflow":
      return t("compaction.reason.overflow");
    case "manual":
      return t("compaction.reason.manual");
    default:
      return "";
  }
});
</script>

<i18n lang="yaml">
en:
  compaction:
    line: "Compaction ({before} → {after} tokens, {seconds} sec.)"
    lineLegacy: "Compaction ({seconds} sec.)"
    reason:
      threshold: "context full"
      overflow: "context overflowed"
      manual: "requested"
ru:
  compaction:
    line: "Сжатие ({before} → {after} токенов, {seconds} сек.)"
    lineLegacy: "Сжатие ({seconds} сек.)"
    reason:
      threshold: "контекст заполнен"
      overflow: "контекст переполнен"
      manual: "по запросу"
</i18n>
