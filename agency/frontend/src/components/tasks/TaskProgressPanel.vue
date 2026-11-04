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
.flex.flex-col.gap-3(v-if="hasPlan")
  .flex.items-center.justify-between.gap-2
    .flex.items-center.gap-2
      span.text-xs.font-semibold Plan
      span.animate-pulse.rounded-full.bg-blue-500(
        v-if="active"
        class="size-1.5"
        title="A step is in progress"
      )
    span.text-muted-foreground.text-xs.tabular-nums
      | {{ count }} done · {{ percent }}%
  .bg-muted.w-full.overflow-hidden.rounded-full(
    class="h-1.5"
    role="progressbar"
    :aria-valuemin="0"
    :aria-valuemax="total"
    :aria-valuenow="done"
    :aria-label="`Plan: ${done} of ${total} steps done`"
  )
    .h-full.rounded-full(
      :class="active ? 'bg-blue-500' : 'bg-emerald-500'"
      :style="{ width: `${percent}%` }"
    )
  .flex.flex-col(class="gap-0.5")
    p.text-xs(v-if="summary") {{ summary }}
    span.text-muted-foreground.text-xs
      | Updated {{ formatRelativeTimestamp(updatedAt) }}
  ul.flex.flex-col(class="gap-0.5")
    li.flex.items-start.gap-2.rounded-md.px-1.py-1(
      v-for="(item, index) in items"
      :key="index"
    )
      template(v-if="item.status === DONE")
        span.flex.shrink-0.items-center.justify-center.rounded-full.border.text-emerald-600(
          class="size-3.5 border-emerald-500/40 bg-emerald-500/15 dark:text-emerald-400"
        )
          CheckIcon(class="size-2.5")
      Loader2Icon.shrink-0.animate-spin.text-blue-500(
        v-else-if="item.status === IN_PROGRESS"
        class="size-3.5"
      )
      span.shrink-0.rounded-full.border(v-else class="size-3.5")
      span.text-xs(
        :class="item.status === DONE ? 'text-muted-foreground line-through' : item.status === IN_PROGRESS ? 'font-medium' : 'text-muted-foreground'"
      ) {{ item.text }}
      span.ml-auto.font-semibold.text-blue-500(
        v-if="item.status === IN_PROGRESS"
        class="text-[10px]"
      ) now
  span.text-muted-foreground.flex.items-center.gap-1.text-xs
    LockIcon.size-3
    | Read-only · the agent may revise this plan when input or CI changes
.flex.flex-col.gap-2(v-else)
  span.text-xs.font-semibold Plan
  .rounded-lg.border.p-3.text-center(
    :class="missing ? 'border-amber-500/40 bg-amber-500/10' : 'border-dashed'"
  )
    p.text-xs.font-medium(
      :class="missing ? 'text-amber-600 dark:text-amber-400' : ''"
    ) {{ missing ? "No plan reported" : "No plan reported yet" }}
    p.text-muted-foreground.text-xs(v-if="missing") The agent must publish a plan for this task.
    p.text-muted-foreground.text-xs(v-else-if="hint") Small tasks do not require a plan; the agent may still publish one with reportTaskProgress.
    p.text-muted-foreground.text-xs(v-else)
      | The agent publishes its plan with reportTaskProgress.
    Button.mt-2(
      v-if="missing && canRemind"
      type="button"
      variant="outline"
      size="sm"
      @click="emit('remind')"
    ) Remind agent to plan
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { formatRelativeTimestamp } from "@/lib/format";
import {
  taskPlanFlag,
  taskProgressCount,
  taskProgressHasPlan,
  taskProgressInProgress,
  taskProgressPercent,
} from "@/lib/task";
import {
  TaskProgressItemStatus,
  TaskSize,
  TaskState,
  type TaskProgress,
} from "@cherrypi/sdk";
import { CheckIcon, Loader2Icon, LockIcon } from "@lucide/vue";
import { computed } from "vue";

const props = defineProps<{
  progress?: TaskProgress;
  state?: TaskState;
  size?: TaskSize;
  canRemind?: boolean;
}>();

const emit = defineEmits<{ (e: "remind"): void }>();

const DONE = TaskProgressItemStatus.DONE;
const IN_PROGRESS = TaskProgressItemStatus.IN_PROGRESS;

const summary = computed(() => props.progress?.summary ?? "");
const updatedAt = computed(() => props.progress?.updatedAt);
const items = computed(() => props.progress?.items ?? []);
const done = computed(() => props.progress?.done ?? 0);
const total = computed(() => props.progress?.total ?? 0);
const count = computed(() => taskProgressCount(props.progress));
const hasPlan = computed(() => taskProgressHasPlan(props.progress));
const percent = computed(() => taskProgressPercent(props.progress));
const flag = computed(() =>
  taskPlanFlag(props.progress, props.size, props.state),
);
const missing = computed(() => flag.value === "missing");
const hint = computed(() => flag.value === "hint");
const active = computed(
  () =>
    props.state === TaskState.RUNNING && taskProgressInProgress(props.progress),
);
</script>
