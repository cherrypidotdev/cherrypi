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

package agentgroupsgrpc

import (
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agentgroups"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	Groups *agentgroups.Service
	Agents *agents.Registry
}

func (s *Server) ListAgentGroups(
	ctx context.Context,
	_ *agencyv1.ListAgentGroupsRequest,
) (*agencyv1.ListAgentGroupsResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	list, err := s.Groups.List(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.AgentGroup, len(list))
	for i, group := range list {
		out[i] = s.groupToProto(ctx, identity, group)
	}
	return &agencyv1.ListAgentGroupsResponse{Groups: out}, nil
}

func (s *Server) GetAgentGroup(
	ctx context.Context,
	req *agencyv1.GetAgentGroupRequest,
) (*agencyv1.GetAgentGroupResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	group, err := s.Groups.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.GetAgentGroupResponse{Group: s.groupToProto(ctx, identity, group)}, nil
}

func (s *Server) ResolveAgentGroupBySlug(
	ctx context.Context,
	req *agencyv1.ResolveAgentGroupBySlugRequest,
) (*agencyv1.ResolveAgentGroupBySlugResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	group, err := s.Groups.GetBySlug(ctx, req.Slug)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.ResolveAgentGroupBySlugResponse{
		Group: s.groupToProto(ctx, identity, group),
	}, nil
}

func (s *Server) CreateAgentGroup(
	ctx context.Context,
	req *agencyv1.CreateAgentGroupRequest,
) (*agencyv1.CreateAgentGroupResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	if identity.Kind != "user" && !identity.IsAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, agentgroups.ErrForbidden)
	}
	group, err := s.Groups.Create(ctx, identity.ActorID, req.Name, req.Slug, req.Description)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.CreateAgentGroupResponse{Group: s.groupToProto(ctx, identity, group)}, nil
}

func (s *Server) UpdateAgentGroup(
	ctx context.Context,
	req *agencyv1.UpdateAgentGroupRequest,
) (*agencyv1.UpdateAgentGroupResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	group, err := s.Groups.Update(ctx, identity, req.Id, req.Name, req.Slug, req.Description)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateAgentGroupResponse{Group: s.groupToProto(ctx, identity, group)}, nil
}

func (s *Server) SetAgentGroupArchived(
	ctx context.Context,
	req *agencyv1.SetAgentGroupArchivedRequest,
) (*agencyv1.SetAgentGroupArchivedResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	group, err := s.Groups.SetArchived(ctx, identity, req.Id, req.Archived)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.SetAgentGroupArchivedResponse{
		Group: s.groupToProto(ctx, identity, group),
	}, nil
}

func (s *Server) ListAgentGroupMembers(
	ctx context.Context,
	req *agencyv1.ListAgentGroupMembersRequest,
) (*agencyv1.ListAgentGroupMembersResponse, error) {
	list, err := s.Groups.Members(ctx, req.GroupId)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.AgentGroupMember, len(list))
	for i, member := range list {
		item := &agencyv1.AgentGroupMember{
			GroupId: member.GroupID,
			AgentId: actorid.Format(member.AgentID),
			Agent: &agencyv1.AgentGroupActor{
				ActorId:  actorid.Format(member.ActorID),
				Username: member.Username,
				Name:     member.Name,
				Kind:     "agent",
				Picture:  picture(member.PictureAttachmentID),
			},
			CreatedAt: timestamppb.New(member.CreatedAt),
		}
		if member.LastSeenAt != nil {
			item.LastSeenAt = timestamppb.New(*member.LastSeenAt)
		}
		if member.AddedByActorID != 0 {
			item.AddedBy = &agencyv1.AgentGroupActor{
				ActorId:  actorid.Format(member.AddedByActorID),
				Username: member.AddedByUsername,
			}
		}
		out[i] = item
	}
	return &agencyv1.ListAgentGroupMembersResponse{Members: out}, nil
}

func (s *Server) ListAgentGroupsForAgent(
	ctx context.Context,
	req *agencyv1.ListAgentGroupsForAgentRequest,
) (*agencyv1.ListAgentGroupsForAgentResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	list, err := s.Groups.ForAgent(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.AgentGroup, len(list))
	for i, group := range list {
		out[i] = s.groupToProto(ctx, identity, group)
	}
	return &agencyv1.ListAgentGroupsForAgentResponse{Groups: out}, nil
}

func (s *Server) AddAgentGroupMember(
	ctx context.Context,
	req *agencyv1.AddAgentGroupMemberRequest,
) (*agencyv1.AddAgentGroupMemberResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	result, err := s.Groups.AddMember(
		ctx,
		identity,
		req.GroupId,
		actorid.ParseAgent(req.AgentId),
		req.Note,
	)
	if err != nil {
		return nil, mapError(err)
	}
	out := &agencyv1.AgentGroupMembershipResult{Added: result.Added}
	if !result.Added {
		out.Request = s.requestToProto(result.Request)
	}
	return &agencyv1.AddAgentGroupMemberResponse{Result: out}, nil
}

func (s *Server) RemoveAgentGroupMember(
	ctx context.Context,
	req *agencyv1.RemoveAgentGroupMemberRequest,
) (*agencyv1.RemoveAgentGroupMemberResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	if err := s.Groups.RemoveMember(
		ctx,
		identity,
		req.GroupId,
		actorid.ParseAgent(req.AgentId),
	); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RemoveAgentGroupMemberResponse{}, nil
}

func (s *Server) ListAgentGroupMemberRequests(
	ctx context.Context,
	req *agencyv1.ListAgentGroupMemberRequestsRequest,
) (*agencyv1.ListAgentGroupMemberRequestsResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	list, err := s.Groups.MemberRequests(ctx, identity, req.GroupId, req.Status)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.AgentGroupMemberRequest, len(list))
	for i, request := range list {
		out[i] = s.requestToProto(request)
	}
	return &agencyv1.ListAgentGroupMemberRequestsResponse{Requests: out}, nil
}

func (s *Server) DecideAgentGroupMember(
	ctx context.Context,
	req *agencyv1.DecideAgentGroupMemberRequest,
) (*agencyv1.DecideAgentGroupMemberResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	request, err := s.Groups.RespondRequest(ctx, identity, req.Id, req.Approve, req.Note)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DecideAgentGroupMemberResponse{
		Request: s.requestToProto(request),
	}, nil
}

func (s *Server) ListAgentGroupOwners(
	ctx context.Context,
	req *agencyv1.ListAgentGroupOwnersRequest,
) (*agencyv1.ListAgentGroupOwnersResponse, error) {
	list, err := s.Groups.Owners(ctx, req.GroupId)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.AgentGroupOwner, len(list))
	for i, owner := range list {
		out[i] = &agencyv1.AgentGroupOwner{
			GroupId: req.GroupId,
			Actor: &agencyv1.AgentGroupActor{
				ActorId:  actorid.Format(owner.ActorID),
				Username: owner.Username,
				Name:     owner.Name,
				Kind:     owner.Kind,
				Picture:  picture(owner.PictureAttachmentID),
			},
			CreatedAt: timestamppb.New(owner.CreatedAt),
		}
	}
	return &agencyv1.ListAgentGroupOwnersResponse{Owners: out}, nil
}

func (s *Server) AddAgentGroupOwner(
	ctx context.Context,
	req *agencyv1.AddAgentGroupOwnerRequest,
) (*agencyv1.AddAgentGroupOwnerResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	owner, err := s.Groups.AddOwner(ctx, identity, req.GroupId, actorid.ParseActor(req.ActorId))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.AddAgentGroupOwnerResponse{
		Owner: &agencyv1.AgentGroupOwner{
			GroupId: req.GroupId,
			Actor: &agencyv1.AgentGroupActor{
				ActorId:  actorid.Format(owner.ActorID),
				Username: owner.Username,
				Name:     owner.Name,
				Kind:     owner.Kind,
				Picture:  picture(owner.PictureAttachmentID),
			},
			CreatedAt: timestamppb.New(owner.CreatedAt),
		},
	}, nil
}

func (s *Server) RemoveAgentGroupOwner(
	ctx context.Context,
	req *agencyv1.RemoveAgentGroupOwnerRequest,
) (*agencyv1.RemoveAgentGroupOwnerResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	if err := s.Groups.RemoveOwner(
		ctx,
		identity,
		req.GroupId,
		actorid.ParseActor(req.ActorId),
	); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RemoveAgentGroupOwnerResponse{}, nil
}

func (s *Server) ListAgentGroupProjects(
	ctx context.Context,
	req *agencyv1.ListAgentGroupProjectsRequest,
) (*agencyv1.ListAgentGroupProjectsResponse, error) {
	list, err := s.Groups.GroupProjects(ctx, req.GroupId)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.AgentGroupProject, len(list))
	for i, item := range list {
		out[i] = &agencyv1.AgentGroupProject{
			Project: &agencyv1.Project{
				Id:          item.Project.ID,
				Name:        item.Project.Name,
				Slug:        item.Project.Slug,
				Prefix:      item.Project.Prefix,
				Description: item.Project.Description,
			},
			MemberCount:    int32(item.MemberCount),
			Prompt:         s.promptToProto(item.Prompt),
			Disabled:       item.Disabled,
			DisabledReason: item.DisabledReason,
		}
	}
	return &agencyv1.ListAgentGroupProjectsResponse{Projects: out}, nil
}

func (s *Server) ListAgentGroupTeams(
	ctx context.Context,
	req *agencyv1.ListAgentGroupTeamsRequest,
) (*agencyv1.ListAgentGroupTeamsResponse, error) {
	list, err := s.Groups.GroupTeams(ctx, req.GroupId)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.AgentGroupTeam, len(list))
	for i, item := range list {
		out[i] = &agencyv1.AgentGroupTeam{
			TeamId:     item.TeamID,
			Name:       item.Name,
			ProjectIds: item.ProjectIDs,
			Prompt:     s.promptToProto(item.Prompt),
		}
	}
	return &agencyv1.ListAgentGroupTeamsResponse{Teams: out}, nil
}

func (s *Server) ListAgentGroupLogs(
	ctx context.Context,
	req *agencyv1.ListAgentGroupLogsRequest,
) (*agencyv1.ListAgentGroupLogsResponse, error) {
	list, err := s.Groups.Logs(ctx, req.GroupId, int(req.Limit), req.BeforeId)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.AgentGroupLog, len(list))
	for i, entry := range list {
		out[i] = &agencyv1.AgentGroupLog{
			Id:       entry.ID,
			GroupId:  entry.GroupID,
			At:       timestamppb.New(entry.At),
			Kind:     entry.Kind,
			Field:    entry.Field,
			OldValue: entry.OldValue,
			NewValue: entry.NewValue,
			Message:  entry.Message,
			DataJson: entry.DataJSON,
		}
		if entry.ActorID != 0 {
			out[i].Actor = &agencyv1.AgentGroupActor{
				ActorId:  actorid.Format(entry.ActorID),
				Username: entry.ActorUsername,
				Name:     entry.ActorName,
				Kind:     entry.ActorKind,
			}
		}
	}
	return &agencyv1.ListAgentGroupLogsResponse{Logs: out}, nil
}

func (s *Server) ListAgentGroupAccess(
	ctx context.Context,
	req *agencyv1.ListAgentGroupAccessRequest,
) (*agencyv1.ListAgentGroupAccessResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	access, err := s.Groups.Access(ctx, identity, req.GroupId)
	if err != nil {
		return nil, mapError(err)
	}
	out := &agencyv1.AgentGroupAccess{
		Projects:    make([]*agencyv1.AgentGroupProjectAccess, len(access.Projects)),
		Teams:       make([]*agencyv1.AgentGroupTeamAccess, len(access.Teams)),
		Invitations: make([]*agencyv1.AgentGroupAccessInvitation, len(access.Invitations)),
	}
	for i, item := range access.Projects {
		out.Projects[i] = projectAccessToProto(item)
	}
	for i, item := range access.Teams {
		out.Teams[i] = teamAccessToProto(item)
	}
	for i, item := range access.Invitations {
		out.Invitations[i] = accessInvitationToProto(item)
	}
	return &agencyv1.ListAgentGroupAccessResponse{Access: out}, nil
}

func (s *Server) InviteAgentGroupToProject(
	ctx context.Context,
	req *agencyv1.InviteAgentGroupToProjectRequest,
) (*agencyv1.InviteAgentGroupToProjectResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	result, err := s.Groups.InviteProject(
		ctx, identity, req.ProjectId, req.GroupId, req.Role, req.Note,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.InviteAgentGroupToProjectResponse{
		Attached:   result.Attached,
		Invitation: accessInvitationToProto(result.Invitation),
	}, nil
}

func (s *Server) InviteAgentGroupToTeam(
	ctx context.Context,
	req *agencyv1.InviteAgentGroupToTeamRequest,
) (*agencyv1.InviteAgentGroupToTeamResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	result, err := s.Groups.InviteTeam(ctx, identity, req.TeamId, req.GroupId, req.Note)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.InviteAgentGroupToTeamResponse{
		Attached:   result.Attached,
		Invitation: accessInvitationToProto(result.Invitation),
	}, nil
}

func (s *Server) DecideAgentGroupAccess(
	ctx context.Context,
	req *agencyv1.DecideAgentGroupAccessRequest,
) (*agencyv1.DecideAgentGroupAccessResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	invitation, err := s.Groups.DecideAccess(
		ctx, identity, req.TargetKind, req.InvitationId, req.Approve, req.Note,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DecideAgentGroupAccessResponse{
		Invitation: accessInvitationToProto(invitation),
	}, nil
}

func (s *Server) RevokeAgentGroupAccess(
	ctx context.Context,
	req *agencyv1.RevokeAgentGroupAccessRequest,
) (*agencyv1.RevokeAgentGroupAccessResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	if err := s.Groups.RevokeAccess(
		ctx, identity, req.TargetKind, req.InvitationId,
	); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RevokeAgentGroupAccessResponse{}, nil
}

func (s *Server) RemoveAgentGroupFromProject(
	ctx context.Context,
	req *agencyv1.RemoveAgentGroupFromProjectRequest,
) (*agencyv1.RemoveAgentGroupFromProjectResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	if err := s.Groups.LeaveProject(ctx, identity, req.ProjectId, req.GroupId); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RemoveAgentGroupFromProjectResponse{}, nil
}

func (s *Server) RemoveAgentGroupFromTeam(
	ctx context.Context,
	req *agencyv1.RemoveAgentGroupFromTeamRequest,
) (*agencyv1.RemoveAgentGroupFromTeamResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	if err := s.Groups.LeaveTeam(ctx, identity, req.TeamId, req.GroupId); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RemoveAgentGroupFromTeamResponse{}, nil
}

func (s *Server) ListProjectAgentGroups(
	ctx context.Context,
	req *agencyv1.ListProjectAgentGroupsRequest,
) (*agencyv1.ListProjectAgentGroupsResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	access, err := s.Groups.ProjectAccess(
		ctx, req.ProjectId, identity.ActorID, identity.IsAdmin(),
	)
	if err != nil {
		return nil, mapError(err)
	}
	out := &agencyv1.ListProjectAgentGroupsResponse{
		Links:       make([]*agencyv1.AgentGroupProjectGroup, len(access.Links)),
		Inherited:   make([]*agencyv1.AgentGroupProjectGroup, len(access.Inherited)),
		Invitations: make([]*agencyv1.AgentGroupAccessInvitation, len(access.Invitations)),
	}
	for i, link := range access.Links {
		out.Links[i] = projectGroupToProto(link)
	}
	for i, link := range access.Inherited {
		out.Inherited[i] = projectGroupToProto(link)
	}
	for i, invitation := range access.Invitations {
		out.Invitations[i] = accessInvitationToProto(invitation)
	}
	return out, nil
}

func (s *Server) ListTeamAgentGroups(
	ctx context.Context,
	req *agencyv1.ListTeamAgentGroupsRequest,
) (*agencyv1.ListTeamAgentGroupsResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, agentgroups.ErrForbidden)
	}
	access, err := s.Groups.TeamAccess(
		ctx, req.TeamId, identity.ActorID, identity.IsAdmin(),
	)
	if err != nil {
		return nil, mapError(err)
	}
	out := &agencyv1.ListTeamAgentGroupsResponse{
		Links:       make([]*agencyv1.AgentGroupTeamGroup, len(access.Links)),
		Invitations: make([]*agencyv1.AgentGroupAccessInvitation, len(access.Invitations)),
	}
	for i, link := range access.Links {
		out.Links[i] = teamGroupToProto(link)
	}
	for i, invitation := range access.Invitations {
		out.Invitations[i] = accessInvitationToProto(invitation)
	}
	return out, nil
}

func projectAccessToProto(item agentgroups.ProjectAccess) *agencyv1.AgentGroupProjectAccess {
	return &agencyv1.AgentGroupProjectAccess{
		ProjectId: item.ProjectID,
		Name:      item.ProjectName,
		Slug:      item.ProjectSlug,
		Role:      item.Role,
		GrantedBy: accessActor(item.GrantedByActorID, item.GrantedByUsername, item.GrantedByName),
		CreatedAt: timestamppb.New(item.CreatedAt),
	}
}

func teamAccessToProto(item agentgroups.TeamAccess) *agencyv1.AgentGroupTeamAccess {
	return &agencyv1.AgentGroupTeamAccess{
		TeamId:      item.TeamID,
		Name:        item.TeamName,
		Description: item.TeamDescription,
		Role:        "member",
		MemberCount: int32(item.MemberCount),
		GrantedBy:   accessActor(item.GrantedByActorID, item.GrantedByUsername, item.GrantedByName),
		CreatedAt:   timestamppb.New(item.CreatedAt),
	}
}

func projectGroupToProto(item agentgroups.ProjectGroupLink) *agencyv1.AgentGroupProjectGroup {
	return &agencyv1.AgentGroupProjectGroup{
		GroupId:                  item.GroupID,
		Name:                     item.GroupName,
		Slug:                     item.GroupSlug,
		Description:              item.GroupDescription,
		Role:                     item.Role,
		MemberCount:              int32(item.MemberCount),
		Inherited:                item.Inherited,
		InheritedFromProjectId:   item.InheritedFromProjectID,
		InheritedFromProjectName: item.InheritedFromProjectName,
		InheritedFromProjectSlug: item.InheritedFromProjectSlug,
		GrantedBy: accessActor(
			item.GrantedByActorID,
			item.GrantedByUsername,
			item.GrantedByName,
		),
		CreatedAt: timestamppb.New(item.CreatedAt),
	}
}

func teamGroupToProto(item agentgroups.TeamGroupLink) *agencyv1.AgentGroupTeamGroup {
	return &agencyv1.AgentGroupTeamGroup{
		GroupId:     item.GroupID,
		Name:        item.GroupName,
		Slug:        item.GroupSlug,
		Description: item.GroupDescription,
		MemberCount: int32(item.MemberCount),
		GrantedBy:   accessActor(item.GrantedByActorID, item.GrantedByUsername, item.GrantedByName),
		CreatedAt:   timestamppb.New(item.CreatedAt),
	}
}

func accessInvitationToProto(
	item agentgroups.AccessInvitation,
) *agencyv1.AgentGroupAccessInvitation {
	out := &agencyv1.AgentGroupAccessInvitation{
		Id:          item.ID,
		TargetKind:  item.TargetKind,
		ProjectId:   item.ProjectID,
		ProjectName: item.ProjectName,
		ProjectSlug: item.ProjectSlug,
		TeamId:      item.TeamID,
		TeamName:    item.TeamName,
		GroupId:     item.GroupID,
		GroupName:   item.GroupName,
		GroupSlug:   item.GroupSlug,
		Role:        item.Role,
		Status:      item.Status,
		Note:        item.Note,
		InvitedBy:   accessActor(item.InvitedByActorID, item.InvitedByUsername, item.InvitedByName),
		CreatedAt:   timestamppb.New(item.CreatedAt),
		ExpiresAt:   timestamppb.New(item.ExpiresAt),
		CanRespond:  item.CanRespond,
	}
	if item.RespondedAt != nil {
		out.RespondedAt = timestamppb.New(*item.RespondedAt)
	}
	if item.RespondedByActorID != 0 {
		out.RespondedBy = accessActor(
			item.RespondedByActorID, item.RespondedByUsername, item.RespondedByName,
		)
	}
	return out
}

func accessActor(actorID ids.ActorID, username, name string) *agencyv1.AgentGroupActor {
	if actorID == 0 {
		return nil
	}
	return &agencyv1.AgentGroupActor{
		ActorId:  actorid.Format(actorID),
		Username: username,
		Name:     name,
	}
}

func (s *Server) groupToProto(
	ctx context.Context,
	identity auth.Identity,
	group agentgroups.Group,
) *agencyv1.AgentGroup {
	out := &agencyv1.AgentGroup{
		Id:          group.ID,
		Name:        group.Name,
		Slug:        group.Slug,
		Description: group.Description,
		Picture:     picture(group.PictureAttachmentID),
		CreatedAt:   timestamppb.New(group.CreatedAt),
		UpdatedAt:   timestamppb.New(group.UpdatedAt),
		MemberCount: int32(group.MemberCount),
	}
	if group.ArchivedAt != nil {
		out.ArchivedAt = timestamppb.New(*group.ArchivedAt)
	}
	if identity.IsAdmin() {
		out.CanEdit = true
		out.IsOwner = true
	} else if owner, err := s.Groups.IsOwner(ctx, group.ID, identity.ActorID); err == nil {
		out.CanEdit = owner
		out.IsOwner = owner
	}
	return out
}

func (s *Server) requestToProto(
	request agentgroups.MemberRequest,
) *agencyv1.AgentGroupMemberRequest {
	out := &agencyv1.AgentGroupMemberRequest{
		Id:        request.ID,
		GroupId:   request.GroupID,
		GroupName: request.GroupName,
		GroupSlug: request.GroupSlug,
		AgentId:   actorid.Format(request.AgentID),
		Agent: &agencyv1.AgentGroupActor{
			Username: request.AgentUsername,
			Name:     request.AgentName,
			Kind:     "agent",
		},
		RequestedBy: &agencyv1.AgentGroupActor{
			ActorId:  actorid.Format(request.RequestedByActorID),
			Username: request.RequestedByUsername,
			Name:     request.RequestedByName,
			Kind:     "user",
		},
		Status:     request.Status,
		Note:       request.Note,
		CreatedAt:  timestamppb.New(request.CreatedAt),
		ExpiresAt:  timestamppb.New(request.ExpiresAt),
		CanRespond: request.CanRespond,
	}
	if request.RespondedByActorID != 0 {
		out.RespondedBy = &agencyv1.AgentGroupActor{
			ActorId:  actorid.Format(request.RespondedByActorID),
			Username: request.RespondedByUsername,
			Name:     request.RespondedByName,
		}
	}
	if request.RespondedAt != nil {
		out.RespondedAt = timestamppb.New(*request.RespondedAt)
	}
	return out
}

func (s *Server) promptToProto(p prompts.Prompt) *agencyv1.Prompt {
	if p.ID == 0 {
		return nil
	}
	return &agencyv1.Prompt{
		Content:   p.Content,
		Version:   p.Version,
		Id:        p.ID,
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}

func picture(id int64) *agencyv1.Attachment {
	if id <= 0 {
		return nil
	}
	return &agencyv1.Attachment{Id: id, Url: attachments.URLFor(id)}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, agentgroups.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, agentgroups.ErrForbidden),
		errors.Is(err, agentgroups.ErrRequestNotDecider):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, agentgroups.ErrAccessForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, agentgroups.ErrAccessNotFound),
		errors.Is(err, agentgroups.ErrAccessNotLinked):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, agentgroups.ErrAccessAlreadyLinked),
		errors.Is(err, agentgroups.ErrAccessInvitationExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, agentgroups.ErrInvalidName),
		errors.Is(err, agentgroups.ErrInvalidSlug),
		errors.Is(err, agentgroups.ErrSlugInUse),
		errors.Is(err, agentgroups.ErrArchived),
		errors.Is(err, agentgroups.ErrMemberExists),
		errors.Is(err, agentgroups.ErrMemberNotFound),
		errors.Is(err, agentgroups.ErrAgentNotFound),
		errors.Is(err, agentgroups.ErrLastOwner),
		errors.Is(err, agentgroups.ErrOwnerNotFound),
		errors.Is(err, agentgroups.ErrOwnerIsNotUser),
		errors.Is(err, agentgroups.ErrRequestExists),
		errors.Is(err, agentgroups.ErrRequestNotPending),
		errors.Is(err, agentgroups.ErrRequestExpired),
		errors.Is(err, agentgroups.ErrAccessNotPending),
		errors.Is(err, agentgroups.ErrAccessExpired),
		errors.Is(err, agentgroups.ErrAccessInvalidRole),
		errors.Is(err, agentgroups.ErrAccessInvalidTarget):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
