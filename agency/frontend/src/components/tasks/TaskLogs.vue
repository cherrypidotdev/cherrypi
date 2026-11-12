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
    | {{ $t("domainLog.empty") }}
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
      template(v-if="row.oldState !== undefined || row.newState !== undefined")
        .flex.flex-wrap.items-center(class="gap-1.5")
          span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs.font-medium.opacity-60(
            v-if="row.oldState !== undefined"
            class="py-0.5"
            :class="taskStateClass(row.oldState)"
          )
            component.size-3.shrink-0(:is="taskStateIcon(row.oldState)")
            | {{ $t(taskStateKey(row.oldState)) }}
          ArrowRightIcon.text-muted-foreground.shrink-0(class="size-3.5")
          span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs.font-medium(
            v-if="row.newState !== undefined"
            class="py-0.5"
            :class="taskStateClass(row.newState)"
          )
            component.size-3.shrink-0(:is="taskStateIcon(row.newState)")
            | {{ $t(taskStateKey(row.newState)) }}
        p.text-muted-foreground.text-xs(v-if="row.reason") {{ row.reason }}

      .flex.flex-wrap.items-center(
        v-else-if="row.oldPriority !== undefined || row.newPriority !== undefined"
        class="gap-1.5"
      )
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium.opacity-60(
          v-if="row.oldPriority !== undefined"
          class="py-0.5"
          :class="taskPriorityClass(row.oldPriority)"
        )
          | {{ $t(taskPriorityKey(row.oldPriority)) }}
        ArrowRightIcon.text-muted-foreground.shrink-0(class="size-3.5")
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
          v-if="row.newPriority !== undefined"
          class="py-0.5"
          :class="taskPriorityClass(row.newPriority)"
        )
          | {{ $t(taskPriorityKey(row.newPriority)) }}

      .flex.flex-wrap.items-center(
        v-else-if="row.oldSize !== undefined || row.newSize !== undefined"
        class="gap-1.5"
      )
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium.opacity-60(
          v-if="row.oldSize !== undefined"
          class="py-0.5"
          :class="taskSizeClass(row.oldSize)"
        )
          | {{ $t(taskSizeKey(row.oldSize)) }}
        ArrowRightIcon.text-muted-foreground.shrink-0(class="size-3.5")
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
          v-if="row.newSize !== undefined"
          class="py-0.5"
          :class="taskSizeClass(row.newSize)"
        )
          | {{ $t(taskSizeKey(row.newSize)) }}

      .flex.flex-wrap.items-center(
        v-else-if="row.oldText !== undefined || row.newText !== undefined"
        class="gap-1.5"
      )
        span.text-muted-foreground.text-xs.line-through(v-if="row.oldText")
          | {{ row.oldText }}
        ArrowRightIcon.text-muted-foreground.shrink-0(class="size-3.5")
        span.text-xs(v-if="row.newText") {{ row.newText }}

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
          LockIcon.size-3.shrink-0.text-amber-500(
            v-if="tag.isRequired"
            :title="$t('tasksTable.requiredTagHint')"
          )
        span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs.line-through.opacity-60(
          v-for="tag in row.removedTags"
          :key="tag.id"
          class="py-0.5"
          :style="tagStyle(tag.color)"
        )
          span.rounded-full.bg-current(class="size-1.5")
          | {{ tag.name }}

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
          component.size-3.shrink-0(
            :class="referenceKindClass(ref.kind)"
            :is="referenceKindIcon(ref.kind)"
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
          component.size-3.shrink-0(
            :class="referenceKindClass(ref.kind)"
            :is="referenceKindIcon(ref.kind)"
          )
          strong.font-semibold {{ ref.id }}
          span(v-if="ref.name") {{ ref.name }}
        RouterLink.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
          v-for="ref in row.changedRefs"
          :key="'changed-' + ref.id"
          class="hover:bg-muted/50 py-0.5"
          :to="`/tasks/${ref.id}`"
          :class="ref.state ? taskStateClass(ref.state) : ''"
          :title="$t('taskLog.kindChanged', { kind: referenceKindLabel(ref.kind) })"
        )
          component.size-3.shrink-0(
            :class="referenceKindClass(ref.kind)"
            :is="referenceKindIcon(ref.kind)"
          )
          strong.font-semibold {{ ref.id }}
          span(v-if="ref.name") {{ ref.name }}

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

      .flex.flex-wrap.items-center.gap-1(
        v-else-if="row.badges && row.badges.length"
      )
        a.inline-flex.items-stretch.overflow-hidden.rounded-md.border.text-xs(
          v-for="badge in row.badges"
          :key="badge.key"
          :class="taskBadgeToneClass(badge.tone)"
          :href="badge.url || undefined"
          :target="badge.url ? '_blank' : undefined"
          :rel="badge.url ? 'noopener noreferrer' : undefined"
          :title="$t('taskLog.pluginBadge', { plugin: row.pluginName || $t('taskBadges.plugin'), badge: badge.label })"
        )
          span.text-muted-foreground.inline-flex.items-center(
            class="bg-muted/60 px-1.5 py-0.5"
          ) {{ badge.label }}
          span.inline-flex.items-center.font-medium(
            v-if="badge.value"
            class="px-1.5 py-0.5"
          ) {{ badge.value }}

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
  taskBadgeToneClass,
  taskPriorityClass,
  taskPriorityKey,
  taskReferenceKindFromLog,
  taskReferenceKindIcon,
  taskReferenceKindKey,
  taskReferenceKindTextClass,
  taskSizeClass,
  taskSizeKey,
  taskStateClass,
  taskStateFromStored,
  taskStateIcon,
  taskStateKey,
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
  EyeOffIcon,
  FlagIcon,
  FolderIcon,
  GitBranchIcon,
  LinkIcon,
  LockIcon,
  LockOpenIcon,
  MessageSquareIcon,
  MessagesSquareIcon,
  PencilIcon,
  PlugIcon,
  PlusIcon,
  RulerIcon,
  SettingsIcon,
  SquareIcon,
  TagIcon,
  UnplugIcon,
  UserIcon,
  UsersIcon,
  type LucideIcon,
} from "@lucide/vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";

const props = defineProps<{ logs: TaskLog[] }>();

function referenceKindIcon(kind: string | undefined): LucideIcon {
  return taskReferenceKindIcon(taskReferenceKindFromLog(kind));
}

function referenceKindClass(kind: string | undefined): string {
  return taskReferenceKindTextClass(taskReferenceKindFromLog(kind));
}

function referenceKindLabel(kind: string | undefined): string {
  return t(taskReferenceKindKey(taskReferenceKindFromLog(kind)));
}

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
  session_lost: UnplugIcon,
  teams: UsersIcon,
  leads: CrownIcon,
  comment: MessagesSquareIcon,
  parent: GitBranchIcon,

  hidden: EyeOffIcon,
  plugin: PlugIcon,
  hold: LockOpenIcon,
  stop: SquareIcon,
};

const KIND_LABEL_KEYS: Record<string, string> = {
  created: "taskLog.kind.created",
  state: "taskLog.kind.state",
  name: "taskLog.kind.name",
  prompt: "taskLog.kind.prompt",
  priority: "taskLog.kind.priority",
  size: "taskLog.kind.size",
  assignee: "taskLog.kind.assignee",
  tags: "taskLog.kind.tags",
  references: "taskLog.kind.references",
  links: "taskLog.kind.links",
  project: "taskLog.kind.project",
  model: "taskLog.kind.model",
  session: "taskLog.kind.session",
  session_lost: "taskLog.kind.sessionLost",
  teams: "taskLog.kind.teams",
  leads: "taskLog.kind.leads",
  comment: "taskLog.kind.comment",
  parent: "taskLog.kind.parent",
  hidden: "taskLog.kind.hidden",
  plugin: "taskLog.kind.plugin",
  hold: "taskLog.kind.hold",
  stop: "taskLog.kind.stop",
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
  isRequired?: boolean;
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

interface LogBadge {
  key: string;
  label: string;
  value?: string;
  tone: string;
  url?: string;
}

interface LogRow {
  log: TaskLog;
  icon: LucideIcon;
  kindLabel: string;
  pluginName?: string;
  badges?: LogBadge[];
  oldState?: TaskState;
  newState?: TaskState;

  reason?: string;
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
    kindLabel: KIND_LABEL_KEYS[log.kind]
      ? t(KIND_LABEL_KEYS[log.kind]!)
      : log.kind,
  };
  const data = parseData(log);
  if (!data) return row;

  switch (log.kind) {
    case "state":
      row.oldState = taskStateFromStored(data.old);
      row.newState = taskStateFromStored(data.new);
      row.reason = typeof data.reason === "string" ? data.reason : undefined;
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
          state: taskStateFromStored(ref.state as string | undefined),
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
    case "plugin":
      row.pluginName = (data.plugin as string) ?? "";
      row.badges = ((data.badges ?? []) as LogBadge[]).map((badge) => ({
        ...badge,
        tone: badge.tone ?? "neutral",
      }));
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
      return t("admin.agents.columnAgent");
    case "system":
      return "System";
    default:
      return actor;
  }
}

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  taskLog:
    system: "System"
    kind:
      parent: "Parent task"
      created: "Created"
      state: "State"
      name: "Title"
      prompt: "Prompt"
      priority: "Priority"
      size: "Size"
      assignee: "Assignee"
      tags: "Tags"
      references: "References"
      links: "Links"
      project: "Project"
      model: "Model"
      session: "Session"
      sessionLost: "Session lost"
      teams: "Owning teams"
      leads: "Leads"
      comment: "Comment"
      hidden: "Visibility"
      plugin: "Plugin badges"
      hold: "Plugin hold"
      stop: "Stop requested"
    kindChanged: "Kind changed to {kind}"
    pluginBadge: "{plugin} — {badge}"
ru:
  taskLog:
    system: "Система"
    kind:
      parent: "Родительская задача"
      created: "Создано"
      state: "Состояние"
      name: "Название"
      prompt: "Промпт"
      priority: "Приоритет"
      size: "Размер"
      assignee: "Исполнитель"
      tags: "Теги"
      references: "Связи"
      links: "Ссылки"
      project: "Проект"
      model: "Модель"
      session: "Сеанс"
      sessionLost: "Сеанс потерян"
      teams: "Команды-владельцы"
      leads: "Руководители"
      comment: "Комментарий"
      hidden: "Видимость"
      plugin: "Значки плагинов"
      hold: "Удержание плагином"
      stop: "Запрошена остановка"
    kindChanged: "Тип изменён на {kind}"
    pluginBadge: "{plugin} — {badge}"
</i18n>
