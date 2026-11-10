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
      CardTitle Project plugins
      CardDescription
        | A plugin works in this project only while one of its owners has it
        | enabled here; disabling it removes its access and keeps its settings
        | and logs. These switches also cap what a plugin may create: whether it
        | may create tasks at all, and the highest priority it may request. They
        | stay editable while the plugin is disabled.
    CardContent.flex.flex-col.gap-2
      p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
      p.text-muted-foreground.text-sm(v-if="loading") Loading…
      p.text-muted-foreground.text-sm(v-else-if="!rows.length")
        | No plugin declares a project settings surface.

  Card(v-for="row in rows" :key="row.plugin?.id")
    CardHeader.flex.flex-row.items-start.justify-between.gap-4
      .flex.flex-col.gap-1
        CardTitle.flex.flex-wrap.items-center.gap-2
          PlugIcon.size-4
          | {{ row.plugin?.name }}
          Badge(v-if="row.member" variant="outline") has access
          Badge(v-else variant="secondary") no access
          Badge(v-if="row.enabled" variant="default") enabled
          Badge(v-else variant="secondary") disabled
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
            span.text-sm.font-medium Allow task creation
            span.text-muted-foreground.text-xs
              | When off, create_task fails even if the plugin is enabled.
          Switch(
            :model-value="row.allowTaskCreation"
            @update:model-value="setAllowTaskCreation(row, Boolean($event))"
            :disabled="busy === row.plugin?.id"
          )
        .flex.items-center.justify-between.gap-3.rounded-md.border.p-3
          .flex.flex-col
            span.text-sm.font-medium Maximum task priority
            span.text-muted-foreground.text-xs
              | Plugin-created tasks may not exceed this priority.
          Select(
            :model-value="String(row.maxPriority)"
            @update:model-value="setMaxPriority(row, Number($event))"
            :disabled="busy === row.plugin?.id"
          )
            SelectTrigger.w-36
              SelectValue
            SelectContent
              SelectItem(value="-1") Low
              SelectItem(value="0") Medium
              SelectItem(value="1") High
              SelectItem(value="2") Critical

      template(v-if="hasSchema(row)")
        Separator
        .flex.flex-col.gap-1
          span.text-sm.font-medium Plugin settings
          span.text-muted-foreground.text-xs
            | Project-scoped values the plugin reads through settings.get("project", id).
        PluginSettingsForm(
          :schema="row.schemaJson"
          :settings="settingsFor(row)"
          :secret-keys="row.secretKeys"
          :saving="saving === row.plugin?.id"
          :error="rowError[row.plugin?.id || '']"
          :saved="saved === row.plugin?.id"
          :errors="rowFieldErrors[row.plugin?.id || ''] || []"
          @save="saveSettings(row, $event)"
        )
      p.text-muted-foreground.text-xs(v-else)
        | This plugin declares no project settings form.

      template(v-if="row.plugin")
        Separator
        .flex.flex-col.gap-1
          span.text-sm.font-medium Runtime logs
          span.text-muted-foreground.text-xs
            | Lines this plugin emitted for this project through cherrypi.log.*.
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
