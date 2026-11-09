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
.flex.flex-col.gap-1(v-if="files.length" data-testid="changed-files")
  span.text-muted-foreground.text-xs.font-medium Changed files
  .flex.flex-col(class="gap-0.5")
    .flex.items-center.gap-2.text-xs(v-for="file in files" :key="file.path")
      FilePenIcon.text-muted-foreground.shrink-0(class="size-3.5")
      span.truncate.font-mono(:title="file.path") {{ displayPath(file.path) }}
      DiffStat.shrink-0(:added="file.added" :removed="file.removed")
</template>

<script setup lang="ts">
import DiffStat from "@/components/tasks/DiffStat.vue";
import { displayPath, type ChangedFile } from "@/lib/toolPresentation";
import { FilePenIcon } from "@lucide/vue";

defineProps<{ files: ChangedFile[] }>();
</script>
