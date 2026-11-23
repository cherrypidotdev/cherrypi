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
Button(
  variant="outline"
  :title="$t('taskExport.hint')"
  :aria-busy="exporting"
  @click="exportHistory"
  :disabled="exporting"
  data-testid="task-export-history"
)
  Spinner(v-if="exporting" data-icon="inline-start")
  DownloadIcon(v-else data-icon="inline-start")
  span {{ exporting ? $t("taskExport.exporting") : $t("taskExport.export") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { agency } from "@/lib/sdk";
import {
  buildTaskHistoryExport,
  collectTaskHistory,
  downloadTaskHistory,
  type TaskHistoryDetails,
} from "@/lib/taskExport";
import type { Task } from "@cherrypi/sdk";
import { DownloadIcon } from "@lucide/vue";
import { ref, watch } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{ task: Task }>();

const exporting = ref(false);

let inFlight: AbortController | undefined;

watch(
  () => props.task.id,
  () => inFlight?.abort(),
);

async function collectDetails(
  task: Task,
  signal: AbortSignal,
): Promise<TaskHistoryDetails> {
  const [project, agents, prompts] = await Promise.all([
    task.projectId
      ? agency.getProject(task.projectId, { signal })
      : Promise.resolve(null),
    task.assigneeActorId ? agency.listAgents({ signal }) : Promise.resolve([]),
    agency.getTaskSystemPrompt(task.id, { signal }),
  ]);
  const assignee =
    agents.find((agent) => agent.actorId === task.assigneeActorId) ?? null;
  const groups = assignee
    ? await agency.listAgentGroupsForAgent(assignee.id, { signal })
    : [];
  return { project, assignee, groups, prompts };
}

async function exportHistory() {
  if (exporting.value) return;

  const task = props.task;
  const controller = new AbortController();
  inFlight = controller;
  exporting.value = true;
  try {
    const [events, details] = await Promise.all([
      collectTaskHistory((page) => agency.getTaskHistory(task.id, page), {
        signal: controller.signal,
      }),
      collectDetails(task, controller.signal),
    ]);
    downloadTaskHistory(buildTaskHistoryExport(task, events, details), task.id);
  } catch (err) {
    if (!controller.signal.aborted) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  } finally {
    if (inFlight === controller) {
      inFlight = undefined;
      exporting.value = false;
    }
  }
}
</script>

<i18n lang="yaml">
en:
  taskExport:
    export: "Export history as JSON"
    exporting: "Exporting history…"
    hint: "Download the task as one JSON file: header, system events, messages, system prompts, project and agent context, plus every console event"
ru:
  taskExport:
    export: "Экспортировать историю в JSON"
    exporting: "Экспорт истории…"
    hint: "Скачать задачу одним JSON-файлом: заголовок, системные события, сообщения, системные промпты, проект и агент, а также все события консоли"
</i18n>
