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
import {
  healthRangeBounds,
  mergeHealthSample,
  type HealthRange,
} from "@/lib/health";
import { agency } from "@/lib/sdk";
import type { AgentHealthSample } from "@cherrypi/sdk";
import { computed, ref, watch, type Ref } from "vue";

export function useAgentHealth(
  agentId: Ref<string>,
  range: Ref<HealthRange>,
  options: { history?: boolean } = {},
) {
  const withHistory = options.history !== false;
  const current = ref<AgentHealthSample>();
  const history = ref<AgentHealthSample[]>([]);
  const loading = ref(false);
  const error = ref("");
  let appending = false;

  async function refresh() {
    const id = agentId.value;
    if (!id) {
      current.value = undefined;
      history.value = [];
      return;
    }

    loading.value = true;
    try {
      const bounds = healthRangeBounds(range.value);
      const [latest, samples] = await Promise.all([
        agency.getAgentHealth(id),
        withHistory
          ? agency.getAgentHealthHistory(
              id,
              bounds.sinceMs,
              bounds.untilMs,
              720,
            )
          : Promise.resolve([]),
      ]);
      current.value = latest;
      if (withHistory) history.value = samples;
      error.value = "";
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  }

  async function appendLatest() {
    const id = agentId.value;
    if (!id || !withHistory || appending) return;
    appending = true;
    try {
      const latest = await agency.getAgentHealth(id);
      if (!latest) return;
      current.value = latest;
      const last = history.value[history.value.length - 1];
      const latestAt = Number(latest.sampledAt);
      if (last && latestAt <= Number(last.sampledAt)) return;

      history.value = mergeHealthSample(
        history.value,
        latest,
        range.value,
        latestAt,
      );
      error.value = "";
    } catch {
    } finally {
      appending = false;
    }
  }

  watch([agentId, range], () => void refresh(), { immediate: true });
  useRealtime(["health", "agents"], (kind) => {
    if (kind === "health") void appendLatest();
    else void refresh();
  });

  return {
    current,
    history,
    loading,
    error,
    refresh,
    hasData: computed(() => !!current.value),
  };
}
