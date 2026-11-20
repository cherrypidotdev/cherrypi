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
.mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
  .flex.flex-col.gap-1
    h1.text-lg.font-semibold {{ $t("nav.orgSettings") }}
    p.text-muted-foreground.text-sm {{ $t("settings.subtitle") }}

  Card(v-if="!isAdmin")
    CardHeader
      CardTitle {{ $t("settings.adminOnly.title") }}
      CardDescription
        | {{ $t("settings.adminOnly.description") }}
  template(v-else)
    Tabs(:model-value="activeTab" @update:model-value="onTabChange")
      TabsList
        TabsTrigger(
          v-for="tab in tabs"
          :key="tab.value"
          :value="tab.value"
          as-child
        )
          RouterLink(:to="{ name: tab.name }") {{ $t(tab.labelKey) }}

    RouterView(v-slot="{ Component }")
      component(:is="Component")
</template>

<script setup lang="ts">
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { currentAccount } from "@/lib/account";
import { computed } from "vue";
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const isAdmin = computed(() => currentAccount.value?.is_admin ?? false);

const tabs = [
  { value: "general", name: "settings", labelKey: "settings.tabs.general" },
  {
    value: "models",
    name: "settings-models",
    labelKey: "settings.tabs.models",
  },
  {
    value: "plugins",
    name: "settings-plugins",
    labelKey: "account.tabs.plugins",
  },
  { value: "auth", name: "settings-auth", labelKey: "account.tabs.auth" },
  {
    value: "integrations",
    name: "settings-integrations",
    labelKey: "settings.tabs.integrations",
  },
  {
    value: "variables",
    name: "settings-variables",
    labelKey: "variables.title",
  },
  { value: "admin", name: "settings-admin", labelKey: "settings.tabs.admin" },
  { value: "logs", name: "settings-logs", labelKey: "settings.tabs.logs" },
];

const activeTab = computed(() => {
  const name = String(route.name ?? "");
  switch (name) {
    case "settings-models":
      return "models";
    case "settings-plugins":
    case "settings-plugin":
      return "plugins";
    case "settings-auth":
      return "auth";
    case "settings-integrations":
      return "integrations";
    case "settings-variables":
      return "variables";
    case "settings-admin":
      return "admin";
    case "settings-logs":
      return "logs";
    default:
      return "general";
  }
});

function onTabChange(value: unknown) {
  const tab = tabs.find((candidate) => candidate.value === value);
  if (!tab) return;
  void router.push({ name: tab.name });
}
</script>

<i18n lang="yaml">
en:
  settings:
    tabs:
      general: "General"
      models: "Models"
      integrations: "Integrations"
      admin: "Admin"
      logs: "Logs"
    subtitle: "Agency-wide configuration. Only administrators can open this page."
    adminOnly:
      title: "Administrators only"
      description: "Ask an administrator if you need an agency-wide setting changed."
ru:
  settings:
    tabs:
      general: "Общие"
      models: "Модели"
      integrations: "Интеграции"
      admin: "Администрирование"
      logs: "Журналы"
    subtitle: "Настройки агентства. Эту страницу могут открывать только администраторы."
    adminOnly:
      title: "Только для администраторов"
      description: "Попросите администратора изменить настройку агентства."
</i18n>
