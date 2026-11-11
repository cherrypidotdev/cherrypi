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
        | {{ $t("projectConcurrency.description") }}
    span.text-muted-foreground.text-3xs
      | {{ $t("projectConcurrency.limitsHint") }}

  p.text-destructive.text-xs(v-if="error") {{ error }}
  .flex.flex-col.gap-3(v-if="loading && !board")
    Skeleton.h-20.w-full.rounded-xl
    Skeleton.h-80.rounded-xl

  template(v-else-if="board")
    .bg-card.ring-border.flex.flex-wrap.items-center.gap-x-8.gap-y-3.rounded-xl.p-4.ring-1
      .flex.min-w-32.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold.tracking-wide.uppercase {{ $t("projectConcurrency.heldHere") }}
        span.font-mono.text-lg.font-semibold.tabular-nums
          | {{ board.held }}
          span.text-muted-foreground /{{ board.capSum }}
        ConcurrencyTrack(
          :label="$t('projectConcurrency.projectSlots')"
          size="sm"
          :used="board.held"
          :cap="board.capSum"
          :reserved="reservedTotal"
          :waiting="board.waiting.length"
          :show-count="false"
        )
      .flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold.tracking-wide.uppercase {{ $t("projectConcurrency.waiting") }}
        span.text-lg.font-semibold.tabular-nums(
          :class="board.waiting.length ? 'text-amber-600 dark:text-amber-400' : ''"
        ) {{ board.waiting.length }}
      .flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold.tracking-wide.uppercase {{ $t("admin.teams.columnMembers") }}
        span.text-lg.font-semibold.tabular-nums {{ board.memberCount }}
        span.text-3xs.text-muted-foreground {{ $t("projectConcurrency.onlineCount", { count: board.onlineCount }) }}
      .flex.flex-col.gap-1
        span.text-2xs.text-muted-foreground.font-semibold.tracking-wide.uppercase {{ $t("projectConcurrency.capsFull") }}
        span.text-lg.font-semibold.tabular-nums(
          :class="board.fullMemberCount ? 'text-amber-600 dark:text-amber-400' : ''"
        ) {{ board.fullMemberCount }}
        span.text-3xs.text-muted-foreground {{ $t("projectConcurrency.ofAgents", { count: board.memberCount }) }}

    .grid.grid-cols-1.items-start.gap-6(
      class="xl:grid-cols-[minmax(0,1.5fr)_minmax(300px,0.9fr)]"
    )
      .flex.min-w-0.flex-col.gap-6
        Card
          CardHeader
            CardTitle {{ $t("projectConcurrency.agentsOnProject") }}
            CardDescription {{ $t("projectConcurrency.agentsOnProjectHint") }}
          CardContent.flex.flex-col.px-4.pb-4
            p.text-muted-foreground.py-2.text-xs(v-if="!board.members.length") {{ $t("projectConcurrency.noAgents") }}
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
                    :aria-label="$t('projectConcurrency.manageLimitsFor', { agent: member.agent.username })"
                  )
                    SettingsIcon.size-4
              .grid.grid-cols-1.gap-3(class="sm:grid-cols-2")
                .flex.flex-col.gap-1
                  span.text-2xs.text-muted-foreground.font-medium {{ $t("projectConcurrency.inThisProject") }}
                  ConcurrencyTrack(
                    :label="$t('projectConcurrency.agentInProject', { agent: member.agent.username })"
                    :used="member.projectRow.used"
                    :cap="member.projectRow.cap"
                    :waiting="member.waiting.length"
                  )
                .flex.flex-col.gap-1
                  span.text-2xs.text-muted-foreground.font-medium {{ $t("agentConcurrency.overall") }}
                  ConcurrencyTrack(
                    :label="$t('projectConcurrency.agentOverall', { agent: member.agent.username })"
                    :used="member.board.overall.used"
                    :cap="member.board.overall.cap"
                    :waiting="member.board.waiting.length"
                  )

        Card
          CardHeader
            CardTitle {{ $t("projectConcurrency.reservedSlots") }}
            CardDescription
              | {{ $t("projectConcurrency.reservedHint") }}
          CardContent.flex.flex-col.gap-3.px-4.pb-4
            p.text-muted-foreground.text-xs(v-if="!reservationRows.length")
              | {{ $t("projectConcurrency.nothingReserved") }}
            .flex.flex-col.gap-2(v-else)
              .flex.flex-wrap.items-center.gap-2.border-b.pb-2(
                v-for="row in reservationRows"
                :key="row.tagName"
              )
                span.text-3xs.inline-flex.items-center.gap-1.rounded-full.border(
                  class="px-1.5 py-0.5"
                  :style="tagStyle(row.color)"
                )
                  span.size-1.rounded-full.bg-current
                  span {{ "#" + row.tagName }}
                span.text-2xs.text-muted-foreground.font-mono.tabular-nums
                  | {{ $t("projectConcurrency.heldCount", { count: row.held }) }}
                .ml-auto.flex.items-center.gap-2
                  Input.w-20(
                    v-if="editable"
                    v-model.number="draft[row.tagName]"
                    type="number"
                    min="0"
                    :aria-label="$t('projectConcurrency.reservedForTag', { tag: row.tagName })"
                  )
                  span.text-2xs(v-else) {{ $t("projectConcurrency.reservedCount", { count: row.reserved }) }}
                  Button(
                    v-if="editable"
                    variant="ghost"
                    size="icon-sm"
                    :title="$t('common.remove')"
                    @click="removeReservation(row.tagName)"
                  )
                    Trash2Icon.size-4
            .flex.flex-wrap.items-center.gap-2(v-if="editable")
              Select(v-model="newTag")
                SelectTrigger.w-40
                  SelectValue(:placeholder="$t('projectConcurrency.addTag')")
                SelectContent
                  SelectItem(
                    v-for="tag in reservableTags"
                    :key="tag.id"
                    :value="tag.name"
                  ) {{ "#" + tag.name }}
              Input.w-20(
                v-model.number="newReserved"
                type="number"
                min="1"
                :aria-label="$t('projectConcurrency.reservedSlots')"
              )
              Button(
                size="sm"
                variant="outline"
                @click="addReservation"
                :disabled="!newTag || newReserved < 1 || saving"
              ) {{ $t("common.add") }}
              Button.ml-auto(
                size="sm"
                @click="saveReservations"
                :disabled="!dirty || saving"
              ) {{ saving ? "Saving…" : "Save" }}
            p.text-destructive.text-3xs(v-if="saveError") {{ saveError }}
            .flex.flex-col.gap-1(v-if="warnings.length")
              p.text-3xs.dark_text-amber-400.text-amber-600(
                v-for="warning in warnings"
                :key="warning"
              ) {{ warning }}
        Card
          CardHeader
            CardTitle {{ $t("projectConcurrency.tagCaps") }}
            CardDescription {{ $t("projectConcurrency.tagCapsHint") }}
          CardContent.px-0.pb-2
            .overflow-x-auto
              table.w-full.text-left.text-xs
                thead
                  tr.border-b
                    th.text-muted-foreground.px-4.py-2.font-medium {{ $t("admin.agents.columnAgent") }}
                    th.text-muted-foreground.px-2.py-2.font-medium {{ $t("tags.columnTag") }}
                    th.text-muted-foreground.min-w-40.px-2.py-2.font-medium {{ $t("projectConcurrency.globalCap") }}
                    th.text-muted-foreground.min-w-40.px-2.py-2.font-medium {{ $t("agentConcurrency.projectByTag") }}
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
                        :label="$t('projectConcurrency.tagGlobalCap', { agent: row.agent.username, tag: row.tag.name })"
                        size="sm"
                        :used="row.global.used"
                        :cap="row.global.cap"
                        :waiting="row.global.waiting"
                      )
                      span.text-muted-foreground(v-else) {{ $t("projectConcurrency.noCap") }}
                    td.px-2.py-2
                      ConcurrencyTrack(
                        v-if="row.project"
                        :label="$t('projectConcurrency.tagInProject', { agent: row.agent.username, tag: row.tag.name })"
                        size="sm"
                        :used="row.project.used"
                        :cap="row.project.cap"
                        :waiting="row.project.waiting"
                      )
                      span.text-muted-foreground(v-else) —
                tbody(v-if="!board.tagRows.length")
                  tr
                    td.text-muted-foreground.px-4.py-3(colspan="4") {{ $t("projectConcurrency.noTagCaps") }}

      .flex.min-w-0.flex-col.gap-6
        ConcurrencyWaitingList(
          show-agent
          :title="$t('projectConcurrency.waitingTitle')"
          :description="$t('projectConcurrency.waitingDescription')"
          :empty="$t('projectConcurrency.nothingWaiting')"
          :waiting="board.waiting"
          :projects="[board.project]"
        )
</template>

<script setup lang="ts">
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
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useProjectConcurrency } from "@/composables/useConcurrency";
import { useTags } from "@/composables/useTags";
import { currentAccount } from "@/lib/account";
import { agentStatusClass, agentStatusLabel } from "@/lib/agent";
import { reservedSlots } from "@/lib/concurrency";
import { agency } from "@/lib/sdk";
import { tagStyle } from "@/lib/task";
import type { Agent, Project } from "@cherrypi/sdk";
import { SettingsIcon, Trash2Icon } from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";

const { t } = useI18n();

const props = defineProps<{ project: Project; canManage: boolean }>();

const { board, loading, error, refresh } = useProjectConcurrency(
  () => props.project,
);
const { all: tags } = useTags(() => props.project?.id);

const isAdmin = computed(() => currentAccount.value?.is_admin === true);
const editable = computed(() => props.canManage || isAdmin.value);

const saved = ref<Record<string, number>>({});

const draft = ref<Record<string, number>>({});
let loadedProjectId = "";
const newTag = ref("");
const newReserved = ref(1);
const saving = ref(false);
const saveError = ref("");

function differs(
  a: Record<string, number>,
  b: Record<string, number>,
): boolean {
  const names = new Set([...Object.keys(a), ...Object.keys(b)]);
  for (const name of names) {
    if ((a[name] ?? 0) !== (b[name] ?? 0)) return true;
  }
  return false;
}

const dirty = computed(() => differs(draft.value, saved.value));

watch(
  board,
  (value) => {
    const next: Record<string, number> = {};
    for (const reservation of value?.reservations ?? []) {
      next[reservation.tagName.toLowerCase()] = reservation.reserved;
    }
    const projectId = value?.project.id ?? "";

    if (projectId === loadedProjectId && dirty.value) {
      saved.value = next;
      return;
    }
    loadedProjectId = projectId;
    saved.value = next;
    draft.value = { ...next };
  },
  { immediate: true },
);

const reservedTotal = computed(() => reservedSlots(draft.value));

const capacity = computed(() => board.value?.capSum ?? 0);

const smallestAgentCap = computed(() => {
  const caps = (board.value?.members ?? []).map(
    (member) => member.board.overall.cap,
  );
  return caps.length ? Math.min(...caps) : 0;
});

const warnings = computed(() => {
  const out: string[] = [];
  const total = reservedTotal.value;
  if (total <= 0) return out;
  if (capacity.value > 0 && total >= capacity.value) {
    out.push(
      t("projectConcurrency.warningAllReserved", {
        total,
        capacity: capacity.value,
      }),
    );
  }
  for (const row of reservationRows.value) {
    if (smallestAgentCap.value > 0 && row.reserved > smallestAgentCap.value) {
      out.push(
        t("projectConcurrency.warningOverReserved", {
          tag: row.tagName,
          reserved: row.reserved,
          cap: smallestAgentCap.value,
        }),
      );
    }
  }
  return out;
});

const tagByName = computed(
  () =>
    new Map(tags.value.map((tag) => [tag.name.toLowerCase(), tag] as const)),
);

const reservationRows = computed(() =>
  Object.entries(draft.value)
    .filter(([, reserved]) => reserved > 0)
    .map(([tagName, reserved]) => ({
      tagName,
      reserved,
      color: tagByName.value.get(tagName)?.color ?? "",
      held: board.value?.floor.byClass[tagName] ?? 0,
    }))
    .sort((a, b) => a.tagName.localeCompare(b.tagName)),
);

const reservableTags = computed(() =>
  tags.value.filter((tag) => !(tag.name.toLowerCase() in draft.value)),
);

function addReservation() {
  const name = newTag.value.trim().toLowerCase();
  if (!name) return;
  draft.value = { ...draft.value, [name]: Math.max(1, newReserved.value || 1) };
  newTag.value = "";
  newReserved.value = 1;
}

function removeReservation(name: string) {
  const next = { ...draft.value };
  delete next[name];
  draft.value = next;
}

async function saveReservations() {
  saving.value = true;
  saveError.value = "";
  try {
    await agency.updateProjectTagReservations(
      props.project.id,
      Object.entries(draft.value)
        .filter(([, reserved]) => reserved > 0)
        .map(([tagName, reserved]) => ({ tagName, reserved })),
    );
    await refresh();
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

function avatar(agent: Agent): string {
  return toSvg(agent.id, 24);
}
</script>

<i18n lang="yaml">
en:
  projectConcurrency:
    waiting: "Waiting"
    capsFull: "Caps full"
    description: "How this project's member agents use their slots, and which project, tag or project × tag cap is currently full."
    limitsHint: "Per-agent limits are edited in Agent Settings → Projects."
    heldHere: "Held here"
    agentsOnProject: "Agents on this project"
    agentsOnProjectHint: "Project caps are per agent, so the same project can have room on one agent and none on another."
    noAgents: "No agents have joined this project."
    inThisProject: "In this project"
    reservedSlots: "Reserved slots"
    reservedHint: "Slots kept free for one tag class. A task without the tag starts only while the project and its member agents have more than the reserved count to spare; a task with the tag may use them."
    nothingReserved: "Nothing is reserved. Every member agent's slots are available to any task."
    tagCaps: "Tag caps in this project"
    tagCapsHint: "Global tag caps apply in every project; a project × tag cap narrows one tag here."
    globalCap: "Global cap"
    noCap: "no cap"
    noTagCaps: "No tag caps or tagged tasks in this project."
    projectSlots: "Project slots"
    addTag: "Add a tag"
    waitingTitle: "Waiting for this project"
    waitingDescription: "Pinned and unpinned queued tasks in this project, with what is blocking them."
    nothingWaiting: "Nothing is waiting for this project."
    reservedForTag: "Reserved slots for #{tag}"
    manageLimitsFor: "Manage limits for {agent}"
    agentInProject: "{agent} in this project"
    agentOverall: "{agent} overall"
    tagGlobalCap: "{agent} #{tag} global cap"
    tagInProject: "{agent} #{tag} in this project"
    warningAllReserved: "The project keeps {total} of its {capacity} slots reserved, so a task without these tags can never start here."
    warningOverReserved: "#{tag} reserves {reserved}, more than the smallest member agent's {cap} slots: that agent can then run no task without #{tag}, in any project."
    onlineCount: "{count} online"
    ofAgents: "of {count} agents"
    heldCount: "{count} held"
    reservedCount: "{count} reserved"
ru:
  projectConcurrency:
    waiting: "В очереди"
    capsFull: "Лимиты исчерпаны"
    description: "Как агенты-участники проекта используют свои слоты и какой лимит проекта, тега или проект × тег сейчас исчерпан."
    limitsHint: "Лимиты агентов редактируются в настройках агента → «Проекты»."
    heldHere: "Занято здесь"
    agentsOnProject: "Агенты в проекте"
    agentsOnProjectHint: "Лимиты проекта задаются для каждого агента, поэтому в одном агенте место может быть, а в другом — нет."
    noAgents: "Ни один агент не вступил в проект."
    inThisProject: "В этом проекте"
    reservedSlots: "Зарезервированные слоты"
    reservedHint: "Слоты, оставленные свободными для одного класса тегов. Задача без тега запускается, только если у проекта и его агентов остаётся больше зарезервированного; задача с тегом может их использовать."
    nothingReserved: "Ничего не зарезервировано. Слоты каждого агента доступны любой задаче."
    tagCaps: "Лимиты тегов в проекте"
    tagCapsHint: "Глобальные лимиты тегов действуют во всех проектах; лимит проект × тег дополнительно ограничивает тег здесь."
    globalCap: "Глобальный лимит"
    noCap: "без лимита"
    noTagCaps: "В проекте нет лимитов тегов и задач с тегами."
    projectSlots: "Слоты проекта"
    addTag: "Добавить тег"
    waitingTitle: "Ожидают этот проект"
    waitingDescription: "Задачи из очереди этого проекта, закреплённые и незакреплённые, с указанием блокирующих причин."
    nothingWaiting: "Ничего не ожидает этот проект."
    reservedForTag: "Зарезервированные слоты для #{tag}"
    manageLimitsFor: "Управлять лимитами для {agent}"
    agentInProject: "{agent} в этом проекте"
    agentOverall: "{agent} в целом"
    tagGlobalCap: "{agent} #{tag} глобальный лимит"
    tagInProject: "{agent} #{tag} в этом проекте"
    warningAllReserved: "Проект держит зарезервированными {total} из {capacity} слотов, поэтому задача без этих тегов здесь никогда не запустится."
    warningOverReserved: "#{tag} резервирует {reserved} — больше, чем {cap} слотов у самого малого агента-участника: такой агент не сможет выполнить ни одной задачи без #{tag} ни в одном проекте."
    onlineCount: "в сети: {count}"
    ofAgents: "из {count} агентов"
    heldCount: "занято {count}"
    reservedCount: "зарезервировано {count}"
</i18n>
