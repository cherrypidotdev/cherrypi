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
  RouterLink.text-muted-foreground.text-sm(
    :to="{ name: 'project-settings-plugins', params: { slug } }"
  ) {{ $t("projectPlugins.back") }}

  p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
  p.text-muted-foreground.text-sm(v-if="loading") {{ $t("common.loading") }}
  Empty(v-else-if="!row")
    EmptyTitle {{ $t("projectPlugins.notFound") }}
    EmptyDescription {{ $t("projectPlugins.notFoundHint") }}

  template(v-else)
    .flex.flex-wrap.items-start.justify-between.gap-4
      .flex.flex-col.gap-1
        h2.flex.flex-wrap.items-center.gap-2.text-base.font-semibold
          PlugIcon.size-4
          | {{ row.plugin?.name }}
          Badge(v-if="row.member" variant="outline") {{ $t("projectPlugins.hasAccess") }}
          Badge(v-else variant="secondary") {{ $t("projectPlugins.noAccess") }}
          Badge(v-if="row.enabled" variant="default") {{ $t("agentPlugins.enabled") }}
          Badge(v-else variant="secondary") {{ $t("settings.auth.disabled") }}
        p.text-muted-foreground.text-sm
          | {{ row.plugin?.description || "No description" }}
      Button(
        :variant="row.enabled ? 'outline' : 'default'"
        @click="setEnabled(row, !row.enabled)"
        :disabled="busy"
      ) {{ row.enabled ? $t("projectPlugins.disable") : $t("projectPlugins.enable") }}

    Card
      CardHeader
        CardTitle {{ $t("projectPlugins.capsTitle") }}
        CardDescription
          | {{ $t("projectPlugins.capsDescription") }}
      CardContent
        .grid.gap-4(class="md:grid-cols-2")
          .flex.items-center.justify-between.gap-3.rounded-md.border.p-3
            .flex.flex-col
              span.text-sm.font-medium {{ $t("projectPlugins.allowTaskCreation") }}
              span.text-muted-foreground.text-xs
                | {{ $t("projectPlugins.allowTaskCreationHint") }}
            Switch(
              :model-value="row.allowTaskCreation"
              @update:model-value="setAllowTaskCreation(row, Boolean($event))"
              :disabled="busy"
            )
          .flex.items-center.justify-between.gap-3.rounded-md.border.p-3
            .flex.flex-col
              span.text-sm.font-medium {{ $t("projectPlugins.maxPriority") }}
              span.text-muted-foreground.text-xs
                | {{ $t("projectPlugins.maxPriorityHint") }}
            Select(
              :model-value="String(row.maxPriority)"
              @update:model-value="setMaxPriority(row, Number($event))"
              :disabled="busy"
            )
              SelectTrigger.w-36
                SelectValue
              SelectContent
                SelectItem(value="-1") {{ $t("projectPlugins.priorityLow") }}
                SelectItem(value="0") {{ $t("projectPlugins.priorityMedium") }}
                SelectItem(value="1") {{ $t("projectPlugins.priorityHigh") }}
                SelectItem(value="2") {{ $t("projectPlugins.priorityCritical") }}

    Card
      CardHeader
        CardTitle {{ $t("projectPlugins.settingsTitle") }}
        CardDescription
          | {{ $t("projectPlugins.settingsHint") }}
      CardContent
        template(v-if="hasSchema(row)")
          PluginSettingsForm(
            scope="project"
            :schema="row.schemaJson"
            :settings="settingsFor(row)"
            :secret-keys="row.secretKeys"
            :saving="saving"
            :error="settingsError"
            :saved="saved"
            :errors="fieldErrors"
            :plugin-id="row.plugin?.id || ''"
            :target-id="projectId"
            @save="saveSettings(row, $event)"
          )
        p.text-muted-foreground.text-xs(v-else)
          | {{ $t("projectPlugins.noSettingsForm") }}

    Card(v-if="row.plugin")
      CardHeader
        CardTitle {{ $t("agentPlugins.runtimeLogs") }}
        CardDescription
          | {{ $t("projectPlugins.runtimeLogsHint") }}
      CardContent
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
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  parseJSONObject,
  pluginSchemaForm,
  settingErrorsFrom,
} from "@/lib/plugins";
import { agency } from "@/lib/sdk";
import type { ProjectPlugin } from "@cherrypi/sdk";
import { PlugIcon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRoute } from "vue-router";

const props = defineProps<{ projectId: string }>();

const { t } = useI18n();
const route = useRoute();
const slug = computed(() => String(route.params.slug ?? ""));
const pluginId = computed(() => String(route.params.pluginId ?? ""));

const row = ref<ProjectPlugin>();
const loading = ref(false);
const busy = ref(false);
const saving = ref(false);
const saved = ref(false);
const error = ref("");
const settingsError = ref("");
const fieldErrors = ref<Array<{ path: string; errorText: string }>>([]);

onMounted(() => void load());

watch([() => props.projectId, pluginId], () => {
  row.value = undefined;
  void load();
});

async function load() {
  if (!props.projectId || !pluginId.value) return;
  loading.value = true;
  error.value = "";
  try {
    const result = await agency.listSurfacePlugins("project", props.projectId);
    row.value = result.projectPlugins.find(
      (candidate) => candidate.plugin?.id === pluginId.value,
    );
  } catch (err) {
    error.value = message(err);
  } finally {
    loading.value = false;
  }
}

function settingsFor(item: ProjectPlugin): Record<string, unknown> {
  return parseJSONObject(item.settingsJson);
}

function hasSchema(item: ProjectPlugin): boolean {
  return pluginSchemaForm(item.schemaJson) !== "none";
}

async function setEnabled(item: ProjectPlugin, enabled: boolean) {
  if (
    !enabled &&
    !window.confirm(
      t("projectPlugins.disableConfirm", { name: item.plugin?.name ?? "" }),
    )
  ) {
    return;
  }
  await update(item, { enabled });
}

async function setAllowTaskCreation(item: ProjectPlugin, allow: boolean) {
  await update(item, { allowTaskCreation: allow });
}

async function setMaxPriority(item: ProjectPlugin, maxPriority: number) {
  await update(item, { maxPriority });
}

async function update(
  item: ProjectPlugin,
  patch: Partial<{
    enabled: boolean;
    allowTaskCreation: boolean;
    maxPriority: number;
  }>,
) {
  if (!item.plugin) return;
  busy.value = true;
  error.value = "";
  try {
    await agency.updateProjectPlugin(item.plugin.id, props.projectId, {
      enabled: item.enabled,
      allowTaskCreation: item.allowTaskCreation,
      maxPriority: item.maxPriority,
      ...patch,
    });
    await load();
  } catch (err) {
    error.value = message(err);
  } finally {
    busy.value = false;
  }
}

async function saveSettings(
  item: ProjectPlugin,
  values: Record<string, unknown>,
) {
  if (!item.plugin) return;
  saving.value = true;
  saved.value = false;
  settingsError.value = "";
  fieldErrors.value = [];
  try {
    await agency.updateProjectPlugin(item.plugin.id, props.projectId, {
      enabled: item.enabled,
      allowTaskCreation: item.allowTaskCreation,
      maxPriority: item.maxPriority,
      settingsJson: JSON.stringify(values),
    });
    saved.value = true;
    await load();
  } catch (err) {
    const errors = settingErrorsFrom(err);
    if (errors.length) fieldErrors.value = errors;
    else settingsError.value = message(err);
  } finally {
    saving.value = false;
  }
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
</script>

<i18n lang="yaml">
en:
  projectPlugins:
    back: "← Plugins"
    enable: "Enable"
    disable: "Disable"
    disableConfirm: "Disable {name}? It loses access to this project; its settings and logs stay."
    notFound: "Plugin not found"
    notFoundHint: "This plugin no longer declares a project settings surface, or it was uninstalled."
    capsTitle: "What the plugin may do"
    capsDescription: "These switches cap what a plugin may create: whether it may create tasks at all, and the highest priority it may request. They stay editable while the plugin is disabled."
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
    back: "← Плагины"
    enable: "Включить"
    disable: "Отключить"
    disableConfirm: "Отключить {name}? Плагин потеряет доступ к этому проекту; его настройки и журналы сохранятся."
    notFound: "Плагин не найден"
    notFoundHint: "Этот плагин больше не объявляет поверхность настроек проекта или был удалён."
    capsTitle: "Что может плагин"
    capsDescription: "Эти переключатели ограничивают возможности плагина: может ли он создавать задачи и какой максимальный приоритет запрашивать. Они остаются доступными, пока плагин отключён."
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
