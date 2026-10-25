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

package audit

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/auth"
)

const Prefix = "/api/rest/events"

type Handler struct {
	Service *Service
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	path := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix), "/")
	switch path {
	case "":
		h.list(w, r)
	case "export.csv":
		h.export(w, r, "csv")
	case "export.json":
		h.export(w, r, "json")
	case "settings":
		h.settings(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	filter, err := parseFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := h.Service.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load events")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"logs":        EntriesJSON(page.Entries),
		"next_cursor": page.NextCursor,
	})
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request, format string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	filter, err := parseFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	filter.Limit = ExportLimit
	filter.Cursor = ""
	page, err := h.Service.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not export events")
		return
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set(
			"Content-Disposition",
			"attachment; filename=\"events-"+stamp+".csv\"",
		)
		if err := WriteCSV(w, page.Entries); err != nil {
			writeError(w, http.StatusInternalServerError, "could not write export")
		}
	default:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set(
			"Content-Disposition",
			"attachment; filename=\"events-"+stamp+".json\"",
		)
		if err := WriteJSON(w, page.Entries); err != nil {
			writeError(w, http.StatusInternalServerError, "could not write export")
		}
	}
}

func (h *Handler) settings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"retention_days": h.Service.RetentionDays(r.Context()),
		})
	case http.MethodPut, http.MethodPatch:
		var in struct {
			RetentionDays int `json:"retention_days"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		if in.RetentionDays <= 0 {
			writeError(w, http.StatusBadRequest, "retention_days must be positive")
			return
		}
		if err := h.Service.SetRetentionDays(r.Context(), in.RetentionDays); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save settings")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"retention_days": in.RetentionDays,
		})
	default:
		methodNotAllowed(w)
	}
}

func parseFilter(r *http.Request) (Filter, error) {
	query := r.URL.Query()
	filter := Filter{
		Actor:  strings.TrimSpace(query.Get("actor")),
		Scope:  Scope(strings.TrimSpace(query.Get("scope"))),
		Kind:   strings.TrimSpace(query.Get("kind")),
		Search: strings.TrimSpace(query.Get("q")),
		Cursor: strings.TrimSpace(query.Get("cursor")),
	}
	if raw := query.Get("limit"); raw != "" {
		if limit, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.Limit = limit
		}
	}
	since, err := parseBound(query.Get("since"), false)
	if err != nil {
		return Filter{}, err
	}
	until, err := parseBound(query.Get("until"), true)
	if err != nil {
		return Filter{}, err
	}
	filter.Since = since
	filter.Until = until
	return filter, nil
}

func parseBound(value string, endOfDay bool) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, errors.New("since/until must be RFC3339 or YYYY-MM-DD")
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Millisecond)
	}
	return parsed, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}
