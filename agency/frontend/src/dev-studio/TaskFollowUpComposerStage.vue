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
TaskFollowUpComposer(
  v-bind="composerProps"
  v-model:prompt="prompt"
  v-model:tab="tab"
  @send="emit('send')"
  @enter="emit('enter')"
  @stop="emit('stop')"
  @model-change="emit('modelChange', $event)"
  @staged-uploaded="emit('stagedUploaded', $event)"
  @remove-staged="emit('removeStaged', $event)"
  @open-attachment="emit('openAttachment', $event)"
  @open-new-sketch="emit('openNewSketch')"
  @open-sketch="emit('openSketch', $event)"
)
</template>

<script setup lang="ts">
import TaskFollowUpComposer from "@/components/tasks/TaskFollowUpComposer.vue";
import { ref, useAttrs, watch, type VNodeProps } from "vue";

defineOptions({ inheritAttrs: false });

type ComposerProps = Omit<
  InstanceType<typeof TaskFollowUpComposer>["$props"],
  keyof VNodeProps | "prompt" | "tab"
>;
const composerProps = useAttrs() as unknown as ComposerProps;

const props = withDefaults(
  defineProps<{ prompt?: string; tab?: "write" | "preview" }>(),
  { prompt: "", tab: "write" },
);

const emit = defineEmits<{
  (e: "send"): void;
  (e: "enter"): void;
  (e: "stop"): void;
  (e: "modelChange", selection: unknown): void;
  (e: "stagedUploaded", attachment: unknown): void;
  (e: "removeStaged", attachment: unknown): void;
  (e: "openAttachment", attachment: unknown): void;
  (e: "openNewSketch"): void;
  (e: "openSketch", attachment: unknown): void;
}>();

const prompt = ref(props.prompt);
const tab = ref(props.tab);

watch(
  () => props.prompt,
  (value) => {
    prompt.value = value;
  },
);
watch(
  () => props.tab,
  (value) => {
    tab.value = value;
  },
);
</script>
