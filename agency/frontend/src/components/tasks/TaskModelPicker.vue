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
.flex.flex-wrap.items-center.gap-2(data-testid="task-model-picker")
  Select(:model-value="modelChoice" @update:model-value="onModelChange")
    SelectTrigger(
      size="sm"
      title="Model for this task"
      :disabled="!canPickModel"
      data-testid="task-model"
    )
      SelectValue(placeholder="Model")
        | {{ modelDisplayLabel }}
    SelectContent
      SelectGroup
        SelectItem(
          v-for="model in modelOptions"
          :key="`${model.provider}/${model.id}`"
          :value="`${model.provider}/${model.id}`"
        ) {{ model.name || model.id }}
  Select(:model-value="thinkingChoice" @update:model-value="onThinkingChange")
    SelectTrigger(
      size="sm"
      title="Thinking level for this task"
      :disabled="!canPickThinking"
      data-testid="task-thinking"
    )
      SelectValue(placeholder="Thinking level")
        | {{ thinkingDisplayLabel }}
    SelectContent
      SelectGroup
        SelectItem(
          v-for="level in thinkingOptions"
          :key="level"
          :value="level"
        ) {{ level }}
</template>

<script setup lang="ts">
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { agency } from "@/lib/sdk";
import type { TaskModelSelection } from "@/lib/task";
import { create } from "@bufbuild/protobuf";
import {
  TaskModelSchema,
  type AgentDefaults,
  type TaskConfig,
  type TaskModel,
} from "@cherrypi/sdk";
import { computed, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    // The concrete agent the task is assigned to. No agent means no picker.
    agentId?: string;
    // The current per-task selection; empty fields mean the agent default.
    selection: TaskModelSelection;
    // An existing task's live config. When supplied, its model catalog wins
    // over probing the agent for defaults (the follow-up composer already has
    // it from GetTaskConfig).
    config?: TaskConfig;
    // The create composer resolves the agent's defaults and shows them as the
    // current values. Selecting a default value clears the override instead of
    // pinning it. The follow-up composer already knows the live session's
    // values, so it passes false.
    useDefaults?: boolean;
    disabled?: boolean;
  }>(),
  { agentId: "", config: undefined, useDefaults: true, disabled: false },
);

const emit = defineEmits<{
  (event: "change", value: TaskModelSelection): void;
}>();

const taskDefaults = ref<TaskConfig | undefined>();
const agentDefaults = ref<AgentDefaults | undefined>();
const loading = ref(false);
let generation = 0;

// The picker probes the agent for its catalog (create form), while the
// follow-up composer passes the task's live config. Agent defaults are only
// needed to show what an unset override resolves to.
watch(
  () => [props.agentId, props.config, props.useDefaults] as const,
  async ([agentId, config, useDefaults]) => {
    const gen = ++generation;
    taskDefaults.value = undefined;
    agentDefaults.value = undefined;
    if (!agentId || (config && !useDefaults)) {
      loading.value = false;
      return;
    }
    loading.value = true;
    const catalogRequest = config
      ? Promise.resolve(config)
      : Promise.resolve().then(() => agency.getTaskDefaults(agentId));
    const defaultsRequest = useDefaults
      ? Promise.resolve().then(() => agency.getAgentDefaults(agentId))
      : Promise.resolve(undefined);
    const [catalog, defaults] = await Promise.allSettled([
      catalogRequest,
      defaultsRequest,
    ]);
    if (gen !== generation) return;
    if (catalog.status === "fulfilled") taskDefaults.value = catalog.value;
    if (defaults.status === "fulfilled") {
      agentDefaults.value = defaults.value ?? undefined;
    }
    loading.value = false;
  },
  { immediate: true },
);

const defaultProvider = computed(
  () =>
    agentDefaults.value?.modelProvider ||
    taskDefaults.value?.modelProvider ||
    "",
);
const defaultModelId = computed(
  () => agentDefaults.value?.modelId || taskDefaults.value?.modelId || "",
);
const defaultModelName = computed(
  () =>
    agentDefaults.value?.modelName ||
    taskDefaults.value?.modelName ||
    defaultModelId.value,
);

// The model the task actually uses when no explicit override is set.
const effectiveProvider = computed(
  () => props.selection.modelProvider || defaultProvider.value,
);
const effectiveModelId = computed(
  () => props.selection.modelId || defaultModelId.value,
);
const onDefaultModel = computed(
  () =>
    !props.selection.modelProvider ||
    (props.selection.modelProvider === defaultProvider.value &&
      props.selection.modelId === defaultModelId.value),
);

const catalogModels = computed<TaskModel[]>(
  () => props.config?.models ?? taskDefaults.value?.models ?? [],
);
const modelOptions = computed<TaskModel[]>(() => {
  const list = [...catalogModels.value];
  const has = (provider: string, id: string) =>
    list.some((model) => model.provider === provider && model.id === id);
  if (
    defaultProvider.value &&
    !has(defaultProvider.value, defaultModelId.value)
  ) {
    list.unshift(
      create(TaskModelSchema, {
        provider: defaultProvider.value,
        id: defaultModelId.value,
        name: defaultModelName.value,
        thinkingLevels: [],
      }),
    );
  }
  if (
    props.selection.modelProvider &&
    !has(props.selection.modelProvider, props.selection.modelId)
  ) {
    // Keep a selection made on another agent (or one Pi no longer lists)
    // visible, so the picker never silently shows a different model.
    list.unshift(
      create(TaskModelSchema, {
        provider: props.selection.modelProvider,
        id: props.selection.modelId,
        name: props.selection.modelName || props.selection.modelId,
        reasoning: false,
        thinkingLevels: [],
      }),
    );
  }
  return list;
});

const selectedModel = computed(() =>
  modelOptions.value.find(
    (model) =>
      model.provider === effectiveProvider.value &&
      model.id === effectiveModelId.value,
  ),
);
const defaultThinkingLevel = computed(() => {
  if (!props.useDefaults || !onDefaultModel.value) return "";
  return (
    agentDefaults.value?.thinkingLevel ||
    taskDefaults.value?.thinkingLevel ||
    ""
  );
});
const thinkingOptions = computed(() => {
  const levels = [
    ...(selectedModel.value
      ? selectedModel.value.thinkingLevels
      : (props.config?.thinkingLevels ??
        taskDefaults.value?.thinkingLevels ??
        [])),
  ];
  const current = props.selection.thinkingLevel || defaultThinkingLevel.value;
  if (current && !levels.includes(current)) levels.push(current);
  return levels;
});

const modelChoice = computed(() =>
  props.selection.modelProvider
    ? `${props.selection.modelProvider}/${props.selection.modelId}`
    : props.useDefaults && defaultProvider.value
      ? `${defaultProvider.value}/${defaultModelId.value}`
      : "",
);
const thinkingChoice = computed(
  () => props.selection.thinkingLevel || defaultThinkingLevel.value,
);
const modelDisplayLabel = computed(() => {
  if (props.selection.modelProvider) {
    return (
      selectedModel.value?.name ||
      props.selection.modelName ||
      props.selection.modelId
    );
  }
  return selectedModel.value?.name || "Model";
});
const thinkingDisplayLabel = computed(
  () =>
    props.selection.thinkingLevel ||
    defaultThinkingLevel.value ||
    "Thinking level",
);

const canPickModel = computed(
  () => !props.disabled && !loading.value && modelOptions.value.length > 0,
);
const canPickThinking = computed(
  () => !props.disabled && !loading.value && thinkingOptions.value.length > 0,
);

function onModelChange(value: unknown) {
  const choice = String(value);
  const model = modelOptions.value.find(
    (item) => `${item.provider}/${item.id}` === choice,
  );
  if (!model) return;
  const levels = model.thinkingLevels ?? [];
  const current = props.selection.thinkingLevel;
  const thinkingLevel = current && levels.includes(current) ? current : "";
  const isDefault =
    props.useDefaults &&
    model.provider === defaultProvider.value &&
    model.id === defaultModelId.value;
  emit("change", {
    modelProvider: isDefault ? "" : model.provider,
    modelId: isDefault ? "" : model.id,
    modelName: isDefault ? "" : model.name,
    thinkingLevel:
      isDefault && thinkingLevel === defaultThinkingLevel.value
        ? ""
        : thinkingLevel,
  });
}

function onThinkingChange(value: unknown) {
  const level = String(value);
  emit("change", {
    ...props.selection,
    thinkingLevel:
      props.useDefaults && level === defaultThinkingLevel.value ? "" : level,
  });
}
</script>
