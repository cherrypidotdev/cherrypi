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
details.rounded-md.border(class="group/tool" data-testid="tool-call")
  summary.flex.cursor-pointer.items-center.gap-2.px-2.py-1.text-xs.select-none
    ChevronRightIcon.text-muted-foreground.size-3.shrink-0.transition-transform(
      class="group-open/tool:rotate-90"
    )
    component.text-muted-foreground.shrink-0(class="size-3.5" :is="icon")
    span.text-muted-foreground.shrink-0(v-if="view.verb") {{ view.verb }}
    span.truncate.font-mono(:title="fullTarget") {{ displayPath(view.target) }}
    DiffStat.shrink-0(
      v-if="change"
      :added="change.added"
      :removed="change.removed"
    )
    span.text-muted-foreground.shrink-0(v-if="running") {{ $t("toolCall.running") }}
    span.text-muted-foreground.shrink-0(
      v-else-if="interrupted || item.phase !== ToolPhase.FINISHED"
      :title="$t('toolCall.noResult')"
    ) {{ $t("toolCalls.statusInterrupted") }}
    span.text-destructive.shrink-0(v-if="item.isError") {{ $t("toolCalls.statusError") }}
    span.text-muted-foreground.shrink-0(v-if="duration") · {{ duration }}

  ToolCallBody(:item="item")
</template>

<script setup lang="ts">
import DiffStat from "@/components/tasks/DiffStat.vue";
import ToolCallBody from "@/components/tasks/ToolCallBody.vue";
import type { ToolItem } from "@/lib/console";
import { displayPath, toolView } from "@/lib/toolPresentation";
import { ToolPhase } from "@cherrypi/sdk";
import {
  ChevronRightIcon,
  FilePlusIcon,
  FileTextIcon,
  PencilIcon,
  SquareTerminalIcon,
  WrenchIcon,
} from "@lucide/vue";
import { computed } from "vue";

const props = defineProps<{
  item: ToolItem;

  running: boolean;

  interrupted?: boolean;

  duration: string;
}>();

const view = computed(() => toolView(props.item));

const icons = {
  bash: SquareTerminalIcon,
  read: FileTextIcon,
  write: FilePlusIcon,
  edit: PencilIcon,
  generic: WrenchIcon,
};
const icon = computed(() => icons[view.value.icon]);

const fullTarget = computed(() => view.value.target);

const change = computed(() => view.value.changes[0]);
</script>

<i18n lang="yaml">
en:
  toolCall:
    running: "running…"
    noResult: "The call's run ended before it recorded a result"
ru:
  toolCall:
    running: "выполняется…"
    noResult: "Выполнение вызова завершилось до записи результата"
</i18n>
