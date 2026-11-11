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
button.w-full.text-left(
  v-if="hasPlan"
  type="button"
  :title="$t('taskProgress.openFull')"
  @click="$emit('open')"
)
  .mx-auto.flex.max-w-3xl.items-center.gap-2.px-4.py-2
    span.text-muted-foreground.text-xs.font-medium {{ $t("taskProgress.title") }}
    span.text-xs.font-semibold.tabular-nums {{ count }}
    .bg-muted.w-16.overflow-hidden.rounded-full(
      class="h-1.5"
      role="progressbar"
      :aria-valuemin="0"
      :aria-valuemax="total"
      :aria-valuenow="done"
      :aria-label="$t('taskProgress.summary', { done, total })"
    )
      .h-full.rounded-full(
        :class="active ? 'bg-blue-500' : 'bg-emerald-500'"
        :style="{ width: `${percent}%` }"
      )
    span.text-muted-foreground.min-w-0.flex-1.truncate.text-xs
      | {{ current }}
    ChevronRightIcon.text-muted-foreground.size-3.shrink-0
button.w-full.text-left(
  v-else-if="missing"
  type="button"
  :title="$t('tasksTable.noPlanHint')"
  @click="$emit('remind')"
  :disabled="!canRemind"
)
  .mx-auto.flex.max-w-3xl.items-center.gap-2.px-4.py-2
    CircleAlertIcon.size-3.shrink-0.text-amber-500
    span.text-xs.font-medium.text-amber-600(class="dark:text-amber-400") {{ $t("taskProgress.missing") }}
    span.text-muted-foreground.min-w-0.flex-1.truncate.text-xs
      | {{ $t("taskProgress.missingHint") }}
    span.text-xs.font-medium.text-amber-600(
      v-if="canRemind"
      class="dark:text-amber-400"
    ) {{ $t("taskProgress.remindShort") }}
</template>

<script setup lang="ts">
import {
  taskPlanFlag,
  taskProgressCount,
  taskProgressCurrent,
  taskProgressHasPlan,
  taskProgressInProgress,
  taskProgressPercent,
} from "@/lib/task";
import { TaskSize, TaskState, type TaskProgress } from "@cherrypi/sdk";
import { ChevronRightIcon, CircleAlertIcon } from "@lucide/vue";
import { computed } from "vue";

const props = defineProps<{
  progress?: TaskProgress;
  state?: TaskState;
  size?: TaskSize;
  canRemind?: boolean;
}>();

defineEmits<{
  (e: "open"): void;
  (e: "remind"): void;
}>();

const hasPlan = computed(() => taskProgressHasPlan(props.progress));
const percent = computed(() => taskProgressPercent(props.progress));
const count = computed(() => taskProgressCount(props.progress));
const done = computed(() => props.progress?.done ?? 0);
const total = computed(() => props.progress?.total ?? 0);
const flag = computed(() =>
  taskPlanFlag(props.progress, props.size, props.state),
);
const missing = computed(() => flag.value === "missing");
const active = computed(
  () =>
    props.state === TaskState.RUNNING && taskProgressInProgress(props.progress),
);
const current = computed(() => taskProgressCurrent(props.progress));
</script>

<i18n lang="yaml">
en:
  taskProgress:
    missing: "Plan missing"
    missingHint: "The agent must publish a plan for this task"
    remindShort: "Remind"
    openFull: "Open the full plan"
ru:
  taskProgress:
    missing: "План отсутствует"
    missingHint: "Агент должен опубликовать план для этой задачи"
    remindShort: "Напомнить"
    openFull: "Открыть полный план"
</i18n>
