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

package taskhistory

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/ids"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

const Prefix = "/api/rest/task-history"

type Handler struct {
	Tasks  *tasks.Service
	Agents *agents.Registry
}

type entryJSON struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Type     string `json:"type"`
	Raw      string `json:"raw"`
}

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

func (h *Handler) resolve(
	w http.ResponseWriter,
	r *http.Request,
	agentID ids.AgentID, projectID string,
	seq int64,
) (tasks.Task, bool) {
	if strings.TrimSpace(projectID) == "" || seq <= 0 {
		log.Printf("[task-history] reject agent=%d: project_id and seq are required", agentID)
		writeError(w, http.StatusBadRequest, "project_id and seq are required")
		return tasks.Task{}, false
	}
	task, err := h.Tasks.GetByProjectSeq(r.Context(), projectID, seq)
	if err != nil {
		log.Printf(
			"[task-history] reject agent=%d project=%s seq=%d: task not found (%v)",
			agentID, projectID, seq, err,
		)
		writeError(w, http.StatusNotFound, "task not found")
		return tasks.Task{}, false
	}
	if task.AssigneeActorID == 0 {
		log.Printf("[task-history] reject agent=%d task=%s: task has no assignee", agentID, task.ID)
		writeError(w, http.StatusForbidden, "task is not assigned to this agent")
		return tasks.Task{}, false
	}

	caller, err := h.Agents.Get(r.Context(), agentID)
	if err != nil ||
		(caller.ActorID != task.AssigneeActorID && task.SessionAgentID != agentID) {
		log.Printf(
			"[task-history] reject agent=%d task=%s: not the task's agent (assignee=%d session_agent=%d)",
			agentID,
			task.ID,
			task.AssigneeActorID,
			task.SessionAgentID,
		)
		writeError(w, http.StatusForbidden, "task is not assigned to this agent")
		return tasks.Task{}, false
	}
	return task, true
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, agentID ids.AgentID) {
	query := r.URL.Query()
	seq, _ := strconv.ParseInt(query.Get("seq"), 10, 64)
	task, ok := h.resolve(w, r, agentID, query.Get("project_id"), seq)
	if !ok {
		return
	}

	tip, count, err := h.Tasks.SessionHistoryTip(r.Context(), task.ID)
	if err != nil {
		log.Printf("[task-history] fetch task=%s agent=%d: tip: %v", task.ID, agentID, err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	entries := []entryJSON{}
	if query.Get("meta") != "1" {
		list, err := h.Tasks.SessionHistory(r.Context(), task.ID)
		if err != nil {
			log.Printf("[task-history] fetch task=%s agent=%d: %v", task.ID, agentID, err)
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
	log.Printf(
		"[task-history] fetch task=%s agent=%d entries=%d tip=%s meta=%t",
		task.ID, agentID, count, tip, query.Get("meta") == "1",
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"entries":      entries,
		"tip_entry_id": tip,
		"count":        count,
	})
}

type postInput struct {
	ProjectID string      `json:"project_id"`
	Seq       int64       `json:"seq"`
	Entries   []entryJSON `json:"entries"`
}

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
		log.Printf(
			"[task-history] push task=%s agent=%d entries=%d: append: %v",
			task.ID, agentID, len(entries), err,
		)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tip, count, err := h.Tasks.SessionHistoryTip(r.Context(), task.ID)
	if err != nil {
		log.Printf("[task-history] push task=%s agent=%d: tip: %v", task.ID, agentID, err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf(
		"[task-history] push task=%s agent=%d received=%d appended=%d tip=%s count=%d",
		task.ID, agentID, len(input.Entries), appended, tip, count,
	)
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
