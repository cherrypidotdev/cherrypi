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

package processlog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"cherrypi.vladfaust.com/agency/internal/auth"
)

const Prefix = "/api/rest/logs"

type Handler struct {
	Log *Log
}

type TailResponse struct {
	Meta Meta `json:"meta"`

	Lines []string `json:"lines"`

	Offset int64 `json:"offset"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAdmin(r.Context()) {
		writeError(w, http.StatusForbidden, "administrators only")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.Log == nil {
		writeError(w, http.StatusServiceUnavailable,
			"the agency is not writing a process log (read-only data directory?)")
		return
	}

	switch r.URL.Path {
	case Prefix, Prefix + "/":
		h.tail(w, r)
	case Prefix + "/stream":
		h.stream(w, r)
	case Prefix + "/download":
		h.download(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) tail(w http.ResponseWriter, r *http.Request) {
	lines := intQuery(r, "lines", 500, 5000)
	rows, offset, err := h.Log.Tail(lines)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, TailResponse{
		Meta:   h.Log.Meta(),
		Lines:  rows,
		Offset: offset,
	})
}

func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)

	// line wins over the stale query offset, so no line is lost or repeated.
	if last := r.Header.Get("Last-Event-ID"); last != "" {
		if parsed, err := strconv.ParseInt(last, 10, 64); err == nil {
			offset = parsed
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	err := h.Log.Follow(r.Context(), offset, func(next int64, line string) bool {
		payload, err := json.Marshal(map[string]any{
			"offset": next,
			"text":   line,
		})
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", next, payload); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}, func() bool {
		if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
			return false
		}
		flusher.Flush()
		return true
	})

	_ = err
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	file, err := http.Dir(h.Log.Dir()).Open(h.Log.Name())
	if err != nil {
		writeError(w, http.StatusNotFound, "log file is not readable")
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+h.Log.Name()+`"`)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeContent(w, r, h.Log.Name(), info.ModTime(), file)
}

func intQuery(r *http.Request, name string, fallback, max int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
