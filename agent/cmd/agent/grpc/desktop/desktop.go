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

package desktopgrpc

import (
	"context"

	"cherrypi.vladfaust.com/agent/internal/desktop"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
)

type DesktopServer struct {
	Desktops *desktop.Manager
}

func (s *DesktopServer) GetDesktop(
	ctx context.Context,
	_ *agentv1.GetDesktopRequest,
) (*agentv1.GetDesktopResponse, error) {
	available, reason := s.Desktops.Availability(ctx)
	res := &agentv1.GetDesktopResponse{
		Available:         available,
		UnavailableReason: reason,
	}
	if info, ok := s.Desktops.Info(ctx); ok {
		res.Desktop = &agentv1.Desktop{
			Display:  info.Display,
			Width:    uint32(info.Width),
			Height:   uint32(info.Height),
			Attached: info.Attached,
		}
		res.WsPath = desktop.WebSocketPath
	}
	return res, nil
}
