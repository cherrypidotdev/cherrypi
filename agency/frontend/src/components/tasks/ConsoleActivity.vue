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
.d-chat.d-chat-start.max-w-full.overflow-hidden
  ChatAvatar(:sender="sender" :picture="picture")

  details.bg-card.w-full.overflow-hidden.rounded-lg.border(
    class="group/activity"
    @toggle="emit('toggle', $event)"
    :data-activity-id="group.id"
  )
    summary.flex.cursor-pointer.items-center.gap-2.px-3.py-2.text-xs.select-none
      ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
        class="group-open/activity:rotate-90"
      )
      Spinner.size-3.shrink-0(
        v-if="group.running"
        data-testid="activity-running"
      )
      CheckIcon.size-3.shrink-0.text-emerald-600(
        v-else
        class="dark:text-emerald-400"
        data-testid="activity-done"
      )
      span {{ workedLabel }}
      span.text-muted-foreground(
        v-if="actionCount"
        data-testid="activity-actions"
      ) · {{ actionCount }}
      span.text-muted-foreground(
        v-if="waitingLabel"
        :title="waitingHint"
        data-testid="activity-waiting"
      ) · {{ waitingLabel }}

    .flex.flex-col.gap-2.px-3.pb-3
      template(v-for="child in group.items" :key="child.id")
        ThinkingBlock(
          v-if="child.kind === 'thinking'"
          :text="child.text"
          :started-at="child.startedAt"
          :ended-at="child.endedAt"
          :duration-ms="child.durationMs"
          :estimated="child.estimated === true"
          :running="isThinkingRunning(child)"
        )

        ActivityWait(
          v-else-if="child.kind === 'wait'"
          :kind="child.waitKind"
          :model="child.model"
          :provider="child.provider"
          :duration-ms="child.durationMs"
          :error="child.error"
          :attempt="child.attempt"
          :reason="child.reason"
          :fallback-provider="modelProvider ?? ''"
          :fallback-model="modelName ?? ''"
        )

        ToolCallPlan(
          v-else-if="child.kind === 'tool' && child.name === 'reportTaskProgress'"
          :item="child"
          :running="isInFlight(child)"
        )

        ToolCall(
          v-else
          :item="child"
          :running="isInFlight(child)"
          :interrupted="isInterrupted(child)"
          :duration="childDuration(child)"
          :can-interrupt="canInterrupt && isInFlight(child)"
          :interrupting="interrupting.has(child.callId)"
          @interrupt="emit('interrupt', child)"
        )

      ActivityWait(
        v-if="group.pending"
        live
        :kind="group.pending.kind"
        :since="group.pending.since"
        :model="group.pending.model"
        :provider="group.pending.provider"
        :error="group.pending.error"
        :attempt="group.pending.attempt"
        :max-attempts="group.pending.maxAttempts"
        :reason="group.pending.reason"
        :fallback-provider="modelProvider ?? ''"
        :fallback-model="modelName ?? ''"
      )

      ChangedFiles(:files="changes")
</template>

<script setup lang="ts">
import ActivityWait from "@/components/tasks/ActivityWait.vue";
import ChangedFiles from "@/components/tasks/ChangedFiles.vue";
import ChatAvatar from "@/components/tasks/ChatAvatar.vue";
import ThinkingBlock from "@/components/tasks/ThinkingBlock.vue";
import ToolCall from "@/components/tasks/ToolCall.vue";
import ToolCallPlan from "@/components/tasks/ToolCallPlan.vue";
import { Spinner } from "@/components/ui/spinner";
import {
  groupElapsed,
  isThinkingStreaming,
  isToolInFlight,
  itemElapsed,
  isInterrupted as runInterrupted,
  type ActivityItem,
  type MessageSender as ChatSender,
  type Item,
  type RenderActivity,
  type RunState,
} from "@/lib/console";
import { formatDuration, formatElapsed } from "@/lib/format";
import { groupChangedFiles, type ChangedFile } from "@/lib/toolPresentation";
import type { Attachment } from "@cherrypi/sdk";
import { CheckIcon, ChevronRightIcon } from "@lucide/vue";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    group: RenderActivity;

    run: RunState;

    now?: number;

    sender?: ChatSender;

    picture?: Attachment;

    modelName?: string;
    modelProvider?: string;

    canInterrupt?: boolean;

    interrupting?: ReadonlySet<string>;
  }>(),
  {
    now: undefined,
    sender: undefined,
    picture: undefined,
    modelName: "",
    modelProvider: "",
    canInterrupt: false,
    interrupting: () => new Set<string>(),
  },
);

const emit = defineEmits<{
  toggle: [event: Event];

  interrupt: [item: ActivityItem];
}>();

function elapsedLabel(ms: number | undefined, precise = false): string {
  if (ms === undefined) return "";
  if (precise) return formatElapsed(ms);
  return ms >= 1000 ? formatDuration(ms) : "";
}

const workedLabel = computed(() => {
  let elapsed = groupElapsed(props.group.span, props.now);

  const step = props.group.pending;
  if (step && props.run.live && props.now !== undefined) {
    const from = props.group.span.reduce<number | undefined>(
      (start, item) =>
        item.startedAt !== undefined &&
        (start === undefined || item.startedAt < start)
          ? item.startedAt
          : start,
      undefined,
    );
    elapsed = Math.max(props.now - (from ?? step.since), elapsed ?? 0);
  }
  const verb = props.group.running ? "Working" : "Worked";
  if (elapsed === undefined) return verb;
  return `${verb} for ${elapsed >= 1000 ? formatDuration(elapsed) : "<1s"}`;
});

const actionCount = computed(() => {
  const count = props.group.items.filter((item) => item.kind !== "wait").length;
  if (!count) return "";
  return `${count} ${count === 1 ? "action" : "actions"}`;
});

const waitingLabel = computed(() =>
  props.group.waiting.ms > 0
    ? `${formatElapsed(props.group.waiting.ms)} waiting`
    : "",
);

const waitingHint = computed(() => {
  const { requests, retries, compactions } = props.group.waiting;
  return [
    `${requests} model request${requests === 1 ? "" : "s"}`,
    `${retries} retr${retries === 1 ? "y" : "ies"}`,
    `${compactions} compaction${compactions === 1 ? "" : "s"}`,
  ].join(" · ");
});

const changes = computed<ChangedFile[]>(() =>
  groupChangedFiles(props.group.items),
);

function isInFlight(item: Item): boolean {
  return isToolInFlight(item, props.run);
}

function isThinkingRunning(item: ActivityItem): boolean {
  return isThinkingStreaming(item, props.run);
}

function isInterrupted(item: ActivityItem): boolean {
  return runInterrupted(item, props.run);
}

function childDuration(item: ActivityItem): string {
  return elapsedLabel(itemElapsed(item, props.now), item.kind === "tool");
}
</script>
