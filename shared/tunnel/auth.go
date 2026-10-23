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

	"connectrpc.com/connect"
)

// BasicAuthInterceptor adds a basic-auth `Authorization` header to every
// request, including streaming clients.
type BasicAuthInterceptor struct {
	Header string
}

// BasicAuthOption builds a client option that authenticates with the given
// credentials.
func BasicAuthOption(username, password string) connect.ClientOption {
	return connect.WithInterceptors(BasicAuthInterceptor{
		Header: "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password)),
	})
}

// BearerAuthOption builds a client option that authenticates with a bearer
// token, used for an agent's personal access token.
func BearerAuthOption(token string) connect.ClientOption {
	return connect.WithInterceptors(BasicAuthInterceptor{
		Header: "Bearer " + token,
	})
}

func (i BasicAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("Authorization", i.Header)
		return next(ctx, req)
	}
}

func (i BasicAuthInterceptor) WrapStreamingClient(
	next connect.StreamingClientFunc,
) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set("Authorization", i.Header)
		return conn
	}
}

func (i BasicAuthInterceptor) WrapStreamingHandler(
	next connect.StreamingHandlerFunc,
) connect.StreamingHandlerFunc {
	return next
}
