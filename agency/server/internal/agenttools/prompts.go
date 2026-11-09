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
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/promptaccess"
	"cherrypi.vladfaust.com/agency/internal/promptflow"
	"cherrypi.vladfaust.com/agency/internal/prompts"
)

const (
	promptsPath          = Prefix + "/prompts"
	promptVersionsPath   = Prefix + "/prompts/versions"
	promptProposalsPath  = Prefix + "/prompts/proposals"
	promptProposalsIDPre = promptProposalsPath + "/"
)

type promptJSON struct {
	ID           int64            `json:"id"`
	Scope        string           `json:"scope"`
	Content      string           `json:"content"`
	Version      int64            `json:"version"`
	UpdatedAt    string           `json:"updated_at,omitempty"`
	Author       *promptActorJSON `json:"author,omitempty"`
	ApprovedBy   *promptActorJSON `json:"approved_by,omitempty"`
	Note         string           `json:"note,omitempty"`
	ProposalID   int64            `json:"proposal_id,omitempty"`
	AutoApproved bool             `json:"auto_approved,omitempty"`
	CanEdit      bool             `json:"can_edit"`
	CanPropose   bool             `json:"can_propose"`
	CanApprove   bool             `json:"can_approve"`
}

type promptActorJSON struct {
	ActorID  string `json:"actor_id"`
	Username string `json:"username,omitempty"`
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

type proposalJSON struct {
	ID            int64            `json:"id"`
	Scope         string           `json:"scope"`
	AgentID       string           `json:"agent_id,omitempty"`
	ProjectID     string           `json:"project_id,omitempty"`
	TeamID        int64            `json:"team_id,omitempty"`
	Content       string           `json:"content"`
	BaseVersion   int64            `json:"base_version"`
	Author        *promptActorJSON `json:"author,omitempty"`
	Note          string           `json:"note,omitempty"`
	Status        string           `json:"status"`
	CreatedAt     string           `json:"created_at,omitempty"`
	DecidedAt     string           `json:"decided_at,omitempty"`
	DecidedBy     *promptActorJSON `json:"decided_by,omitempty"`
	DecisionNote  string           `json:"decision_note,omitempty"`
	Stale         bool             `json:"stale"`
	AutoApproved  bool             `json:"auto_approved"`
	LatestVersion int64            `json:"latest_version"`
	IsMine        bool             `json:"is_mine"`
}

func (h *Handler) getPrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	identity, target, ok := h.promptTarget(w, r, false)
	if !ok {
		return
	}
	p, access, err := h.PromptFlow.Get(r.Context(), identity, target)
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"prompt": h.toPromptJSON(r.Context(), p, access),
	})
}

func (h *Handler) listPromptVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	_, target, ok := h.promptTarget(w, r, false)
	if !ok {
		return
	}
	limit := intQuery(r, "limit", 50, 200)
	before, _ := strconv.ParseInt(r.URL.Query().Get("before_version"), 10, 64)
	versions, err := h.PromptFlow.ListVersions(r.Context(), target, limit, before)
	if err != nil {
		writePromptError(w, err)
		return
	}
	out := make([]promptJSON, len(versions))
	for i, p := range versions {
		out[i] = h.toPromptJSON(r.Context(), p, promptflow.Access{})
	}
	writeJSON(w, http.StatusOK, map[string]any{"versions": out})
}

func (h *Handler) listPromptProposals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	query := promptflow.ProposalQuery{
		Mine:   true,
		Limit:  intQuery(r, "limit", 50, 200),
		Status: prompts.ProposalStatus(strings.TrimSpace(r.URL.Query().Get("status"))),
	}
	query.Before, _ = strconv.ParseInt(r.URL.Query().Get("before_id"), 10, 64)
	if scope := strings.TrimSpace(r.URL.Query().Get("scope")); scope != "" {
		target, err := h.agentPromptTarget(r.Context(), identity, scope,
			strings.TrimSpace(r.URL.Query().Get("project_id")))
		if err != nil {
			writePromptError(w, err)
			return
		}
		query.Target = target
		if strings.TrimSpace(r.URL.Query().Get("mine")) != "true" {
			query.Mine = false
		}
	}
	list, err := h.PromptFlow.ListProposals(r.Context(), identity, query)
	if err != nil {
		writePromptError(w, err)
		return
	}
	out := make([]proposalJSON, len(list))
	for i, p := range list {
		out[i] = h.toProposalJSON(r.Context(), p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"proposals": out})
}

type createProposalInput struct {
	Scope     string `json:"scope"`
	ProjectID string `json:"project_id"`
	Content   string `json:"content"`
	Note      string `json:"note"`
}

func (h *Handler) createPromptProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input createProposalInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	target, err := h.agentPromptTarget(
		r.Context(),
		identity,
		strings.TrimSpace(input.Scope),
		strings.TrimSpace(input.ProjectID),
	)
	if err != nil {
		writePromptError(w, err)
		return
	}
	proposal, err := h.PromptFlow.CreateProposal(
		r.Context(),
		identity,
		target,
		input.Content,
		input.Note,
	)
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"proposal": h.toProposalJSON(r.Context(), proposal),
	})
}

type withdrawProposalInput struct {
	Note string `json:"note"`
}

func (h *Handler) withdrawPromptProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	raw := strings.TrimPrefix(r.URL.Path, promptProposalsIDPre)
	idPart, action, found := strings.Cut(raw, "/")
	if !found || action != "withdraw" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid proposal id")
		return
	}
	var input withdrawProposalInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	proposal, err := h.PromptFlow.WithdrawProposal(r.Context(), identity, id, input.Note)
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"proposal": h.toProposalJSON(r.Context(), proposal),
	})
}

func (h *Handler) promptTarget(
	w http.ResponseWriter,
	r *http.Request,
	requireProject bool,
) (auth.Identity, promptaccess.Target, bool) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return auth.Identity{}, promptaccess.Target{}, false
	}
	target, err := h.agentPromptTarget(
		r.Context(),
		identity,
		strings.TrimSpace(r.URL.Query().Get("scope")),
		strings.TrimSpace(r.URL.Query().Get("project_id")),
	)
	if err != nil {
		writePromptError(w, err)
		return auth.Identity{}, promptaccess.Target{}, false
	}
	if requireProject && target.ProjectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return auth.Identity{}, promptaccess.Target{}, false
	}
	return identity, target, true
}

func (h *Handler) agentPromptTarget(
	ctx context.Context,
	identity auth.Identity,
	rawScope, projectID string,
) (promptaccess.Target, error) {
	agent, err := h.PromptFlow.Access.AgentForActor(ctx, identity.ActorID)
	if err != nil {
		return promptaccess.Target{}, err
	}
	switch strings.TrimSpace(rawScope) {
	case "agent":
		return promptaccess.Target{
			Scope:   prompts.ScopeAgent,
			AgentID: agent.ID,
		}, nil
	case "agent_project", "agent-project":
		if projectID == "" {
			return promptaccess.Target{}, errProjectRequired
		}
		resolved, err := h.resolvePromptProject(ctx, projectID)
		if err != nil {
			return promptaccess.Target{}, err
		}
		return promptaccess.Target{
			Scope:     prompts.ScopeAgentProject,
			AgentID:   agent.ID,
			ProjectID: resolved,
		}, nil
	case "project":
		if projectID == "" {
			return promptaccess.Target{}, errProjectRequired
		}
		resolved, err := h.resolvePromptProject(ctx, projectID)
		if err != nil {
			return promptaccess.Target{}, err
		}
		return promptaccess.Target{
			Scope:     prompts.ScopeProject,
			ProjectID: resolved,
		}, nil
	default:
		return promptaccess.Target{}, errInvalidPromptScope
	}
}

func (h *Handler) resolvePromptProject(
	ctx context.Context,
	projectID string,
) (string, error) {
	project, err := h.Projects.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return project.ID, nil
}

var (
	errProjectRequired    = errors.New("project_id is required for this scope")
	errInvalidPromptScope = errors.New("scope must be agent, agent_project, or project")
)

func (h *Handler) toPromptJSON(
	ctx context.Context,
	p prompts.Prompt,
	access promptflow.Access,
) promptJSON {
	out := promptJSON{
		ID:           p.ID,
		Content:      p.Content,
		Version:      p.Version,
		Author:       h.promptActor(ctx, p.AuthorActorID),
		ApprovedBy:   h.promptActor(ctx, p.ApprovedByActorID),
		Note:         p.Note,
		ProposalID:   p.ProposalID,
		AutoApproved: p.AutoApproved,
		CanEdit:      access.CanEdit,
		CanPropose:   access.CanPropose,
		CanApprove:   access.CanApprove,
	}
	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = p.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func (h *Handler) toProposalJSON(ctx context.Context, p prompts.Proposal) proposalJSON {
	out := proposalJSON{
		ID:            p.ID,
		Scope:         string(p.Scope),
		AgentID:       actorid.Format(p.AgentID),
		ProjectID:     p.ProjectID,
		TeamID:        p.TeamID,
		Content:       p.Content,
		BaseVersion:   p.BaseVersion,
		Author:        h.promptActor(ctx, p.AuthorActorID),
		Note:          p.Note,
		Status:        string(p.Status),
		DecidedBy:     h.promptActor(ctx, p.DecidedByActorID),
		DecisionNote:  p.DecisionNote,
		Stale:         p.Stale,
		AutoApproved:  p.AutoApproved,
		LatestVersion: p.LatestVersion,
		IsMine:        callerActorID(ctx) == p.AuthorActorID,
	}
	if !p.CreatedAt.IsZero() {
		out.CreatedAt = p.CreatedAt.UTC().Format(time.RFC3339)
	}
	if !p.DecidedAt.IsZero() {
		out.DecidedAt = p.DecidedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func callerActorID(ctx context.Context) ids.ActorID {
	id, _ := auth.ActorID(ctx)
	return id
}

func (h *Handler) promptActor(ctx context.Context, id ids.ActorID) *promptActorJSON {
	if id == 0 {
		return nil
	}
	out := &promptActorJSON{ActorID: actorid.Format(id)}
	if h.PromptFlow.Accounts == nil {
		return out
	}
	actor, err := h.PromptFlow.Accounts.GetActor(ctx, id)
	if err != nil {
		return out
	}
	out.Username = actor.Username
	out.Name = actor.Name
	out.Kind = actor.Kind
	return out
}

func writePromptError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, projects.ErrProjectMissing):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, promptflow.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, prompts.ErrInvalidScope),
		errors.Is(err, prompts.ErrInvalidTarget),
		errors.Is(err, errProjectRequired),
		errors.Is(err, errInvalidPromptScope):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, prompts.ErrNoChange):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, prompts.ErrProposalNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, prompts.ErrProposalNotPending),
		errors.Is(err, prompts.ErrProposalStale):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
