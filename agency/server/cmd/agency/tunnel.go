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
	"log"
	"net/http"
	"strings"

	"cherrypi.vladfaust.com/agency/cmd/agency/appcontext"
	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	tunnelv1 "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1"
	sharedprotocol "cherrypi.vladfaust.com/shared/protocol"
	sharedtunnel "cherrypi.vladfaust.com/shared/tunnel"
	"connectrpc.com/connect"
	"github.com/coder/websocket"
)

type tunnelServer struct {
	appCtx *appcontext.AppContext
}

func (t *tunnelServer) Connect(
	ctx context.Context,
	stream *connect.BidiStream[tunnelv1.Frame, tunnelv1.Frame],
) error {
	agentID, err := t.authenticate(ctx, stream.RequestHeader().Get("Authorization"))
	if err != nil {
		return err
	}

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
	if err := t.checkHello(agentID, hello, remoteAddr); err != nil {
		return err
	}

	err = t.appCtx.Hub.Serve(ctx, agentID, hello, remoteAddr, stream)
	if errors.Is(err, tunnel.ErrAlreadyConnected) {
		return connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf(
				"agent %s already has a live tunnel; refusing a second connection from %s",
				t.agentUsername(ctx, agentID),
				remoteAddr,
			),
		)
	}
	return err
}

func (t *tunnelServer) agentUsername(ctx context.Context, agentID ids.AgentID) string {
	agent, err := t.appCtx.Agents.Get(ctx, agentID)
	if err != nil {
		return fmt.Sprint(agentID)
	}
	return agent.Username
}

func (t *tunnelServer) WS(w http.ResponseWriter, r *http.Request) {
	agentID, err := t.authenticate(r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		log.Printf("[tunnel] websocket auth from %s failed: %v", r.RemoteAddr, err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	hello := &tunnelv1.Hello{
		BuildVersion:    r.Header.Get(sharedtunnel.HelloBuildVersionHeader),
		ProtocolVersion: r.Header.Get(sharedtunnel.HelloProtocolVersionHeader),
	}
	remoteAddr := r.RemoteAddr
	if err := t.checkHello(agentID, hello, remoteAddr); err != nil {
		http.Error(w, err.Error(), http.StatusUpgradeRequired)
		return
	}

	if _, err := t.appCtx.Hub.Connection(agentID); err == nil {
		log.Printf(
			"[tunnel] refusing a second connection for agent %s from %s",
			t.agentUsername(r.Context(), agentID),
			remoteAddr,
		)
		http.Error(w, tunnel.ErrAlreadyConnected.Error(), http.StatusConflict)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{

		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()

	pipe := sharedtunnel.NewWSConn(conn)
	defer func() { _ = pipe.Close() }()

	if err := t.appCtx.Hub.ServeConn(r.Context(), agentID, hello, remoteAddr, pipe); err != nil {
		log.Printf("[tunnel] agent %s: %v", t.agentUsername(r.Context(), agentID), err)
	}
}

func (t *tunnelServer) authenticate(
	ctx context.Context,
	header string,
) (ids.AgentID, error) {

	if token, ok := bearerAuth(header); ok {
		identity, _, err := t.appCtx.Accounts.ResolvePAT(ctx, token)
		if err == nil && identity.Kind == accounts.KindAgent {
			agent, err := t.appCtx.Agents.GetByActor(ctx, identity.ActorID)
			if err == nil {
				return agent.ID, nil
			}
		}
		return 0, connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("invalid tunnel token"),
		)
	}

	user, pass, ok := basicAuth(header)
	if !ok {
		return 0, connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("missing tunnel credentials"),
		)
	}
	agentID, ok := t.appCtx.Agents.Authenticate(ctx, user, pass)
	if !ok {
		return 0, connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("invalid tunnel credentials"),
		)
	}
	return agentID, nil
}

// version, or one on a different compatibility line (the minor while pre-1.0,

func (t *tunnelServer) checkHello(
	agentID ids.AgentID,
	hello *tunnelv1.Hello,
	remoteAddr string,
) error {
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
		t.appCtx.Realtime.Publish(agencyv1.RealtimeKind_REALTIME_KIND_AGENTS)
		return connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf(
				"agent protocol %q is incompatible with %q; rebuild the agent",
				hello.GetProtocolVersion(),
				sharedprotocol.Version,
			),
		)
	}
	return nil
}

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
