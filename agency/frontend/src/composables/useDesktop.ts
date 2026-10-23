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
import type { Desktop } from "@cherrypi/sdk";
import { computed, ref, watch } from "vue";

/**
 * Tracks the agent's shared desktop: its geometry, whether it can be served,
 * and the agency relay path. CherryPi reuses the X session already running on
 * the agent — it never starts one — so there is nothing to start or stop here.
 * The RFB connection itself lives in the view, since it needs the container.
 */
export function useDesktop() {
  const desktop = ref<Desktop>();
  const wsPath = ref("");
  const available = ref(false);
  const unavailableReason = ref("");
  const loading = ref(true);
  const error = ref("");

  async function refresh() {
    loading.value = true;
    try {
      const state = await agent.getDesktop();
      desktop.value = state.desktop;
      wsPath.value = state.wsPath;
      available.value = state.available;
      unavailableReason.value = state.unavailableReason;
      error.value = "";
    } catch (err) {
      available.value = false;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  }

  // Switching agents (a param-only route change) must drop the old desktop and
  // refetch; `currentAgentId` is a ref in production and a plain object under
  // test, so wrapping it keeps the watch source valid either way.
  const agentId = computed(() => currentAgentId.value);
  watch(agentId, () => {
    desktop.value = undefined;
    wsPath.value = "";
    available.value = false;
    unavailableReason.value = "";
    loading.value = true;
    void refresh();
  });

  return {
    desktop,
    wsPath,
    available,
    unavailableReason,
    loading,
    error,
    refresh,
  };
}
