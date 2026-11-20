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

package agentconfig

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

const BundlePath = "/api/rest/agent-config"

const AdminPath = "/api/rest/agent-config/admin"

type Handler struct {
	Service *Service

	AgentActorID func(context.Context) string

	IsAdmin func(*http.Request) bool

	TaskScope TaskScope
}

const TaskHeader = "X-CherryPi-Task"

const ModelHeader = "X-CherryPi-Model"

type TaskScope func(
	ctx context.Context,
	agentActorID string,
	taskID string,
) (projectID string, teamIDs []string, ok bool)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == BundlePath:
		h.bundle(w, r)
	case r.URL.Path == AdminPath:
		if !h.adminAllowed(w, r) {
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.admin(w, r)
	case strings.HasPrefix(r.URL.Path, AdminPath+"/providers"):
		if !h.adminAllowed(w, r) {
			return
		}
		h.providers(w, r)
	case strings.HasPrefix(r.URL.Path, AdminPath+"/test/"):
		if !h.adminAllowed(w, r) {
			return
		}
		h.probe(w, r)
	case r.URL.Path == DiscoverPath:
		if !h.adminAllowed(w, r) {
			return
		}
		h.discover(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) adminAllowed(w http.ResponseWriter, r *http.Request) bool {
	if h.IsAdmin == nil || h.IsAdmin(r) {
		return true
	}
	writeError(w, http.StatusForbidden, "administrators only")
	return false
}

func (h *Handler) bundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	caller := Caller{}
	if h.AgentActorID != nil {
		caller.AgentActorID = h.AgentActorID(r.Context())
	}
	if caller.AgentActorID == "" {
		writeError(w, http.StatusUnauthorized, "agent identity is required")
		return
	}
	if taskID := r.URL.Query().Get("task"); taskID != "" {

		if h.TaskScope == nil {
			writeError(w, http.StatusForbidden, "task-scoped grants are not available")
			return
		}
		projectID, teamIDs, ok := h.TaskScope(r.Context(), caller.AgentActorID, taskID)
		if !ok {
			writeError(w, http.StatusForbidden, "this task does not belong to the calling agent")
			return
		}
		caller.ProjectID, caller.TeamIDs = projectID, teamIDs

		caller.TaskScoped = true
	}

	bundle, changed, err := h.Service.Bundle(r.Context(), caller, r.URL.Query().Get("revision"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !changed {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeJSON(w, http.StatusOK, bundle)
}

func (h *Handler) admin(w http.ResponseWriter, r *http.Request) {
	view, err := h.Service.Admin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *Handler) providers(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, AdminPath+"/providers")
	slug = strings.TrimPrefix(slug, "/")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "provider slug is required")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		if err := h.Service.DeleteProvider(r.Context(), slug); err != nil {
			writeConfigError(w, err)
			return
		}
		writeJSON(w, http.StatusNoContent, nil)
	case http.MethodPost, http.MethodPut:
		var input ProviderInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if input.Slug == "" {
			input.Slug = slug
		}
		if err := h.Service.SaveProvider(r.Context(), input); err != nil {
			writeConfigError(w, err)
			return
		}
		h.admin(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) probe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	slug := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, AdminPath+"/test/"))
	if slug == "" {
		writeError(w, http.StatusBadRequest, "provider slug is required")
		return
	}
	result, err := h.Service.Probe(r.Context(), slug)
	if err != nil {
		writeConfigError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) discover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input DiscoverInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.Service.Discover(r.Context(), input)
	if err != nil {
		writeConfigError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type AdminView struct {
	Revision  string         `json:"revision"`
	Providers []ProviderView `json:"providers"`

	Sealing bool `json:"sealing"`
}

type ProviderView struct {
	Slug     string            `json:"slug"`
	Name     string            `json:"name"`
	API      string            `json:"api"`
	BaseURL  string            `json:"baseUrl"`
	HasKey   bool              `json:"hasKey"`
	Headers  map[string]string `json:"headers"`
	Enabled  bool              `json:"enabled"`
	Models   []ModelView       `json:"models"`
	Grantees []Scope           `json:"grantees"`
}

type ModelView struct {
	Model
	Enabled bool `json:"enabled"`
}

func (s *Service) Admin(ctx context.Context) (*AdminView, error) {
	revision, err := s.store.AgentConfigRevision(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := s.store.ListModelProviders(ctx)
	if err != nil {
		return nil, err
	}
	models, err := s.store.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	grants, err := s.store.ListModelGrants(ctx)
	if err != nil {
		return nil, err
	}

	byProvider := map[int64][]ModelView{}
	for _, model := range models {
		byProvider[model.ProviderID] = append(byProvider[model.ProviderID], ModelView{
			Model:   modelOf(model),
			Enabled: model.Enabled != 0,
		})
	}
	grantees := map[int64][]Scope{}
	for _, grant := range grants {
		grantees[grant.ProviderID] = append(grantees[grant.ProviderID], Scope{
			Kind:    grant.ScopeKind,
			ID:      grant.ScopeID,
			ModelID: grant.ModelID,
		})
	}

	view := &AdminView{
		Revision:  revision,
		Providers: make([]ProviderView, 0, len(providers)),
		Sealing:   s.Enabled(),
	}
	for _, row := range providers {

		models := byProvider[row.ID]
		if models == nil {
			models = []ModelView{}
		}
		scopes := grantees[row.ID]
		if scopes == nil {
			scopes = []Scope{}
		}
		view.Providers = append(view.Providers, ProviderView{
			Slug:     row.Slug,
			Name:     row.Name,
			API:      row.Api,
			BaseURL:  row.BaseUrl,
			HasKey:   len(row.ApiKeySealed) > 0,
			Headers:  decodeMap(row.HeadersJson),
			Enabled:  row.Enabled != 0,
			Models:   models,
			Grantees: scopes,
		})
	}
	return view, nil
}

func writeConfigError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnknownProvider):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidConfig):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNoSecretKey):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
