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
span.inline-flex.items-baseline.gap-1
  img.self-center.rounded(
    v-if="iconUrl"
    :src="iconUrl"
    class="size-3.5"
    :alt="''"
    data-testid="hold-icon"
  )
  span {{ pluginName }}
  template(v-if="reason")
    span :
    a.font-semibold.underline.underline-offset-2(
      v-if="url"
      rel="noopener noreferrer"
      class="hover:text-foreground/80"
      :href="url"
      target="_blank"
      data-testid="hold-reason-link"
    ) {{ reason }}
    span(v-else) {{ reason }}
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    pluginName: string;
    reason?: string;
    url?: string;
    iconUrl?: string;
  }>(),
  { reason: "", url: "", iconUrl: "" },
);
</script>
