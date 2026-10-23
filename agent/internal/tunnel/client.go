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

// Package tunnel runs the agent's side of the agency tunnel: it dials the
// agency over a Connect bidi stream, authenticates with a tunnel credential,
// and serves the agent's HTTP API over yamux on that stream. The agency reuses
// its generated Connect clients against it, so nothing about the agent's API
// changes.
package tunnel

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agent/internal/logging"
	tunnelv1 "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1"
	tunnelv1connect "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1/tunnelv1connect"
	sharedtunnel "cherrypi.vladfaust.com/shared/tunnel"
	"connectrpc.com/connect"
	"github.com/hashicorp/yamux"
)

// Config configures the tunnel client.
type Config struct {
	// AgencyURL is the agency base URL, e.g. https://cherrypi.vladfaust.com.
	AgencyURL string
	// Token returns the agent's current personal access token (`pat_…`). It is
	// re-read on every reconnect, so a rotated token file applies without an
	// agent restart. When it returns "" the legacy Username/Password is used.
	Token func() string
	// Username/Password are one of the agent's legacy tunnel credentials.
	Username string
	Password string
	// Handler is the agent's HTTP API, served over the tunnel.
	Handler http.Handler
	// BuildVersion is the agent build version (git describe), reported to the
	// agency in the tunnel hello for operators to identify the binary.
	BuildVersion string
	// ProtocolVersion is the agent↔agency protocol version (semver), reported
	// in the tunnel hello. Compatibility is judged on this, not BuildVersion.
	ProtocolVersion string
}

// Client keeps a tunnel connection to the agency alive.
type Client struct {
	cfg Config
}

// New builds a tunnel client.
func New(cfg Config) *Client { return &Client{cfg: cfg} }

// Run connects and reconnects with exponential backoff until ctx is done.
func (c *Client) Run(ctx context.Context) {
	const maxBackoff = 30 * time.Second
	backoff := time.Second

	for {
		err := c.connect(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			logging.Local().Warn(
				"tunnel disconnected",
				"agency", c.cfg.AgencyURL,
				"agent", c.cfg.Username,
				"error", err,
				"reconnect_in", backoff,
			)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (c *Client) connect(ctx context.Context) error {
	httpClient, err := sharedtunnel.NewHTTPClient(c.cfg.AgencyURL)
	if err != nil {
		return err
	}

	client := tunnelv1connect.NewTunnelServiceClient(
		httpClient,
		strings.TrimRight(c.cfg.AgencyURL, "/")+"/api/grpc",
		tunnelAuthOption(c.cfg),
	)

	stream, err := client.Connect(ctx)
	if err != nil {
		return err
	}

	// The hello must be the first frame on the stream: the agency reads it
	// before it builds the yamux pipe and refuses an incompatible binary. A
	// binary that predates the handshake sends its first yamux frame instead
	// and is refused too.
	if err := stream.Send(&tunnelv1.Frame{Hello: &tunnelv1.Hello{
		BuildVersion:    c.cfg.BuildVersion,
		ProtocolVersion: c.cfg.ProtocolVersion,
	}}); err != nil {
		return err
	}
	logging.Local().Info(
		"tunnel connected",
		"agency", c.cfg.AgencyURL,
		"agent", c.cfg.Username,
		"version", c.cfg.BuildVersion,
		"protocol", c.cfg.ProtocolVersion,
		"token", c.cfg.Token != nil && c.cfg.Token() != "",
	)

	session, err := yamux.Server(sharedtunnel.ClientConn(stream), yamux.DefaultConfig())
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	server := &http.Server{Handler: c.cfg.Handler}
	return server.Serve(yamuxListener{session})
}

// yamuxListener adapts a yamux session to net.Listener so http.Server can
// serve the agent's API over tunneled streams.
type yamuxListener struct {
	*yamux.Session
}

// tunnelAuthOption picks the tunnel credential: the personal access token when
// one is configured, else the legacy basic-auth pair.
func tunnelAuthOption(cfg Config) connect.ClientOption {
	if cfg.Token != nil {
		if token := cfg.Token(); token != "" {
			return sharedtunnel.BearerAuthOption(token)
		}
	}
	return sharedtunnel.BasicAuthOption(cfg.Username, cfg.Password)
}

func (l yamuxListener) Accept() (net.Conn, error) { return l.Session.Accept() }
func (l yamuxListener) Addr() net.Addr            { return l.LocalAddr() }
func (l yamuxListener) Close() error              { return l.Session.Close() }
