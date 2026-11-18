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
      :title="$t('taskModel.title')"
      :class="manualModel ? MANUAL_CLASS : ''"
      :disabled="!canPickModel"
      data-testid="task-model"
    )
      SparklesIcon.text-primary.size-3.shrink-0(
        v-if="showingRouted"
        :aria-label="$t('taskModel.routed')"
        data-testid="task-model-routed"
      )
      SelectValue(:placeholder="$t('domainLog.kind.model')")
        ModelName(v-if="selectedModel" :model="selectedModel")
    SelectContent
      SelectGroup
        SelectItem(v-if="autoChoice" value="__auto__")
          .flex.items-center.gap-1
            SparklesIcon.text-primary.size-3
            span {{ autoLabel }}
        SelectItem(
          v-for="model in modelOptions"
          :key="`${model.provider}/${model.id}`"
          :value="`${model.provider}/${model.id}`"
        )
          ModelName(:model="model")
  Select(:model-value="thinkingChoice" @update:model-value="onThinkingChange")
    SelectTrigger(
      size="sm"
      :title="$t('taskModel.thinkingHint')"
      :class="manualThinking ? MANUAL_CLASS : ''"
      :disabled="!canPickThinking"
      data-testid="task-thinking"
    )
      SparklesIcon.text-primary.size-3.shrink-0(
        v-if="showingRoutedThinking"
        :aria-label="$t('taskModel.routed')"
        data-testid="task-thinking-routed"
      )
      SelectValue(:placeholder="$t('taskModel.thinking')")
        | {{ thinkingDisplayLabel }}
    SelectContent
      SelectGroup
        SelectItem(v-if="showAutomaticThinking" value="__auto__")
          .flex.items-center.gap-1
            SparklesIcon.text-primary.size-3
            span {{ $t("taskModel.automatic") }}
        SelectItem(
          v-for="level in thinkingOptions"
          :key="level"
          :value="level"
        ) {{ level }}
</template>

<script setup lang="ts">
import ModelName from "@/components/ModelName.vue";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { sortThinkingLevels, type TaskRoutingContext } from "@/lib/model";
import { agency } from "@/lib/sdk";
import type { TaskModelSelection } from "@/lib/task";
import type {
  AgentDefaults,
  ModelSelectionInput,
  RoutingModelOption,
  TaskConfig,
  TaskModel,
} from "@cherrypi/sdk";
import { SparklesIcon } from "@lucide/vue";
import { computed, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const MANUAL_CLASS = "ring-1 ring-primary/40";

const props = withDefaults(
  defineProps<{
    agentId?: string;

    selection: TaskModelSelection;

    config?: TaskConfig;

    useDefaults?: boolean;
    disabled?: boolean;

    routing?: TaskRoutingContext;
  }>(),
  {
    agentId: "",
    config: undefined,
    useDefaults: true,
    disabled: false,
    routing: undefined,
  },
);

const emit = defineEmits<{
  (event: "change", value: TaskModelSelection): void;
}>();

const { t } = useI18n();

interface ModelChoice {
  provider: string;
  id: string;
  name: string;
  reasoning: boolean;
  thinkingLevels: string[];
}

const taskDefaults = ref<TaskConfig | undefined>();
const agentDefaults = ref<AgentDefaults | undefined>();
const loadingCatalog = ref(false);
const agencyModels = ref<RoutingModelOption[]>([]);
const routingPreview = ref<ModelSelectionInput | undefined>();
let catalogGeneration = 0;
let routingGeneration = 0;
let routingTimer: ReturnType<typeof setTimeout> | undefined;

watch(
  () => [props.agentId, props.config, props.useDefaults] as const,
  async ([agentId, config, useDefaults]) => {
    const gen = ++catalogGeneration;
    taskDefaults.value = undefined;
    agentDefaults.value = undefined;
    if (!agentId) {
      loadingCatalog.value = false;
      return;
    }
    loadingCatalog.value = true;
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
    if (gen !== catalogGeneration) return;
    if (catalog.status === "fulfilled") taskDefaults.value = catalog.value;
    if (defaults.status === "fulfilled") {
      agentDefaults.value = defaults.value ?? undefined;
    }
    loadingCatalog.value = false;
  },
  { immediate: true },
);

watch(
  () => [props.agentId, props.routing?.projectId ?? ""] as const,
  async ([agentId, projectId]) => {
    if (agentId && !props.routing) {
      agencyModels.value = [];
      return;
    }
    try {
      agencyModels.value = await agency.listRoutingModels(projectId);
    } catch {
      agencyModels.value = [];
    }
  },
  { immediate: true },
);

watch(
  () => {
    const routing = props.routing;
    if (!routing) return undefined;
    return [
      routing.projectId,
      routing.priority,
      routing.size,
      routing.tags,
    ] as const;
  },
  (shape) => {
    if (routingTimer) clearTimeout(routingTimer);
    if (!shape) {
      routingPreview.value = undefined;
      return;
    }
    const gen = ++routingGeneration;
    routingTimer = setTimeout(async () => {
      try {
        const preview = await agency.previewModelRouting({
          projectId: shape[0],
          priority: shape[1],
          size: shape[2],
          tags: shape[3] ?? [],
        });
        if (gen === routingGeneration) routingPreview.value = preview.model;
      } catch {
        if (gen === routingGeneration) routingPreview.value = undefined;
      }
    }, 180);
  },
  { immediate: true, deep: true },
);

const agentCatalog = computed<ModelChoice[]>(() => {
  const models: TaskModel[] =
    props.config?.models ?? taskDefaults.value?.models ?? [];
  return models.map((model) => ({
    provider: model.provider,
    id: model.id,
    name: model.name,
    reasoning: model.reasoning,
    thinkingLevels: [...model.thinkingLevels],
  }));
});

const declaredCatalog = computed<ModelChoice[]>(() =>
  agencyModels.value.map((model) => ({
    provider: model.provider,
    id: model.modelId,
    name: model.name,
    reasoning: model.thinkingLevels.length > 1,
    thinkingLevels: [...model.thinkingLevels],
  })),
);

const baseModels = computed<ModelChoice[]>(() =>
  agentCatalog.value.length ? agentCatalog.value : declaredCatalog.value,
);

function findModel(provider: string, id: string): ModelChoice | undefined {
  if (!provider || !id) return undefined;
  return baseModels.value.find(
    (model) => model.provider === provider && model.id === id,
  );
}

const defaultModel = computed<ModelChoice | undefined>(() => {
  const provider =
    agentDefaults.value?.modelProvider ||
    taskDefaults.value?.modelProvider ||
    "";
  const id = agentDefaults.value?.modelId || taskDefaults.value?.modelId || "";
  if (!provider || !id) return undefined;
  return {
    provider,
    id,
    name: agentDefaults.value?.modelName || taskDefaults.value?.modelName || id,
    reasoning: true,
    thinkingLevels: [
      ...(taskDefaults.value?.thinkingLevels ??
        props.config?.thinkingLevels ??
        []),
    ],
  };
});

const routedModel = computed<ModelChoice | undefined>(() => {
  const model = routingPreview.value;
  if (!model?.provider || !model.modelId) return undefined;
  return (
    findModel(model.provider, model.modelId) ?? {
      provider: model.provider,
      id: model.modelId,
      name: model.modelName || model.modelId,
      reasoning: true,
      thinkingLevels: [],
    }
  );
});

const pinnedModel = computed(
  () => !!props.selection.modelProvider || !!props.selection.modelId,
);
const manualModel = computed(() => props.useDefaults && pinnedModel.value);
const showingRouted = computed(() => !pinnedModel.value && !!routedModel.value);

const routedThinking = computed(() => {
  if (pinnedModel.value) return "";
  return routingPreview.value?.thinkingLevel ?? "";
});

const selectedModel = computed<ModelChoice | undefined>(() => {
  if (pinnedModel.value) {
    const found = findModel(
      props.selection.modelProvider,
      props.selection.modelId,
    );
    if (found) return found;

    return {
      provider: props.selection.modelProvider,
      id: props.selection.modelId,
      name: props.selection.modelName || props.selection.modelId,
      reasoning: true,
      thinkingLevels: [],
    };
  }
  return (
    routedModel.value ?? (props.useDefaults ? defaultModel.value : undefined)
  );
});

const modelOptions = computed<ModelChoice[]>(() => {
  const list = [...baseModels.value];
  const add = (model: ModelChoice | undefined) => {
    if (!model?.provider || !model.id) return;
    if (
      list.some(
        (item) => item.provider === model.provider && item.id === model.id,
      )
    ) {
      return;
    }
    list.unshift(model);
  };

  add(defaultModel.value);
  add(routedModel.value);
  add(selectedModel.value);
  return list;
});

const defaultThinkingLevel = computed(() => {
  if (!props.useDefaults || pinnedModel.value) return "";
  return (
    agentDefaults.value?.thinkingLevel ||
    taskDefaults.value?.thinkingLevel ||
    ""
  );
});

const thinkingOptions = computed(() =>
  sortThinkingLevels(selectedModel.value?.thinkingLevels ?? []),
);

const pinnedThinking = computed(() => !!props.selection.thinkingLevel);
const manualThinking = computed(
  () => props.useDefaults && pinnedThinking.value,
);
const showingRoutedThinking = computed(
  () => !pinnedThinking.value && !!routedThinking.value,
);

const modelChoice = computed(() => {
  const model = selectedModel.value;
  return model ? `${model.provider}/${model.id}` : "";
});
const thinkingChoice = computed(
  () =>
    props.selection.thinkingLevel ||
    routedThinking.value ||
    defaultThinkingLevel.value,
);

const autoChoice = computed(
  () => !!routedModel.value || (props.useDefaults && !!defaultModel.value),
);

const autoLabel = computed(() =>
  routedModel.value
    ? t("taskModel.automaticRouting")
    : t("taskModel.automaticDefault"),
);

const automaticThinkingLevel = computed(
  () => routedThinking.value || defaultThinkingLevel.value,
);

const showAutomaticThinking = computed(() => props.useDefaults);

const thinkingDisplayLabel = computed(
  () => thinkingChoice.value || t("taskModel.thinking"),
);

onUnmounted(() => {
  if (routingTimer) clearTimeout(routingTimer);
});

const canPickModel = computed(
  () =>
    !props.disabled && !loadingCatalog.value && modelOptions.value.length > 0,
);
const canPickThinking = computed(
  () =>
    !props.disabled &&
    !loadingCatalog.value &&
    thinkingOptions.value.length > 0,
);

function levelFor(model: ModelChoice | undefined, level: string): string {
  if (!model || !level) return level;
  return model.thinkingLevels.includes(level) ? level : "";
}

function onModelChange(value: unknown) {
  const choice = String(value);
  if (choice === "__auto__") {
    const next = routedModel.value ?? defaultModel.value;
    emit("change", {
      modelProvider: "",
      modelId: "",
      modelName: "",
      thinkingLevel: levelFor(next, props.selection.thinkingLevel),
    });
    return;
  }
  const model = modelOptions.value.find(
    (item) => `${item.provider}/${item.id}` === choice,
  );
  if (!model) return;
  const thinkingLevel = levelFor(model, props.selection.thinkingLevel);
  const isDefault =
    props.useDefaults &&
    model.provider === defaultModel.value?.provider &&
    model.id === defaultModel.value?.id;
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

  const automatic =
    level === "__auto__" || level === automaticThinkingLevel.value;
  emit("change", {
    ...props.selection,
    thinkingLevel: props.useDefaults && automatic ? "" : level,
  });
}
</script>

<i18n lang="yaml">
en:
  taskModel:
    title: "Model for this task"
    thinkingHint: "Thinking level for this task"
    thinking: "Thinking level"
    automatic: "Automatic"
    automaticRouting: "Automatic (routing)"
    automaticDefault: "Automatic (agent default)"
    routed: "Picked automatically by the model routing"
ru:
  taskModel:
    title: "Модель для этой задачи"
    thinkingHint: "Уровень размышления для этой задачи"
    thinking: "Уровень размышления"
    automatic: "Автоматически"
    automaticRouting: "Автоматически (маршрутизация)"
    automaticDefault: "Автоматически (модель агента)"
    routed: "Выбрано автоматически по маршрутизации моделей"
</i18n>
