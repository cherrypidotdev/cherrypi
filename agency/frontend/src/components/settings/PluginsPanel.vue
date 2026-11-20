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
            h3.flex.items-center.gap-2.text-base.font-semibold
              PluginIcon(:src="preview?.iconDataUrl" size="size-5")
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
            p.text-muted-foreground.text-xs(v-if="!previewPermissions.length")
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
          PluginIcon(:src="item.plugin.iconUrl")
          RouterLink.text-primary.px-0.text-left(
            :to="{ name: 'settings-plugin', params: { pluginId: item.id } }"
          ) {{ item.plugin.name }}
          Badge(:variant="item.plugin.enabled ? 'default' : 'secondary'")
            | {{ item.plugin.enabled ? "active" : "disabled" }}
        CardDescription {{ item.plugin.description || "No description" }}
        p.text-destructive.text-xs(v-if="item.plugin.loadError" role="alert")
          | {{ $t("plugins.loadErrorHint") }}
          | {{ item.plugin.loadError }}
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
        RouterLink(
          :to="{ name: 'settings-plugin', params: { pluginId: item.id } }"
        )
          Button(variant="outline" size="sm")
            | {{ $t("common.open") }}
        Button(
          variant="ghost"
          size="sm"
          @click="toggle(item.plugin)"
          :disabled="busy === item.id"
        ) {{ item.plugin.enabled ? $t("plugins.disable") : $t("plugins.enable") }}

  Card(v-else-if="!loadingPlugins")
    CardHeader
      CardTitle {{ $t("plugins.empty") }}
      CardDescription
        | {{ $t("plugins.emptyHint") }}
</template>

<script setup lang="ts">
import PluginIcon from "@/components/plugins/PluginIcon.vue";
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
  PLUGIN_SCOPE_KEYS,
  allowsAnyHost,
  handlerLabel,
  manifestSurfaces,
  parseManifest,
  parseManifestJSON,
  pluginPermissions,
  type PluginManifest,
  type PluginScope,
} from "@/lib/plugins";
import { agency } from "@/lib/sdk";
import type { Plugin, PluginPreview } from "@cherrypi/sdk";
import { PlusIcon, SearchIcon, UploadIcon } from "@lucide/vue";
import { computed, onMounted, ref } from "vue";
import { RouterLink, useRouter } from "vue-router";

interface PluginCard {
  plugin: Plugin;
  id: string;
  handlers: string[];
  tools: unknown[];
  surfaces: PluginScope[];
  anyHost: boolean;
}

const router = useRouter();

const plugins = ref<PluginCard[]>([]);
const loadingPlugins = ref(false);
const busy = ref("");
const error = ref("");
const installOpen = ref(false);
const file = ref<File | null>(null);
const selectedFile = ref<File | null>(null);
const installing = ref(false);
const inspecting = ref(false);
const preview = ref<PluginPreview | null>(null);
const bundleBytes = ref<Uint8Array | null>(null);

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

onMounted(() => void loadPlugins());

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

async function toggle(item: Plugin) {
  busy.value = item.id;
  error.value = "";
  try {
    await agency.setPluginEnabled(item.id, !item.enabled);
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
    await router.push({
      name: "settings-plugin",
      params: { pluginId: installed.id },
    });
  } catch (err) {
    error.value = message(err);
  } finally {
    installing.value = false;
  }
}

function shortChecksum(value: string): string {
  return value ? value.slice(0, 12) : "—";
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
</script>

<i18n lang="yaml">
en:
  plugins:
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
    fileSize: "{name} · {size} KiB"
    previewMeta: "id {id} · runtime {runtime} · entry {entry} · sha256 {checksum}"
    handlers: "{count} handlers"
    toolsCount: "{count} tools"
    surfacesCount: "{count} surfaces"
    anyHost: "Any HTTPS host"
    enable: "Enable"
    disable: "Disable"
    loadErrorHint: "This plugin could not be loaded and is disabled. Fix the cause, then enable it again:"
ru:
  plugins:
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
    fileSize: "{name} · {size} КиБ"
    previewMeta: "id {id} · среда {runtime} · точка входа {entry} · sha256 {checksum}"
    handlers: "обработчиков: {count}"
    toolsCount: "инструментов: {count}"
    surfacesCount: "поверхностей: {count}"
    anyHost: "Любой хост HTTPS"
    enable: "Включить"
    disable: "Отключить"
    loadErrorHint: "Плагин не удалось загрузить, поэтому он отключён. Устраните причину и включите его снова:"
</i18n>
