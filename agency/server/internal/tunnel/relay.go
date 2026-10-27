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
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"net/http"
	"strconv"
	"strings"

	agencyauth "cherrypi.vladfaust.com/agency/internal/auth"
	"github.com/coder/websocket"
)

const relayReadLimit = 1 << 20

const wsPrefix = "/api/agents/"

func (h *Hub) WSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentID, upstreamPath, ok := routeRelay(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.relayWS(w, r, agentID, upstreamPath)
	})
}

func routeRelay(path string) (agentID ids.AgentID, upstreamPath string, ok bool) {
	if agentID, terminalID, ok := parseTerminalPath(path); ok {
		return agentID, "/api/terminal/" + terminalID + "/ws", true
	}
	if agentID, ok := parseDesktopPath(path); ok {
		return agentID, "/api/desktop/ws", true
	}
	return 0, "", false
}

func (h *Hub) relayWS(
	w http.ResponseWriter,
	r *http.Request,
	agentID ids.AgentID, upstreamPath string,
) {
	desktop := strings.HasSuffix(upstreamPath, "/desktop/ws")
	owner := h.canManageAgent(r.Context(), agentID)

	if !desktop && !owner {
		http.Error(w, "agent owner or administrator required", http.StatusForbidden)
		return
	}

	conn, err := h.Connection(agentID)
	if err != nil {
		http.Error(w, "agent offline", http.StatusNotFound)
		return
	}

	user, _ := agencyauth.Username(r.Context())
	headers := http.Header{"X-CherryPi-User": {user}}
	if desktop {
		headers.Set("X-CherryPi-View-Only", boolHeader(!owner))
	}
	upstream, _, err := websocket.Dial(
		r.Context(),
		"ws://agent"+upstreamPath,
		&websocket.DialOptions{
			HTTPClient: conn.HTTPClient(),
			HTTPHeader: headers,
		},
	)
	if err != nil {
		http.Error(w, "relay unavailable", http.StatusBadGateway)
		return
	}
	upstream.SetReadLimit(relayReadLimit)
	defer func() { _ = upstream.CloseNow() }()

	client, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// The handler is behind the auth middleware; accept any origin so the Vite

		OriginPatterns: []string{"*"},

		Subprotocols: []string{"binary"},
	})
	if err != nil {
		return
	}
	defer func() { _ = client.CloseNow() }()

	proxyWS(r.Context(), client, upstream)

	if !desktop && h.OnRelayDone != nil {
		h.OnRelayDone(agentID)
	}
}

func (h *Hub) canManageAgent(ctx context.Context, agentID ids.AgentID) bool {
	if agencyauth.IsAdmin(ctx) {
		return true
	}
	identity, ok := agencyauth.IdentityFrom(ctx)
	if !ok || h.agents == nil {
		return false
	}
	owner, err := h.agents.IsOwner(ctx, agentID, identity.ActorID)
	return err == nil && owner
}

func boolHeader(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func proxyWS(ctx context.Context, client, upstream *websocket.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		for {
			typ, data, err := client.Read(ctx)
			if err != nil {
				cancel()
				return
			}
			if err := upstream.Write(ctx, typ, data); err != nil {
				cancel()
				return
			}
		}
	}()

	for {
		typ, data, err := upstream.Read(ctx)
		if err != nil {
			return
		}
		if err := client.Write(ctx, typ, data); err != nil {
			return
		}
	}
}

func parseTerminalPath(path string) (agentID ids.AgentID, terminalID string, ok bool) {
	rest, found := strings.CutPrefix(path, wsPrefix)
	if !found {
		return 0, "", false
	}

	parts := strings.Split(rest, "/")
	if len(parts) != 4 || parts[1] != "terminal" || parts[3] != "ws" || parts[0] == "" ||
		parts[2] == "" {
		return 0, "", false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}
	return ids.AgentID(id), parts[2], true
}

func parseDesktopPath(path string) (agentID ids.AgentID, ok bool) {
	rest, found := strings.CutPrefix(path, wsPrefix)
	if !found {
		return 0, false
	}

	parts := strings.Split(rest, "/")
	if len(parts) != 3 || parts[1] != "desktop" || parts[2] != "ws" || parts[0] == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return ids.AgentID(id), true
}
