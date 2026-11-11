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
Dialog(v-model:open="open")
  DialogContent.overflow-y-auto.overscroll-contain(class="max-h-[90vh] sm:max-w-2xl")
    DialogHeader
      DialogTitle {{ $t("nav.newTask") }}
      DialogDescription(v-if="relationNote")
        | {{ relationNote }}
      DialogDescription(v-else)
        | {{ $t("taskComposer.description") }}

    ExpressTaskForm(
      bare
      :project-id="relatedTask?.projectId"
      :default-project-id="defaultProjectId"
      :related-task="relatedTask"
      :relation="relation"
      @created="$emit('created', $event)"
    )
</template>

<script setup lang="ts">
import ExpressTaskForm from "@/components/tasks/ExpressTaskForm.vue";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { newTaskRelationNote, type NewTaskRelation } from "@/lib/task";
import type { Task } from "@cherrypi/sdk";
import { computed } from "vue";

const props = defineProps<{
  relatedTask?: Task | null;

  relation?: NewTaskRelation;

  defaultProjectId?: string;
}>();

const emit = defineEmits<{ created: [Task] }>();

const open = defineModel<boolean>("open", { default: false });

const relationNote = computed(() =>
  props.relatedTask
    ? newTaskRelationNote(props.relation, props.relatedTask.id)
    : "",
);
</script>

<i18n lang="yaml">
en:
  taskComposer:
    description: "Describe the work and queue as many as you need — creating a task keeps this composer open."
ru:
  taskComposer:
    description: "Опишите работу и ставьте в очередь сколько нужно — после создания задачи композер остаётся открытым."
</i18n>
