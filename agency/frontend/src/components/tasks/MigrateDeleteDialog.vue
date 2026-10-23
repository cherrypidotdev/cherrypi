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
  DialogContent(class="sm:max-w-md")
    DialogHeader
      DialogTitle History is not stored in the agency
      DialogDescription
        | "{{ task ? taskName(task) : "" }}" still has its Pi conversation only
        | on the agent. Migrating copies it into the agency so it survives this
        | deletion; deleting now would lose the conversation permanently.
    Alert(v-if="error" variant="destructive")
      AlertTitle Migration failed
      AlertDescription {{ error }}
    DialogFooter.flex.flex-col.gap-2(class="sm:flex-row sm:justify-end")
      Button(
        type="button"
        variant="outline"
        @click="emit('update:open', false)"
        :disabled="busy"
      ) Cancel
      Button(
        type="button"
        variant="outline"
        @click="removeWithoutMigrating"
        :disabled="busy || !task"
      )
        span Delete without migrating
      Button(
        type="button"
        variant="destructive"
        @click="migrateThenDelete"
        :disabled="busy || !task"
      )
        Spinner(v-if="busy" data-icon="inline-start")
        DatabaseIcon(v-else data-icon="inline-start")
        span Migrate &amp; delete
</template>

<script setup lang="ts">
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import { agent } from "@/lib/sdk";
import { taskName } from "@/lib/task";
import type { Task } from "@cherrypi/sdk";
import { DatabaseIcon } from "@lucide/vue";
import { ref, watch } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{
  open: boolean;
  task: Task | null;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  deleted: [];
}>();

const busy = ref(false);
const error = ref("");

watch(
  () => [props.open, props.task?.id] as const,
  ([open]) => {
    if (open) error.value = "";
  },
);

// migrateThenDelete stores the conversation first, then deletes the task; a
// failed migration keeps the dialog open so the user can retry or delete
// anyway.
async function migrateThenDelete() {
  const task = props.task;
  if (!task) return;
  busy.value = true;
  error.value = "";
  try {
    const result = await agent.migrateTaskHistory(task.id);
    if (!result.task.historyMigrated) {
      error.value =
        "The agent had no stored history, so nothing could be migrated.";
      return;
    }
    await finishDelete(task.id);
    toast.success("History migrated, task deleted");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = false;
  }
}

async function removeWithoutMigrating() {
  const task = props.task;
  if (!task) return;
  busy.value = true;
  error.value = "";
  try {
    await finishDelete(task.id);
    toast.success("Task deleted");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = false;
  }
}

async function finishDelete(id: string) {
  await agent.deleteTask(id);
  emit("update:open", false);
  emit("deleted");
}
</script>
