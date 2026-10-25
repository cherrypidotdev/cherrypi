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

package tunnel

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sort"
	"sync"

	"cherrypi.vladfaust.com/agency/internal/agents"
	agencyauth "cherrypi.vladfaust.com/agency/internal/auth"
	agentv1connect "cherrypi.vladfaust.com/shared/gen/buf/agent/v1/agentv1connect"
	tunnelv1 "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1"
	sharedtunnel "cherrypi.vladfaust.com/shared/tunnel"
	"connectrpc.com/connect"
	"github.com/hashicorp/yamux"
)

var ErrOffline = errors.New("agent is offline")

type Hub struct {
	agents *agents.Registry

	mu    sync.RWMutex
	conns map[ids.AgentID]*Connection

	OnConnect func(ctx context.Context, conn *Connection)

	OnDisconnect func(agentID ids.AgentID)

	OnRelayDone func(agentID ids.AgentID)
}

type Connection struct {
	ID      ids.AgentID
	session *yamux.Session

	cancel context.CancelFunc

	httpClient *http.Client

	Tasks     agentv1connect.TaskServiceClient
	Dashboard agentv1connect.DashboardServiceClient
	Terminals agentv1connect.TerminalServiceClient
	Desktops  agentv1connect.DesktopServiceClient
}

func (c *Connection) HTTPClient() *http.Client { return c.httpClient }

func NewHub(reg *agents.Registry) *Hub {
	return &Hub{agents: reg, conns: map[ids.AgentID]*Connection{}}
}

func (h *Hub) Serve(
	ctx context.Context,
	agentID ids.AgentID, hello *tunnelv1.Hello, remoteAddr string,
	stream *connect.BidiStream[tunnelv1.Frame, tunnelv1.Frame],
) error {
	return h.ServeConn(
		ctx,
		agentID,
		hello,
		remoteAddr,
		sharedtunnel.ServerConn(stream),
	)
}

func (h *Hub) ServeConn(
	ctx context.Context,
	agentID ids.AgentID, hello *tunnelv1.Hello, remoteAddr string,
	pipe io.ReadWriteCloser,
) error {
	session, err := yamux.Client(pipe, yamux.DefaultConfig())
	if err != nil {
		return err
	}

	connCtx, cancel := context.WithCancel(ctx)
	conn := newConnection(agentID, session)
	conn.cancel = cancel

	h.mu.Lock()
	previous := h.conns[agentID]
	h.conns[agentID] = conn
	h.agents.SetOnline(
		agentID,
		hello.GetBuildVersion(),
		hello.GetProtocolVersion(),
		remoteAddr,
	)
	h.mu.Unlock()

	if previous != nil && previous != conn {
		previous.cancel()
		_ = previous.session.Close()
	}

	if h.OnConnect != nil {
		h.OnConnect(connCtx, conn)
	}

	defer func() {
		cancel()

		h.mu.Lock()
		current := h.conns[agentID] == conn
		if current {
			delete(h.conns, agentID)
			h.agents.SetOffline(agentID)
		}
		h.mu.Unlock()

		_ = session.Close()
		if current && h.OnDisconnect != nil {
			h.OnDisconnect(agentID)
		}
	}()

	<-connCtx.Done()
	return nil
}

func (h *Hub) Connection(agentID ids.AgentID) (*Connection, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conn, ok := h.conns[agentID]
	if !ok {
		return nil, ErrOffline
	}
	return conn, nil
}

func (h *Hub) Agents() []ids.AgentID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]ids.AgentID, 0, len(h.conns))
	for id := range h.conns {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func newConnection(id ids.AgentID, session *yamux.Session) *Connection {
	transport := &http.Transport{

		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return session.Open()
		},
		ForceAttemptHTTP2: false,
	}
	client := &http.Client{Transport: transport}
	opts := []connect.ClientOption{connect.WithInterceptors(userForward{})}

	return &Connection{
		ID:         id,
		session:    session,
		httpClient: client,
		Tasks: agentv1connect.NewTaskServiceClient(
			client,
			"http://agent/api/grpc",
			opts...),
		Dashboard: agentv1connect.NewDashboardServiceClient(
			client,
			"http://agent/api/grpc",
			opts...),
		Terminals: agentv1connect.NewTerminalServiceClient(
			client,
			"http://agent/api/grpc",
			opts...),
		Desktops: agentv1connect.NewDesktopServiceClient(
			client,
			"http://agent/api/grpc",
			opts...),
	}
}

type userForward struct{}

func (userForward) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		forwardUser(ctx, req.Header())
		return next(ctx, req)
	}
}

func (userForward) WrapStreamingClient(
	next connect.StreamingClientFunc,
) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		forwardUser(ctx, conn.RequestHeader())
		return conn
	}
}

func (userForward) WrapStreamingHandler(
	next connect.StreamingHandlerFunc,
) connect.StreamingHandlerFunc {
	return next
}

func forwardUser(ctx context.Context, header http.Header) {
	if user, ok := agencyauth.Username(ctx); ok {
		header.Set("X-CherryPi-User", user)
	}
}
