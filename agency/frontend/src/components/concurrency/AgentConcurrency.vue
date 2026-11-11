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
      h2.text-lg.font-semibold {{ $t("agentConcurrency.title") }}
      p.text-muted-foreground.max-w-2xl.text-xs
        | {{ $t("agentConcurrency.description") }}
    Button(v-if="board" as-child variant="outline" size="sm")
      RouterLink(:to="settingsTo")
        SettingsIcon(data-icon="inline-start")
        span {{ $t("agentConcurrency.manageLimits") }}

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
            CardTitle {{ $t("agentConcurrency.concurrentSlots") }}
            CardDescription
              | {{ $t("agentConcurrency.slotsHint") }}
          span.bg-muted.text-3xs.text-muted-foreground.rounded-full.px-2(class="py-0.5") {{ board.agent.username }}
      CardContent.flex.flex-col.gap-4
        .flex.flex-wrap.items-end.justify-between.gap-4
          .flex.items-end.gap-2
            .text-3xl.leading-none.font-semibold.tabular-nums {{ board.overall.used }}
            .flex.flex-col(class="gap-0.5 pb-0.5")
              span.text-sm.font-medium {{ $t("agentConcurrency.ofSlotsHeld", { cap: board.overall.cap }) }}
              span.text-2xs.text-muted-foreground {{ $t("agentConcurrency.freeCount", { free: board.free }) }}
          .flex.items-end.gap-2(v-if="board.waiting.length")
            .text-2xl.leading-none.font-semibold.text-amber-600.tabular-nums(
              class="dark:text-amber-400"
            ) {{ board.waiting.length }}
            span.text-2xs.text-muted-foreground {{ $t("agentConcurrency.waiting") }}
        ConcurrencyTrack(
          :label="$t('agentConcurrency.agentSlots')"
          size="lg"
          :used="board.overall.used"
          :cap="board.overall.cap"
          :reserved="board.overall.reserved"
          :waiting="board.waiting.length"
          :show-count="false"
        )
        .flex.flex-wrap.items-center.gap-x-4.gap-y-1(
          v-if="Object.keys(board.reservedByClass).length"
        )
          span.text-2xs.text-muted-foreground {{ $t("agentConcurrency.reservedOnAgent") }}
          span.text-2xs.inline-flex.items-center.gap-1(
            v-for="(slots, name) in board.reservedByClass"
            :key="name"
          )
            span.size-2.rounded-full.border(class="border-amber-500/60 bg-amber-500/40")
            | {{ "#" + name }} {{ slots }}
        .flex.flex-wrap.items-center.justify-between.gap-3
          .flex.flex-wrap.items-center.gap-x-4.gap-y-1
            span.text-2xs.inline-flex.items-center.gap-1(
              v-for="item in states"
              :key="item.state"
              :class="taskStateTextClass(item.state)"
            )
              component.size-3(:is="taskStateIcon(item.state)")
              | {{ $t(taskStateKey(item.state)) }}
              b.font-mono.font-medium {{ item.count }}
          span.text-2xs.text-muted-foreground
            | {{ $t("agentConcurrency.heldFree", { used: board.overall.used, free: board.free }) }}
        .flex.items-start.gap-2.rounded-md.border.px-3.py-2.text-xs.text-amber-700(
          v-if="board.agent.status !== AgentStatus.ONLINE && board.held.length"
          class="border-amber-500/30 bg-amber-500/10 dark:text-amber-300"
        )
          TriangleAlertIcon.size-4.shrink-0
          span
            | {{ $t("agentConcurrency.offlineHint") }}

    .grid.grid-cols-1.items-start.gap-6(
      class="xl:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)]"
    )
      .contents
        Card
          CardHeader
            CardTitle {{ $t("agentConcurrency.capacityByDimension") }}
            CardDescription {{ $t("agentConcurrency.capacityHint") }}
          CardContent.flex.flex-col.px-4.pb-4
            section.flex.flex-col
              .flex.items-baseline.justify-between.gap-2.border-b.pb-1
                span.text-2xs.font-semibold.tracking-wide.uppercase {{ $t("agentConcurrency.overall") }}
                span.text-3xs.text-muted-foreground {{ $t("agentConcurrency.agentCap") }}
              ConcurrencyCapacityRow(:row="board.overall")
            section.flex.flex-col
              .flex.items-baseline.justify-between.gap-2.border-b.pb-1
                span.text-2xs.font-semibold.tracking-wide.uppercase {{ $t("nav.projects") }}
                span.text-3xs.text-muted-foreground {{ $t("agentConcurrency.overrideElseDefault") }}
              p.text-muted-foreground.py-2.text-xs(
                v-if="!board.projectRows.length"
              ) {{ $t("agentConcurrency.noProject") }}
              ConcurrencyCapacityRow(
                v-for="row in board.projectRows"
                :key="row.key"
                :row="row"
              )
            section.flex.flex-col
              .flex.items-baseline.justify-between.gap-2.border-b.pb-1
                span.text-2xs.font-semibold.tracking-wide.uppercase {{ $t("domainLog.kind.tags") }}
                span.text-3xs.text-muted-foreground {{ $t("agentConcurrency.globalByTag") }}
              p.text-muted-foreground.py-2.text-xs(
                v-if="!board.tagRows.length"
              ) {{ $t("agentConcurrency.noGlobalTagCaps") }}
              ConcurrencyCapacityRow(
                v-for="row in board.tagRows"
                :key="row.key"
                :row="row"
              )
              .flex.flex-wrap.items-center.gap-x-3.gap-y-1(
                v-if="board.uncappedTags.length"
              )
                span.text-3xs.text-muted-foreground.font-medium {{ $t("agentConcurrency.noGlobalCap") }}
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
                    | {{ $t("agentConcurrency.tagHeldWaiting", { used: tag.used, waiting: tag.waiting }) }}
            section.flex.flex-col
              .flex.items-baseline.justify-between.gap-2.border-b.pb-1
                span.text-2xs.font-semibold.tracking-wide.uppercase {{ $t("agentConcurrency.projectByTag") }}
                span.text-3xs.text-muted-foreground {{ $t("agentConcurrency.explicitOverridesOnly") }}
              p.text-muted-foreground.py-2.text-xs(
                v-if="!board.projectTagRows.length"
              ) {{ $t("agentConcurrency.noProjectTagOverrides") }}
              ConcurrencyCapacityRow(
                v-for="row in board.projectTagRows"
                :key="row.key"
                :row="row"
              )

      .contents
        ConcurrencyHoldingList(
          :title="$t('agentConcurrency.holdingSlots')"
          :empty="$t('agentConcurrency.noHoldingSlots')"
          :tasks="board.held"
          :projects="board.projects"
          :description="t('agentConcurrency.heldCount', { count: board.held.length })"
        )
        ConcurrencyWaitingList(
          :title="$t('agentConcurrency.waitingTitle')"
          :description="$t('agentConcurrency.waitingDescription')"
          :empty="$t('agentConcurrency.noPinnedWaiting')"
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
  taskStateKey,
  taskStateTextClass,
} from "@/lib/task";
import { AgentStatus, TaskState } from "@cherrypi/sdk";
import { SettingsIcon, TriangleAlertIcon } from "@lucide/vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";

const { t } = useI18n();

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

<i18n lang="yaml">
en:
  agentConcurrency:
    title: "Concurrency"
    description: "A task holds a slot while it is running, waiting for input, settled or failed. Queued tasks start only when every applicable cap has room."
    manageLimits: "Manage limits"
    concurrentSlots: "Concurrent slots"
    slotsHint: "Settled and failed tasks keep their slot until they are closed out; archived tasks never hold one."
    waiting: "waiting"
    reservedOnAgent: "Reserved on this agent"
    offlineHint: "This agent is offline. Its held slots stay frozen and queued tasks cannot start until the tunnel reconnects."
    capacityByDimension: "Capacity by dimension"
    capacityHint: "Every applicable cap must have room for the next task."
    overall: "Overall"
    agentCap: "agent cap"
    overrideElseDefault: "override, else the agent default"
    noProject: "This agent has not joined a project."
    globalByTag: "global by tag name"
    noGlobalTagCaps: "No global tag caps configured."
    noGlobalCap: "No global cap"
    projectByTag: "Project × tag"
    explicitOverridesOnly: "explicit overrides only"
    noProjectTagOverrides: "No project × tag overrides configured."
    agentSlots: "Agent slots"
    holdingSlots: "Holding slots"
    noHoldingSlots: "No tasks hold a slot."
    waitingTitle: "Waiting for this agent"
    waitingDescription: "Queued tasks pinned to this agent, with every cap that is blocking them."
    noPinnedWaiting: "No pinned tasks are waiting."
    heldCount: "{count} tasks keep this agent's slots occupied."
    ofSlotsHeld: "of {cap} slots held"
    freeCount: "{free} free"
    heldFree: "{used} held · {free} free"
    tagHeldWaiting: "{used} held · {waiting} waiting"
ru:
  agentConcurrency:
    title: "Параллельность"
    description: "Задача занимает слот, пока выполняется, ждёт ввода, завершена или завершилась ошибкой. Задачи из очереди запускаются, только когда во всех применимых лимитах есть место."
    manageLimits: "Управлять лимитами"
    concurrentSlots: "Одновременные слоты"
    slotsHint: "Завершённые задачи и задачи с ошибкой сохраняют слот, пока их не закроют; архивные задачи слот не занимают."
    waiting: "в очереди"
    reservedOnAgent: "Зарезервировано на этом агенте"
    offlineHint: "Агент не в сети. Занятые слоты заморожены, а задачи из очереди не запустятся, пока туннель не восстановится."
    capacityByDimension: "Ёмкость по измерениям"
    capacityHint: "Для следующей задачи место должно быть во всех применимых лимитах."
    overall: "Общий"
    agentCap: "лимит агента"
    overrideElseDefault: "переопределение, иначе значение агента"
    noProject: "Агент не входит ни в один проект."
    globalByTag: "глобально по имени тега"
    noGlobalTagCaps: "Глобальные лимиты тегов не настроены."
    noGlobalCap: "Глобального лимита нет"
    projectByTag: "Проект × тег"
    explicitOverridesOnly: "только явные переопределения"
    noProjectTagOverrides: "Переопределения проект × тег не настроены."
    agentSlots: "Слоты агента"
    holdingSlots: "Занятые слоты"
    noHoldingSlots: "Ни одна задача не занимает слот."
    waitingTitle: "Ожидают этого агента"
    waitingDescription: "Задачи из очереди, закреплённые за этим агентом, со всеми блокирующими их лимитами."
    noPinnedWaiting: "Закреплённых задач в очереди нет."
    heldCount: "Задач занимают слоты агента: {count}."
    ofSlotsHeld: "из {cap} слотов занято"
    freeCount: "свободно {free}"
    heldFree: "занято {used} · свободно {free}"
    tagHeldWaiting: "занято {used} · в очереди {waiting}"
</i18n>
