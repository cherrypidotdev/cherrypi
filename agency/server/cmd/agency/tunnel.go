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

package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"cherrypi.vladfaust.com/agency/cmd/agency/appcontext"
	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/ids"
	tunnelv1 "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1"
	sharedprotocol "cherrypi.vladfaust.com/shared/protocol"
	"connectrpc.com/connect"
)

// tunnelServer authenticates an agent's tunnel stream with a personal access
// token (preferred) or a legacy tunnel credential and hands it to the hub.
type tunnelServer struct {
	appCtx *appcontext.AppContext
}

func (t *tunnelServer) Connect(
	ctx context.Context,
	stream *connect.BidiStream[tunnelv1.Frame, tunnelv1.Frame],
) error {
	header := stream.RequestHeader().Get("Authorization")

	// A personal access token is the go-forward credential.
	if token, ok := bearerAuth(header); ok {
		identity, _, err := t.appCtx.Accounts.ResolvePAT(ctx, token)
		if err == nil && identity.Kind == accounts.KindAgent {
			agent, err := t.appCtx.Agents.GetByActor(ctx, identity.ActorID)
			if err == nil {
				return t.serve(ctx, agent.ID, stream)
			}
		}
		return connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("invalid tunnel token"),
		)
	}

	// The legacy credential remains a fallback during the migration.
	user, pass, ok := basicAuth(header)
	if !ok {
		return connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("missing tunnel credentials"),
		)
	}
	agentID, ok := t.appCtx.Agents.Authenticate(ctx, user, pass)
	if !ok {
		return connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("invalid tunnel credentials"),
		)
	}

	return t.serve(ctx, agentID, stream)
}

// serve reads and validates the agent's opening hello, then hands the stream
// to the hub. An agent that reports no version, or one on a different
// compatibility line (the minor while pre-1.0, else the major) than the agency,
// is refused: its binary cannot speak the current protocol.
// The refusal is recorded so the UI can show why the agent stayed offline.
func (t *tunnelServer) serve(
	ctx context.Context,
	agentID ids.AgentID,
	stream *connect.BidiStream[tunnelv1.Frame, tunnelv1.Frame],
) error {
	remoteAddr := stream.Peer().Addr

	first, err := stream.Receive()
	if err != nil {
		return connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("read tunnel hello: %w", err),
		)
	}
	hello := first.GetHello()
	if hello == nil {
		t.appCtx.Agents.Reject(agentID, "", "", remoteAddr)
		return connect.NewError(
			connect.CodeFailedPrecondition,
			errors.New("agent sent no hello frame; rebuild it with a current binary"),
		)
	}

	if sharedprotocol.Compare(
		hello.GetProtocolVersion(),
		sharedprotocol.Version,
	) == sharedprotocol.Incompatible {
		t.appCtx.Agents.Reject(
			agentID,
			hello.GetBuildVersion(),
			hello.GetProtocolVersion(),
			remoteAddr,
		)
		return connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf(
				"agent protocol %q is incompatible with %q; rebuild the agent",
				hello.GetProtocolVersion(),
				sharedprotocol.Version,
			),
		)
	}

	return t.appCtx.Hub.Serve(ctx, agentID, hello, remoteAddr, stream)
}

// bearerAuth decodes an `Authorization: Bearer …` header.
func bearerAuth(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimPrefix(header, prefix)
	if token == "" {
		return "", false
	}
	return token, true
}

// basicAuth decodes an `Authorization: Basic …` header.
func basicAuth(header string) (string, string, bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return "", "", false
	}
	user, pass, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", "", false
	}
	return user, pass, true
}
