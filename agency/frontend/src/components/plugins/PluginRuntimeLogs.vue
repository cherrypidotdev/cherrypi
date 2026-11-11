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
.flex.flex-col.gap-3
  .flex.flex-wrap.items-center.gap-2
    Select(:model-value="minLevel" @update:model-value="setLevel")
      SelectTrigger.w-36
        SelectValue
      SelectContent
        SelectItem(value="all") {{ $t("pluginLog.allLevels") }}
        SelectItem(
          v-for="level in PLUGIN_LOG_LEVELS"
          :key="level"
          :value="level"
        ) {{ $t("pluginLog.andAbove", { level: $t(PLUGIN_LOG_LEVEL_KEYS[level]) }) }}
    Input.flex-1(
      v-model="search"
      :placeholder="$t('pluginLog.filterPlaceholder')"
      :aria-label="$t('pluginLog.filterAria')"
      @keyup.enter="reload"
    )
    Button(variant="outline" size="sm" @click="reload" :disabled="loading")
      Spinner(v-if="loading" data-icon="inline-start")
      RefreshCwIcon(v-else data-icon="inline-start")
      span {{ $t("common.refresh") }}
  p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
  .max-h-80.overflow-auto.rounded-md.border(v-if="logs.length")
    Table
      TableHeader
        TableRow
          TableHead.w-28 {{ $t("plugins.column.time") }}
          TableHead.w-20 {{ $t("plugins.column.level") }}
          TableHead {{ $t("plugins.column.message") }}
          TableHead.w-40 {{ $t("pluginLog.surfaces") }}
      TableBody
        TableRow(v-for="line in logs" :key="String(line.id)")
          TableCell.text-xs.whitespace-nowrap {{ formatTime(line.at) }}
          TableCell
            Badge(:variant="levelVariant(line.level)") {{ line.level }}
          TableCell.text-xs.break-all
            p {{ line.message }}
            PluginLogData(:data-json="line.dataJson")
          TableCell.text-xs
            .flex.flex-wrap.gap-1(v-if="surfaceEntries(line).length")
              template(
                v-for="surface in surfaceEntries(line)"
                :key="surface.key"
              )
                RouterLink.surface-chip(
                  v-if="surface.to"
                  :to="surface.to"
                  :title="surface.title"
                ) {{ surface.label }}
                span.surface-chip(v-else :title="surface.title") {{ surface.label }}
            span.text-muted-foreground(v-else) —
    PluginLogLoadMore(
      :has-more="hasMore"
      :auto="auto"
      :loaded="loaded"
      @load="loadMore"
      :loading="loadingMore"
    )
  p.text-muted-foreground.text-sm(v-else-if="!loading && !error")
    | {{ scope ? $t("pluginLog.emptyScoped") : $t("pluginLog.empty") }}
</template>

<script setup lang="ts">
import PluginLogData from "@/components/plugins/PluginLogData.vue";
import PluginLogLoadMore from "@/components/plugins/PluginLogLoadMore.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAgents } from "@/composables/useAgents";
import { usePluginLogPager } from "@/composables/usePluginLogPager";
import { useProjects } from "@/composables/useProjects";
import { useUsers } from "@/composables/useUsers";
import { currentAccount } from "@/lib/account";
import { PLUGIN_LOG_LEVELS, PLUGIN_LOG_LEVEL_KEYS } from "@/lib/plugins";
import { agency } from "@/lib/sdk";
import type { PluginLog } from "@cherrypi/sdk";
import { RefreshCwIcon } from "@lucide/vue";
import { onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { RouteLocationRaw } from "vue-router";

const props = defineProps<{
  pluginId: string;
  scope?: string;
  targetId?: string;
}>();

interface SurfaceChip {
  key: string;
  label: string;
  to?: RouteLocationRaw;
  title?: string;
}

const { projects, loadProjects } = useProjects();
const { agents, loadAgents } = useAgents();
const { users, loadUsers } = useUsers();

const minLevel = ref("all");
const search = ref("");

const {
  logs,
  loading,
  loadingMore,
  error,
  hasMore,
  auto,
  loaded,
  load,
  loadMore,
  reset,
} = usePluginLogPager((beforeId, limit) =>
  agency.listPluginRuntimeLogs(props.pluginId, {
    scope: props.scope ?? "",
    targetId: props.targetId ?? "",
    minLevel: minLevel.value === "all" ? "" : minLevel.value,
    search: search.value,
    limit,
    beforeId,
  }),
);

onMounted(() => {
  void loadProjects();
  void loadAgents();
  void loadUsers();
  void load();
});

watch(
  () => [props.pluginId, props.scope, props.targetId],
  () => {
    reset();
    void load();
  },
);

function reload() {
  reset();
  void load();
}

function setLevel(value: unknown) {
  minLevel.value = String(value ?? "");
  reload();
}

function levelVariant(
  level: string,
): "default" | "secondary" | "destructive" | "outline" {
  switch (level) {
    case "error":
      return "destructive";
    case "warn":
      return "outline";
    case "debug":
    case "trace":
      return "secondary";
    default:
      return "default";
  }
}

function isCurrentSurface(scope: string, targetId: string): boolean {
  if (!props.scope) return false;
  return scope === props.scope && targetId === props.targetId;
}

function surfaceEntries(line: PluginLog): SurfaceChip[] {
  return (line.surfaces ?? [])
    .filter((surface) => !isCurrentSurface(surface.scope, surface.targetId))
    .map((surface) => surfaceChip(surface.scope, surface.targetId));
}

function surfaceChip(scope: string, targetId: string): SurfaceChip {
  const key = `${scope}:${targetId}`;
  if (scope === "org") {
    return {
      key,
      label: "Organization",
      title: t("nav.orgSettings"),
      to: currentAccount.value?.is_admin
        ? { name: "settings", query: { tab: "plugins" } }
        : undefined,
    };
  }
  if (scope === "project") {
    const project = projects.value.find((item) => item.id === targetId);
    return {
      key,
      label: project?.name || `Project ${targetId}`,
      title: project?.slug ? `Project ${project.slug}` : undefined,
      to: project
        ? {
            name: "project",
            params: { slug: project.slug },
            query: { tab: "settings", section: "plugins" },
          }
        : undefined,
    };
  }
  if (scope === "agent") {
    const agent = agents.value.find((item) => item.actorId === targetId);
    return {
      key,
      label: agent?.displayName || agent?.username || `Agent ${targetId}`,
      title: agent ? `Agent @${agent.username}` : undefined,
      to: agent
        ? {
            name: "agent",
            params: { username: agent.username },
            query: { tab: "settings", section: "plugins" },
          }
        : undefined,
    };
  }
  if (scope === "user") {
    const user = users.value.find((item) => String(item.id) === targetId);
    const isSelf = String(currentAccount.value?.id ?? "") === targetId;
    return {
      key,
      label: user?.name || (user ? `@${user.username}` : `User ${targetId}`),
      title: user ? `User @${user.username}` : undefined,
      to: isSelf
        ? { name: "account", query: { tab: "plugins" } }
        : user
          ? { name: "user", params: { username: user.username } }
          : undefined,
    };
  }
  return { key, label: `${scope} ${targetId}` };
}

function formatTime(value: PluginLog["at"]): string {
  if (!value) return "—";
  return new Date(
    Number(value.seconds) * 1000 + Math.floor(value.nanos / 1_000_000),
  ).toLocaleTimeString();
}

const { t } = useI18n();
</script>

<style scoped>
.surface-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  border: 1px solid var(--border-color);
  border-radius: 0.375rem;
  padding: 0.0625rem 0.375rem;
  font-size: 0.75rem;
  line-height: 1.25rem;
  color: var(--foreground);
  text-decoration: none;
  transition:
    background-color 120ms ease,
    color 120ms ease;
}

a.surface-chip:hover {
  background-color: var(--muted);
  color: var(--muted-foreground);
}
</style>

<i18n lang="yaml">
en:
  pluginLog:
    allLevels: "All levels"
    surfaces: "Surfaces"
    filterPlaceholder: "Filter messages…"
    filterAria: "Filter runtime log messages"
    andAbove: "{level} and above"
    empty: "No runtime log lines yet."
    emptyScoped: "No runtime log lines for this surface yet."
    level:
      trace: "Trace"
      debug: "Debug"
      info: "Info"
      warn: "Warn"
      error: "Error"
ru:
  pluginLog:
    allLevels: "Все уровни"
    surfaces: "Поверхности"
    filterPlaceholder: "Фильтр сообщений…"
    filterAria: "Фильтр сообщений журнала выполнения"
    andAbove: "{level} и выше"
    empty: "Записей журнала выполнения пока нет."
    emptyScoped: "Записей журнала выполнения для этой поверхности пока нет."
    level:
      trace: "Trace"
      debug: "Debug"
      info: "Info"
      warn: "Warn"
      error: "Error"
</i18n>
