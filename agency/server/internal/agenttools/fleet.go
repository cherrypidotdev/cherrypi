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

package agenttools

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agentlogs"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
)

func (h *Handler) requireAgentAdmin(w http.ResponseWriter, r *http.Request) (agents.Agent, bool) {
	agentID, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return agents.Agent{}, false
	}
	caller, err := h.Agents.Get(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return agents.Agent{}, false
	}
	if caller.Role != agents.RoleAdmin {
		writeError(w, http.StatusForbidden, "the admin agent role is required")
		return agents.Agent{}, false
	}
	return caller, true
}

type agentMetaJSON struct {
	ID              string `json:"id"`
	ActorID         string `json:"actor_id"`
	Username        string `json:"username"`
	DisplayName     string `json:"display_name"`
	Description     string `json:"description"`
	Role            string `json:"role"`
	Status          string `json:"status"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocol_version"`
	RemoteAddr      string `json:"remote_addr"`
	LastSeenAt      string `json:"last_seen_at,omitempty"`
}

func toAgentMetaJSON(a agents.Agent) agentMetaJSON {
	out := agentMetaJSON{
		ID:              actorid.Format(a.ID),
		ActorID:         actorid.Format(a.ActorID),
		Username:        a.Username,
		DisplayName:     a.DisplayName,
		Description:     a.Description,
		Role:            a.Role,
		Version:         a.Version,
		ProtocolVersion: a.ProtocolVersion,
		RemoteAddr:      a.RemoteAddr,
	}
	switch a.Status {
	case agents.StatusOnline:
		out.Status = "online"
	default:
		out.Status = "offline"
	}
	if !a.LastSeenAt.IsZero() {
		out.LastSeenAt = a.LastSeenAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return out
}

func (h *Handler) agencyMeta(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireAgentAdmin(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if h.ProcessLogs == nil {
		writeError(w, http.StatusServiceUnavailable, "the agency is not writing a process log")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agency": h.ProcessLogs.Meta(),
		"caller": toAgentMetaJSON(caller),
	})
}

func (h *Handler) agencyLogs(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAgentAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if h.ProcessLogs == nil {
		writeError(w, http.StatusServiceUnavailable, "the agency is not writing a process log")
		return
	}
	lines := intQuery(r, "lines", 200, 2000)
	rows, offset, err := h.ProcessLogs.Tail(lines)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"meta":   h.ProcessLogs.Meta(),
		"lines":  rows,
		"offset": offset,
	})
}

func (h *Handler) listFleet(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAgentAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	list, err := h.Agents.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]agentMetaJSON, len(list))
	for i, a := range list {
		out[i] = toAgentMetaJSON(a)
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": out})
}

func (h *Handler) agentLogs(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAgentAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	rest, found := strings.CutPrefix(r.URL.Path, Prefix+"/agents/")
	if !found {
		http.NotFound(w, r)
		return
	}
	idPart, action, found := strings.Cut(rest, "/")
	if !found || action != "logs" || idPart == "" || strings.ContainsRune(idPart, '/') {
		http.NotFound(w, r)
		return
	}
	agentID := actorid.ParseAgent(idPart)
	agent, err := h.Agents.Get(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if h.Logs == nil {
		writeError(w, http.StatusServiceUnavailable, "the agent log store is unavailable")
		return
	}

	level := agentlogs.ParseLevel(r.URL.Query().Get("min_level"))
	if r.URL.Query().Get("min_level") == "" {
		level = agentlogs.LevelDebug
	}
	limit := intQuery(r, "limit", 200, 1000)
	before, _ := strconv.ParseInt(r.URL.Query().Get("before_id"), 10, 64)

	rows, err := h.Logs.List(r.Context(), agentID, level, before, int64(limit))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	logs := make([]agentLogJSON, len(rows))
	for i, row := range rows {
		logs[i] = toAgentLogJSON(row)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agent": toAgentMetaJSON(agent),
		"logs":  logs,
	})
}

type agentLogJSON struct {
	ID        int64           `json:"id"`
	At        int64           `json:"at"`
	Level     string          `json:"level"`
	Message   string          `json:"message"`
	TaskID    string          `json:"task_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Attrs     json.RawMessage `json:"attrs,omitempty"`
}

func toAgentLogJSON(row agentlogs.Row) agentLogJSON {
	out := agentLogJSON{
		ID:        row.ID,
		At:        row.At,
		Level:     agentlogs.LevelName(row.Level),
		Message:   row.Message,
		TaskID:    row.TaskID,
		SessionID: row.SessionID,
	}
	if row.AttrsJSON != "" {
		out.Attrs = json.RawMessage(row.AttrsJSON)
	}
	return out
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
