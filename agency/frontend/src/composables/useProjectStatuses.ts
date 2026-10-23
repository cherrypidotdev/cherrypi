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

import { projectStatuses, type ProjectStatus } from "@/lib/project";
import { agency } from "@/lib/sdk";
import { onBeforeUnmount, onMounted, ref } from "vue";

// Module-scoped so the shell's project list and any other consumer share one
// near-live map; the sidebar is mounted once for the app's lifetime.
const statuses = ref<Map<string, ProjectStatus>>(new Map());

/**
 * Polls the agency-wide task list and derives each project's aggregate
 * status for the sidebar. The agency has no server-push task list, so polling
 * keeps the dots near-live without leaning on the task views' per-scope
 * lists.
 */
export function useProjectStatuses(intervalMs = 3000) {
  let timer: number | undefined;

  async function refresh() {
    try {
      statuses.value = projectStatuses(await agency.listTasks());
    } catch {
      // Keep the last known statuses; a transient error should not blank the
      // sidebar.
    }
  }

  onMounted(() => {
    void refresh();
    timer = window.setInterval(() => void refresh(), intervalMs);
  });

  onBeforeUnmount(() => {
    if (timer !== undefined) window.clearInterval(timer);
  });

  return { statuses, refresh };
}
