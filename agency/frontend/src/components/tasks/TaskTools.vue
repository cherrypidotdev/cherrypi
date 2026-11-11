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
.flex.flex-col.gap-1(data-testid="task-tools")
  .flex.items-baseline.justify-between.gap-2
    span.text-muted-foreground.text-xs {{ $t("taskTools.title") }}
    span.text-muted-foreground.text-xs.tabular-nums(v-if="tools.length")
      | {{ tools.length }}
  p.text-muted-foreground.text-xs(v-if="loading") {{ $t("taskTools.loading") }}
  p.text-muted-foreground.text-xs(v-else-if="error") {{ error }}
  p.text-muted-foreground.text-xs(v-else-if="!recorded")
    | {{ $t("taskTools.notRecorded") }}
  p.text-muted-foreground.text-xs(v-else-if="!tools.length")
    | {{ $t("taskTools.empty") }}

  .flex.flex-col.overflow-hidden.rounded-md.border(v-if="tools.length")
    .flex.flex-col.border-b(
      v-for="tool in tools"
      :key="tool.name"
      class="last:border-b-0"
      data-testid="task-tool"
    )
      .flex.items-center.gap-1.pr-1
        button.flex.min-w-0.flex-1.items-center.gap-1.px-2.text-left(
          type="button"
          class="hover:bg-muted/40 py-1.5"
          :aria-expanded="isOpen(tool.name)"
          :title="isOpen(tool.name) ? $t('taskTools.hideHint') : $t('taskTools.showHint')"
          @click="toggle(tool.name)"
          data-testid="task-tool-toggle"
        )
          ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
            :class="isOpen(tool.name) ? 'rotate-90' : ''"
          )
          component.text-muted-foreground.shrink-0(
            class="size-3.5"
            :is="icon(tool.name)"
          )
          span.truncate.font-mono.text-xs {{ tool.name }}
          span.shrink-0.rounded-full.border.px-2.text-xs(
            class="py-0.5"
            :class="originClass(tool)"
            :title="originTitle(tool)"
            data-testid="task-tool-origin"
          ) {{ originLabel(tool) }}
        button.shrink-0.rounded-md.px-1.text-xs(
          type="button"
          :class="tool.callCount > 0 ? 'text-muted-foreground hover:bg-muted hover:text-foreground' : 'text-muted-foreground/50'"
          :title="tool.callCount > 0 ? $t('taskTools.filterHint') : $t('taskTools.noCallsHint')"
          @click="$emit('filter', tool.name)"
          :disabled="tool.callCount === 0"
          data-testid="task-tool-calls"
        ) {{ callsLabel(tool) }}

      .flex.flex-col.gap-2.border-t.px-2.py-2(v-if="isOpen(tool.name)")
        p.text-muted-foreground.text-xs.whitespace-pre-wrap(
          v-if="tool.description"
        )
          | {{ tool.description }}
        span.text-muted-foreground.text-xs.italic(v-else)
          | {{ $t("taskTools.noDescription") }}
        .flex.flex-col.gap-1
          span.text-muted-foreground.text-xs.font-medium {{ $t("taskTools.arguments") }}
          p.text-muted-foreground.text-xs(v-if="!args(tool).length")
            | {{ $t("taskTools.noArguments") }}
          template(v-for="arg in args(tool)" :key="arg.name")
            .flex.flex-wrap.items-baseline.gap-1.text-xs
              span.font-mono {{ arg.name }}
              span.text-muted-foreground {{ arg.type }}
              span.text-muted-foreground.text-xs(v-if="arg.required") {{ $t("taskTools.required") }}
            p.text-muted-foreground.pl-3.text-xs(v-if="arg.description")
              | {{ arg.description }}
</template>

<script setup lang="ts">
import { agency } from "@/lib/sdk";
import { TaskToolOrigin, type TaskTool } from "@cherrypi/sdk";
import {
  ChevronRightIcon,
  FilePlusIcon,
  FileTextIcon,
  PencilIcon,
  SquareTerminalIcon,
  WrenchIcon,
  type LucideIcon,
} from "@lucide/vue";
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{ taskId: string }>();
const emit = defineEmits<{
  (e: "filter", name: string): void;
}>();

interface ToolArg {
  name: string;
  type: string;
  required: boolean;
  description: string;
}

const tools = ref<TaskTool[]>([]);
const recorded = ref(false);
const loading = ref(true);
const error = ref("");

const open = reactive(new Set<string>());

const ICONS: Record<string, LucideIcon> = {
  bash: SquareTerminalIcon,
  read: FileTextIcon,
  write: FilePlusIcon,
  edit: PencilIcon,
  generic: WrenchIcon,
};

function isOpen(name: string): boolean {
  return open.has(name);
}

function toggle(name: string) {
  if (open.has(name)) open.delete(name);
  else open.add(name);
}

function icon(name: string): LucideIcon {
  return ICONS[name] ?? WrenchIcon;
}

function originLabel(tool: TaskTool): string {
  if (tool.origin !== TaskToolOrigin.PLUGIN) return t("taskTools.builtIn");
  return tool.pluginName || tool.pluginId || t("taskTools.plugin");
}

function originTitle(tool: TaskTool): string {
  if (tool.origin !== TaskToolOrigin.PLUGIN) return t("taskTools.builtInHint");
  const plugin = tool.pluginName || tool.pluginId;
  return t("taskTools.pluginHint", { plugin });
}

function originClass(tool: TaskTool): string {
  return tool.origin === TaskToolOrigin.PLUGIN
    ? "border-violet-500/40 text-violet-700 dark:text-violet-300"
    : "text-muted-foreground";
}

function callsLabel(tool: TaskTool): string {
  const count = Number(tool.callCount);
  if (count === 0) return t("taskTools.noCalls");
  return t("taskTools.callCount", { count });
}

function args(tool: TaskTool): ToolArg[] {
  let schema: unknown;
  try {
    schema = JSON.parse(tool.parametersJson || "{}");
  } catch {
    return [];
  }
  if (typeof schema !== "object" || schema === null) return [];
  const shape = schema as {
    properties?: Record<string, unknown>;
    required?: unknown;
  };
  const properties = shape.properties ?? {};
  const required = new Set(
    Array.isArray(shape.required)
      ? shape.required.filter(
          (name): name is string => typeof name === "string",
        )
      : [],
  );
  return Object.entries(properties).map(([name, value]) => {
    const property =
      typeof value === "object" && value !== null
        ? (value as { type?: unknown; description?: unknown; items?: unknown })
        : {};
    return {
      name,
      type: typeLabel(property),
      required: required.has(name),
      description:
        typeof property.description === "string" ? property.description : "",
    };
  });
}

function typeLabel(property: { type?: unknown; items?: unknown }): string {
  if (typeof property.type === "string") {
    if (property.type === "array") {
      const items = property.items;
      const item =
        typeof items === "object" && items !== null
          ? (items as { type?: unknown }).type
          : undefined;
      return typeof item === "string" ? `${item}[]` : "array";
    }
    return property.type;
  }
  if (Array.isArray(property.type)) {
    return property.type.filter((type) => typeof type === "string").join(" | ");
  }
  return "any";
}

onMounted(async () => {
  try {
    const result = await agency.listTaskTools(props.taskId);
    tools.value = result.tools;
    recorded.value = result.recorded;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
});
</script>

<i18n lang="yaml">
en:
  taskTools:
    title: "Tools"
    loading: "Loading tools…"
    notRecorded: "Tools are recorded when the task's first run starts."
    empty: "No tools were declared for this task."
    hideHint: "Hide this tool’s description and arguments"
    showHint: "Show this tool’s description and arguments"
    filterHint: "Filter the Logs to this tool’s calls"
    noCallsHint: "This tool recorded no calls"
    noDescription: "No description."
    arguments: "Arguments"
    noArguments: "None."
    required: "required"
    builtIn: "built-in"
    plugin: "plugin"
    builtInHint: "Provided by Pi or CherryPi"
    pluginHint: "Attached by the {plugin} plugin"
    noCalls: "no calls"
    callCount: "{count} call | {count} calls"
ru:
  taskTools:
    title: "Инструменты"
    loading: "Загружаем инструменты…"
    notRecorded: "Инструменты записываются при первом запуске задачи."
    empty: "Для этой задачи не объявлено ни одного инструмента."
    hideHint: "Скрыть описание и аргументы инструмента"
    showHint: "Показать описание и аргументы инструмента"
    filterHint: "Отфильтровать журнал по вызовам этого инструмента"
    noCallsHint: "Этот инструмент не записал ни одного вызова"
    noDescription: "Описания нет."
    arguments: "Аргументы"
    noArguments: "Нет."
    required: "обязательный"
    builtIn: "встроенный"
    plugin: "плагин"
    builtInHint: "Предоставлено Pi или CherryPi"
    pluginHint: "Добавлено плагином {plugin}"
    noCalls: "нет вызовов"
    callCount: "{count} вызов | {count} вызова | {count} вызовов"
</i18n>
