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
import { agent, currentAgentId } from "@/lib/sdk";
import { type Terminal } from "@cherrypi/sdk";
import { computed, onMounted, ref, watch } from "vue";

export interface TerminalStats {
  root: number;
  total: number;
}

export function useTerminals() {
  const terminals = ref<Terminal[]>([]);
  const stats = ref<TerminalStats>({ root: 0, total: 0 });
  const loading = ref(true);
  const error = ref("");
  let generation = 0;

  async function refresh() {
    const gen = ++generation;
    try {
      const [list, counts] = await Promise.all([
        agent.listTerminals(),
        agent.getTerminalStats(),
      ]);
      if (gen !== generation) return;
      terminals.value = list;
      stats.value = {
        root: counts.root,
        total: counts.total,
      };
      error.value = "";
    } catch (err) {
      if (gen !== generation) return;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (gen === generation) loading.value = false;
    }
  }

  const agentId = computed(() => currentAgentId.value);
  watch(agentId, () => {
    generation++;
    terminals.value = [];
    stats.value = { root: 0, total: 0 };
    loading.value = true;
    error.value = "";
    void refresh();
  });

  onMounted(() => {
    void refresh();
  });

  useRealtime(["terminals", "agents"], () => void refresh());

  return { terminals, stats, loading, error, refresh };
}
