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
.flex.flex-col.gap-4
  Card
    CardHeader
      CardTitle {{ $t("projectPlugins.title") }}
      CardDescription
        | {{ $t("projectPlugins.description") }}
    CardContent.flex.flex-col.gap-2
      p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
      p.text-muted-foreground.text-sm(v-if="loading") {{ $t("common.loading") }}
      p.text-muted-foreground.text-sm(v-else-if="!rows.length")
        | {{ $t("projectPlugins.empty") }}

  Card(v-for="row in rows" :key="row.plugin?.id")
    CardHeader.flex.flex-row.items-start.justify-between.gap-4
      .flex.flex-col.gap-1
        CardTitle.flex.flex-wrap.items-center.gap-2
          PlugIcon.size-4
          | {{ row.plugin?.name }}
          Badge(v-if="row.member" variant="outline") {{ $t("projectPlugins.hasAccess") }}
          Badge(v-else variant="secondary") {{ $t("projectPlugins.noAccess") }}
          Badge(v-if="row.enabled" variant="default") {{ $t("agentPlugins.enabled") }}
          Badge(v-else variant="secondary") {{ $t("settings.auth.disabled") }}
        CardDescription {{ row.plugin?.description || "No description" }}
      .flex.items-center.gap-2
        Button(
          size="sm"
          :variant="row.enabled ? 'outline' : 'default'"
          @click="setEnabled(row, !row.enabled)"
          :disabled="busy === row.plugin?.id"
        ) {{ row.enabled ? "Disable" : "Enable" }}

    CardContent.flex.flex-col.gap-5
      .grid.gap-4(class="md:grid-cols-2")
        .flex.items-center.justify-between.gap-3.rounded-md.border.p-3
          .flex.flex-col
            span.text-sm.font-medium {{ $t("projectPlugins.allowTaskCreation") }}
            span.text-muted-foreground.text-xs
              | {{ $t("projectPlugins.allowTaskCreationHint") }}
          Switch(
            :model-value="row.allowTaskCreation"
            @update:model-value="setAllowTaskCreation(row, Boolean($event))"
            :disabled="busy === row.plugin?.id"
          )
        .flex.items-center.justify-between.gap-3.rounded-md.border.p-3
          .flex.flex-col
            span.text-sm.font-medium {{ $t("projectPlugins.maxPriority") }}
            span.text-muted-foreground.text-xs
              | {{ $t("projectPlugins.maxPriorityHint") }}
          Select(
            :model-value="String(row.maxPriority)"
            @update:model-value="setMaxPriority(row, Number($event))"
            :disabled="busy === row.plugin?.id"
          )
            SelectTrigger.w-36
              SelectValue
            SelectContent
              SelectItem(value="-1") {{ $t("projectPlugins.priorityLow") }}
              SelectItem(value="0") {{ $t("projectPlugins.priorityMedium") }}
              SelectItem(value="1") {{ $t("projectPlugins.priorityHigh") }}
              SelectItem(value="2") {{ $t("projectPlugins.priorityCritical") }}

      template(v-if="hasSchema(row)")
        Separator
        .flex.flex-col.gap-1
          span.text-sm.font-medium {{ $t("projectPlugins.settingsTitle") }}
          span.text-muted-foreground.text-xs
            | {{ $t("projectPlugins.settingsHint") }}
        PluginSettingsForm(
          scope="project"
          :schema="row.schemaJson"
          :settings="settingsFor(row)"
          :secret-keys="row.secretKeys"
          :saving="saving === row.plugin?.id"
          :error="rowError[row.plugin?.id || '']"
          :saved="saved === row.plugin?.id"
          :errors="rowFieldErrors[row.plugin?.id || ''] || []"
          :plugin-id="row.plugin?.id || ''"
          :target-id="projectId"
          @save="saveSettings(row, $event)"
        )
      p.text-muted-foreground.text-xs(v-else)
        | {{ $t("projectPlugins.noSettingsForm") }}

      template(v-if="row.plugin")
        Separator
        .flex.flex-col.gap-1
          span.text-sm.font-medium {{ $t("agentPlugins.runtimeLogs") }}
          span.text-muted-foreground.text-xs
            | {{ $t("projectPlugins.runtimeLogsHint") }}
        PluginRuntimeLogs(
          scope="project"
          :plugin-id="row.plugin.id"
          :target-id="projectId"
        )
</template>

<script setup lang="ts">
import PluginRuntimeLogs from "@/components/plugins/PluginRuntimeLogs.vue";
import PluginSettingsForm from "@/components/plugins/PluginSettingsForm.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import {
  parseJSONObject,
  pluginSchemaForm,
  settingErrorsFrom,
} from "@/lib/plugins";
import { agency } from "@/lib/sdk";
import type { ProjectPlugin } from "@cherrypi/sdk";
import { PlugIcon } from "@lucide/vue";
import { onMounted, reactive, ref, watch } from "vue";

const props = defineProps<{ projectId: string }>();

const rows = ref<ProjectPlugin[]>([]);
const loading = ref(false);
const busy = ref("");
const saving = ref("");
const saved = ref("");
const error = ref("");
const rowError = reactive<Record<string, string>>({});
const rowFieldErrors = reactive<
  Record<string, Array<{ path: string; errorText: string }>>
>({});

onMounted(() => void load());
watch(
  () => props.projectId,
  () => void load(),
);

async function load() {
  if (!props.projectId) return;
  loading.value = true;
  error.value = "";
  try {
    const result = await agency.listSurfacePlugins("project", props.projectId);
    rows.value = result.projectPlugins;
  } catch (err) {
    error.value = message(err);
  } finally {
    loading.value = false;
  }
}

function settingsFor(row: ProjectPlugin): Record<string, unknown> {
  return parseJSONObject(row.settingsJson);
}

function hasSchema(row: ProjectPlugin): boolean {
  return pluginSchemaForm(row.schemaJson) !== "none";
}

async function setEnabled(row: ProjectPlugin, enabled: boolean) {
  if (
    !enabled &&
    !window.confirm(
      `Disable ${row.plugin?.name}? It loses access to this project; its settings and logs stay.`,
    )
  ) {
    return;
  }
  await update(row, { enabled });
}

async function setAllowTaskCreation(row: ProjectPlugin, allow: boolean) {
  await update(row, { allowTaskCreation: allow });
}

async function setMaxPriority(row: ProjectPlugin, maxPriority: number) {
  await update(row, { maxPriority });
}

async function update(
  row: ProjectPlugin,
  patch: Partial<{
    enabled: boolean;
    allowTaskCreation: boolean;
    maxPriority: number;
  }>,
) {
  if (!row.plugin) return;
  busy.value = row.plugin.id;
  error.value = "";
  try {
    await agency.updateProjectPlugin(row.plugin.id, props.projectId, {
      enabled: row.enabled,
      allowTaskCreation: row.allowTaskCreation,
      maxPriority: row.maxPriority,
      ...patch,
    });
    await load();
  } catch (err) {
    error.value = message(err);
  } finally {
    busy.value = "";
  }
}

async function saveSettings(
  row: ProjectPlugin,
  values: Record<string, unknown>,
) {
  if (!row.plugin) return;
  saving.value = row.plugin.id;
  saved.value = "";
  rowError[row.plugin.id] = "";
  rowFieldErrors[row.plugin.id] = [];
  try {
    await agency.updateProjectPlugin(row.plugin.id, props.projectId, {
      enabled: row.enabled,
      allowTaskCreation: row.allowTaskCreation,
      maxPriority: row.maxPriority,
      settingsJson: JSON.stringify(values),
    });
    saved.value = row.plugin.id;
    await load();
  } catch (err) {
    const fieldErrors = settingErrorsFrom(err);
    if (fieldErrors.length) {
      rowFieldErrors[row.plugin.id] = fieldErrors;
    } else {
      rowError[row.plugin.id] = message(err);
    }
  } finally {
    saving.value = "";
  }
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
</script>

<i18n lang="yaml">
en:
  projectPlugins:
    title: "Project plugins"
    description: "A plugin works in this project only while one of its owners has it enabled here; disabling it removes its access and keeps its settings and logs. These switches also cap what a plugin may create: whether it may create tasks at all, and the highest priority it may request. They stay editable while the plugin is disabled."
    empty: "No plugin declares a project settings surface."
    hasAccess: "has access"
    noAccess: "no access"
    allowTaskCreation: "Allow task creation"
    allowTaskCreationHint: "When off, create_task fails even if the plugin is enabled."
    maxPriority: "Maximum task priority"
    maxPriorityHint: "Plugin-created tasks may not exceed this priority."
    priorityLow: "Low"
    priorityMedium: "Medium"
    priorityHigh: "High"
    priorityCritical: "Critical"
    settingsTitle: "Plugin settings"
    settingsHint: 'Project-scoped values the plugin reads through settings.get("project", id).'
    noSettingsForm: "This plugin declares no project settings form."
    runtimeLogsHint: "Lines this plugin emitted for this project through cherrypi.log.*."
ru:
  projectPlugins:
    title: "Плагины проекта"
    description: "Плагин работает в этом проекте, только пока один из его владельцев включил его здесь; отключение лишает доступа, но сохраняет настройки и журналы. Эти переключатели также ограничивают возможности плагина: может ли он создавать задачи и какой максимальный приоритет запрашивать. Они остаются доступными, пока плагин отключён."
    empty: "Ни один плагин не объявляет поверхность настроек проекта."
    hasAccess: "есть доступ"
    noAccess: "нет доступа"
    allowTaskCreation: "Разрешить создание задач"
    allowTaskCreationHint: "Если выключено, create_task завершится ошибкой даже при включённом плагине."
    maxPriority: "Максимальный приоритет задачи"
    maxPriorityHint: "Созданные плагином задачи не могут превышать этот приоритет."
    priorityLow: "Низкий"
    priorityMedium: "Средний"
    priorityHigh: "Высокий"
    priorityCritical: "Критический"
    settingsTitle: "Настройки плагина"
    settingsHint: 'Значения уровня проекта, которые плагин читает через settings.get("project", id).'
    noSettingsForm: "Этот плагин не объявляет форму настроек проекта."
    runtimeLogsHint: "Строки, которые плагин записал для этого проекта через cherrypi.log.*."
</i18n>
