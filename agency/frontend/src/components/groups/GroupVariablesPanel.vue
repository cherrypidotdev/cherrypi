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
VariablesPanel(
  v-if="group"
  :target="{ scope: 4, groupId: group.id }"
  :can-manage="group.canEdit"
)
</template>

<script setup lang="ts">
import VariablesPanel from "@/components/settings/VariablesPanel.vue";
import { useAgentGroups } from "@/composables/useAgentGroups";
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();
const { groupBySlug } = useAgentGroups();

const slug = computed(() => String(route.params.slug ?? ""));
const group = computed(() => groupBySlug(slug.value));
</script>
