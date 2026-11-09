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
      CardTitle Plugins
      CardDescription
        | Plugins that declare a user settings surface, for example a notification
        | bridge. Enable one to edit its personal configuration; disabling keeps
        | the stored values but stops the plugin from using them.
    CardContent
      p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
      p.text-muted-foreground.text-sm(v-if="loading") Loading…
      p.text-muted-foreground.text-sm(v-else-if="!rows.length")
        | No installed plugin declares a user settings surface.

  Card(v-for="row in rows" :key="row.pluginId")
    CardHeader.flex.flex-row.items-start.justify-between.gap-4
      .flex.flex-col.gap-1
        CardTitle.flex.flex-wrap.items-center.gap-2
          PlugIcon.size-4
          | {{ pluginName(row.pluginId) }}
          Badge(v-if="row.enabled" variant="default") enabled
          Badge(v-else variant="secondary") disabled
        CardDescription {{ pluginDescription(row.pluginId) }}
      Button(
        size="sm"
        :variant="row.enabled ? 'outline' : 'default'"
        @click="setEnabled(row, !row.enabled)"
        :disabled="busy === row.pluginId"
      ) {{ row.enabled ? "Disable" : "Enable" }}

    CardContent.flex.flex-col.gap-4
      template(v-if="row.enabled")
        PluginSettingsForm(
          :schema="row.schemaJson"
          :settings="settingsFor(row)"
          :secret-keys="row.secretKeys"
          :saving="saving === row.pluginId"
          :error="rowError[row.pluginId]"
          :saved="saved === row.pluginId"
          :errors="rowFieldErrors[row.pluginId] || []"
          @save="saveSettings(row, $event)"
        )
      p.text-muted-foreground.text-sm(v-else)
        | Enable the plugin to edit your settings.
      template(v-if="plugins[row.pluginId]")
        Separator
        .flex.flex-col.gap-1
          span.text-sm.font-medium Runtime logs
          span.text-muted-foreground.text-xs
            | Lines this plugin emitted for this user through cherrypi.log.*.
        PluginRuntimeLogs(
          scope="user"
          :plugin-id="row.pluginId"
          :target-id="props.actorId"
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
import { Separator } from "@/components/ui/separator";
import { parseJSONObject, settingErrorsFrom } from "@/lib/plugins";
import { agency } from "@/lib/sdk";
import type { Plugin, PluginSettings } from "@cherrypi/sdk";
import { PlugIcon } from "@lucide/vue";
import { onMounted, reactive, ref, watch } from "vue";

const props = defineProps<{ actorId: string }>();

const rows = ref<PluginSettings[]>([]);
const plugins = ref<Record<string, Plugin>>({});
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
  () => props.actorId,
  () => void load(),
);

async function load() {
  if (!props.actorId) return;
  loading.value = true;
  error.value = "";
  try {
    const [available, list] = await Promise.all([
      agency.listSurfacePlugins("user", props.actorId),
      agency.listPlugins(),
    ]);
    rows.value = available.settings;
    plugins.value = Object.fromEntries(
      list.map((plugin) => [plugin.id, plugin]),
    );
  } catch (err) {
    error.value = message(err);
  } finally {
    loading.value = false;
  }
}

function pluginName(id: string): string {
  return plugins.value[id]?.name || id;
}

function pluginDescription(id: string): string {
  return plugins.value[id]?.description || "No description";
}

function settingsFor(row: PluginSettings): Record<string, unknown> {
  return parseJSONObject(row.settingsJson);
}

async function setEnabled(row: PluginSettings, enabled: boolean) {
  busy.value = row.pluginId;
  error.value = "";
  try {
    await agency.setPluginSettingsEnabled(
      row.pluginId,
      "user",
      props.actorId,
      enabled,
    );
    await load();
  } catch (err) {
    error.value = message(err);
  } finally {
    busy.value = "";
  }
}

async function saveSettings(
  row: PluginSettings,
  values: Record<string, unknown>,
) {
  saving.value = row.pluginId;
  saved.value = "";
  rowError[row.pluginId] = "";
  rowFieldErrors[row.pluginId] = [];
  try {
    await agency.updatePluginSettings(
      row.pluginId,
      "user",
      props.actorId,
      values,
    );
    saved.value = row.pluginId;
    await load();
  } catch (err) {
    const fieldErrors = settingErrorsFrom(err);
    if (fieldErrors.length) {
      rowFieldErrors[row.pluginId] = fieldErrors;
    } else {
      rowError[row.pluginId] = message(err);
    }
  } finally {
    saving.value = "";
  }
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
</script>
