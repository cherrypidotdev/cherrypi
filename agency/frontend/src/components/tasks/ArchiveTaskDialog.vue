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
      DialogTitle Archive {{ noun }}
      DialogDescription
        | Archiving removes the {{ possessive }}
        | agent-side Pi session and workspace. That is safe: the session
        | history, attachments, and artifacts are stored in the agency.
    .flex.flex-col.gap-2
      p.text-muted-foreground.text-xs(v-if="single")
        | {{ taskName(single) }}
      p.text-muted-foreground.text-xs(v-else)
        | {{ tasks.length }} tasks selected:
        |
        span.font-mono {{ idsLabel }}
      label.mt-1.flex.items-center.gap-2.text-sm
        Switch(
          :model-value="dontShowAgain"
          @update:model-value="dontShowAgain = Boolean($event)"
        )
        span Do not show this again
    DialogFooter
      Button(
        type="button"
        variant="outline"
        @click="emit('update:open', false)"
      )
        | Cancel
      Button(type="button" @click="confirm" :disabled="busy")
        Spinner(v-if="busy" data-icon="inline-start")
        ArchiveIcon(v-else data-icon="inline-start")
        span Archive
</template>

<script setup lang="ts">
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
import { Switch } from "@/components/ui/switch";
import { taskName } from "@/lib/task";
import type { Task } from "@cherrypi/sdk";
import { ArchiveIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";

const props = defineProps<{
  open: boolean;
  tasks: Task[];
  busy?: boolean;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  confirm: [dontShowAgain: boolean];
}>();

const dontShowAgain = ref(false);

const single = computed(() =>
  props.tasks.length === 1 ? props.tasks[0] : undefined,
);
const noun = computed(() => (props.tasks.length === 1 ? "task" : "tasks"));
const possessive = computed(() =>
  props.tasks.length === 1 ? "task's" : "tasks'",
);
const idsLabel = computed(() => props.tasks.map((task) => task.id).join(", "));

watch(
  () => props.open,
  (open) => {
    if (open) dontShowAgain.value = false;
  },
);

function confirm() {
  emit("confirm", dontShowAgain.value);
}
</script>
