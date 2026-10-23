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

import { agent, currentAgentId } from "@/lib/sdk";
import { type Terminal } from "@cherrypi/sdk";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

/** Counts of the authenticated user's live terminals. */
export interface TerminalStats {
  root: number;
  total: number;
}

/**
 * Polls the live terminal list and stats while mounted. The dashboard has no
 * server-push terminal counts, so a short interval keeps the numbers current.
 *
 * Switching agents refetches immediately; the previous agent's terminals are
 * dropped so counts never briefly belong to the wrong agent.
 */
export function useTerminals(intervalMs = 3000) {
  const terminals = ref<Terminal[]>([]);
  const stats = ref<TerminalStats>({ root: 0, total: 0 });
  const loading = ref(true);
  const error = ref("");
  let timer: number | undefined;
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

  // `currentAgentId` is a ref in production and a plain object under test;
  // wrapping it keeps the watch source valid either way.
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
    timer = window.setInterval(() => void refresh(), intervalMs);
  });

  onBeforeUnmount(() => {
    if (timer !== undefined) {
      window.clearInterval(timer);
    }
  });

  return { terminals, stats, loading, error, refresh };
}
