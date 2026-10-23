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

package auth

import "context"

// usernameKey is the context key for the authenticated username. It is a
// private struct type so no other package can collide with it.
type usernameKey struct{}

// WithUsername returns a copy of ctx carrying the authenticated username. It
// is set by the basic-auth middleware and read by handlers (and the terminal
// broker) that need to attribute work to a credential.
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

// Username returns the authenticated username carried by ctx, if any. Handlers
// run behind the auth middleware, so it is present for every protected route;
// tests may call handlers directly and get ("", false).
func Username(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey{}).(string)
	return username, ok && username != ""
}
