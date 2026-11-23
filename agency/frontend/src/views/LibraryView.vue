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
.mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6(data-testid="library-view")
  .flex.flex-col.gap-1
    h1.text-lg.font-semibold {{ $t("libraryPage.title") }}
    p.text-muted-foreground.text-sm {{ $t("libraryPage.subtitle") }}

  Card
    CardHeader
      CardTitle {{ $t("libraryPage.how.title") }}
      CardDescription {{ $t("libraryPage.how.description") }}
    CardContent.flex.flex-col.gap-2
      .flex.items-start.gap-2.text-sm
        span.text-muted-foreground.font-mono 1
        span {{ $t("libraryPage.how.own") }}
      .flex.items-start.gap-2.text-sm
        span.text-muted-foreground.font-mono 2
        span {{ $t("libraryPage.how.install") }}
      .flex.items-start.gap-2.text-sm
        span.text-muted-foreground.font-mono 3
        span {{ $t("libraryPage.how.resolve") }}
      .flex.items-start.gap-2.text-sm
        span.text-muted-foreground.font-mono 4
        span {{ $t("libraryPage.how.fuse") }}

  IntegrationsPanel(data-testid="library-panel")
</template>

<script setup lang="ts">
import IntegrationsPanel from "@/components/integrations/IntegrationsPanel.vue";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
</script>

<i18n lang="yaml">
en:
  libraryPage:
    title: "MCP & Skills Library"
    subtitle: "Manually authored MCP servers and skills, installed per project, team, agent group and agent."
    how:
      title: "How it works"
      description: "Ownership and installation are separate, so one entry can serve many scopes."
      own: "An entry belongs to one scope. Ownership decides who may edit it."
      install: "An install switches an entry on (or off) in a scope: a project, a team, an agent group, an agent, or the whole organization."
      resolve: "A task receives the union of the installs for its organization, project, owning teams, the assignee's groups and the assignee."
      fuse: "The worker merges that set with the agent host's own ~/.agents skills and MCP servers, and reports what it applied to Task details → System."
ru:
  libraryPage:
    title: "Библиотека MCP и навыков"
    subtitle: "MCP-серверы и навыки, добавленные вручную и установленные для проекта, команды, группы агентов и агента."
    how:
      title: "Как это работает"
      description: "Владение и установка разделены, поэтому одна запись может обслуживать несколько областей."
      own: "Запись принадлежит одной области. Владение решает, кто может её править."
      install: "Установка включает (или выключает) запись в области: проект, команда, группа агентов, агент или вся организация."
      resolve: "Задача получает объединение установок для своей организации, проекта, команд-владельцев, групп исполнителя и самого исполнителя."
      fuse: "Воркер объединяет этот набор с навыками и MCP-серверами хоста агента из ~/.agents и отправляет применённое в «Детали задачи → Система»."
</i18n>
