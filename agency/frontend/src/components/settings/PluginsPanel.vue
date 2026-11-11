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
  template(v-if="selectedId")
    .flex.items-start.justify-between.gap-4
      .flex.flex-col.gap-1
        Button.text-muted-foreground.px-0(
          variant="link"
          size="sm"
          @click="closeDetail"
        )
          | {{ $t("plugins.back") }}
        h2.flex.items-center.gap-2.text-base.font-semibold
          PlugIcon.size-5
          | {{ plugin?.name || selectedId }}
          Badge(
            v-if="plugin"
            :variant="plugin.enabled ? 'default' : 'secondary'"
          )
            | {{ plugin.enabled ? "active" : "disabled" }}
        p.text-muted-foreground.text-sm
          | {{ plugin?.description || "No description" }}
      .flex.items-center.gap-2(v-if="plugin")
        Button(variant="outline" @click="toggle(plugin)" :disabled="busy")
          | {{ plugin.enabled ? "Disable" : "Enable" }}
        Button(variant="destructive" @click="remove(plugin)" :disabled="busy")
          | {{ $t("plugins.uninstall") }}

    p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
    p.text-muted-foreground.text-sm(v-if="loadingDetail") {{ $t("common.loading") }}

    template(v-if="plugin")
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

  template(v-else)
    .flex.items-start.justify-between.gap-4
      .flex.flex-col.gap-1
        h2.text-base.font-semibold {{ $t("account.tabs.plugins") }}
        p.text-muted-foreground.text-sm
          | {{ $t("plugins.description") }}
      Button(@click="openInstall")
        PlusIcon(data-icon="inline-start")
        span {{ $t("plugins.install") }}

    Dialog(v-model:open="installOpen")
      DialogContent.overflow-y-auto(class="max-h-[85vh] sm:max-w-2xl")
        DialogHeader
          DialogTitle {{ $t("plugins.reviewBundle") }}
          DialogDescription
            | {{ $t("plugins.reviewBundleHint") }}
        .flex.flex-col.gap-4
          template(v-if="!preview")
            input.text-sm(
              type="file"
              accept=".zip,application/zip"
              @change="onFile"
            )
            p.text-muted-foreground.text-xs(v-if="selectedFile")
              | {{ $t("plugins.fileSize", { name: selectedFile.name, size: Math.ceil(selectedFile.size / 1024) }) }}
          template(v-else)
            .flex.flex-col.gap-1
              h3.text-base.font-semibold
                | {{ preview.name }}
                span.text-muted-foreground v{{ preview.version }}
              p.text-muted-foreground.text-sm
                | {{ preview.description || "No description" }}
              p.text-muted-foreground.text-xs
                | {{ $t("plugins.previewMeta", { id: preview.id, runtime: preview.runtime, entry: preview.entry, checksum: shortChecksum(preview.checksum) }) }}
            .flex.flex-wrap.gap-1
              Badge(
                v-for="handler in preview.handlers"
                :key="handler"
                variant="outline"
              ) {{ handlerLabel(handler) }}
            .flex.flex-col.gap-2(v-if="preview.warnings.length")
              h4.text-sm.font-medium {{ $t("plugins.reviewNotes") }}
              p.text-muted-foreground.text-xs(
                v-for="warning in preview.warnings"
                :key="warning"
              ) {{ warning }}
            .flex.flex-col.gap-2
              h4.text-sm.font-medium {{ $t("plugins.permissions") }}
              p.text-muted-foreground.text-xs(
                v-if="!previewPermissions.length"
              )
                | {{ $t("plugins.noPermissions") }}
              .flex.flex-col.gap-2
                .rounded-md.border.p-3(
                  v-for="permission in previewPermissions"
                  :key="permission.key"
                )
                  .flex.items-center.gap-2
                    Badge(
                      :variant="permission.tone === 'danger' ? 'destructive' : permission.tone === 'warning' ? 'secondary' : 'outline'"
                    ) {{ permission.label }}
                  p.text-muted-foreground.text-xs {{ permission.detail }}
            details.rounded-md.border.p-3
              summary.cursor-pointer.text-sm.font-medium {{ $t("plugins.manifest") }}
              pre.mt-2.max-h-64.overflow-auto.text-xs {{ prettyManifest }}
          p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
        DialogFooter
          DialogClose
            Button(type="button" variant="outline" @click="closeInstall")
              | {{ $t("common.cancel") }}
          Button(
            v-if="!preview"
            type="button"
            @click="review"
            :disabled="!file || inspecting"
          )
            Spinner(v-if="inspecting" data-icon="inline-start")
            SearchIcon(v-else data-icon="inline-start")
            span {{ $t("plugins.review") }}
          Button(
            v-else
            type="button"
            @click="install"
            :disabled="installing || !preview.compatible"
          )
            Spinner(v-if="installing" data-icon="inline-start")
            UploadIcon(v-else data-icon="inline-start")
            span {{ $t("plugins.install") }}

    .grid.gap-4(v-if="plugins.length" class="md:grid-cols-2")
      Card.flex.flex-col(v-for="item in plugins" :key="item.id")
        CardHeader
          CardTitle.flex.items-center.gap-2
            PlugIcon.size-4
            Button.text-primary.px-0.text-left(
              variant="link"
              @click="openDetail(item.id)"
            )
              | {{ item.plugin.name }}
            Badge(:variant="item.plugin.enabled ? 'default' : 'secondary'")
              | {{ item.plugin.enabled ? "active" : "disabled" }}
          CardDescription {{ item.plugin.description || "No description" }}
        CardContent.flex.flex-1.flex-col.gap-3
          .text-muted-foreground.flex.flex-wrap.gap-2.text-xs
            span v{{ item.plugin.version }}
            span {{ $t("plugins.handlers", { count: item.handlers.length }) }}
            span {{ $t("plugins.toolsCount", { count: item.tools.length }) }}
            span {{ $t("plugins.surfacesCount", { count: item.surfaces.length }) }}
            Badge(v-if="item.anyHost" variant="destructive") {{ $t("plugins.anyHost") }}
          .flex.flex-wrap.gap-1
            Badge(
              v-for="surface in item.surfaces"
              :key="surface"
              variant="outline"
            )
              | {{ $t(PLUGIN_SCOPE_KEYS[surface]) }}
        CardFooter.flex.justify-between.gap-2
          Button(variant="outline" size="sm" @click="openDetail(item.id)")
            | {{ $t("common.open") }}
          Button(
            variant="ghost"
            size="sm"
            @click="toggle(item.plugin)"
            :disabled="busy === item.id"
          ) {{ item.plugin.enabled ? "Disable" : "Enable" }}

    Card(v-else-if="!loadingPlugins")
      CardHeader
        CardTitle {{ $t("plugins.empty") }}
        CardDescription
          | {{ $t("plugins.emptyHint") }}
</template>

<script setup lang="ts">
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
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
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
  parseManifestJSON,
  pluginPermissions,
  settingErrorsFrom,
  type PluginManifest,
  type PluginScope,
} from "@/lib/plugins";
import { agency } from "@/lib/sdk";
import type {
  Plugin,
  PluginLog,
  PluginPreview,
  PluginSettings,
} from "@cherrypi/sdk";
import { PlugIcon, PlusIcon, SearchIcon, UploadIcon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

interface PluginCard {
  plugin: Plugin;
  id: string;
  handlers: string[];
  tools: unknown[];
  surfaces: PluginScope[];
  anyHost: boolean;
}

const route = useRoute();
const router = useRouter();

const plugins = ref<PluginCard[]>([]);
const loadingPlugins = ref(false);
const selectedId = ref(
  typeof route.query.plugin === "string" ? route.query.plugin : "",
);
const plugin = ref<Plugin | null>(null);
const loadingDetail = ref(false);
const orgSettings = ref<PluginSettings | null>(null);
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
const busy = ref("");
const error = ref("");
const installOpen = ref(false);
const file = ref<File | null>(null);
const selectedFile = ref<File | null>(null);
const installing = ref(false);
const inspecting = ref(false);
const preview = ref<PluginPreview | null>(null);
const bundleBytes = ref<Uint8Array | null>(null);
const savingSettings = ref(false);
const orgFieldErrors = ref<Array<{ path: string; errorText: string }>>([]);
const settingsError = ref("");
const settingsSaved = ref(false);

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
const previewManifest = computed<PluginManifest>(() =>
  preview.value
    ? parseManifestJSON(preview.value.manifestJson, {
        id: preview.value.id,
        name: preview.value.name,
        version: preview.value.version,
        description: preview.value.description,
        runtime: preview.value.runtime,
        entry: preview.value.entry,
      })
    : emptyManifest(),
);
const previewPermissions = computed(() =>
  pluginPermissions(previewManifest.value),
);
const prettyManifest = computed(() => {
  if (!preview.value) return "{}";
  try {
    return JSON.stringify(JSON.parse(preview.value.manifestJson), null, 2);
  } catch {
    return preview.value.manifestJson;
  }
});

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

watch(
  () => route.query.plugin,
  (value) => {
    selectedId.value = typeof value === "string" ? value : "";
  },
);

watch(selectedId, (value) => {
  if (value) {
    void loadDetail(value);
  } else {
    plugin.value = null;
    orgSettings.value = null;
    resetLogs();
  }
});

onMounted(() => {
  void loadPlugins();
  if (selectedId.value) void loadDetail(selectedId.value);
});

async function loadPlugins() {
  loadingPlugins.value = true;
  error.value = "";
  try {
    const list = await agency.listPlugins();
    plugins.value = list.map((item) => {
      const m = parseManifest(item);
      return {
        plugin: item,
        id: item.id,
        handlers: m.handlers,
        tools: m.tools,
        surfaces: manifestSurfaces(m),
        anyHost: allowsAnyHost(m),
      };
    });
  } catch (err) {
    error.value = message(err);
  } finally {
    loadingPlugins.value = false;
  }
}

async function loadDetail(id: string) {
  loadingDetail.value = true;
  error.value = "";
  settingsError.value = "";
  settingsSaved.value = false;
  orgFieldErrors.value = [];
  try {
    plugin.value = await agency.getPlugin(id);
    if (hasSurface(parseManifest(plugin.value), "org")) {
      orgSettings.value = await agency.getPluginSettings(id, "org");
    }
    resetLogs();
    await loadLogs();
  } catch (err) {
    error.value = message(err);
  } finally {
    loadingDetail.value = false;
  }
}

function reloadLogs() {
  resetLogs();
  void loadLogs();
}

function openDetail(id: string) {
  selectedId.value = id;
  void router.push({ query: { ...route.query, tab: "plugins", plugin: id } });
}

function closeDetail() {
  selectedId.value = "";
  plugin.value = null;
  const query = { ...route.query };
  delete query.plugin;
  void router.push({ query });
}

async function toggle(item: Plugin) {
  busy.value = item.id;
  error.value = "";
  try {
    const updated = await agency.setPluginEnabled(item.id, !item.enabled);
    if (plugin.value?.id === updated.id) plugin.value = updated;
    await loadPlugins();
  } catch (err) {
    error.value = message(err);
  } finally {
    busy.value = "";
  }
}

async function remove(item: Plugin) {
  busy.value = item.id;
  error.value = "";
  try {
    await agency.uninstallPlugin(item.id, false);
    closeDetail();
    await loadPlugins();
  } catch (err) {
    error.value = message(err);
  } finally {
    busy.value = "";
  }
}

function onFile(event: Event) {
  const input = event.target as HTMLInputElement;
  file.value = input.files?.[0] ?? null;
  selectedFile.value = file.value;
  preview.value = null;
  bundleBytes.value = null;
  error.value = "";
}

function openInstall() {
  installOpen.value = true;
  preview.value = null;
  bundleBytes.value = null;
  file.value = null;
  selectedFile.value = null;
  error.value = "";
}

function closeInstall() {
  installOpen.value = false;
  preview.value = null;
  bundleBytes.value = null;
  file.value = null;
  selectedFile.value = null;
  error.value = "";
}

async function review() {
  if (!file.value) return;
  inspecting.value = true;
  error.value = "";
  try {
    const bytes = new Uint8Array(await file.value.arrayBuffer());
    bundleBytes.value = bytes;
    preview.value = await agency.inspectPlugin(bytes);
  } catch (err) {
    error.value = message(err);
  } finally {
    inspecting.value = false;
  }
}

async function install() {
  if (!bundleBytes.value) return;
  installing.value = true;
  error.value = "";
  try {
    const installed = await agency.installPlugin(bundleBytes.value);
    closeInstall();
    await loadPlugins();
    openDetail(installed.id);
  } catch (err) {
    error.value = message(err);
  } finally {
    installing.value = false;
  }
}

function shortChecksum(value: string): string {
  return value ? value.slice(0, 12) : "—";
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
    column:
      time: "Time"
      level: "Level"
      message: "Message"
      route: "Route"
      methods: "Methods"
    back: "← Plugins"
    uninstall: "Uninstall"
    manifest: "Manifest"
    anyHost: "Any HTTPS host"
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
    description: "Sandboxed Lua extensions installed at the agency level. Each call runs in a fresh Lua state."
    install: "Install plugin"
    reviewBundle: "Review plugin bundle"
    reviewBundleHint: "Inspect the manifest and the permissions it requests before it is installed for the whole organization."
    reviewNotes: "Review notes"
    permissions: "Permissions"
    noPermissions: "This bundle requests no outbound network, routes, events, tools, or settings surfaces."
    review: "Review"
    empty: "No plugins installed"
    emptyHint: "Upload a bundle to install the first plugin. Each call runs in a fresh Lua state."
    versionRuntimeEntry: "Version {version} · runtime {runtime} · entry {entry}"
    outboundHosts: "Outbound hosts: {hosts}"
    actorLine: "Plugin actor: {'@'}{username} (id {id})"
    servedUnder: "Served under /plugins/{id}/. A provider webhook may call this route."
    fileSize: "{name} · {size} KiB"
    previewMeta: "id {id} · runtime {runtime} · entry {entry} · sha256 {checksum}"
    handlers: "{count} handlers"
    toolsCount: "{count} tools"
    surfacesCount: "{count} surfaces"
ru:
  plugins:
    column:
      time: "Время"
      level: "Уровень"
      message: "Сообщение"
      route: "Маршрут"
      methods: "Методы"
    back: "← Плагины"
    uninstall: "Удалить"
    manifest: "Манифест"
    anyHost: "Любой хост HTTPS"
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
    description: "Изолированные Lua-расширения, установленные на уровне агентства. Каждый вызов выполняется в новом состоянии Lua."
    install: "Установить плагин"
    reviewBundle: "Проверить пакет плагина"
    reviewBundleHint: "Проверьте манифест и запрашиваемые разрешения, прежде чем устанавливать плагин для всей организации."
    reviewNotes: "Заметки о проверке"
    permissions: "Разрешения"
    noPermissions: "Этот пакет не запрашивает сетевого доступа, маршрутов, событий, инструментов или поверхностей настроек."
    review: "Проверить"
    empty: "Плагины не установлены"
    emptyHint: "Загрузите пакет, чтобы установить первый плагин. Каждый вызов выполняется в новом состоянии Lua."
    versionRuntimeEntry: "Версия {version} · среда {runtime} · точка входа {entry}"
    outboundHosts: "Исходящие хосты: {hosts}"
    actorLine: "Субъект плагина: {'@'}{username} (id {id})"
    servedUnder: "Доступен по пути /plugins/{id}/. Вебхук провайдера может вызывать этот маршрут."
    fileSize: "{name} · {size} КиБ"
    previewMeta: "id {id} · среда {runtime} · точка входа {entry} · sha256 {checksum}"
    handlers: "обработчиков: {count}"
    toolsCount: "инструментов: {count}"
    surfacesCount: "поверхностей: {count}"
</i18n>
