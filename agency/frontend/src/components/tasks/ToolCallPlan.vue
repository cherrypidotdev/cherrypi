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
details.rounded-md.border(class="group/plan" data-testid="tool-call-plan")
  summary.flex.cursor-pointer.items-center.gap-2.px-2.py-1.text-xs.select-none
    ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
      class="group-open/plan:rotate-90"
    )
    ListChecksIcon.text-muted-foreground.shrink-0(class="size-3.5")
    span.font-medium {{ $t("toolCallPlan.title") }}
    span.text-muted-foreground.shrink-0(v-if="plan") {{ $t("toolCallPlan.count", { done: plan.done, total: plan.total }) }}
    span.text-muted-foreground.truncate(v-if="plan && plan.summary") · {{ plan.summary }}
    span.text-muted-foreground.shrink-0(v-if="!plan") {{ $t("toolCallPlan.read") }}
    span.text-muted-foreground.shrink-0(v-if="running") {{ $t("toolCall.running") }}
  .flex.flex-col.gap-1.px-2.pb-2(v-if="plan")
    .bg-muted.w-full.overflow-hidden.rounded-full(
      class="h-1.5"
      role="progressbar"
      :aria-valuemin="0"
      :aria-valuemax="plan.total"
      :aria-valuenow="plan.done"
      :aria-label="$t('toolCallPlan.progress', { done: plan.done, total: plan.total })"
    )
      .h-full.rounded-full(
        :class="plan.done === plan.total && plan.total > 0 ? 'bg-emerald-500' : 'bg-blue-500'"
        :style="{ width: `${plan.percent}%` }"
      )
    ul.flex.flex-col(class="gap-0.5")
      li.flex.items-start.gap-2.rounded-md.px-1.py-1(
        v-for="(step, index) in plan.steps"
        :key="index"
      )
        span.flex.shrink-0.items-center.justify-center.rounded-full.border.text-emerald-600(
          v-if="step.status === 'done'"
          class="mt-0.5 size-3.5 border-emerald-500/40 bg-emerald-500/15 dark:text-emerald-400"
        )
          CheckIcon(class="size-2.5")
        Loader2Icon.shrink-0.animate-spin.text-blue-500(
          v-else-if="step.status === 'in_progress'"
          class="mt-0.5 size-3.5"
        )
        span.shrink-0.rounded-full.border(v-else class="mt-0.5 size-3.5")
        span.text-xs(
          :class="step.status === 'done' ? 'text-muted-foreground line-through' : step.status === 'in_progress' ? 'font-medium' : 'text-muted-foreground'"
        ) {{ step.text }}
</template>

<script setup lang="ts">
import type { ToolItem } from "@/lib/console";
import { planCall } from "@/lib/toolPresentation";
import {
  CheckIcon,
  ChevronRightIcon,
  ListChecksIcon,
  Loader2Icon,
} from "@lucide/vue";
import { computed } from "vue";

const props = defineProps<{
  item: ToolItem;

  running: boolean;
}>();

const plan = computed(() => planCall(props.item.args));
</script>

<i18n lang="yaml">
en:
  toolCallPlan:
    title: "Plan"
    count: "{done}/{total}"
    read: "read the current plan"
    progress: "Plan: {done} of {total} steps done"
ru:
  toolCallPlan:
    title: "План"
    count: "{done}/{total}"
    read: "чтение текущего плана"
    progress: "План: выполнено {done} из {total} шагов"
</i18n>
