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
Dialog(:open="open" @update:open="emit('update:open', $event)")
  DialogScrollContent.max-w-4xl
    DialogHeader
      DialogTitle Prompt history
      DialogDescription
        | Every saved version, newest first. Select a version to see its diff
        | against the one before it.
    p.text-destructive.text-sm(v-if="error") {{ error }}
    .grid.gap-4(class="md:grid-cols-[16rem_1fr]")
      .flex.flex-col.gap-1.overflow-y-auto.pr-1(class="max-h-[60vh]")
        Skeleton.h-12.w-full(v-if="loading && !versions.length")
        button.rounded-md.border.p-2.text-left.transition-colors(
          v-for="version in versions"
          :key="version.id.toString()"
          type="button"
          :class="selected?.id === version.id ? 'bg-muted' : 'hover:bg-muted/60'"
          @click="selected = version"
        )
          .flex.items-center.justify-between.gap-2
            span.text-sm.font-medium Version {{ version.version }}
            Badge(
              v-if="version.version === currentVersion"
              variant="secondary"
            ) current
          .text-muted-foreground.flex.items-center.text-xs(class="gap-1.5")
            ActorAvatar(
              :name="version.author?.name"
              :username="version.author?.username"
              :size="14"
            )
            span {{ authorLabel(version) }}
          span.text-muted-foreground.text-xs {{ formatTimestamp(version.updatedAt) }}
          p.text-xs(v-if="version.note") {{ version.note }}
          .flex.flex-wrap.gap-1(
            v-if="version.approvedBy || version.autoApproved"
          )
            Badge(v-if="version.autoApproved" variant="outline") auto-approved
            span.text-muted-foreground.text-xs(v-else-if="version.approvedBy") Approved by {{ actorLabel(version.approvedBy) }}
      .flex.flex-col.gap-2
        p.text-muted-foreground.text-sm(v-if="!versions.length && !loading")
          | No versions yet.
        template(v-if="selected")
          PromptDiff(
            :label="`Version ${selected.version} vs ${selected.version > 1n ? selected.version - 1n : 0n}`"
            :before="olderContent"
            :after="selected.content"
          )
          .text-muted-foreground.flex.flex-wrap.items-center.gap-2.text-xs
            span By {{ authorLabel(selected) }}
            span ·
            span {{ formatTimestamp(selected.updatedAt) }}
            template(v-if="selected.approvedBy")
              span ·
              span Approved by {{ actorLabel(selected.approvedBy) }}
            span(v-else-if="selected.autoApproved") · Auto-approved
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import PromptDiff from "@/components/PromptDiff.vue";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogDescription,
  DialogHeader,
  DialogScrollContent,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { formatTimestamp } from "@/lib/format";
import { agency } from "@/lib/sdk";
import type {
  Prompt,
  PromptActor,
  PromptScope,
  PromptTarget,
} from "@cherrypi/sdk";
import { computed, ref, watch } from "vue";

const props = defineProps<{
  open: boolean;
  scope: PromptScope;
  target: PromptTarget;
  currentVersion: bigint;
}>();

const emit = defineEmits<{ "update:open": [boolean] }>();

const versions = ref<Prompt[]>([]);
const selected = ref<Prompt>();
const loading = ref(false);
const error = ref("");

function authorLabel(version: Prompt): string {
  return actorLabel(version.author);
}

function actorLabel(actor?: PromptActor): string {
  if (!actor) return "Unknown";
  return actor.name || actor.username || "Unknown";
}

const olderContent = computed(() => {
  if (!selected.value) return "";
  const index = versions.value.findIndex((v) => v.id === selected.value?.id);
  if (index < 0) return "";
  const older = versions.value[index + 1];
  return older?.content ?? "";
});

async function load() {
  loading.value = true;
  try {
    versions.value = await agency.listPromptVersions(props.scope, props.target);
    selected.value = versions.value[0];
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

watch(
  () => [
    props.open,
    props.scope,
    props.target.agentId,
    props.target.teamId,
    props.target.projectId,
  ],
  ([open]) => {
    if (open) void load();
  },
  { immediate: true },
);
</script>
