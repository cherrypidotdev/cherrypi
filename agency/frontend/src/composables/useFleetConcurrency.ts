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

import { useRealtime } from "@/composables/useRealtime";
import { CONCURRENCY_HELD_STATES } from "@/lib/concurrency";
import { agency } from "@/lib/sdk";
import {
  TaskState,
  type Agent,
  type AgentTaskLimits,
  type Task,
} from "@cherrypi/sdk";
import { computed, ref, toValue, watch, type MaybeRefOrGetter } from "vue";

export interface FleetConcurrencyStat {
  used: number;
  cap: number;
  waiting: number;
  full: boolean;
}

export function useFleetConcurrency(agents: MaybeRefOrGetter<Agent[]>) {
  const stats = ref<Record<string, FleetConcurrencyStat>>({});
  const loading = ref(false);
  const error = ref("");

  const tasks = ref<Task[]>([]);
  const limits = ref<Map<string, AgentTaskLimits>>(new Map());
  let generation = 0;

  function derive() {
    const list = toValue(agents).filter((agent) => agent.id);
    const next: Record<string, FleetConcurrencyStat> = {};
    for (const agent of list) {
      const own = tasks.value.filter(
        (task) => task.assigneeActorId === agent.actorId,
      );
      const used = own.filter((task) =>
        CONCURRENCY_HELD_STATES.has(task.state),
      ).length;
      const waiting = own.filter(
        (task) => task.state === TaskState.QUEUED,
      ).length;
      const cap = limits.value.get(agent.id)?.maxConcurrent ?? 1;
      next[agent.id] = { used, cap, waiting, full: used >= cap };
    }
    stats.value = next;
  }

  async function loadLimits(current: number) {
    const list = toValue(agents).filter((agent) => agent.id);
    const values = await Promise.all(
      list.map(async (agent) => {
        const value = await agency.getAgentTaskLimits(agent.id);
        return [agent.id, value] as const;
      }),
    );
    if (current !== generation) return;
    limits.value = new Map(values);
  }

  async function load() {
    const list = toValue(agents).filter((agent) => agent.id);
    if (!list.length) {
      stats.value = {};
      tasks.value = [];
      limits.value = new Map();
      loading.value = false;
      return;
    }
    const current = ++generation;
    loading.value = true;
    try {
      const [next] = await Promise.all([
        agency.listTasks(),
        loadLimits(current),
      ]);
      if (current !== generation) return;
      tasks.value = next;
      derive();
      error.value = "";
    } catch (err) {
      if (current !== generation) return;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (current === generation) loading.value = false;
    }
  }

  function applyTask(task: Task) {
    const index = tasks.value.findIndex((item) => item.id === task.id);
    if (task.archivedAt) {
      if (index >= 0) tasks.value = tasks.value.filter((t) => t.id !== task.id);
    } else if (index >= 0) {
      const next = tasks.value.slice();
      next[index] = task;
      tasks.value = next;
    } else {
      tasks.value = [...tasks.value, task];
    }
    derive();
  }

  const agentKey = computed(() =>
    toValue(agents)
      .map((agent) => agent.id)
      .join(","),
  );

  watch(agentKey, load, { immediate: true });
  useRealtime(
    ["tasks", "agents"],
    (event) => {
      if (event.kind === "agents" || !event.task) {
        void load();
        return;
      }
      applyTask(event.task);
    },
    { rows: true },
  );

  return { stats, loading, error, refresh: load };
}
