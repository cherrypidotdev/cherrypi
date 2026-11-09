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

package prompts

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

type Scope string

const (
	ScopeOrg          Scope = "org"
	ScopeAgent        Scope = "agent"
	ScopeProject      Scope = "project"
	ScopeAgentProject Scope = "agent_project"
	ScopeTeam         Scope = "team"
	ScopeTeamProject  Scope = "team_project"
)

var (
	ErrInvalidScope = errors.New("invalid prompt scope")

	ErrInvalidTarget = errors.New("prompt scope requires the matching agent and project ids")

	ErrNoChange = errors.New("the proposed content does not change the prompt")

	ErrProposalNotFound = errors.New("prompt proposal not found")

	ErrProposalNotPending = errors.New("prompt proposal is not pending")

	ErrProposalStale = errors.New("prompt proposal is stale")
)

type Prompt struct {
	ID        int64
	Content   string
	Version   int64
	UpdatedAt time.Time

	AuthorActorID ids.ActorID

	ApprovedByActorID ids.ActorID

	Note string

	ProposalID int64

	AutoApproved bool
}

type ProposalStatus string

const (
	ProposalPending   ProposalStatus = "pending"
	ProposalApproved  ProposalStatus = "approved"
	ProposalRejected  ProposalStatus = "rejected"
	ProposalWithdrawn ProposalStatus = "withdrawn"
)

type Proposal struct {
	ID               int64
	Scope            Scope
	AgentID          ids.AgentID
	TeamID           int64
	ProjectID        string
	Content          string
	BaseVersion      int64
	AuthorActorID    ids.ActorID
	Note             string
	Status           ProposalStatus
	AutoApproved     bool
	CreatedAt        time.Time
	DecidedAt        time.Time
	DecidedByActorID ids.ActorID
	DecisionNote     string

	Stale bool

	LatestVersion int64
}

type SetParams struct {
	Scope     Scope
	AgentID   ids.AgentID
	TeamID    int64
	ProjectID string
	Content   string

	AuthorActorID ids.ActorID

	ApprovedByActorID ids.ActorID

	Note string

	ProposalID int64

	AutoApproved bool
}

type CreateProposalParams struct {
	Scope         Scope
	AgentID       ids.AgentID
	TeamID        int64
	ProjectID     string
	Content       string
	AuthorActorID ids.ActorID
	Note          string
}

type ProposalFilter struct {
	Scope     Scope
	AgentID   ids.AgentID
	TeamID    int64
	ProjectID string
	Status    ProposalStatus

	MineActorID ids.ActorID
	Limit       int
	BeforeID    int64
}

type TeamVersionIDs struct {
	TeamID        int64
	Prompt        int64
	ProjectPrompt int64
}

type VersionIDs struct {
	Org          int64
	Agent        int64
	Project      int64
	AgentProject int64
	Teams        []TeamVersionIDs
}

type Resolved struct {
	IDs  VersionIDs
	Text string
}

const BuiltinGuidance = "# Presentation\n\n" +
	"Prefer rich, visual answers when they make the result clearer. The CherryPi dashboard renders Markdown plus a safe subset of HTML:\n\n" +
	"- Use HTML for structure and emphasis: `<details>`/`<summary>` for collapsible sections, `<table>` for tabular data, `<kbd>` for keybindings, `<mark>` for highlights, and `<sub>`/`<sup>` where notation calls for them.\n" +
	"- Fence diagrams in a ```mermaid block (flowchart, sequence, class, state, ER, gantt, …); the dashboard draws them as diagrams.\n" +
	"- Otherwise use Markdown as usual for prose, lists, code and links.\n\n" +
	"Reach for these only when they make the answer clearer; plain prose is fine for everything else.\n\n" +
	"# Planning\n\n" +
	"Every task starts with a plan, and the plan is part of the work, not a report about it. Before your first tool call — and always before doing the work — publish a short plan with `reportTaskProgress`: 3–8 ordered items, each `pending` (a genuinely one-step task gets one item). Updating the plan is bookkeeping, never a stopping point: call the tool and carry on with the work in the same turn, and only end the run when the task is actually done. Then keep it current:\n\n" +
	"- At the end of every turn in which you used tools, update the plan: mark finished items `done`, the step you are on `in_progress`, and the next ones `pending`.\n" +
	"- Update the one-line summary when what you are doing changes, so the operator can see it at a glance.\n" +
	"- Plans are expected to change — when new user input, a CI result, or another external event invalidates a step, replace the list instead of forcing the old plan.\n" +
	"- Before you finish a run, reconcile the plan: everything you completed is `done`, nothing you finished is left `pending`, and a task that turned out different gets a new list. A run does not settle while the plan is missing, unfinished, or stale."

type Service struct {
	store store.Store
}

func New(st store.Store) *Service { return &Service{store: st} }

func (s *Service) Get(
	ctx context.Context,
	scope Scope,
	agentID ids.AgentID, projectID string,
) (Prompt, error) {
	return s.get(ctx, scope, agentID, 0, projectID)
}

func (s *Service) GetAt(
	ctx context.Context,
	scope Scope,
	agentID ids.AgentID, teamID int64, projectID string,
) (Prompt, error) {
	return s.get(ctx, scope, agentID, teamID, projectID)
}

func (s *Service) GetForTeam(
	ctx context.Context,
	scope Scope,
	teamID int64, projectID string,
) (Prompt, error) {
	return s.get(ctx, scope, 0, teamID, projectID)
}

func (s *Service) get(
	ctx context.Context,
	scope Scope,
	agentID ids.AgentID, teamID int64, projectID string,
) (Prompt, error) {
	if err := scope.validate(agentID, teamID, projectID); err != nil {
		return Prompt{}, err
	}
	row, err := s.store.LatestPrompt(ctx, store.LatestPromptParams{
		Scope:     string(scope),
		AgentID:   sql.NullInt64{Int64: int64(agentID), Valid: true},
		TeamID:    sql.NullInt64{Int64: teamID, Valid: true},
		ProjectID: sql.NullString{String: projectID, Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Prompt{}, nil
	}
	if err != nil {
		return Prompt{}, err
	}
	return toPrompt(row), nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (Prompt, error) {
	if id == 0 {
		return Prompt{}, nil
	}
	row, err := s.store.GetPrompt(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Prompt{}, nil
	}
	if err != nil {
		return Prompt{}, err
	}
	return toPrompt(row), nil
}

func (s *Service) Set(
	ctx context.Context,
	scope Scope,
	agentID ids.AgentID, projectID, content string,
) (Prompt, error) {
	return s.SetVersion(ctx, SetParams{
		Scope:     scope,
		AgentID:   agentID,
		ProjectID: projectID,
		Content:   content,
	})
}

func (s *Service) SetForTeam(
	ctx context.Context,
	scope Scope,
	teamID int64, projectID, content string,
) (Prompt, error) {
	return s.SetVersion(ctx, SetParams{
		Scope:     scope,
		TeamID:    teamID,
		ProjectID: projectID,
		Content:   content,
	})
}

func (s *Service) SetVersion(ctx context.Context, p SetParams) (Prompt, error) {
	if err := p.Scope.validate(p.AgentID, p.TeamID, p.ProjectID); err != nil {
		return Prompt{}, err
	}
	cur, err := s.get(ctx, p.Scope, p.AgentID, p.TeamID, p.ProjectID)
	if err != nil {
		return Prompt{}, err
	}
	if cur.Content == p.Content {
		return cur, nil
	}

	version, err := s.store.LatestPromptVersion(ctx, store.LatestPromptVersionParams{
		Scope:     string(p.Scope),
		AgentID:   sql.NullInt64{Int64: int64(p.AgentID), Valid: true},
		TeamID:    sql.NullInt64{Int64: p.TeamID, Valid: true},
		ProjectID: sql.NullString{String: p.ProjectID, Valid: true},
	})
	if err != nil {
		return Prompt{}, err
	}
	autoApproved := int64(0)
	if p.AutoApproved {
		autoApproved = 1
	}
	row, err := s.store.InsertPrompt(ctx, store.InsertPromptParams{
		Scope:             string(p.Scope),
		AgentID:           p.AgentID,
		TeamID:            p.TeamID,
		ProjectID:         p.ProjectID,
		Content:           p.Content,
		Version:           version + 1,
		CreatedAt:         time.Now().UnixMilli(),
		AuthorActorID:     int64(p.AuthorActorID),
		ApprovedByActorID: int64(p.ApprovedByActorID),
		Note:              p.Note,
		ProposalID:        p.ProposalID,
		AutoApproved:      autoApproved,
	})
	if err != nil {
		return Prompt{}, err
	}
	return toPrompt(row), nil
}

func (s *Service) Resolve(
	ctx context.Context,
	agentID ids.AgentID,
	projectID string,
	teamIDs ...int64,
) (Resolved, error) {
	org, err := s.Get(ctx, ScopeOrg, 0, "")
	if err != nil {
		return Resolved{}, err
	}
	var agent Prompt
	if agentID != 0 {
		if agent, err = s.Get(ctx, ScopeAgent, agentID, ""); err != nil {
			return Resolved{}, err
		}
	}
	var project, agentProject Prompt
	if projectID != "" {
		if project, err = s.Get(ctx, ScopeProject, 0, projectID); err != nil {
			return Resolved{}, err
		}
		if agentProject, err = s.Get(ctx, ScopeAgentProject, agentID, projectID); err != nil {
			return Resolved{}, err
		}
	}

	teamIDs = dedupeSorted(teamIDs)
	teamPins := make([]TeamVersionIDs, 0, len(teamIDs))
	teamParts := make([]string, 0, len(teamIDs)*2)
	for _, teamID := range teamIDs {
		if teamID == 0 {
			continue
		}
		general, err := s.GetForTeam(ctx, ScopeTeam, teamID, "")
		if err != nil {
			return Resolved{}, err
		}
		pin := TeamVersionIDs{TeamID: teamID, Prompt: general.ID}
		teamParts = append(teamParts, general.Content)
		if projectID != "" {
			perProject, err := s.GetForTeam(ctx, ScopeTeamProject, teamID, projectID)
			if err != nil {
				return Resolved{}, err
			}
			pin.ProjectPrompt = perProject.ID
			teamParts = append(teamParts, perProject.Content)
		}
		teamPins = append(teamPins, pin)
	}

	parts := make([]string, 0, 6+len(teamParts))
	parts = append(parts, org.Content, project.Content)
	parts = append(parts, teamParts...)
	parts = append(parts,
		agent.Content,
		agentProject.Content,
		BuiltinGuidance,
	)
	return Resolved{
		IDs: VersionIDs{
			Org:          org.ID,
			Agent:        agent.ID,
			Project:      project.ID,
			AgentProject: agentProject.ID,
			Teams:        teamPins,
		},
		Text: joinNonEmpty(parts...),
	}, nil
}

func dedupeSorted(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	out := append([]int64(nil), values...)
	slices.Sort(out)
	n := 0
	for _, v := range out {
		if v == 0 || (n > 0 && out[n-1] == v) {
			continue
		}
		out[n] = v
		n++
	}
	return out[:n]
}

func (s *Service) ListVersions(
	ctx context.Context,
	scope Scope,
	agentID ids.AgentID, teamID int64, projectID string,
	limit int, beforeVersion int64,
) ([]Prompt, error) {
	if err := scope.validate(agentID, teamID, projectID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.store.ListPromptVersions(ctx, store.ListPromptVersionsParams{
		Scope:         string(scope),
		AgentID:       sql.NullInt64{Int64: int64(agentID), Valid: true},
		TeamID:        sql.NullInt64{Int64: teamID, Valid: true},
		ProjectID:     sql.NullString{String: projectID, Valid: true},
		BeforeVersion: beforeVersion,
		MaxCount:      int64(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Prompt, len(rows))
	for i, row := range rows {
		out[i] = toPrompt(row)
	}
	return out, nil
}

func (s *Service) CreateProposal(
	ctx context.Context,
	p CreateProposalParams,
) (Proposal, error) {
	if err := p.Scope.validate(p.AgentID, p.TeamID, p.ProjectID); err != nil {
		return Proposal{}, err
	}
	cur, err := s.get(ctx, p.Scope, p.AgentID, p.TeamID, p.ProjectID)
	if err != nil {
		return Proposal{}, err
	}
	if cur.Content == p.Content {
		return Proposal{}, ErrNoChange
	}
	row, err := s.store.InsertPromptProposal(
		ctx,
		store.InsertPromptProposalParams{
			Scope:         string(p.Scope),
			AgentID:       p.AgentID,
			TeamID:        p.TeamID,
			ProjectID:     p.ProjectID,
			Content:       p.Content,
			BaseVersion:   cur.Version,
			AuthorActorID: int64(p.AuthorActorID),
			Note:          p.Note,
			CreatedAt:     time.Now().UnixMilli(),
		},
	)
	if err != nil {
		return Proposal{}, err
	}
	out := toProposal(row)
	out.LatestVersion = cur.Version
	return out, nil
}

func (s *Service) GetProposal(ctx context.Context, id int64) (Proposal, error) {
	row, err := s.store.GetPromptProposal(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Proposal{}, ErrProposalNotFound
	}
	if err != nil {
		return Proposal{}, err
	}
	out := toProposal(row)
	if err := s.fillProposalState(ctx, &out); err != nil {
		return Proposal{}, err
	}
	return out, nil
}

func (s *Service) ListProposals(
	ctx context.Context,
	f ProposalFilter,
) ([]Proposal, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	var (
		rows []store.PromptProposal
		err  error
	)

	status := sql.NullString{String: string(f.Status), Valid: f.Status != ""}
	if f.MineActorID != 0 {
		rows, err = s.store.ListPromptProposalsByAuthor(
			ctx,
			store.ListPromptProposalsByAuthorParams{
				AuthorActorID: int64(f.MineActorID),
				Status:        status,
				BeforeID:      f.BeforeID,
				MaxCount:      int64(f.Limit),
			},
		)
	} else {
		if err := f.Scope.validate(f.AgentID, f.TeamID, f.ProjectID); err != nil {
			return nil, err
		}
		rows, err = s.store.ListPromptProposalsByTarget(
			ctx,
			store.ListPromptProposalsByTargetParams{
				Scope:     string(f.Scope),
				AgentID:   sql.NullInt64{Int64: int64(f.AgentID), Valid: true},
				TeamID:    sql.NullInt64{Int64: f.TeamID, Valid: true},
				ProjectID: sql.NullString{String: f.ProjectID, Valid: true},
				Status:    status,
				BeforeID:  f.BeforeID,
				MaxCount:  int64(f.Limit),
			},
		)
	}
	if err != nil {
		return nil, err
	}
	out := make([]Proposal, len(rows))
	for i, row := range rows {
		out[i] = toProposal(row)
		if err := s.fillProposalState(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) ApproveProposal(
	ctx context.Context,
	id int64, approverActorID ids.ActorID, note string, autoApproved bool,
) (Prompt, Proposal, error) {
	prompt, row, err := s.store.ApprovePromptProposal(
		ctx,
		store.ApprovePromptProposalParams{
			ID:               id,
			DecidedByActorID: approverActorID,
			DecisionNote:     note,
			AutoApproved:     autoApproved,
			Now:              time.Now().UnixMilli(),
		},
	)
	switch {
	case errors.Is(err, store.ErrPromptProposalNotFound):
		return Prompt{}, Proposal{}, ErrProposalNotFound
	case errors.Is(err, store.ErrPromptProposalNotPending):
		return Prompt{}, Proposal{}, ErrProposalNotPending
	case errors.Is(err, store.ErrPromptProposalStale):
		return Prompt{}, Proposal{}, ErrProposalStale
	case err != nil:
		return Prompt{}, Proposal{}, err
	}
	out := toProposal(row)
	out.LatestVersion = prompt.Version
	return toPrompt(prompt), out, nil
}

func (s *Service) RejectProposal(
	ctx context.Context,
	id int64, approverActorID ids.ActorID, note string,
) (Proposal, error) {
	row, err := s.store.SetPromptProposalDecision(
		ctx,
		store.SetPromptProposalDecisionParams{
			Status:           string(ProposalRejected),
			DecidedAt:        sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			DecidedByActorID: int64(approverActorID),
			DecisionNote:     note,
			AutoApproved:     0,
			ID:               id,
		},
	)
	if errors.Is(err, sql.ErrNoRows) {
		if _, getErr := s.GetProposal(ctx, id); errors.Is(getErr, ErrProposalNotFound) {
			return Proposal{}, ErrProposalNotFound
		}
		return Proposal{}, ErrProposalNotPending
	}
	if err != nil {
		return Proposal{}, err
	}
	out := toProposal(row)
	if err := s.fillProposalState(ctx, &out); err != nil {
		return Proposal{}, err
	}
	return out, nil
}

func (s *Service) WithdrawProposal(
	ctx context.Context,
	id int64, authorActorID ids.ActorID, note string,
) (Proposal, error) {
	row, err := s.store.WithdrawPromptProposal(
		ctx,
		store.WithdrawPromptProposalParams{
			DecidedAt:     sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			DecisionNote:  note,
			ID:            id,
			AuthorActorID: int64(authorActorID),
		},
	)
	if errors.Is(err, sql.ErrNoRows) {
		if _, getErr := s.GetProposal(ctx, id); errors.Is(getErr, ErrProposalNotFound) {
			return Proposal{}, ErrProposalNotFound
		}
		return Proposal{}, ErrProposalNotPending
	}
	if err != nil {
		return Proposal{}, err
	}
	out := toProposal(row)
	if err := s.fillProposalState(ctx, &out); err != nil {
		return Proposal{}, err
	}
	return out, nil
}

func (s *Service) fillProposalState(ctx context.Context, p *Proposal) error {
	latest, err := s.store.LatestPromptVersion(ctx, store.LatestPromptVersionParams{
		Scope:     string(p.Scope),
		AgentID:   sql.NullInt64{Int64: int64(p.AgentID), Valid: true},
		TeamID:    sql.NullInt64{Int64: p.TeamID, Valid: true},
		ProjectID: sql.NullString{String: p.ProjectID, Valid: true},
	})
	if err != nil {
		return err
	}
	p.LatestVersion = latest
	p.Stale = p.Status == ProposalPending && latest != p.BaseVersion
	return nil
}

func (sc Scope) TeamScoped() bool {
	return sc == ScopeTeam || sc == ScopeTeamProject
}

func (sc Scope) valid() bool {
	switch sc {
	case ScopeOrg, ScopeAgent, ScopeProject, ScopeAgentProject,
		ScopeTeam, ScopeTeamProject:
		return true
	default:
		return false
	}
}

func (sc Scope) validate(agentID ids.AgentID, teamID int64, projectID string) error {
	if !sc.valid() {
		return ErrInvalidScope
	}
	needAgent := sc == ScopeAgent || sc == ScopeAgentProject
	needTeam := sc == ScopeTeam || sc == ScopeTeamProject
	needProject := sc == ScopeProject || sc == ScopeAgentProject ||
		sc == ScopeTeamProject
	if needAgent != (agentID != 0) ||
		needTeam != (teamID != 0) ||
		needProject != (projectID != "") {
		return ErrInvalidTarget
	}
	return nil
}
func joinNonEmpty(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "\n\n")
}

func toPrompt(row store.Prompt) Prompt {
	return Prompt{
		ID:                row.ID,
		Content:           row.Content,
		Version:           row.Version,
		UpdatedAt:         time.UnixMilli(row.CreatedAt),
		AuthorActorID:     ids.ActorID(row.AuthorActorID.Int64),
		ApprovedByActorID: ids.ActorID(row.ApprovedByActorID.Int64),
		Note:              row.Note,
		ProposalID:        row.ProposalID.Int64,
		AutoApproved:      row.AutoApproved != 0,
	}
}

func toProposal(row store.PromptProposal) Proposal {
	p := Proposal{
		ID:               row.ID,
		Scope:            Scope(row.Scope),
		AgentID:          ids.AgentID(row.AgentID.Int64),
		TeamID:           row.TeamID.Int64,
		ProjectID:        row.ProjectID.String,
		Content:          row.Content,
		BaseVersion:      row.BaseVersion,
		AuthorActorID:    ids.ActorID(row.AuthorActorID),
		Note:             row.Note,
		Status:           ProposalStatus(row.Status),
		AutoApproved:     row.AutoApproved != 0,
		CreatedAt:        time.UnixMilli(row.CreatedAt),
		DecidedByActorID: ids.ActorID(row.DecidedByActorID.Int64),
		DecisionNote:     row.DecisionNote,
	}
	if row.DecidedAt.Valid {
		p.DecidedAt = time.UnixMilli(row.DecidedAt.Int64)
	}
	return p
}
