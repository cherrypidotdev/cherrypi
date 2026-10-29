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
import { useRealtime } from "@/composables/useRealtime";
import { useTags } from "@/composables/useTags";
import {
  buildAgentBoard,
  buildProjectBoard,
  type AgentConcurrencyBoard,
  type ProjectConcurrencyBoard,
} from "@/lib/concurrency";
import { agency } from "@/lib/sdk";
import type { AgentTaskLimits, Project } from "@cherrypi/sdk";
import { ref, toValue, watch, type MaybeRefOrGetter } from "vue";

export function useAgentConcurrency(agentId: MaybeRefOrGetter<string>) {
  const { agents, loadAgents } = useAgents();
  const { all: tags, refresh: refreshTags } = useTags();
  const board = ref<AgentConcurrencyBoard>();
  const loading = ref(true);
  const error = ref("");
  let generation = 0;

  async function load() {
    const id = toValue(agentId);
    if (!id) {
      board.value = undefined;
      loading.value = false;
      return;
    }
    const current = ++generation;
    loading.value = true;
    try {
      await Promise.all([loadAgents(), refreshTags()]);
      const agent = agents.value.find((item) => item.id === id);
      if (!agent) throw new Error("Agent not found");
      const [tasks, limits, projects] = await Promise.all([
        agency.listTasks({ agentId: id }),
        agency.getAgentTaskLimits(id),
        agency.listAgentProjects(id),
      ]);
      if (current !== generation) return;
      board.value = buildAgentBoard({
        agent,
        projects,
        limits,
        tasks,
        allTags: tags.value,
      });
      error.value = "";
    } catch (err) {
      if (current !== generation) return;
      board.value = undefined;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (current === generation) loading.value = false;
    }
  }

  watch(() => toValue(agentId), load, { immediate: true });
  useRealtime(["tasks", "agents"], () => void load());

  return { board, loading, error, refresh: load };
}

export function useProjectConcurrency(
  project: MaybeRefOrGetter<Project | undefined>,
) {
  const { agents, loadAgents } = useAgents();
  const { all: tags, refresh: refreshTags } = useTags(
    () => toValue(project)?.id,
  );
  const board = ref<ProjectConcurrencyBoard>();
  const loading = ref(true);
  const error = ref("");
  let generation = 0;

  async function load() {
    const currentProject = toValue(project);
    if (!currentProject) {
      board.value = undefined;
      loading.value = false;
      return;
    }
    const current = ++generation;
    loading.value = true;
    try {
      await Promise.all([loadAgents(), refreshTags()]);
      const [memberList, tasks] = await Promise.all([
        agency.listProjectAgents(currentProject.id),
        agency.listTasks(),
      ]);
      const memberAgents = memberList
        .map((member) =>
          agents.value.find((agent) => agent.id === member.agentId),
        )
        .filter((agent): agent is NonNullable<typeof agent> => !!agent);
      const limits = await Promise.all(
        memberAgents.map(async (agent) => {
          const value = await agency.getAgentTaskLimits(agent.id);
          return [agent.id, value] as const;
        }),
      );
      if (current !== generation) return;
      board.value = buildProjectBoard({
        project: currentProject,
        members: memberAgents,
        tasks,
        limitsByAgent: new Map<string, AgentTaskLimits>(limits),
        allTags: tags.value,
      });
      error.value = "";
    } catch (err) {
      if (current !== generation) return;
      board.value = undefined;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (current === generation) loading.value = false;
    }
  }

  watch(() => toValue(project)?.id, load, { immediate: true });
  useRealtime(["tasks", "agents"], () => void load());

  return { board, loading, error, refresh: load };
}
