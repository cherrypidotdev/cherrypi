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
      CardTitle {{ $t("agentModels.title") }}
      CardDescription
        | {{ $t("agentModels.description") }}
    CardContent.flex.flex-col.gap-4
      Alert(v-if="config && !config.sealing")
        AlertTitle {{ $t("agentModels.noSealingTitle") }}
        AlertDescription
          | {{ $t("agentModels.noSealingDescription") }}
      p.text-destructive.text-sm(v-if="error" role="alert") {{ error }}
      p.text-muted-foreground.text-sm(v-if="loading") {{ $t("common.loading") }}

      .flex.flex-col.gap-3
        .flex.items-start.justify-between.gap-3.rounded-md.border.p-3(
          v-for="provider in config?.providers || []"
          :key="provider.slug"
        )
          .flex.flex-col.gap-1
            .flex.flex-wrap.items-center.gap-2
              span.font-medium {{ provider.name || provider.slug }}
              Badge(variant="outline") {{ apiLabel(provider.api) }}
              Badge(v-if="!provider.enabled" variant="secondary")
                | {{ $t("agentModels.disabled") }}
              Badge(v-if="provider.hasKey" variant="default")
                | {{ $t("agentModels.keyStored") }}
              Badge(v-else variant="destructive") {{ $t("agentModels.noKey") }}
              Badge(
                v-if="!provider.models.length"
                variant="destructive"
                data-test="provider-no-models"
              ) {{ $t("agentModels.noModels") }}
            p.text-muted-foreground.text-xs
              | {{ $t("agentModels.providerSummary", { url: provider.baseUrl, count: provider.models.length }) }}
            .flex.flex-wrap.items-center.gap-1
              Badge(
                v-for="grant in provider.grantees"
                :key="grant.kind + grant.id"
                variant="secondary"
              )
                | {{ scopeLabel(grant) }}
          .flex.items-center.gap-2
            Button(
              variant="outline"
              size="sm"
              @click="test(provider.slug)"
              :disabled="busy || testing === provider.slug"
            )
              Spinner(
                v-if="testing === provider.slug"
                data-icon="inline-start"
              )
              span {{ $t("agentModels.test") }}
            Button(as-child variant="outline" size="sm")
              RouterLink(
                :to="{ name: 'provider-edit', params: { slug: provider.slug } }"
              ) {{ $t("common.edit") }}
            Button(
              variant="destructive"
              size="sm"
              @click="remove(provider.slug)"
              :disabled="busy"
            ) {{ $t("common.delete") }}
      p.text-muted-foreground.text-sm(
        v-if="!loading && !(config?.providers || []).length"
      )
        | {{ $t("agentModels.empty") }}

      Button(as-child variant="outline" size="sm")
        RouterLink(:to="{ name: 'provider-create' }")
          | {{ $t("agentModels.addProvider") }}

  Card
    CardHeader
      CardTitle {{ $t("agentModels.selection.title") }}
      CardDescription {{ $t("agentModels.selection.description") }}
    CardContent
      ol.text-muted-foreground.flex.list-decimal.flex-col.gap-2.pl-5.text-sm
        li {{ $t("agentModels.selection.explicit") }}
        li {{ $t("agentModels.selection.agentDefault") }}
        li {{ $t("agentModels.selection.firstAvailable") }}
</template>

<script setup lang="ts">
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { useAgentGroups } from "@/composables/useAgentGroups";
import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { useUsers } from "@/composables/useUsers";
import {
  agentConfigApi,
  SCOPE_KINDS,
  WIRE_APIS,
  type AgentConfig,
  type ModelScope,
} from "@/lib/agentConfig";
import { teamApi, type Team } from "@/lib/team";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const config = ref<AgentConfig | null>(null);
const loading = ref(true);
const busy = ref(false);
const error = ref("");
const testing = ref("");
const teams = ref<Team[]>([]);

const { projects, loadProjects } = useProjects();
const { groups, loadAgentGroups } = useAgentGroups();
const { agents, loadAgents } = useAgents();
const { users, loadUsers } = useUsers();

function apiLabel(api: string): string {
  return WIRE_APIS.find((entry) => entry.value === api)?.label ?? api;
}

function targetName(kind: string, id: string): string {
  switch (kind) {
    case "project":
      return projects.value.find((project) => project.id === id)?.name ?? id;
    case "team":
      return teams.value.find((team) => String(team.id) === id)?.name ?? id;
    case "group":
      return groups.value.find((group) => String(group.id) === id)?.name ?? id;
    case "agent": {
      const agent = agents.value.find((item) => item.actorId === id);
      if (agent) return agent.displayName || agent.username;
      const user = users.value.find((item) => String(item.id) === id);
      return user?.name || user?.username || id;
    }
    default:
      return id;
  }
}

function scopeLabel(grant: ModelScope): string {
  const kind = SCOPE_KINDS.find((entry) => entry.value === grant.kind);
  const target =
    grant.kind === "org"
      ? (kind?.label ?? t("agentModels.everyone"))
      : `${kind?.label ?? grant.kind}: ${targetName(grant.kind, grant.id)}`;
  return grant.modelId ? `${target} · ${grant.modelId}` : target;
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    config.value = await agentConfigApi.get();
  } catch (err) {
    error.value = String(err);
  } finally {
    loading.value = false;
  }
}

async function test(slug: string) {
  testing.value = slug;
  try {
    const result = await agentConfigApi.testProvider(slug);
    const detail = result.latencyMs ? ` (${result.latencyMs} ms)` : "";
    if (result.ok) {
      toast.success(`${result.message}${detail}`);
    } else {
      toast.error(`${result.message}${detail}`);
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    testing.value = "";
  }
}

async function remove(slug: string) {
  busy.value = true;
  try {
    await agentConfigApi.deleteProvider(slug);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    await load();
    busy.value = false;
  }
}

onMounted(async () => {
  await load();
  await Promise.all([
    loadProjects(),
    loadAgentGroups(),
    loadAgents(),
    loadUsers(),
    teamApi
      .list()
      .then((payload) => {
        teams.value = payload.teams;
      })
      .catch(() => {
        teams.value = [];
      }),
  ]);
});
</script>

<i18n lang="yaml">
en:
  agentModels:
    title: "Models"
    description: "Providers the agency holds keys for. Every request goes through the agency, which attaches the key: an agent never receives it, and these are the only models its tasks can choose."
    noSealingTitle: "Provider keys cannot be stored"
    noSealingDescription: "Agency-held provider keys are disabled because CHERRYPI_SECRET_KEY is not set. Set it to a base64 value that decodes to exactly 32 bytes (for example openssl rand -base64 32) to enable them; a malformed or wrong-length value stops the agency from starting. Until then no agent offers any model."
    disabled: "disabled"
    keyStored: "key stored"
    noKey: "no key"
    noModels: "no models configured"
    everyone: "Everyone"
    providerSummary: "{url} · {count} model(s)"
    empty: "No providers declared yet, so agents offer no models."
    addProvider: "Add a provider"
    test: "Test"
    selection:
      title: "How a task picks a model"
      description: "The picker lists the models the agency declares, narrowed by grants. These are the only models an agent can run: it holds no model configuration of its own."
      explicit: "A model override on the task wins."
      agentDefault: "Otherwise the agent's default model in Agent settings applies."
      firstAvailable: "If neither is set, the first model granted to the task is used."
ru:
  agentModels:
    title: "Модели"
    description: "Провайдеры, ключи которых хранит agency. Каждый запрос идёт через agency, которое подставляет ключ: агент его никогда не получает, и это единственные модели, доступные его задачам."
    noSealingTitle: "Ключи провайдеров нельзя сохранить"
    noSealingDescription: "Ключи провайдеров в agency отключены, потому что CHERRYPI_SECRET_KEY не задан. Задайте base64-значение, которое декодируется ровно в 32 байта (например, openssl rand -base64 32), чтобы включить их; некорректное значение или неверная длина не даст agency запуститься. До этого агенты не предлагают ни одной модели."
    disabled: "отключён"
    keyStored: "ключ сохранён"
    noKey: "нет ключа"
    noModels: "модели не настроены"
    everyone: "Все"
    providerSummary: "{url} · моделей: {count}"
    empty: "Провайдеры ещё не объявлены, поэтому агенты не предлагают модели."
    addProvider: "Добавить провайдера"
    test: "Проверить"
    selection:
      title: "Как задача выбирает модель"
      description: "Выбор показывает модели, объявленные в agency, с учётом грантов. Это единственные модели, доступные агенту: собственной конфигурации моделей у него нет."
      explicit: "Явно выбранная в задаче модель имеет приоритет."
      agentDefault: "Иначе применяется модель агента по умолчанию из настроек агента."
      firstAvailable: "Если не задано ни то, ни другое, берётся первая модель, выданная задаче."
</i18n>
