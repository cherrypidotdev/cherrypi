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
        SelectItem(value="all") All levels
        SelectItem(
          v-for="level in PLUGIN_LOG_LEVELS"
          :key="level"
          :value="level"
        ) {{ PLUGIN_LOG_LEVEL_LABELS[level] }} and above
    Input.flex-1(
      v-model="search"
      placeholder="Filter messages…"
      aria-label="Filter runtime log messages"
      @keyup.enter="load"
    )
    Button(variant="outline" size="sm" @click="load" :disabled="loading")
      Spinner(v-if="loading" data-icon="inline-start")
      RefreshCwIcon(v-else data-icon="inline-start")
      span Refresh
  p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
  .max-h-80.overflow-auto.rounded-md.border(v-if="logs.length")
    Table
      TableHeader
        TableRow
          TableHead.w-28 Time
          TableHead.w-20 Level
          TableHead Message
          TableHead.w-40 Surfaces
      TableBody
        TableRow(v-for="line in logs" :key="String(line.id)")
          TableCell.text-xs.whitespace-nowrap {{ formatTime(line.at) }}
          TableCell
            Badge(:variant="levelVariant(line.level)") {{ line.level }}
          TableCell.text-xs.break-all {{ line.message }}
          TableCell.text-muted-foreground.text-xs
            | {{ surfaceText(line) || "—" }}
  p.text-muted-foreground.text-sm(v-else-if="!loading")
    | No runtime log lines{{ scope ? " for this surface" : "" }} yet.
</template>

<script setup lang="ts">
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
import {
  PLUGIN_LOG_LEVELS,
  PLUGIN_LOG_LEVEL_LABELS,
  PLUGIN_SCOPE_LABELS,
  type PluginScope,
} from "@/lib/plugins";
import { agency } from "@/lib/sdk";
import type { PluginLog } from "@cherrypi/sdk";
import { RefreshCwIcon } from "@lucide/vue";
import { onMounted, ref, watch } from "vue";

const props = defineProps<{
  pluginId: string;
  scope?: string;
  targetId?: string;
}>();

const logs = ref<PluginLog[]>([]);
const loading = ref(false);
const error = ref("");
const minLevel = ref("all");
const search = ref("");

onMounted(() => void load());
watch(
  () => [props.pluginId, props.scope, props.targetId],
  () => void load(),
);

function setLevel(value: unknown) {
  minLevel.value = String(value ?? "");
  void load();
}

async function load() {
  if (!props.pluginId) return;
  loading.value = true;
  error.value = "";
  try {
    logs.value = await agency.listPluginRuntimeLogs(props.pluginId, {
      scope: props.scope ?? "",
      targetId: props.targetId ?? "",
      minLevel: minLevel.value === "all" ? "" : minLevel.value,
      search: search.value,
      limit: 200,
    });
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
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

function surfaceText(line: PluginLog): string {
  return (line.surfaces ?? [])
    .map((surface) => {
      const scope = surface.scope as PluginScope;
      const label = PLUGIN_SCOPE_LABELS[scope] ?? surface.scope;
      return surface.targetId ? `${label} ${surface.targetId}` : label;
    })
    .join(", ");
}

function formatTime(value: PluginLog["at"]): string {
  if (!value) return "—";
  return new Date(
    Number(value.seconds) * 1000 + Math.floor(value.nanos / 1_000_000),
  ).toLocaleTimeString();
}
</script>
