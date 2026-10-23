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
)

// tunnelPathPrefix is the agent tunnel's route; it authenticates with an agent
// personal access token or the legacy tunnel credential, never a user
// credential.
const tunnelPathPrefix = "/api/grpc/tunnel.v1.TunnelService/"

// attachmentsPathPrefix is the attachment REST surface. It authenticates with
// a user (session/PAT/Basic) or an agent (PAT/Basic), so it runs its own
// middleware inside withAuth.
const (
	attachmentsPathPrefix = "/api/rest/attachments"
)

// accountAuthenticator is the slice of accounts.Service the middleware needs,
// so tests can supply a stub.
type accountAuthenticator interface {
	ResolveSession(ctx context.Context, token string) (accounts.Identity, accounts.Session, error)
	ResolvePAT(ctx context.Context, token string) (accounts.Identity, accounts.PAT, error)
	AuthenticateBasic(ctx context.Context, username, password string) (accounts.Identity, bool)
}

// agentAuthenticator authenticates an agent tunnel credential and resolves an
// agent actor. internal/agents.Registry satisfies it.
type agentAuthenticator interface {
	Authenticate(ctx context.Context, username, password string) (ids.AgentID, bool)
	GetByActor(ctx context.Context, actorID ids.ActorID) (agents.Agent, error)
}

// withAuth is the browser/API entry point. It resolves a session cookie, a
// personal access token or (legacy) HTTP Basic into an identity and enforces
// CSRF on cookie-authenticated mutations. Static assets and the public auth
// routes pass through so the SPA can load its login page.
func withAuth(a accountAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health and the non-API SPA/assets are always reachable.
		if r.URL.Path == "/api/rest/health" || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// These surfaces authenticate the agent (personal access token or legacy
		// tunnel credential) and run their own middleware.
		if strings.HasPrefix(r.URL.Path, tunnelPathPrefix) ||
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
			// password before anything but the password form.
			if identity.MustChangePassword && !passwordChangeExempt(r) {
				writePasswordChangeRequired(w)
				return
			}
			ctx := auth.WithIdentity(r.Context(), identity)
			ctx = auth.WithUsername(ctx, identity.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Public auth routes stay reachable unauthenticated.
		if strings.HasPrefix(r.URL.Path, accounts.AuthPrefix) {
			next.ServeHTTP(w, r)
			return
		}

		writeUnauthorized(w)
	})
}

// resolveIdentity tries the session cookie, then a PAT bearer token, then the
// legacy basic-auth pair. viaCookie reports whether the session cookie matched
// (which is what makes CSRF checks apply).
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

// csrfOK enforces the double-submit cookie on cookie-authenticated mutations.
func csrfOK(r *http.Request) bool {
	cookie, err := r.Cookie(accounts.CSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	header := r.Header.Get(accounts.CSRFHeaderName)
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) == 1
}

// resolveAgentRequest authenticates an agent request from a personal access
// token (preferred) or a legacy tunnel credential and returns the agent id.
// Only agent-kind PATs are accepted: a user PAT never grants agent access. The
// returned identity is set for PAT callers and zero for Basic callers.
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

// withAgentContext records an authenticated agent on the request: the agent id
// always, plus the full identity when a personal access token supplied one.
// The username is intentionally not set so attribution keeps using the agent
// id (as the legacy Basic path did).
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

// withAttachmentAuth authenticates the attachment routes: an agent (PAT or
// legacy tunnel credential), or a user (session, PAT, or legacy Basic).
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

// withAgentToolsAuth authenticates the agent-tool routes: a personal access
// token (Pi's bundled agent-tools extension) or a legacy tunnel credential.
// Nothing else may use these routes.
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

// withAgentAuth authenticates a route with an agent personal access token or a
// legacy tunnel credential and sets the caller's agent id in the request
// context. It is the narrow surface the session-history sync uses; no user
// credential is accepted.
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

// bearerToken extracts an `Authorization: Bearer …` token, if present.
func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token == "" {
		return "", false
	}
	return token, true
}

// writeUnauthorized returns a bare 401 with no challenge, so browsers do not
// pop their native basic-auth prompt; the SPA routes to its login page.
func writeUnauthorized(w http.ResponseWriter) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// passwordChangeExempt reports the routes a first-login user may still call
// while a password reset is outstanding: the session itself, logout, the
// account read and the password change.
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

// writePasswordChangeRequired refuses an API call until the password reset.
func writePasswordChangeRequired(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"password change required"}`))
}
