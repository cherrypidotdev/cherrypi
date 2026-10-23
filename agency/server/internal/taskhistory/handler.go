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

// Package taskhistory exposes a task's canonical Pi session history to its
// agent over the agency REST surface. The agent authenticates with its tunnel
// credential and names the task by its stable (project_id, seq) reference, so
// the sync survives a project-prefix rename. The agency is the durable home of
// the conversation: the agent pushes newly written entries here, and pulls
// them back to reconstruct a resumable session file on a fresh agent.
package taskhistory

import (
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

// Prefix is the session-history REST surface. It authenticates with the
// agent's tunnel credential, so it runs its own middleware inside withAuth.
const Prefix = "/api/rest/task-history"

// Handler implements the session-history endpoints. It must run behind the
// agent auth middleware, which sets the caller's agent id in the request
// context.
type Handler struct {
	Tasks  *tasks.Service
	Agents *agents.Registry
}

// entryJSON is the wire shape of one Pi session entry.
type entryJSON struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Type     string `json:"type"`
	Raw      string `json:"raw"`
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	agentID, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, agentID)
	case http.MethodPost:
		h.post(w, r, agentID)
	default:
		methodNotAllowed(w)
	}
}

// resolve reads (project_id, seq) and loads the task the agent names, checking
// that the caller is its assignee.
func (h *Handler) resolve(
	w http.ResponseWriter,
	r *http.Request,
	agentID ids.AgentID, projectID string,
	seq int64,
) (tasks.Task, bool) {
	if strings.TrimSpace(projectID) == "" || seq <= 0 {
		writeError(w, http.StatusBadRequest, "project_id and seq are required")
		return tasks.Task{}, false
	}
	task, err := h.Tasks.GetByProjectSeq(r.Context(), projectID, seq)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return tasks.Task{}, false
	}
	if task.AssigneeActorID == 0 {
		writeError(w, http.StatusForbidden, "task is not assigned to this agent")
		return tasks.Task{}, false
	}
	// The pushing agent owns the task's session: its assignee, or the agent a
	// cross-agent continuation moved the session to.
	caller, err := h.Agents.Get(r.Context(), agentID)
	if err != nil ||
		(caller.ActorID != task.AssigneeActorID && task.SessionAgentID != agentID) {
		writeError(w, http.StatusForbidden, "task is not assigned to this agent")
		return tasks.Task{}, false
	}
	return task, true
}

// get returns a task's stored conversation. `?meta=1` reports only the tip and
// count, so a healthy agent can compare cheaply before pushing.
func (h *Handler) get(w http.ResponseWriter, r *http.Request, agentID ids.AgentID) {
	query := r.URL.Query()
	seq, _ := strconv.ParseInt(query.Get("seq"), 10, 64)
	task, ok := h.resolve(w, r, agentID, query.Get("project_id"), seq)
	if !ok {
		return
	}

	tip, count, err := h.Tasks.SessionHistoryTip(r.Context(), task.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	entries := []entryJSON{}
	if query.Get("meta") != "1" {
		list, err := h.Tasks.SessionHistory(r.Context(), task.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		entries = make([]entryJSON, len(list))
		for i, e := range list {
			entries[i] = entryJSON{
				ID:       e.EntryID,
				ParentID: e.ParentID,
				Type:     e.Type,
				Raw:      e.Raw,
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries":      entries,
		"tip_entry_id": tip,
		"count":        count,
	})
}

// postInput is the push body: a task reference and the entries the agent has
// that the agency may be missing.
type postInput struct {
	ProjectID string      `json:"project_id"`
	Seq       int64       `json:"seq"`
	Entries   []entryJSON `json:"entries"`
}

// post appends entries the agent reported. Duplicates are ignored, so a
// replayed push is safe.
func (h *Handler) post(w http.ResponseWriter, r *http.Request, agentID ids.AgentID) {
	var input postInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	task, ok := h.resolve(w, r, agentID, input.ProjectID, input.Seq)
	if !ok {
		return
	}

	entries := make([]tasks.SessionHistoryEntry, 0, len(input.Entries))
	for _, e := range input.Entries {
		if e.ID == "" || e.Raw == "" {
			continue
		}
		entries = append(entries, tasks.SessionHistoryEntry{
			EntryID:  e.ID,
			ParentID: e.ParentID,
			Type:     e.Type,
			Raw:      e.Raw,
		})
	}

	appended, err := h.Tasks.AppendSessionHistory(r.Context(), task.ID, entries)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tip, count, err := h.Tasks.SessionHistoryTip(r.Context(), task.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"appended":     appended,
		"tip_entry_id": tip,
		"count":        count,
	})
}

func decodeJSON(r *http.Request, out any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty request body")
	}
	return json.Unmarshal(body, out)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
