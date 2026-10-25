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
  Textarea.field-sizing-fixed.resize-y(
    v-model="content"
    :placeholder="placeholder"
    :disabled="loading"
  )
  p.text-destructive.text-sm(v-if="error") {{ error }}
  .flex.items-center.justify-end.gap-2
    span.text-muted-foreground.text-xs Version {{ version }}
    Button(type="button" @click="saveClick" :disabled="saving || loading")
      Spinner(v-if="saving" data-icon="inline-start")
      CheckIcon(v-else data-icon="inline-start")
      span Save
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import { usePrompt } from "@/composables/usePrompt";
import { PromptScope } from "@cherrypi/sdk";
import { CheckIcon } from "@lucide/vue";
import { toast } from "vue3-toastify";

const props = defineProps<{
  scope: PromptScope;
  agentId?: string;
  teamId?: string;
  projectId?: string;
  placeholder?: string;
}>();

const { content, version, loading, saving, error, save } = usePrompt(
  props.scope,
  () => ({
    agentId: props.agentId ?? "",
    teamId: props.teamId ?? "",
    projectId: props.projectId ?? "",
  }),
);

async function saveClick() {
  if (await save()) {
    toast.success("Prompt saved");
  }
}
</script>
