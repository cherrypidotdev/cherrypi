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

// Package promptsgrpc implements agency.v1.PromptService over the versioned
// prompt store.
package promptsgrpc

import (
	"context"
	"errors"
	"fmt"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/settings"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/teams"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var errTeamPromptOwner = errors.New("team owner or administrator required")

// PromptServer implements agency.v1.PromptService.
type PromptServer struct {
	Prompts  *prompts.Service
	Projects *projects.Service
	Agents   *agents.Registry
	Tasks    *tasks.Service
	Settings *settings.Service
	Teams    *teams.Service
}

func (s *PromptServer) GetPrompt(
	ctx context.Context,
	req *agencyv1.GetPromptRequest,
) (*agencyv1.GetPromptResponse, error) {
	scope, err := fromProtoScope(req.Scope)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var p prompts.Prompt
	if scope.TeamScoped() {
		if err := s.authorizeTeamPrompt(ctx, scope, req.TeamId, false); err != nil {
			return nil, err
		}
		p, err = s.Prompts.GetForTeam(ctx, scope, req.TeamId, req.ProjectId)
	} else {
		p, err = s.Prompts.Get(ctx, scope, actorid.ParseAgent(req.AgentId), req.ProjectId)
	}
	if err != nil {
		return nil, promptError(err)
	}
	return &agencyv1.GetPromptResponse{Prompt: toProto(p)}, nil
}

func (s *PromptServer) SetPrompt(
	ctx context.Context,
	req *agencyv1.SetPromptRequest,
) (*agencyv1.SetPromptResponse, error) {
	scope, err := fromProtoScope(req.Scope)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.validateTargets(
		ctx,
		scope,
		actorid.ParseAgent(req.AgentId),
		req.TeamId,
		req.ProjectId,
	); err != nil {
		return nil, err
	}
	var p prompts.Prompt
	if scope.TeamScoped() {
		if err := s.authorizeTeamPrompt(ctx, scope, req.TeamId, true); err != nil {
			return nil, err
		}
		p, err = s.Prompts.SetForTeam(ctx, scope, req.TeamId, req.ProjectId, req.Content)
	} else {
		p, err = s.Prompts.Set(
			ctx,
			scope,
			actorid.ParseAgent(req.AgentId),
			req.ProjectId,
			req.Content,
		)
	}
	if err != nil {
		return nil, promptError(err)
	}
	return &agencyv1.SetPromptResponse{Prompt: toProto(p)}, nil
}

func (s *PromptServer) ListAgentProjectPrompts(
	ctx context.Context,
	req *agencyv1.ListAgentProjectPromptsRequest,
) (*agencyv1.ListAgentProjectPromptsResponse, error) {
	// Only the agent's member projects are relevant: it can only run sessions in
	// those.
	list, err := s.Projects.ListAgentProjects(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*agencyv1.AgentProjectPrompt, len(list))
	for i, project := range list {
		p, err := s.Prompts.Get(
			ctx,
			prompts.ScopeAgentProject,
			actorid.ParseAgent(req.AgentId),
			project.ID,
		)
		if err != nil {
			return nil, promptError(err)
		}
		out[i] = &agencyv1.AgentProjectPrompt{
			Project: projectToProto(project),
			Prompt:  toProto(p),
		}
	}
	return &agencyv1.ListAgentProjectPromptsResponse{Prompts: out}, nil
}

// GetTaskSystemPrompt reports the scoped prompts a task applies, in injection
// order. A task that has started reports the exact versions it pinned; an
// unstarted one previews the latest versions it would start with.
func (s *PromptServer) GetTaskSystemPrompt(
	ctx context.Context,
	req *agencyv1.GetTaskSystemPromptRequest,
) (*agencyv1.GetTaskSystemPromptResponse, error) {
	task, err := s.Tasks.Get(ctx, req.TaskId)
	if err != nil {
		if errors.Is(err, tasks.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// A started task reports the versions it pinned; an unstarted one previews
	// the latest versions the scheduler would start it with.
	pinned := !task.StartedAt.IsZero()
	pinnedIDs := task.PromptIDs
	agent, agentErr := s.Agents.GetByActor(ctx, task.AssigneeActorID)
	if agentErr != nil {
		return nil, connect.NewError(connect.CodeInternal, agentErr)
	}
	if !pinned {
		resolved, err := s.Prompts.Resolve(
			ctx,
			agent.ID,
			task.ProjectID,
			task.TeamIDs...,
		)
		if err != nil {
			return nil, promptError(err)
		}
		pinnedIDs = resolved.IDs
	}

	agencyName, err := s.settingsName(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	agentName := s.agentName(ctx, agent.ID)
	projectName := s.projectName(ctx, task.ProjectID)

	sections := make([]*agencyv1.SystemPromptSection, 0, 6+2*len(pinnedIDs.Teams))
	appendSection := func(scope prompts.Scope, id int64, owner string) error {
		p, err := s.Prompts.GetByID(ctx, id)
		if err != nil {
			return err
		}
		sections = append(sections, &agencyv1.SystemPromptSection{
			Scope:   scopeToProto(scope),
			Owner:   owner,
			Version: p.Version,
			Content: p.Content,
			Pinned:  pinned,
		})
		return nil
	}

	// Injection order: org, project, teams (general then per-project), agent,
	// agent-project. Scopes that do not apply to the task are omitted; an
	// applicable scope with no prompt still renders (version 0, empty content)
	// so its absence is visible.
	if err := appendSection(prompts.ScopeOrg, pinnedIDs.Org, agencyName); err != nil {
		return nil, promptError(err)
	}
	if task.ProjectID != "" {
		if err := appendSection(prompts.ScopeProject, pinnedIDs.Project, projectName); err != nil {
			return nil, promptError(err)
		}
	}
	for _, pin := range pinnedIDs.Teams {
		teamName := s.teamName(ctx, pin.TeamID)
		if err := appendSection(prompts.ScopeTeam, pin.Prompt, teamName); err != nil {
			return nil, promptError(err)
		}
		if task.ProjectID == "" {
			continue
		}
		if err := appendSection(
			prompts.ScopeTeamProject,
			pin.ProjectPrompt,
			teamName+" × "+projectName,
		); err != nil {
			return nil, promptError(err)
		}
	}
	if task.AssigneeActorID != 0 {
		if err := appendSection(prompts.ScopeAgent, pinnedIDs.Agent, agentName); err != nil {
			return nil, promptError(err)
		}
	}
	if task.AssigneeActorID != 0 && task.ProjectID != "" {
		owner := agentName + " × " + projectName
		if err := appendSection(
			prompts.ScopeAgentProject,
			pinnedIDs.AgentProject,
			owner,
		); err != nil {
			return nil, promptError(err)
		}
	}

	// The built-in guidance is not stored or pinned per task; it always applies
	// last, so it reports as a permanent version 1.
	sections = append(sections, &agencyv1.SystemPromptSection{
		Scope:   agencyv1.PromptScope_PROMPT_SCOPE_BUILTIN,
		Owner:   "CherryPi",
		Version: 1,
		Content: prompts.BuiltinGuidance,
		Pinned:  true,
	})

	return &agencyv1.GetTaskSystemPromptResponse{Sections: sections}, nil
}

func (s *PromptServer) settingsName(ctx context.Context) (string, error) {
	if s.Settings == nil {
		return "", nil
	}
	cfg, err := s.Settings.Get(ctx)
	if err != nil {
		return "", err
	}
	return cfg.AgencyName, nil
}

func (s *PromptServer) agentName(ctx context.Context, id ids.AgentID) string {
	if id == 0 || s.Agents == nil {
		return ""
	}
	a, err := s.Agents.Get(ctx, id)
	if err != nil {
		return ""
	}
	if a.DisplayName != "" {
		return a.DisplayName
	}
	return a.Username
}

func (s *PromptServer) teamName(ctx context.Context, id int64) string {
	if id == 0 {
		return ""
	}
	if s.Teams != nil {
		if name, err := s.Teams.Name(ctx, id); err == nil && name != "" {
			return name
		}
	}
	return fmt.Sprintf("Team %d", id)
}

func (s *PromptServer) projectName(ctx context.Context, id string) string {
	if id == "" || s.Projects == nil {
		return ""
	}
	p, err := s.Projects.Get(ctx, id)
	if err != nil {
		return ""
	}
	return p.Name
}

func scopeToProto(scope prompts.Scope) agencyv1.PromptScope {
	switch scope {
	case prompts.ScopeOrg:
		return agencyv1.PromptScope_PROMPT_SCOPE_ORG
	case prompts.ScopeAgent:
		return agencyv1.PromptScope_PROMPT_SCOPE_AGENT
	case prompts.ScopeProject:
		return agencyv1.PromptScope_PROMPT_SCOPE_PROJECT
	case prompts.ScopeAgentProject:
		return agencyv1.PromptScope_PROMPT_SCOPE_AGENT_PROJECT
	case prompts.ScopeTeam:
		return agencyv1.PromptScope_PROMPT_SCOPE_TEAM
	case prompts.ScopeTeamProject:
		return agencyv1.PromptScope_PROMPT_SCOPE_TEAM_PROJECT
	default:
		return agencyv1.PromptScope_PROMPT_SCOPE_UNSPECIFIED
	}
}

// validateTargets confirms the agent/project a prompt belongs to exists, so an
// unknown id is a NotFound rather than a foreign-key failure.
func (s *PromptServer) validateTargets(
	ctx context.Context,
	scope prompts.Scope,
	agentID ids.AgentID,
	teamID int64, projectID string,
) error {
	if scope == prompts.ScopeAgent || scope == prompts.ScopeAgentProject {
		if _, err := s.Agents.Get(ctx, agentID); err != nil {
			if errors.Is(err, agents.ErrAgentMissing) {
				return connect.NewError(connect.CodeNotFound, err)
			}
			return connect.NewError(connect.CodeInternal, err)
		}
	}
	if scope == prompts.ScopeProject || scope == prompts.ScopeAgentProject ||
		scope == prompts.ScopeTeamProject {
		if _, err := s.Projects.Get(ctx, projectID); err != nil {
			if errors.Is(err, projects.ErrProjectMissing) {
				return connect.NewError(connect.CodeNotFound, err)
			}
			return connect.NewError(connect.CodeInternal, err)
		}
	}
	if scope.TeamScoped() {
		if s.Teams == nil {
			return connect.NewError(connect.CodeUnavailable, errors.New("teams unavailable"))
		}
		if _, err := s.Teams.Name(ctx, teamID); err != nil {
			if errors.Is(err, teams.ErrTeamNotFound) {
				return connect.NewError(connect.CodeNotFound, err)
			}
			return connect.NewError(connect.CodeInternal, err)
		}
	}
	return nil
}

// authorizeTeamPrompt restricts team prompt reads to team members and writes
// to team owners (administrators may do both). The team page renders the
// editors with the same rule.
func (s *PromptServer) authorizeTeamPrompt(
	ctx context.Context,
	scope prompts.Scope,
	teamID int64,
	write bool,
) error {
	if !scope.TeamScoped() {
		return nil
	}
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("not authenticated"))
	}
	if s.Teams == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("teams unavailable"))
	}
	team, err := s.Teams.Get(ctx, teamID, identity.ActorID, identity.IsAdmin())
	if err != nil {
		if errors.Is(err, teams.ErrTeamNotFound) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	if write && !identity.IsAdmin() && team.Role != teams.RoleOwner {
		return connect.NewError(connect.CodePermissionDenied, errTeamPromptOwner)
	}
	return nil
}

func fromProtoScope(scope agencyv1.PromptScope) (prompts.Scope, error) {
	switch scope {
	case agencyv1.PromptScope_PROMPT_SCOPE_ORG:
		return prompts.ScopeOrg, nil
	case agencyv1.PromptScope_PROMPT_SCOPE_AGENT:
		return prompts.ScopeAgent, nil
	case agencyv1.PromptScope_PROMPT_SCOPE_PROJECT:
		return prompts.ScopeProject, nil
	case agencyv1.PromptScope_PROMPT_SCOPE_AGENT_PROJECT:
		return prompts.ScopeAgentProject, nil
	case agencyv1.PromptScope_PROMPT_SCOPE_TEAM:
		return prompts.ScopeTeam, nil
	case agencyv1.PromptScope_PROMPT_SCOPE_TEAM_PROJECT:
		return prompts.ScopeTeamProject, nil
	default:
		return "", prompts.ErrInvalidScope
	}
}

func projectToProto(p projects.Project) *agencyv1.Project {
	out := &agencyv1.Project{
		Id:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
	if !p.ArchivedAt.IsZero() {
		out.ArchivedAt = timestamppb.New(p.ArchivedAt)
	}
	return out
}

func toProto(p prompts.Prompt) *agencyv1.Prompt {
	out := &agencyv1.Prompt{Content: p.Content, Version: p.Version}
	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(p.UpdatedAt)
	}
	return out
}

func promptError(err error) *connect.Error {
	switch {
	case errors.Is(err, prompts.ErrInvalidScope),
		errors.Is(err, prompts.ErrInvalidTarget):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
