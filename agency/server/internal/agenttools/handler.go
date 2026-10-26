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
	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/agentlogs"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/processlog"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/taskhistory"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
)

const Prefix = "/api/rest/agent-tools"

const TokenPath = Prefix + "/token"

const ResolvePath = Prefix + "/tasks/resolve"

const bootstrapTokenName = "agent-bootstrap"

type Handler struct {
	Tasks    *tasks.Service
	Projects *projects.Service
	Agents   *agents.Registry

	Accounts *accounts.Service

	Logs *agentlogs.Service

	ProcessLogs *processlog.Log
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == TokenPath:
		h.token(w, r)
	case path == ResolvePath:
		h.resolveTask(w, r)
	case path == Prefix+"/tasks":
		switch r.Method {
		case http.MethodGet:
			h.listTasks(w, r)
		case http.MethodPost:
			h.createTask(w, r)
		default:
			methodNotAllowed(w)
		}
	case strings.HasPrefix(path, Prefix+"/tasks/"):
		h.taskByID(w, r)
	case path == Prefix+"/projects":
		h.listProjects(w, r)
	case path == Prefix+"/agency":
		h.agencyMeta(w, r)
	case path == Prefix+"/agency/logs":
		h.agencyLogs(w, r)
	case path == Prefix+"/agents":
		h.listFleet(w, r)
	case strings.HasPrefix(path, Prefix+"/agents/"):
		h.agentLogs(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	agentID, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.Accounts == nil {
		writeError(w, http.StatusServiceUnavailable, "token service is unavailable")
		return
	}
	agent, err := h.Agents.Get(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	if pats, err := h.Accounts.ListPATs(r.Context(), agent.ActorID); err == nil {
		for _, pat := range pats {
			if pat.Name == bootstrapTokenName {
				_ = h.Accounts.RevokePAT(r.Context(), agent.ActorID, pat.ID)
			}
		}
	}

	token, _, err := h.Accounts.CreatePAT(r.Context(), agent.ActorID, bootstrapTokenName, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token})
}

func (h *Handler) resolveTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	caller, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	query := r.URL.Query()
	projectID := strings.TrimSpace(query.Get("project_id"))
	seq, err := strconv.ParseInt(strings.TrimSpace(query.Get("seq")), 10, 64)
	if projectID == "" || err != nil || seq <= 0 {
		writeError(w, http.StatusBadRequest, "project_id and seq are required")
		return
	}
	task, err := h.Tasks.GetByProjectSeq(r.Context(), projectID, seq)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	callerAgent, err := h.Agents.Get(r.Context(), caller)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if task.AssigneeActorID != callerAgent.ActorID {
		writeError(w, http.StatusForbidden, "task is not assigned to this agent")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_id": task.ID})
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	caller, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	query := r.URL.Query()
	projectID := strings.TrimSpace(query.Get("project_id"))
	agentFilter := actorid.ParseAgent(strings.TrimSpace(query.Get("agent_id")))

	filter := tasks.Filter{
		ProjectID: projectID,
		AgentID:   agentFilter,
		State:     tasks.State(strings.TrimSpace(query.Get("state"))),
		TagID:     strings.TrimSpace(query.Get("tag_id")),
		Search:    strings.TrimSpace(query.Get("search")),
		Size:      strings.TrimSpace(query.Get("size")),
	}

	if projectID == "" {

		filter.AgentID = caller
	} else {

		project, err := h.Projects.Get(r.Context(), projectID)
		if errors.Is(err, projects.ErrProjectMissing) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		filter.ProjectID = project.ID
		_, member, err := h.Projects.Permissions(r.Context(), project.ID, caller)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !member {
			writeError(w, http.StatusForbidden, "not a member of the project")
			return
		}
	}

	list, err := h.Tasks.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]taskJSON, len(list))
	for i, t := range list {
		out[i] = toTaskJSON(t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": out})
}

func (h *Handler) taskByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, Prefix+"/tasks/")
	if id, isHistory := strings.CutSuffix(rest, "/history"); isHistory {
		if id == "" || strings.ContainsRune(id, '/') {
			http.NotFound(w, r)
			return
		}
		h.getTaskHistory(w, r, id)
		return
	}
	if rest == "" || strings.ContainsRune(rest, '/') {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getTask(w, r, rest)
	case http.MethodPatch:
		h.updateTask(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request, id string) {
	caller, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	task, err := h.Tasks.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	readable, err := h.taskReadable(r.Context(), caller, task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !readable {
		writeError(w, http.StatusForbidden, "missing can_list_tasks permission")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": toTaskJSON(task)})
}

func (h *Handler) getTaskHistory(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	caller, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	task, err := h.Tasks.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	readable, err := h.taskReadable(r.Context(), caller, task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !readable {
		writeError(w, http.StatusForbidden, "missing can_list_tasks permission")
		return
	}

	query := r.URL.Query()
	var before int64
	if raw := strings.TrimSpace(query.Get("cursor")); raw != "" {
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || before < 0 {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
	}
	limit := 0
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}

	entries, err := h.Tasks.SessionHistory(r.Context(), task.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	events, next := taskhistory.Page(task.ID, entries, before, limit)
	out := make([]historyEventJSON, len(events))
	for i, event := range events {
		out[i] = toHistoryEventJSON(event)
	}
	nextCursor := ""
	if next > 0 {
		nextCursor = strconv.FormatInt(next, 10)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id":     task.ID,
		"events":      out,
		"next_cursor": nextCursor,
	})
}

func (h *Handler) taskReadable(
	ctx context.Context,
	caller ids.AgentID,
	task tasks.Task,
) (bool, error) {
	if callerAgent, err := h.Agents.Get(ctx, caller); err == nil &&
		task.AssigneeActorID != 0 && task.AssigneeActorID == callerAgent.ActorID {
		return true, nil
	}
	if strings.TrimSpace(task.ProjectID) == "" {
		return false, nil
	}
	perms, member, err := h.Projects.Permissions(ctx, task.ProjectID, caller)
	if errors.Is(err, projects.ErrProjectMissing) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return member && perms.CanListTasks, nil
}

type historyEventJSON struct {
	Type       string `json:"type"`
	At         string `json:"at,omitempty"`
	Text       string `json:"text,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	Phase      string `json:"phase,omitempty"`
	Args       string `json:"args,omitempty"`
	Result     string `json:"result,omitempty"`
	IsError    bool   `json:"is_error,omitempty"`
}

func toHistoryEventJSON(event *agentv1.TaskEvent) historyEventJSON {
	out := historyEventJSON{}
	if at := event.GetAt(); at != nil {
		out.At = at.AsTime().UTC().Format(time.RFC3339Nano)
	}
	switch payload := event.GetEvent().(type) {
	case *agentv1.TaskEvent_UserMessage:
		out.Type = "user_message"
		out.Text = payload.UserMessage.GetText()
	case *agentv1.TaskEvent_Text:
		out.Type = "text"
		out.Text = payload.Text.GetDelta()
	case *agentv1.TaskEvent_Thinking:
		out.Type = "thinking"
		out.Text = payload.Thinking.GetDelta()
	case *agentv1.TaskEvent_Tool:
		out.Type = "tool_call"
		out.ToolCallID = payload.Tool.GetToolCallId()
		out.ToolName = payload.Tool.GetToolName()
		out.Phase = toolPhaseName(payload.Tool.GetPhase())
		out.Args = payload.Tool.GetArgsJson()
		out.Result = payload.Tool.GetResultJson()
		out.IsError = payload.Tool.GetIsError()
	}
	return out
}

func toolPhaseName(phase agentv1.ToolPhase) string {
	switch phase {
	case agentv1.ToolPhase_TOOL_PHASE_STARTED:
		return "started"
	case agentv1.ToolPhase_TOOL_PHASE_UPDATED:
		return "updated"
	case agentv1.ToolPhase_TOOL_PHASE_FINISHED:
		return "finished"
	default:
		return "unspecified"
	}
}

type referenceInput struct {
	TaskID string `json:"task_id"`
	Kind   string `json:"kind"`

	Blocking bool `json:"blocking"`
}

type createTaskInput struct {
	ProjectID       string         `json:"project_id"`
	Prompt          string         `json:"prompt"`
	Name            string         `json:"name"`
	AssigneeActorID actorid.WireID `json:"assignee_actor_id"`

	AssigneeAgentID actorid.WireID   `json:"assignee_agent_id"`
	TagIDs          []string         `json:"tag_ids"`
	References      []referenceInput `json:"references"`
	Priority        *int64           `json:"priority"`
	Size            string           `json:"size"`

	State string `json:"state"`
	Draft bool   `json:"draft"`
}

func (h *Handler) resolveAssignee(
	ctx context.Context,
	actor, legacy actorid.WireID,
) (ids.ActorID, bool, error) {
	if actor.Set() {
		return actor.ID(), true, nil
	}
	if !legacy.Set() {
		return 0, false, nil
	}
	agentID := legacy.AgentID()
	if agentID == 0 {

		return 0, true, nil
	}
	agent, err := h.Agents.Get(ctx, agentID)
	if err != nil {
		return 0, false, fmt.Errorf("unknown assignee agent id %d", agentID)
	}
	return agent.ActorID, true, nil
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	caller, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input createTaskInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}
	perms, member, err := h.Projects.Permissions(r.Context(), projectID, caller)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !member {
		writeError(w, http.StatusForbidden, "not a member of the project")
		return
	}
	if !perms.CanCreateTasks {
		writeError(w, http.StatusForbidden, "missing can_create_tasks permission")
		return
	}

	priority := tasks.PriorityNormal
	if input.Priority != nil {
		priority = *input.Priority
	}
	assignee, _, err := h.resolveAssignee(
		r.Context(),
		input.AssigneeActorID,
		input.AssigneeAgentID,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := taskActorContext(r)
	state, ok := agentCreateState(input.State, input.Draft)
	if !ok {
		writeError(w, http.StatusBadRequest, "state must be draft or backlog")
		return
	}
	task, err := h.Tasks.Create(ctx, tasks.CreateParams{
		Prompt:          input.Prompt,
		Name:            input.Name,
		ProjectID:       projectID,
		AssigneeActorID: assignee,
		TagIDs:          input.TagIDs,
		References:      referenceInputs(input.References),
		Priority:        priority,
		Size:            input.Size,
		State:           state,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": toTaskJSON(task)})
}

func taskActorContext(r *http.Request) context.Context {
	ctx := tasks.WithActor(r.Context(), tasks.ActorAgent)
	if id, ok := auth.ActorID(r.Context()); ok {
		ctx = tasks.WithActorID(ctx, id)
	}
	return ctx
}

func agentCreateState(raw string, draft bool) (tasks.State, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "draft":
		return tasks.StateDraft, true
	case "backlog":
		return tasks.StateBacklog, true
	case "":
		if draft {
			return tasks.StateDraft, true
		}
		return "", false
	default:
		return "", false
	}
}

type updateTaskInput struct {
	Name            *string           `json:"name"`
	Links           *[]linkJSON       `json:"links"`
	TagIDs          *[]string         `json:"tag_ids"`
	References      *[]referenceInput `json:"references"`
	Priority        *int64            `json:"priority"`
	Size            *string           `json:"size"`
	AssigneeActorID actorid.WireID    `json:"assignee_actor_id"`

	AssigneeAgentID actorid.WireID `json:"assignee_agent_id"`
}

type linkJSON struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		methodNotAllowed(w)
		return
	}
	caller, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := strings.CutPrefix(r.URL.Path, Prefix+"/tasks/")
	if !ok || id == "" || strings.ContainsRune(id, '/') {
		http.NotFound(w, r)
		return
	}

	task, err := h.Tasks.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	callerAgent, agentErr := h.Agents.Get(r.Context(), caller)
	if agentErr != nil || task.AssigneeActorID != callerAgent.ActorID {
		perms, member, err := h.Projects.Permissions(r.Context(), task.ProjectID, caller)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !member || !perms.CanUpdateTasks {
			writeError(w, http.StatusForbidden, "missing can_update_tasks permission")
			return
		}
	}

	var input updateTaskInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	assignee, setAssignee, err := h.resolveAssignee(
		r.Context(),
		input.AssigneeActorID,
		input.AssigneeAgentID,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := taskActorContext(r)
	if input.Name != nil {
		if task, err = h.Tasks.Rename(ctx, id, *input.Name); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	attrs := tasks.AttributeSet{
		Priority: input.Priority,
		Size:     input.Size,
	}
	if setAssignee {
		attrs.AssigneeActorID = &assignee
	}
	if !attrs.Empty() {
		if task, err = h.Tasks.UpdateAttributes(ctx, id, attrs); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if input.Links != nil {
		links := make([]tasks.Link, len(*input.Links))
		for i, l := range *input.Links {
			links[i] = tasks.Link{URL: l.URL, Label: l.Label}
		}
		if task, err = h.Tasks.SetLinks(ctx, id, links); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if input.TagIDs != nil {
		if err := h.Tasks.ValidateTags(ctx, task.ProjectID, *input.TagIDs); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.Tasks.SetTags(ctx, id, *input.TagIDs); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if task, err = h.Tasks.Get(ctx, id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if input.References != nil {
		if task, err = h.Tasks.SetReferences(
			ctx,
			id,
			referenceInputs(*input.References),
		); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": toTaskJSON(task)})
}

type permissionsJSON struct {
	CanListTasks   bool `json:"can_list_tasks"`
	CanCreateTasks bool `json:"can_create_tasks"`
	CanUpdateTasks bool `json:"can_update_tasks"`
}

type memberJSON struct {
	AgentID     ids.AgentID     `json:"agent_id"`
	ActorID     ids.ActorID     `json:"actor_id"`
	Name        string          `json:"name"`
	Username    string          `json:"username"`
	Description string          `json:"description"`
	Permissions permissionsJSON `json:"permissions"`
}

type projectJSON struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	Prefix          string          `json:"prefix"`
	Description     string          `json:"description"`
	ParentProjectID string          `json:"parent_project_id,omitempty"`
	Archived        bool            `json:"archived"`
	Permissions     permissionsJSON `json:"permissions"`
	Members         []memberJSON    `json:"members"`
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	caller, ok := auth.AgentID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	memberships, err := h.Projects.ListAgentMemberships(r.Context(), caller)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]projectJSON, len(memberships))
	for i, m := range memberships {
		out[i] = projectJSON{
			ID:              m.Project.ID,
			Name:            m.Project.Name,
			Slug:            m.Project.Slug,
			Prefix:          m.Project.Prefix,
			Description:     m.Project.Description,
			ParentProjectID: m.Project.ParentID,
			Archived:        !m.Project.ArchivedAt.IsZero(),
			Permissions:     toPermissionsJSON(m.Permissions),
			Members:         h.members(r, m.Project.ID),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": out})
}

func (h *Handler) members(r *http.Request, projectID string) []memberJSON {
	list, err := h.Projects.ListAgents(r.Context(), projectID)
	if err != nil {
		return []memberJSON{}
	}
	out := make([]memberJSON, len(list))
	for i, member := range list {
		out[i] = memberJSON{
			AgentID:     member.AgentID,
			Permissions: toPermissionsJSON(member.Permissions),
		}
		if agent, err := h.Agents.Get(r.Context(), member.AgentID); err == nil {
			out[i].ActorID = agent.ActorID
			out[i].Name = agent.DisplayName
			out[i].Username = agent.Username
			out[i].Description = agent.Description
		}
	}
	return out
}

type taskJSON struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Prompt       string          `json:"prompt"`
	ProjectID    string          `json:"project_id"`
	AssigneeID   string          `json:"assignee_actor_id"`
	State        string          `json:"state"`
	Priority     string          `json:"priority"`
	Size         string          `json:"size"`
	Tags         []string        `json:"tags"`
	References   []referenceJSON `json:"references"`
	ReferencedBy []referenceJSON `json:"referenced_by"`
	Links        []linkJSON      `json:"links"`
	CreatedAt    string          `json:"created_at"`
}

type referenceJSON struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
	Kind  string `json:"kind"`
}

func referenceInputs(in []referenceInput) []tasks.ReferenceInput {
	out := make([]tasks.ReferenceInput, 0, len(in))
	for _, ref := range in {
		kind := tasks.ReferenceKind(strings.TrimSpace(ref.Kind))
		if kind == "" {
			kind = tasks.ReferenceInformational
			if ref.Blocking {
				kind = tasks.ReferenceBlocking
			}
		}
		out = append(out, tasks.ReferenceInput{
			TaskID: strings.TrimSpace(ref.TaskID),
			Kind:   kind,
		})
	}
	return out
}

func toTaskJSON(t tasks.Task) taskJSON {
	tags := make([]string, len(t.Tags))
	for i, tag := range t.Tags {
		tags[i] = tag.Name
	}
	refs := make([]referenceJSON, len(t.References))
	for i, ref := range t.References {
		refs[i] = referenceJSON{
			ID:    ref.ID,
			Name:  ref.Name,
			State: string(ref.State),
			Kind:  string(ref.Kind),
		}
	}
	incoming := make([]referenceJSON, len(t.ReferencedBy))
	for i, ref := range t.ReferencedBy {
		incoming[i] = referenceJSON{
			ID:    ref.ID,
			Name:  ref.Name,
			State: string(ref.State),
			Kind:  string(ref.Kind),
		}
	}
	links := make([]linkJSON, len(t.Links))
	for i, link := range t.Links {
		links[i] = linkJSON{URL: link.URL, Label: link.Label}
	}
	out := taskJSON{
		ID:           t.ID,
		Name:         t.Name,
		Prompt:       t.Prompt,
		ProjectID:    t.ProjectID,
		AssigneeID:   actorid.Format(t.AssigneeActorID),
		State:        string(t.State),
		Priority:     priorityName(t.Priority),
		Size:         t.Size,
		Tags:         tags,
		References:   refs,
		ReferencedBy: incoming,
		Links:        links,
	}
	if !t.CreatedAt.IsZero() {
		out.CreatedAt = t.CreatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func priorityName(priority int64) string {
	switch priority {
	case tasks.PriorityLow:
		return "low"
	case tasks.PriorityHigh:
		return "high"
	case tasks.PriorityCritical:
		return "critical"
	default:
		return "normal"
	}
}

func toPermissionsJSON(p projects.Permissions) permissionsJSON {
	return permissionsJSON{
		CanListTasks:   p.CanListTasks,
		CanCreateTasks: p.CanCreateTasks,
		CanUpdateTasks: p.CanUpdateTasks,
	}
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid JSON body: " + err.Error())
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid JSON body: unexpected trailing data")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
