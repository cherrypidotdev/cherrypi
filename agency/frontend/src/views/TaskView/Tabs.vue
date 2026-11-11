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

<script setup lang="ts">
import type { Site } from "@cherrypi/sdk";
import { GlobeIcon, MessageSquareIcon } from "@lucide/vue";

defineProps<{
  taskSites: Site[];
  activeTaskTab: String;
  detailsOpen: Boolean;
}>();

defineEmits<{
  (e: "selectSiteTab", site: Site): void;
  (e: "selectChat"): void;
}>();
</script>

<template lang="pug">
.bg-base-200.px-2.pt-2
  .d-tabs.d-tabs-lift(role="tablist")
    button.d-tab._tab(
      type="button"
      role="tab"
      :title="$t('taskTabs.chatTitle')"
      :class="activeTaskTab === 'chat' ? 'd-tab-active' : ''"
      @click="$emit('selectChat')"
    )
      MessageSquareIcon.size-4(:stroke-width="2")
      span {{ $t("taskTabs.chat") }}

    button.d-tab._tab(
      v-for="site in taskSites"
      :key="site.id"
      type="button"
      role="tab"
      :title="$t('taskTabs.openSite', { name: site.title || site.id })"
      :class="activeTaskTab === `site:${site.id}` ? 'd-tab-active' : ''"
      @click="$emit('selectSiteTab', site)"
    )
      GlobeIcon.size-4(:stroke-width="2")
      span {{ site.title || site.id }}
</template>

<style lang="postcss" scoped>
@reference '#style.css';

._tab {
  @apply gap-2 px-4 text-sm font-semibold;
}
</style>

<i18n lang="yaml">
en:
  taskTabs:
    chatTitle: "Task chat"
    chat: "Chat"
    openSite: "Open {name}"
ru:
  taskTabs:
    chatTitle: "Чат задачи"
    chat: "Чат"
    openSite: "Открыть {name}"
</i18n>
