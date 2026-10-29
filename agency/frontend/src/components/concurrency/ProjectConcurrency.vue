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
        | How this project's member agents use their slots, and which project,
        | tag or project × tag cap is currently full.
    span.text-muted-foreground.text-3xs
      | Per-agent limits are edited in Agent Settings → Projects.

  p.text-destructive.text-xs(v-if="error") {{ error }}
  .flex.flex-col.gap-3(v-if="loading && !board")
    Skeleton.h-20.w-full.rounded-xl
    Skeleton.h-80.rounded-xl

  template(v-else-if="board")
    .bg-card.ring-border.flex.flex-wrap.items-center.gap-x-8.gap-y-3.rounded-xl.p-4.ring-1
      .flex.min-w-32.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold.tracking-wide.uppercase Held here
        span.font-mono.text-lg.font-semibold.tabular-nums
          | {{ board.held }}
          span.text-muted-foreground /{{ board.capSum }}
        ConcurrencyTrack(
          label="Project slots"
          size="sm"
          :used="board.held"
          :cap="board.capSum"
          :waiting="board.waiting.length"
          :show-count="false"
        )
      .flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold.tracking-wide.uppercase Waiting
        span.text-lg.font-semibold.tabular-nums(
          :class="board.waiting.length ? 'text-amber-600 dark:text-amber-400' : ''"
        ) {{ board.waiting.length }}
      .flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold.tracking-wide.uppercase Members
        span.text-lg.font-semibold.tabular-nums {{ board.memberCount }}
        span.text-3xs.text-muted-foreground {{ board.onlineCount }} online
      .flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold.tracking-wide.uppercase Caps full
        span.text-lg.font-semibold.tabular-nums(
          :class="board.fullMemberCount ? 'text-amber-600 dark:text-amber-400' : ''"
        ) {{ board.fullMemberCount }}
        span.text-3xs.text-muted-foreground of {{ board.memberCount }} agents

    .grid.grid-cols-1.items-start.gap-6(
      class="xl:grid-cols-[minmax(0,1.5fr)_minmax(300px,0.9fr)]"
    )
      .flex.min-w-0.flex-col.gap-6
        Card
          CardHeader
            CardTitle Agents on this project
            CardDescription Project caps are per agent, so the same project can have room on one agent and none on another.
          CardContent.flex.flex-col.px-4.pb-4
            p.text-muted-foreground.py-2.text-xs(v-if="!board.members.length") No agents have joined this project.
            .flex.flex-col.gap-4.border-b.py-4(
              v-for="member in board.members"
              :key="member.agent.id"
            )
              .flex.flex-wrap.items-center.justify-between.gap-2
                RouterLink.flex.min-w-0.items-center.gap-2(
                  class="hover:text-primary"
                  :to="{ name: 'agent', params: { username: member.agent.username }, query: { tab: 'concurrency' } }"
                )
                  span.size-6.shrink-0(:innerHTML="avatar(member.agent)")
                  .flex.flex-col
                    span.text-sm.font-medium {{ member.agent.displayName || member.agent.username }}
                    span.text-3xs.text-muted-foreground.font-mono @{{ member.agent.username }}
                .flex.items-center.gap-2
                  span.text-3xs.font-medium(
                    :class="agentStatusClass(member.agent.status)"
                  ) {{ agentStatusLabel(member.agent.status) }}
                  RouterLink.text-muted-foreground(
                    class="hover:text-foreground"
                    :to="{ name: 'agent', params: { username: member.agent.username }, query: { tab: 'settings', section: 'projects' } }"
                    :aria-label="`Manage limits for ${member.agent.username}`"
                  )
                    SettingsIcon.size-4
              .grid.grid-cols-1.gap-3(class="sm:grid-cols-2")
                .flex.flex-col.gap-1
                  span.text-2xs.text-muted-foreground.font-medium In this project
                  ConcurrencyTrack(
                    :label="`${member.agent.username} in this project`"
                    :used="member.projectRow.used"
                    :cap="member.projectRow.cap"
                    :waiting="member.waiting.length"
                  )
                .flex.flex-col.gap-1
                  span.text-2xs.text-muted-foreground.font-medium Overall
                  ConcurrencyTrack(
                    :label="`${member.agent.username} overall`"
                    :used="member.board.overall.used"
                    :cap="member.board.overall.cap"
                    :waiting="member.board.waiting.length"
                  )

        Card
          CardHeader
            CardTitle Tag caps in this project
            CardDescription Global tag caps apply in every project; a project × tag cap narrows one tag here.
          CardContent.px-0.pb-2
            .overflow-x-auto
              table.w-full.text-left.text-xs
                thead
                  tr.border-b
                    th.text-muted-foreground.px-4.py-2.font-medium Agent
                    th.text-muted-foreground.px-2.py-2.font-medium Tag
                    th.text-muted-foreground.min-w-40.px-2.py-2.font-medium Global cap
                    th.text-muted-foreground.min-w-40.px-2.py-2.font-medium Project × tag
                tbody
                  tr.border-b(
                    v-for="row in board.tagRows"
                    :key="`${row.agent.id}:${row.tag.id}`"
                  )
                    td.px-4.py-2
                      RouterLink.text-muted-foreground(
                        class="hover:text-foreground"
                        :to="{ name: 'agent', params: { username: row.agent.username }, query: { tab: 'concurrency' } }"
                      ) {{ row.agent.username }}
                    td.px-2.py-2
                      span.text-3xs.inline-flex.items-center.gap-1.rounded-full.border(
                        class="px-1.5 py-0.5"
                        :style="tagStyle(row.tag.color)"
                      )
                        span.size-1.rounded-full.bg-current
                        | {{ row.tag.name }}
                    td.px-2.py-2
                      ConcurrencyTrack(
                        v-if="row.global"
                        :label="`${row.agent.username} #${row.tag.name} global cap`"
                        size="sm"
                        :used="row.global.used"
                        :cap="row.global.cap"
                        :waiting="row.global.waiting"
                      )
                      span.text-muted-foreground(v-else) no cap
                    td.px-2.py-2
                      ConcurrencyTrack(
                        v-if="row.project"
                        :label="`${row.agent.username} #${row.tag.name} in this project`"
                        size="sm"
                        :used="row.project.used"
                        :cap="row.project.cap"
                        :waiting="row.project.waiting"
                      )
                      span.text-muted-foreground(v-else) —
                tbody(v-if="!board.tagRows.length")
                  tr
                    td.text-muted-foreground.px-4.py-3(colspan="4") No tag caps or tagged tasks in this project.

      .flex.min-w-0.flex-col.gap-6
        ConcurrencyWaitingList(
          title="Waiting for this project"
          description="Pinned and unpinned queued tasks in this project, with what is blocking them."
          empty="Nothing is waiting for this project."
          show-agent
          :waiting="board.waiting"
          :projects="[board.project]"
        )
</template>

<script setup lang="ts">
import ConcurrencyTrack from "@/components/concurrency/ConcurrencyTrack.vue";
import ConcurrencyWaitingList from "@/components/concurrency/ConcurrencyWaitingList.vue";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useProjectConcurrency } from "@/composables/useConcurrency";
import { agentStatusClass, agentStatusLabel } from "@/lib/agent";
import { tagStyle } from "@/lib/task";
import type { Agent, Project } from "@cherrypi/sdk";
import { SettingsIcon } from "@lucide/vue";
import { toSvg } from "jdenticon";
import { RouterLink } from "vue-router";

const props = defineProps<{ project: Project }>();

const { board, loading, error } = useProjectConcurrency(() => props.project);

function avatar(agent: Agent): string {
  return toSvg(agent.id, 24);
}
</script>
