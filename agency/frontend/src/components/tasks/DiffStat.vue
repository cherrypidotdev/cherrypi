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
span.inline-flex.items-center.overflow-hidden.rounded.text-xs.tabular-nums(
  :title="title"
  data-testid="diff-stat"
)
  span.px-1.text-emerald-700(class="bg-emerald-500/10 dark:text-emerald-400") +{{ added }}
  span.px-1.text-red-700(class="bg-red-500/10 dark:text-red-400") −{{ removed }}
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  added: number;
  removed: number;
}>();

const title = computed(
  () => `${props.added} line(s) added, ${props.removed} line(s) removed`,
);
</script>
