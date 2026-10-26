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
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	tunnelv1 "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1"
	tunnelv1connect "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1/tunnelv1connect"
	sharedtunnel "cherrypi.vladfaust.com/shared/tunnel"
	"connectrpc.com/connect"
	"github.com/hashicorp/yamux"
	"log/slog"
)

type transport int

const (
	transportAuto transport = iota

	transportHTTP2

	transportWebSocket
)

type Config struct {
	AgencyURL string

	Transport string

	Token func() string

	Username string
	Password string

	Handler http.Handler

	BuildVersion string

	ProtocolVersion string
}

type Client struct {
	cfg Config

	transport transport
}

func New(cfg Config) *Client {
	client := &Client{cfg: cfg}
	switch strings.ToLower(strings.TrimSpace(cfg.Transport)) {
	case "http2", "h2", "connect":
		client.transport = transportHTTP2
	case "websocket", "ws":
		client.transport = transportWebSocket
	default:
		client.transport = transportAuto
	}
	return client
}

func (c *Client) Run(ctx context.Context) {
	const maxBackoff = 30 * time.Second
	backoff := time.Second

	for {
		err := c.connect(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			slog.Warn(
				"tunnel disconnected",
				"agency", c.cfg.AgencyURL,
				"agent", c.cfg.Username,
				"transport", c.transportName(),
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
	if c.transport == transportWebSocket {
		return c.connectWS(ctx)
	}

	err := c.connectHTTP2(ctx)
	if err == nil {
		return nil
	}
	if c.forcedHTTP2() || !isTransportUnsupported(err) {
		return err
	}

	c.transport = transportWebSocket
	slog.Info(
		"tunnel transport fallback",
		"from", "http2",
		"to", "websocket",
		"reason", err,
	)
	return c.connectWS(ctx)
}

func (c *Client) forcedHTTP2() bool {
	switch strings.ToLower(strings.TrimSpace(c.cfg.Transport)) {
	case "http2", "h2", "connect":
		return true
	default:
		return false
	}
}

func isTransportUnsupported(err error) bool {
	var unsupported *transportUnsupportedError
	return errors.As(err, &unsupported)
}

type transportUnsupportedError struct {
	err error
}

func (e *transportUnsupportedError) Error() string {
	return fmt.Sprintf("HTTP/2 tunnel unavailable: %v", e.err)
}

func (e *transportUnsupportedError) Unwrap() error { return e.err }

func (c *Client) connectHTTP2(ctx context.Context) error {
	httpClient, err := sharedtunnel.NewHTTPClient(c.cfg.AgencyURL)
	if err != nil {
		return err
	}

	client := tunnelv1connect.NewTunnelServiceClient(
		httpClient,
		strings.TrimRight(c.cfg.AgencyURL, "/")+"/api/grpc",
		connect.WithInterceptors(sharedtunnel.BasicAuthInterceptor{Header: c.authHeader()}),
	)

	stream, err := client.Connect(ctx)
	if err != nil {
		return err
	}

	if err := stream.Send(&tunnelv1.Frame{Hello: &tunnelv1.Hello{
		BuildVersion:    c.cfg.BuildVersion,
		ProtocolVersion: c.cfg.ProtocolVersion,
	}}); err != nil {
		return err
	}

	conn := sharedtunnel.ClientConn(stream)

	logger := &probeLogger{}
	config := yamux.DefaultConfig()
	config.LogOutput = nil
	config.Logger = logger
	session, err := yamux.Server(conn, config)
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := probeHTTP2(ctx, session); err != nil {
		_ = session.Close()
		return &transportUnsupportedError{err: err}
	}
	logger.verbose.Store(true)

	c.transport = transportHTTP2
	c.logConnected("http2")

	server := &http.Server{Handler: c.cfg.Handler}
	return server.Serve(yamuxListener{session})
}

func (c *Client) connectWS(ctx context.Context) error {
	header := http.Header{}
	header.Set("Authorization", c.authHeader())
	header.Set(sharedtunnel.HelloBuildVersionHeader, c.cfg.BuildVersion)
	header.Set(sharedtunnel.HelloProtocolVersionHeader, c.cfg.ProtocolVersion)

	conn, err := sharedtunnel.DialWS(ctx, c.cfg.AgencyURL, header)
	if err != nil {
		return err
	}
	pipe := sharedtunnel.NewWSConn(conn)
	defer func() { _ = pipe.Close() }()

	session, err := yamux.Server(pipe, yamux.DefaultConfig())
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	c.transport = transportWebSocket
	c.logConnected("websocket")

	server := &http.Server{Handler: c.cfg.Handler}
	return server.Serve(yamuxListener{session})
}

func (c *Client) logConnected(transport string) {
	slog.Info(
		"tunnel connected",
		"agency", c.cfg.AgencyURL,
		"agent", c.cfg.Username,
		"version", c.cfg.BuildVersion,
		"protocol", c.cfg.ProtocolVersion,
		"transport", transport,
		"token", c.cfg.Token != nil && c.cfg.Token() != "",
	)
}

func (c *Client) transportName() string {
	switch c.transport {
	case transportHTTP2:
		return "http2"
	case transportWebSocket:
		return "websocket"
	default:
		return "auto"
	}
}

func (c *Client) authHeader() string {
	if c.cfg.Token != nil {
		if token := c.cfg.Token(); token != "" {
			return "Bearer " + token
		}
	}
	return "Basic " + base64.StdEncoding.EncodeToString(
		[]byte(c.cfg.Username+":"+c.cfg.Password),
	)
}

const probeHTTP2Timeout = 2 * time.Second

func probeHTTP2(ctx context.Context, session *yamux.Session) error {
	done := make(chan error, 1)
	go func() {
		_, err := session.Ping()
		done <- err
	}()

	timer := time.NewTimer(probeHTTP2Timeout)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		return errors.New("timed out waiting for the HTTP/2 tunnel handshake")
	case <-ctx.Done():
		return ctx.Err()
	}
}

type probeLogger struct {
	verbose atomic.Bool
}

func (l *probeLogger) Print(v ...any) {
	if l.verbose.Load() {
		log.Print(v...)
	}
}

func (l *probeLogger) Printf(format string, v ...any) {
	if l.verbose.Load() {
		log.Printf(format, v...)
	}
}

func (l *probeLogger) Println(v ...any) {
	if l.verbose.Load() {
		log.Println(v...)
	}
}

type yamuxListener struct {
	*yamux.Session
}

func (l yamuxListener) Accept() (net.Conn, error) { return l.Session.Accept() }
func (l yamuxListener) Addr() net.Addr            { return l.LocalAddr() }
func (l yamuxListener) Close() error              { return l.Session.Close() }
