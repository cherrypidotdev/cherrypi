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
import { projectStatuses, type ProjectStatus } from "@/lib/project";
import { agency } from "@/lib/sdk";
import { onMounted, ref } from "vue";

const statuses = ref<Map<string, ProjectStatus>>(new Map());

export function useProjectStatuses() {
  async function refresh() {
    try {
      statuses.value = projectStatuses(await agency.listTasks());
    } catch {}
  }

  onMounted(() => {
    void refresh();
  });

  useRealtime(["tasks"], () => void refresh());

  return { statuses, refresh };
}
