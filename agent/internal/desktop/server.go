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

package desktop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/coder/websocket"
)

// WebSocketPath is the mount point of the RFB data plane.
const WebSocketPath = "/api/desktop/ws"

// ViewOnlyHeader is set by the agency relay to select the read-only bridge.
const ViewOnlyHeader = "X-CherryPi-View-Only"

// WSHandler bridges a browser's noVNC WebSocket to the desktop's VNC server.
// The bridge is started on demand. Requests are authenticated by the
// surrounding auth middleware (the agency forwards the authenticated user);
// the browser reuses its cached basic-auth entry on the handshake.
func (m *Manager) WSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != WebSocketPath {
			http.NotFound(w, r)
			return
		}

		// The agency sets this header from the caller's agent ownership. Absent
		// (or anything other than an explicit "0") defaults to view-only, so a
		// path that bypasses the agency relay can never control the desktop.
		viewOnly := r.Header.Get(ViewOnlyHeader) != "0"
		port, err := m.Ensure(r.Context(), viewOnly)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// The handler is behind auth; accept any origin so the Vite dev
			// server (a different port) can attach too.
			OriginPatterns: []string{"*"},
			// noVNC's RFB transport uses the "binary" subprotocol.
			Subprotocols: []string{"binary"},
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		_ = proxy(r.Context(), fmt.Sprintf("127.0.0.1:%d", port), conn)
	})
}

// proxy copies bytes between an RFB WebSocket and the VNC TCP socket. Chunk
// boundaries are irrelevant to RFB, so each WebSocket message is written
// straight through in both directions.
func proxy(ctx context.Context, addr string, conn *websocket.Conn) error {
	var d net.Dialer
	tcp, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial vnc: %w", err)
	}
	defer func() { _ = tcp.Close() }()

	conn.SetReadLimit(1 << 20)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				errCh <- err
				return
			}
			if _, werr := tcp.Write(data); werr != nil {
				errCh <- werr
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := tcp.Read(buf)
			if n > 0 {
				if werr := conn.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
					errCh <- werr
					return
				}
			}
			if err != nil {
				errCh <- err
				return
			}
		}
	}()

	err = <-errCh
	cancel()
	_ = conn.Close(websocket.StatusNormalClosure, "")

	switch {
	case err == nil,
		errors.Is(err, io.EOF),
		errors.Is(err, net.ErrClosed),
		websocket.CloseStatus(err) == websocket.StatusNormalClosure,
		websocket.CloseStatus(err) == websocket.StatusGoingAway:
		return nil
	default:
		return err
	}
}
