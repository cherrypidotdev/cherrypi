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

import { agency } from "@/lib/sdk";
import type { AgencyUsageResponse } from "@cherrypi/sdk";
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
  toValue,
  watch,
  type MaybeRefOrGetter,
} from "vue";

/**
 * One task's usage totals, shaped for the task list/table. A running task's
 * numbers come from its agent's live stats until it settles (the `live` flag is
 * not carried by the agency aggregate yet).
 */
export interface UsageBucket {
  taskId: string;
  inputTokens: number;
  outputTokens: number;
  cost: number;
  // Active work time (settled runs plus the in-flight run), in milliseconds.
  activeMs: number;
  live: boolean;
}

/**
 * Polls the agency's usage aggregate on an interval while the component is
 * mounted, exposing per-task buckets so lists can join usage onto tasks.
 * Pass an empty `agentId` for the fleet-wide aggregate, or one agent's id for
 * its scoped view; `projectId` narrows either to a single project. Both may be
 * refs/getters (e.g. `currentAgentId`), in which case switching filters
 * refetches immediately and drops the previous result.
 */
export function useUsage(
  agentId: MaybeRefOrGetter<string> = "",
  projectId: MaybeRefOrGetter<string> = "",
  intervalMs = 3000,
) {
  const usage = ref<AgencyUsageResponse | undefined>();
  const error = ref("");
  const loading = ref(true);
  let timer: number | undefined;
  let generation = 0;

  const agentIdValue = computed(() => toValue(agentId));
  const projectIdValue = computed(() => toValue(projectId));

  async function refresh() {
    const gen = ++generation;
    try {
      const next = await agency.getUsage(
        agentIdValue.value,
        projectIdValue.value,
      );
      if (gen !== generation) return;
      usage.value = next;
      error.value = "";
    } catch (err) {
      if (gen !== generation) return;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (gen === generation) loading.value = false;
    }
  }

  const byTask = computed(() => {
    const map = new Map<string, UsageBucket>();
    for (const bucket of usage.value?.tasks ?? []) {
      map.set(bucket.taskId, {
        taskId: bucket.taskId,
        inputTokens: Number(bucket.inputTokens),
        outputTokens: Number(bucket.outputTokens),
        cost: bucket.cost,
        activeMs: Number(bucket.activeMs),
        live: false,
      });
    }
    return map;
  });

  watch([agentIdValue, projectIdValue], () => {
    generation++;
    usage.value = undefined;
    loading.value = true;
    error.value = "";
    void refresh();
  });

  onMounted(() => {
    void refresh();
    timer = window.setInterval(() => void refresh(), intervalMs);
  });

  onBeforeUnmount(() => {
    if (timer !== undefined) window.clearInterval(timer);
  });

  return { usage, byTask, loading, error, refresh };
}
