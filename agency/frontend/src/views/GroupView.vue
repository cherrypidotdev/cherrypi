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
.flex.h-full.flex-col.overflow-hidden
  header.border-b.px-6.py-4
    .mx-auto.flex.w-full.max-w-6xl.items-start.justify-between.gap-4
      .flex.min-w-0.items-start.gap-3
        BackButton(
          :fallback="{ to: { name: 'groups' }, label: t('groups.title') }"
        )
        .flex.min-w-0.flex-col
          .flex.flex-wrap.items-center.gap-2
            h1.truncate.text-lg.font-semibold {{ group?.name || t("groups.group") }}
            Badge(v-if="group?.archivedAt" variant="outline")
              | {{ t("groups.archived") }}
            Badge(v-else variant="secondary")
              | {{ t("groups.memberCount", { count: group?.memberCount ?? 0 }) }}
          p.text-muted-foreground.truncate.text-sm(v-if="group?.description")
            | {{ group.description }}
          p.text-muted-foreground.text-xs(v-if="group")
            | {{ t("groups.slugPath", { slug: group.slug }) }}
      .flex.shrink-0.items-center.gap-2
        RouterLink(
          v-if="group && group.canEdit"
          :to="{ name: 'group-prompts', params: { slug: group.slug } }"
        )
          Button(size="sm")
            | {{ t("groups.editPrompts") }}

  nav.border-b.px-6
    .mx-auto.flex.w-full.max-w-6xl.flex-wrap.items-center.gap-1
      RouterLink.text-muted-foreground.rounded-md.px-3.py-2.text-sm.font-medium(
        v-for="tab in tabs"
        :key="tab.name"
        :to="{ name: tab.name, params: { slug } }"
        :class="isActive(tab.name) ? 'text-primary bg-sidebar-accent/60' : 'hover:bg-sidebar-accent/40 hover:text-foreground'"
      ) {{ t(tab.labelKey) }}

  .flex-1.overflow-auto
    .mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
      p.text-destructive.text-sm(v-if="error") {{ error }}
      Empty(v-if="!group && !loading")
        EmptyTitle {{ t("groups.notFound") }}
        EmptyDescription {{ t("groups.notFoundHint") }}
      RouterView(v-else-if="group")
</template>

<script setup lang="ts">
import BackButton from "@/components/BackButton.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
import { useAgentGroups } from "@/composables/useAgentGroups";
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { RouterView, useRoute } from "vue-router";

const { t } = useI18n();
const route = useRoute();
const { loading, error, groupBySlug, loadAgentGroups } = useAgentGroups();

const slug = computed(() => String(route.params.slug ?? ""));
const group = computed(() => groupBySlug(slug.value));

const tabs = [
  { name: "group-overview", labelKey: "groups.tabs.overview" },
  { name: "group-members", labelKey: "groups.tabs.members" },
  { name: "group-prompts", labelKey: "groups.tabs.prompts" },
  { name: "group-access", labelKey: "groups.tabs.access" },
  { name: "group-settings", labelKey: "groups.tabs.settings" },
];

function isActive(name: string): boolean {
  return route.name === name;
}

onMounted(async () => {
  await loadAgentGroups(true);
});
</script>

<i18n lang="yaml">
en:
  groups:
    group: Agent group
    slugPath: "/groups/{slug}"
    memberCount: "{count} members"
    editPrompts: Edit prompts
    notFound: Group not found
    notFoundHint: The group does not exist, or its slug changed. The old address 404s by design.
    tabs:
      overview: Overview
      members: Members
      prompts: Prompts
      access: Access
      settings: Settings
ru:
  groups:
    group: Группа агентов
    slugPath: "/groups/{slug}"
    memberCount: "Участников: {count}"
    editPrompts: Редактировать промпты
    notFound: Группа не найдена
    notFoundHint: Группы не существует, или её слаг изменился. Старый адрес по задумке возвращает 404.
    tabs:
      overview: Обзор
      members: Участники
      prompts: Промпты
      access: Доступ
      settings: Настройки
</i18n>
