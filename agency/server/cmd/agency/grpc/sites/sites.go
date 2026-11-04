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

package sitesgrpc

import (
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/sites"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SiteServer struct {
	Sites       *sites.Service
	Accounts    *accounts.Service
	Attachments *attachments.Service
}

func (s *SiteServer) ListSites(
	ctx context.Context,
	req *agencyv1.ListSitesRequest,
) (*agencyv1.ListSitesResponse, error) {
	identity, _ := auth.IdentityFrom(ctx)
	list, err := s.Sites.List(ctx, identity, req.OriginTaskId, req.AgencyWideOnly)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*agencyv1.Site, len(list))
	for i, site := range list {
		out[i] = s.toProto(ctx, site, identity)
	}
	return &agencyv1.ListSitesResponse{Sites: out}, nil
}

func (s *SiteServer) GetSite(
	ctx context.Context,
	req *agencyv1.GetSiteRequest,
) (*agencyv1.GetSiteResponse, error) {
	site, err := s.Sites.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	identity, _ := auth.IdentityFrom(ctx)
	return &agencyv1.GetSiteResponse{Site: s.toProto(ctx, site, identity)}, nil
}

func (s *SiteServer) UpdateSite(
	ctx context.Context,
	req *agencyv1.UpdateSiteRequest,
) (*agencyv1.UpdateSiteResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, sites.ErrForbidden)
	}
	patch := sites.SettingsPatch{
		Title:              req.Title,
		Description:        req.Description,
		OnlyCreatorMayEdit: req.OnlyCreatorMayEdit,
		AgencyWide:         req.AgencyWide,
	}
	site, err := s.Sites.UpdateSettings(ctx, identity, req.Id, patch)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateSiteResponse{Site: s.toProto(ctx, site, identity)}, nil
}

func (s *SiteServer) DeleteSite(
	ctx context.Context,
	req *agencyv1.DeleteSiteRequest,
) (*agencyv1.DeleteSiteResponse, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, sites.ErrForbidden)
	}
	if err := s.Sites.Delete(ctx, identity, req.Id); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DeleteSiteResponse{}, nil
}

func (s *SiteServer) toProto(
	ctx context.Context,
	site sites.Site,
	identity auth.Identity,
) *agencyv1.Site {
	out := &agencyv1.Site{
		Id:                 site.ID,
		Title:              site.Title,
		Description:        site.Description,
		OnlyCreatorMayEdit: site.OnlyCreatorMayEdit,
		AgencyWide:         site.AgencyWide,
		CreatorActorId:     actorid.Format(site.CreatorActorID),
		OriginTaskId:       site.OriginTaskID,
		Url:                sites.URLFor(site.ID),
		ContentUrl:         sites.ContentURLFor(site.ID),
		CreatedAt:          timestamppb.New(site.CreatedAt),
		UpdatedAt:          timestamppb.New(site.UpdatedAt),
		CanEdit:            s.Sites.CanEdit(ctx, site, identity),
		CanDelete:          s.Sites.CanEdit(ctx, site, identity),
	}
	if s.Accounts != nil {
		if creator, err := s.Accounts.GetActor(ctx, site.CreatorActorID); err == nil {
			out.CreatorName = creator.Name
			out.CreatorUsername = creator.Username
			if creator.PictureAttachmentID > 0 {
				out.CreatorPictureUrl = attachments.URLFor(creator.PictureAttachmentID)
			}
		}
	}
	return out
}

func mapError(err error) error {
	switch {
	case errors.Is(err, sites.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, sites.ErrForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, sites.ErrInvalidID),
		errors.Is(err, sites.ErrInvalidArchive),
		errors.Is(err, sites.ErrInvalidOriginTask):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, sites.ErrTooLarge):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, sites.ErrDisabled):
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
