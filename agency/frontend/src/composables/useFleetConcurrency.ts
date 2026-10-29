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
import { TaskState, type Agent } from "@cherrypi/sdk";
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
  let generation = 0;

  async function load() {
    const list = toValue(agents).filter((agent) => agent.id);
    if (!list.length) {
      stats.value = {};
      loading.value = false;
      return;
    }
    const current = ++generation;
    loading.value = true;
    try {
      const [tasks, limits] = await Promise.all([
        agency.listTasks(),
        Promise.all(
          list.map(async (agent) => {
            const value = await agency.getAgentTaskLimits(agent.id);
            return [agent.id, value] as const;
          }),
        ),
      ]);
      if (current !== generation) return;
      const byAgent = new Map(limits);
      const next: Record<string, FleetConcurrencyStat> = {};
      for (const agent of list) {
        const own = tasks.filter(
          (task) => task.assigneeActorId === agent.actorId,
        );
        const used = own.filter((task) =>
          CONCURRENCY_HELD_STATES.has(task.state),
        ).length;
        const waiting = own.filter(
          (task) => task.state === TaskState.QUEUED,
        ).length;
        const cap = byAgent.get(agent.id)?.maxConcurrent ?? 1;
        next[agent.id] = { used, cap, waiting, full: used >= cap };
      }
      stats.value = next;
      error.value = "";
    } catch (err) {
      if (current !== generation) return;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (current === generation) loading.value = false;
    }
  }

  const agentKey = computed(() =>
    toValue(agents)
      .map((agent) => agent.id)
      .join(","),
  );

  watch(agentKey, load, { immediate: true });
  useRealtime(["tasks", "agents"], () => void load());

  return { stats, loading, error, refresh: load };
}
