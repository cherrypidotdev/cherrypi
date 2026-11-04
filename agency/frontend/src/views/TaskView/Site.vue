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
import SiteFrame from "@/components/sites/SiteFrame.vue";
import type { Site } from "@cherrypi/sdk";
import { ExternalLinkIcon, MaximizeIcon } from "@lucide/vue";

defineProps<{
  site: Site;
}>();

defineEmits<{
  (e: "open-full"): void;
  (e: "open-new"): void;
}>();
</script>

<template lang="pug">
.flex.min-h-0.min-w-0.flex-1.flex-col
  .flex.shrink-0.items-center.justify-between.gap-2.border-b.px-3.py-2
    span.truncate.text-sm.font-semibold
      | {{ site.title || site.id }}

    .flex.items-center.gap-2
      button.d-btn.d-btn-sm.d-btn-square(
        title="Open the site in a full-size agency view"
        @click="$emit('open-full')"
      )
        MaximizeIcon(:size="18")

      button.d-btn.d-btn-sm.d-btn-square(
        title="Open the site in a new browser tab"
        @click="$emit('open-new')"
      )
        ExternalLinkIcon(:size="18")

  SiteFrame.bg-base-100.min-h-0.w-full(:site="site")
</template>

<style lang="postcss" scoped></style>
