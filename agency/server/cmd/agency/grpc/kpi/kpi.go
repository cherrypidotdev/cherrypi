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

package kpigrpc

import (
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/kpilayout"
	"cherrypi.vladfaust.com/agency/internal/projects"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
)

type Server struct {
	Layouts *kpilayout.Service

	Projects *projects.Service

	Agents *agents.Registry
}

func (s *Server) GetKpiLayout(
	ctx context.Context,
	req *agencyv1.GetKpiLayoutRequest,
) (*agencyv1.GetKpiLayoutResponse, error) {
	page := fromProtoPage(req.GetKind(), req.GetId())
	if err := s.requireRead(ctx, page); err != nil {
		return nil, err
	}
	return s.respond(ctx, page)
}

func (s *Server) UpdateKpiLayout(
	ctx context.Context,
	req *agencyv1.UpdateKpiLayoutRequest,
) (*agencyv1.UpdateKpiLayoutResponse, error) {
	page := fromProtoPage(req.GetKind(), req.GetId())
	if err := s.requireEdit(ctx, page); err != nil {
		return nil, err
	}
	if _, err := s.Layouts.SetPage(ctx, page, fromProtoLayout(req.GetLayout())); err != nil {
		return nil, mapError(err)
	}
	current, err := s.respond(ctx, page)
	if err != nil {
		return nil, err
	}
	return &agencyv1.UpdateKpiLayoutResponse{
		Layout:    current.Layout,
		Inherited: current.Inherited,
		CanEdit:   current.CanEdit,
		Fallback:  current.Fallback,
	}, nil
}

func (s *Server) ResetKpiLayout(
	ctx context.Context,
	req *agencyv1.ResetKpiLayoutRequest,
) (*agencyv1.ResetKpiLayoutResponse, error) {
	page := fromProtoPage(req.GetKind(), req.GetId())
	if err := s.requireEdit(ctx, page); err != nil {
		return nil, err
	}
	if err := s.Layouts.ResetPage(ctx, page); err != nil {
		return nil, mapError(err)
	}
	current, err := s.respond(ctx, page)
	if err != nil {
		return nil, err
	}
	return &agencyv1.ResetKpiLayoutResponse{
		Layout:    current.Layout,
		Inherited: current.Inherited,
		CanEdit:   current.CanEdit,
		Fallback:  current.Fallback,
	}, nil
}

func (s *Server) GetKpiLayoutDefaults(
	ctx context.Context,
	_ *agencyv1.GetKpiLayoutDefaultsRequest,
) (*agencyv1.GetKpiLayoutDefaultsResponse, error) {
	project, agent, err := s.Layouts.Defaults(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.GetKpiLayoutDefaultsResponse{
		Project: toProtoLayout(project),
		Agent:   toProtoLayout(agent),
		CanEdit: auth.IsAdmin(ctx),
	}, nil
}

func (s *Server) UpdateKpiLayoutDefaults(
	ctx context.Context,
	req *agencyv1.UpdateKpiLayoutDefaultsRequest,
) (*agencyv1.UpdateKpiLayoutDefaultsResponse, error) {

	if !auth.IsAdmin(ctx) {
		return nil, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("only administrators may change the default KPI layouts"),
		)
	}
	project, agent, err := s.Layouts.SetDefaults(
		ctx,
		fromProtoLayout(req.GetProject()),
		fromProtoLayout(req.GetAgent()),
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateKpiLayoutDefaultsResponse{
		Project: toProtoLayout(project),
		Agent:   toProtoLayout(agent),
		CanEdit: true,
	}, nil
}

func (s *Server) respond(
	ctx context.Context,
	page kpilayout.Page,
) (*agencyv1.GetKpiLayoutResponse, error) {
	layout, inherited, err := s.Layouts.Page(ctx, page)
	if err != nil {
		return nil, mapError(err)
	}
	canEdit, err := s.canEdit(ctx, page)
	if err != nil {
		return nil, err
	}
	out := &agencyv1.GetKpiLayoutResponse{
		Layout:    toProtoLayout(layout),
		Inherited: inherited,
		CanEdit:   canEdit,
	}
	if page.Kind != kpilayout.KindAgency {
		fallback, err := s.Layouts.Fallback(ctx, page.Kind)
		if err != nil {
			return nil, mapError(err)
		}
		out.Fallback = toProtoLayout(fallback)
	}
	return out, nil
}

func (s *Server) requireRead(ctx context.Context, page kpilayout.Page) error {
	if err := page.Validate(); err != nil {
		return mapError(err)
	}
	identity, ok := auth.IdentityFrom(ctx)
	if !ok || identity.IsAdmin() {
		return s.requirePageExists(ctx, page)
	}
	switch page.Kind {
	case kpilayout.KindProject:
		if _, err := s.Projects.RequireRole(
			ctx, page.ID, identity.ActorID, false, projects.RoleViewer,
		); err != nil {
			return mapError(err)
		}
	}
	return s.requirePageExists(ctx, page)
}

func (s *Server) requirePageExists(ctx context.Context, page kpilayout.Page) error {
	switch page.Kind {
	case kpilayout.KindProject:
		if _, err := s.Projects.Get(ctx, page.ID); err != nil {
			return mapError(err)
		}
	case kpilayout.KindAgent:
		if _, err := s.Agents.Get(ctx, actorid.ParseAgent(page.ID)); err != nil {
			return mapError(err)
		}
	}
	return nil
}

func (s *Server) requireEdit(ctx context.Context, page kpilayout.Page) error {
	if err := page.Validate(); err != nil {
		return mapError(err)
	}
	if err := s.requirePageExists(ctx, page); err != nil {
		return err
	}
	canEdit, err := s.canEdit(ctx, page)
	if err != nil {
		return err
	}
	if !canEdit {
		return connect.NewError(
			connect.CodePermissionDenied,
			errors.New("only an administrator or the page's owner may change its KPI layout"),
		)
	}
	return nil
}

func (s *Server) canEdit(ctx context.Context, page kpilayout.Page) (bool, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return true, nil
	}
	if identity.IsAdmin() {
		return true, nil
	}
	switch page.Kind {
	case kpilayout.KindAgency:
		return false, nil
	case kpilayout.KindProject:
		role, member, err := s.Projects.EffectiveRole(
			ctx, page.ID, identity.ActorID, false,
		)
		if err != nil {
			return false, mapError(err)
		}
		return member && role == projects.RoleOwner, nil
	case kpilayout.KindAgent:
		owner, err := s.Agents.IsOwner(
			ctx, actorid.ParseAgent(page.ID), identity.ActorID,
		)
		if err != nil {
			return false, mapError(err)
		}
		return owner, nil
	default:
		return false, nil
	}
}

func fromProtoPage(kind agencyv1.KpiPageKind, id string) kpilayout.Page {
	page := kpilayout.Page{ID: id}
	switch kind {
	case agencyv1.KpiPageKind_KPI_PAGE_KIND_AGENCY:
		page.Kind = kpilayout.KindAgency
		page.ID = ""
	case agencyv1.KpiPageKind_KPI_PAGE_KIND_PROJECT:
		page.Kind = kpilayout.KindProject
	case agencyv1.KpiPageKind_KPI_PAGE_KIND_AGENT:
		page.Kind = kpilayout.KindAgent
	}
	return page
}

func fromProtoLayout(in *agencyv1.KpiLayout) kpilayout.Layout {
	if in == nil {
		return kpilayout.Layout{}
	}
	out := kpilayout.Layout{Widgets: make([]kpilayout.Widget, len(in.GetWidgets()))}
	for i, widget := range in.GetWidgets() {
		out.Widgets[i] = kpilayout.Widget{
			ID:     widget.GetId(),
			Kind:   widget.GetKind(),
			Width:  int(widget.GetWidth()),
			Height: int(widget.GetHeight()),
			Range:  widget.GetRange(),
			Scope:  widget.GetScope(),
		}
	}
	return out
}

func toProtoLayout(in kpilayout.Layout) *agencyv1.KpiLayout {
	out := &agencyv1.KpiLayout{Widgets: make([]*agencyv1.KpiWidget, len(in.Widgets))}
	for i, widget := range in.Widgets {
		out.Widgets[i] = &agencyv1.KpiWidget{
			Id:     widget.ID,
			Kind:   widget.Kind,
			Width:  int32(widget.Width),
			Height: int32(widget.Height),
			Range:  widget.Range,
			Scope:  widget.Scope,
		}
	}
	return out
}

func mapError(err error) *connect.Error {
	switch {
	case errors.Is(err, kpilayout.ErrInvalidPage):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, kpilayout.ErrInvalidLayout):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, projects.ErrProjectMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, agents.ErrAgentMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, projects.ErrForbidden),
		errors.Is(err, projects.ErrNotMember):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
