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
import type { ListTasksFilter, Task } from "@cherrypi/sdk";
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
 * Polls the agency-level task list on an interval while mounted. `filter`
 * narrows by project and/or agent; the caller (a task list view) may also
 * narrow by state client-side.
 *
 * There is no server-push task list, so polling keeps the dashboard near-live
 * without a long-lived stream per task. Changing the filter refetches
 * immediately and drops the previous list, so a list never shows the wrong
 * scope while waiting for the next poll.
 */
export function useTasks(
  filter: MaybeRefOrGetter<ListTasksFilter> = {},
  intervalMs = 3000,
) {
  const tasks = ref<Task[]>([]);
  const loading = ref(true);
  const error = ref("");
  let timer: number | undefined;
  // Invalidates in-flight responses when the filter changes, so a slow reply
  // from the previous scope cannot overwrite the new scope's list.
  let generation = 0;

  const filterValue = computed(() => toValue(filter));

  async function refresh() {
    const gen = ++generation;
    try {
      const next = await agency.listTasks(filterValue.value);
      if (gen !== generation) return;
      tasks.value = next;
      error.value = "";
    } catch (err) {
      if (gen !== generation) return;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (gen === generation) loading.value = false;
    }
  }

  watch(filterValue, () => {
    generation++;
    tasks.value = [];
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

  return { tasks, loading, error, refresh };
}
