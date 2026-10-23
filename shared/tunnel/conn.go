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

// Package tunnel adapts a tunnel.v1 Connect bidi stream into a full-duplex
// io.ReadWriteCloser and provides the HTTP client used to run yamux over it.
//
// The agency and the agent both speak the same proto; this package is the
// protocol plumbing they share.
package tunnel

import (
	"io"
	"sync"

	tunnelv1 "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1"
	"connectrpc.com/connect"
)

// Conn is a full-duplex byte stream over a tunnel.v1 Frame stream. It follows
// yamux's contract: one concurrent reader, one concurrent writer.
type Conn struct {
	send    func([]byte) error
	recv    func() ([]byte, error)
	closeFn func() error

	writeMu sync.Mutex
	buf     []byte
}

// NewConn builds a Conn from send/receive/close hooks.
func NewConn(
	send func([]byte) error,
	recv func() ([]byte, error),
	closeFn func() error,
) *Conn {
	return &Conn{send: send, recv: recv, closeFn: closeFn}
}

// ClientConn adapts the client side of TunnelService.Connect.
func ClientConn(stream *connect.BidiStreamForClientSimple[tunnelv1.Frame, tunnelv1.Frame]) *Conn {
	return NewConn(
		func(b []byte) error { return stream.Send(&tunnelv1.Frame{Data: b}) },
		func() ([]byte, error) {
			msg, err := stream.Receive()
			if err != nil {
				return nil, err
			}
			return msg.Data, nil
		},
		func() error {
			_ = stream.CloseRequest()
			return stream.CloseResponse()
		},
	)
}

// ServerConn adapts the server side of TunnelService.Connect.
func ServerConn(stream *connect.BidiStream[tunnelv1.Frame, tunnelv1.Frame]) *Conn {
	return NewConn(
		func(b []byte) error { return stream.Send(&tunnelv1.Frame{Data: b}) },
		func() ([]byte, error) {
			msg, err := stream.Receive()
			if err != nil {
				return nil, err
			}
			return msg.Data, nil
		},
		nil,
	)
}

// Read fills p from the next received frames. Frames are opaque chunks of the
// byte stream, not message boundaries.
func (c *Conn) Read(p []byte) (int, error) {
	for len(c.buf) == 0 {
		data, err := c.recv()
		if err != nil {
			return 0, err
		}
		if len(data) == 0 {
			continue
		}
		c.buf = data
	}
	n := copy(p, c.buf)
	c.buf = c.buf[n:]
	return n, nil
}

// Write sends p as one frame.
func (c *Conn) Write(p []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	data := make([]byte, len(p))
	copy(data, p)
	if err := c.send(data); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close closes the underlying stream (client side); the server closes when its
// handler returns.
func (c *Conn) Close() error {
	if c.closeFn == nil {
		return nil
	}
	return c.closeFn()
}

var _ io.ReadWriteCloser = (*Conn)(nil)
