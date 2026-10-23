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

import (
	"context"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

// Identity is the authenticated caller: a user or an agent actor. It is set by
// the request middleware and read by handlers.
type Identity struct {
	ActorID  ids.ActorID
	Kind     string
	Username string
	Name     string
	Role     string
	Status   string
	// MustChangePassword forces a password reset before other API use (a
	// username invite's first login). Only cookie sessions carry it.
	MustChangePassword bool
}

// IsAdmin reports whether the caller is an administrator.
func (i Identity) IsAdmin() bool { return i.Kind == "user" && i.Role == "admin" }

// identityKey is the context key for the authenticated identity. It is a
// private struct type so no other package can collide with it.
type identityKey struct{}

// WithIdentity returns a copy of ctx carrying the authenticated identity.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// IdentityFrom returns the authenticated identity carried by ctx, if any.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok && id.ActorID != 0
}

// ActorID returns the authenticated actor id, if any.
func ActorID(ctx context.Context) (ids.ActorID, bool) {
	id, ok := IdentityFrom(ctx)
	return id.ActorID, ok
}

// IsAdmin reports whether the caller is an authenticated administrator.
func IsAdmin(ctx context.Context) bool {
	id, ok := IdentityFrom(ctx)
	return ok && id.IsAdmin()
}

// usernameKey is the context key for the authenticated username, kept for the
// handful of places (terminal ownership) that only need the name.
type usernameKey struct{}

// WithUsername returns a copy of ctx carrying the authenticated username. It
// is set by the middleware alongside the full identity.
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

// Username returns the authenticated username carried by ctx, if any.
func Username(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey{}).(string)
	return username, ok && username != ""
}

// agentIDKey is the context key for the authenticated agent id.
type agentIDKey struct{}

// WithAgentID returns a copy of ctx carrying the authenticated agent id. It is
// set where an agent authenticates with its tunnel credential (the attachment
// routes and the agent-tool surface).
func WithAgentID(ctx context.Context, agentID ids.AgentID) context.Context {
	return context.WithValue(ctx, agentIDKey{}, agentID)
}

// AgentID returns the authenticated agent id carried by ctx, if any.
func AgentID(ctx context.Context) (ids.AgentID, bool) {
	agentID, ok := ctx.Value(agentIDKey{}).(ids.AgentID)
	return agentID, ok && agentID != 0
}
