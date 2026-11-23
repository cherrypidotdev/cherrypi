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
  KnowledgeWorkspace(scope="all")
</template>

<script setup lang="ts">
import KnowledgeWorkspace from "@/components/knowledge/KnowledgeWorkspace.vue";
import { setPageTitle } from "@/composables/usePageTitle";
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

onMounted(() => setPageTitle(t("pageTitle.knowledge")));
</script>

<i18n lang="yaml">
en:
  pageTitle:
    knowledge: "Knowledge"
ru:
  pageTitle:
    knowledge: "База знаний"
</i18n>
