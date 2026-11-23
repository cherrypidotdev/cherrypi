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
.flex.flex-col.gap-2(data-testid="task-compactions")
  p.text-muted-foreground.text-xs(v-if="loading") {{ $t("taskCompactions.loading") }}
  p.text-muted-foreground.text-xs(v-else-if="error") {{ error }}
  p.text-muted-foreground.text-xs(
    v-else-if="compactions.length === 0"
    data-testid="task-compactions-empty"
  ) {{ $t("taskCompactions.empty") }}
  CompactionLine(
    v-for="(compaction, index) in compactions"
    :key="index"
    :before-tokens="compaction.contextTokensBefore"
    :after-tokens="compaction.contextTokensAfter"
    :duration-ms="compaction.durationMs"
    :reason="compaction.reason"
    :at="atMs(compaction.at)"
  )
</template>

<script setup lang="ts">
import CompactionLine from "@/components/tasks/CompactionLine.vue";
import { timestampMs } from "@/lib/console";
import { agency } from "@/lib/sdk";
import type { TaskCompaction } from "@cherrypi/sdk";
import { onMounted, ref } from "vue";

const props = defineProps<{
  taskId: string;

  preview?: TaskCompaction[];
}>();

const compactions = ref<TaskCompaction[]>([]);
const loading = ref(true);
const error = ref("");

function atMs(at: TaskCompaction["at"]): number | undefined {
  return timestampMs(at);
}

onMounted(async () => {
  if (props.preview !== undefined) {
    compactions.value = props.preview;
    loading.value = false;
    return;
  }
  try {
    compactions.value = await agency.listTaskCompactions(props.taskId);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
});
</script>

<i18n lang="yaml">
en:
  taskCompactions:
    loading: "Loading compactions…"
    empty: "No context compactions recorded yet."
ru:
  taskCompactions:
    loading: "Загружаем сжатия…"
    empty: "Сжатий контекста пока нет."
</i18n>
