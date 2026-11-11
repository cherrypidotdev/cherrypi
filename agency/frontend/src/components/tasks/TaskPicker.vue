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
.bg-popover.text-popover-foreground.overflow-hidden.rounded-md.border.shadow-md(
  :class="embedded ? 'rounded-none border-0 bg-transparent shadow-none' : ''"
  data-testid="task-picker"
)
  .border-border.flex.flex-wrap.items-center.gap-x-3.border-b.px-2(
    class="gap-y-1.5 py-1.5"
  )
    label.flex.items-center.gap-1
      span.text-muted-foreground.text-xs {{ $t("domainLog.kind.state") }}
      select.cursor-pointer.rounded-md.border.bg-transparent.px-1.text-xs(
        v-model="stateFilter"
        class="py-0.5"
        :aria-label="$t('taskPicker.filterState')"
        data-testid="task-picker-state-filter"
      )
        option(value) {{ $t("taskPicker.anyState") }}
        option(v-for="tab in STATE_FILTERS" :key="tab.key" :value="tab.state") {{ $t(tab.labelKey) }}
    label.flex.items-center.gap-1(v-if="showProjectFilter")
      span.text-muted-foreground.text-xs {{ $t("admin.projects.columnProject") }}
      select.max-w-40.cursor-pointer.rounded-md.border.bg-transparent.px-1.text-xs(
        v-model="projectFilter"
        class="py-0.5"
        :aria-label="$t('taskPicker.filterProject')"
        data-testid="task-picker-project-filter"
      )
        option(value) {{ $t("taskPicker.anyProject") }}
        option(
          v-for="project in projectOptions"
          :key="project.id"
          :value="project.id"
        ) {{ project.name }}
    label.flex.items-center.gap-1
      span.text-muted-foreground.text-xs {{ $t("domainLog.kind.assignee") }}
      select.max-w-40.cursor-pointer.rounded-md.border.bg-transparent.px-1.text-xs(
        v-model="agentFilter"
        class="py-0.5"
        :aria-label="$t('taskPicker.filterAssignee')"
        data-testid="task-picker-agent-filter"
      )
        option(value) {{ $t("taskPicker.anyAgent") }}
        option(v-for="agent in agentOptions" :key="agent.id" :value="agent.id") {{ agent.displayName || agent.username }}
    label.flex.items-center.gap-1
      span.text-muted-foreground.text-xs {{ $t("domainLog.kind.priority") }}
      select.cursor-pointer.rounded-md.border.bg-transparent.px-1.text-xs(
        v-model="priorityFilter"
        class="py-0.5"
        :aria-label="$t('taskPicker.filterPriority')"
        data-testid="task-picker-priority-filter"
      )
        option(value) {{ $t("taskPicker.anyPriority") }}
        option(
          v-for="option in PRIORITY_FILTERS"
          :key="option.value"
          :value="option.value"
        ) {{ option.label }}
    label.flex.items-center.gap-1
      span.text-muted-foreground.text-xs {{ $t("domainLog.kind.size") }}
      select.cursor-pointer.rounded-md.border.bg-transparent.px-1.text-xs(
        v-model="sizeFilter"
        class="py-0.5"
        :aria-label="$t('taskPicker.filterSize')"
        data-testid="task-picker-size-filter"
      )
        option(value) {{ $t("tasks.filter.anySize") }}
        option(
          v-for="option in SIZE_FILTERS"
          :key="option.value"
          :value="option.value"
        ) {{ $t(option.labelKey) }}
    label.flex.items-center.gap-1
      span.text-muted-foreground.text-xs {{ $t("tags.columnTag") }}
      select.max-w-40.cursor-pointer.rounded-md.border.bg-transparent.px-1.text-xs(
        v-model="tagFilter"
        class="py-0.5"
        :aria-label="$t('taskPicker.filterTag')"
        data-testid="task-picker-tag-filter"
      )
        option(value) {{ $t("taskPicker.anyTag") }}
        option(v-for="tag in tagOptions" :key="tag.id" :value="tag.id") {{ tag.name }}
  .max-h-56.overflow-y-auto.p-1(
    ref="list"
    role="listbox"
    @scroll.passive="onScroll"
    data-testid="task-picker-list"
  )
    p.text-muted-foreground.px-2.text-xs(v-if="loading && !candidates.length")
      | {{ $t("taskPicker.searching") }}
    p.text-muted-foreground.px-2.text-xs(
      v-else-if="!candidates.length && !loading"
    )
      | {{ $t("taskPicker.empty") }}
    template(v-else)
      p.text-muted-foreground.px-2.text-xs(v-if="!search.trim()")
        | {{ $t("taskPicker.recent") }}
      button.inline-flex.w-full.min-w-0.items-center.gap-1.rounded-full.border.px-2.text-left.text-xs(
        v-for="(candidate, index) in candidates"
        :key="candidate.id"
        type="button"
        role="option"
        class="py-0.5"
        :class="candidateClass(candidate, index)"
        :aria-selected="index === activeIndex"
        :title="taskReferenceTitle(candidate)"
        @click="pick(candidate)"
        @mouseenter="activeIndex = index"
        data-task-candidate
      )
        component.size-3.shrink-0(:is="taskStateIcon(candidate.state)")
        strong.shrink-0.font-semibold {{ candidate.id }}:
        span.truncate(:class="taskNameClass(candidate)") {{ taskReferenceName(candidate) }}
      .flex.justify-center.py-1(v-if="loading")
        Spinner.size-3
      p.text-muted-foreground.px-2.text-xs(
        v-else-if="!hasMore && candidates.length"
      )
        | {{ $t("taskPicker.endOfList") }}
</template>

<script setup lang="ts">
import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { sortProjectsByName } from "@/lib/project";
import { agency } from "@/lib/sdk";
import {
  TASK_SIZES,
  TASK_STATE_TABS,
  taskNameClass,
  taskPriorityKey,
  taskReferenceName,
  taskReferenceTitle,
  taskStateClassFor,
  taskStateIcon,
} from "@/lib/task";
import {
  TaskPriority,
  TaskSize,
  TaskState,
  type Tag,
  type Task,
} from "@cherrypi/sdk";
import { refDebounced } from "@vueuse/core";
import { computed, nextTick, onMounted, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const PAGE_SIZE = 20;

const STATE_FILTERS = TASK_STATE_TABS.filter(
  (tab): tab is { key: string; labelKey: string; state: TaskState } =>
    tab.state !== undefined,
);

const PRIORITY_FILTERS = [
  TaskPriority.LOW,
  TaskPriority.NORMAL,
  TaskPriority.HIGH,
  TaskPriority.CRITICAL,
].map((value) => ({ value, label: t(taskPriorityKey(value)) }));

const SIZE_FILTERS = TASK_SIZES;

const props = withDefaults(
  defineProps<{
    search?: string;

    referenceProjectId?: string;

    projectFilter?: string;

    showProjectFilter?: boolean;

    excludeIds?: string[];

    embedded?: boolean;
  }>(),
  {
    search: "",
    referenceProjectId: "",
    projectFilter: "",
    showProjectFilter: false,
    excludeIds: () => [],
    embedded: false,
  },
);

const emit = defineEmits<{ pick: [Task] }>();

const { active: projects, loadProjects } = useProjects();
const { agents, loadAgents } = useAgents();

const list = ref<HTMLElement | null>(null);
const search = toRef(props, "search");
const debouncedSearch = refDebounced(search, 300);
const stateFilter = ref<TaskState | "">("");
const projectFilter = ref(props.projectFilter);
const agentFilter = ref("");
const priorityFilter = ref<TaskPriority | "">("");
const sizeFilter = ref<TaskSize | "">("");
const tagFilter = ref("");
const tags = ref<Tag[]>([]);
const candidates = ref<Task[]>([]);
const loading = ref(false);
const hasMore = ref(true);
const offset = ref(0);
const activeIndex = ref(0);

const projectOptions = computed(() => sortProjectsByName(projects.value));
const agentOptions = computed(() =>
  [...agents.value].sort((a, b) =>
    (a.displayName || a.username).localeCompare(b.displayName || b.username),
  ),
);
const tagOptions = computed(() =>
  [...tags.value].sort((a, b) => a.name.localeCompare(b.name)),
);

const tagProjectId = computed(
  () => projectFilter.value || props.referenceProjectId || "",
);

let generation = 0;
let tagGeneration = 0;

onMounted(() => {
  void Promise.all([loadProjects(), loadAgents(), loadPage(true)]);
});

async function loadTags() {
  const gen = ++tagGeneration;
  try {
    const next = await agency.listTags({
      projectId: tagProjectId.value,
      includeArchived: true,
    });
    if (gen !== tagGeneration) return;
    tags.value = next.filter((tag) => !tag.archived);

    if (
      tagFilter.value &&
      !tags.value.some((tag) => tag.id === tagFilter.value)
    ) {
      tagFilter.value = "";
    }
  } catch (err) {
    if (gen === tagGeneration) {
      tags.value = [];
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }
}

watch(tagProjectId, () => void loadTags(), { immediate: true });
watch(
  () => props.projectFilter,
  (value) => {
    projectFilter.value = value;
  },
);

function excluded() {
  return new Set(props.excludeIds);
}

function mergeCandidates(existing: Task[], incoming: Task[]): Task[] {
  const excludedIds = excluded();
  const seen = new Set(existing.map((task) => task.id));
  const merged = [...existing];
  for (const task of incoming) {
    if (excludedIds.has(task.id) || seen.has(task.id)) continue;
    seen.add(task.id);
    merged.push(task);
  }
  return merged;
}

async function loadPage(reset: boolean) {
  if (loading.value && !reset) return;
  const gen = ++generation;
  const query = debouncedSearch.value.trim();
  const state = stateFilter.value;
  const project = projectFilter.value;
  const agent = agentFilter.value;
  const priority = priorityFilter.value;
  const size = sizeFilter.value;
  const tag = tagFilter.value;
  const referenceProject = props.referenceProjectId;
  if (reset) {
    offset.value = 0;
    hasMore.value = true;
    candidates.value = [];
  }
  loading.value = true;
  try {
    const page = await agency.listTasks({
      projectId: project || undefined,
      referenceProjectId: referenceProject || undefined,
      agentId: agent || undefined,
      state: state === "" ? undefined : state,
      priority: priority === "" ? undefined : priority,
      size: size === "" ? undefined : size,
      tagId: tag || undefined,
      search: query || undefined,
      limit: PAGE_SIZE + 1,
      offset: reset ? 0 : offset.value,
    });

    if (
      gen !== generation ||
      props.referenceProjectId !== referenceProject ||
      projectFilter.value !== project ||
      agentFilter.value !== agent ||
      stateFilter.value !== state ||
      priorityFilter.value !== priority ||
      sizeFilter.value !== size ||
      tagFilter.value !== tag ||
      debouncedSearch.value.trim() !== query
    ) {
      return;
    }
    const hasNext = page.length > PAGE_SIZE;
    const visible = hasNext ? page.slice(0, PAGE_SIZE) : page;
    candidates.value = mergeCandidates(candidates.value, visible);
    offset.value = (reset ? 0 : offset.value) + visible.length;
    hasMore.value = hasNext;

    if (!candidates.value.length && hasNext) {
      void loadPage(false);
    }
  } catch (err) {
    if (gen === generation) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  } finally {
    if (gen === generation) loading.value = false;
  }
}

function restart() {
  activeIndex.value = 0;
  void loadPage(true);
}

function resetFilters() {
  stateFilter.value = "";
  agentFilter.value = "";
  priorityFilter.value = "";
  sizeFilter.value = "";
  tagFilter.value = "";
  projectFilter.value = props.projectFilter;
  restart();
}

function onScroll(event: Event) {
  const element = event.currentTarget as HTMLElement;
  const nearBottom =
    element.scrollTop + element.clientHeight >= element.scrollHeight - 24;
  if (nearBottom && hasMore.value && !loading.value) void loadPage(false);
}

function candidateClass(candidate: Task, index: number) {
  return [
    taskStateClassFor(candidate),
    index === activeIndex.value ? "bg-muted" : "hover:bg-muted",
  ];
}

function pick(candidate: Task) {
  emit("pick", candidate);
  candidates.value = candidates.value.filter(
    (item) => item.id !== candidate.id,
  );
  if (activeIndex.value >= candidates.value.length) {
    activeIndex.value = Math.max(0, candidates.value.length - 1);
  }
}

async function ensureActiveVisible() {
  await nextTick();
  const rows = list.value?.querySelectorAll<HTMLElement>(
    "[data-task-candidate]",
  );
  rows?.[activeIndex.value]?.scrollIntoView({ block: "nearest" });
}

function moveActive(delta: number) {
  const count = candidates.value.length;
  if (!count) return;
  activeIndex.value = (activeIndex.value + delta + count) % count;
  void ensureActiveVisible();
}

function pickActive(): boolean {
  const candidate = candidates.value[activeIndex.value];
  if (!candidate) return false;
  pick(candidate);
  return true;
}

defineExpose({ moveActive, pickActive });

watch(
  () => candidates.value.length,
  (length) => {
    if (activeIndex.value >= length) {
      activeIndex.value = Math.max(0, length - 1);
    }
  },
);

watch(debouncedSearch, restart);
watch(
  [
    stateFilter,
    projectFilter,
    agentFilter,
    priorityFilter,
    sizeFilter,
    tagFilter,
  ],
  restart,
);
watch(
  () => props.referenceProjectId,
  () => {
    resetFilters();
  },
);
watch(
  () => props.excludeIds,
  (ids) => {
    const excludedIds = new Set(ids);
    candidates.value = candidates.value.filter(
      (task) => !excludedIds.has(task.id),
    );

    if (!loading.value) {
      const element = list.value;
      if (
        element &&
        element.scrollHeight <= element.clientHeight &&
        hasMore.value
      ) {
        void loadPage(false);
      }
    }
  },
);
</script>

<i18n lang="yaml">
en:
  taskPicker:
    anyState: "Any state"
    anyProject: "Any project"
    anyAgent: "Any agent"
    anyPriority: "Any priority"
    anyTag: "Any tag"
    searching: "Searching…"
    empty: "No available tasks."
    recent: "Recent tasks"
    endOfList: "End of list."
    filterState: "Filter tasks by state"
    filterProject: "Filter tasks by project"
    filterAssignee: "Filter tasks by assignee"
    filterPriority: "Filter tasks by priority"
    filterSize: "Filter tasks by size"
    filterTag: "Filter tasks by tag"
ru:
  taskPicker:
    anyState: "Любое состояние"
    anyProject: "Любой проект"
    anyAgent: "Любой агент"
    anyPriority: "Любой приоритет"
    anyTag: "Любой тег"
    searching: "Поиск…"
    empty: "Доступных задач нет."
    recent: "Недавние задачи"
    endOfList: "Конец списка."
    filterState: "Фильтр задач по состоянию"
    filterProject: "Фильтр задач по проекту"
    filterAssignee: "Фильтр задач по исполнителю"
    filterPriority: "Фильтр задач по приоритету"
    filterSize: "Фильтр задач по размеру"
    filterTag: "Фильтр задач по тегу"
</i18n>
