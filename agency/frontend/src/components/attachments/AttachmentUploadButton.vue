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
span.inline-flex
  Button(
    type="button"
    variant="outline"
    size="sm"
    :title="title"
    @click="pick"
    :disabled="uploading"
  )
    Spinner(v-if="uploading" data-icon="inline-start")
    PaperclipIcon(v-else data-icon="inline-start")
    span {{ label }}
  input.hidden(
    ref="input"
    type="file"
    multiple
    :accept="accept"
    @change="onChange"
  )
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useAttachmentUpload } from "@/composables/useAttachmentUpload";
import type { Attachment } from "@cherrypi/sdk";
import { PaperclipIcon } from "@lucide/vue";
import { ref } from "vue";

const props = withDefaults(
  defineProps<{
    taskId?: string;
    agentId?: string;
    projectId?: string;
    label?: string;
    title?: string;
    accept?: string;
  }>(),
  { label: "Attach", title: "Attach files", accept: "" },
);
const emit = defineEmits<{ uploaded: [attachment: Attachment] }>();

const input = ref<HTMLInputElement>();
const { uploading, upload } = useAttachmentUpload(() => ({
  taskId: props.taskId,
  agentId: props.agentId,
  projectId: props.projectId,
}));

function pick() {
  input.value?.click();
}

async function onChange(event: Event) {
  const element = event.target as HTMLInputElement;
  const files = Array.from(element.files ?? []);
  element.value = "";
  if (!files.length) return;

  for (const attachment of await upload(files)) {
    emit("uploaded", attachment);
  }
}
</script>
