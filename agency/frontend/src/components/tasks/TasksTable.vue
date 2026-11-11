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
Table(class="[&_td]:px-1 [&_th]:px-1" container-class="no-scrollbar")
  TableHeader
    TableRow
      TableHead.w-10
        button.text-muted-foreground.flex.items-center.justify-center.rounded-md.transition-colors(
          type="button"
          class="hover:text-foreground"
          :title="allSelected ? $t('tasksTable.clearSelection') : $t('tasksTable.selectAllInTab')"
          :aria-label="allSelected ? $t('tasksTable.clearSelection') : $t('tasksTable.selectAllInTab')"
          @click="$emit('toggleSelectAll')"
          :disabled="!selectableCount"
        )
          CheckSquareIcon.size-4(v-if="allSelected")
          MinusSquareIcon.size-4(v-else-if="someSelected")
          SquareIcon.size-4(v-else)
      TableHead.w-8(:title="$t('tasksTable.favoritesHint')")
        StarIcon.text-muted-foreground(
          class="size-3.5"
          :aria-label="$t('tasksTable.favorites')"
        )
      template(v-for="column in visibleColumns" :key="column.key")
        TableHead(
          :class="column.widthClass"
          :title="column.hintKey ? $t(column.hintKey) : undefined"
        )
          button.flex.items-center.gap-1(
            v-if="column.sortKey"
            type="button"
            @click="$emit('sort', column.sortKey ?? '')"
          )
            span {{ $t(column.labelKey) }}
            component.text-muted-foreground.size-3(
              :is="sortIcon(column.sortKey ?? '')"
            )
          span(v-else) {{ $t(column.labelKey) }}
  TableBody
    TableRowsSkeleton(v-if="loading && !tasks.length" :widths="skeletonWidths")
    TableRow.cursor-pointer(
      v-for="task in tasks"
      :key="task.id"
      @click="$emit('open', task.id)"
    )
      TableCell.w-10(@click.stop)
        button.flex.items-center.justify-center.rounded-md.transition-colors(
          type="button"
          :class="selected.has(task.id) ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'"
          :title="selected.has(task.id) ? $t('tasksTable.deselectTask') : $t('tasksTable.selectTask')"
          :aria-label="selected.has(task.id) ? $t('tasksTable.deselectTask') : $t('tasksTable.selectTask')"
          :aria-pressed="selected.has(task.id)"
          @click="$emit('toggleSelect', task.id)"
        )
          CheckSquareIcon.size-4(v-if="selected.has(task.id)")
          SquareIcon.size-4(v-else)
      TableCell.w-8(@click.stop)
        button.flex.items-center.justify-center.rounded-md.p-1.transition-colors(
          type="button"
          :class="task.favorite ? 'text-amber-500' : 'text-muted-foreground/60 hover:text-foreground'"
          :title="task.favorite ? $t('tasksTable.removeFavorite') : $t('tasksTable.addFavorite')"
          :aria-label="task.favorite ? $t('tasksTable.removeFavorite') : $t('tasksTable.addFavorite')"
          :aria-pressed="task.favorite"
          @click="$emit('toggleFavorite', task.id)"
        )
          StarIcon.size-4(:fill="task.favorite ? 'currentColor' : 'none'")
      template(v-for="column in visibleColumns" :key="column.key")
        TableCell(v-if="column.key === 'state'")
          .flex.flex-nowrap.items-center(class="gap-1.5")
            span.inline-flex.shrink-0.items-center.rounded-full.border.px-2.text-xs.font-medium(
              class="gap-1.5 py-0.5"
              :class="taskStateClassFor(task)"
              :title="stateUpdatedTitle(task)"
            )
              component.size-3.shrink-0(:is="taskStateIcon(task.state)")
              | {{ $t(taskStateKeyFor(task)) }}
            span.text-muted-foreground.inline-flex.shrink-0.cursor-help.items-center(
              v-if="taskWaitingReason(task)"
              role="img"
              :title="taskWaitingReason(task)"
              :aria-label="taskWaitingReason(task)"
              data-testid="task-waiting-reason"
            )
              InfoIcon.size-3
            .border-border.divide-border.inline-flex.items-center.divide-x.overflow-hidden.rounded-md.border(
              v-if="rowActions(task).length"
            )
              .flex.w-9.items-stretch(
                v-for="action in rowActions(task)"
                :key="action.key"
              )
                Button.flex-1.rounded-none.px-2(
                  variant="ghost"
                  size="icon-sm"
                  :title="action.title"
                  :aria-label="action.title"
                  @click.stop="runRowAction(task, action)"
                  :disabled="actionBusy === task.id"
                )
                  component(:is="action.icon")
        TableCell(v-else-if="column.key === 'project'")
          RouterLink.inline-flex.rounded-full.border.px-2.text-xs.transition-colors(
            v-if="task.projectId && projectSlug(task.projectId)"
            class="hover:bg-muted/50 py-0.5"
            :to="{ name: 'project', params: { slug: projectSlug(task.projectId) } }"
            @click.stop
          ) {{ projectName(task.projectId) }}
          span.text-muted-foreground(v-else) —
        TableCell(v-else-if="column.key === 'title'")
          .flex.min-w-0.flex-col
            Skeleton.h-4.w-40(v-if="taskTitlePending(task)")
            .flex.min-w-0.items-center.gap-1(v-else)
              span.max-w-xs.truncate(
                :class="taskNameClass(task)"
                :title="taskName(task)"
              )
                | {{ taskName(task) }}
              span.inline-flex.shrink-0.items-center.rounded-full.bg-amber-500.px-1.text-xs.font-semibold.text-white(
                v-if="task.unreadComments > 0"
                class="gap-0.5"
                :title="$t('tasksTable.unreadComments', { count: task.unreadComments })"
                data-testid="task-unread-comments"
              )
                MessageSquareIcon.size-3
                span {{ task.unreadComments > 99 ? "99+" : task.unreadComments }}
              span.text-muted-foreground.shrink-0(
                v-if="task.archived && task.sessionId && !task.sessionDeletedAt"
                :title="$t('tasksTable.archivedOffline')"
              )
                HardDriveDownloadIcon.size-3.text-amber-500
              span.text-muted-foreground.shrink-0(
                v-else-if="task.sessionDeletedAt"
                :title="$t('tasksTable.sessionGone')"
              )
                CircleSlashIcon.size-3
            .flex.flex-wrap.items-center(class="gap-1.5")
              span.text-muted-foreground.font-mono.text-xs(:title="task.id")
                | {{ shortTaskId(task.id) }}
              span.text-muted-foreground.inline-flex.shrink-0.items-center.text-xs(
                v-if="task.parentTaskId"
                class="gap-0.5"
                :title="$t('tasksTable.subTask')"
                data-testid="task-subtask-badge"
              )
                GitBranchIcon.size-3
              span.text-muted-foreground.inline-flex.shrink-0.items-center.rounded-full.border.text-xs(
                v-if="!task.interactive"
                class="gap-0.5 px-1.5"
                :title="$t('tasksTable.nonInteractiveHint')"
                data-testid="task-non-interactive-badge"
              )
                MessageSquareOffIcon.size-3
                | {{ $t("tasksTable.nonInteractive") }}
        TableCell(v-else-if="column.key === 'progress'")
          .flex.flex-col.gap-1(v-if="taskProgressHasPlan(task.progress)")
            .flex.items-center(class="gap-1.5")
              .bg-muted.w-14.overflow-hidden.rounded-full(class="h-1.5")
                .h-full.rounded-full.bg-emerald-500(
                  :style="{ width: `${taskProgressPercent(task.progress)}%` }"
                )
              span.text-xs.tabular-nums
                | {{ taskProgressCount(task.progress) }}
              span.animate-pulse.rounded-full.bg-blue-500(
                v-if="task.state === TaskState.RUNNING && taskProgressInProgress(task.progress)"
                class="size-1.5"
                :title="$t('tasksTable.stepInProgress')"
              )
            span.text-muted-foreground.max-w-28.truncate.text-xs(
              v-if="task.progress?.summary"
              :title="task.progress.summary"
            ) {{ task.progress.summary }}
          span.text-muted-foreground.text-xs(
            v-else-if="taskPlanFlag(task.progress, task.size, task.state) === 'hint'"
            :title="$t('tasksTable.planningOptional')"
          ) {{ $t("tasksTable.noPlan") }}
          span.inline-flex.items-center.gap-1.text-xs.text-amber-600(
            v-else-if="taskPlanFlag(task.progress, task.size, task.state) === 'missing'"
            class="dark:text-amber-400"
            :title="$t('tasksTable.noPlanHint')"
          )
            CircleAlertIcon.size-3.shrink-0
            | {{ $t("tasksTable.noPlan") }}
          span.text-muted-foreground.text-xs(v-else) —
        TableCell(v-else-if="column.key === 'links'")
          .flex.flex-wrap.items-center.gap-1(v-if="task.links.length")
            a.text-muted-foreground.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
              v-for="link in task.links.slice(0, 2)"
              :key="link.url"
              rel="noopener noreferrer"
              class="hover:bg-muted/50 hover:text-foreground py-0.5"
              :href="link.url"
              :title="link.url"
              target="_blank"
              @click.stop
            )
              span.max-w-24.truncate {{ link.label || link.url }}
              ExternalLinkIcon.size-3.shrink-0
            span.text-muted-foreground.text-xs(v-if="task.links.length > 2")
              | +{{ task.links.length - 2 }}
          span.text-muted-foreground(v-else) —
        TableCell(v-else-if="column.key === 'tags'")
          .flex.flex-wrap.items-center.gap-1(v-if="task.tags.length")
            span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
              v-for="tag in task.tags.slice(0, 3)"
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
            span.text-muted-foreground.text-xs(v-if="task.tags.length > 3")
              | +{{ task.tags.length - 3 }}
          span.text-muted-foreground.text-xs(v-else) —
        TableCell(v-else-if="column.key === 'badges'")
          TaskBadges(:badges="task.badges" :limit="2")
        TableCell(v-else-if="column.key === 'references'")
          .flex.flex-wrap.items-center.gap-1(v-if="task.references.length")
            RouterLink.inline-flex.max-w-full.min-w-0.items-center.gap-1.rounded-full.border.px-2.text-xs.transition-colors(
              v-for="ref in task.references.slice(0, 3)"
              :key="ref.id"
              class="hover:bg-muted/50 py-0.5"
              :to="{ name: 'task', params: { id: ref.id } }"
              :class="taskStateClassFor(ref)"
              :title="taskReferenceTitle(ref)"
              @click.stop
            )
              LockIcon.size-3.shrink-0.text-amber-500(
                v-if="ref.kind === TaskReferenceKind.BLOCKING"
              )
              component.size-3.shrink-0(:is="taskStateIcon(ref.state)")
              strong.shrink-0.font-semibold {{ ref.id }}:
              span.max-w-32.truncate(:class="taskNameClass(ref)") {{ taskReferenceName(ref) }}
            span.text-muted-foreground.text-xs(
              v-if="task.references.length > 3"
            )
              | +{{ task.references.length - 3 }}
          span.text-muted-foreground.text-xs(v-else) —
        TableCell(v-else-if="column.key === 'priority'")
          span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
            class="py-0.5"
            :class="taskPriorityClass(task.priority)"
          )
            | {{ $t(taskPriorityKey(task.priority)) }}
        TableCell(v-else-if="column.key === 'size'")
          span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
            class="py-0.5"
            :class="taskSizeClass(task.size)"
          )
            | {{ $t(taskSizeKey(task.size)) }}
        TableCell(v-else-if="column.key === 'assignee'")
          .flex.items-center.gap-2(v-if="task.assigneeActorId")
            RouterLink.flex.items-center.gap-2(
              v-if="agentUsername(task.assigneeActorId)"
              class="hover:underline"
              :to="{ name: 'agent', params: { username: agentUsername(task.assigneeActorId) } }"
              @click.stop
            )
              span.size-5.shrink-0(
                :innerHTML="agentAvatar(task.assigneeActorId)"
              )
              span.truncate.text-sm {{ agentName(task.assigneeActorId) }}
            template(v-else)
              span.size-5.shrink-0(
                :innerHTML="agentAvatar(task.assigneeActorId)"
              )
              span.truncate.text-sm {{ agentName(task.assigneeActorId) }}
            PinIcon.text-muted-foreground.size-3.shrink-0(
              v-if="task.assigneePinned"
              :title="$t('tasksTable.pinned')"
              :aria-label="$t('tasksTable.pinned')"
            )
          span.text-muted-foreground.text-xs(v-else) {{ $t("tasks.toasts.unassigned") }}
        TableCell.text-muted-foreground.text-xs(
          v-else-if="column.key === 'cost'"
        )
          | {{ taskCost(task) }}
    TableRow(v-if="!tasks.length && !loading")
      TableCell.text-muted-foreground(:colspan="columnCount")
        | {{ emptyText }}
</template>

<script setup lang="ts">
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import TaskBadges from "@/components/tasks/TaskBadges.vue";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { useTaskColumns } from "@/composables/useTaskColumns";
import type { UsageBucket } from "@/composables/useUsage";
import {
  formatCost,
  formatRelativeTimestamp,
  formatTimestamp,
} from "@/lib/format";
import {
  shortTaskId,
  tagStyle,
  taskName,
  taskNameClass,
  taskPlanFlag,
  taskPriorityClass,
  taskPriorityKey,
  taskProgressCount,
  taskProgressHasPlan,
  taskProgressInProgress,
  taskProgressPercent,
  taskReferenceName,
  taskReferenceTitle,
  taskSizeClass,
  taskSizeKey,
  taskStateClassFor,
  taskStateIcon,
  taskStateKeyFor,
  taskTitlePending,
  taskWaitingReason,
} from "@/lib/task";
import { TaskReferenceKind, TaskState, type Task } from "@cherrypi/sdk";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  ArrowDownIcon,
  ArrowUpDownIcon,
  ArrowUpIcon,
  BanIcon,
  CheckIcon,
  CheckSquareIcon,
  CircleAlertIcon,
  CircleSlashIcon,
  EraserIcon,
  ExternalLinkIcon,
  GitBranchIcon,
  HardDriveDownloadIcon,
  InfoIcon,
  LockIcon,
  MessageSquareIcon,
  MessageSquareOffIcon,
  MinusSquareIcon,
  PinIcon,
  PlayIcon,
  RotateCcwIcon,
  SquareIcon,
  StarIcon,
  Undo2Icon,
  type LucideIcon,
} from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    tasks: Task[];
    loading: boolean;
    emptyText: string;
    usageByTask: Map<string, UsageBucket>;
    actionBusy?: string;
    sortKey?: string;
    sortDir?: "asc" | "desc";
    selected?: Set<string>;

    hideProject?: boolean;
    hideAssignee?: boolean;
  }>(),
  {
    actionBusy: "",
    sortKey: "state",
    sortDir: "desc",
    selected: () => new Set<string>(),
    hideProject: false,
    hideAssignee: false,
  },
);

const emit = defineEmits<{
  open: [id: string];
  enqueue: [task: Task];
  dequeue: [task: Task];
  retry: [task: Task];
  complete: [task: Task];
  cancel: [task: Task];
  archive: [task: Task];
  unarchive: [task: Task];
  cleanup: [task: Task];
  sort: [key: string];
  toggleSelect: [id: string];
  toggleSelectAll: [];
  toggleFavorite: [id: string];
}>();

const { columns } = useTaskColumns();

const visibleColumns = computed(() =>
  columns.value.filter((column) => {
    if (!column.visible) return false;
    if (column.key === "project" && props.hideProject) return false;
    if (column.key === "assignee" && props.hideAssignee) return false;
    return true;
  }),
);

const selectableCount = computed(() => props.tasks.length);
const allSelected = computed(
  () =>
    selectableCount.value > 0 &&
    props.tasks.every((t) => props.selected.has(t.id)),
);
const someSelected = computed(
  () => props.tasks.some((t) => props.selected.has(t.id)) && !allSelected.value,
);

const columnCount = computed(() => 2 + visibleColumns.value.length);

const skeletonWidths = computed(() => [
  "w-5",
  "w-4",
  ...visibleColumns.value.map((column) => column.skeleton),
]);

const { agents } = useAgents();
const { active: projects } = useProjects();

function sortIcon(key: string) {
  if (props.sortKey !== key) return ArrowUpDownIcon;
  return props.sortDir === "asc" ? ArrowUpIcon : ArrowDownIcon;
}

function stateUpdatedTitle(task: Task): string {
  const at = task.stateUpdatedAt;
  const updated = at
    ? `${formatTimestamp(at)} (${formatRelativeTimestamp(at)})`
    : "";

  return [taskWaitingReason(task), updated].filter(Boolean).join(" — ");
}

function projectName(id: string): string {
  return (
    projects.value.find((project) => project.id === id)?.name ?? id.slice(0, 8)
  );
}

function projectSlug(id: string): string {
  return projects.value.find((project) => project.id === id)?.slug ?? "";
}

function agentName(id: string): string {
  const item = agents.value.find(
    (agent) => agent.id === id || agent.actorId === id,
  );
  return item?.displayName || item?.username || id.slice(0, 8);
}

function agentUsername(id: string): string {
  return (
    agents.value.find((agent) => agent.id === id || agent.actorId === id)
      ?.username ?? ""
  );
}

function agentAvatar(id: string): string {
  return toSvg(id, 20);
}

function taskCost(task: Task): string {
  const usage = task.sessionId
    ? props.usageByTask.get(task.sessionId)
    : undefined;
  return usage ? formatCost(usage.cost) : "—";
}

type RowActionKey =
  | "enqueue"
  | "dequeue"
  | "retry"
  | "complete"
  | "cancel"
  | "archive"
  | "unarchive"
  | "cleanup";

interface RowAction {
  key: RowActionKey;
  title: string;
  icon: LucideIcon;
}

function rowActions(task: Task): RowAction[] {
  const actions = rowActionsFor(task);

  if (task.canCloseOut) return actions;
  return actions.filter(
    (action) =>
      action.key !== "complete" &&
      action.key !== "cancel" &&
      action.key !== "archive" &&
      action.key !== "unarchive",
  );
}

function rowActionsFor(task: Task): RowAction[] {
  if (task.archived) {
    const actions: RowAction[] = [
      { key: "unarchive", title: "Unarchive", icon: ArchiveRestoreIcon },
    ];
    if (task.sessionId && !task.sessionDeletedAt) {
      actions.unshift({
        key: "cleanup",
        title: t("tasksTable.cleanUpSession"),
        icon: EraserIcon,
      });
    }
    return actions;
  }
  switch (task.state) {
    case TaskState.BACKLOG:
      return task.startedAt
        ? []
        : [{ key: "enqueue", title: t("tasks.action.queue"), icon: PlayIcon }];
    case TaskState.QUEUED:
      return [
        {
          key: "dequeue",
          title: t("tasks.action.moveToBacklog"),
          icon: Undo2Icon,
        },
      ];

    case TaskState.WAITING:
      return [
        {
          key: "dequeue",
          title: t("tasks.action.moveToBacklog"),
          icon: Undo2Icon,
        },
        { key: "complete", title: t("tasks.action.complete"), icon: CheckIcon },
        { key: "cancel", title: "Cancel", icon: BanIcon },
      ];

    case TaskState.FAILED:
      return [
        { key: "retry", title: "Retry", icon: RotateCcwIcon },
        { key: "cancel", title: "Cancel", icon: BanIcon },
      ];
    case TaskState.SETTLED:
      return [
        { key: "complete", title: t("tasks.action.complete"), icon: CheckIcon },
        { key: "cancel", title: "Cancel", icon: BanIcon },
      ];

    case TaskState.COMPLETED:
    case TaskState.CANCELLED:
      return [{ key: "archive", title: t("tags.archive"), icon: ArchiveIcon }];
    default:
      return [];
  }
}

function runRowAction(task: Task, action: RowAction) {
  switch (action.key) {
    case "enqueue":
      emit("enqueue", task);
      break;
    case "dequeue":
      emit("dequeue", task);
      break;
    case "retry":
      emit("retry", task);
      break;
    case "complete":
      emit("complete", task);
      break;
    case "cancel":
      emit("cancel", task);
      break;
    case "archive":
      emit("archive", task);
      break;
    case "unarchive":
      emit("unarchive", task);
      break;
    case "cleanup":
      emit("cleanup", task);
      break;
  }
}
</script>

<i18n lang="yaml">
en:
  tasksTable:
    requiredTagHint: "Only runs on agents that declare this tag"
    stepInProgress: "A step is in progress"
    noPlanHint: "The agent has not published a plan for this task"
    nonInteractive: "Non-interactive"
    noPlan: "No plan"
    favoritesHint: "Favorites stay on top of every list"
    favorites: "Favorites"
    archivedOffline: "Archived while the agent was offline: its Pi session and workspace still need cleanup"
    sessionGone: "The agent-side Pi session and workspace are gone (cleaned up, or the agent was restarted with fresh data); the conversation is stored in the agency"
    subTask: "Sub-task"
    nonInteractiveHint: "Non-interactive: the agent cannot show questionnaires or extension dialogs"
    planningOptional: "Planning is optional for small tasks"
    pinned: "Pinned to this agent"
    cleanUpSession: "Clean up session"
    unarchive: "Unarchive"
    clearSelection: "Clear selection"
    selectAllInTab: "Select all in tab"
    selectTask: "Select task"
    deselectTask: "Deselect task"
    addFavorite: "Add to favorites"
    removeFavorite: "Remove from favorites"
    unreadComments: "{count} unread comment | {count} unread comments"
ru:
  tasksTable:
    requiredTagHint: "Выполняется только на агентах, которые объявляют этот тег"
    stepInProgress: "Шаг выполняется"
    noPlanHint: "Агент не опубликовал план для этой задачи"
    nonInteractive: "Неинтерактивная"
    noPlan: "Плана нет"
    favoritesHint: "Избранное всегда сверху списка"
    favorites: "Избранное"
    archivedOffline: "Архивировано, пока агент был не в сети: сеанс Pi и рабочее пространство всё ещё требуют очистки"
    sessionGone: "Сеанс Pi и рабочее пространство на стороне агента отсутствуют (очищены или агент перезапущен с новыми данными); переписка хранится в агентстве"
    subTask: "Подзадача"
    nonInteractiveHint: "Неинтерактивная: агент не может показывать опросники и диалоги расширений"
    planningOptional: "Для небольших задач планирование необязательно"
    pinned: "Закреплена за этим агентом"
    cleanUpSession: "Очистить сеанс"
    unarchive: "Разархивировать"
    clearSelection: "Снять выделение"
    selectAllInTab: "Выбрать все во вкладке"
    selectTask: "Выбрать задачу"
    deselectTask: "Снять выбор задачи"
    addFavorite: "Добавить в избранное"
    removeFavorite: "Убрать из избранного"
    unreadComments: "{count} непрочитанный комментарий | {count} непрочитанных комментария | {count} непрочитанных комментариев"
</i18n>
