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

package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	AdminPath = "/api/rest/integrations"

	ReportPath = "/api/rest/task-integrations"
)

type TaskScope func(
	ctx context.Context,
	agentActorID, taskID string,
) (projectID string, teamIDs []string, ok bool)

type Identity struct {
	ActorID int64
	Admin   bool
}

type Handler struct {
	Service *Service

	Identity func(*http.Request) (Identity, bool)

	AgentActorID func(context.Context) string

	TaskForAgent TaskScope

	TaskVisible func(ctx context.Context, identity Identity, taskID string) (bool, error)

	Groups func(ctx context.Context, agentActorID string) ([]string, error)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == AdminPath:
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.library(w, r)
	case strings.HasPrefix(r.URL.Path, AdminPath+"/servers"):
		h.servers(w, r)
	case strings.HasPrefix(r.URL.Path, AdminPath+"/skills"):
		h.skills(w, r)
	case strings.HasPrefix(r.URL.Path, AdminPath+"/sources"):
		h.sources(w, r)
	case strings.HasPrefix(r.URL.Path, AdminPath+"/git-credentials"):
		h.credentials(w, r)
	case r.URL.Path == AdminPath+"/git":
		h.gitFeature(w, r)
	case strings.HasPrefix(r.URL.Path, AdminPath+"/task/"):
		h.task(w, r)
	case r.URL.Path == ReportPath:
		h.report(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) library(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identity(w, r)
	if !ok {
		return
	}
	view, err := h.Service.Library(r.Context(), identity.ActorID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *Handler) servers(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identity(w, r)
	if !ok {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, AdminPath+"/servers")
	if handled := h.installRoute(w, r, identity, KindMCP, rest); handled {
		return
	}
	if rest == "" || rest == "/" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var input Input
		if !decodeBody(w, r, &input) {
			return
		}
		saved, err := h.Service.SaveServer(r.Context(), identity.ActorID, input)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
		return
	}
	id, ok := parseID(w, rest)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		entry, err := h.Service.GetServer(r.Context(), identity.ActorID, id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, entry)
	case http.MethodDelete:
		if err := h.Service.DeleteServer(r.Context(), identity.ActorID, id); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) skills(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identity(w, r)
	if !ok {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, AdminPath+"/skills")
	if handled := h.installRoute(w, r, identity, KindSkill, rest); handled {
		return
	}
	if handled := h.skillActions(w, r, identity, rest); handled {
		return
	}
	if rest == "" || rest == "/" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var input Input
		if !decodeBody(w, r, &input) {
			return
		}
		saved, err := h.Service.SaveSkill(r.Context(), identity.ActorID, input)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
		return
	}
	id, ok := parseID(w, rest)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		entry, err := h.Service.GetSkill(r.Context(), identity.ActorID, id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, entry)
	case http.MethodDelete:
		if err := h.Service.DeleteSkill(r.Context(), identity.ActorID, id); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) installRoute(
	w http.ResponseWriter,
	r *http.Request,
	identity Identity,
	kind, rest string,
) bool {
	trimmed := strings.Trim(rest, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[1] != "install" {
		return false
	}
	id, ok := parseID(w, "/"+parts[0])
	if !ok {
		return true
	}
	switch r.Method {
	case http.MethodPut:
		var body struct {
			Scope   Scope `json:"scope"`
			Enabled bool  `json:"enabled"`
		}
		if !decodeBody(w, r, &body) {
			return true
		}
		if err := h.Service.SetInstall(
			r.Context(), identity.ActorID, kind, id, body.Scope, body.Enabled, false,
		); err != nil {
			writeServiceError(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		var body struct {
			Scope Scope `json:"scope"`
		}
		if !decodeBody(w, r, &body) {
			return true
		}
		if err := h.Service.SetInstall(
			r.Context(), identity.ActorID, kind, id, body.Scope, false, true,
		); err != nil {
			writeServiceError(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
	return true
}

func (h *Handler) skillActions(
	w http.ResponseWriter,
	r *http.Request,
	identity Identity,
	rest string,
) bool {
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 {
		return false
	}
	id, ok := parseID(w, "/"+parts[0])
	if !ok {
		return true
	}
	switch parts[1] {
	case "detach":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return true
		}
		entry, err := h.Service.DetachSkill(r.Context(), identity.ActorID, id)
		if err != nil {
			writeServiceError(w, err)
			return true
		}
		writeJSON(w, http.StatusOK, entry)
		return true
	case "enabled":
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return true
		}
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if !decodeBody(w, r, &body) {
			return true
		}
		if err := h.Service.SetSkillEnabled(
			r.Context(), identity.ActorID, id, body.Enabled,
		); err != nil {
			writeServiceError(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	default:
		return false
	}
}

func (h *Handler) gitFeature(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	identity, ok := h.identity(w, r)
	if !ok {
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if err := h.Service.SetGitSources(r.Context(), identity.ActorID, body.Enabled); err != nil {
		writeServiceError(w, err)
		return
	}
	status, err := h.Service.GitStatus(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) credentials(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identity(w, r)
	if !ok {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, AdminPath+"/git-credentials")
	trimmed := strings.Trim(rest, "/")
	if trimmed == "" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var input CredentialInput
		if !decodeBody(w, r, &input) {
			return
		}
		saved, err := h.Service.SaveCredential(r.Context(), identity.ActorID, input)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
		return
	}
	if trimmed == "scan" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var body struct {
			Host string `json:"host"`
			Port int64  `json:"port"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		keys, knownHosts, err := h.Service.ScanHostKeys(
			r.Context(), identity.ActorID, body.Host, body.Port,
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"keys":       keys,
			"knownHosts": knownHosts,
		})
		return
	}
	id, ok := parseID(w, "/"+trimmed)
	if !ok {
		return
	}
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := h.Service.DeleteCredential(r.Context(), identity.ActorID, id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) sources(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identity(w, r)
	if !ok {
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, AdminPath+"/sources"), "/")
	if rest == "" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var input SourceInput
		if !decodeBody(w, r, &input) {
			return
		}
		saved, err := h.Service.SaveSource(r.Context(), identity.ActorID, input)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
		return
	}
	if rest == "preview" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var input SourceInput
		if !decodeBody(w, r, &input) {
			return
		}
		preview, err := h.Service.PreviewSource(r.Context(), identity.ActorID, input)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, preview)
		return
	}
	parts := strings.Split(rest, "/")
	id, ok := parseID(w, "/"+parts[0])
	if !ok {
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			source, err := h.Service.GetSource(r.Context(), identity.ActorID, id)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, source)
		case http.MethodDelete:
			if err := h.Service.DeleteSource(r.Context(), identity.ActorID, id); err != nil {
				writeServiceError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	switch parts[1] {
	case "sync", "apply":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var (
			source Source
			err    error
		)
		if parts[1] == "sync" {
			source, err = h.Service.SyncSource(r.Context(), identity.ActorID, id)
		} else {
			source, err = h.Service.ApplySource(r.Context(), identity.ActorID, id)
		}
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, source)
	case "install":
		switch r.Method {
		case http.MethodPut:
			var body struct {
				Scope   Scope `json:"scope"`
				Enabled bool  `json:"enabled"`
			}
			if !decodeBody(w, r, &body) {
				return
			}
			if err := h.Service.SetSourceInstall(
				r.Context(), identity.ActorID, id, body.Scope, body.Enabled, false,
			); err != nil {
				writeServiceError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			var body struct {
				Scope Scope `json:"scope"`
			}
			if !decodeBody(w, r, &body) {
				return
			}
			if err := h.Service.SetSourceInstall(
				r.Context(), identity.ActorID, id, body.Scope, false, true,
			); err != nil {
				writeServiceError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) task(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	identity, ok := h.identity(w, r)
	if !ok {
		return
	}
	taskID := strings.TrimPrefix(r.URL.Path, AdminPath+"/task/")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "a task id is required")
		return
	}
	if h.TaskVisible != nil {
		visible, err := h.TaskVisible(r.Context(), identity, taskID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if !visible {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
	}
	report, err := h.Service.TaskReport(r.Context(), taskID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) report(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	agentActorID := ""
	if h.AgentActorID != nil {
		agentActorID = h.AgentActorID(r.Context())
	}
	if agentActorID == "" {
		writeError(w, http.StatusUnauthorized, "agent identity is required")
		return
	}
	var body struct {
		TaskID   string `json:"taskId"`
		Revision string `json:"revision"`
		Report   Report `json:"report"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if h.TaskForAgent == nil {
		writeError(w, http.StatusForbidden, "task-scoped reports are not available")
		return
	}
	if _, _, ok := h.TaskForAgent(r.Context(), agentActorID, body.TaskID); !ok {
		writeError(w, http.StatusForbidden, "this task does not belong to the calling agent")
		return
	}
	if err := h.Service.PutReport(
		r.Context(),
		body.TaskID,
		body.Revision,
		body.Report,
	); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) identity(w http.ResponseWriter, r *http.Request) (Identity, bool) {
	if h.Identity == nil {
		writeError(w, http.StatusUnauthorized, "authentication is required")
		return Identity{}, false
	}
	identity, ok := h.Identity(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication is required")
		return Identity{}, false
	}
	return identity, true
}

func parseID(w http.ResponseWriter, rest string) (int64, bool) {
	id, err := strconv.ParseInt(strings.Trim(rest, "/"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "a numeric id is required")
		return 0, false
	}
	return id, true
}

func decodeBody(w http.ResponseWriter, r *http.Request, into any) bool {
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound), IsNoRows(err):
		writeError(w, http.StatusNotFound, "library entry not found")
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrFeatureDisabled):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrGitUnavailable):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Service) CallerFor(
	ctx context.Context,
	agentActorID, projectID string,
	teamIDs []string,
	groups func(context.Context, string) ([]string, error),
) (Caller, error) {
	caller := Caller{AgentActorID: agentActorID, ProjectID: projectID, TeamIDs: teamIDs}
	if groups != nil {
		resolved, err := groups(ctx, agentActorID)
		if err != nil {
			return Caller{}, err
		}
		caller.GroupIDs = resolved
	}
	return caller, nil
}

func ActorIDOf(raw string) ids.ActorID {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return ids.ActorID(value)
}

var _ = store.McpServer{}
