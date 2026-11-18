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
Card
  CardHeader
    CardTitle {{ title }}
    CardDescription {{ description }}
  CardContent.flex.flex-col.gap-5
    .flex.items-center.justify-center.py-6(v-if="loading")
      Spinner
    template(v-else)
      fieldset.flex.flex-col.gap-5(:disabled="!canEdit")
        p.text-muted-foreground.text-xs(v-if="!canEdit")
          | {{ $t("modelRouting.readOnly") }}
        p.text-muted-foreground.text-xs
          | {{ $t("modelRouting.catalogHint") }}

        Separator

        .flex.flex-col.gap-2
          .flex.flex-col
            span.text-sm.font-medium {{ $t("modelRouting.default.title") }}
            span.text-muted-foreground.text-xs {{ $t("modelRouting.default.description") }}
          .flex.flex-wrap.items-center.gap-2(v-if="scope === 'project'")
            .flex.items-center.gap-2.rounded-md.border.px-3.py-2
              span.text-xs.font-medium {{ $t("modelRouting.default.inherit") }}
              Switch(
                :model-value="inheritDefault"
                :aria-label="$t('modelRouting.default.inherit')"
                @update:model-value="setInheritDefault"
                :disabled="!canEdit"
              )
            span.text-muted-foreground.text-xs(v-if="inheritDefault")
              | {{ inheritLabel }}
          ModelRoutingPicker(
            v-if="scope === 'org' || !inheritDefault"
            :placeholder="$t('modelRouting.default.placeholder')"
            :model-value="defaultSelection"
            :models="models"
            @update:model-value="defaultSelection = $event"
          )

        template(v-if="scope === 'project'")
          Separator
          .flex.items-center.justify-between.gap-2
            .flex.flex-col
              span.text-sm.font-medium {{ $t("modelRouting.pins.title") }}
              span.text-muted-foreground.text-xs {{ $t("modelRouting.pins.description") }}
            Button(
              type="button"
              variant="outline"
              size="sm"
              @click="addPin"
              :disabled="!canEdit"
            )
              PlusIcon(data-icon="inline-start")
              span {{ $t("modelRouting.pins.add") }}
          .flex.flex-col.gap-2(v-if="tagPins.length")
            .flex.flex-wrap.items-center.gap-2(
              v-for="(pin, index) in tagPins"
              :key="`pin-${index}`"
            )
              Input.w-40(
                :placeholder="$t('modelRouting.pins.tag')"
                :model-value="pin.tag ?? ''"
                @update:model-value="pin.tag = String($event)"
              )
              ModelRoutingPicker(
                :placeholder="$t('modelRouting.chooseModel')"
                :model-value="pin.model"
                :models="models"
                @update:model-value="pin.model = $event"
              )
              Button(
                type="button"
                variant="ghost"
                size="icon-sm"
                :aria-label="$t('modelRouting.deleteRule')"
                @click="removePin(index)"
                :disabled="!canEdit"
              )
                Trash2Icon
          p.text-muted-foreground.text-xs(v-else)
            | {{ $t("modelRouting.pins.empty") }}

        Separator

        .flex.items-center.justify-between.gap-2
          .flex.flex-col
            span.text-sm.font-medium {{ $t("modelRouting.rules.title") }}
            span.text-muted-foreground.text-xs {{ $t("modelRouting.rules.description") }}
          Button(
            type="button"
            variant="outline"
            size="sm"
            @click="addRule"
            :disabled="!canEdit"
          )
            PlusIcon(data-icon="inline-start")
            span {{ $t("modelRouting.rules.add") }}
        .flex.flex-col.gap-2(v-if="routes.length")
          .flex.flex-wrap.items-center.gap-2.rounded-md.border.p-2(
            v-for="(route, index) in routes"
            :key="`route-${index}`"
            :class="dropIndex === index ? 'border-ring' : ''"
            @dragover.prevent="onDragOver(index)"
            @drop.prevent="onDrop(index)"
          )
            span.text-muted-foreground.cursor-grab(
              :draggable="canEdit"
              :aria-label="$t('modelRouting.rules.drag')"
              @dragstart="onDragStart(index)"
              @dragend="onDragEnd"
            )
              GripVerticalIcon.size-4
            Select(
              :model-value="String(route.priority ?? 0)"
              @update:model-value="route.priority = priorityFromInput($event)"
              :disabled="!canEdit"
            )
              SelectTrigger.w-32(
                :aria-label="$t('modelRouting.rules.priority')"
              )
                SelectValue
              SelectContent
                SelectItem(value="0") {{ $t("modelRouting.anyPriority") }}
                SelectItem(
                  v-for="priority in PRIORITIES"
                  :key="priority.value"
                  :value="String(priority.value)"
                )
                  | {{ $t(priority.key) }}
            Select(
              :model-value="String(route.size ?? 0)"
              @update:model-value="route.size = sizeFromInput($event)"
              :disabled="!canEdit"
            )
              SelectTrigger.w-32(:aria-label="$t('modelRouting.rules.size')")
                SelectValue
              SelectContent
                SelectItem(value="0") {{ $t("modelRouting.anySize") }}
                SelectItem(
                  v-for="size in SIZES"
                  :key="size.value"
                  :value="String(size.value)"
                )
                  | {{ $t(size.key) }}
            Input.w-36(
              v-if="scope === 'project'"
              :placeholder="$t('modelRouting.rules.tagPlaceholder')"
              :model-value="route.tag ?? ''"
              @update:model-value="route.tag = String($event)"
            )
            ModelRoutingPicker(
              :placeholder="$t('modelRouting.chooseModel')"
              :model-value="route.model"
              :models="models"
              @update:model-value="route.model = $event"
            )
            Button(
              type="button"
              variant="ghost"
              size="icon-sm"
              :aria-label="$t('modelRouting.deleteRule')"
              @click="removeRule(index)"
              :disabled="!canEdit"
            )
              Trash2Icon
        p.text-muted-foreground.text-xs(v-else)
          | {{ $t("modelRouting.rules.empty") }}

        template(v-if="scope === 'project'")
          Separator
          .flex.flex-col.gap-1
            span.text-sm.font-medium {{ $t("modelRouting.inherited.title") }}
            span.text-muted-foreground.text-xs {{ $t("modelRouting.inherited.description") }}
          .flex.flex-col.gap-1.text-xs(v-if="orgModelHints.length")
            .flex.items-center.justify-between.gap-2.rounded-md.border.px-2.py-1(
              v-for="hint in orgModelHints"
              :key="hint"
            )
              span.truncate {{ hint }}
              span.text-muted-foreground.rounded-full.border.px-2(
                class="py-0.5 text-[10px]"
              )
              | {{ $t("modelRouting.inherited.org") }}
          p.text-muted-foreground.text-xs(v-else)
            | {{ $t("modelRouting.inherited.empty") }}

        p.text-destructive.text-sm(v-if="error") {{ error }}
        p.text-muted-foreground.text-sm(v-else-if="saved") {{ $t("modelRouting.saved") }}
        .flex.justify-end
          Button(
            type="button"
            @click="save"
            :disabled="saving || loading || !canEdit"
          )
            Spinner(v-if="saving" data-icon="inline-start")
            CheckIcon(v-else data-icon="inline-start")
            span {{ $t("common.save") }}
</template>

<script setup lang="ts">
import ModelRoutingPicker from "@/components/settings/ModelRoutingPicker.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { modelLabel } from "@/lib/model";
import { agency } from "@/lib/sdk";
import { taskPriorityKey, taskSizeKey } from "@/lib/task";
import {
  TaskPriority,
  TaskSize,
  type ModelRoute,
  type ModelRouteInput,
  type ModelSelection,
  type ModelSelectionInput,
  type OrganizationModelRouting,
  type RoutingModelOption,
} from "@cherrypi/sdk";
import { CheckIcon, GripVerticalIcon, PlusIcon, Trash2Icon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  scope: "org" | "project";
  projectId?: string;
  canEdit?: boolean;
}>();

const loading = ref(true);
const saving = ref(false);
const saved = ref(false);
const error = ref("");
const models = ref<RoutingModelOption[]>([]);
const defaultSelection = ref<ModelSelectionInput>(emptySelection());
const inheritDefault = ref(true);
const routes = ref<ModelRouteInput[]>([]);
const tagPins = ref<ModelRouteInput[]>([]);
const orgRouting = ref<OrganizationModelRouting | undefined>();

const title = computed(() =>
  props.scope === "org"
    ? t("modelRouting.title.org")
    : t("modelRouting.title.project"),
);
const description = computed(() =>
  props.scope === "org"
    ? t("modelRouting.description.org")
    : t("modelRouting.description.project"),
);
const canEdit = computed(() => props.canEdit !== false);
const inheritLabel = computed(() => {
  const model = orgRouting.value?.defaultModel;
  if (!model?.provider || !model.modelId) {
    return t("modelRouting.default.inheritEmpty");
  }
  return t("modelRouting.default.inheritValue", {
    model: modelLabel(model),
  });
});
const orgModelHints = computed(() => {
  const hints: string[] = [];
  const org = orgRouting.value;
  if (!org) return hints;
  if (org.defaultModel?.provider && org.defaultModel.modelId) {
    hints.push(
      t("modelRouting.inherited.default", {
        model: modelLabel(org.defaultModel),
      }),
    );
  }
  for (const route of org.routes) {
    hints.push(
      routeHint(routeFromProto(route), t("modelRouting.inherited.rule")),
    );
  }
  return hints;
});

const PRIORITIES = [
  { value: TaskPriority.CRITICAL, key: taskPriorityKey(TaskPriority.CRITICAL) },
  { value: TaskPriority.HIGH, key: taskPriorityKey(TaskPriority.HIGH) },
  { value: TaskPriority.NORMAL, key: taskPriorityKey(TaskPriority.NORMAL) },
  { value: TaskPriority.LOW, key: taskPriorityKey(TaskPriority.LOW) },
];
const SIZES = [
  { value: TaskSize.SMALL, key: taskSizeKey(TaskSize.SMALL) },
  { value: TaskSize.MEDIUM, key: taskSizeKey(TaskSize.MEDIUM) },
  { value: TaskSize.BIG, key: taskSizeKey(TaskSize.BIG) },
  { value: TaskSize.EPIC, key: taskSizeKey(TaskSize.EPIC) },
];

function emptySelection(): ModelSelectionInput {
  return { provider: "", modelId: "", modelName: "", thinkingLevel: "" };
}

function selectionFromProto(value?: ModelSelection): ModelSelectionInput {
  if (!value) return emptySelection();
  return {
    provider: value.provider,
    modelId: value.modelId,
    modelName: value.modelName,
    thinkingLevel: value.thinkingLevel,
  };
}

function routeFromProto(value: ModelRoute): ModelRouteInput {
  return {
    priority: value.priority,
    size: value.size,
    tag: value.tag,
    model: selectionFromProto(value.model),
  };
}

function selectionIsSet(value: ModelSelectionInput): boolean {
  return Boolean(value.provider && value.modelId);
}

function routeHint(route: ModelRouteInput, prefix: string): string {
  const conditions: string[] = [];
  if (route.priority) {
    const key = PRIORITIES.find((entry) => entry.value === route.priority)?.key;
    conditions.push(key ? t(key) : String(route.priority));
  }
  if (route.size) {
    const key = SIZES.find((entry) => entry.value === route.size)?.key;
    conditions.push(key ? t(key) : String(route.size));
  }
  if (route.tag) conditions.push(`#${route.tag}`);
  return `${prefix} ${conditions.join(" + ") || "—"} → ${modelLabel(route.model)}`;
}

function priorityFromInput(value: unknown): TaskPriority {
  const parsed = Number(value);
  return Number.isFinite(parsed)
    ? (parsed as TaskPriority)
    : TaskPriority.UNSPECIFIED;
}

function sizeFromInput(value: unknown): TaskSize {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? (parsed as TaskSize) : TaskSize.UNSPECIFIED;
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const [org, routingModels] = await Promise.all([
      agency.getOrganizationModelRouting(),
      agency.listRoutingModels(
        props.scope === "project" ? (props.projectId ?? "") : "",
      ),
    ]);
    orgRouting.value = org;
    models.value = routingModels;
    if (props.scope === "org") {
      defaultSelection.value = selectionFromProto(org.defaultModel);
      routes.value = org.routes.map(routeFromProto);
    } else if (props.projectId) {
      const project = await agency.getProjectModelRouting(props.projectId);
      inheritDefault.value = !project.defaultModel;
      defaultSelection.value = selectionFromProto(project.defaultModel);
      tagPins.value = project.tagPins.map(routeFromProto);
      routes.value = project.routes.map(routeFromProto);
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function save() {
  if (!canEdit.value) return;
  if (props.scope === "project" && !props.projectId) return;
  saving.value = true;
  error.value = "";
  saved.value = false;
  try {
    if (props.scope === "org") {
      const updated = await agency.updateOrganizationModelRouting({
        defaultModel: selectionIsSet(defaultSelection.value)
          ? defaultSelection.value
          : undefined,
        routes: routes.value,
      });
      orgRouting.value = updated;
      defaultSelection.value = selectionFromProto(updated.defaultModel);
      routes.value = updated.routes.map(routeFromProto);
    } else if (props.projectId) {
      const updated = await agency.updateProjectModelRouting(props.projectId, {
        defaultModel:
          !inheritDefault.value && selectionIsSet(defaultSelection.value)
            ? defaultSelection.value
            : undefined,
        tagPins: tagPins.value,
        routes: routes.value,
      });
      inheritDefault.value = !updated.defaultModel;
      defaultSelection.value = selectionFromProto(updated.defaultModel);
      tagPins.value = updated.tagPins.map(routeFromProto);
      routes.value = updated.routes.map(routeFromProto);
    }
    saved.value = true;
    window.setTimeout(() => (saved.value = false), 2500);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

function setInheritDefault(value: boolean) {
  inheritDefault.value = value;
  if (value) defaultSelection.value = emptySelection();
}

function addRule() {
  routes.value.push({
    priority: TaskPriority.UNSPECIFIED,
    size: TaskSize.UNSPECIFIED,
    tag: "",
    model: emptySelection(),
  });
}

function removeRule(index: number) {
  routes.value.splice(index, 1);
}

function addPin() {
  tagPins.value.push({
    priority: TaskPriority.UNSPECIFIED,
    size: TaskSize.UNSPECIFIED,
    tag: "",
    model: emptySelection(),
  });
}

function removePin(index: number) {
  tagPins.value.splice(index, 1);
}

const dragIndex = ref<number | null>(null);
const dropIndex = ref<number | null>(null);

function onDragStart(index: number) {
  dragIndex.value = index;
}

function onDragOver(index: number) {
  dropIndex.value = index;
}

function onDrop(index: number) {
  if (dragIndex.value !== null && dragIndex.value !== index) {
    const [moved] = routes.value.splice(dragIndex.value, 1);
    if (moved) routes.value.splice(index, 0, moved);
  }
  onDragEnd();
}

function onDragEnd() {
  dragIndex.value = null;
  dropIndex.value = null;
}

onMounted(load);

watch(
  () => props.projectId,
  (id, previous) => {
    if (props.scope !== "project" || !id || id === previous) return;
    void load();
  },
);
</script>

<i18n lang="yaml">
en:
  modelRouting:
    title:
      org: "Model routing"
      project: "Model routing overrides"
    description:
      org: "Choose the models tasks start with. Rules match a priority, size or tag; the first match wins."
      project: "Override the organization routing for this project. Empty sections inherit."
    catalogHint: "Models come from the agency catalog on this Models tab; only granted, enabled models appear here."
    default:
      title: "Default model"
      description: "Used when no rule matches. Empty falls back to the assigned agent."
      inherit: "Inherit organization"
      inheritEmpty: "Inheriting the agent default."
      inheritValue: "Inheriting {model}."
      placeholder: "Use the agent default"
    pins:
      title: "Tag pins"
      description: "A tag pin wins before the project's other rules, in list order."
      add: "Pin tag"
      tag: "Tag name"
      empty: "No tag pins."
    rules:
      title: "Rules"
      description: "First match wins. Drag a row to reorder it."
      add: "Add rule"
      priority: "Priority"
      size: "Size"
      tagPlaceholder: "Tag (optional)"
      drag: "Drag to reorder"
      empty: "No rules yet."
    inherited:
      title: "Inherited from organization"
      description: "These still apply after this project's rules and default."
      org: "Org"
      default: "Default → {model}"
      rule: "Rule"
      empty: "The organization has no routing configured."
    chooseModel: "Choose a model"
    anyPriority: "Any priority"
    anySize: "Any size"
    deleteRule: "Delete rule"
    readOnly: "Only a project owner or an administrator may change model routing."
    saved: "Saved."
ru:
  modelRouting:
    title:
      org: "Маршрутизация моделей"
      project: "Переопределения маршрутизации"
    description:
      org: "Выберите модели, с которых начинаются задачи. Правила сопоставляют приоритет, размер или тег; срабатывает первое совпадение."
      project: "Переопределите маршрутизацию организации для этого проекта. Пустые разделы наследуются."
    catalogHint: "Модели берутся из каталога агентства на этой вкладке «Модели»; здесь видны только включённые модели с подходящим грантом."
    default:
      title: "Модель по умолчанию"
      description: "Используется, когда ни одно правило не совпало. Пустое значение — модель агента."
      inherit: "Наследовать организацию"
      inheritEmpty: "Наследуется модель агента."
      inheritValue: "Наследуется {model}."
      placeholder: "Модель агента"
    pins:
      title: "Закрепления тегов"
      description: "Закрепление тега срабатывает раньше остальных правил проекта, по порядку."
      add: "Закрепить тег"
      tag: "Имя тега"
      empty: "Закреплений нет."
    rules:
      title: "Правила"
      description: "Срабатывает первое совпадение. Перетащите строку, чтобы изменить порядок."
      add: "Добавить правило"
      priority: "Приоритет"
      size: "Размер"
      tagPlaceholder: "Тег (необязательно)"
      drag: "Перетащите для изменения порядка"
      empty: "Правил пока нет."
    inherited:
      title: "Наследовано от организации"
      description: "Применяется после правил и модели по умолчанию этого проекта."
      org: "Орг."
      default: "По умолчанию → {model}"
      rule: "Правило"
      empty: "В организации маршрутизация не настроена."
    chooseModel: "Выберите модель"
    anyPriority: "Любой приоритет"
    anySize: "Любой размер"
    deleteRule: "Удалить правило"
    readOnly: "Менять маршрутизацию может только владелец проекта или администратор."
    saved: "Сохранено."
</i18n>
