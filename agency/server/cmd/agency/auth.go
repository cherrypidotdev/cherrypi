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

package main

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/agenttools"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/taskhistory"
	sharedtunnel "cherrypi.vladfaust.com/shared/tunnel"
)

const (
	tunnelPathPrefix = "/api/grpc/tunnel.v1.TunnelService/"
	tunnelWSPath     = sharedtunnel.WSPath
)

const (
	attachmentsPathPrefix = "/api/rest/attachments"
)

type accountAuthenticator interface {
	ResolveSession(ctx context.Context, token string) (accounts.Identity, accounts.Session, error)
	ResolvePAT(ctx context.Context, token string) (accounts.Identity, accounts.PAT, error)
	AuthenticateBasic(ctx context.Context, username, password string) (accounts.Identity, bool)
}

type agentAuthenticator interface {
	Authenticate(ctx context.Context, username, password string) (ids.AgentID, bool)
	GetByActor(ctx context.Context, actorID ids.ActorID) (agents.Agent, error)
}

func withAuth(a accountAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/api/rest/health" || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, tunnelPathPrefix) ||
			r.URL.Path == tunnelWSPath ||
			strings.HasPrefix(r.URL.Path, attachmentsPathPrefix) ||
			strings.HasPrefix(r.URL.Path, taskhistory.Prefix) ||
			strings.HasPrefix(r.URL.Path, agenttools.Prefix) {
			next.ServeHTTP(w, r)
			return
		}

		identity, viaCookie, ok := resolveIdentity(a, r)
		if ok {
			if viaCookie && isMutating(r.Method) && !csrfOK(r) {
				writeUnauthorized(w)
				return
			}
			// A username invite's first login must replace its generated

			if identity.MustChangePassword && !passwordChangeExempt(r) {
				writePasswordChangeRequired(w)
				return
			}
			ctx := auth.WithIdentity(r.Context(), identity)
			ctx = auth.WithUsername(ctx, identity.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if strings.HasPrefix(r.URL.Path, accounts.AuthPrefix) {
			next.ServeHTTP(w, r)
			return
		}

		writeUnauthorized(w)
	})
}

func resolveIdentity(a accountAuthenticator, r *http.Request) (accounts.Identity, bool, bool) {
	if cookie, err := r.Cookie(accounts.SessionCookieName); err == nil && cookie.Value != "" {
		if identity, _, err := a.ResolveSession(r.Context(), cookie.Value); err == nil {
			return identity, true, true
		}
	}
	if token, ok := bearerToken(r); ok {
		if identity, _, err := a.ResolvePAT(r.Context(), token); err == nil {
			return identity, false, true
		}
	}
	if username, password, ok := r.BasicAuth(); ok {
		if identity, ok := a.AuthenticateBasic(r.Context(), username, password); ok {
			return identity, false, true
		}
	}
	return accounts.Identity{}, false, false
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func csrfOK(r *http.Request) bool {
	cookie, err := r.Cookie(accounts.CSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	header := r.Header.Get(accounts.CSRFHeaderName)
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) == 1
}

func resolveAgentRequest(
	ctx context.Context,
	users accountAuthenticator,
	agents agentAuthenticator,
	r *http.Request,
) (ids.AgentID, accounts.Identity, bool) {
	if token, ok := bearerToken(r); ok {
		identity, _, err := users.ResolvePAT(ctx, token)
		if err != nil || identity.Kind != accounts.KindAgent {
			return 0, accounts.Identity{}, false
		}
		agent, err := agents.GetByActor(ctx, identity.ActorID)
		if err != nil {
			return 0, accounts.Identity{}, false
		}

		identity.Role = agent.Role
		return agent.ID, identity, true
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		return 0, accounts.Identity{}, false
	}
	agentID, ok := agents.Authenticate(ctx, username, password)
	if !ok {
		return 0, accounts.Identity{}, false
	}
	return agentID, accounts.Identity{}, true
}

func withAgentContext(
	ctx context.Context,
	agentID ids.AgentID,
	identity accounts.Identity,
) context.Context {
	ctx = auth.WithAgentID(ctx, agentID)
	if identity.ActorID != 0 {
		ctx = auth.WithIdentity(ctx, identity)
	}
	return ctx
}

func withAttachmentAuth(
	users accountAuthenticator,
	agents agentAuthenticator,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if agentID, identity, ok := resolveAgentRequest(r.Context(), users, agents, r); ok {
			next.ServeHTTP(w, r.WithContext(withAgentContext(r.Context(), agentID, identity)))
			return
		}

		if identity, _, ok := resolveIdentity(users, r); ok {
			ctx := auth.WithIdentity(r.Context(), identity)
			ctx = auth.WithUsername(ctx, identity.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		writeUnauthorized(w)
	})
}

func withAgentToolsAuth(
	users accountAuthenticator,
	agents agentAuthenticator,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentID, identity, ok := resolveAgentRequest(r.Context(), users, agents, r)
		if !ok {
			writeUnauthorized(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(withAgentContext(r.Context(), agentID, identity)))
	})
}

func withAgentAuth(
	users accountAuthenticator,
	agents agentAuthenticator,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentID, identity, ok := resolveAgentRequest(r.Context(), users, agents, r)
		if !ok {
			writeUnauthorized(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(withAgentContext(r.Context(), agentID, identity)))
	})
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token == "" {
		return "", false
	}
	return token, true
}

func writeUnauthorized(w http.ResponseWriter) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func passwordChangeExempt(r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, accounts.AuthPrefix) {
		return true
	}
	switch r.URL.Path {
	case accounts.AccountPrefix, accounts.AccountPrefix + "/",
		accounts.AccountPrefix + "/password":
		return true
	default:
		return false
	}
}

func writePasswordChangeRequired(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"password change required"}`))
}
