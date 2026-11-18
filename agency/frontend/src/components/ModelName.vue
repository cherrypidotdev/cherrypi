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
span.inline-flex.items-baseline.gap-0(data-testid="model-name")
  span.font-semibold(v-if="model.provider") {{ model.provider }}
  span.opacity-60(v-if="model.provider") /
  span {{ name }}
</template>

<script setup lang="ts">
import { modelName, type ModelLike } from "@/lib/model";
import { computed } from "vue";

const props = defineProps<{ model: ModelLike }>();

const name = computed(() => modelName(props.model));
</script>
