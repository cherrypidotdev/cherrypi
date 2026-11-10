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

import type { PluginLog } from "@cherrypi/sdk";
import { ref, type Ref } from "vue";

export const PLUGIN_LOG_PAGE_SIZE = 100;

/**
 * Pages a plugin's Activity or Runtime log by line id. The first page is the
 * newest; `loadMore` asks for the page older than the last line it holds, so
 * lines written while the panel is open do not shift the window.
 */
export function usePluginLogPager(
  fetchPage: (beforeId: number, limit: number) => Promise<PluginLog[]>,
  limit = PLUGIN_LOG_PAGE_SIZE,
) {
  const logs = ref<PluginLog[]>([]) as Ref<PluginLog[]>;
  const loading = ref(false);
  const loadingMore = ref(false);
  const error = ref("");
  const hasMore = ref(false);

  const auto = ref(false);
  const loaded = ref(false);

  let generation = 0;

  function reset() {
    generation += 1;
    logs.value = [];
    hasMore.value = false;
    auto.value = false;
    loaded.value = false;
    loading.value = false;
    loadingMore.value = false;
    error.value = "";
  }

  function message(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }

  async function load() {
    const request = generation;
    loading.value = true;
    error.value = "";
    try {
      const page = await fetchPage(0, limit);
      if (request !== generation) return;
      logs.value = page;
      hasMore.value = page.length === limit;
      loaded.value = true;
    } catch (err) {
      if (request !== generation) return;
      error.value = message(err);
    } finally {
      if (request === generation) loading.value = false;
    }
  }

  async function loadMore(manual = false) {
    if (loadingMore.value || !hasMore.value) return;
    if (manual) auto.value = true;
    const last = logs.value[logs.value.length - 1];
    const beforeId = last ? Number(last.id) : 0;
    if (!beforeId) return;
    const request = generation;
    loadingMore.value = true;
    try {
      const page = await fetchPage(beforeId, limit);

      if (request !== generation) return;
      const seen = new Set(logs.value.map((line) => String(line.id)));
      logs.value = [
        ...logs.value,
        ...page.filter((line) => !seen.has(String(line.id))),
      ];
      hasMore.value = page.length === limit;
      loaded.value = true;
    } catch (err) {
      if (request !== generation) return;
      error.value = message(err);
    } finally {
      if (request === generation) loadingMore.value = false;
    }
  }

  return {
    logs,
    loading,
    loadingMore,
    error,
    hasMore,
    auto,
    loaded,
    load,
    loadMore,
    reset,
  };
}
