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

import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import type { RouteLocationRaw } from "vue-router";

const PROMPT_SCOPES = [
  "org",
  "agent",
  "project",
  "agent_project",
  "team",
  "team_project",
] as const;

export type PromptScope = (typeof PROMPT_SCOPES)[number];

export interface PromptTarget {
  agentId?: string | number | bigint;
  promptScope?: string;
  promptProjectId?: string;
  promptTeamId?: string | number | bigint;
}

export function isPromptTarget(target: PromptTarget): boolean {
  return PROMPT_SCOPES.includes(target.promptScope as PromptScope);
}

export function promptRoute(target: PromptTarget): RouteLocationRaw | null {
  switch (target.promptScope) {
    case "org":
      return { name: "settings", query: { tab: "general" } };
    case "agent":
    case "agent_project": {
      const username = agentUsername(String(target.agentId ?? ""));
      if (!username) return null;
      return {
        name: "agent",
        params: { username },

        query: {
          tab: "settings",
          section:
            target.promptScope === "agent_project" ? "projects" : "tasks",
        },
      };
    }
    case "project": {
      const slug = projectSlug(target.promptProjectId ?? "");
      if (!slug) return null;
      return {
        name: "project",
        params: { slug },

        query: { tab: "settings", section: "tasks" },
      };
    }
    case "team":
    case "team_project": {
      const id = String(target.promptTeamId ?? "");
      if (!id || id === "0") return null;

      return { name: "team", params: { id }, query: { tab: "projects" } };
    }
    default:
      return null;
  }
}

export async function loadPromptRouteTargets(): Promise<void> {
  await Promise.all([useAgents().loadAgents(), useProjects().loadProjects()]);
}

function agentUsername(agentID: string): string {
  if (!agentID) return "";
  return (
    useAgents().agents.value.find((agent) => agent.id === agentID)?.username ??
    ""
  );
}

function projectSlug(projectID: string): string {
  if (!projectID) return "";
  return (
    useProjects().projects.value.find((project) => project.id === projectID)
      ?.slug ?? ""
  );
}
