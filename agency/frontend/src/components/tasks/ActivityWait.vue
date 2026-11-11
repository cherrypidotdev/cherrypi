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
.flex.items-baseline.gap-2.px-2.text-xs(
  :class="kind === ModelWait_Kind.RETRY ? 'text-amber-600 dark:text-amber-400' : 'text-muted-foreground'"
  :title="hint"
  :data-live="live ? 'true' : undefined"
  data-testid="activity-wait"
)
  Spinner.text-muted-foreground.size-3.shrink-0(v-if="live")
  component.text-muted-foreground.size-3.shrink-0(v-else :is="icon")
  span.shrink-0 {{ lead }}
  span.flex-1.-translate-y-1.border-b.border-dotted(class="border-border/70")
  LiveSpan.shrink-0(v-if="live" :since="since")
  span.shrink-0(v-else) {{ formatElapsed(durationMs) }}
</template>

<script setup lang="ts">
import LiveSpan from "@/components/tasks/LiveSpan.vue";
import { Spinner } from "@/components/ui/spinner";
import { formatElapsed } from "@/lib/format";
import { ModelWait_Kind, PendingActivity_Kind } from "@cherrypi/sdk";
import { ArchiveIcon, ClockIcon, RefreshCwIcon } from "@lucide/vue";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    kind: ModelWait_Kind | PendingActivity_Kind;

    durationMs?: number;

    since?: number;

    live?: boolean;

    model?: string;

    provider?: string;

    error?: string;

    attempt?: number;

    maxAttempts?: number;

    reason?: string;

    fallbackProvider?: string;

    fallbackModel?: string;
  }>(),
  {
    durationMs: 0,
    since: 0,
    live: false,
    model: "",
    provider: "",
    error: "",
    attempt: 0,
    maxAttempts: 0,
    reason: "",
    fallbackProvider: "",
    fallbackModel: "",
  },
);

const label = computed(() => {
  switch (props.kind) {
    case ModelWait_Kind.RETRY:
      return props.live ? "Retrying the request" : "Retry";
    case ModelWait_Kind.COMPACTION:
      return props.live ? "Compacting the context" : "Compacted context";
    default:
      return "Waiting for model";
  }
});

const model = computed(() => {
  const provider = props.provider || props.fallbackProvider;
  const name = props.model || props.fallbackModel;
  return provider && name ? `${provider}/${name}` : name;
});

const detail = computed(() => {
  switch (props.kind) {
    case ModelWait_Kind.RETRY:
      if (props.live) {
        return props.maxAttempts
          ? `attempt ${props.attempt} of ${props.maxAttempts}`
          : "";
      }
      return [model.value, truncateError(props.error)]
        .filter(Boolean)
        .join(" · ");
    case ModelWait_Kind.COMPACTION:
      return props.reason ? compactionReason(props.reason) : "";
    default:
      return model.value;
  }
});

const lead = computed(() =>
  [label.value, detail.value].filter(Boolean).join(" · "),
);

const hint = computed(() => {
  switch (props.kind) {
    case ModelWait_Kind.RETRY: {
      if (props.live) return props.error;

      const attempt = props.attempt ? `attempt ${props.attempt}` : "";
      return [attempt, props.error].filter(Boolean).join(" · ");
    }
    case ModelWait_Kind.COMPACTION:
      return compactionReason(props.reason);
    default:
      return "";
  }
});

const icon = computed(() => {
  switch (props.kind) {
    case ModelWait_Kind.RETRY:
      return RefreshCwIcon;
    case ModelWait_Kind.COMPACTION:
      return ArchiveIcon;
    default:
      return ClockIcon;
  }
});

function compactionReason(reason: string): string {
  switch (reason) {
    case "threshold":
      return "context full";
    case "overflow":
      return "context overflowed";
    case "manual":
      return "requested";
    default:
      return reason;
  }
}

function truncateError(error: string): string {
  const text = error.trim();
  if (!text) return "";
  return text.length > 60 ? `${text.slice(0, 59)}…` : text;
}
</script>
