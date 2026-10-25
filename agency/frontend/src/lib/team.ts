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

export type TeamRole = "owner" | "member";

export interface Team {
  id: number;
  name: string;
  description: string;
  member_count: number;
  role: TeamRole | "";
  created_by_actor_id?: string;
  created_at: string;
  updated_at: string;
}

export interface TeamProject {
  id: string;
  name: string;
  slug: string;
  role: string;
}

export interface TeamMember {
  actor_id: string;
  kind: string;
  username: string;
  name: string;
  description: string;
  role: TeamRole;
  joined_at: string;
}

export interface TeamInvitation {
  id: number;
  team_id: number;
  team_name: string;
  team_description: string;
  actor_id: string;
  kind: string;
  username: string;
  name: string;
  role: TeamRole;
  status: "pending" | "accepted" | "declined" | "revoked";
  note: string;
  invited_by_username: string;
  responded_by_username: string;
  created_at: string;
  expires_at: string;
  responded_at?: string;
}

export interface TeamLog {
  id: number;
  team_id: number;
  at: string;
  actor_id: string;
  actor_kind: string;
  actor_name: string;
  actor_username: string;
  kind: string;
  field: string;
  old_value: string;
  new_value: string;
  message: string;
  data_json: string;
}

export interface AgentOwner {
  actor_id: string;
  kind: string;
  username: string;
  name: string;
  added_at: string;
}

const TEAMS = "/api/rest/teams";

export const teamApi = {
  async list(): Promise<{ teams: Team[] }> {
    return request("GET", TEAMS);
  },

  async create(name: string, description: string): Promise<{ team: Team }> {
    return request("POST", TEAMS, { name, description });
  },

  async get(id: number): Promise<{ team: Team }> {
    return request("GET", `${TEAMS}/${id}`);
  },

  async update(
    id: number,
    name: string,
    description: string,
  ): Promise<{ team: Team }> {
    return request("PUT", `${TEAMS}/${id}`, { name, description });
  },

  async remove(id: number): Promise<void> {
    await request("DELETE", `${TEAMS}/${id}`);
  },

  async members(id: number): Promise<{ members: TeamMember[] }> {
    return request("GET", `${TEAMS}/${id}/members`);
  },

  async removeMember(id: number, actorId: string): Promise<void> {
    await request("DELETE", `${TEAMS}/${id}/members/${actorId}`);
  },

  async invitations(id: number): Promise<{ invitations: TeamInvitation[] }> {
    return request("GET", `${TEAMS}/${id}/invitations`);
  },

  async invite(
    id: number,
    username: string,
    role: TeamRole,
    note = "",
  ): Promise<{ invitation: TeamInvitation }> {
    return request("POST", `${TEAMS}/${id}/invitations`, {
      username,
      role,
      note,
    });
  },

  async revokeInvitation(id: number, invitationId: number): Promise<void> {
    await request("DELETE", `${TEAMS}/${id}/invitations/${invitationId}`);
  },

  async pendingInvitations(): Promise<{ invitations: TeamInvitation[] }> {
    return request("GET", `${TEAMS}/invitations/pending`);
  },

  async respond(
    invitationId: number,
    approve: boolean,
  ): Promise<{ invitation: TeamInvitation }> {
    return request("POST", `${TEAMS}/invitations/${invitationId}/respond`, {
      approve,
    });
  },

  async logs(
    id: number,
    limit = 100,
    before = 0,
  ): Promise<{ logs: TeamLog[] }> {
    const params = new URLSearchParams({ limit: String(limit) });
    if (before) params.set("before", String(before));
    return request("GET", `${TEAMS}/${id}/logs?${params.toString()}`);
  },

  async projects(id: number): Promise<{ projects: TeamProject[] }> {
    return request("GET", `${TEAMS}/${id}/projects`);
  },
};

export const agentOwnerApi = {
  async list(agentId: string): Promise<{ owners: AgentOwner[] }> {
    return request("GET", `/api/rest/agents/${agentId}/owners`);
  },

  async add(agentId: string, username: string): Promise<{ owner: AgentOwner }> {
    return request("POST", `/api/rest/agents/${agentId}/owners`, { username });
  },

  async remove(agentId: string, ownerActorId: string): Promise<void> {
    await request(
      "DELETE",
      `/api/rest/agents/${agentId}/owners/${ownerActorId}`,
    );
  },
};

export const agentTeamApi = {
  async list(agentId: string): Promise<{ teams: Team[] }> {
    return request("GET", `/api/rest/agents/${agentId}/teams`);
  },

  async invitations(
    agentId: string,
  ): Promise<{ invitations: TeamInvitation[] }> {
    return request("GET", `/api/rest/agents/${agentId}/invitations`);
  },

  async respond(
    agentId: string,
    invitationId: number,
    approve: boolean,
  ): Promise<{ invitation: TeamInvitation }> {
    return request(
      "POST",
      `/api/rest/agents/${agentId}/invitations/${invitationId}/respond`,
      { approve },
    );
  },

  async leave(agentId: string, teamId: number): Promise<void> {
    await request("DELETE", `/api/rest/agents/${agentId}/teams/${teamId}`);
  },
};
