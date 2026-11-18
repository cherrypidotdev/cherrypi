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
.relative.flex.w-full.overflow-hidden.rounded-lg.border(class="h-[34rem]")
  TaskDetails(v-bind="drawerProps" :tab="tab" @update:tab="tab = $event")
</template>

<script setup lang="ts">
import TaskDetails from "@/views/TaskView/TaskDetails.vue";
import { ref, useAttrs, type VNodeProps } from "vue";

defineOptions({ inheritAttrs: false });

type DrawerProps = Omit<
  InstanceType<typeof TaskDetails>["$props"],
  keyof VNodeProps
>;
const drawerProps = useAttrs() as unknown as DrawerProps;

const props = withDefaults(defineProps<{ tab?: string }>(), { tab: "general" });
const tab = ref(props.tab);
</script>
