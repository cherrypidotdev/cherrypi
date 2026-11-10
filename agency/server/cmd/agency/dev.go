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

//go:build dev

package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/cmd/agency/appcontext"
	"cherrypi.vladfaust.com/agency/internal/auth"
	pluginhost "cherrypi.vladfaust.com/agency/internal/plugins"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

// proxy the frontend to the Vite dev server via CHERRYPI_FRONTEND_PROXY_URL and

const devBuild = true

// frontendFS returns nothing in dev — the frontend is served by Vite, so no

func frontendFS() (fs.FS, error) {
	return nil, fs.ErrNotExist
}

// docsFS returns nothing in dev — the docs are served by VitePress through

func docsFS() (fs.FS, error) {
	return nil, fs.ErrNotExist
}

const (
	devTaskPath  = "/api/dev/plugins/tasks"
	devEventPath = "/api/dev/plugins/events"
	devTickPath  = "/api/dev/plugins/tick"
)

var devSeedStates = map[string]bool{
	string(tasks.StateBacklog):   true,
	string(tasks.StateQueued):    true,
	string(tasks.StateRunning):   true,
	string(tasks.StateInput):     true,
	string(tasks.StateSettled):   true,
	string(tasks.StateFailed):    true,
	string(tasks.StateCompleted): true,
	string(tasks.StateCancelled): true,
}

func serveDev(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	h := &devHandler{app: appCtx}
	mux.HandleFunc("POST "+devTaskPath, h.seedTask)
	mux.HandleFunc("POST "+devEventPath, h.dispatchEvent)
	mux.HandleFunc("POST "+devTickPath, h.invokeTick)
}

type devHandler struct {
	app *appcontext.AppContext
}

func (h *devHandler) seedTask(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	var req struct {
		ProjectID       string `json:"project_id"`
		Name            string `json:"name"`
		Prompt          string `json:"prompt"`
		State           string `json:"state"`
		AssigneeActorID string `json:"assignee_actor_id"`
	}
	if err := decodeDevBody(r, &req); err != nil {
		writeDevError(w, http.StatusBadRequest, err)
		return
	}
	ctx := r.Context()
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		writeDevError(w, http.StatusBadRequest, errors.New("prompt is required"))
		return
	}
	state := normalizeDevState(req.State)
	if state == "" {
		state = string(tasks.StateBacklog)
	}
	if !devSeedStates[state] {
		writeDevError(w, http.StatusBadRequest, errors.New("unknown task state "+state))
		return
	}
	project, err := h.app.Projects.Get(ctx, req.ProjectID)
	if err != nil {
		writeDevError(w, http.StatusBadRequest, err)
		return
	}
	number, err := h.app.Store.AllocateTaskNumber(ctx, project.ID)
	if err != nil {
		writeDevError(w, http.StatusInternalServerError, err)
		return
	}
	now := time.Now().UnixMilli()
	params := store.CreateTaskParams{
		ID:          tasks.TaskID(project.Prefix, number),
		Seq:         number,
		Prompt:      prompt,
		Name:        strings.TrimSpace(req.Name),
		ProjectID:   project.ID,
		State:       state,
		Size:        tasks.SizeMedium,
		Interactive: 1,
		QueuedAt:    devSeedTimestamp(state, string(tasks.StateBacklog), now),
		StartedAt:   devSeedTimestamp(state, string(tasks.StateQueued), now),
		SettledAt:   devSeedTimestamp(state, string(tasks.StateRunning), now),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if actorID, ok := auth.ActorID(ctx); ok {
		params.CreatorActorID = sql.NullInt64{Int64: int64(actorID), Valid: true}
	}
	if raw := strings.TrimSpace(req.AssigneeActorID); raw != "" {
		assignee, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || assignee <= 0 {
			writeDevError(w, http.StatusBadRequest, errors.New("invalid assignee_actor_id"))
			return
		}
		params.AssigneeActorID = sql.NullInt64{Int64: assignee, Valid: true}
		params.AssigneePinned = 1
	}
	if _, err := h.app.Store.CreateTask(ctx, params); err != nil {
		writeDevError(w, http.StatusInternalServerError, err)
		return
	}

	if params.CreatorActorID.Valid {
		if err := h.app.Store.AddTaskLead(ctx, store.AddTaskLeadParams{
			TaskID:  params.ID,
			ActorID: params.CreatorActorID.Int64,
			AddedAt: now,
		}); err != nil {
			writeDevError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeDevJSON(w, http.StatusOK, map[string]any{
		"task_id": params.ID,
		"state":   state,
	})
}

func (h *devHandler) dispatchEvent(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	if h.app.Plugins == nil {
		writeDevError(w, http.StatusServiceUnavailable, errors.New("plugins are not running"))
		return
	}
	var req struct {
		Name      string         `json:"name"`
		TaskID    string         `json:"task_id"`
		ProjectID string         `json:"project_id"`
		Payload   map[string]any `json:"payload"`
	}
	if err := decodeDevBody(r, &req); err != nil {
		writeDevError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeDevError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	delivered, err := h.app.Plugins.DispatchEventWait(r.Context(), pluginhost.Event{
		Name:      name,
		TaskID:    strings.TrimSpace(req.TaskID),
		ProjectID: strings.TrimSpace(req.ProjectID),
		Extra:     req.Payload,
	})
	if err != nil {
		writeDevJSON(w, http.StatusInternalServerError, map[string]any{
			"delivered": delivered,
			"error":     err.Error(),
		})
		return
	}
	writeDevJSON(w, http.StatusOK, map[string]any{"delivered": delivered})
}

func (h *devHandler) invokeTick(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	if h.app.Plugins == nil {
		writeDevError(w, http.StatusServiceUnavailable, errors.New("plugins are not running"))
		return
	}
	var req struct {
		PluginID string `json:"plugin_id"`
		NowMs    int64  `json:"now_ms"`
	}
	if err := decodeDevBody(r, &req); err != nil {
		writeDevError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(req.PluginID)
	if pluginID == "" {
		writeDevError(w, http.StatusBadRequest, errors.New("plugin_id is required"))
		return
	}
	if err := h.app.Plugins.InvokeTick(r.Context(), pluginID, req.NowMs); err != nil {
		writeDevError(w, http.StatusBadRequest, err)
		return
	}
	writeDevJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *devHandler) authorize(w http.ResponseWriter, r *http.Request) bool {
	if auth.IsAdmin(r.Context()) {
		return true
	}
	writeDevError(w, http.StatusForbidden, errors.New("the dev endpoint requires an administrator"))
	return false
}

var devStateRank = map[string]int{
	string(tasks.StateBacklog):   0,
	string(tasks.StateQueued):    1,
	string(tasks.StateRunning):   2,
	string(tasks.StateInput):     2,
	string(tasks.StateSettled):   3,
	string(tasks.StateFailed):    3,
	string(tasks.StateCompleted): 3,
	string(tasks.StateCancelled): 3,
}

func devSeedTimestamp(state, after string, now int64) sql.NullInt64 {
	if devStateRank[state] > devStateRank[after] {
		return sql.NullInt64{Int64: now, Valid: true}
	}
	return sql.NullInt64{}
}

func normalizeDevState(value string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(value), "TASK_STATE_"))
}

func decodeDevBody(r *http.Request, out any) error {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func writeDevJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeDevError(w http.ResponseWriter, status int, err error) {
	writeDevJSON(w, status, map[string]any{"error": err.Error()})
}
