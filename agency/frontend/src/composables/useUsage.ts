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
import { agency } from "@/lib/sdk";
import { usageRangeBounds, type UsageRange } from "@/lib/usage";
import type { AgencyUsageResponse } from "@cherrypi/sdk";
import {
  computed,
  onMounted,
  ref,
  toValue,
  watch,
  type MaybeRefOrGetter,
} from "vue";

export interface UsageBucket {
  agentId: string;
  taskId: string;
  inputTokens: number;
  outputTokens: number;
  cost: number;

  activeMs: number;

  runs: number;
  live: boolean;
}

export function useUsage(
  agentId: MaybeRefOrGetter<string> = "",
  projectId: MaybeRefOrGetter<string> = "",
  range: MaybeRefOrGetter<UsageRange> = "all",
) {
  const usage = ref<AgencyUsageResponse | undefined>();
  const error = ref("");
  const loading = ref(true);
  let generation = 0;

  const agentIdValue = computed(() => toValue(agentId));
  const projectIdValue = computed(() => toValue(projectId));
  const rangeValue = computed(() => toValue(range));

  async function refresh() {
    const gen = ++generation;
    const bounds = usageRangeBounds(rangeValue.value);
    try {
      const next = await agency.getUsage(
        agentIdValue.value,
        projectIdValue.value,
        { sinceMs: bounds.sinceMs, untilMs: bounds.untilMs },
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
        agentId: bucket.agentId,
        taskId: bucket.taskId,
        inputTokens: Number(bucket.inputTokens),
        outputTokens: Number(bucket.outputTokens),
        cost: bucket.cost,
        activeMs: Number(bucket.activeMs),
        runs: Number(bucket.runs ?? 0),
        live: false,
      });
    }
    return map;
  });

  watch([agentIdValue, projectIdValue, rangeValue], () => {
    generation++;
    usage.value = undefined;
    loading.value = true;
    error.value = "";
    void refresh();
  });

  onMounted(() => {
    void refresh();
  });

  useRealtime(["usage"], () => void refresh());

  return { usage, byTask, loading, error, refresh };
}
