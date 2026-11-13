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
RouterLink.flex.min-w-0.items-center.gap-2.font-medium(
  v-if="linkable"
  class="hover:text-primary"
  :to="to"
)
  ActorAvatar(:src="pictureUrl" :name="name" :username="username" :size="size")
  span.truncate {{ label }}
span.flex.min-w-0.items-center.gap-2(v-else)
  ActorAvatar(:src="pictureUrl" :name="name" :username="username" :size="size")
  span.truncate {{ label }}
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    kind: string;
    username?: string;
    name?: string;
    pictureUrl?: string;
    size?: number;
  }>(),
  { username: "", name: "", pictureUrl: "", size: 20 },
);

const label = computed(() => props.name || props.username || "?");
const linkable = computed(() => !!props.username);
const to = computed(() =>
  props.kind === "agent"
    ? { name: "agent", params: { username: props.username } }
    : { name: "user", params: { username: props.username } },
);
</script>
