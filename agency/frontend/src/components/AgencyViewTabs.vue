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
Tabs(:model-value="active" @update:model-value="navigate")
  TabsList
    TabsTrigger(value="tasks") Tasks
    TabsTrigger(value="usage") Usage
</template>

<script setup lang="ts">
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const active = computed(() =>
  route.name === "agency-usage" ? "usage" : "tasks",
);

function navigate(value: string | number) {
  if (value === "tasks") {
    if (route.name !== "agency-tasks")
      void router.push({ name: "agency-tasks" });
    return;
  }
  if (route.name !== "agency-usage") void router.push({ name: "agency-usage" });
}
</script>
