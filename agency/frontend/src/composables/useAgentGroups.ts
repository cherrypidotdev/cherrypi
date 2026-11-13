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
  AgentGroup,
  AgentGroupLog,
  AgentGroupMember,
  AgentGroupMemberRequest,
  AgentGroupOwner,
  AgentGroupProject,
  AgentGroupTeam,
} from "@cherrypi/sdk";
import { computed, ref } from "vue";

const groups = ref<AgentGroup[]>([]);
const loading = ref(false);
const error = ref("");

const activeGroups = computed(() => groups.value.filter((g) => !g.archivedAt));
const archivedGroups = computed(() =>
  groups.value.filter((g) => !!g.archivedAt),
);

async function refresh() {
  loading.value = true;
  try {
    groups.value = await agency.listAgentGroups();
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

let loaded = false;
let pending: Promise<void> | null = null;

async function loadAgentGroups(force = false) {
  if (pending) return pending;
  if (loaded && !force) return;
  loaded = true;
  pending = refresh().finally(() => {
    pending = null;
  });
  return pending;
}

function groupBySlug(slug: string): AgentGroup | undefined {
  return groups.value.find((group) => group.slug === slug);
}

function groupById(id: bigint | number | string): AgentGroup | undefined {
  const key = String(id);
  return groups.value.find((group) => String(group.id) === key);
}

async function createGroup(input: {
  name: string;
  slug?: string;
  description?: string;
}): Promise<AgentGroup> {
  const group = await agency.createAgentGroup(input);
  await refresh();
  return group;
}

async function updateGroup(
  id: bigint | number,
  input: { name: string; slug?: string; description?: string },
): Promise<AgentGroup> {
  const group = await agency.updateAgentGroup(id, input);
  await refresh();
  return group;
}

async function setArchived(
  id: bigint | number,
  archived: boolean,
): Promise<AgentGroup> {
  const group = await agency.setAgentGroupArchived(id, archived);
  await refresh();
  return group;
}

async function loadMembers(
  groupId: bigint | number,
): Promise<AgentGroupMember[]> {
  return agency.listAgentGroupMembers(groupId);
}

async function loadOwners(
  groupId: bigint | number,
): Promise<AgentGroupOwner[]> {
  return agency.listAgentGroupOwners(groupId);
}

async function loadProjects(
  groupId: bigint | number,
): Promise<AgentGroupProject[]> {
  return agency.listAgentGroupProjects(groupId);
}

async function loadTeams(groupId: bigint | number): Promise<AgentGroupTeam[]> {
  return agency.listAgentGroupTeams(groupId);
}

async function loadRequests(
  groupId: bigint | number,
  status = "pending",
): Promise<AgentGroupMemberRequest[]> {
  return agency.listAgentGroupMemberRequests(groupId, status);
}

async function loadLogs(
  groupId: bigint | number,
  beforeId: bigint | number = 0,
  limit = 25,
): Promise<AgentGroupLog[]> {
  return agency.listAgentGroupLogs(groupId, { beforeId, limit });
}

async function loadAllMembers(): Promise<Record<string, AgentGroupMember[]>> {
  const entries = await Promise.all(
    groups.value.map(async (group) => {
      try {
        const members = await agency.listAgentGroupMembers(group.id);
        return [String(group.id), members] as const;
      } catch {
        return [String(group.id), []] as const;
      }
    }),
  );
  return Object.fromEntries(entries);
}

async function groupsForAgent(agentId: string): Promise<AgentGroup[]> {
  return agency.listAgentGroupsForAgent(agentId);
}

export function useAgentGroups() {
  return {
    groups,
    activeGroups,
    archivedGroups,
    loading,
    error,
    loadAgentGroups,
    refresh,
    groupBySlug,
    groupById,
    createGroup,
    updateGroup,
    setArchived,
    loadMembers,
    loadOwners,
    loadProjects,
    loadTeams,
    loadRequests,
    loadLogs,
    loadAllMembers,
    groupsForAgent,
  };
}
