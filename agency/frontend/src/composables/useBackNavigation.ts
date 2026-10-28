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

import type { Team } from "@/lib/team";
import type { Project } from "@cherrypi/sdk";
import { computed, type ComputedRef } from "vue";
import { useRoute, useRouter, type RouteLocationRaw } from "vue-router";
import { useDirectory } from "./useDirectory";
import { useProjects } from "./useProjects";

export interface BackFallback {
  to: RouteLocationRaw;
  label: string;
}

export interface BackNavigation {
  label: ComputedRef<string>;
  go: () => void;
}

const ROUTE_LABELS: Record<string, string> = {
  account: "Account",
  "agency-tasks": "Tasks",
  agents: "Agents",
  inbox: "Inbox",
  project: "Project",
  projects: "Projects",
  settings: "Settings",
  task: "Task",
  team: "Team",
  teams: "Teams",
  user: "User",
  users: "Users",
};

function historyBack(router: ReturnType<typeof useRouter>): string | null {
  const back = router.options.history.state?.back;
  return typeof back === "string" && back.length > 0 ? back : null;
}

function historyLabel(
  router: ReturnType<typeof useRouter>,
  path: string,
  projects: Project[],
  teams: Team[],
): string {
  const resolved = router.resolve(path);
  const name = typeof resolved.name === "string" ? resolved.name : "";
  const username = resolved.params.username;
  if (
    (name === "agent" || name === "user") &&
    typeof username === "string" &&
    username
  ) {
    return `@${username}`;
  }
  if (name === "project") {
    const slug = resolved.params.slug;
    const project = projects.find((item) => item.slug === slug);
    if (project) return project.name;
  }
  if (name === "team") {
    const id = Number(resolved.params.id);
    const team = teams.find((item) => item.id === id);
    if (team) return team.name;
  }
  if (name && ROUTE_LABELS[name]) return ROUTE_LABELS[name];
  return typeof resolved.meta.title === "string" ? resolved.meta.title : "Back";
}

export function useBackNavigation(
  fallback: () => BackFallback,
): BackNavigation {
  const router = useRouter();
  const route = useRoute();
  const { projects } = useProjects();
  const { teams } = useDirectory();

  const historyPath = computed(() => {
    void route.fullPath;
    return historyBack(router);
  });

  const label = computed(() =>
    historyPath.value
      ? historyLabel(router, historyPath.value, projects.value, teams.value)
      : fallback().label,
  );

  function go() {
    if (historyPath.value) {
      router.back();
      return;
    }
    void router.push(fallback().to);
  }

  return { label, go };
}
