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
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"connectrpc.com/connect"
)

// DesktopServer implements agency.v1.DesktopService by forwarding GetDesktop
// to the worker; the RFB data plane is relayed separately (see Hub's WS
// handler).
type DesktopServer struct {
	Hub *tunnel.Hub
}

// GetDesktop forwards the availability/geometry probe to the agent.
func (s *DesktopServer) GetDesktop(
	ctx context.Context,
	req *agencyv1.GetDesktopRequest,
) (*agencyv1.GetDesktopResponse, error) {
	conn, err := s.Hub.Connection(actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	res, err := conn.Desktops.GetDesktop(ctx, &agentv1.GetDesktopRequest{})
	if err != nil {
		return nil, err
	}
	out := &agencyv1.GetDesktopResponse{
		Available:         res.Available,
		UnavailableReason: res.UnavailableReason,
		Desktop:           res.Desktop,
	}
	if res.WsPath != "" {
		out.WsPath = DesktopWSPath(actorid.ParseAgent(req.AgentId))
	}
	return out, nil
}

// DesktopWSPath is the agency-side desktop WebSocket path for an agent.
func DesktopWSPath(agentID ids.AgentID) string {
	return "/api/agents/" + actorid.Format(agentID) + "/desktop/ws"
}
