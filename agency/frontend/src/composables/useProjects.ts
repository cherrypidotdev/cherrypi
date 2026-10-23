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
import type {
  CreateProjectInput,
  Project,
  UpdateProjectInput,
} from "@cherrypi/sdk";
import { computed, ref } from "vue";

// Module-scoped so the shell's projects tree and the projects pages share one
// list; every mutation refreshes it.
const projects = ref<Project[]>([]);
const loading = ref(false);
const error = ref("");

const active = computed(() => projects.value.filter((p) => !p.archivedAt));
const archived = computed(() => projects.value.filter((p) => !!p.archivedAt));

async function refresh() {
  loading.value = true;
  try {
    projects.value = await agency.listProjects();
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

let loaded = false;

async function loadProjects(force = false) {
  if (loaded && !force) return;
  loaded = true;
  await refresh();
}

/** Creates a project and refreshes the shared list. */
async function createProject(input: CreateProjectInput): Promise<Project> {
  const project = await agency.createProject(input);
  await refresh();
  return project;
}

async function updateProject(
  id: string,
  input: UpdateProjectInput,
): Promise<Project> {
  const project = await agency.updateProject(id, input);
  await refresh();
  return project;
}

/**
 * Archives the project and its tasks. Returns the agents that were offline,
 * so the caller can warn that some tasks were not archived yet.
 */
async function archiveProject(id: string): Promise<string[]> {
  const { offlineAgents } = await agency.archiveProject(id);
  await refresh();
  return offlineAgents;
}

async function unarchiveProject(id: string): Promise<Project> {
  const project = await agency.unarchiveProject(id);
  await refresh();
  return project;
}

async function deleteProject(id: string): Promise<void> {
  await agency.deleteProject(id);
  await refresh();
}

export function useProjects() {
  return {
    projects,
    active,
    archived,
    loading,
    error,
    loadProjects,
    refresh,
    createProject,
    updateProject,
    archiveProject,
    unarchiveProject,
    deleteProject,
  };
}
