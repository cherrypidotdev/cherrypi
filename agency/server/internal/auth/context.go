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

type Identity struct {
	ActorID  ids.ActorID
	Kind     string
	Username string
	Name     string
	Role     string
	Status   string

	// username invite's first login). Only cookie sessions carry it.
	MustChangePassword bool
}

func (i Identity) IsAdmin() bool { return i.Kind == "user" && i.Role == "admin" }

type identityKey struct{}

func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok && id.ActorID != 0
}

func ActorID(ctx context.Context) (ids.ActorID, bool) {
	id, ok := IdentityFrom(ctx)
	return id.ActorID, ok
}

func IsAdmin(ctx context.Context) bool {
	id, ok := IdentityFrom(ctx)
	return ok && id.IsAdmin()
}

type usernameKey struct{}

func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

func Username(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey{}).(string)
	return username, ok && username != ""
}

type agentIDKey struct{}

func WithAgentID(ctx context.Context, agentID ids.AgentID) context.Context {
	return context.WithValue(ctx, agentIDKey{}, agentID)
}

func AgentID(ctx context.Context) (ids.AgentID, bool) {
	agentID, ok := ctx.Value(agentIDKey{}).(ids.AgentID)
	return agentID, ok && agentID != 0
}
