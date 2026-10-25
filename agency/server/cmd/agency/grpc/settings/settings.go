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

package settingsgrpc

import (
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/settings"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
)

type SettingsServer struct {
	Settings *settings.Service

	Attachments *attachments.Service
}

func (s *SettingsServer) GetSettings(
	ctx context.Context,
	_ *agencyv1.GetSettingsRequest,
) (*agencyv1.GetSettingsResponse, error) {
	cur, err := s.Settings.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.GetSettingsResponse{Settings: s.toProto(ctx, cur)}, nil
}

func (s *SettingsServer) UpdateSettings(
	ctx context.Context,
	req *agencyv1.UpdateSettingsRequest,
) (*agencyv1.UpdateSettingsResponse, error) {
	in := req.Settings
	if in == nil {
		in = &agencyv1.Settings{}
	}
	updated, err := s.Settings.Update(ctx, settings.Settings{
		AgencyName:   in.AgencyName,
		PrimaryColor: in.PrimaryColor,
	})
	if err != nil {
		if errors.Is(err, settings.ErrInvalidColor) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.UpdateSettingsResponse{Settings: s.toProto(ctx, updated)}, nil
}

func (s *SettingsServer) toProto(
	ctx context.Context,
	in settings.Settings,
) *agencyv1.Settings {
	return &agencyv1.Settings{
		AgencyName:   in.AgencyName,
		PrimaryColor: in.PrimaryColor,
		Logo:         s.attachment(ctx, in.LogoAttachmentID),
		Favicon:      s.attachment(ctx, in.FaviconAttachmentID),
	}
}

func (s *SettingsServer) attachment(
	ctx context.Context,
	id int64,
) *agencyv1.Attachment {
	if id <= 0 || s.Attachments == nil {
		return nil
	}
	stored, err := s.Attachments.Get(ctx, id)
	if err != nil {
		return nil
	}
	return attachments.ToProto(stored)
}
