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
.flex.flex-col.gap-2
  .bg-muted.inline-flex.flex-wrap.items-center.gap-1.self-start.rounded-lg.border(
    class="p-0.5"
    data-testid="task-tabs"
  )
    button.inline-flex.items-center.rounded-md.py-1.text-xs.font-medium.whitespace-nowrap.transition-colors(
      v-for="tab in TASK_STATE_TABS"
      :key="tab.key"
      type="button"
      class="gap-1.5 px-2.5"
      :class="tabClass(tab)"
      @click="activeTab = tab.key"
    )
      component.size-3.shrink-0(:is="taskStateIcon(tab.state)")
      span {{ tab.label }}
      span.font-normal(:class="countClass(tab.state)")
        | {{ countFor(tab.state) }}
  .flex.w-full.flex-wrap.items-center.gap-2
    .relative.min-w-0.flex-1
      SearchIcon.text-muted-foreground.pointer-events-none.absolute.size-4(
        class="top-1/2 left-2.5 -translate-y-1/2"
      )
      Input#task-search.w-full.pl-8(
        v-model="search"
        type="search"
        placeholder="Search title or prompt…"
        autocomplete="off"
      )
    Select(v-model="sizeFilter")
      SelectTrigger.w-32(size="sm" title="Filter by size")
        SelectValue(placeholder="Any size")
      SelectContent
        SelectGroup
          SelectItem(:value="TaskSize.UNSPECIFIED") Any size
          SelectItem(
            v-for="option in SIZE_FILTERS"
            :key="option.value"
            :value="option.value"
          ) {{ option.label }}
    slot(name="actions")

  .bg-card.sticky.top-0.z-20.flex.flex-wrap.items-center.gap-2.rounded-xl.border.px-3.py-2.shadow-sm(
    v-if="selectedTasks.length"
    data-testid="task-bulk-bar"
  )
    span.text-sm.font-medium
      | {{ selectedTasks.length }} selected
    template(v-if="tabActions.length")
      .mx-1.h-5.border-l
      Button(
        v-for="action in tabActions"
        :key="action.key"
        type="button"
        variant="outline"
        size="sm"
        :title="action.unavailableHint"
        @click="runStateAction(action)"
        :disabled="bulkBusy || action.unavailable"
      )
        component.size-3(:is="action.icon" data-icon="inline-start")
        span {{ action.label }}
    .mx-1.h-5.border-l
    Button(
      type="button"
      variant="outline"
      size="sm"
      :title="mergeHint"
      @click="mergeSelected"
      :disabled="bulkBusy || !mergeableSelection"
    )
      MergeIcon(data-icon="inline-start")
      span Merge
    Button(
      type="button"
      variant="outline"
      size="sm"
      :title="archiveHint"
      @click="requestArchive(selectedTasks)"
      :disabled="bulkBusy || !archiveableSelection"
    )
      ArchiveIcon(data-icon="inline-start")
      span Archive
    DropdownMenu
      DropdownMenuTrigger(as-child)
        Button(type="button" variant="outline" size="sm" :disabled="bulkBusy")
          FlagIcon(data-icon="inline-start")
          span Priority
      DropdownMenuContent.w-40(align="start")
        DropdownMenuLabel Set priority
        DropdownMenuSeparator
        DropdownMenuItem(
          v-for="option in BULK_PRIORITIES"
          :key="option.value"
          @click="applyPriority(option.value)"
        ) {{ option.label }}
    DropdownMenu
      DropdownMenuTrigger(as-child)
        Button(type="button" variant="outline" size="sm" :disabled="bulkBusy")
          RulerIcon(data-icon="inline-start")
          span Size
      DropdownMenuContent.w-40(align="start")
        DropdownMenuLabel Set size
        DropdownMenuSeparator
        DropdownMenuItem(
          v-for="option in BULK_SIZES"
          :key="option.value"
          @click="applySize(option.value)"
        ) {{ option.label }}
    DropdownMenu
      DropdownMenuTrigger(as-child)
        Button(type="button" variant="outline" size="sm" :disabled="bulkBusy")
          UserIcon(data-icon="inline-start")
          span Assignee
      DropdownMenuContent.w-56(align="start")
        DropdownMenuLabel Set assignee
        DropdownMenuSeparator
        DropdownMenuItem(@click="applyAssignee('')") Unassign
        DropdownMenuItem(
          v-for="option in assigneeOptions"
          :key="option.id"
          @click="applyAssignee(option.id)"
        ) {{ option.label }}
    DropdownMenu
      DropdownMenuTrigger(as-child)
        Button(
          type="button"
          variant="outline"
          size="sm"
          :title="tagsEnabled ? undefined : TAGS_HINT"
          :disabled="bulkBusy || !tagsEnabled"
        )
          TagsIcon(data-icon="inline-start")
          span Add tags
      DropdownMenuContent.w-56(align="start")
        DropdownMenuLabel Add tags
        DropdownMenuSeparator
        .text-muted-foreground.px-2.py-1.text-xs(v-if="!projectTags.length")
          | No tags in this project
        DropdownMenuItem(
          v-for="tag in projectTags"
          :key="tag.id"
          @click="applyAddTag(tag)"
        )
          span.size-2.shrink-0.rounded-full(
            :style="{ backgroundColor: tagColor(tag.color) }"
          )
          span {{ tag.name }}
    DropdownMenu
      DropdownMenuTrigger(as-child)
        Button(
          type="button"
          variant="outline"
          size="sm"
          :title="tagsEnabled ? undefined : TAGS_HINT"
          :disabled="bulkBusy || !tagsEnabled"
        )
          TagsIcon(data-icon="inline-start")
          span Remove tags
      DropdownMenuContent.w-56(align="start")
        DropdownMenuLabel Remove tags
        DropdownMenuSeparator
        .text-muted-foreground.px-2.py-1.text-xs(v-if="!removableTags.length")
          | No tags on the selection
        DropdownMenuItem(
          v-for="tag in removableTags"
          :key="tag.id"
          @click="applyRemoveTag(tag)"
        )
          span.size-2.shrink-0.rounded-full(
            :style="{ backgroundColor: tagColor(tag.color) }"
          )
          span {{ tag.name }}
    Button.ml-auto(
      type="button"
      variant="ghost"
      size="sm"
      @click="clearSelection"
    )
      XIcon(data-icon="inline-start")
      span Clear

  p.text-destructive.text-sm(v-if="error") {{ error }}

  .bg-card.overflow-hidden.rounded-xl.border
    TasksTable(
      :tasks="visibleTasks"
      :empty-text="emptyMessage"
      :usage-by-task="usageByTask"
      :action-busy="actionBusy"
      :sort-key="sortKey"
      :sort-dir="sortDir"
      :selected="selected"
      :hide-project="hideProject"
      :hide-assignee="hideAssignee"
      @open="open"
      @enqueue="enqueue"
      @promote="promote"
      @demote="demote"
      @dequeue="dequeue"
      @complete="complete"
      @cancel="cancel"
      @archive="archiveTask"
      @sort="toggleSort"
      @toggle-select="toggleSelect"
      @toggle-select-all="toggleSelectAll"
      @toggle-favorite="toggleFavorite"
      :loading="loading"
    )

  .bg-card.overflow-hidden.rounded-xl.border(
    v-if="archivedTasks.length || archivedLoading"
  )
    button.flex.w-full.items-center.gap-2.px-3.py-2.text-left.text-sm(
      type="button"
      @click="showArchived = !showArchived"
    )
      ChevronRightIcon.size-4.shrink-0.transition-transform(
        :class="showArchived ? 'rotate-90' : ''"
      )
      ArchiveIcon.size-4.shrink-0
      span.font-medium Archived
      span.text-muted-foreground
        | {{ archivedTasks.length }}
    .border-t(v-if="showArchived")
      .bg-card.sticky.top-0.z-20.flex.flex-wrap.items-center.gap-2.border-b.px-3.py-2(
        v-if="archivedSelectedTasks.length"
        data-testid="archive-bulk-bar"
      )
        span.text-sm.font-medium
          | {{ archivedSelectedTasks.length }} selected
        DropdownMenu
          DropdownMenuTrigger(as-child)
            Button(
              type="button"
              variant="outline"
              size="sm"
              :disabled="bulkBusy"
            )
              FlagIcon(data-icon="inline-start")
              span Priority
          DropdownMenuContent.w-40(align="start")
            DropdownMenuLabel Set priority
            DropdownMenuSeparator
            DropdownMenuItem(
              v-for="option in BULK_PRIORITIES"
              :key="option.value"
              @click="applyArchivedPriority(option.value)"
            ) {{ option.label }}
        DropdownMenu
          DropdownMenuTrigger(as-child)
            Button(
              type="button"
              variant="outline"
              size="sm"
              :disabled="bulkBusy"
            )
              RulerIcon(data-icon="inline-start")
              span Size
          DropdownMenuContent.w-40(align="start")
            DropdownMenuLabel Set size
            DropdownMenuSeparator
            DropdownMenuItem(
              v-for="option in BULK_SIZES"
              :key="option.value"
              @click="applyArchivedSize(option.value)"
            ) {{ option.label }}
        DropdownMenu
          DropdownMenuTrigger(as-child)
            Button(
              type="button"
              variant="outline"
              size="sm"
              :title="archivedTagsEnabled ? undefined : TAGS_HINT"
              :disabled="bulkBusy || !archivedTagsEnabled"
            )
              TagsIcon(data-icon="inline-start")
              span Add tags
          DropdownMenuContent.w-56(align="start")
            DropdownMenuLabel Add tags
            DropdownMenuSeparator
            .text-muted-foreground.px-2.py-1.text-xs(
              v-if="!archivedProjectTags.length"
            )
              | No tags in this project
            DropdownMenuItem(
              v-for="tag in archivedProjectTags"
              :key="tag.id"
              @click="applyArchivedAddTag(tag)"
            )
              span.size-2.shrink-0.rounded-full(
                :style="{ backgroundColor: tagColor(tag.color) }"
              )
              span {{ tag.name }}
        DropdownMenu
          DropdownMenuTrigger(as-child)
            Button(
              type="button"
              variant="outline"
              size="sm"
              :title="archivedTagsEnabled ? undefined : TAGS_HINT"
              :disabled="bulkBusy || !archivedTagsEnabled"
            )
              TagsIcon(data-icon="inline-start")
              span Remove tags
          DropdownMenuContent.w-56(align="start")
            DropdownMenuLabel Remove tags
            DropdownMenuSeparator
            .text-muted-foreground.px-2.py-1.text-xs(
              v-if="!archivedRemovableTags.length"
            )
              | No tags on the selection
            DropdownMenuItem(
              v-for="tag in archivedRemovableTags"
              :key="tag.id"
              @click="applyArchivedRemoveTag(tag)"
            )
              span.size-2.shrink-0.rounded-full(
                :style="{ backgroundColor: tagColor(tag.color) }"
              )
              span {{ tag.name }}
        Button.ml-auto(
          type="button"
          variant="ghost"
          size="sm"
          @click="clearArchivedSelection"
        )
          XIcon(data-icon="inline-start")
          span Clear
      .px-3.pt-3
        .bg-muted.inline-flex.flex-wrap.items-center.gap-1.self-start.rounded-lg.border(
          class="p-0.5"
          data-testid="archive-tabs"
        )
          button.inline-flex.items-center.rounded-md.py-1.text-xs.font-medium.whitespace-nowrap.transition-colors(
            v-for="tab in ARCHIVED_TABS"
            :key="tab.key"
            type="button"
            class="gap-1.5 px-2.5"
            :class="tabClassFor(archivedTab === tab.key, tab.state)"
            @click="archivedTab = tab.key"
          )
            component.size-3.shrink-0(:is="taskStateIcon(tab.state)")
            span {{ tab.label }}
            span.font-normal(:class="countClass(tab.state)")
              | {{ archivedCountFor(tab.state) }}
      .pt-2
        TasksTable(
          :empty-text="archivedEmptyMessage"
          :tasks="archivedVisibleTasks"
          :usage-by-task="usageByTask"
          :action-busy="actionBusy"
          :sort-key="sortKey"
          :sort-dir="sortDir"
          :hide-project="hideProject"
          :hide-assignee="hideAssignee"
          :selected="archivedSelected"
          @open="open"
          @enqueue="enqueue"
          @promote="promote"
          @demote="demote"
          @dequeue="dequeue"
          @complete="complete"
          @cancel="cancel"
          @unarchive="unarchive"
          @cleanup="cleanup"
          @toggle-favorite="toggleFavorite"
          @toggle-select="toggleArchivedSelect"
          @toggle-select-all="toggleArchivedSelectAll"
          :loading="archivedLoading"
        )

  ArchiveTaskDialog(
    :open="archiveDialog"
    :tasks="archiveTargets"
    :busy="bulkBusy"
    @update:open="archiveDialog = $event"
    @confirm="confirmArchive"
  )
</template>

<script setup lang="ts">
import ArchiveTaskDialog from "@/components/tasks/ArchiveTaskDialog.vue";
import TasksTable from "@/components/tasks/TasksTable.vue";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { useTags } from "@/composables/useTags";
import { useTasks } from "@/composables/useTasks";
import { useUsage } from "@/composables/useUsage";
import { accountApi, type UserPreferences } from "@/lib/account";
import { agency, agent } from "@/lib/sdk";
import {
  tagColor,
  TASK_SIZE_RANK,
  TASK_SIZES,
  TASK_STATE_RANK,
  TASK_STATE_TABS,
  taskPriorityLabel,
  taskSizeLabel,
  taskStateIcon,
  taskStateTextClass,
} from "@/lib/task";
import {
  TaskPriority,
  TaskSize,
  TaskState,
  type Tag,
  type Task,
  type TaskAttributeInput,
} from "@cherrypi/sdk";
import {
  ArchiveIcon,
  BanIcon,
  CheckIcon,
  ChevronRightIcon,
  FlagIcon,
  InboxIcon,
  MergeIcon,
  PlayIcon,
  RulerIcon,
  SearchIcon,
  TagsIcon,
  Undo2Icon,
  UserIcon,
  XIcon,
} from "@lucide/vue";
import { refDebounced } from "@vueuse/core";
import { computed, onMounted, ref, watch, type Component } from "vue";
import { useRouter } from "vue-router";
import { toast } from "vue3-toastify";

type SortKey = "state" | "priority" | "size" | "project" | "task" | "cost";

const props = withDefaults(
  defineProps<{
    projectId?: string;
    agentId?: string;
    emptyText?: string;

    hideProject?: boolean;
    hideAssignee?: boolean;

    teamId?: number;
    leadActorId?: string;
  }>(),
  { emptyText: "No tasks." },
);

const router = useRouter();

const search = ref("");
const debouncedSearch = refDebounced(search, 300);
const searchActive = computed(() => search.value.trim() !== "");

const sizeFilter = ref<TaskSize>(TaskSize.UNSPECIFIED);
const SIZE_FILTERS = TASK_SIZES;

const filter = computed(() => ({
  projectId: props.projectId,
  agentId: props.agentId,
  search: debouncedSearch.value.trim() || undefined,
  size:
    sizeFilter.value === TaskSize.UNSPECIFIED ? undefined : sizeFilter.value,
}));

const { tasks: allTasks, loading, error, refresh } = useTasks(filter);

function scopeTasks(list: Task[]): Task[] {
  return list.filter((task) => {
    if (
      props.teamId !== undefined &&
      !task.teams.some((team) => Number(team.id) === props.teamId)
    ) {
      return false;
    }
    if (
      props.leadActorId &&
      !task.leads.some((lead) => lead.actorId === props.leadActorId)
    ) {
      return false;
    }
    return true;
  });
}

const tasks = computed(() => scopeTasks(allTasks.value));

const archivedFilter = computed(() => ({ ...filter.value, archived: true }));
const {
  tasks: allArchivedTasks,
  loading: archivedLoading,
  refresh: refreshArchived,
} = useTasks(archivedFilter);
const archivedTasks = computed(() => scopeTasks(allArchivedTasks.value));
const showArchived = ref(false);

interface ArchivedTab {
  key: "all" | "completed" | "cancelled";
  label: string;
  state?: TaskState;
}
const ARCHIVED_TABS: ArchivedTab[] = [
  { key: "all", label: "All" },
  { key: "completed", label: "Completed", state: TaskState.COMPLETED },
  { key: "cancelled", label: "Cancelled", state: TaskState.CANCELLED },
];
const archivedTab = ref<ArchivedTab["key"]>("all");
const archivedVisibleTasks = computed(() => {
  const tab = ARCHIVED_TABS.find((item) => item.key === archivedTab.value);
  if (!tab?.state) return archivedTasks.value;
  return archivedTasks.value.filter((task) => task.state === tab.state);
});
const archivedEmptyMessage = computed(() => {
  const tab = ARCHIVED_TABS.find((item) => item.key === archivedTab.value);
  return tab?.state
    ? `No archived ${tab.label.toLowerCase()} tasks.`
    : "No archived tasks.";
});
function archivedCountFor(state?: TaskState): number {
  if (!state) return archivedTasks.value.length;
  return archivedTasks.value.filter((task) => task.state === state).length;
}
function tabClassFor(active: boolean, state?: TaskState): string[] {
  const shape = active
    ? "bg-card shadow-sm ring-1 ring-border"
    : "hover:bg-card/60";
  const color = state
    ? taskStateTextClass(state)
    : active
      ? "text-foreground"
      : "text-muted-foreground";
  return [shape, color];
}

const archivedSelected = ref<Set<string>>(new Set());
const archivedSelectedTasks = computed(() =>
  archivedVisibleTasks.value.filter((task) =>
    archivedSelected.value.has(task.id),
  ),
);

function toggleArchivedSelect(id: string) {
  const next = new Set(archivedSelected.value);
  if (next.has(id)) {
    next.delete(id);
  } else {
    next.add(id);
  }
  archivedSelected.value = next;
}

function toggleArchivedSelectAll() {
  const ids = archivedVisibleTasks.value.map((task) => task.id);
  const all =
    ids.length > 0 && ids.every((id) => archivedSelected.value.has(id));
  const next = new Set(archivedSelected.value);
  for (const id of ids) {
    if (all) next.delete(id);
    else next.add(id);
  }
  archivedSelected.value = next;
}

function clearArchivedSelection() {
  archivedSelected.value = new Set();
}

watch(archivedTab, () => clearArchivedSelection());

const sizeFilterActive = computed(
  () => sizeFilter.value !== TaskSize.UNSPECIFIED,
);
const emptyMessage = computed(() => {
  if (searchActive.value) return "No tasks match your search.";
  if (sizeFilterActive.value) return "No tasks match the size filter.";
  return props.emptyText;
});
const { agents, loadAgents, actorIdFor } = useAgents();
const { active: projects, loadProjects } = useProjects();

const { byTask: usageByTask } = useUsage(
  computed(() => props.agentId ?? ""),
  computed(() => props.projectId ?? ""),
);

const ACTIVE_TAB_KEY = "cherrypi:tasks:active-tab";

function loadActiveTab(): string {
  try {
    const stored = sessionStorage.getItem(ACTIVE_TAB_KEY);
    if (stored && TASK_STATE_TABS.some((tab) => tab.key === stored)) {
      return stored;
    }
  } catch {}
  return "all";
}

const activeTab = ref(loadActiveTab());

watch(activeTab, (key) => {
  try {
    sessionStorage.setItem(ACTIVE_TAB_KEY, key);
  } catch {}
});

const sortKey = ref<SortKey>("state");
const sortDir = ref<"asc" | "desc">("desc");

watch(activeTab, () => {
  sortKey.value = "state";
  sortDir.value = "desc";
});

function toggleSort(key: string) {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === "asc" ? "desc" : "asc";
    return;
  }
  sortKey.value = key as SortKey;

  sortDir.value = "asc";
}

function sortTasks(list: Task[]): Task[] {
  const dir = sortDir.value === "asc" ? 1 : -1;
  const key = sortKey.value;
  return [...list].sort((a, b) => {
    const favorite = Number(b.favorite) - Number(a.favorite);
    if (favorite !== 0) return favorite;
    let cmp = 0;
    switch (key) {
      case "state": {
        const rank =
          (TASK_STATE_RANK[a.state] ?? 9) - (TASK_STATE_RANK[b.state] ?? 9);
        if (rank !== 0) return rank;
        cmp = stateUpdatedMs(a) - stateUpdatedMs(b);
        break;
      }
      case "priority":
        cmp = (a.priority ?? 0) - (b.priority ?? 0);
        break;
      case "size":
        cmp = (TASK_SIZE_RANK[a.size] ?? 0) - (TASK_SIZE_RANK[b.size] ?? 0);
        break;
      case "project":
        cmp = projectName(a.projectId).localeCompare(projectName(b.projectId));
        break;
      case "task":
        cmp = Number((a.seq ?? 0n) - (b.seq ?? 0n));
        break;
      case "cost":
        cmp = costValue(a) - costValue(b);
        break;
    }
    cmp *= dir;
    if (cmp !== 0) return cmp;

    return (
      stateUpdatedMs(b) - stateUpdatedMs(a) ||
      Number((b.seq ?? 0n) - (a.seq ?? 0n))
    );
  });
}

const visibleTasks = computed(() => {
  const tab = TASK_STATE_TABS.find((item) => item.key === activeTab.value);
  const filtered = !tab?.state
    ? tasks.value
    : tasks.value.filter((task) => task.state === tab.state);
  return sortTasks(filtered);
});

const selected = ref<Set<string>>(new Set());

const selectedTasks = computed(() =>
  visibleTasks.value.filter((task) => selected.value.has(task.id)),
);

function toggleSelect(id: string) {
  const next = new Set(selected.value);
  if (next.has(id)) {
    next.delete(id);
  } else {
    next.add(id);
  }
  selected.value = next;
}

function toggleSelectAll() {
  const ids = visibleTasks.value.map((task) => task.id);
  const all = ids.length > 0 && ids.every((id) => selected.value.has(id));
  const next = new Set(selected.value);
  for (const id of ids) {
    if (all) next.delete(id);
    else next.add(id);
  }
  selected.value = next;
}

function clearSelection() {
  selected.value = new Set();
}

watch(activeTab, () => clearSelection());

const TAGS_HINT = "Select tasks from one project";
const selectionProjectId = computed(() => {
  const ids = new Set(selectedTasks.value.map((task) => task.projectId));
  return ids.size === 1 ? [...ids][0]! : "";
});
const tagsEnabled = computed(() => selectionProjectId.value !== "");
const { active: projectTags } = useTags(
  computed(() => selectionProjectId.value || undefined),
);

const removableTags = computed(() => {
  const byId = new Map<string, Tag>();
  for (const task of selectedTasks.value) {
    for (const tag of task.tags) byId.set(tag.id, tag);
  }
  return [...byId.values()].sort((a, b) => a.name.localeCompare(b.name));
});

const archivedSelectionProjectId = computed(() => {
  const ids = new Set(
    archivedSelectedTasks.value.map((task) => task.projectId),
  );
  return ids.size === 1 ? [...ids][0]! : "";
});
const archivedTagsEnabled = computed(
  () => archivedSelectionProjectId.value !== "",
);
const { active: archivedProjectTags } = useTags(
  computed(() => archivedSelectionProjectId.value || undefined),
);
const archivedRemovableTags = computed(() => {
  const byId = new Map<string, Tag>();
  for (const task of archivedSelectedTasks.value) {
    for (const tag of task.tags) byId.set(tag.id, tag);
  }
  return [...byId.values()].sort((a, b) => a.name.localeCompare(b.name));
});

interface BulkAction {
  key: string;
  label: string;
  done: string;
  icon: Component;
  run: (task: Task) => Promise<unknown>;

  unavailable?: boolean;
  unavailableHint?: string;
}

const COMPLETE_ACTION: BulkAction = {
  key: "complete",
  label: "Complete",
  done: "Completed",
  icon: CheckIcon,
  run: (task) => agent.closeOutTask(task.id, TaskState.COMPLETED),
};
const CANCEL_ACTION: BulkAction = {
  key: "cancel",
  label: "Cancel",
  done: "Cancelled",
  icon: BanIcon,
  run: (task) => agent.closeOutTask(task.id, TaskState.CANCELLED),
};

const MOVE_TO_BACKLOG_ACTION: BulkAction = {
  key: "reopen",
  label: "Move to Backlog",
  done: "Moved to backlog",
  icon: InboxIcon,
  run: (task) => agent.reopenTask(task.id, { queue: false }),
};

const tabActions = computed<BulkAction[]>(() => {
  const state = TASK_STATE_TABS.find(
    (tab) => tab.key === activeTab.value,
  )?.state;
  if (state === TaskState.DRAFT) {
    return [
      {
        key: "enqueue",
        label: "Queue",
        done: "Queued",
        icon: PlayIcon,
        run: (task) => agent.queueTask(task.id),
      },
      {
        key: "promote",
        label: "Move to Backlog",
        done: "Moved to backlog",
        icon: InboxIcon,
        run: (task) => agent.promoteTask(task.id),
      },
      COMPLETE_ACTION,
      CANCEL_ACTION,
    ];
  }
  if (state === TaskState.BACKLOG) {
    const postponed = selectedTasks.value.some((task) =>
      Boolean(task.startedAt),
    );
    return [
      {
        key: "enqueue",
        label: "Queue",
        done: "Queued",
        icon: PlayIcon,
        run: (task) => agent.queueTask(task.id),
        unavailable: postponed,
        unavailableHint:
          "A task that already started is queued by sending a follow-up",
      },
      {
        key: "demote",
        label: "Back to Draft",
        done: "Moved to draft",
        icon: Undo2Icon,
        run: (task) => agent.demoteTask(task.id),
        unavailable: postponed,
        unavailableHint: "A task that already started cannot return to Draft",
      },
      COMPLETE_ACTION,
      CANCEL_ACTION,
    ];
  }
  if (state === TaskState.QUEUED) {
    return [
      {
        key: "dequeue",
        label: "Back to Backlog",
        done: "Moved to backlog",
        icon: Undo2Icon,
        run: (task) => agent.dequeueTask(task.id),
      },
    ];
  }
  if (state === TaskState.SETTLED) {
    return [MOVE_TO_BACKLOG_ACTION, COMPLETE_ACTION, CANCEL_ACTION];
  }
  if (state === TaskState.FAILED) {
    return [COMPLETE_ACTION, CANCEL_ACTION];
  }

  if (state === TaskState.COMPLETED) {
    return [MOVE_TO_BACKLOG_ACTION, CANCEL_ACTION];
  }
  if (state === TaskState.CANCELLED) {
    return [MOVE_TO_BACKLOG_ACTION, COMPLETE_ACTION];
  }
  return [];
});

const BULK_PRIORITIES = [
  TaskPriority.LOW,
  TaskPriority.NORMAL,
  TaskPriority.HIGH,
  TaskPriority.CRITICAL,
].map((value) => ({ value, label: taskPriorityLabel(value) }));

const BULK_SIZES = TASK_SIZES.map(({ value }) => ({
  value,
  label: taskSizeLabel(value),
}));

const assigneeOptions = computed(() =>
  agents.value.map((item) => ({
    id: item.id,
    label: item.displayName || item.username,
  })),
);

const bulkBusy = ref(false);

const MERGEABLE_STATES: TaskState[] = [
  TaskState.DRAFT,
  TaskState.BACKLOG,
  TaskState.QUEUED,
  TaskState.FAILED,
];
const mergeableSelection = computed(
  () =>
    selectedTasks.value.length >= 2 &&
    selectedTasks.value.every((task) => MERGEABLE_STATES.includes(task.state)),
);

const CLOSED_OUT_STATES: TaskState[] = [
  TaskState.COMPLETED,
  TaskState.CANCELLED,
];
const archiveableSelection = computed(
  () =>
    selectedTasks.value.length > 0 &&
    selectedTasks.value.every((task) => CLOSED_OUT_STATES.includes(task.state)),
);
const archiveHint = computed(() =>
  archiveableSelection.value
    ? "Archive selected tasks"
    : "Only completed or cancelled tasks can be archived",
);
const mergeHint = computed(() => {
  if (selectedTasks.value.length < 2) return "Select at least two tasks";
  if (!mergeableSelection.value)
    return "Only draft, backlog, queued, or failed tasks can be merged";
  return "Merge selected tasks";
});

function reportBulk(done: string, total: number, failed: number) {
  const tasks = `${total} task${total === 1 ? "" : "s"}`;
  if (failed === 0) {
    toast.success(`${done} for ${tasks}`);
  } else if (failed >= total) {
    toast.error(`No tasks updated (${tasks})`);
  } else {
    toast.warning(`${done} for ${total - failed} of ${tasks}`);
  }
}

async function runStateAction(action: BulkAction) {
  const targets = selectedTasks.value;
  if (!targets.length || bulkBusy.value) return;
  bulkBusy.value = true;
  const outcomes = await Promise.allSettled(
    targets.map((task) => action.run(task)),
  );
  bulkBusy.value = false;
  await refresh();
  const failed = outcomes.filter(
    (outcome) => outcome.status === "rejected",
  ).length;
  reportBulk(action.done, targets.length, failed);
}

async function mergeSelected() {
  const targets = selectedTasks.value;
  if (targets.length < 2 || bulkBusy.value) return;
  bulkBusy.value = true;
  try {
    const merged = await agent.mergeTasks(targets.map((task) => task.id));
    await refresh();
    clearSelection();
    toast.success(`Merged into ${merged.id}`);
    void router.push({ name: "task", params: { id: merged.id } });
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    bulkBusy.value = false;
  }
}

async function applyAttribute(
  attributes: TaskAttributeInput,
  done: string,
  targets: Task[] = selectedTasks.value,
) {
  if (!targets.length || bulkBusy.value) return;
  bulkBusy.value = true;
  try {
    const results = await agent.bulkUpdateTask({
      ids: targets.map((task) => task.id),
      attributes,
    });
    const failed = results.filter((result) => result.error).length;
    await Promise.all([refresh(), refreshArchived()]);
    reportBulk(done, targets.length, failed);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    bulkBusy.value = false;
  }
}

function applyPriority(priority: TaskPriority, targets?: Task[]) {
  void applyAttribute({ priority }, "Priority set", targets);
}

function applySize(size: TaskSize, targets?: Task[]) {
  void applyAttribute({ size }, "Size set", targets);
}

function applyAssignee(agentId: string) {
  void applyAttribute(
    { assigneeActorId: agentId ? actorIdFor(agentId) : "" },
    agentId ? "Assigned" : "Unassigned",
  );
}

function applyAddTag(tag: Tag, targets?: Task[]) {
  void applyAttribute({ addTagIds: [tag.id] }, "Tag added", targets);
}

function applyRemoveTag(tag: Tag, targets?: Task[]) {
  void applyAttribute({ removeTagIds: [tag.id] }, "Tag removed", targets);
}

function applyArchivedPriority(priority: TaskPriority) {
  applyPriority(priority, archivedSelectedTasks.value);
}

function applyArchivedSize(size: TaskSize) {
  applySize(size, archivedSelectedTasks.value);
}

function applyArchivedAddTag(tag: Tag) {
  applyAddTag(tag, archivedSelectedTasks.value);
}

function applyArchivedRemoveTag(tag: Tag) {
  applyRemoveTag(tag, archivedSelectedTasks.value);
}

const prefs = ref<UserPreferences>({ notify_first_archive: true });
const archiveDialog = ref(false);
const archiveTargets = ref<Task[]>([]);

onMounted(async () => {
  try {
    prefs.value = await accountApi.getPreferences();
  } catch {}
});

function requestArchive(targets: Task[]) {
  if (!targets.length) return;
  if (prefs.value.notify_first_archive) {
    archiveTargets.value = targets;
    archiveDialog.value = true;
    return;
  }
  void runArchive(targets);
}

async function confirmArchive(dontShowAgain: boolean) {
  archiveDialog.value = false;
  if (dontShowAgain && prefs.value.notify_first_archive) {
    prefs.value = { ...prefs.value, notify_first_archive: false };
    try {
      await accountApi.updatePreferences({ notify_first_archive: false });
    } catch {}
  }
  const targets = archiveTargets.value;
  archiveTargets.value = [];
  await runArchive(targets);
}

async function runArchive(targets: Task[]) {
  if (!targets.length) return;
  bulkBusy.value = true;
  const outcomes = await Promise.allSettled(
    targets.map((task) => agent.archiveTask(task.id)),
  );
  bulkBusy.value = false;
  await Promise.all([refresh(), refreshArchived()]);
  const failed = outcomes.filter(
    (outcome) => outcome.status === "rejected",
  ).length;
  reportBulk("Archived", targets.length, failed);
}

function stateUpdatedMs(task: Task): number {
  const ts = task.stateUpdatedAt ?? task.updatedAt ?? task.createdAt;
  return ts
    ? Number(ts.seconds) * 1000 + Math.floor(Number(ts.nanos) / 1e6)
    : 0;
}

function tabClass(tab: (typeof TASK_STATE_TABS)[number]): string[] {
  return tabClassFor(activeTab.value === tab.key, tab.state);
}

function countClass(state?: TaskState): string {
  return state ? taskStateTextClass(state) : "";
}

function countFor(state?: TaskState): number {
  if (!state) return tasks.value.length;
  return tasks.value.filter((task) => task.state === state).length;
}

function projectName(id: string): string {
  return (
    projects.value.find((project) => project.id === id)?.name ?? id.slice(0, 8)
  );
}

function costValue(task: Task): number {
  const usage = task.sessionId
    ? usageByTask.value.get(task.sessionId)
    : undefined;
  return usage?.cost ?? 0;
}

function open(id: string) {
  void router.push({ name: "task", params: { id } });
}

const actionBusy = ref("");

async function runRowAction(
  task: Task,
  action: () => Promise<unknown>,
  done: string,
) {
  actionBusy.value = task.id;
  try {
    await action();
    await Promise.all([refresh(), refreshArchived()]);
    if (done) toast.success(done);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = "";
  }
}

async function toggleFavorite(id: string) {
  const task =
    allTasks.value.find((item) => item.id === id) ??
    allArchivedTasks.value.find((item) => item.id === id);
  if (!task) return;
  const next = !task.favorite;
  task.favorite = next;
  try {
    await agency.setTaskFavorite(id, { favorite: next });
  } catch (err) {
    task.favorite = !next;
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

function enqueue(task: Task) {
  void runRowAction(task, () => agent.queueTask(task.id), "");
}

function promote(task: Task) {
  void runRowAction(task, () => agent.promoteTask(task.id), "");
}

function demote(task: Task) {
  void runRowAction(task, () => agent.demoteTask(task.id), "");
}

function dequeue(task: Task) {
  void runRowAction(task, () => agent.dequeueTask(task.id), "");
}

function unarchive(task: Task) {
  void runRowAction(
    task,
    () => agent.unarchiveTask(task.id),
    "Task unarchived",
  );
}

function cleanup(task: Task) {
  void runRowAction(
    task,
    () => agent.cleanupTaskSession(task.id),
    "Session cleaned up",
  );
}

async function closeOut(task: Task, outcome: TaskState) {
  await runRowAction(
    task,
    () => agent.closeOutTask(task.id, outcome),
    outcome === TaskState.CANCELLED ? "Task cancelled" : "Task completed",
  );
}

function complete(task: Task) {
  void closeOut(task, TaskState.COMPLETED);
}

function cancel(task: Task) {
  void closeOut(task, TaskState.CANCELLED);
}

function archiveTask(task: Task) {
  requestArchive([task]);
}

onMounted(() => {
  void loadAgents();
  void loadProjects();
});
</script>
