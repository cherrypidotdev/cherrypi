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
  RouterLink.text-muted-foreground.text-sm(:to="{ name: 'settings-plugins' }") {{ $t("plugins.back") }}

  p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
  p.text-muted-foreground.text-sm(v-if="loading") {{ $t("common.loading") }}
  Empty(v-else-if="!plugin")
    EmptyTitle {{ $t("plugins.notFound") }}
    EmptyDescription {{ $t("plugins.notFoundHint") }}

  template(v-else)
    .flex.items-start.justify-between.gap-4
      .flex.flex-col.gap-1
        h2.flex.items-center.gap-2.text-base.font-semibold
          PluginIcon(:src="plugin.iconUrl" size="size-5")
          | {{ plugin.name || pluginId }}
          Badge(:variant="plugin.enabled ? 'default' : 'secondary'")
            | {{ plugin.enabled ? "active" : "disabled" }}
        p.text-muted-foreground.text-sm
          | {{ plugin.description || "No description" }}
        p.text-destructive.text-sm(v-if="plugin.loadError" role="alert")
          | {{ $t("plugins.loadErrorHint") }}
        pre.text-destructive.text-xs.whitespace-pre-wrap(
          v-if="plugin.loadError"
        ) {{ plugin.loadError }}
      .flex.items-center.gap-2
        Button(variant="outline" @click="toggle" :disabled="busy")
          | {{ plugin.enabled ? $t("plugins.disable") : $t("plugins.enable") }}
        Button(variant="destructive" @click="remove" :disabled="busy")
          | {{ $t("plugins.uninstall") }}

    Card
      CardHeader
        CardTitle {{ $t("plugins.manifest") }}
        CardDescription
          | {{ $t("plugins.versionRuntimeEntry", { version: plugin.version, runtime: plugin.runtime, entry: plugin.entry }) }}
      CardContent.flex.flex-col.gap-4
        .flex.flex-wrap.gap-1
          Badge(
            v-for="handler in manifest.handlers"
            :key="handler"
            variant="outline"
          )
            | {{ handlerLabel(handler) }}
        p.text-muted-foreground.text-xs(v-if="manifest.network.length")
          | {{ $t("plugins.outboundHosts", { hosts: manifest.network.join(", ") }) }}
        .flex.flex-wrap.items-center.gap-2(v-if="anyHostNetwork")
          Badge(variant="destructive") {{ $t("plugins.anyHost") }}
          span.text-muted-foreground.text-xs
            | {{ $t("plugins.anyHostHint") }}
        p.text-muted-foreground.text-xs(v-if="plugin.actorUsername")
          | {{ $t("plugins.actorLine", { username: plugin.actorUsername, id: plugin.actorId }) }}

    Card(v-if="manifest.routes.length")
      CardHeader
        CardTitle {{ $t("plugins.routes") }}
        CardDescription
          | {{ $t("plugins.servedUnder", { id: plugin.id }) }}
          | {{ $t("plugins.routesHint") }}
      CardContent
        Table
          TableHeader
            TableRow
              TableHead {{ $t("plugins.column.route") }}
              TableHead {{ $t("plugins.column.methods") }}
              TableHead {{ $t("common.description") }}
          TableBody
            TableRow(v-for="route in manifest.routes" :key="route.path")
              TableCell.font-mono.text-xs /plugins/{{ plugin.id }}{{ route.path }}
              TableCell.text-xs {{ route.methods.join(", ") }}
              TableCell.text-muted-foreground.text-xs {{ route.description || "—" }}

    Card(v-if="manifest.tools.length")
      CardHeader
        CardTitle {{ $t("plugins.tools") }}
        CardDescription
          | {{ $t("plugins.toolsHint") }}
      CardContent.flex.flex-col.gap-2
        .flex.flex-col.gap-1(v-for="tool in manifest.tools" :key="tool.name")
          span.font-mono.text-xs {{ tool.name }}
          span.text-muted-foreground.text-xs {{ tool.description }}

    Card(v-if="manifest.events.length")
      CardHeader
        CardTitle {{ $t("webhooks.events") }}
        CardDescription
          | {{ $t("plugins.eventsHint") }}
      CardContent.flex.flex-wrap.gap-1
        Badge(
          v-for="event in manifest.events"
          :key="event"
          variant="secondary"
        )
          | {{ event }}

    Card(v-if="hasOrgSurface")
      CardHeader
        CardTitle {{ $t("nav.orgSettings") }}
        CardDescription
          | {{ $t("plugins.settingsHint") }}
      CardContent
        PluginSettingsForm(
          scope="org"
          :target-id="''"
          :schema="orgSettings?.schemaJson"
          :settings="orgValues"
          :secret-keys="orgSettings?.secretKeys || []"
          :saving="savingSettings"
          :error="settingsError"
          :saved="settingsSaved"
          :errors="orgFieldErrors"
          :plugin-id="plugin.id"
          @save="saveOrgSettings"
        )

    Card(v-if="otherSurfaces.length")
      CardHeader
        CardTitle {{ $t("plugins.otherSurfaces") }}
        CardDescription
          | {{ $t("plugins.otherSurfacesHint") }}
      CardContent.flex.flex-wrap.gap-1
        Badge(v-for="scope in otherSurfaces" :key="scope" variant="outline")
          | {{ $t(PLUGIN_SCOPE_KEYS[scope]) }}

    Card
      CardHeader.flex.flex-row.items-center.justify-between
        .flex.flex-col.gap-1
          CardTitle {{ $t("plugins.activityLog") }}
          CardDescription {{ $t("plugins.activityLogHint") }}
        Button(
          variant="outline"
          size="sm"
          @click="reloadLogs"
          :disabled="loadingLogs"
        )
          | {{ $t("common.refresh") }}
      CardContent
        .max-h-80.overflow-auto.rounded-md.border(v-if="logs.length")
          Table
            TableHeader
              TableRow
                TableHead {{ $t("plugins.column.time") }}
                TableHead {{ $t("plugins.column.level") }}
                TableHead {{ $t("plugins.column.message") }}
            TableBody
              TableRow(v-for="line in logs" :key="String(line.id)")
                TableCell.text-xs.whitespace-nowrap {{ formatTime(line.at) }}
                TableCell.text-xs {{ line.level }}
                TableCell.text-xs.break-all
                  p {{ line.message }}
                  PluginLogData(:data-json="line.dataJson")
          PluginLogLoadMore(
            :has-more="logsHasMore"
            :auto="logsAuto"
            :loaded="logsLoaded"
            @load="loadMoreLogs"
            :loading="loadingMoreLogs"
          )
        p.text-destructive.text-sm(v-if="logsError" role="alert") {{ logsError }}
        p.text-muted-foreground.text-sm(
          v-else-if="!loadingLogs && !logs.length"
        )
          | {{ $t("plugins.noActivity") }}

    Card
      CardHeader
        CardTitle {{ $t("plugins.runtimeLogs") }}
        CardDescription
          | {{ $t("plugins.runtimeLogsHint") }}
      CardContent
        PluginRuntimeLogs(:plugin-id="plugin.id")
</template>

<script setup lang="ts">
import PluginIcon from "@/components/plugins/PluginIcon.vue";
import PluginLogData from "@/components/plugins/PluginLogData.vue";
import PluginLogLoadMore from "@/components/plugins/PluginLogLoadMore.vue";
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { usePluginLogPager } from "@/composables/usePluginLogPager";
import {
  PLUGIN_SCOPE_KEYS,
  allowsAnyHost,
  handlerLabel,
  hasSurface,
  manifestSurfaces,
  parseJSONObject,
  parseManifest,
  settingErrorsFrom,
  type PluginManifest,
} from "@/lib/plugins";
import { agency } from "@/lib/sdk";
import type { Plugin, PluginLog, PluginSettings } from "@cherrypi/sdk";
import { computed, onMounted, ref, watch } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const pluginId = computed(() => String(route.params.pluginId ?? ""));
const plugin = ref<Plugin | null>(null);
const loading = ref(false);
const busy = ref(false);
const error = ref("");
const orgSettings = ref<PluginSettings | null>(null);
const savingSettings = ref(false);
const orgFieldErrors = ref<Array<{ path: string; errorText: string }>>([]);
const settingsError = ref("");
const settingsSaved = ref(false);

const {
  logs,
  loading: loadingLogs,
  loadingMore: loadingMoreLogs,
  error: logsError,
  hasMore: logsHasMore,
  auto: logsAuto,
  loaded: logsLoaded,
  load: loadLogs,
  loadMore: loadMoreLogs,
  reset: resetLogs,
} = usePluginLogPager((beforeId, limit) => {
  const id = plugin.value?.id;
  if (!id) return Promise.resolve([]);
  return agency.listPluginLogs(id, { limit, beforeId });
});

const manifest = computed<PluginManifest>(() =>
  plugin.value ? parseManifest(plugin.value) : emptyManifest(),
);
const orgValues = computed(() =>
  parseJSONObject(orgSettings.value?.settingsJson),
);
const hasOrgSurface = computed(() =>
  plugin.value ? hasSurface(manifest.value, "org") : false,
);
const otherSurfaces = computed(() =>
  manifestSurfaces(manifest.value).filter((scope) => scope !== "org"),
);
const anyHostNetwork = computed(() => allowsAnyHost(manifest.value));

function emptyManifest(): PluginManifest {
  return {
    id: "",
    name: "",
    version: "",
    description: "",
    runtime: "lua",
    entry: "main.lua",
    handlers: [],
    settings: {},
    tools: [],
    events: [],
    network: [],
    routes: [],
  };
}

onMounted(() => void load());

watch(pluginId, () => {
  plugin.value = null;
  resetLogs();
  void load();
});

async function load() {
  if (!pluginId.value) return;
  loading.value = true;
  error.value = "";
  settingsError.value = "";
  settingsSaved.value = false;
  orgFieldErrors.value = [];
  try {
    plugin.value = await agency.getPlugin(pluginId.value);
    if (hasSurface(parseManifest(plugin.value), "org")) {
      orgSettings.value = await agency.getPluginSettings(pluginId.value, "org");
    } else {
      orgSettings.value = null;
    }
    resetLogs();
    await loadLogs();
  } catch (err) {
    error.value = message(err);
  } finally {
    loading.value = false;
  }
}

function reloadLogs() {
  resetLogs();
  void loadLogs();
}

async function toggle() {
  const current = plugin.value;
  if (!current) return;
  busy.value = true;
  error.value = "";
  try {
    plugin.value = await agency.setPluginEnabled(current.id, !current.enabled);
  } catch (err) {
    error.value = message(err);
  } finally {
    busy.value = false;
  }
}

async function remove() {
  const current = plugin.value;
  if (!current) return;
  busy.value = true;
  error.value = "";
  try {
    await agency.uninstallPlugin(current.id, false);
    await router.push({ name: "settings-plugins" });
  } catch (err) {
    error.value = message(err);
  } finally {
    busy.value = false;
  }
}

async function saveOrgSettings(values: Record<string, unknown>) {
  if (!plugin.value) return;
  savingSettings.value = true;
  settingsError.value = "";
  settingsSaved.value = false;

  orgFieldErrors.value = [];
  try {
    orgSettings.value = await agency.updatePluginSettings(
      plugin.value.id,
      "org",
      "",
      values,
    );
    settingsSaved.value = true;
  } catch (err) {
    const fieldErrors = settingErrorsFrom(err);
    if (fieldErrors.length) {
      orgFieldErrors.value = fieldErrors;
    } else {
      settingsError.value = message(err);
    }
  } finally {
    savingSettings.value = false;
  }
}

function formatTime(value: PluginLog["at"]): string {
  if (!value) return "—";
  return new Date(
    Number(value.seconds) * 1000 + Math.floor(value.nanos / 1_000_000),
  ).toLocaleTimeString();
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
</script>

<i18n lang="yaml">
en:
  plugins:
    back: "← Plugins"
    notFound: "Plugin not found"
    notFoundHint: "This plugin is no longer installed, or its page link is stale."
    uninstall: "Uninstall"
    manifest: "Manifest"
    anyHostHint: "This bundle may call any public HTTPS host. Review the manifest before enabling it."
    routes: "Routes"
    routesHint: "any path anonymously; the plugin validates its own secret."
    tools: "Tools"
    toolsHint: "Named tools this plugin can attach to the tasks it creates."
    eventsHint: "handle_event receives the events the plugin subscribes to."
    settingsHint: "Agency-wide values. Secret fields are masked; reveal or copy them when needed."
    otherSurfaces: "Other settings surfaces"
    otherSurfacesHint: "Agents, users, and projects enable this plugin in their own settings."
    activityLog: "Activity Log"
    activityLogHint: "Lifecycle, route, event, and failure entries."
    noActivity: "No activity lines yet."
    runtimeLogs: "Runtime Logs"
    runtimeLogsHint: "Surface-scoped lines the plugin emits through cherrypi.log.*, including trace, debug, info, warn, and error levels."
    versionRuntimeEntry: "Version {version} · runtime {runtime} · entry {entry}"
    outboundHosts: "Outbound hosts: {hosts}"
    actorLine: "Plugin actor: {'@'}{username} (id {id})"
    servedUnder: "Served under /plugins/{id}/. A provider webhook may call this route."
    column:
      time: "Time"
      level: "Level"
      message: "Message"
      route: "Route"
      methods: "Methods"
ru:
  plugins:
    back: "← Плагины"
    notFound: "Плагин не найден"
    notFoundHint: "Этот плагин больше не установлен или ссылка на его страницу устарела."
    uninstall: "Удалить"
    manifest: "Манифест"
    anyHostHint: "Этот пакет может обращаться к любому публичному хосту HTTPS. Проверьте манифест перед включением."
    routes: "Маршруты"
    routesHint: "любой путь анонимно; плагин сам проверяет свой секрет."
    tools: "Инструменты"
    toolsHint: "Именованные инструменты, которые плагин может прикреплять к создаваемым задачам."
    eventsHint: "handle_event получает события, на которые подписан плагин."
    settingsHint: "Значения для всего агентства. Секретные поля скрыты; при необходимости откройте их или скопируйте."
    otherSurfaces: "Другие поверхности настроек"
    otherSurfacesHint: "Агенты, пользователи и проекты включают этот плагин в своих настройках."
    activityLog: "Журнал активности"
    activityLogHint: "Записи о жизненном цикле, маршрутах, событиях и сбоях."
    noActivity: "Записей активности пока нет."
    runtimeLogs: "Журнал выполнения"
    runtimeLogsHint: "Строки с привязкой к поверхности, которые плагин пишет через cherrypi.log.*, включая уровни trace, debug, info, warn и error."
    versionRuntimeEntry: "Версия {version} · среда {runtime} · точка входа {entry}"
    outboundHosts: "Исходящие хосты: {hosts}"
    actorLine: "Субъект плагина: {'@'}{username} (id {id})"
    servedUnder: "Доступен по пути /plugins/{id}/. Вебхук провайдера может вызывать этот маршрут."
    column:
      time: "Время"
      level: "Уровень"
      message: "Сообщение"
      route: "Маршрут"
      methods: "Методы"
</i18n>
