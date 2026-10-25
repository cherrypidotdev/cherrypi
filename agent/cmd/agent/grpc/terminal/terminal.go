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
	"context"
	"database/sql"
	"errors"

	"cherrypi.vladfaust.com/agent/internal/auth"
	"cherrypi.vladfaust.com/agent/internal/terminal"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TerminalServer struct {
	Terminals *terminal.Manager
}

func (s *TerminalServer) CreateTerminal(
	ctx context.Context,
	req *agentv1.CreateTerminalRequest,
) (*agentv1.CreateTerminalResponse, error) {
	owner, _ := auth.Username(ctx)
	info, err := s.Terminals.Create(
		ctx,
		terminal.Kind(req.Kind),
		uint16(req.Cols),
		uint16(req.Rows),
		owner,
	)
	if err != nil {
		return nil, connectError(err)
	}

	return &agentv1.CreateTerminalResponse{
		Terminal: toProto(info),
		WsPath:   "/api/terminal/" + info.ID + "/ws",
	}, nil
}

func (s *TerminalServer) ListTerminals(
	ctx context.Context,
	_ *agentv1.ListTerminalsRequest,
) (*agentv1.ListTerminalsResponse, error) {
	owner, _ := auth.Username(ctx)
	infos := s.Terminals.List(owner)
	out := make([]*agentv1.Terminal, len(infos))
	for i, info := range infos {
		out[i] = toProto(info)
	}
	return &agentv1.ListTerminalsResponse{Terminals: out}, nil
}

func (s *TerminalServer) ResizeTerminal(
	ctx context.Context,
	req *agentv1.ResizeTerminalRequest,
) (*agentv1.ResizeTerminalResponse, error) {
	owner, _ := auth.Username(ctx)
	if err := s.Terminals.Resize(req.Id, uint16(req.Cols), uint16(req.Rows), owner); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.ResizeTerminalResponse{}, nil
}

func (s *TerminalServer) CloseTerminal(
	ctx context.Context,
	req *agentv1.CloseTerminalRequest,
) (*agentv1.CloseTerminalResponse, error) {
	owner, _ := auth.Username(ctx)
	if err := s.Terminals.Close(req.Id, owner); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.CloseTerminalResponse{}, nil
}

func (s *TerminalServer) CloseAllTerminals(
	ctx context.Context,
	req *agentv1.CloseAllTerminalsRequest,
) (*agentv1.CloseAllTerminalsResponse, error) {
	owner, _ := auth.Username(ctx)
	closed := s.Terminals.CloseFor(owner, terminal.Kind(req.Kind))
	return &agentv1.CloseAllTerminalsResponse{Closed: uint32(closed)}, nil
}

func (s *TerminalServer) GetTerminalStats(
	ctx context.Context,
	_ *agentv1.GetTerminalStatsRequest,
) (*agentv1.GetTerminalStatsResponse, error) {
	owner, _ := auth.Username(ctx)
	stats := s.Terminals.Stats(owner)
	return &agentv1.GetTerminalStatsResponse{
		Stats: &agentv1.TerminalStats{
			Root:  uint32(stats.Root),
			Total: uint32(stats.Total),
		},
	}, nil
}

func (s *TerminalServer) ListTerminalLogs(
	ctx context.Context,
	req *agentv1.ListTerminalLogsRequest,
) (*agentv1.ListTerminalLogsResponse, error) {
	logs, err := s.Terminals.Logs(ctx, int64(req.Limit))
	if err != nil {
		return nil, connectError(err)
	}
	out := make([]*agentv1.TerminalLog, len(logs))
	for i, l := range logs {
		out[i] = &agentv1.TerminalLog{
			Id:         l.ID,
			TerminalId: l.TerminalID,
			Kind:       agentv1.TerminalKind(l.Kind),
			Owner:      l.Owner,
			Event:      l.Event,
			Reason:     l.Reason,
			Detail:     l.Detail,
			At:         timestamppb.New(l.At),
		}
	}
	return &agentv1.ListTerminalLogsResponse{Logs: out}, nil
}

func connectError(err error) error {
	switch {
	case errors.Is(err, terminal.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, terminal.ErrForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, terminal.ErrInvalidKind):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func toProto(info terminal.Info) *agentv1.Terminal {
	return &agentv1.Terminal{
		Id:        info.ID,
		Kind:      agentv1.TerminalKind(info.Kind),
		Owner:     info.Owner,
		Cwd:       info.Cwd,
		Cols:      uint32(info.Cols),
		Rows:      uint32(info.Rows),
		CreatedAt: timestamppb.New(info.CreatedAt),
	}
}
