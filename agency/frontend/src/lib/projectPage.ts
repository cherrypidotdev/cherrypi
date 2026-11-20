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

import type { ProjectAccess, ProjectHierarchy } from "@/lib/projectAccess";
import type { Project } from "@cherrypi/sdk";

export interface ProjectPageState {
  project: Project | undefined;
  access: ProjectAccess | undefined;

  canPrompt: boolean;

  canManageProject: boolean;
  error: string;
  setProject(updated: Project): void;
  reload(): Promise<void>;
  reloadAccess(): Promise<void>;
}

export interface ProjectSettingsState {
  hierarchy: ProjectHierarchy | undefined;
  reloadHierarchy(): Promise<void>;
}

export function isProjectRouteName(name: unknown): boolean {
  const value = String(name ?? "");
  return value === "project" || value.startsWith("project-");
}

export function isSettingsRouteName(name: unknown): boolean {
  const value = String(name ?? "");
  return value === "settings" || value.startsWith("settings-");
}
