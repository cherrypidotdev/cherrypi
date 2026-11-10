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
import type { Task } from "@cherrypi/sdk";
import { onMounted, ref } from "vue";

const statuses = ref<Map<string, ProjectStatus>>(new Map());

const tasks = ref<Task[]>([]);

function patch(list: Task[], task: Task): Task[] {
  const index = list.findIndex((item) => item.id === task.id);
  if (task.archivedAt) {
    return index < 0 ? list : list.filter((item) => item.id !== task.id);
  }
  if (index < 0) return [...list, task];
  const next = list.slice();
  next[index] = task;
  return next;
}

export function useProjectStatuses() {
  async function refresh() {
    try {
      tasks.value = await agency.listTasks();
      statuses.value = projectStatuses(tasks.value);
    } catch {}
  }

  onMounted(() => {
    void refresh();
  });

  useRealtime(
    ["tasks"],
    (event) => {
      if (!event.task) {
        void refresh();
        return;
      }
      tasks.value = patch(tasks.value, event.task);
      statuses.value = projectStatuses(tasks.value);
    },
    { rows: true },
  );

  return { statuses, refresh };
}
