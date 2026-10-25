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
import type { Agent } from "@cherrypi/sdk";
import { ref } from "vue";

const agents = ref<Agent[]>([]);
const loading = ref(false);
const error = ref("");

async function refresh() {
  loading.value = true;
  try {
    agents.value = await agency.listAgents();
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

let loaded = false;

async function loadAgents(force = false) {
  if (loaded && !force) return;
  loaded = true;
  await refresh();
}

function usernameFor(id: string): string {
  const agent = agents.value.find(
    (item) => item.id === id || item.actorId === id,
  );
  return agent?.username ?? id;
}

function actorIdFor(id: string): string {
  return agents.value.find((agent) => agent.id === id)?.actorId ?? id;
}

function agentIdForActor(actorId: string): string {
  return agents.value.find((agent) => agent.actorId === actorId)?.id ?? actorId;
}

export function useAgents() {
  return {
    agents,
    loading,
    error,
    loadAgents,
    refresh,
    usernameFor,
    actorIdFor,
    agentIdForActor,
  };
}
