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
.flex.flex-col.gap-6
  .flex.flex-wrap.items-start.justify-between.gap-3
    .flex.flex-col.gap-1
      h2.text-lg.font-semibold Concurrency
      p.text-muted-foreground.max-w-2xl.text-xs
        | A task holds a slot while it is running, waiting for input, settled or
        | failed. Queued tasks start only when every applicable cap has room.
    Button(v-if="board" as-child variant="outline" size="sm")
      RouterLink(:to="settingsTo")
        SettingsIcon(data-icon="inline-start")
        span Manage limits

  p.text-destructive.text-xs(v-if="error") {{ error }}
  .flex.flex-col.gap-3(v-if="loading && !board")
    Skeleton.h-36.w-full.rounded-xl
    .grid.grid-cols-1.gap-3(class="lg:grid-cols-2")
      Skeleton.h-72.rounded-xl
      Skeleton.h-72.rounded-xl

  template(v-else-if="board")
    Card
      CardHeader
        .flex.flex-wrap.items-start.justify-between.gap-3
          .flex.flex-col.gap-1
            CardTitle Concurrent slots
            CardDescription
              | Settled and failed tasks keep their slot until they are closed
              | out; archived tasks never hold one.
          span.bg-muted.text-3xs.text-muted-foreground.rounded-full.px-2(class="py-0.5") {{ board.agent.username }}
      CardContent.flex.flex-col.gap-4
        .flex.flex-wrap.items-end.justify-between.gap-4
          .flex.items-end.gap-2
            .text-3xl.leading-none.font-semibold.tabular-nums {{ board.overall.used }}
            .flex.flex-col(class="gap-0.5 pb-0.5")
              span.text-sm.font-medium of {{ board.overall.cap }} slots held
              span.text-2xs.text-muted-foreground {{ board.free }} free
          .flex.items-end.gap-2(v-if="board.waiting.length")
            .text-2xl.leading-none.font-semibold.text-amber-600.tabular-nums(
              class="dark:text-amber-400"
            ) {{ board.waiting.length }}
            span.text-2xs.text-muted-foreground waiting
        ConcurrencyTrack(
          label="Agent slots"
          size="lg"
          :used="board.overall.used"
          :cap="board.overall.cap"
          :waiting="board.waiting.length"
          :show-count="false"
        )
        .flex.flex-wrap.items-center.justify-between.gap-3
          .flex.flex-wrap.items-center.gap-x-4.gap-y-1
            span.text-2xs.inline-flex.items-center.gap-1(
              v-for="item in states"
              :key="item.state"
              :class="taskStateTextClass(item.state)"
            )
              component.size-3(:is="taskStateIcon(item.state)")
              | {{ taskStateLabel(item.state) }}
              b.font-mono.font-medium {{ item.count }}
          span.text-2xs.text-muted-foreground
            | {{ board.overall.used }} held · {{ board.free }} free
        .flex.items-start.gap-2.rounded-md.border.px-3.py-2.text-xs.text-amber-700(
          v-if="board.agent.status !== AgentStatus.ONLINE && board.held.length"
          class="border-amber-500/30 bg-amber-500/10 dark:text-amber-300"
        )
          TriangleAlertIcon.size-4.shrink-0
          span
            | This agent is offline. Its held slots stay frozen and queued
            | tasks cannot start until the tunnel reconnects.

    .grid.grid-cols-1.items-start.gap-6(
      class="xl:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)]"
    )
      .contents
        Card
          CardHeader
            CardTitle Capacity by dimension
            CardDescription Every applicable cap must have room for the next task.
          CardContent.flex.flex-col.px-4.pb-4
            section.flex.flex-col
              .flex.items-baseline.justify-between.gap-2.border-b.pb-1
                span.text-2xs.font-semibold.tracking-wide.uppercase Overall
                span.text-3xs.text-muted-foreground agent cap
              ConcurrencyCapacityRow(:row="board.overall")
            section.flex.flex-col
              .flex.items-baseline.justify-between.gap-2.border-b.pb-1
                span.text-2xs.font-semibold.tracking-wide.uppercase Projects
                span.text-3xs.text-muted-foreground override, else the agent default
              p.text-muted-foreground.py-2.text-xs(
                v-if="!board.projectRows.length"
              ) This agent has not joined a project.
              ConcurrencyCapacityRow(
                v-for="row in board.projectRows"
                :key="row.key"
                :row="row"
              )
            section.flex.flex-col
              .flex.items-baseline.justify-between.gap-2.border-b.pb-1
                span.text-2xs.font-semibold.tracking-wide.uppercase Tags
                span.text-3xs.text-muted-foreground global by tag name
              p.text-muted-foreground.py-2.text-xs(
                v-if="!board.tagRows.length"
              ) No global tag caps configured.
              ConcurrencyCapacityRow(
                v-for="row in board.tagRows"
                :key="row.key"
                :row="row"
              )
              .flex.flex-wrap.items-center.gap-x-3.gap-y-1(
                v-if="board.uncappedTags.length"
              )
                span.text-3xs.text-muted-foreground.font-medium No global cap
                span.text-3xs.inline-flex.items-center.gap-1(
                  v-for="tag in board.uncappedTags"
                  :key="tag.tag?.id || tag.name"
                )
                  span.inline-flex.items-center.gap-1.rounded-full.border(
                    v-if="tag.tag"
                    class="px-1.5 py-0.5"
                    :style="tagStyle(tag.tag.color)"
                  )
                    span.size-1.rounded-full.bg-current
                    | {{ tag.tag.name }}
                  span.text-muted-foreground.font-mono
                    | {{ tag.used }} held · {{ tag.waiting }} waiting
            section.flex.flex-col
              .flex.items-baseline.justify-between.gap-2.border-b.pb-1
                span.text-2xs.font-semibold.tracking-wide.uppercase Project × tag
                span.text-3xs.text-muted-foreground explicit overrides only
              p.text-muted-foreground.py-2.text-xs(
                v-if="!board.projectTagRows.length"
              ) No project × tag overrides configured.
              ConcurrencyCapacityRow(
                v-for="row in board.projectTagRows"
                :key="row.key"
                :row="row"
              )

      .contents
        ConcurrencyHoldingList(
          title="Holding slots"
          empty="No tasks hold a slot."
          :tasks="board.held"
          :projects="board.projects"
          :description="`${board.held.length} task${board.held.length === 1 ? '' : 's'} keep this agent's slots occupied.`"
        )
        ConcurrencyWaitingList(
          title="Waiting for this agent"
          description="Queued tasks pinned to this agent, with every cap that is blocking them."
          empty="No pinned tasks are waiting."
          :waiting="board.waiting"
          :projects="board.projects"
        )
</template>

<script setup lang="ts">
import ConcurrencyCapacityRow from "@/components/concurrency/ConcurrencyCapacityRow.vue";
import ConcurrencyHoldingList from "@/components/concurrency/ConcurrencyHoldingList.vue";
import ConcurrencyTrack from "@/components/concurrency/ConcurrencyTrack.vue";
import ConcurrencyWaitingList from "@/components/concurrency/ConcurrencyWaitingList.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useAgentConcurrency } from "@/composables/useConcurrency";
import {
  tagStyle,
  taskStateIcon,
  taskStateLabel,
  taskStateTextClass,
} from "@/lib/task";
import { AgentStatus, TaskState } from "@cherrypi/sdk";
import { SettingsIcon, TriangleAlertIcon } from "@lucide/vue";
import { computed } from "vue";
import { RouterLink } from "vue-router";

const props = defineProps<{ agentId: string }>();

const { board, loading, error } = useAgentConcurrency(() => props.agentId);

const states = computed(() => {
  const value = board.value;
  if (!value) return [];
  return [
    { state: TaskState.RUNNING, count: value.stateCounts.running },
    { state: TaskState.INPUT, count: value.stateCounts.input },
    { state: TaskState.SETTLED, count: value.stateCounts.settled },
    { state: TaskState.FAILED, count: value.stateCounts.failed },
  ];
});

const settingsTo = computed(() => ({
  name: "agent",
  params: { username: board.value?.agent.username ?? "" },
  query: { tab: "settings", section: "tasks" },
}));
</script>
