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

package promptsgrpc

import (
	"context"
	"errors"
	"fmt"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agentgroups"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/promptaccess"
	"cherrypi.vladfaust.com/agency/internal/promptflow"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/settings"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/teams"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PromptServer struct {
	Flow     *promptflow.Service
	Prompts  *prompts.Service
	Projects *projects.Service
	Agents   *agents.Registry
	Tasks    *tasks.Service
	Settings *settings.Service
	Teams    *teams.Service
	Groups   *agentgroups.Service
	Accounts *accounts.Service
}

func (s *PromptServer) GetPrompt(
	ctx context.Context,
	req *agencyv1.GetPromptRequest,
) (*agencyv1.GetPromptResponse, error) {
	identity, err := requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	target, err := targetFromProto(req.Scope, req.AgentId, req.TeamId, req.GroupId, req.ProjectId)
	if err != nil {
		return nil, err
	}
	target, err = s.validateTargets(ctx, target)
	if err != nil {
		return nil, err
	}
	p, access, err := s.Flow.Get(ctx, identity, target)
	if err != nil {
		return nil, promptError(err)
	}
	return &agencyv1.GetPromptResponse{
		Prompt:     s.promptToProto(ctx, p),
		CanEdit:    access.CanEdit,
		CanPropose: access.CanPropose,
		CanApprove: access.CanApprove,
	}, nil
}

func (s *PromptServer) SetPrompt(
	ctx context.Context,
	req *agencyv1.SetPromptRequest,
) (*agencyv1.SetPromptResponse, error) {
	identity, err := requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	target, err := targetFromProto(req.Scope, req.AgentId, req.TeamId, req.GroupId, req.ProjectId)
	if err != nil {
		return nil, err
	}
	target, err = s.validateTargets(ctx, target)
	if err != nil {
		return nil, err
	}
	p, err := s.Flow.Set(ctx, identity, target, req.Content, req.Note)
	if err != nil {
		return nil, promptError(err)
	}
	return &agencyv1.SetPromptResponse{Prompt: s.promptToProto(ctx, p)}, nil
}

func (s *PromptServer) ListPromptVersions(
	ctx context.Context,
	req *agencyv1.ListPromptVersionsRequest,
) (*agencyv1.ListPromptVersionsResponse, error) {
	target, err := targetFromProto(req.Scope, req.AgentId, req.TeamId, req.GroupId, req.ProjectId)
	if err != nil {
		return nil, err
	}
	target, err = s.validateTargets(ctx, target)
	if err != nil {
		return nil, err
	}
	versions, err := s.Flow.ListVersions(
		ctx,
		target,
		int(req.Limit),
		req.BeforeVersion,
	)
	if err != nil {
		return nil, promptError(err)
	}
	out := make([]*agencyv1.Prompt, len(versions))
	for i, p := range versions {
		out[i] = s.promptToProto(ctx, p)
	}
	return &agencyv1.ListPromptVersionsResponse{Versions: out}, nil
}

func (s *PromptServer) ListAgentProjectPrompts(
	ctx context.Context,
	req *agencyv1.ListAgentProjectPromptsRequest,
) (*agencyv1.ListAgentProjectPromptsResponse, error) {

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
			Prompt:  s.promptToProto(ctx, p),
		}
	}
	return &agencyv1.ListAgentProjectPromptsResponse{Prompts: out}, nil
}

func (s *PromptServer) CreatePromptProposal(
	ctx context.Context,
	req *agencyv1.CreatePromptProposalRequest,
) (*agencyv1.CreatePromptProposalResponse, error) {
	identity, err := requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	target, err := targetFromProto(req.Scope, req.AgentId, req.TeamId, req.GroupId, req.ProjectId)
	if err != nil {
		return nil, err
	}
	target, err = s.validateTargets(ctx, target)
	if err != nil {
		return nil, err
	}
	proposal, err := s.Flow.CreateProposal(
		ctx,
		identity,
		target,
		req.Content,
		req.Note,
	)
	if err != nil {
		return nil, promptError(err)
	}
	return &agencyv1.CreatePromptProposalResponse{
		Proposal: s.proposalToProto(ctx, proposal),
	}, nil
}

func (s *PromptServer) ListPromptProposals(
	ctx context.Context,
	req *agencyv1.ListPromptProposalsRequest,
) (*agencyv1.ListPromptProposalsResponse, error) {
	identity, err := requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	query := promptflow.ProposalQuery{
		Status: fromProtoProposalStatus(req.Status),
		Mine:   req.Mine,
		Limit:  int(req.Limit),
		Before: req.BeforeId,
	}
	if !req.Mine {
		target, err := targetFromProto(
			req.Scope,
			req.AgentId,
			req.TeamId,
			req.GroupId,
			req.ProjectId,
		)
		if err != nil {
			return nil, err
		}
		target, err = s.validateTargets(ctx, target)
		if err != nil {
			return nil, err
		}
		query.Target = target
	}
	list, err := s.Flow.ListProposals(ctx, identity, query)
	if err != nil {
		return nil, promptError(err)
	}
	out := make([]*agencyv1.PromptProposal, len(list))
	for i, p := range list {
		out[i] = s.proposalToProto(ctx, p)
	}
	return &agencyv1.ListPromptProposalsResponse{Proposals: out}, nil
}

func (s *PromptServer) DecidePromptProposal(
	ctx context.Context,
	req *agencyv1.DecidePromptProposalRequest,
) (*agencyv1.DecidePromptProposalResponse, error) {
	identity, err := requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	prompt, proposal, err := s.Flow.DecideProposal(
		ctx,
		identity,
		req.Id,
		req.Approve,
		req.Note,
	)
	if err != nil {
		return nil, promptError(err)
	}
	out := &agencyv1.DecidePromptProposalResponse{
		Proposal: s.proposalToProto(ctx, proposal),
	}
	if prompt.ID != 0 {
		out.Prompt = s.promptToProto(ctx, prompt)
	}
	return out, nil
}

func (s *PromptServer) WithdrawPromptProposal(
	ctx context.Context,
	req *agencyv1.WithdrawPromptProposalRequest,
) (*agencyv1.WithdrawPromptProposalResponse, error) {
	identity, err := requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	proposal, err := s.Flow.WithdrawProposal(ctx, identity, req.Id, req.Note)
	if err != nil {
		return nil, promptError(err)
	}
	return &agencyv1.WithdrawPromptProposalResponse{
		Proposal: s.proposalToProto(ctx, proposal),
	}, nil
}

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

	pinned := !task.StartedAt.IsZero()
	pinnedIDs := task.PromptIDs
	var agent agents.Agent
	if task.AssigneeActorID != 0 {
		var agentErr error
		agent, agentErr = s.Agents.GetByActor(ctx, task.AssigneeActorID)
		if agentErr != nil {
			return nil, connect.NewError(connect.CodeInternal, agentErr)
		}
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
	appendSection := func(
		scope prompts.Scope,
		id int64,
		owner string,
		groupID int64,
		groupName string,
		teamID int64,
	) error {
		p, err := s.Prompts.GetByID(ctx, id)
		if err != nil {
			return err
		}
		sections = append(sections, &agencyv1.SystemPromptSection{
			Scope:     scopeToProto(scope),
			Owner:     owner,
			Version:   p.Version,
			Content:   p.Content,
			Pinned:    pinned,
			GroupId:   groupID,
			GroupName: groupName,
			TeamId:    teamID,

			Preview: !pinned && task.AssigneeActorID != 0,
		})
		return nil
	}

	if err := appendSection(prompts.ScopeOrg, pinnedIDs.Org, agencyName, 0, "", 0); err != nil {
		return nil, promptError(err)
	}
	if task.ProjectID != "" {
		if err := appendSection(
			prompts.ScopeProject,
			pinnedIDs.Project,
			projectName,
			0,
			"",
			0,
		); err != nil {
			return nil, promptError(err)
		}
	}
	for _, pin := range pinnedIDs.Teams {
		teamName := s.teamName(ctx, pin.TeamID)
		if err := appendSection(
			prompts.ScopeTeam,
			pin.Prompt,
			teamName,
			0,
			"",
			pin.TeamID,
		); err != nil {
			return nil, promptError(err)
		}
		if task.ProjectID == "" {
			continue
		}
		if err := appendSection(
			prompts.ScopeTeamProject,
			pin.ProjectPrompt,
			teamName+" × "+projectName,
			0,
			"",
			pin.TeamID,
		); err != nil {
			return nil, promptError(err)
		}
	}

	for _, pin := range pinnedIDs.Groups {
		groupName := s.groupName(ctx, pin.GroupID)

		if pin.Prompt != 0 {
			if err := appendSection(
				prompts.ScopeGroup,
				pin.Prompt,
				groupName,
				pin.GroupID,
				groupName,
				0,
			); err != nil {
				return nil, promptError(err)
			}
		}
		if task.ProjectID != "" && pin.ProjectPrompt != 0 {
			if err := appendSection(
				prompts.ScopeGroupProject,
				pin.ProjectPrompt,
				groupName+" × "+projectName,
				pin.GroupID,
				groupName,
				0,
			); err != nil {
				return nil, promptError(err)
			}
		}
		for _, teamPin := range pin.Teams {
			if teamPin.Prompt == 0 {
				continue
			}
			teamName := s.teamName(ctx, teamPin.TeamID)
			if err := appendSection(
				prompts.ScopeGroupTeam,
				teamPin.Prompt,
				groupName+" × "+teamName,
				pin.GroupID,
				groupName,
				teamPin.TeamID,
			); err != nil {
				return nil, promptError(err)
			}
		}
	}
	if task.AssigneeActorID != 0 {
		if err := appendSection(
			prompts.ScopeAgent,
			pinnedIDs.Agent,
			agentName,
			0,
			"",
			0,
		); err != nil {
			return nil, promptError(err)
		}
	}
	if task.AssigneeActorID != 0 && task.ProjectID != "" {
		owner := agentName + " × " + projectName
		if err := appendSection(
			prompts.ScopeAgentProject,
			pinnedIDs.AgentProject,
			owner,
			0,
			"",
			0,
		); err != nil {
			return nil, promptError(err)
		}
	}

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

func (s *PromptServer) groupName(ctx context.Context, id int64) string {
	if id == 0 || s.Groups == nil {
		return ""
	}
	if name, err := s.Groups.Name(ctx, id); err == nil && name != "" {
		return name
	}
	return fmt.Sprintf("Group %d", id)
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

func (s *PromptServer) validateTargets(
	ctx context.Context,
	target promptaccess.Target,
) (promptaccess.Target, error) {
	if target.Scope == prompts.ScopeAgent || target.Scope == prompts.ScopeAgentProject {
		if _, err := s.Agents.Get(ctx, target.AgentID); err != nil {
			if errors.Is(err, agents.ErrAgentMissing) {
				return target, connect.NewError(connect.CodeNotFound, err)
			}
			return target, connect.NewError(connect.CodeInternal, err)
		}
	}
	if target.Scope == prompts.ScopeProject ||
		target.Scope == prompts.ScopeAgentProject ||
		target.Scope == prompts.ScopeTeamProject ||
		target.Scope == prompts.ScopeGroupProject {
		project, err := s.Projects.Get(ctx, target.ProjectID)
		if errors.Is(err, projects.ErrProjectMissing) {
			return target, connect.NewError(connect.CodeNotFound, err)
		}
		if err != nil {
			return target, connect.NewError(connect.CodeInternal, err)
		}
		target.ProjectID = project.ID
	}
	if target.Scope.GroupScoped() {
		if s.Groups == nil {
			return target, connect.NewError(
				connect.CodeUnavailable,
				errors.New("agent groups unavailable"),
			)
		}
		if _, err := s.Groups.Get(ctx, target.GroupID); err != nil {
			if errors.Is(err, agentgroups.ErrNotFound) {
				return target, connect.NewError(connect.CodeNotFound, err)
			}
			return target, connect.NewError(connect.CodeInternal, err)
		}
	}
	if target.Scope.TeamScoped() || target.Scope == prompts.ScopeGroupTeam {
		if s.Teams == nil {
			return target, connect.NewError(
				connect.CodeUnavailable,
				errors.New("teams unavailable"),
			)
		}
		if _, err := s.Teams.Name(ctx, target.TeamID); err != nil {
			if errors.Is(err, teams.ErrTeamNotFound) {
				return target, connect.NewError(connect.CodeNotFound, err)
			}
			return target, connect.NewError(connect.CodeInternal, err)
		}
	}
	return target, nil
}

func requireIdentity(ctx context.Context) (auth.Identity, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return auth.Identity{}, connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("not authenticated"),
		)
	}
	return identity, nil
}

func targetFromProto(
	scope agencyv1.PromptScope,
	agentID string, teamID, groupID int64, projectID string,
) (promptaccess.Target, error) {
	s, err := fromProtoScope(scope)
	if err != nil {
		return promptaccess.Target{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return promptaccess.Target{
		Scope:     s,
		AgentID:   actorid.ParseAgent(agentID),
		TeamID:    teamID,
		GroupID:   groupID,
		ProjectID: projectID,
	}, nil
}

func (s *PromptServer) promptToProto(ctx context.Context, p prompts.Prompt) *agencyv1.Prompt {
	out := &agencyv1.Prompt{
		Id:           p.ID,
		Content:      p.Content,
		Version:      p.Version,
		Author:       s.actorToProto(ctx, p.AuthorActorID),
		ApprovedBy:   s.actorToProto(ctx, p.ApprovedByActorID),
		Note:         p.Note,
		ProposalId:   p.ProposalID,
		AutoApproved: p.AutoApproved,
	}
	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(p.UpdatedAt)
	}
	return out
}

func (s *PromptServer) proposalToProto(
	ctx context.Context,
	p prompts.Proposal,
) *agencyv1.PromptProposal {
	out := &agencyv1.PromptProposal{
		Id:            p.ID,
		Scope:         scopeToProto(p.Scope),
		AgentId:       actorid.Format(p.AgentID),
		ProjectId:     p.ProjectID,
		TeamId:        p.TeamID,
		GroupId:       p.GroupID,
		Content:       p.Content,
		BaseVersion:   p.BaseVersion,
		Author:        s.actorToProto(ctx, p.AuthorActorID),
		Note:          p.Note,
		Status:        toProtoProposalStatus(p.Status),
		DecidedBy:     s.actorToProto(ctx, p.DecidedByActorID),
		DecisionNote:  p.DecisionNote,
		Stale:         p.Stale,
		AutoApproved:  p.AutoApproved,
		LatestVersion: p.LatestVersion,
		IsMine:        callerActorID(ctx) == p.AuthorActorID,
	}
	if !p.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(p.CreatedAt)
	}
	if !p.DecidedAt.IsZero() {
		out.DecidedAt = timestamppb.New(p.DecidedAt)
	}
	return out
}

func (s *PromptServer) actorToProto(
	ctx context.Context,
	id ids.ActorID,
) *agencyv1.PromptActor {
	if id == 0 {
		return nil
	}
	out := &agencyv1.PromptActor{ActorId: actorid.Format(id)}
	if s.Accounts == nil {
		return out
	}
	actor, err := s.Accounts.GetActor(ctx, id)
	if err != nil {
		return out
	}
	out.Username = actor.Username
	out.Name = actor.Name
	out.Kind = actor.Kind
	return out
}

func callerActorID(ctx context.Context) ids.ActorID {
	id, _ := auth.ActorID(ctx)
	return id
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
	case prompts.ScopeGroup:
		return agencyv1.PromptScope_PROMPT_SCOPE_GROUP
	case prompts.ScopeGroupProject:
		return agencyv1.PromptScope_PROMPT_SCOPE_GROUP_PROJECT
	case prompts.ScopeGroupTeam:
		return agencyv1.PromptScope_PROMPT_SCOPE_GROUP_TEAM
	default:
		return agencyv1.PromptScope_PROMPT_SCOPE_UNSPECIFIED
	}
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
	case agencyv1.PromptScope_PROMPT_SCOPE_GROUP:
		return prompts.ScopeGroup, nil
	case agencyv1.PromptScope_PROMPT_SCOPE_GROUP_PROJECT:
		return prompts.ScopeGroupProject, nil
	case agencyv1.PromptScope_PROMPT_SCOPE_GROUP_TEAM:
		return prompts.ScopeGroupTeam, nil
	default:
		return "", prompts.ErrInvalidScope
	}
}

func toProtoProposalStatus(status prompts.ProposalStatus) agencyv1.PromptProposalStatus {
	switch status {
	case prompts.ProposalPending:
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_PENDING
	case prompts.ProposalApproved:
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_APPROVED
	case prompts.ProposalRejected:
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_REJECTED
	case prompts.ProposalWithdrawn:
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_WITHDRAWN
	default:
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_UNSPECIFIED
	}
}

func fromProtoProposalStatus(
	status agencyv1.PromptProposalStatus,
) prompts.ProposalStatus {
	switch status {
	case agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_PENDING:
		return prompts.ProposalPending
	case agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_APPROVED:
		return prompts.ProposalApproved
	case agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_REJECTED:
		return prompts.ProposalRejected
	case agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_WITHDRAWN:
		return prompts.ProposalWithdrawn
	default:
		return ""
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

func promptError(err error) *connect.Error {
	switch {
	case errors.Is(err, prompts.ErrInvalidScope),
		errors.Is(err, prompts.ErrInvalidTarget):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, prompts.ErrNoChange):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, prompts.ErrProposalNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, prompts.ErrProposalNotPending),
		errors.Is(err, prompts.ErrProposalStale):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, promptflow.ErrForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
