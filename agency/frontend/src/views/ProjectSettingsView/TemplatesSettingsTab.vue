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
Card
  CardHeader
    CardTitle {{ $t("projectSettings.tabs.templates") }}
    CardDescription
      | {{ $t("projectSettings.templates.description") }}
  CardContent
    TaskTemplatesPanel(:project-id="page.project?.id ?? ''")
</template>

<script setup lang="ts">
import TaskTemplatesPanel from "@/components/templates/TaskTemplatesPanel.vue";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { ProjectPageState, ProjectSettingsState } from "@/lib/projectPage";

defineProps<{ page: ProjectPageState; settings: ProjectSettingsState }>();
</script>

<i18n lang="yaml">
en:
  projectSettings:
    tabs:
      templates: "Templates"
    templates:
      description: "Reusable Jinja prompts saved per project. A task created from a template records the version and the argument values it used."
ru:
  projectSettings:
    tabs:
      templates: "Шаблоны"
    templates:
      description: "Многоразовые Jinja-промпты, сохранённые для проекта. Созданная из шаблона задача запоминает версию и значения аргументов."
</i18n>
