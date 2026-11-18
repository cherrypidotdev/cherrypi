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
TooltipProvider(:delay-duration="120")
  Tooltip
    TooltipTrigger(as-child)
      slot
        RouterLink.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs.transition-colors(
          class="hover:bg-muted/50 py-0.5"
          :to="{ name: 'task', params: { id } }"
          :class="stateClass"
          @click.stop
        )
          LockIcon.size-3.shrink-0.text-amber-500(v-if="blocking")
          component.size-3.shrink-0(:is="stateIcon")
          span.font-semibold {{ id }}
    TipCard(data-slot="task-tip")
      .flex.items-center(class="gap-1.5")
        span.text-3xs.inline-flex.items-center.gap-1.rounded-full.border.font-medium(
          class="px-1.5 py-0.5"
          :class="stateClass"
        )
          component(class="size-2.5" :is="stateIcon")
          | {{ stateLabel }}
        span.text-muted-foreground.text-3xs.font-mono {{ id }}
        span.text-muted-foreground.text-3xs.ml-auto.font-medium(
          v-if="kindLabel"
        ) {{ kindLabel }}

      p.mt-1.text-xs.leading-snug.font-medium.break-words
        | {{ title }}
      .flex.flex-wrap.items-center.gap-1(v-if="meta.length" class="mt-1.5")
        span.text-muted-foreground.text-3xs.rounded.border(
          v-for="item in meta"
          :key="item"
          class="px-1.5 py-0.5"
        ) {{ item }}
      .flex.flex-wrap.items-center.gap-1(v-if="tags.length" class="mt-1.5")
        span.text-3xs.inline-flex.items-center.gap-1.rounded-full.border(
          v-for="tag in tags.slice(0, 6)"
          :key="tag.id"
          class="px-1.5 py-0.5"
          :style="tagStyle(tag.color)"
        )
          span.size-1.rounded-full.bg-current
          | {{ tag.name }}
        span.text-muted-foreground.text-3xs(v-if="tags.length > 6")
          | +{{ tags.length - 6 }}
</template>

<script setup lang="ts">
import TipCard from "@/components/TipCard.vue";
import {
  Tooltip,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  tagStyle,
  taskName,
  taskPriorityKey,
  taskSizeKey,
  taskStateClassFor,
  taskStateIcon,
  taskStateKeyFor,
} from "@/lib/task";
import {
  TaskPriority,
  TaskReferenceKind,
  TaskSize,
  type Tag,
  type TaskState,
} from "@cherrypi/sdk";
import { LockIcon } from "@lucide/vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    id: string;

    name?: string;

    prompt?: string;
    state: TaskState;

    priority?: TaskPriority;

    size?: TaskSize;

    tags?: Tag[];

    kind?: TaskReferenceKind;
  }>(),
  {
    name: "",
    prompt: "",
    priority: TaskPriority.UNSPECIFIED,
    size: TaskSize.UNSPECIFIED,
    tags: () => [],
    kind: TaskReferenceKind.UNSPECIFIED,
  },
);

const title = computed(() => taskName(props));

const stateLabel = computed(() => t(taskStateKeyFor({ state: props.state })));

const stateClass = computed(() => taskStateClassFor({ state: props.state }));

const stateIcon = computed(() => taskStateIcon(props.state));

const blocking = computed(() => props.kind === TaskReferenceKind.BLOCKING);

const meta = computed(() =>
  [
    props.priority === TaskPriority.UNSPECIFIED
      ? ""
      : t(taskPriorityKey(props.priority)),
    props.size === TaskSize.UNSPECIFIED ? "" : t(taskSizeKey(props.size)),
  ].filter(Boolean),
);

const kindLabel = computed(() => {
  switch (props.kind) {
    case TaskReferenceKind.BLOCKING:
      return t("taskTooltip.blocking");
    case TaskReferenceKind.INFORMATIONAL:
      return t("taskTooltip.informational");
    case TaskReferenceKind.SUPERSEDES:
      return t("taskTooltip.supersedes");
    default:
      return "";
  }
});
</script>

<i18n lang="yaml">
en:
  taskTooltip:
    blocking: "Blocking"
    informational: "Informational"
    supersedes: "Supersedes"
ru:
  taskTooltip:
    blocking: "Блокирует"
    informational: "Информационная"
    supersedes: "Заменяет"
</i18n>
