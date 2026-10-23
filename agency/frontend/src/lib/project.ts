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

import { projectAccessApi } from "@/lib/projectAccess";
import { TaskState, type Task } from "@cherrypi/sdk";

/**
 * A project's aggregate attention status, derived from its tasks. `input`
 * means a task is waiting on the user, `running` means a run is in flight,
 * and `settled` means a finished run is waiting for a follow-up or close-out.
 */
export type ProjectStatus = "input" | "running" | "settled";

/**
 * Derives one project's status from its tasks: `input` while any task waits
 * for the user, otherwise `running` while any task is in flight, otherwise
 * `settled` while any task is settled. Returns `undefined` when none of those
 * states is present (an empty project, or one whose tasks are all draft,
 * queued, completed, cancelled, ...), so callers render no indicator.
 */
export function projectStatus(
  tasks: Pick<Task, "state">[],
): ProjectStatus | undefined {
  if (tasks.some((task) => task.state === TaskState.INPUT)) return "input";
  if (tasks.some((task) => task.state === TaskState.RUNNING)) return "running";
  if (tasks.some((task) => task.state === TaskState.SETTLED)) return "settled";
  return undefined;
}

/**
 * Groups tasks by project and derives each project's status. Projects with no
 * status (the state map returns `undefined`) are absent, so the sidebar
 * leaves them unmarked.
 */
export function projectStatuses(
  tasks: Pick<Task, "projectId" | "state">[],
): Map<string, ProjectStatus> {
  const grouped = new Map<string, Pick<Task, "state">[]>();
  for (const task of tasks) {
    if (!task.projectId) continue;
    const group = grouped.get(task.projectId);
    if (group) group.push(task);
    else grouped.set(task.projectId, [task]);
  }

  const statuses = new Map<string, ProjectStatus>();
  for (const [projectId, group] of grouped) {
    const status = projectStatus(group);
    if (status) statuses.set(projectId, status);
  }
  return statuses;
}

/** Human-readable label for a project status, used as the dot's tooltip. */
export const PROJECT_STATUS_LABEL: Record<ProjectStatus, string> = {
  input: "Input required",
  running: "Running",
  settled: "Settled",
};

/**
 * The task state whose icon and color represent each project status, so the
 * sidebar reuses the task palette instead of inventing a second one.
 */
export const PROJECT_STATUS_STATE: Record<ProjectStatus, TaskState> = {
  input: TaskState.INPUT,
  running: TaskState.RUNNING,
  settled: TaskState.SETTLED,
};

/**
 * Lists the projects a task in `projectId` may reference: itself, its
 * descendants and its siblings. Falls back to the project itself when the
 * hierarchy cannot be read, so the picker still offers same-project tasks.
 */
export async function referenceProjectIds(
  projectId: string,
): Promise<string[]> {
  try {
    const hierarchy = await projectAccessApi.hierarchy(projectId);
    return hierarchy.reference_project_ids.length
      ? hierarchy.reference_project_ids
      : [projectId];
  } catch {
    return [projectId];
  }
}

/**
 * Returns a copy of `projects` ordered by name. The agency lists projects by
 * creation date; select dropdowns want a stable alphabetical order so long
 * pickers stay scannable.
 */
export function sortProjectsByName<T extends { name: string }>(
  projects: T[],
): T[] {
  return [...projects].sort((a, b) => a.name.localeCompare(b.name));
}
