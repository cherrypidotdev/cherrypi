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
.flex.flex-col.gap-4
  .flex.flex-wrap.items-center.justify-between.gap-3
    .flex.flex-col
      h2.text-lg.font-semibold {{ $t("agentMemory.title") }}
      p.text-muted-foreground.max-w-3xl.text-sm
        | {{ $t("agentMemory.hint") }}
    Button(variant="outline" size="sm" @click="load" :disabled="loading")
      Spinner(v-if="loading" data-icon="inline-start")
      span {{ $t("agentMemory.refresh") }}
  p.text-muted-foreground.text-sm(v-if="forbidden")
    | {{ $t("agentMemory.private") }}
  p.text-destructive.text-sm(v-if="error") {{ error }}
  .grid.items-start.gap-4(v-if="!forbidden" class="lg:grid-cols-[18rem_minmax(0,1fr)]")
    Card
      CardHeader.pb-3
        CardTitle.text-sm {{ $t("agentMemory.files") }}
      CardContent.flex.flex-col.gap-1
        button.w-full.rounded-lg.px-2.py-2.text-left.transition-colors(
          v-for="file in files"
          :key="file.path"
          type="button"
          :class="selected?.path === file.path ? 'bg-muted' : 'hover:bg-muted/60'"
          @click="select(file)"
        )
          .flex.items-center.gap-2
            FileTextIcon.text-muted-foreground.size-3.shrink-0
            span.min-w-0.flex-1.truncate.text-sm.font-medium {{ file.title || file.path }}
          .flex.items-center.justify-between.gap-2.pl-5(class="mt-0.5")
            span.text-muted-foreground.truncate.font-mono(class="text-[10px]") {{ file.path }}
            span.text-muted-foreground.shrink-0(class="text-[10px]") {{ formatBytes(file.bodySize) }}
        p.text-muted-foreground.text-sm(v-if="!files.length && !loading")
          | {{ $t("agentMemory.empty") }}
    Card(v-if="selected")
      CardHeader
        CardTitle.text-base {{ selected.title || selected.path }}
        CardDescription.font-mono.text-xs {{ selected.path }}
      CardContent
        MarkdownText.prose.prose-sm.max-w-none(
          class="prose-p:my-1 prose-pre:my-2"
          :text="selected.body"
        )
    Card(v-else)
      CardContent.flex.flex-col.items-center.gap-3.p-10.text-center
        FileTextIcon.text-muted-foreground.size-8
        p.text-sm.font-medium {{ $t("agentMemory.select") }}
        p.text-muted-foreground.text-sm
          | {{ $t("agentMemory.selectHint") }}
</template>

<script setup lang="ts">
import MarkdownText from "@/components/MarkdownText.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { formatBytes } from "@/lib/format";
import { agency } from "@/lib/sdk";
import type { MemoryFile } from "@cherrypi/sdk";
import { FileTextIcon } from "@lucide/vue";
import { onMounted, ref, watch } from "vue";

const props = defineProps<{ agentId: string }>();

const files = ref<MemoryFile[]>([]);
const selected = ref<MemoryFile>();
const loading = ref(false);
const error = ref("");
const forbidden = ref(false);

async function select(file: MemoryFile) {
  try {
    selected.value = await agency.getAgentMemoryFile(props.agentId, file.path);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function load() {
  if (!props.agentId) return;
  loading.value = true;
  error.value = "";
  forbidden.value = false;
  try {
    files.value = await agency.listAgentMemory(props.agentId);
    if (files.value.length) await select(files.value[0]!);
    else selected.value = undefined;
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    if (/permission|denied|forbidden/i.test(message)) {
      forbidden.value = true;
    } else {
      error.value = message;
    }
    files.value = [];
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.agentId,
  () => void load(),
);
onMounted(() => void load());
</script>

<i18n lang="yaml">
en:
  agentMemory:
    title: "Agent memory"
    hint: "The agent's private working memory. It is read-only here: only the agent writes it, and it ranks below approved knowledge in its context."
    refresh: "Refresh"
    private: "This memory is private to the agent. Its owners and administrators may inspect it; nobody else can."
    files: "Files"
    empty: "This agent has not written any memory yet."
    select: "Select a memory file"
    selectHint: "Long-term memory, the scratchpad and daily logs are stored in the agency and injected as a stable snapshot."
ru:
  agentMemory:
    title: "Память агента"
    hint: "Приватная рабочая память агента. Здесь она только для чтения: писать может лишь сам агент, и в его контексте она ниже подтверждённых знаний."
    refresh: "Обновить"
    private: "Эта память приватна для агента. Её владельцы и администраторы могут просматривать её; больше никто."
    files: "Файлы"
    empty: "Этот агент ещё ничего не записал в память."
    select: "Выберите файл памяти"
    selectHint: "Долговременная память, черновик и дневные журналы хранятся в агентстве и внедряются как стабильный снимок."
</i18n>
