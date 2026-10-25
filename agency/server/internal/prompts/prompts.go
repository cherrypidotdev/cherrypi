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
)

type Prompt struct {
	ID        int64
	Content   string
	Version   int64
	UpdatedAt time.Time
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
	"Reach for these only when they make the answer clearer; plain prose is fine for everything else."

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
	return s.set(ctx, scope, agentID, 0, projectID, content)
}

func (s *Service) SetForTeam(
	ctx context.Context,
	scope Scope,
	teamID int64, projectID, content string,
) (Prompt, error) {
	return s.set(ctx, scope, 0, teamID, projectID, content)
}

func (s *Service) set(
	ctx context.Context,
	scope Scope,
	agentID ids.AgentID, teamID int64, projectID, content string,
) (Prompt, error) {
	if err := scope.validate(agentID, teamID, projectID); err != nil {
		return Prompt{}, err
	}
	cur, err := s.get(ctx, scope, agentID, teamID, projectID)
	if err != nil {
		return Prompt{}, err
	}
	if cur.Content == content {
		return cur, nil
	}

	version, err := s.store.LatestPromptVersion(ctx, store.LatestPromptVersionParams{
		Scope:     string(scope),
		AgentID:   sql.NullInt64{Int64: int64(agentID), Valid: true},
		TeamID:    sql.NullInt64{Int64: teamID, Valid: true},
		ProjectID: sql.NullString{String: projectID, Valid: true},
	})
	if err != nil {
		return Prompt{}, err
	}
	row, err := s.store.InsertPrompt(ctx, store.InsertPromptParams{
		Scope:     string(scope),
		AgentID:   agentID,
		TeamID:    teamID,
		ProjectID: projectID,
		Content:   content,
		Version:   version + 1,
		CreatedAt: time.Now().UnixMilli(),
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
		ID:        row.ID,
		Content:   row.Content,
		Version:   row.Version,
		UpdatedAt: time.UnixMilli(row.CreatedAt),
	}
}
