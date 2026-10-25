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

package terminal

import (
	"errors"
	"net/http"
	"strings"

	"cherrypi.vladfaust.com/agent/internal/auth"
	"github.com/coder/websocket"
)

const wsPrefix = "/api/terminal/"

func (m *Manager) WSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, wsPrefix)
		id = strings.TrimSuffix(id, "/ws")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}

		if _, err := m.Get(id); err != nil {
			http.NotFound(w, r)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// The handler is behind basic auth; accept any origin so the Vite

			OriginPatterns: []string{"*"},
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		owner, _ := auth.Username(r.Context())
		if err := m.Attach(r.Context(), id, owner, conn); err != nil {
			status := websocket.StatusInternalError
			if errors.Is(err, ErrForbidden) {
				status = websocket.StatusPolicyViolation
			}
			_ = conn.Close(status, err.Error())
		}
	})
}
