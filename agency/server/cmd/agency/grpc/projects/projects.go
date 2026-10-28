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

package projectsgrpc

import (
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"context"
	"errors"
	"fmt"

	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProjectServer struct {
	Projects *projects.Service
	Agents   *agents.Registry
	Hub      *tunnel.Hub

	Attachments *attachments.Service
}

func (s *ProjectServer) ListProjects(
	ctx context.Context,
	_ *agencyv1.ListProjectsRequest,
) (*agencyv1.ListProjectsResponse, error) {
	var (
		list []projects.Project
		err  error
	)

	if identity, ok := auth.IdentityFrom(ctx); ok && !identity.IsAdmin() {
		list, err = s.Projects.ProjectsForActor(ctx, identity.ActorID)
	} else {
		list, err = s.Projects.List(ctx)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var pictures map[string]attachments.Attachment
	if s.Attachments != nil {
		pictures, _ = s.Attachments.ProjectPictures(ctx)
	}
	out := make([]*agencyv1.Project, len(list))
	for i, p := range list {
		out[i] = toProto(p)
		if picture, ok := pictures[p.ID]; ok {
			out[i].Picture = attachments.ToProto(picture)
		}
	}
	return &agencyv1.ListProjectsResponse{Projects: out}, nil
}

func (s *ProjectServer) GetProject(
	ctx context.Context,
	req *agencyv1.GetProjectRequest,
) (*agencyv1.GetProjectResponse, error) {
	p, err := s.Projects.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	out := toProto(p)
	s.enrichPicture(ctx, p.ID, out)
	return &agencyv1.GetProjectResponse{Project: out}, nil
}

func (s *ProjectServer) enrichPicture(
	ctx context.Context,
	projectID string,
	out *agencyv1.Project,
) {
	if s.Attachments == nil {
		return
	}
	if picture, ok := s.Attachments.ProjectPicture(ctx, projectID); ok {
		out.Picture = attachments.ToProto(picture)
	}
}

func (s *ProjectServer) CreateProject(
	ctx context.Context,
	req *agencyv1.CreateProjectRequest,
) (*agencyv1.CreateProjectResponse, error) {
	p, err := s.Projects.Create(ctx, req.Name, req.Slug, req.Prefix, req.Description)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.CreateProjectResponse{Project: toProto(p)}, nil
}

func (s *ProjectServer) UpdateProject(
	ctx context.Context,
	req *agencyv1.UpdateProjectRequest,
) (*agencyv1.UpdateProjectResponse, error) {
	p, err := s.Projects.Update(ctx, req.Id, req.Name, req.Description, req.Prefix)
	if err != nil {
		return nil, mapError(err)
	}
	out := toProto(p)
	s.enrichPicture(ctx, p.ID, out)
	return &agencyv1.UpdateProjectResponse{Project: out}, nil
}

func (s *ProjectServer) requireOwner(ctx context.Context, projectID string) error {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil
	}
	_, err := s.Projects.RequireRole(
		ctx, projectID, identity.ActorID, identity.IsAdmin(), projects.RoleOwner,
	)
	return err
}

func (s *ProjectServer) ArchiveProject(
	ctx context.Context,
	req *agencyv1.ArchiveProjectRequest,
) (*agencyv1.ArchiveProjectResponse, error) {
	if err := s.requireOwner(ctx, req.Id); err != nil {
		return nil, mapError(err)
	}
	p, err := s.Projects.SetArchived(ctx, req.Id, true)
	if err != nil {
		return nil, mapError(err)
	}

	out := toProto(p)
	s.enrichPicture(ctx, p.ID, out)
	return &agencyv1.ArchiveProjectResponse{Project: out}, nil
}

func (s *ProjectServer) UnarchiveProject(
	ctx context.Context,
	req *agencyv1.UnarchiveProjectRequest,
) (*agencyv1.UnarchiveProjectResponse, error) {
	if err := s.requireOwner(ctx, req.Id); err != nil {
		return nil, mapError(err)
	}
	p, err := s.Projects.SetArchived(ctx, req.Id, false)
	if err != nil {
		return nil, mapError(err)
	}
	out := toProto(p)
	s.enrichPicture(ctx, p.ID, out)
	return &agencyv1.UnarchiveProjectResponse{Project: out}, nil
}

func (s *ProjectServer) DeleteProject(
	ctx context.Context,
	req *agencyv1.DeleteProjectRequest,
) (*agencyv1.DeleteProjectResponse, error) {
	if err := s.requireOwner(ctx, req.Id); err != nil {
		return nil, mapError(err)
	}
	if err := s.Projects.Delete(ctx, req.Id); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DeleteProjectResponse{}, nil
}

func (s *ProjectServer) ListProjectAgents(
	ctx context.Context,
	req *agencyv1.ListProjectAgentsRequest,
) (*agencyv1.ListProjectAgentsResponse, error) {
	list, err := s.Projects.ListAgents(ctx, req.ProjectId)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.ProjectAgent, len(list))
	for i, member := range list {
		out[i] = toProtoAgent(member)
	}
	return &agencyv1.ListProjectAgentsResponse{Agents: out}, nil
}

func (s *ProjectServer) ListAgentProjects(
	ctx context.Context,
	req *agencyv1.ListAgentProjectsRequest,
) (*agencyv1.ListAgentProjectsResponse, error) {
	if _, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId)); err != nil {
		if errors.Is(err, agents.ErrAgentMissing) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	list, err := s.Projects.ListAgentProjects(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}

	if identity, ok := auth.IdentityFrom(ctx); ok && !identity.IsAdmin() {
		visible, err := s.Projects.ProjectsForActor(ctx, identity.ActorID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		allowed := make(map[string]bool, len(visible))
		for _, project := range visible {
			allowed[project.ID] = true
		}
		kept := list[:0]
		for _, project := range list {
			if allowed[project.ID] {
				kept = append(kept, project)
			}
		}
		list = kept
	}
	var pictures map[string]attachments.Attachment
	if s.Attachments != nil {
		pictures, _ = s.Attachments.ProjectPictures(ctx)
	}
	out := make([]*agencyv1.Project, len(list))
	for i, p := range list {
		out[i] = toProto(p)
		if picture, ok := pictures[p.ID]; ok {
			out[i].Picture = attachments.ToProto(picture)
		}
	}
	return &agencyv1.ListAgentProjectsResponse{Projects: out}, nil
}

func (s *ProjectServer) ListAgentMemberships(
	ctx context.Context,
	req *agencyv1.ListAgentMembershipsRequest,
) (*agencyv1.ListAgentMembershipsResponse, error) {
	if _, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId)); err != nil {
		if errors.Is(err, agents.ErrAgentMissing) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	list, err := s.Projects.ListAgentMemberships(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}
	var pictures map[string]attachments.Attachment
	if s.Attachments != nil {
		pictures, _ = s.Attachments.ProjectPictures(ctx)
	}
	out := make([]*agencyv1.AgentProjectMembership, len(list))
	for i, m := range list {
		project := toProto(m.Project)
		if picture, ok := pictures[m.Project.ID]; ok {
			project.Picture = attachments.ToProto(picture)
		}
		out[i] = &agencyv1.AgentProjectMembership{
			Project:        project,
			CanListTasks:   m.CanListTasks,
			CanCreateTasks: m.CanCreateTasks,
			CanUpdateTasks: m.CanUpdateTasks,
		}
	}
	return &agencyv1.ListAgentMembershipsResponse{Memberships: out}, nil
}

func (s *ProjectServer) AddProjectAgent(
	ctx context.Context,
	req *agencyv1.AddProjectAgentRequest,
) (*agencyv1.AddProjectAgentResponse, error) {
	if _, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId)); err != nil {
		if errors.Is(err, agents.ErrAgentMissing) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.requireOwner(ctx, req.ProjectId); err != nil {
		return nil, mapError(err)
	}
	member, err := s.Projects.AddAgent(ctx, req.ProjectId, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.AddProjectAgentResponse{Agent: toProtoAgent(member)}, nil
}

func (s *ProjectServer) RemoveProjectAgent(
	ctx context.Context,
	req *agencyv1.RemoveProjectAgentRequest,
) (*agencyv1.RemoveProjectAgentResponse, error) {
	if err := s.requireOwner(ctx, req.ProjectId); err != nil {
		return nil, mapError(err)
	}
	ids, err := s.Projects.AgentTaskIDs(ctx, req.ProjectId, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}

	if len(ids) > 0 {
		conn, err := s.Hub.Connection(actorid.ParseAgent(req.AgentId))
		if err != nil {
			return nil, connect.NewError(
				connect.CodeFailedPrecondition,
				fmt.Errorf(
					"agent is offline; its %d task(s) in the project must be deleted first",
					len(ids),
				),
			)
		}
		for _, id := range ids {
			if _, err := conn.Tasks.DeleteTask(
				ctx,
				&agentv1.DeleteTaskRequest{Id: id},
			); err != nil && connect.CodeOf(err) != connect.CodeNotFound {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
		if err := s.Projects.DeleteAgentTasks(
			ctx,
			req.ProjectId,
			actorid.ParseAgent(req.AgentId),
		); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := s.Projects.RemoveAgent(
		ctx,
		req.ProjectId,
		actorid.ParseAgent(req.AgentId),
	); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RemoveProjectAgentResponse{}, nil
}

func (s *ProjectServer) UpdateProjectAgent(
	ctx context.Context,
	req *agencyv1.UpdateProjectAgentRequest,
) (*agencyv1.UpdateProjectAgentResponse, error) {
	if err := s.requireOwner(ctx, req.ProjectId); err != nil {
		return nil, mapError(err)
	}
	member, err := s.Projects.UpdateAgentPermissions(
		ctx,
		req.ProjectId,
		actorid.ParseAgent(req.AgentId),
		projects.Permissions{
			CanListTasks:   req.CanListTasks,
			CanCreateTasks: req.CanCreateTasks,
			CanUpdateTasks: req.CanUpdateTasks,
		},
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateProjectAgentResponse{Agent: toProtoAgent(member)}, nil
}

func toProtoAgent(member projects.ProjectAgent) *agencyv1.ProjectAgent {
	return &agencyv1.ProjectAgent{
		AgentId:        actorid.Format(member.AgentID),
		TaskCount:      int32(member.TaskCount),
		CanListTasks:   member.CanListTasks,
		CanCreateTasks: member.CanCreateTasks,
		CanUpdateTasks: member.CanUpdateTasks,
	}
}

func toProto(p projects.Project) *agencyv1.Project {
	out := &agencyv1.Project{
		Id:              p.ID,
		Name:            p.Name,
		Slug:            p.Slug,
		Prefix:          p.Prefix,
		Description:     p.Description,
		ParentProjectId: p.ParentID,
		CreatedAt:       timestamppb.New(p.CreatedAt),
		UpdatedAt:       timestamppb.New(p.UpdatedAt),
	}
	if !p.ArchivedAt.IsZero() {
		out.ArchivedAt = timestamppb.New(p.ArchivedAt)
	}
	return out
}

func mapError(err error) *connect.Error {
	switch {
	case errors.Is(err, projects.ErrProjectMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, projects.ErrInvalidName):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, projects.ErrProjectHasTasks),
		errors.Is(err, projects.ErrProjectHasChildren),
		errors.Is(err, projects.ErrAgentHasTasks),
		errors.Is(err, projects.ErrAlreadyHasParent),
		errors.Is(err, projects.ErrHierarchyCycle),
		errors.Is(err, projects.ErrHierarchyHasReferences):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, projects.ErrParentInvitationNotFound),
		errors.Is(err, projects.ErrNoParent):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, projects.ErrForbidden),
		errors.Is(err, projects.ErrNotMember):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
