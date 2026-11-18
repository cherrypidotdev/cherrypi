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

package modelroutinggrpc

import (
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/agentconfig"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/modelrouting"
	"cherrypi.vladfaust.com/agency/internal/projects"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
)

type Server struct {
	Routing *modelrouting.Service

	Projects *projects.Service

	Catalog *agentconfig.Service
}

func (s *Server) GetOrganizationModelRouting(
	ctx context.Context,
	_ *agencyv1.GetOrganizationModelRoutingRequest,
) (*agencyv1.GetOrganizationModelRoutingResponse, error) {
	cur, err := s.Routing.GetOrganization(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.GetOrganizationModelRoutingResponse{
		Routing: toProtoOrganization(cur),
	}, nil
}

func (s *Server) UpdateOrganizationModelRouting(
	ctx context.Context,
	req *agencyv1.UpdateOrganizationModelRoutingRequest,
) (*agencyv1.UpdateOrganizationModelRoutingResponse, error) {
	if !auth.IsAdmin(ctx) {
		return nil, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("only administrators may change organization model routing"),
		)
	}
	next, err := s.Routing.UpdateOrganization(ctx, fromProtoOrganization(req.GetRouting()))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateOrganizationModelRoutingResponse{
		Routing: toProtoOrganization(next),
	}, nil
}

func (s *Server) GetProjectModelRouting(
	ctx context.Context,
	req *agencyv1.GetProjectModelRoutingRequest,
) (*agencyv1.GetProjectModelRoutingResponse, error) {
	if s.Projects != nil {
		if _, err := s.Projects.Get(ctx, req.GetProjectId()); err != nil {
			return nil, mapError(err)
		}
	}
	cur, err := s.Routing.GetProject(ctx, req.GetProjectId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.GetProjectModelRoutingResponse{
		Routing: toProtoProject(cur),
	}, nil
}

func (s *Server) UpdateProjectModelRouting(
	ctx context.Context,
	req *agencyv1.UpdateProjectModelRoutingRequest,
) (*agencyv1.UpdateProjectModelRoutingResponse, error) {
	if err := s.requireOwner(ctx, req.GetProjectId()); err != nil {
		return nil, mapError(err)
	}
	before, err := s.Routing.GetProject(ctx, req.GetProjectId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	next, err := s.Routing.UpdateProject(
		ctx,
		req.GetProjectId(),
		fromProtoProject(req.GetRouting()),
	)
	if err != nil {
		return nil, mapError(err)
	}
	if s.Projects != nil {
		s.Projects.LogSettingsChange(
			ctx, req.GetProjectId(), "model_routing",
			routingLabel(before), routingLabel(next),
			"Changed the project model routing", nil,
		)
	}
	return &agencyv1.UpdateProjectModelRoutingResponse{
		Routing: toProtoProject(next),
	}, nil
}

func (s *Server) PreviewModelRouting(
	ctx context.Context,
	req *agencyv1.PreviewModelRoutingRequest,
) (*agencyv1.PreviewModelRoutingResponse, error) {
	match, ok, err := s.Routing.ResolveMatch(ctx, modelrouting.ResolveInput{
		ProjectID: req.GetProjectId(),
		Priority:  int64(req.GetPriority()),
		Size:      sizeFromProto(req.GetSize()),
		Tags:      req.GetTags(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return &agencyv1.PreviewModelRoutingResponse{}, nil
	}
	return &agencyv1.PreviewModelRoutingResponse{
		Model: toProtoSelection(match.Selection),
		Scope: match.Scope,
		Rule:  int32(match.Index),
	}, nil
}

func (s *Server) ListRoutingModels(
	ctx context.Context,
	req *agencyv1.ListRoutingModelsRequest,
) (*agencyv1.ListRoutingModelsResponse, error) {
	if s.Catalog == nil {
		return &agencyv1.ListRoutingModelsResponse{}, nil
	}
	admin, err := s.Catalog.Admin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	seen := map[string]bool{}
	out := make([]*agencyv1.RoutingModel, 0)
	for _, provider := range admin.Providers {
		if !provider.Enabled {
			continue
		}
		for _, model := range provider.Models {
			if !model.Enabled {
				continue
			}
			if !routingGranted(provider.Grantees, model.ID, req.GetProjectId()) {
				continue
			}
			key := provider.Slug + "/" + model.ID
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, &agencyv1.RoutingModel{
				Provider: provider.Slug,
				ModelId:  model.ID,
				Name:     model.Name,
				ThinkingLevels: agentconfig.PickerThinkingLevels(
					model.Reasoning,
					model.ThinkingLevels,
				),
			})
		}
	}
	return &agencyv1.ListRoutingModelsResponse{Models: out}, nil
}

func routingGranted(grants []agentconfig.Scope, modelID, projectID string) bool {
	for _, grant := range grants {
		if grant.ModelID != "" && grant.ModelID != modelID {
			continue
		}
		if grant.Kind == agentconfig.ScopeOrg {
			return true
		}
		if projectID != "" && grant.Kind == agentconfig.ScopeProject &&
			grant.ID == projectID {
			return true
		}
	}
	return false
}

func (s *Server) requireOwner(ctx context.Context, projectID string) error {
	if s.Projects == nil {
		return nil
	}
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil
	}
	_, err := s.Projects.RequireRole(
		ctx, projectID, identity.ActorID, identity.IsAdmin(), projects.RoleOwner,
	)
	return err
}

func mapError(err error) error {
	switch {
	case errors.Is(err, projects.ErrProjectMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, projects.ErrForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, modelrouting.ErrInvalidSelection),
		errors.Is(err, modelrouting.ErrInvalidRoute),
		errors.Is(err, modelrouting.ErrInvalidTagPin):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func toProtoOrganization(
	in modelrouting.OrganizationModelRouting,
) *agencyv1.OrganizationModelRouting {
	return &agencyv1.OrganizationModelRouting{
		DefaultModel: toProtoSelection(in.Default),
		Routes:       toProtoRoutes(in.Routes),
	}
}

func toProtoProject(in modelrouting.ProjectModelRouting) *agencyv1.ProjectModelRouting {
	out := &agencyv1.ProjectModelRouting{
		TagPins: toProtoRoutes(in.TagPins),
		Routes:  toProtoRoutes(in.Routes),
	}
	if in.Default != nil {
		out.DefaultModel = toProtoSelection(*in.Default)
	}
	return out
}

func toProtoRoutes(in []modelrouting.Route) []*agencyv1.ModelRoute {
	out := make([]*agencyv1.ModelRoute, len(in))
	for i, route := range in {
		out[i] = &agencyv1.ModelRoute{
			Priority: agencyv1.TaskPriority(route.Priority),
			Size:     sizeToProto(route.Size),
			Tag:      route.Tag,
			Model:    toProtoSelection(route.Model),
		}
	}
	return out
}

func toProtoSelection(in modelrouting.Selection) *agencyv1.ModelSelection {
	return &agencyv1.ModelSelection{
		Provider:      in.Provider,
		ModelId:       in.ModelID,
		ModelName:     in.ModelName,
		ThinkingLevel: in.ThinkingLevel,
	}
}

func fromProtoOrganization(
	in *agencyv1.OrganizationModelRouting,
) modelrouting.OrganizationModelRouting {
	return modelrouting.OrganizationModelRouting{
		Default: fromProtoSelection(in.GetDefaultModel()),
		Routes:  fromProtoRoutes(in.GetRoutes()),
	}
}

func fromProtoProject(in *agencyv1.ProjectModelRouting) modelrouting.ProjectModelRouting {
	out := modelrouting.ProjectModelRouting{
		TagPins: fromProtoRoutes(in.GetTagPins()),
		Routes:  fromProtoRoutes(in.GetRoutes()),
	}
	if in.GetDefaultModel() != nil {
		selection := fromProtoSelection(in.GetDefaultModel())
		out.Default = &selection
	}
	return out
}

func fromProtoRoutes(in []*agencyv1.ModelRoute) []modelrouting.Route {
	out := make([]modelrouting.Route, len(in))
	for i, route := range in {
		out[i] = modelrouting.Route{
			Priority: int64(route.GetPriority()),
			Size:     sizeFromProto(route.GetSize()),
			Tag:      route.GetTag(),
			Model:    fromProtoSelection(route.GetModel()),
		}
	}
	return out
}

func fromProtoSelection(in *agencyv1.ModelSelection) modelrouting.Selection {
	if in == nil {
		return modelrouting.Selection{}
	}
	return modelrouting.Selection{
		Provider:      in.GetProvider(),
		ModelID:       in.GetModelId(),
		ModelName:     in.GetModelName(),
		ThinkingLevel: in.GetThinkingLevel(),
	}
}

func sizeToProto(size string) agencyv1.TaskSize {
	switch size {
	case "small":
		return agencyv1.TaskSize_TASK_SIZE_SMALL
	case "medium":
		return agencyv1.TaskSize_TASK_SIZE_MEDIUM
	case "big":
		return agencyv1.TaskSize_TASK_SIZE_BIG
	case "epic":
		return agencyv1.TaskSize_TASK_SIZE_EPIC
	default:
		return agencyv1.TaskSize_TASK_SIZE_UNSPECIFIED
	}
}

func sizeFromProto(size agencyv1.TaskSize) string {
	switch size {
	case agencyv1.TaskSize_TASK_SIZE_SMALL:
		return "small"
	case agencyv1.TaskSize_TASK_SIZE_MEDIUM:
		return "medium"
	case agencyv1.TaskSize_TASK_SIZE_BIG:
		return "big"
	case agencyv1.TaskSize_TASK_SIZE_EPIC:
		return "epic"
	default:
		return ""
	}
}

func routingLabel(in modelrouting.ProjectModelRouting) string {
	parts := make([]string, 0, 3)
	if in.Default != nil {
		parts = append(parts, "default "+in.Default.Provider+"/"+in.Default.ModelID)
	}
	if len(in.TagPins) > 0 {
		parts = append(parts, "tag pins")
	}
	if len(in.Routes) > 0 {
		parts = append(parts, "routes")
	}
	if len(parts) == 0 {
		return "inherit"
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += ", " + part
	}
	return out
}
