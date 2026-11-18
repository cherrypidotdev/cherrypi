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

package agencygrpc

import (
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/processlog"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
)

type Server struct {
	Agents *agents.Registry

	ProcessLogs *processlog.Log
}

func (s *Server) GetMetadata(
	ctx context.Context,
	_ *agencyv1.GetAgencyMetadataRequest,
) (*agencyv1.GetAgencyMetadataResponse, error) {
	caller, err := s.adminAgent(ctx)
	if err != nil {
		return nil, err
	}
	if s.ProcessLogs == nil {
		return nil, connect.NewError(connect.CodeUnavailable,
			errors.New("the agency is not writing a process log"))
	}
	return &agencyv1.GetAgencyMetadataResponse{
		Agency: processMeta(s.ProcessLogs.Meta()),
		Caller: agentMeta(caller),
	}, nil
}

func (s *Server) ReadLogs(
	ctx context.Context,
	req *agencyv1.ReadAgencyLogsRequest,
) (*agencyv1.ReadAgencyLogsResponse, error) {
	if _, err := s.adminAgent(ctx); err != nil {
		return nil, err
	}
	if s.ProcessLogs == nil {
		return nil, connect.NewError(connect.CodeUnavailable,
			errors.New("the agency is not writing a process log"))
	}
	lines := 200
	if req.Lines > 0 {
		lines = int(req.Lines)
	}
	if lines > 2000 {
		lines = 2000
	}
	rows, offset, err := s.ProcessLogs.Tail(lines)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.ReadAgencyLogsResponse{
		Meta:   processMeta(s.ProcessLogs.Meta()),
		Lines:  rows,
		Offset: offset,
	}, nil
}

func (s *Server) ListAgents(
	ctx context.Context,
	_ *agencyv1.ListAgencyAgentsRequest,
) (*agencyv1.ListAgencyAgentsResponse, error) {
	if _, err := s.adminAgent(ctx); err != nil {
		return nil, err
	}
	list, err := s.Agents.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*agencyv1.AgencyAgent, len(list))
	for i, a := range list {
		out[i] = agentMeta(a)
	}
	return &agencyv1.ListAgencyAgentsResponse{Agents: out}, nil
}

func (s *Server) adminAgent(ctx context.Context) (agents.Agent, error) {
	if s.Agents == nil {
		return agents.Agent{}, connect.NewError(connect.CodeUnavailable,
			errors.New("the agent registry is unavailable"))
	}
	agentID, ok := auth.AgentID(ctx)
	if !ok {
		return agents.Agent{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("agent authentication is required"))
	}
	caller, err := s.Agents.Get(ctx, agentID)
	if err != nil {
		return agents.Agent{}, connect.NewError(connect.CodeNotFound,
			errors.New("agent not found"))
	}
	if caller.Role != agents.RoleAdmin {
		return agents.Agent{}, connect.NewError(connect.CodePermissionDenied,
			errors.New("the admin agent role is required"))
	}
	return caller, nil
}

func processMeta(meta processlog.Meta) *agencyv1.AgencyProcess {
	return &agencyv1.AgencyProcess{
		Version:   meta.Version,
		File:      meta.File,
		Dir:       meta.Dir,
		StartedAt: meta.StartedAt,
		Pid:       int32(meta.PID),
		UptimeMs:  meta.UptimeMs,
		Keep:      int32(meta.Keep),
	}
}

func agentMeta(a agents.Agent) *agencyv1.AgencyAgent {
	out := &agencyv1.AgencyAgent{
		Id:              actorid.Format(a.ID),
		ActorId:         actorid.Format(a.ActorID),
		Username:        a.Username,
		DisplayName:     a.DisplayName,
		Description:     a.Description,
		Role:            a.Role,
		Version:         a.Version,
		ProtocolVersion: a.ProtocolVersion,
		RemoteAddr:      a.RemoteAddr,
	}
	if a.Status == agents.StatusOnline {
		out.Status = "online"
	} else {
		out.Status = "offline"
	}
	if !a.LastSeenAt.IsZero() {
		out.LastSeenAt = a.LastSeenAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return out
}
