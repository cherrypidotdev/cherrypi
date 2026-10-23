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

/** Project access roles, ordered viewer < member < owner. */
export type ProjectRole = "owner" | "member" | "viewer";

/** One direct actor grant on a project. */
export interface ProjectMemberGrant {
  actor_id: string;
  kind: string;
  username: string;
  name: string;
  description: string;
  role: ProjectRole;
  granted_by_actor_id: string;
  created_at: string;
  updated_at: string;
}

/** One team attached to a project. */
export interface ProjectTeamLink {
  team_id: number;
  team_name: string;
  team_description: string;
  role: ProjectRole;
  member_count: number;
  created_at: string;
  updated_at: string;
}

/** A pending or answered project-team invitation. */
export interface ProjectTeamInvitation {
  id: number;
  project_id: string;
  project_name: string;
  project_slug: string;
  team_id: number;
  team_name: string;
  team_description: string;
  role: ProjectRole;
  status: "pending" | "accepted" | "declined" | "revoked";
  note: string;
  invited_by_username: string;
  responded_by_username: string;
  created_at: string;
  expires_at: string;
  responded_at?: string;
}

/** The caller's view of a project's access. */
export interface ProjectAccess {
  project_id: string;
  role: ProjectRole;
  members: ProjectMemberGrant[];
  teams: ProjectTeamLink[];
  invitations: ProjectTeamInvitation[];
  can_delete: boolean;
  can_manage_access: boolean;
}

/** A project summary in the hierarchy surface. */
export interface ProjectRef {
  id: string;
  name: string;
  slug: string;
  prefix: string;
  description: string;
  parent_id: string;
  archived: boolean;
}

/** An owner-role team shared by both sides of a prospective parent link. */
export interface SharedOwnerTeam {
  team_id: number;
  team_name: string;
}

/** A project that may become this one's parent or child. */
export interface ParentOption {
  project: ProjectRef;
  /** The caller owns the parent side and may create an invitation. */
  can_invite: boolean;
  /** A shared owner team (or admin) allows linking without an invitation. */
  can_link_directly: boolean;
  shared_owner_teams: SharedOwnerTeam[];
}

/** A pending or answered invitation for a project to become a child. */
export interface ParentInvitation {
  id: number;
  parent_project_id: string;
  parent_project_name: string;
  parent_project_slug: string;
  child_project_id: string;
  child_project_name: string;
  child_project_slug: string;
  status: "pending" | "accepted" | "declined" | "revoked";
  note: string;
  invited_by_username: string;
  responded_by_username: string;
  created_at: string;
  expires_at: string;
  responded_at?: string;
}

/** One project's position in the tree and the link actions it exposes. */
export interface ProjectHierarchy {
  project: ProjectRef;
  parent?: ProjectRef;
  children: ProjectRef[];
  incoming_invitation?: ParentInvitation;
  outgoing_invitations: ParentInvitation[];
  parent_options: ParentOption[];
  child_options: ParentOption[];
  can_detach: boolean;
  can_respond: boolean;
  can_invite_child: boolean;
  /** Projects a task here may reference (self, descendants, siblings). */
  reference_project_ids: string[];
}

/** One actor a task in a project may name as a lead. */
export interface LeadCandidate {
  actor_id: string;
  kind: string;
  username: string;
  name: string;
  /** Uploaded picture URL; unset for an agent or a user without one. */
  picture_url?: string;
}

/** One project domain-log entry. */
export interface ProjectLog {
  id: number;
  project_id: string;
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

/** Human-readable role label for the access tables and selects. */
export const PROJECT_ROLE_LABEL: Record<ProjectRole, string> = {
  owner: "Owner",
  member: "Member",
  viewer: "Viewer",
};

/** One-line description of what a role may do, used in pickers. */
export const PROJECT_ROLE_HINT: Record<ProjectRole, string> = {
  owner: "Manage access, prompt and edit",
  member: "Prompt and edit",
  viewer: "View and comment only",
};

const PROJECTS = "/api/rest/projects";

export const projectAccessApi = {
  async access(projectId: string): Promise<ProjectAccess> {
    return request("GET", `${PROJECTS}/${projectId}/access`);
  },

  async hierarchy(projectId: string): Promise<ProjectHierarchy> {
    return request("GET", `${PROJECTS}/${projectId}/hierarchy`);
  },

  async inviteParent(
    childId: string,
    parentId: string,
    note = "",
  ): Promise<{
    project: ProjectRef;
    invitation?: ParentInvitation;
    direct: boolean;
  }> {
    return request("POST", `${PROJECTS}/${childId}/parent`, {
      parent_id: parentId,
      note,
    });
  },

  async detachParent(childId: string): Promise<{ project: ProjectRef }> {
    return request("DELETE", `${PROJECTS}/${childId}/parent`);
  },

  async respondParentInvitation(
    invitationId: number,
    approve: boolean,
  ): Promise<{ invitation: ParentInvitation }> {
    return request(
      "POST",
      `${PROJECTS}/parent-invitations/${invitationId}/respond`,
      { approve },
    );
  },

  async revokeParentInvitation(invitationId: number): Promise<void> {
    await request("DELETE", `${PROJECTS}/parent-invitations/${invitationId}`);
  },

  async leadCandidates(projectId: string): Promise<LeadCandidate[]> {
    const res = await request<{ candidates: LeadCandidate[] }>(
      "GET",
      `${PROJECTS}/${projectId}/lead-candidates`,
    );
    return res.candidates;
  },

  async addMember(
    projectId: string,
    username: string,
    role: ProjectRole,
  ): Promise<{ member: ProjectMemberGrant }> {
    return request("POST", `${PROJECTS}/${projectId}/members`, {
      username,
      role,
    });
  },

  async setMemberRole(
    projectId: string,
    actorId: string,
    role: ProjectRole,
  ): Promise<{ member: ProjectMemberGrant }> {
    return request("PUT", `${PROJECTS}/${projectId}/members/${actorId}`, {
      role,
    });
  },

  async removeMember(projectId: string, actorId: string): Promise<void> {
    await request("DELETE", `${PROJECTS}/${projectId}/members/${actorId}`);
  },

  async inviteTeam(
    projectId: string,
    teamId: number,
    role: ProjectRole,
    note = "",
  ): Promise<{ invitation: ProjectTeamInvitation }> {
    return request("POST", `${PROJECTS}/${projectId}/teams`, {
      team_id: teamId,
      role,
      note,
    });
  },

  async setTeamRole(
    projectId: string,
    teamId: number,
    role: ProjectRole,
  ): Promise<{ team: ProjectTeamLink }> {
    return request("PUT", `${PROJECTS}/${projectId}/teams/${teamId}`, { role });
  },

  async removeTeam(projectId: string, teamId: number): Promise<void> {
    await request("DELETE", `${PROJECTS}/${projectId}/teams/${teamId}`);
  },

  async revokeInvitation(
    projectId: string,
    invitationId: number,
  ): Promise<void> {
    await request(
      "DELETE",
      `${PROJECTS}/${projectId}/invitations/${invitationId}`,
    );
  },

  /** Pending invitations addressed to teams the caller owns. */
  async pendingInvitations(): Promise<{
    invitations: ProjectTeamInvitation[];
  }> {
    return request("GET", `${PROJECTS}/invitations/pending`);
  },

  async respond(
    invitationId: number,
    approve: boolean,
  ): Promise<{ invitation: ProjectTeamInvitation }> {
    return request("POST", `${PROJECTS}/invitations/${invitationId}/respond`, {
      approve,
    });
  },

  async logs(
    projectId: string,
    limit = 100,
    before = 0,
  ): Promise<{ logs: ProjectLog[] }> {
    const params = new URLSearchParams({ limit: String(limit) });
    if (before) params.set("before", String(before));
    return request("GET", `${PROJECTS}/${projectId}/logs?${params.toString()}`);
  },
};
