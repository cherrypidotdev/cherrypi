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

package terminalgrpc

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"errors"
	"reflect"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	agentv1connect "cherrypi.vladfaust.com/shared/gen/buf/agent/v1/agentv1connect"
	"connectrpc.com/connect"
)

// TerminalServer implements agency.v1.TerminalService by forwarding control
// calls to the worker; the PTY data plane is relayed separately (see Hub's WS
// handler).
type TerminalServer struct {
	Hub *tunnel.Hub
	// Agents decides whether the caller owns the agent. Terminals are
	// owner-only (administrators always pass).
	Agents *agents.Registry
}

func (s *TerminalServer) client(
	ctx context.Context,
	agentID ids.AgentID,
) (agentv1connect.TerminalServiceClient, error) {
	if err := s.requireOwner(ctx, agentID); err != nil {
		return nil, err
	}
	conn, err := s.Hub.Connection(agentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return conn.Terminals, nil
}

// requireOwner gates every terminal call on agent ownership. A non-owner must
// not be able to list, open, resize, or close the agent's shells.
func (s *TerminalServer) requireOwner(ctx context.Context, agentID ids.AgentID) error {
	if auth.IsAdmin(ctx) {
		return nil
	}
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("not authenticated"))
	}
	if s.Agents == nil {
		return connect.NewError(connect.CodePermissionDenied, errors.New("agent owner required"))
	}
	owner, err := s.Agents.IsOwner(ctx, agentID, identity.ActorID)
	if err != nil {
		if errors.Is(err, agents.ErrAgentMissing) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	if !owner {
		return connect.NewError(
			connect.CodePermissionDenied,
			errors.New("agent owner or administrator required"),
		)
	}
	return nil
}

func (s *TerminalServer) CreateTerminal(
	ctx context.Context,
	req *agencyv1.CreateTerminalRequest,
) (*agencyv1.CreateTerminalResponse, error) {
	c, err := s.client(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, err
	}
	res, err := c.CreateTerminal(ctx, valueOr(req.Request, &agentv1.CreateTerminalRequest{}))
	if err != nil {
		return nil, err
	}
	return &agencyv1.CreateTerminalResponse{
		Terminal: res.Terminal,
		WsPath:   WSPath(req.AgentId, res.Terminal.Id),
	}, nil
}

func (s *TerminalServer) ListTerminals(
	ctx context.Context,
	req *agencyv1.ListTerminalsRequest,
) (*agencyv1.ListTerminalsResponse, error) {
	c, err := s.client(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, err
	}
	res, err := c.ListTerminals(ctx, &agentv1.ListTerminalsRequest{})
	if err != nil {
		return nil, err
	}
	return &agencyv1.ListTerminalsResponse{Terminals: res.Terminals}, nil
}

func (s *TerminalServer) ResizeTerminal(
	ctx context.Context,
	req *agencyv1.ResizeTerminalRequest,
) (*agencyv1.ResizeTerminalResponse, error) {
	c, err := s.client(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, err
	}
	if _, err := c.ResizeTerminal(
		ctx,
		valueOr(req.Request, &agentv1.ResizeTerminalRequest{}),
	); err != nil {
		return nil, err
	}
	return &agencyv1.ResizeTerminalResponse{}, nil
}

func (s *TerminalServer) CloseTerminal(
	ctx context.Context,
	req *agencyv1.CloseTerminalRequest,
) (*agencyv1.CloseTerminalResponse, error) {
	c, err := s.client(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, err
	}
	if _, err := c.CloseTerminal(
		ctx,
		valueOr(req.Request, &agentv1.CloseTerminalRequest{}),
	); err != nil {
		return nil, err
	}
	return &agencyv1.CloseTerminalResponse{}, nil
}

func (s *TerminalServer) CloseAllTerminals(
	ctx context.Context,
	req *agencyv1.CloseAllTerminalsRequest,
) (*agencyv1.CloseAllTerminalsResponse, error) {
	c, err := s.client(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, err
	}
	res, err := c.CloseAllTerminals(ctx, valueOr(req.Request, &agentv1.CloseAllTerminalsRequest{}))
	if err != nil {
		return nil, err
	}
	return &agencyv1.CloseAllTerminalsResponse{Closed: res.Closed}, nil
}

func (s *TerminalServer) GetTerminalStats(
	ctx context.Context,
	req *agencyv1.GetTerminalStatsRequest,
) (*agencyv1.GetTerminalStatsResponse, error) {
	c, err := s.client(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, err
	}
	res, err := c.GetTerminalStats(ctx, &agentv1.GetTerminalStatsRequest{})
	if err != nil {
		return nil, err
	}
	return &agencyv1.GetTerminalStatsResponse{Stats: res.Stats}, nil
}

// ListTerminalLogs forwards the agent's persisted terminal lifecycle log.
func (s *TerminalServer) ListTerminalLogs(
	ctx context.Context,
	req *agencyv1.ListTerminalLogsRequest,
) (*agencyv1.ListTerminalLogsResponse, error) {
	c, err := s.client(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, err
	}
	res, err := c.ListTerminalLogs(
		ctx,
		valueOr(req.Request, &agentv1.ListTerminalLogsRequest{}),
	)
	if err != nil {
		return nil, err
	}
	return &agencyv1.ListTerminalLogsResponse{Logs: res.Logs}, nil
}

// WSPath is the agency-side terminal WebSocket path for an agent's terminal.
func WSPath(agentID, terminalID string) string {
	return "/api/agents/" + agentID + "/terminal/" + terminalID + "/ws"
}

func valueOr[T any](v T, fallback T) T {
	if reflect.ValueOf(v).IsZero() {
		return fallback
	}
	return v
}
