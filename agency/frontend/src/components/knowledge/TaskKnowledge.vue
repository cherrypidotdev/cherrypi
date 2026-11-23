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
.flex.flex-col.gap-2
  p.text-muted-foreground.text-xs {{ $t("taskKnowledge.hint") }}
  p.text-destructive.text-xs(v-if="error") {{ error }}
  .flex.flex-col.gap-1
    RouterLink.rounded-lg.border.p-2.transition-colors(
      v-for="bit in bits"
      :key="bit.id"
      class="hover:bg-muted/60"
      :to="{ name: 'knowledge', query: { bit: bit.id } }"
    )
      .flex.items-center.gap-2
        span.min-w-0.flex-1.truncate.text-sm.font-medium {{ bit.title || bit.content }}
        Badge(
          class="text-[10px]"
          :variant="bit.status === 'approved' ? 'default' : 'secondary'"
        )
          | {{ bit.status }}
      .mt-1.flex.flex-wrap.items-center.gap-1
        span.text-muted-foreground.font-mono(class="text-[10px]") {{ bit.originTaskId === taskId ? $t("taskKnowledge.origin") : bit.originTaskId }}
        span.text-muted-foreground(v-if="bit.tags.length" class="text-[10px]")
          | · {{ bit.tags.map((tag) => "#" + tag).join(" ") }}
  p.text-muted-foreground.text-sm(v-if="!bits.length && !loading")
    | {{ $t("taskKnowledge.empty") }}
  Spinner.size-3(v-if="loading")
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { useRealtime } from "@/composables/useRealtime";
import { agency } from "@/lib/sdk";
import type { KnowledgeBit } from "@cherrypi/sdk";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";

const props = defineProps<{ taskId: string }>();

const { t } = useI18n();
const bits = ref<KnowledgeBit[]>([]);
const loading = ref(false);
const error = ref("");

async function load() {
  if (!props.taskId) return;
  loading.value = true;
  error.value = "";
  try {
    const [approved, pending] = await Promise.all([
      agency.listKnowledge({ taskId: props.taskId, status: "approved" }),
      agency.listKnowledge({ taskId: props.taskId, status: "pending" }),
    ]);
    bits.value = [...pending, ...approved];
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});

useRealtime(["knowledge"], () => {
  void load();
});

void t;
</script>

<i18n lang="yaml">
en:
  taskKnowledge:
    hint: "Knowledge bits recorded from this task or citing it as a source."
    origin: "recorded here"
    empty: "No knowledge recorded from this task yet."
ru:
  taskKnowledge:
    hint: "Факты знаний, записанные из этой задачи или ссылающиеся на неё."
    origin: "записано здесь"
    empty: "Из этой задачи пока не записано знаний."
</i18n>
