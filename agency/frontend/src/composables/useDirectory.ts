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

import { request } from "@/lib/account";
import { teamApi, type Team, type TeamMember } from "@/lib/team";
import { computed, ref } from "vue";

const teams = ref<Team[]>([]);
const membersByTeam = ref<Record<number, TeamMember[]>>({});
const ownedAgentActorIds = ref<string[]>([]);
const loading = ref(false);
const error = ref("");

async function fetchOwnedAgentActorIds(): Promise<string[]> {
  try {
    const payload = await request<{ actor_ids: string[] }>(
      "GET",
      "/api/rest/agents/owned",
    );
    return payload.actor_ids ?? [];
  } catch {
    return [];
  }
}

async function refresh() {
  loading.value = true;
  try {
    const [teamPayload, owned] = await Promise.all([
      teamApi.list(),
      fetchOwnedAgentActorIds(),
    ]);

    const mine = teamPayload.teams.filter((team) => !!team.role);
    const memberLists = await Promise.all(
      mine.map(async (team) => {
        try {
          const payload = await teamApi.members(team.id);
          return [team.id, payload.members] as const;
        } catch {
          return [team.id, [] as TeamMember[]] as const;
        }
      }),
    );
    teams.value = mine;
    membersByTeam.value = Object.fromEntries(memberLists);
    ownedAgentActorIds.value = owned;
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

let loaded = false;

async function loadDirectory(force = false) {
  if (loaded && !force) return;
  loaded = true;
  await refresh();
}

const teammateUsers = computed(() => {
  const byId = new Map<string, TeamMember>();
  for (const members of Object.values(membersByTeam.value)) {
    for (const member of members) {
      if (member.kind !== "user") continue;
      byId.set(member.actor_id, member);
    }
  }
  return [...byId.values()].sort((a, b) =>
    (a.name || a.username).localeCompare(b.name || b.username),
  );
});

const teammateAgentActorIds = computed(() => {
  const ids = new Set<string>();
  for (const members of Object.values(membersByTeam.value)) {
    for (const member of members) {
      if (member.kind === "agent") ids.add(member.actor_id);
    }
  }
  return ids;
});

export function useDirectory() {
  return {
    teams,
    membersByTeam,
    ownedAgentActorIds,
    loading,
    error,
    loadDirectory,
    refresh,
    teammateUsers,
    teammateAgentActorIds,
  };
}
