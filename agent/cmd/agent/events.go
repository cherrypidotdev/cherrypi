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
	"encoding/json"
	"net/http"

	"cherrypi.vladfaust.com/agent/cmd/agent/appcontext"
)

// serveEvents streams the agent's event hub as newline-delimited JSON. The
// agency consumes it over the tunnel (one stream per connected agency).
func serveEvents(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	mux.HandleFunc("/api/rest/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		ch, unsubscribe := appCtx.Events.Subscribe()
		defer unsubscribe()

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		enc := json.NewEncoder(w)
		for {
			select {
			case <-r.Context().Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if err := enc.Encode(msg); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	})
}
