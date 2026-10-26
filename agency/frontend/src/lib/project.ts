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

export function slugify(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export function normalizeTaskPrefix(value: string): string {
  return value
    .trim()
    .toUpperCase()
    .replace(/[^A-Za-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export type ProjectStatus = "input" | "running" | "settled";

export function projectStatus(
  tasks: Pick<Task, "state">[],
): ProjectStatus | undefined {
  if (tasks.some((task) => task.state === TaskState.INPUT)) return "input";
  if (tasks.some((task) => task.state === TaskState.RUNNING)) return "running";
  if (tasks.some((task) => task.state === TaskState.SETTLED)) return "settled";
  return undefined;
}

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

export const PROJECT_STATUS_LABEL: Record<ProjectStatus, string> = {
  input: "Input required",
  running: "Running",
  settled: "Settled",
};

export const PROJECT_STATUS_STATE: Record<ProjectStatus, TaskState> = {
  input: TaskState.INPUT,
  running: TaskState.RUNNING,
  settled: TaskState.SETTLED,
};

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

export function sortProjectsByName<T extends { name: string }>(
  projects: T[],
): T[] {
  return [...projects].sort((a, b) => a.name.localeCompare(b.name));
}
