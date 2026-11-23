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
.flex.flex-col.gap-2(data-testid="task-system")
  .flex.flex-wrap.items-center.gap-2
    span.text-muted-foreground.text-xs {{ $t("taskSystem.view") }}
    .bg-muted.inline-flex.items-center.gap-1.rounded-lg.border(class="p-0.5")
      button.rounded-md.px-2.text-xs.font-medium.transition-colors(
        v-for="option in VIEWS"
        :key="option.value"
        type="button"
        class="py-0.5"
        :class="view === option.value ? 'bg-card shadow-sm ring-1 ring-border' : 'text-muted-foreground hover:text-foreground'"
        @click="view = option.value"
        :data-testid="`task-system-view-${option.value}`"
      ) {{ $t(option.labelKey) }}
    span.text-muted-foreground.ml-auto.text-xs.tabular-nums
      | {{ $t("taskSystem.counts", { prompts: prompts.length, tools: tools.length }) }}

  p.text-muted-foreground.text-xs(v-if="loading") {{ $t("taskTools.loading") }}
  p.text-muted-foreground.text-xs(v-else-if="error") {{ error }}

  template(v-if="view === 'kind'")
    Disclosure(v-slot="{ open }" :default-open="true")
      DisclosureButton.group.flex.w-full.items-center.gap-1.text-left
        ChevronRightIcon.size-3.shrink-0.transition-transform(
          :class="open ? 'rotate-90' : ''"
        )
        span.text-muted-foreground.text-xs.font-medium {{ $t("taskSystem.prompts") }}
        span.text-muted-foreground.text-xs.tabular-nums {{ prompts.length }}
      DisclosurePanel.mt-2
        TaskPromptList(:sections="prompts")

    TaskIntegrations(:task-id="taskId" :preview="integrationsPreview")

    Disclosure(v-slot="{ open }")
      DisclosureButton.group.flex.w-full.items-center.gap-1.text-left
        ChevronRightIcon.size-3.shrink-0.transition-transform(
          :class="open ? 'rotate-90' : ''"
        )
        span.text-muted-foreground.text-xs.font-medium {{ $t("taskTools.title") }}
        span.text-muted-foreground.text-xs.tabular-nums {{ tools.length }}
      DisclosurePanel.mt-2.flex.flex-col.gap-2
        p.text-muted-foreground.text-xs(v-if="!recorded") {{ $t("taskTools.notRecorded") }}
        p.text-muted-foreground.text-xs(v-else-if="!tools.length") {{ $t("taskTools.empty") }}
        TaskToolList(v-else :tools="tools" @filter="$emit('filter', $event)")

  template(v-else)
    p.text-muted-foreground.text-xs(v-if="!recorded") {{ $t("taskTools.notRecorded") }}
    p.text-muted-foreground.text-xs(v-else-if="!tools.length") {{ $t("taskTools.empty") }}
    Disclosure.flex.flex-col.gap-2(
      v-for="origin in origins"
      :key="origin.key"
      as="div"
      v-slot="{ open }"
      :default-open="true"
    )
      DisclosureButton.flex.items-center.gap-1.text-left
        ChevronRightIcon.size-3.shrink-0.transition-transform(
          :class="open ? 'rotate-90' : ''"
        )
        span.text-sm.font-medium {{ origin.label }}
        span.text-muted-foreground.text-xs.tabular-nums(:title="origin.title") {{ origin.prompts.length + origin.tools.length }}
      DisclosurePanel.flex.flex-col.gap-2
        Disclosure(
          v-if="origin.prompts.length"
          v-slot="{ open: promptOpen }"
          :default-open="true"
        )
          DisclosureButton.flex.items-center.gap-1.text-left
            ChevronRightIcon.size-3.shrink-0.transition-transform(
              :class="promptOpen ? 'rotate-90' : ''"
            )
            span.text-muted-foreground.text-xs.font-medium {{ $t("taskSystem.prompts") }}
          DisclosurePanel.mt-1
            TaskPromptList(:sections="origin.prompts")

        Disclosure(v-if="origin.tools.length" v-slot="{ open: toolOpen }")
          DisclosureButton.flex.items-center.gap-1.text-left
            ChevronRightIcon.size-3.shrink-0.transition-transform(
              :class="toolOpen ? 'rotate-90' : ''"
            )
            span.text-muted-foreground.text-xs.font-medium {{ $t("taskTools.title") }}
          DisclosurePanel.mt-1
            TaskToolList(
              :tools="origin.tools"
              :group-by-origin="false"
              @filter="$emit('filter', $event)"
            )
</template>

<script setup lang="ts">
import TaskIntegrations from "@/components/tasks/TaskIntegrations.vue";
import TaskPromptList from "@/components/tasks/TaskPromptList.vue";
import TaskToolList from "@/components/tasks/TaskToolList.vue";
import type { TaskIntegrationReport } from "@/lib/integrations";
import { agency } from "@/lib/sdk";
import {
  PromptScope,
  TaskToolOrigin,
  type SystemPromptSection,
  type TaskTool,
} from "@cherrypi/sdk";
import { Disclosure, DisclosureButton, DisclosurePanel } from "@headlessui/vue";
import { ChevronRightIcon } from "@lucide/vue";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  taskId: string;
  sections: SystemPromptSection[];

  preview?: boolean | { tools: TaskTool[]; recorded: boolean };

  integrationsPreview?: TaskIntegrationReport;
}>();

defineEmits<{
  (e: "filter", name: string): void;
}>();

const VIEWS = [
  { value: "kind", labelKey: "taskSystem.byKind" },
  { value: "origin", labelKey: "taskSystem.byOrigin" },
] as const;

type View = (typeof VIEWS)[number]["value"];
const view = ref<View>("kind");

const tools = ref<TaskTool[]>([]);
const recorded = ref(false);
const loading = ref(true);
const error = ref("");

const prompts = computed(() =>
  props.sections.filter((section) => section.content.trim() !== ""),
);

interface Origin {
  key: string;
  label: string;
  title: string;
  prompts: SystemPromptSection[];
  tools: TaskTool[];
}

const origins = computed<Origin[]>(() => {
  const builtIn: Origin = {
    key: "builtin",
    label: t("taskSystem.builtInOrigin"),
    title: t("taskTools.builtInHint"),
    prompts: prompts.value.filter(
      (section) => section.scope === PromptScope.BUILTIN,
    ),
    tools: tools.value.filter((tool) => tool.origin !== TaskToolOrigin.PLUGIN),
  };
  const agency: Origin = {
    key: "agency",
    label: t("taskSystem.agencyOrigin"),
    title: t("taskSystem.agencyOriginHint"),
    prompts: prompts.value.filter(
      (section) => section.scope !== PromptScope.BUILTIN,
    ),
    tools: [],
  };
  const out = [builtIn, agency].filter(
    (origin) => origin.prompts.length || origin.tools.length,
  );
  const plugins = new Map<string, Origin>();
  for (const tool of tools.value) {
    if (tool.origin !== TaskToolOrigin.PLUGIN) continue;
    const id = tool.pluginId || tool.pluginName || "plugin";
    let origin = plugins.get(id);
    if (!origin) {
      const label = tool.pluginName || tool.pluginId || t("taskTools.plugin");
      origin = {
        key: `plugin:${id}`,
        label,
        title: t("taskToolPluginOriginHint", { plugin: label }),
        prompts: [],
        tools: [],
      };
      plugins.set(id, origin);
    }
    origin.tools.push(tool);
  }
  out.push(
    ...[...plugins.values()].sort((a, b) => a.label.localeCompare(b.label)),
  );
  return out;
});

onMounted(async () => {
  if (props.preview) {
    if (typeof props.preview === "object") {
      tools.value = props.preview.tools;
      recorded.value = props.preview.recorded;
    }
    loading.value = false;
    return;
  }
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
  taskSystem:
    view: "Group by"
    byKind: "Prompts / Tools"
    byOrigin: "Origins"
    prompts: "Prompts"
    counts: "{prompts} prompts · {tools} tools"
    builtInOrigin: "Built-in"
    agencyOrigin: "Agency"
    agencyOriginHint: "Organization, project, agent, team, and group prompts"
  taskToolPluginOriginHint: "Provided by the {plugin} plugin"
ru:
  taskTools:
    title: "Инструменты"
    loading: "Загружаем инструменты…"
    notRecorded: "Инструменты записываются при первом запуске задачи."
    empty: "Для этой задачи не объявлено ни одного инструмента."
  taskSystem:
    view: "Группировать"
    byKind: "Промпты / Инструменты"
    byOrigin: "Источники"
    prompts: "Промпты"
    counts: "{prompts} промптов · {tools} инструментов"
    builtInOrigin: "Встроенное"
    agencyOrigin: "Агентство"
    agencyOriginHint: "Промпты организации, проекта, агента, команды и группы"
  taskToolPluginOriginHint: "Предоставлено плагином {plugin}"
</i18n>
