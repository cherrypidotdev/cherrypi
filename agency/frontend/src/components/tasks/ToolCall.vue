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
details.rounded-md.border(class="group/tool")
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
    span.text-muted-foreground.shrink-0(v-if="running") running…
    span.text-muted-foreground.shrink-0(
      v-else-if="interrupted || item.phase !== ToolPhase.FINISHED"
      title="The call's run ended before it recorded a result"
    ) interrupted
    span.text-destructive.shrink-0(v-if="item.isError") error
    span.text-muted-foreground.shrink-0(v-if="duration") · {{ duration }}

  .flex.flex-col.gap-2.px-2.pb-2(v-if="view.sections.length")
    .overflow-hidden.rounded-md.border(
      v-for="(section, index) in view.sections"
      :key="section.label"
    )
      .flex.items-center.gap-2.px-2.py-1(class="bg-muted/40")
        span.text-muted-foreground.text-xs.font-medium {{ section.label }}
        .ml-auto.flex.items-center.gap-2
          time.text-muted-foreground.text-xs.tabular-nums(
            v-if="stampFor(section) !== undefined"
            :datetime="isoInstant(stampFor(section))"
            :title="stampTitle(section)"
          ) {{ formatClockPrecise(stampFor(section)) }}
          CopyButton(
            :label="`Copy ${section.label.toLowerCase()}`"
            :text="section.text"
          )
      pre.max-w-full.overflow-auto.text-xs.wrap-break-word(
        v-if="section.kind === 'diff'"
        :class="bodyClass(index, section)"
      )
        template(
          v-for="(line, lineIndex) in lines(section.text)"
          :key="lineIndex"
        )
          span.block.whitespace-pre(:class="diffLineClass(line)") {{ line }}
      pre.max-w-full.overflow-auto.p-2.text-xs.wrap-break-word.whitespace-pre-wrap(
        v-else
        :class="bodyClass(index, section)"
      ) {{ section.text }}
      button.text-muted-foreground.flex.w-full.items-center.justify-center.gap-1.border-t.px-2.py-1.text-xs(
        v-if="isLong(section)"
        type="button"
        class="hover:bg-muted hover:text-foreground"
        :aria-expanded="expanded.has(index)"
        :title="expanded.has(index) ? `Collapse ${section.label.toLowerCase()}` : `Show all of the ${section.label.toLowerCase()}`"
        @click="toggle(index)"
      )
        ChevronDownIcon.size-3(
          :class="expanded.has(index) ? 'rotate-180' : ''"
        )
        span {{ expanded.has(index) ? "Show less" : "Show all" }}
</template>

<script setup lang="ts">
import CopyButton from "@/components/CopyButton.vue";
import DiffStat from "@/components/tasks/DiffStat.vue";
import type { ToolItem } from "@/lib/console";
import { formatClockPrecise, formatInstant, isoInstant } from "@/lib/format";
import {
  displayPath,
  toolView,
  visualLines,
  type ToolSection,
} from "@/lib/toolPresentation";
import { ToolPhase } from "@cherrypi/sdk";
import {
  ChevronDownIcon,
  ChevronRightIcon,
  FilePlusIcon,
  FileTextIcon,
  PencilIcon,
  SquareTerminalIcon,
  WrenchIcon,
} from "@lucide/vue";
import { computed, reactive } from "vue";

const props = defineProps<{
  item: ToolItem;

  running: boolean;

  interrupted?: boolean;

  duration: string;
}>();

const LONG_LINES = 14;
const LONG_CHARS = 1200;

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

function stampFor(section: ToolSection): number | undefined {
  return section.source === "result"
    ? props.item.endedAt
    : props.item.startedAt;
}

function stampTitle(section: ToolSection): string {
  const at = stampFor(section);
  const what = section.source === "result" ? "Result arrived" : "Called";
  return `${what} at ${formatInstant(at)}`;
}

const expanded = reactive(new Set<number>());

function lines(text: string): string[] {
  return text.split("\n");
}

function isLong(section: ToolSection): boolean {
  return (
    section.text.length > LONG_CHARS || visualLines(section.text) > LONG_LINES
  );
}

function bodyClass(index: number, section: ToolSection): string[] {
  const collapsed = isLong(section) && !expanded.has(index);

  const padding = section.kind === "diff" ? "py-1" : "";
  return [padding, collapsed ? "max-h-40 overflow-hidden" : ""].filter(
    (name) => name !== "",
  );
}

function diffLineClass(line: string): string {
  if (line.startsWith("+++") || line.startsWith("---")) {
    return "text-muted-foreground";
  }
  if (line.startsWith("+")) return "text-emerald-700 dark:text-emerald-400";
  if (line.startsWith("-")) return "text-red-700 dark:text-red-400";
  return "text-muted-foreground";
}

function toggle(index: number) {
  if (expanded.has(index)) expanded.delete(index);
  else expanded.add(index);
}
</script>
