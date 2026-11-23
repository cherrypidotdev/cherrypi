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
.flex.flex-col.gap-2(data-testid="task-integrations")
  p.text-muted-foreground.text-xs(v-if="loading") {{ $t("taskIntegrations.loading") }}
  p.text-destructive.text-xs(v-else-if="error") {{ error }}

  template(v-else)
    p.text-muted-foreground.text-xs(v-if="!recorded")
      | {{ $t("taskIntegrations.notRecorded") }}

    Disclosure(v-slot="{ open }" :default-open="true")
      DisclosureButton.group.flex.w-full.items-center.gap-1.text-left
        ChevronRightIcon.size-3.shrink-0.transition-transform(
          :class="open ? 'rotate-90' : ''"
        )
        span.text-muted-foreground.text-xs.font-medium {{ $t("taskIntegrations.skills") }}
        span.text-muted-foreground.text-xs.tabular-nums {{ report.skills.length }}
      DisclosurePanel.mt-2.flex.flex-col.gap-1
        p.text-muted-foreground.text-xs(v-if="!report.skills.length")
          | {{ $t("taskIntegrations.noSkills") }}
        .flex.items-start.gap-2(
          v-for="skill in report.skills"
          :key="`skill-${skill.name}`"
        )
          FileTextIcon.text-muted-foreground.size-3.shrink-0(class="mt-0.5")
          .flex.min-w-0.flex-1.flex-col
            .flex.flex-wrap.items-center.gap-1
              span.text-xs.font-medium {{ skill.name }}
              Badge(variant="outline") {{ originLabel(skill.origin, skill.owner) }}
              Badge(v-if="skill.disableModelInvocation" variant="secondary")
                | {{ $t("taskIntegrations.manual") }}
            span.text-muted-foreground.truncate.text-xs {{ skill.description }}

    Disclosure(v-slot="{ open }" :default-open="true")
      DisclosureButton.group.flex.w-full.items-center.gap-1.text-left
        ChevronRightIcon.size-3.shrink-0.transition-transform(
          :class="open ? 'rotate-90' : ''"
        )
        span.text-muted-foreground.text-xs.font-medium {{ $t("taskIntegrations.servers") }}
        span.text-muted-foreground.text-xs.tabular-nums {{ report.servers.length }}
      DisclosurePanel.mt-2.flex.flex-col.gap-1
        p.text-muted-foreground.text-xs(v-if="!report.servers.length")
          | {{ $t("taskIntegrations.noServers") }}
        .flex.items-start.gap-2(
          v-for="server in report.servers"
          :key="`server-${server.name}`"
        )
          span.mt-1.size-2.shrink-0.rounded-full(
            :class="server.connected ? 'bg-emerald-500' : 'bg-destructive'"
            :title="server.connected ? $t('taskIntegrations.connected') : $t('taskIntegrations.failed')"
          )
          .flex.min-w-0.flex-1.flex-col
            .flex.flex-wrap.items-center.gap-1
              span.text-xs.font-medium {{ server.name }}
              Badge(variant="outline") {{ originLabel(server.origin, server.owner) }}
              Badge(v-if="server.transport" variant="secondary")
                | {{ transportLabel(server.transport) }}
              Badge(v-if="server.connected" variant="secondary")
                | {{ $t("taskIntegrations.toolCount", { count: server.toolCount }) }}
            span.text-muted-foreground.truncate.text-xs(v-if="server.error") {{ server.error }}
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import {
  integrationsApi,
  scopeLabel,
  type TaskIntegrationReport,
} from "@/lib/integrations";
import { Disclosure, DisclosureButton, DisclosurePanel } from "@headlessui/vue";
import { ChevronRightIcon, FileTextIcon } from "@lucide/vue";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  taskId: string;

  preview?: TaskIntegrationReport;
}>();

const loading = ref(true);
const error = ref("");
const recorded = ref(false);
const report = ref<TaskIntegrationReport["report"]>({
  skills: [],
  servers: [],
});

function transportLabel(transport: string): string {
  if (transport === "agency") return t("taskIntegrations.transportAgency");
  return transport;
}

function originLabel(origin: string, owner: string): string {
  if (origin === "host") return t("taskIntegrations.hostOrigin");
  return owner ? scopeLabel(owner) : t("taskIntegrations.agencyOrigin");
}

onMounted(async () => {
  if (props.preview) {
    recorded.value = props.preview.recorded;
    report.value = props.preview.report;
    loading.value = false;
    return;
  }
  try {
    const result = await integrationsApi.taskReport(props.taskId);
    recorded.value = result.recorded;
    report.value = result.report;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
});
</script>

<i18n lang="yaml">
en:
  taskIntegrations:
    loading: "Loading MCP servers and skills…"
    notRecorded: "MCP servers and skills are recorded when the task's first run starts."
    skills: "Skills"
    servers: "MCP servers"
    noSkills: "No skills applied to this run."
    noServers: "No MCP servers applied to this run."
    manual: "manual only"
    connected: "Connected"
    failed: "Failed to connect"
    toolCount: "{count} tools"
    transportAgency: "via agency"
    hostOrigin: "Agent host"
    agencyOrigin: "Agency"
ru:
  taskIntegrations:
    loading: "Загружаем MCP-серверы и навыки…"
    notRecorded: "MCP-серверы и навыки записываются при первом запуске задачи."
    skills: "Навыки"
    servers: "MCP-серверы"
    noSkills: "К этому запуску не применялось ни одного навыка."
    noServers: "К этому запуску не применялось ни одного MCP-сервера."
    manual: "только вручную"
    connected: "Подключён"
    failed: "Не удалось подключиться"
    toolCount: "{count} инструментов"
    transportAgency: "через агентство"
    hostOrigin: "Хост агента"
    agencyOrigin: "Агентство"
</i18n>
