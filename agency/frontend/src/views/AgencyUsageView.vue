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
  header.flex.flex-wrap.items-start.justify-between.gap-4
    .flex.min-w-0.flex-col
      h1.truncate.text-lg.font-semibold Usage
      p.text-muted-foreground.text-sm
        | Cost, tokens and task executions across every agent and project.
    AgencyViewTabs

  UsageDashboard(context="agency")
</template>

<script setup lang="ts">
import AgencyViewTabs from "@/components/AgencyViewTabs.vue";
import UsageDashboard from "@/components/usage/UsageDashboard.vue";
</script>
