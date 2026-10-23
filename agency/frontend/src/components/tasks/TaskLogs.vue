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
.flex.flex-col(class="gap-0.5")
  p.text-muted-foreground.text-xs(v-if="!logs.length")
    | No activity yet.
  .flex.items-start.gap-2.rounded-md.px-1(
    v-for="row in rows"
    :key="String(row.log.id)"
    class="py-1.5"
  )
    span.text-muted-foreground.shrink-0(
      class="mt-0.5"
      :title="row.kindLabel"
      :class="row.log.kind === 'created' ? 'text-foreground' : ''"
    )
      component(class="size-3.5" :is="row.icon")
    .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
      //- State: old badge, arrow, new badge.
      .flex.flex-wrap.items-center(
        v-if="row.oldState !== undefined || row.newState !== undefined"
        class="gap-1.5"
      )
        span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs.font-medium.opacity-60(
          v-if="row.oldState !== undefined"
          class="py-0.5"
          :class="taskStateClass(row.oldState)"
        )
          component.size-3.shrink-0(:is="taskStateIcon(row.oldState)")
          | {{ taskStateLabel(row.oldState) }}
        ArrowRightIcon.text-muted-foreground.shrink-0(class="size-3.5")
        span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs.font-medium(
          v-if="row.newState !== undefined"
          class="py-0.5"
          :class="taskStateClass(row.newState)"
        )
          component.size-3.shrink-0(:is="taskStateIcon(row.newState)")
          | {{ taskStateLabel(row.newState) }}

      //- Priority: old pill, arrow, new pill.
      .flex.flex-wrap.items-center(
        v-else-if="row.oldPriority !== undefined || row.newPriority !== undefined"
        class="gap-1.5"
      )
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium.opacity-60(
          v-if="row.oldPriority !== undefined"
          class="py-0.5"
          :class="taskPriorityClass(row.oldPriority)"
        )
          | {{ taskPriorityLabel(row.oldPriority) }}
        ArrowRightIcon.text-muted-foreground.shrink-0(class="size-3.5")
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
          v-if="row.newPriority !== undefined"
          class="py-0.5"
          :class="taskPriorityClass(row.newPriority)"
        )
          | {{ taskPriorityLabel(row.newPriority) }}

      //- Size: old pill, arrow, new pill.
      .flex.flex-wrap.items-center(
        v-else-if="row.oldSize !== undefined || row.newSize !== undefined"
        class="gap-1.5"
      )
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium.opacity-60(
          v-if="row.oldSize !== undefined"
          class="py-0.5"
          :class="taskSizeClass(row.oldSize)"
        )
          | {{ taskSizeLabel(row.oldSize) }}
        ArrowRightIcon.text-muted-foreground.shrink-0(class="size-3.5")
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
          v-if="row.newSize !== undefined"
          class="py-0.5"
          :class="taskSizeClass(row.newSize)"
        )
          | {{ taskSizeLabel(row.newSize) }}

      //- A renamed title or a changed model/thinking level.
      .flex.flex-wrap.items-center(
        v-else-if="row.oldText !== undefined || row.newText !== undefined"
        class="gap-1.5"
      )
        span.text-muted-foreground.text-xs.line-through(v-if="row.oldText")
          | {{ row.oldText }}
        ArrowRightIcon.text-muted-foreground.shrink-0(class="size-3.5")
        span.text-xs(v-if="row.newText") {{ row.newText }}

      //- Assignee / project reference change.
      .flex.flex-wrap.items-center(
        v-else-if="row.oldRef || row.newRef"
        class="gap-1.5"
      )
        template(v-if="row.oldRef")
          RouterLink.text-muted-foreground.text-xs.line-through(
            v-if="row.oldRefLink"
            class="hover:underline"
            :to="row.oldRefLink"
          ) {{ row.oldRef.name }}
          span.text-muted-foreground.text-xs.line-through(v-else)
            | {{ row.oldRef.name }}
        ArrowRightIcon.text-muted-foreground.shrink-0(class="size-3.5")
        template(v-if="row.newRef")
          RouterLink.text-xs(
            v-if="row.newRefLink"
            class="hover:underline"
            :to="row.newRefLink"
          ) {{ row.newRef.name }}
          span.text-xs(v-else) {{ row.newRef.name }}

      //- Tags: added chips, removed struck-through chips.
      .flex.flex-wrap.items-center.gap-1(
        v-else-if="(row.addedTags && row.addedTags.length) || (row.removedTags && row.removedTags.length)"
      )
        span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
          v-for="tag in row.addedTags"
          :key="tag.id"
          class="py-0.5"
          :style="tagStyle(tag.color)"
        )
          span.rounded-full.bg-current(class="size-1.5")
          | {{ tag.name }}
        span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs.line-through.opacity-60(
          v-for="tag in row.removedTags"
          :key="tag.id"
          class="py-0.5"
          :style="tagStyle(tag.color)"
        )
          span.rounded-full.bg-current(class="size-1.5")
          | {{ tag.name }}

      //- References: clickable task pills.
      .flex.flex-wrap.items-center.gap-1(
        v-else-if="(row.addedRefs && row.addedRefs.length) || (row.removedRefs && row.removedRefs.length) || (row.changedRefs && row.changedRefs.length)"
      )
        RouterLink.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
          v-for="ref in row.addedRefs"
          :key="ref.id"
          class="hover:bg-muted/50 py-0.5"
          :to="`/tasks/${ref.id}`"
          :class="ref.state ? taskStateClass(ref.state) : ''"
        )
          LockIcon.size-3.shrink-0.text-amber-500(
            v-if="ref.kind === 'blocking'"
          )
          component.size-3.shrink-0(
            v-if="ref.state"
            :is="taskStateIcon(ref.state)"
          )
          strong.font-semibold {{ ref.id }}
          span(v-if="ref.name") {{ ref.name }}
        RouterLink.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs.line-through.opacity-60(
          v-for="ref in row.removedRefs"
          :key="ref.id"
          class="hover:bg-muted/50 py-0.5"
          :to="`/tasks/${ref.id}`"
        )
          LockIcon.size-3.shrink-0.text-amber-500(
            v-if="ref.kind === 'blocking'"
          )
          strong.font-semibold {{ ref.id }}
          span(v-if="ref.name") {{ ref.name }}
        RouterLink.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
          v-for="ref in row.changedRefs"
          :key="'changed-' + ref.id"
          class="hover:bg-muted/50 py-0.5"
          :to="`/tasks/${ref.id}`"
          :class="ref.state ? taskStateClass(ref.state) : ''"
          :title="`Kind changed to ${ref.kind}`"
        )
          LockIcon.size-3.shrink-0.text-amber-500(
            v-if="ref.kind === 'blocking'"
          )
          LinkIcon.text-muted-foreground.size-3.shrink-0(v-else)
          strong.font-semibold {{ ref.id }}
          span(v-if="ref.name") {{ ref.name }}

      //- Links: clickable external anchors.
      .flex.flex-col(
        v-else-if="(row.addedLinks && row.addedLinks.length) || (row.removedLinks && row.removedLinks.length)"
        class="gap-0.5"
      )
        a.inline-flex.items-center.gap-1.text-xs(
          v-for="link in row.addedLinks"
          :key="link.url"
          class="hover:underline"
          rel="noopener noreferrer"
          :href="link.url"
          :title="link.url"
          target="_blank"
        )
          LinkIcon.shrink-0(class="size-3.5")
          span.font-medium {{ link.label || link.url }}
          ExternalLinkIcon.text-muted-foreground.size-3.shrink-0
        a.inline-flex.items-center.gap-1.text-xs.line-through.opacity-60(
          v-for="link in row.removedLinks"
          :key="link.url"
          class="hover:underline"
          rel="noopener noreferrer"
          :href="link.url"
          :title="link.url"
          target="_blank"
        )
          LinkIcon.shrink-0(class="size-3.5")
          span.font-medium {{ link.label || link.url }}

      //- Everything else (create, prompt edit, reference promotion).
      span.text-xs.leading-snug(v-else) {{ row.log.message }}

      .text-muted-foreground.flex.flex-wrap.items-center.gap-x-1.text-xs
        span(:title="formatTimestamp(row.log.at)")
          | {{ formatRelativeTimestamp(row.log.at) }}
        span ·
        span.inline-flex.items-center.gap-1
          component.size-3.shrink-0(:is="actorIcon(row.log.actor)")
          | {{ actorLabel(row.log.actor) }}
</template>

<script setup lang="ts">
import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { formatRelativeTimestamp, formatTimestamp } from "@/lib/format";
import {
  tagStyle,
  taskPriorityClass,
  taskPriorityLabel,
  taskSizeClass,
  taskSizeLabel,
  taskStateClass,
  taskStateIcon,
  taskStateLabel,
} from "@/lib/task";
import { TaskPriority, TaskSize, TaskState, type TaskLog } from "@cherrypi/sdk";
import {
  ActivityIcon,
  ArrowRightIcon,
  BotIcon,
  CpuIcon,
  CrownIcon,
  DatabaseZapIcon,
  ExternalLinkIcon,
  FlagIcon,
  FolderIcon,
  GitBranchIcon,
  LinkIcon,
  LockIcon,
  MessageSquareIcon,
  PencilIcon,
  PlusIcon,
  RulerIcon,
  SettingsIcon,
  TagIcon,
  UserIcon,
  UsersIcon,
  type LucideIcon,
} from "@lucide/vue";
import { computed } from "vue";
import { RouterLink } from "vue-router";

const props = defineProps<{ logs: TaskLog[] }>();

const { usernameFor } = useAgents();
const { active: projects } = useProjects();

const KIND_ICONS: Record<string, LucideIcon> = {
  created: PlusIcon,
  state: ActivityIcon,
  name: PencilIcon,
  prompt: MessageSquareIcon,
  priority: FlagIcon,
  size: RulerIcon,
  assignee: UserIcon,
  tags: TagIcon,
  references: GitBranchIcon,
  links: LinkIcon,
  project: FolderIcon,
  model: CpuIcon,
  session: DatabaseZapIcon,
  teams: UsersIcon,
  leads: CrownIcon,
};

const KIND_LABELS: Record<string, string> = {
  created: "Created",
  state: "State",
  name: "Title",
  prompt: "Prompt",
  priority: "Priority",
  size: "Size",
  assignee: "Assignee",
  tags: "Tags",
  references: "References",
  links: "Links",
  project: "Project",
  model: "Model",
  session: "Session",
  teams: "Owning teams",
  leads: "Leads",
};

// Raw stored state/size values map onto the SDK's proto enums, and the stored
// priority onto its proto enum, so the log can reuse the table's badges.
const STATE_VALUES: Record<string, TaskState> = {
  draft: TaskState.DRAFT,
  backlog: TaskState.BACKLOG,
  queued: TaskState.QUEUED,
  running: TaskState.RUNNING,
  input: TaskState.INPUT,
  settled: TaskState.SETTLED,
  failed: TaskState.FAILED,
  completed: TaskState.COMPLETED,
  cancelled: TaskState.CANCELLED,
};

const SIZE_VALUES: Record<string, TaskSize> = {
  small: TaskSize.SMALL,
  medium: TaskSize.MEDIUM,
  big: TaskSize.BIG,
};

const PRIORITY_VALUES: Record<number, TaskPriority> = {
  [-1]: TaskPriority.LOW,
  [0]: TaskPriority.NORMAL,
  [1]: TaskPriority.HIGH,
  [2]: TaskPriority.CRITICAL,
};

interface LogTag {
  id: string;
  name: string;
  color: string;
}

interface LogReference {
  id: string;
  name: string;
  state?: TaskState;
  kind?: string;
}

interface LogLink {
  url: string;
  label: string;
}

interface LogRef {
  id: string;
  name: string;
}

interface LogRow {
  log: TaskLog;
  icon: LucideIcon;
  kindLabel: string;
  oldState?: TaskState;
  newState?: TaskState;
  oldPriority?: TaskPriority;
  newPriority?: TaskPriority;
  oldSize?: TaskSize;
  newSize?: TaskSize;
  oldText?: string;
  newText?: string;
  oldRef?: LogRef;
  newRef?: LogRef;
  oldRefLink?: string;
  newRefLink?: string;
  addedTags?: LogTag[];
  removedTags?: LogTag[];
  addedRefs?: LogReference[];
  removedRefs?: LogReference[];
  changedRefs?: LogReference[];
  addedLinks?: LogLink[];
  removedLinks?: LogLink[];
}

function parseData(log: TaskLog): Record<string, any> | undefined {
  if (!log.dataJson) return undefined;
  try {
    const parsed = JSON.parse(log.dataJson);
    return parsed && typeof parsed === "object" ? parsed : undefined;
  } catch {
    return undefined;
  }
}

function refLink(kind: string, id: string): string | undefined {
  if (!id) return undefined;
  if (kind === "assignee") {
    const username = usernameFor(id);
    return username ? `/agents/${username}` : undefined;
  }
  const project = projects.value.find((item) => item.id === id);
  return project ? `/projects/${project.slug}` : undefined;
}

function buildRow(log: TaskLog): LogRow {
  const row: LogRow = {
    log,
    icon: KIND_ICONS[log.kind] ?? ActivityIcon,
    kindLabel: KIND_LABELS[log.kind] ?? log.kind,
  };
  const data = parseData(log);
  if (!data) return row;

  switch (log.kind) {
    case "state":
      row.oldState = STATE_VALUES[data.old];
      row.newState = STATE_VALUES[data.new];
      break;
    case "priority":
      row.oldPriority = PRIORITY_VALUES[data.old];
      row.newPriority = PRIORITY_VALUES[data.new];
      break;
    case "size":
      row.oldSize = SIZE_VALUES[data.old];
      row.newSize = SIZE_VALUES[data.new];
      break;
    case "name":
    case "model":
    case "thinking":
      row.oldText = data.old || "";
      row.newText = data.new || "";
      break;
    case "assignee":
    case "project":
      if (data.oldName !== undefined || data.oldId !== undefined) {
        row.oldRef = { id: data.oldId ?? "", name: data.oldName ?? "—" };
        row.oldRefLink = refLink(log.kind, data.oldId ?? "");
      }
      if (data.newName !== undefined || data.newId !== undefined) {
        row.newRef = { id: data.newId ?? "", name: data.newName ?? "—" };
        row.newRefLink = refLink(log.kind, data.newId ?? "");
      }
      break;
    case "tags":
      row.addedTags = (data.added ?? []) as LogTag[];
      row.removedTags = (data.removed ?? []) as LogTag[];
      break;
    case "references": {
      const withState = (refs: LogReference[]) =>
        refs.map((ref) => ({
          ...ref,
          state: STATE_VALUES[ref.state as unknown as string],
        }));
      row.addedRefs = withState((data.added ?? []) as LogReference[]);
      row.removedRefs = withState((data.removed ?? []) as LogReference[]);
      row.changedRefs = withState((data.changed ?? []) as LogReference[]);
      break;
    }
    case "links":
      row.addedLinks = (data.added ?? []) as LogLink[];
      row.removedLinks = (data.removed ?? []) as LogLink[];
      break;
  }
  return row;
}

const rows = computed(() => props.logs.map(buildRow));

function actorIcon(actor: string): LucideIcon {
  switch (actor) {
    case "user":
      return UserIcon;
    case "agent":
      return BotIcon;
    default:
      return SettingsIcon;
  }
}

function actorLabel(actor: string): string {
  switch (actor) {
    case "user":
      return "User";
    case "agent":
      return "Agent";
    case "system":
      return "System";
    default:
      return actor;
  }
}
</script>
