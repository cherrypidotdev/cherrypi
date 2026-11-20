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
    CardContent
      p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
      p.text-muted-foreground.text-sm(v-if="loading") {{ $t("common.loading") }}
      p.text-muted-foreground.text-sm(v-else-if="!rows.length")
        | {{ $t("projectPlugins.empty") }}

  Card(v-for="row in rows" :key="row.plugin?.id")
    CardHeader.flex.flex-row.items-start.justify-between.gap-4
      .flex.flex-col.gap-1
        CardTitle.flex.flex-wrap.items-center.gap-2
          PlugIcon.size-4
          RouterLink.text-primary(
            :to="{ name: 'project-settings-plugin', params: { slug, pluginId: row.plugin?.id } }"
          ) {{ row.plugin?.name }}
          Badge(v-if="row.member" variant="outline") {{ $t("projectPlugins.hasAccess") }}
          Badge(v-else variant="secondary") {{ $t("projectPlugins.noAccess") }}
          Badge(v-if="row.enabled" variant="default") {{ $t("agentPlugins.enabled") }}
          Badge(v-else variant="secondary") {{ $t("settings.auth.disabled") }}
        CardDescription {{ row.plugin?.description || "No description" }}
      RouterLink(
        :to="{ name: 'project-settings-plugin', params: { slug, pluginId: row.plugin?.id } }"
      )
        Button(variant="outline" size="sm") {{ $t("common.open") }}
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { agency } from "@/lib/sdk";
import type { ProjectPlugin } from "@cherrypi/sdk";
import { PlugIcon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { RouterLink, useRoute } from "vue-router";

const props = defineProps<{ projectId: string }>();

const route = useRoute();
const slug = computed(() => String(route.params.slug ?? ""));

const rows = ref<ProjectPlugin[]>([]);
const loading = ref(false);
const error = ref("");

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

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
</script>

<i18n lang="yaml">
en:
  projectPlugins:
    title: "Project plugins"
    description: "A plugin works in this project only while one of its owners has it enabled here; disabling it removes its access and keeps its settings and logs. Open a plugin to enable it, cap what it may create, and edit its project-scoped settings."
    empty: "No plugin declares a project settings surface."
    hasAccess: "has access"
    noAccess: "no access"
ru:
  projectPlugins:
    title: "Плагины проекта"
    description: "Плагин работает в этом проекте, только пока один из его владельцев включил его здесь; отключение лишает доступа, но сохраняет настройки и журналы. Откройте плагин, чтобы включить его, ограничить создаваемые им задачи и изменить его настройки уровня проекта."
    empty: "Ни один плагин не объявляет поверхность настроек проекта."
    hasAccess: "есть доступ"
    noAccess: "нет доступа"
</i18n>
