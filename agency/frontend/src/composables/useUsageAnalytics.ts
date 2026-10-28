// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { useRealtime } from "@/composables/useRealtime";
import { useUsage, type UsageBucket } from "@/composables/useUsage";
import { agency } from "@/lib/sdk";
import {
  DAY_MS,
  usageAccent,
  usageRangeBounds,
  utcDayStart,
  type UsageRange,
  type UsageSummary,
} from "@/lib/usage";
import { TaskState, type Tag, type Task } from "@cherrypi/sdk";
import {
  computed,
  onMounted,
  ref,
  toValue,
  watch,
  type MaybeRefOrGetter,
} from "vue";

export type UsageContext = "project" | "agent" | "agency";

export interface UsageSeriesPoint {
  date: Date;
  cost: number;
  input: number;
  output: number;
  runs: number;
  completed: number;
}

export interface UsageTagSeries {
  id: string;
  name: string;
  color: string;
  points: { date: Date; value: number }[];
}

export interface UsageBreakdownRow {
  id: string;
  name: string;
  accent: string;
  cost: number;
  runs: number;
  tasks: number;
  tokens: number;
}

export interface UsageTaskRow {
  taskId: string;
  task?: Task;
  name: string;
  agentId: string;
  agentName: string;
  projectId: string;
  projectName: string;
  tags: Tag[];
  state: TaskState;
  archived: boolean;
  inputTokens: number;
  outputTokens: number;
  cost: number;
  activeMs: number;
  runs: number;
  live: boolean;
}

export interface UsageAnalyticsOptions {
  context: MaybeRefOrGetter<UsageContext>;
  agentId: MaybeRefOrGetter<string>;
  projectId: MaybeRefOrGetter<string>;
  range: MaybeRefOrGetter<UsageRange>;
  includeArchived: MaybeRefOrGetter<boolean>;
}

function timestampMs(ts: { seconds: bigint } | undefined): number {
  return ts ? Number(ts.seconds) * 1000 : 0;
}

function dayKey(ms: number): number {
  return utcDayStart(ms);
}

function daysBetween(startMs: number, endMs: number): number[] {
  const days: number[] = [];
  for (let day = dayKey(startMs); day <= dayKey(endMs); day += DAY_MS) {
    days.push(day);
  }
  return days;
}

export function useUsageAnalytics(options: UsageAnalyticsOptions) {
  const { agents, loadAgents, agentIdForActor } = useAgents();
  const { projects, loadProjects } = useProjects();

  const context = computed(() => toValue(options.context));
  const agentId = computed(() => toValue(options.agentId));
  const projectId = computed(() => toValue(options.projectId));
  const range = computed(() => toValue(options.range));
  const includeArchived = computed(() => toValue(options.includeArchived));

  const {
    usage,
    byTask,
    loading: usageLoading,
    error: usageError,
  } = useUsage(agentId, projectId, range);

  const tasks = ref<Task[]>([]);
  const tasksLoading = ref(true);
  const tasksError = ref("");
  let taskGeneration = 0;

  async function refreshTasks() {
    const gen = ++taskGeneration;
    const filter = {
      agentId: agentId.value || undefined,
      projectId: projectId.value || undefined,
    };
    try {
      const [active, archived] = await Promise.all([
        agency.listTasks({ ...filter, archived: false }),
        agency.listTasks({ ...filter, archived: true }),
      ]);
      if (gen !== taskGeneration) return;
      tasks.value = [...active, ...archived];
      tasksError.value = "";
    } catch (err) {
      if (gen !== taskGeneration) return;
      tasksError.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (gen === taskGeneration) tasksLoading.value = false;
    }
  }

  watch([agentId, projectId], () => {
    taskGeneration++;
    tasks.value = [];
    tasksLoading.value = true;
    tasksError.value = "";
    void refreshTasks();
  });

  onMounted(async () => {
    await Promise.all([loadAgents(), loadProjects()]);
    void refreshTasks();
  });

  useRealtime(["tasks"], () => void refreshTasks());

  const bounds = computed(() => usageRangeBounds(range.value));

  const agentById = computed(() => {
    const map = new Map<string, (typeof agents.value)[number]>();
    for (const agent of agents.value) map.set(agent.id, agent);
    return map;
  });

  const projectById = computed(() => {
    const map = new Map<string, (typeof projects.value)[number]>();
    for (const project of projects.value) map.set(project.id, project);
    return map;
  });

  function agentName(id: string): string {
    const agent = agentById.value.get(id);
    return (
      agent?.displayName ||
      agent?.username ||
      (id ? `Agent ${id}` : "Unknown agent")
    );
  }

  function projectName(id: string): string {
    return (
      projectById.value.get(id)?.name ||
      (id ? `Project ${id}` : "Unknown project")
    );
  }

  const completedAll = computed(() =>
    tasks.value.filter((task) => task.state === TaskState.COMPLETED),
  );

  const completedInRange = computed(() => {
    const start = bounds.value.sinceMs;
    const end = bounds.value.untilMs || Date.now();
    return completedAll.value.filter((task) => {
      const at = timestampMs(task.settledAt);
      return at >= start && at <= end;
    });
  });

  const completedByDay = computed(() => {
    const map = new Map<number, number>();
    for (const task of completedAll.value) {
      const at = timestampMs(task.settledAt);
      if (!at) continue;
      const key = dayKey(at);
      map.set(key, (map.get(key) ?? 0) + 1);
    }
    return map;
  });

  const earliestCompletedDay = computed(() => {
    const times = completedAll.value
      .map((task) => timestampMs(task.settledAt))
      .filter((value) => value > 0);
    return times.length ? Math.min(...times) : 0;
  });

  const daily = computed<UsageSeriesPoint[]>(() => {
    const days = usage.value?.days ?? [];
    const usageByDay = new Map<number, (typeof days)[number]>();
    for (const day of days) {
      const at = timestampMs(day.day);
      if (at) usageByDay.set(dayKey(at), day);
    }

    const earliestUsage = days.length
      ? Math.min(...days.map((day) => timestampMs(day.day)).filter(Boolean))
      : 0;
    const earliest =
      range.value === "all"
        ? Math.min(
            ...[earliestUsage, earliestCompletedDay.value, Date.now()].filter(
              (value) => value > 0,
            ),
          )
        : bounds.value.sinceMs;
    const start = utcDayStart(earliest || Date.now());
    const end = Date.now();

    return daysBetween(start, end).map((day) => {
      const row = usageByDay.get(day);
      return {
        date: new Date(day),
        cost: row?.cost ?? 0,
        input: Number(row?.inputTokens ?? 0),
        output: Number(row?.outputTokens ?? 0),
        runs: Number(row?.runs ?? 0),
        completed: completedByDay.value.get(day) ?? 0,
      };
    });
  });

  const summary = computed<UsageSummary>(() => {
    const buckets = usage.value?.tasks ?? [];
    const activeMs = buckets.reduce(
      (sum, bucket) => sum + Number(bucket.activeMs),
      0,
    );
    const runs = buckets.reduce(
      (sum, bucket) => sum + Number(bucket.runs ?? 0),
      0,
    );
    return {
      cost: usage.value?.cost ?? 0,
      inputTokens: Number(usage.value?.inputTokens ?? 0),
      outputTokens: Number(usage.value?.outputTokens ?? 0),
      activeMs,
      runs,
      tasks: buckets.length,
      completed: completedInRange.value.length,
      archived: completedInRange.value.filter((task) => task.archived).length,
      allTimeCompleted: completedAll.value.length,
    };
  });

  const taskBySession = computed(() => {
    const map = new Map<string, Task>();
    for (const task of tasks.value) {
      if (task.sessionId) map.set(task.sessionId, task);
    }
    return map;
  });

  function rowFor(bucket: UsageBucket): UsageTaskRow {
    const task = taskBySession.value.get(bucket.taskId);
    const project = task?.projectId || projectId.value;
    return {
      taskId: bucket.taskId,
      task,
      name: task?.name?.trim() || task?.prompt?.trim() || bucket.taskId,
      agentId: bucket.agentId,
      agentName: agentName(bucket.agentId),
      projectId: project,
      projectName: projectName(project),
      tags: task?.tags ?? [],
      state: task?.state ?? TaskState.UNSPECIFIED,
      archived: task?.archived ?? false,
      inputTokens: bucket.inputTokens,
      outputTokens: bucket.outputTokens,
      cost: bucket.cost,
      activeMs: bucket.activeMs,
      runs: bucket.runs,
      live: bucket.live,
    };
  }

  const rows = computed(() =>
    Array.from(byTask.value.values())
      .map(rowFor)
      .sort((a, b) => b.cost - a.cost),
  );

  const tagSeries = computed<UsageTagSeries[]>(() => {
    const eligible = completedAll.value.filter(
      (task) => includeArchived.value || !task.archived,
    );
    if (!eligible.length) return [];

    const customTags = new Map<string, Tag>();
    const allTimeCounts = new Map<string, number>();
    const rangeCounts = new Map<string, number>();
    const rangeStart = bounds.value.sinceMs;

    function keysFor(task: Task): (Tag | "untagged")[] {
      return task.tags.length ? task.tags : ["untagged"];
    }

    for (const task of eligible) {
      const at = timestampMs(task.settledAt);
      for (const key of keysFor(task)) {
        const id = key === "untagged" ? "untagged" : key.id;
        if (key !== "untagged") customTags.set(key.id, key);
        allTimeCounts.set(id, (allTimeCounts.get(id) ?? 0) + 1);
        if (at >= rangeStart) {
          rangeCounts.set(id, (rangeCounts.get(id) ?? 0) + 1);
        }
      }
    }

    const ordered = (counts: Map<string, number>) =>
      Array.from(counts.entries())
        .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
        .map(([id]) => id);
    const ids = [...ordered(rangeCounts), ...ordered(allTimeCounts)]
      .filter((id, index, list) => list.indexOf(id) === index)
      .slice(0, 6);

    const earliest =
      range.value === "all"
        ? earliestCompletedDay.value || Date.now()
        : bounds.value.sinceMs;
    const start = utcDayStart(earliest);
    const days = daysBetween(start, Date.now());

    return ids.map((id) => {
      const tag =
        id === "untagged"
          ? { id, name: "Untagged", color: "#a1a1aa" }
          : customTags.get(id);
      const color = tag?.color || usageAccent(id);
      let count = 0;
      const completedByTaskDay = new Map<number, number>();
      for (const task of eligible) {
        if (
          !keysFor(task).some((key) =>
            key === "untagged" ? id === "untagged" : key.id === id,
          )
        ) {
          continue;
        }
        const at = timestampMs(task.settledAt);
        if (!at) continue;
        if (at < start) {
          count++;
          continue;
        }
        const key = dayKey(at);
        completedByTaskDay.set(key, (completedByTaskDay.get(key) ?? 0) + 1);
      }
      return {
        id,
        name: tag?.name || id,
        color,
        points: days.map((day) => {
          count += completedByTaskDay.get(day) ?? 0;
          return { date: new Date(day), value: count };
        }),
      };
    });
  });

  const breakdown = computed<UsageBreakdownRow[]>(() => {
    const groups = new Map<
      string,
      {
        id: string;
        name: string;
        accent: string;
        cost: number;
        runs: number;
        tasks: number;
        tokens: number;
      }
    >();

    const byAgent = context.value === "project";
    for (const row of rows.value) {
      const id = byAgent ? row.agentId : row.projectId;
      const name = byAgent ? row.agentName : row.projectName;
      const group = groups.get(id) ?? {
        id,
        name,
        accent: usageAccent(id || name),
        cost: 0,
        runs: 0,
        tasks: 0,
        tokens: 0,
      };
      group.cost += row.cost;
      group.runs += row.runs;
      group.tasks += 1;
      group.tokens += row.inputTokens + row.outputTokens;
      groups.set(id, group);
    }
    return Array.from(groups.values())
      .filter((row) => row.cost > 0 || row.runs > 0)
      .sort((a, b) => b.cost - a.cost);
  });

  const availableAgents = computed(() =>
    agents.value.map((agent) => ({
      id: agent.id,
      name: agent.displayName || agent.username,
    })),
  );

  const availableProjects = computed(() =>
    projects.value.map((project) => ({ id: project.id, name: project.name })),
  );

  return {
    usage,
    tasks,
    loading: computed(() => usageLoading.value || tasksLoading.value),
    error: computed(() => usageError.value || tasksError.value),
    summary,
    daily,
    rows,
    tagSeries,
    breakdown,
    availableAgents,
    availableProjects,
    agentName,
    projectName,
    agentIdForActor,
  };
}
