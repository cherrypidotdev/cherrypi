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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const WSPath = "/api/tunnel/ws"

const (
	HelloBuildVersionHeader    = "X-CherryPi-Build-Version"
	HelloProtocolVersionHeader = "X-CherryPi-Protocol-Version"
)

const WSReadLimit = 1 << 20

type WSConn struct {
	conn *websocket.Conn

	ctx    context.Context
	cancel context.CancelFunc

	writeTimeout time.Duration

	readMu sync.Mutex
	reader io.Reader

	closeOnce sync.Once
}

func NewWSConn(conn *websocket.Conn) *WSConn {
	ctx, cancel := context.WithCancel(context.Background())
	conn.SetReadLimit(WSReadLimit)
	return &WSConn{
		conn:         conn,
		ctx:          ctx,
		cancel:       cancel,
		writeTimeout: 10 * time.Second,
	}
}

func (c *WSConn) Read(p []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	for {
		if c.reader == nil {
			typ, reader, err := c.conn.Reader(c.ctx)
			if err != nil {
				return 0, err
			}
			if typ != websocket.MessageBinary {

				_, _ = io.Copy(io.Discard, reader)
				continue
			}
			c.reader = reader
		}

		n, err := c.reader.Read(p)
		if errors.Is(err, io.EOF) {
			c.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (c *WSConn) Write(p []byte) (int, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.writeTimeout)
	defer cancel()
	if err := c.conn.Write(ctx, websocket.MessageBinary, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *WSConn) Close() error {
	c.closeOnce.Do(func() {
		c.cancel()
		_ = c.conn.CloseNow()
	})
	return nil
}

func DialWS(
	ctx context.Context,
	baseURL string,
	header http.Header,
) (*websocket.Conn, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		return nil, fmt.Errorf("unsupported agency scheme %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/") + WSPath
	u.RawQuery = ""

	conn, resp, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		HTTPHeader: header,
	})
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			_ = resp.Body.Close()
			reason := strings.TrimSpace(string(body))
			if reason != "" {
				return nil, fmt.Errorf("websocket tunnel: %s: %s", resp.Status, reason)
			}
			return nil, fmt.Errorf("websocket tunnel: %s", resp.Status)
		}
		return nil, err
	}
	conn.SetReadLimit(WSReadLimit)
	return conn, nil
}
