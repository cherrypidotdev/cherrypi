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

package plugins

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultMaxBody int64 = 10 << 20

type Handler struct {
	Service *Service

	Auth func(*http.Request) (Caller, bool)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/plugins/")
	if rest == r.URL.Path || rest == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	sub := "/"
	if len(parts) > 1 {
		sub = "/" + strings.TrimPrefix(parts[1], "/")
	}
	p, err := h.Service.Get(r.Context(), id)
	if err != nil || !p.Enabled {
		http.NotFound(w, r)
		return
	}

	maxBody := DefaultMaxBody
	if p.Manifest.Limits.MaxBody > 0 {
		maxBody = p.Manifest.Limits.MaxBody
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil || int64(len(body)) > maxBody {
		writeHTTPJSON(
			w,
			http.StatusRequestEntityTooLarge,
			map[string]any{"error": "request body too large"},
		)
		return
	}

	caller := Caller{Kind: "anonymous"}
	if h.Auth != nil {
		if resolved, ok := h.Auth(r); ok {
			caller = resolved
		}
	}
	inv := Invocation{
		ID:     newInvocationID(),
		Kind:   HandlerHTTP,
		Caller: caller,
		Route: &RouteRequest{
			Method:  r.Method,
			Path:    sub,
			Query:   r.URL.Query(),
			Headers: routeHeaders(r.Header),
			Body:    body,
		},
	}
	started := time.Now()
	resp, err := h.Service.Invoke(r.Context(), p, inv)
	if err != nil {
		h.Service.Store.Log(
			r.Context(),
			p.ID,
			"error",
			"route",
			inv.ID,
			"invocation failed",
			map[string]any{"error": err.Error(), "path": sub},
		)
		writeHTTPJSON(w, http.StatusBadGateway, map[string]any{"error": "plugin invocation failed"})
		return
	}
	h.Service.Store.Log(
		r.Context(),
		p.ID,
		"info",
		"route",
		inv.ID,
		r.Method+" "+sub,
		map[string]any{
			"status":   resp.Status,
			"duration": time.Since(started).Milliseconds(),
		},
	)
	writePluginResponse(w, resp)
}

func writePluginResponse(w http.ResponseWriter, resp Response) {
	for key, value := range resp.Headers {
		if strings.EqualFold(key, "content-length") {
			continue
		}
		w.Header().Set(key, value)
	}
	if w.Header().Get("content-type") == "" {
		w.Header().Set("content-type", "text/plain; charset=utf-8")
	}
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write(resp.Body)
}

func routeHeaders(headers http.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for key, values := range headers {
		out[strings.ToLower(key)] = strings.Join(values, ", ")
	}
	return out
}

func writeHTTPJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
