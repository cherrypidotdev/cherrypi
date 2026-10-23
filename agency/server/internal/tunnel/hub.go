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

// Package tunnel is the agency's tunnel hub: it tracks connected agents and
// gives each one typed Connect clients that reach the agent's API over its
// yamux session. Presence flows from here into the agent registry.
package tunnel

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"errors"
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

// ErrOffline is returned when an agent has no live tunnel.
var ErrOffline = errors.New("agent is offline")

// Hub owns the live agent connections.
type Hub struct {
	agents *agents.Registry

	mu    sync.RWMutex
	conns map[ids.AgentID]*Connection

	// OnConnect, if set, is called with each new connection (from Serve).
	// Callbacks must not block.
	OnConnect func(ctx context.Context, conn *Connection)

	// OnDisconnect, if set, is called after a connection's tunnel is torn
	// down. Callbacks must not block.
	OnDisconnect func(agentID ids.AgentID)
}

// Connection is one live agent tunnel plus typed clients to its API.
type Connection struct {
	ID      ids.AgentID
	session *yamux.Session

	// cancel stops the work started for this connection (the event consumer).
	// It is called when the connection is torn down or superseded.
	cancel context.CancelFunc

	httpClient *http.Client

	Tasks     agentv1connect.TaskServiceClient
	Dashboard agentv1connect.DashboardServiceClient
	Terminals agentv1connect.TerminalServiceClient
	Desktops  agentv1connect.DesktopServiceClient
}

// HTTPClient returns the client that dials the agent over the tunnel (used to
// relay WebSocket upgrades).
func (c *Connection) HTTPClient() *http.Client { return c.httpClient }

// NewHub builds a hub over the agent registry.
func NewHub(reg *agents.Registry) *Hub {
	return &Hub{agents: reg, conns: map[ids.AgentID]*Connection{}}
}

// Serve runs for the lifetime of one agent tunnel stream. hello is the agent's
// validated opening handshake; remoteAddr is the socket address the connection
// arrived from, surfaced as agent presence. It blocks until the stream's
// context is done, then tears the connection down.
func (h *Hub) Serve(
	ctx context.Context,
	agentID ids.AgentID, hello *tunnelv1.Hello, remoteAddr string,
	stream *connect.BidiStream[tunnelv1.Frame, tunnelv1.Frame],
) error {
	session, err := yamux.Client(sharedtunnel.ServerConn(stream), yamux.DefaultConfig())
	if err != nil {
		return err
	}

	connCtx, cancel := context.WithCancel(ctx)
	conn := newConnection(agentID, session)
	conn.cancel = cancel

	// Publish presence and the connection under one lock. A superseded
	// connection's teardown must never report this one offline, so both
	// transitions are ordered here rather than racing in separate steps.
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

	// Stop the tunnel this connection replaces. Its stream may linger (a proxy
	// can hold it open after a reconnect), but it no longer owns presence and
	// must not keep consuming the agent's events.
	if previous != nil && previous != conn {
		previous.cancel()
		_ = previous.session.Close()
	}

	if h.OnConnect != nil {
		h.OnConnect(connCtx, conn)
	}

	defer func() {
		cancel()

		// Only the connection still registered in the hub owns presence and the
		// disconnect event. A superseded connection tears down quietly.
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

// Connection returns the live connection for an agent, or ErrOffline.
func (h *Hub) Connection(agentID ids.AgentID) (*Connection, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conn, ok := h.conns[agentID]
	if !ok {
		return nil, ErrOffline
	}
	return conn, nil
}

// Agents returns the ids of every agent with a live tunnel, sorted. Callers
// that need per-agent presence should still confirm via Connection, since a
// connection can drop between the two calls.
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
		// Every dial opens a fresh yamux stream; the host is irrelevant.
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

// userForward carries the authenticated agency user to the agent as
// `X-CherryPi-User`, so the worker attributes terminals and per-user work to
// the right user. The agent's tunnel server turns it back into request
// context.
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
