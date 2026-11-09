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

import { useProjects } from "@/composables/useProjects";
import { agency } from "@/lib/sdk";
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";

export function usePageProject() {
  const route = useRoute();
  const { projects, loadProjects } = useProjects();

  const taskProjectId = ref("");
  let resolvedTaskId = "";

  const taskId = computed(() =>
    route.name === "task" ? String(route.params.id ?? "") : "",
  );

  watch(
    taskId,
    (id) => {
      if (id === resolvedTaskId) return;
      resolvedTaskId = "";
      taskProjectId.value = "";
    },
    { flush: "sync" },
  );

  const projectId = computed(() => {
    if (route.name === "project") {
      const slug = String(route.params.slug ?? "");
      return projects.value.find((item) => item.slug === slug)?.id ?? "";
    }
    return taskProjectId.value;
  });

  async function resolve(): Promise<void> {
    void loadProjects();
    const id = taskId.value;
    if (!id || id === resolvedTaskId) return;
    try {
      const task = await agency.getTask(id);
      if (taskId.value !== id) return;
      resolvedTaskId = id;
      taskProjectId.value = task.projectId ?? "";
    } catch {}
  }

  return { projectId, resolve };
}
