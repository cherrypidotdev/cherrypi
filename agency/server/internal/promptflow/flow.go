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

package promptflow

import (
	"context"
	"errors"
	"fmt"
	"log"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/notifications"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/promptaccess"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/teams"
)

const (
	KindProposalCreated   = "prompt_proposal_created"
	KindProposalApproved  = "prompt_proposal_approved"
	KindProposalRejected  = "prompt_proposal_rejected"
	KindProposalWithdrawn = "prompt_proposal_withdrawn"
)

type Access struct {
	CanEdit    bool
	CanPropose bool
	CanApprove bool
}

type ProposalQuery struct {
	Target promptaccess.Target
	Status prompts.ProposalStatus
	Mine   bool
	Limit  int
	Before int64
}

type Service struct {
	Prompts       *prompts.Service
	Access        *promptaccess.Authorizer
	Notifications *notifications.Service
	Audit         *audit.Service
	Projects      *projects.Service
	Teams         *teams.Service
	Agents        *agents.Registry
	Accounts      *accounts.Service
}

func (s *Service) Get(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
) (prompts.Prompt, Access, error) {
	p, err := s.latest(ctx, target)
	if err != nil {
		return prompts.Prompt{}, Access{}, err
	}
	access, err := s.access(ctx, identity, target)
	return p, access, err
}

func (s *Service) Set(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
	content, note string,
) (prompts.Prompt, error) {
	allowed, err := s.Access.CanEdit(ctx, target, identity)
	if err != nil {
		return prompts.Prompt{}, err
	}
	if !allowed {
		return prompts.Prompt{}, ErrForbidden
	}
	old, err := s.latest(ctx, target)
	if err != nil {
		return prompts.Prompt{}, err
	}
	p, err := s.Prompts.SetVersion(ctx, prompts.SetParams{
		Scope:         target.Scope,
		AgentID:       target.AgentID,
		TeamID:        target.TeamID,
		ProjectID:     target.ProjectID,
		Content:       content,
		AuthorActorID: identity.ActorID,
		Note:          note,
	})
	if err != nil {
		return prompts.Prompt{}, err
	}
	if p.ID != old.ID {
		s.auditPrompt(ctx, identity, target, audit.KindPrompt, old.Content, p,
			"Updated the prompt", nil)
	}
	return p, nil
}

func (s *Service) ListVersions(
	ctx context.Context,
	target promptaccess.Target,
	limit int, beforeVersion int64,
) ([]prompts.Prompt, error) {
	return s.Prompts.ListVersions(
		ctx,
		target.Scope,
		target.AgentID,
		target.TeamID,
		target.ProjectID,
		limit,
		beforeVersion,
	)
}

func (s *Service) CreateProposal(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
	content, note string,
) (prompts.Proposal, error) {
	allowed, err := s.Access.CanPropose(ctx, target, identity)
	if err != nil {
		return prompts.Proposal{}, err
	}
	if !allowed {
		return prompts.Proposal{}, ErrForbidden
	}
	proposal, err := s.Prompts.CreateProposal(ctx, prompts.CreateProposalParams{
		Scope:         target.Scope,
		AgentID:       target.AgentID,
		TeamID:        target.TeamID,
		ProjectID:     target.ProjectID,
		Content:       content,
		AuthorActorID: identity.ActorID,
		Note:          note,
	})
	if err != nil {
		return prompts.Proposal{}, err
	}
	s.auditProposal(ctx, identity, target, audit.KindPromptProposalCreate,
		proposal, "Proposed a prompt change", nil)

	if s.shouldAutoApprove(ctx, identity, target) {
		prompt, approved, err := s.Prompts.ApproveProposal(
			ctx,
			proposal.ID,
			0,
			"Auto-approved by the agent's setting",
			true,
		)
		if err == nil {
			s.auditProposal(ctx, identity, target, audit.KindPromptProposalApprove,
				approved, "Auto-approved a prompt proposal", nil)
			s.auditPrompt(ctx, identity, target, audit.KindPrompt, "",
				prompt, "Applied an auto-approved prompt proposal", nil)
			s.notifyDecision(ctx, identity, target, approved, prompt)
			return approved, nil
		}

		if !errors.Is(err, prompts.ErrProposalStale) {
			return prompts.Proposal{}, err
		}
	}

	s.notifyCreated(ctx, identity, target, proposal)
	return proposal, nil
}

func (s *Service) ListProposals(
	ctx context.Context,
	identity auth.Identity,
	query ProposalQuery,
) ([]prompts.Proposal, error) {
	if query.Mine {
		return s.Prompts.ListProposals(ctx, prompts.ProposalFilter{
			Status:      query.Status,
			MineActorID: identity.ActorID,
			Limit:       query.Limit,
			BeforeID:    query.Before,
		})
	}
	allowed, err := s.Access.CanApprove(ctx, query.Target, identity)
	if err != nil {
		return nil, err
	}
	if allowed {
		return s.Prompts.ListProposals(ctx, prompts.ProposalFilter{
			Scope:     query.Target.Scope,
			AgentID:   query.Target.AgentID,
			TeamID:    query.Target.TeamID,
			ProjectID: query.Target.ProjectID,
			Status:    query.Status,
			Limit:     query.Limit,
			BeforeID:  query.Before,
		})
	}

	mine, err := s.Prompts.ListProposals(ctx, prompts.ProposalFilter{
		Status:      query.Status,
		MineActorID: identity.ActorID,
		Limit:       query.Limit,
		BeforeID:    query.Before,
	})
	if err != nil {
		return nil, err
	}
	out := make([]prompts.Proposal, 0, len(mine))
	for _, p := range mine {
		if sameTarget(p, query.Target) {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Service) DecideProposal(
	ctx context.Context,
	identity auth.Identity,
	id int64,
	approve bool,
	note string,
) (prompts.Prompt, prompts.Proposal, error) {
	proposal, err := s.Prompts.GetProposal(ctx, id)
	if err != nil {
		return prompts.Prompt{}, prompts.Proposal{}, err
	}
	target := targetOf(proposal)
	allowed, err := s.Access.CanApprove(ctx, target, identity)
	if err != nil {
		return prompts.Prompt{}, prompts.Proposal{}, err
	}
	if !allowed {
		return prompts.Prompt{}, prompts.Proposal{}, ErrForbidden
	}
	if !approve {
		rejected, err := s.Prompts.RejectProposal(ctx, id, identity.ActorID, note)
		if err != nil {
			return prompts.Prompt{}, prompts.Proposal{}, err
		}
		s.auditProposal(ctx, identity, target, audit.KindPromptProposalReject,
			rejected, "Rejected a prompt proposal", nil)
		s.notifyDecision(ctx, identity, target, rejected, prompts.Prompt{})
		return prompts.Prompt{}, rejected, nil
	}
	prompt, approved, err := s.Prompts.ApproveProposal(
		ctx,
		id,
		identity.ActorID,
		note,
		false,
	)
	if err != nil {
		return prompts.Prompt{}, prompts.Proposal{}, err
	}
	s.auditProposal(ctx, identity, target, audit.KindPromptProposalApprove,
		approved, "Approved a prompt proposal", nil)
	s.auditPrompt(ctx, identity, target, audit.KindPrompt, "",
		prompt, "Applied an approved prompt proposal", nil)
	s.notifyDecision(ctx, identity, target, approved, prompt)
	return prompt, approved, nil
}

func (s *Service) WithdrawProposal(
	ctx context.Context,
	identity auth.Identity,
	id int64,
	note string,
) (prompts.Proposal, error) {
	proposal, err := s.Prompts.GetProposal(ctx, id)
	if err != nil {
		return prompts.Proposal{}, err
	}
	if proposal.AuthorActorID != identity.ActorID {
		return prompts.Proposal{}, ErrForbidden
	}
	withdrawn, err := s.Prompts.WithdrawProposal(ctx, id, identity.ActorID, note)
	if err != nil {
		return prompts.Proposal{}, err
	}
	target := targetOf(withdrawn)
	s.auditProposal(ctx, identity, target, audit.KindPromptProposalWithdraw,
		withdrawn, "Withdrew a prompt proposal", nil)
	s.notifyWithdrawn(ctx, identity, target, withdrawn)
	return withdrawn, nil
}

var ErrForbidden = errors.New("prompt access denied")

func (s *Service) access(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
) (Access, error) {
	edit, err := s.Access.CanEdit(ctx, target, identity)
	if err != nil {
		return Access{}, err
	}
	propose, err := s.Access.CanPropose(ctx, target, identity)
	if err != nil {
		return Access{}, err
	}
	approve, err := s.Access.CanApprove(ctx, target, identity)
	if err != nil {
		return Access{}, err
	}
	return Access{CanEdit: edit, CanPropose: propose, CanApprove: approve}, nil
}

func (s *Service) latest(
	ctx context.Context,
	target promptaccess.Target,
) (prompts.Prompt, error) {
	return s.Prompts.GetAt(
		ctx,
		target.Scope,
		target.AgentID,
		target.TeamID,
		target.ProjectID,
	)
}

func (s *Service) shouldAutoApprove(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
) bool {
	if identity.Kind != accounts.KindAgent {
		return false
	}
	if target.Scope != prompts.ScopeAgent && target.Scope != prompts.ScopeAgentProject {
		return false
	}
	agent, err := s.Access.AgentForActor(ctx, identity.ActorID)
	if err != nil || agent.ID != target.AgentID {
		return false
	}
	return agent.PromptAutoApprove
}

func (s *Service) auditPrompt(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
	kind, oldValue string,
	prompt prompts.Prompt,
	message string,
	data any,
) {
	if data == nil {
		data = map[string]any{
			"scope":       string(target.Scope),
			"version":     prompt.Version,
			"proposal_id": prompt.ProposalID,
		}
	}
	switch target.Scope {
	case prompts.ScopeAgent, prompts.ScopeAgentProject:
		if s.Agents == nil || target.AgentID == 0 {
			return
		}
		agent, err := s.Agents.Get(ctx, target.AgentID)
		if err != nil {
			return
		}
		s.logAgent(ctx, agent.ActorID, identity.ActorID, kind, oldValue, prompt,
			message, data)
	case prompts.ScopeProject:
		if s.Projects != nil && target.ProjectID != "" {
			s.Projects.LogPrompt(ctx, target.ProjectID, kind, "content",
				truncate(oldValue), truncate(prompt.Content), message, data)
		}
	case prompts.ScopeTeam, prompts.ScopeTeamProject:
		if s.Teams != nil && target.TeamID != 0 {
			s.Teams.LogPrompt(ctx, target.TeamID, kind, "content",
				truncate(oldValue), truncate(prompt.Content), message, data)
		}
	default:
		if s.Audit != nil {
			s.Audit.LogUser(ctx, identity.ActorID, identity.ActorID, kind, "content",
				truncate(oldValue), truncate(prompt.Content), message, data)
		}
	}
}

func (s *Service) logAgent(
	ctx context.Context,
	agentActorID, actorID ids.ActorID,
	kind, oldValue string,
	prompt prompts.Prompt,
	message string,
	data any,
) {
	if s.Audit == nil {
		return
	}
	s.Audit.LogAgent(ctx, agentActorID, actorID, kind, "content",
		truncate(oldValue), truncate(prompt.Content), message, data)
}

func (s *Service) auditProposal(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
	kind string,
	proposal prompts.Proposal,
	message string,
	data any,
) {
	if data == nil {
		data = map[string]any{
			"scope":        string(target.Scope),
			"proposal_id":  proposal.ID,
			"base_version": proposal.BaseVersion,
			"status":       string(proposal.Status),
		}
	}
	switch target.Scope {
	case prompts.ScopeAgent, prompts.ScopeAgentProject:
		if s.Agents == nil || target.AgentID == 0 {
			return
		}
		agent, err := s.Agents.Get(ctx, target.AgentID)
		if err != nil {
			return
		}
		s.logAgent(ctx, agent.ActorID, identity.ActorID, kind, "", prompts.Prompt{
			Content: proposal.Content,
		}, message, data)
	case prompts.ScopeProject:
		if s.Projects != nil && target.ProjectID != "" {
			s.Projects.LogPrompt(ctx, target.ProjectID, kind, "content", "",
				truncate(proposal.Content), message, data)
		}
	case prompts.ScopeTeam, prompts.ScopeTeamProject:
		if s.Teams != nil && target.TeamID != 0 {
			s.Teams.LogPrompt(ctx, target.TeamID, kind, "content", "",
				truncate(proposal.Content), message, data)
		}
	default:
		if s.Audit != nil {
			s.Audit.LogUser(ctx, identity.ActorID, identity.ActorID, kind, "content",
				"", truncate(proposal.Content), message, data)
		}
	}
}

func (s *Service) notifyCreated(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
	proposal prompts.Proposal,
) {
	recipients, err := s.Access.Owners(ctx, target)
	if err != nil {
		log.Printf("prompt proposal owners: %v", err)
		return
	}
	author := s.actorLabel(ctx, identity.ActorID)
	label := s.targetLabel(ctx, target)
	for _, recipient := range recipients {
		if recipient == identity.ActorID {
			continue
		}
		s.notify(ctx, recipient, target, KindProposalCreated,
			"Prompt change proposed",
			fmt.Sprintf("%s proposed a change to %s.", author, label))
	}
}

func (s *Service) notifyDecision(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
	proposal prompts.Proposal,
	prompt prompts.Prompt,
) {
	if proposal.AuthorActorID == 0 || proposal.AuthorActorID == identity.ActorID {
		return
	}
	label := s.targetLabel(ctx, target)
	title := "Prompt change rejected"
	body := fmt.Sprintf(
		"%s rejected your change to %s.",
		s.actorLabel(ctx, identity.ActorID),
		label,
	)
	if proposal.Status == prompts.ProposalApproved {
		title = "Prompt change approved"
		body = fmt.Sprintf(
			"%s approved your change to %s.",
			s.actorLabel(ctx, identity.ActorID),
			label,
		)
		if proposal.AutoApproved {
			title = "Prompt change auto-approved"
			body = fmt.Sprintf("Your change to %s was auto-approved.", label)
		}
	}
	if proposal.DecisionNote != "" {
		body += " " + proposal.DecisionNote
	}
	s.notify(ctx, proposal.AuthorActorID, target, decisionKind(proposal.Status), title, body)
	_ = prompt
}

func (s *Service) notifyWithdrawn(
	ctx context.Context,
	identity auth.Identity,
	target promptaccess.Target,
	proposal prompts.Proposal,
) {
	recipients, err := s.Access.Owners(ctx, target)
	if err != nil {
		return
	}
	label := s.targetLabel(ctx, target)
	for _, recipient := range recipients {
		if recipient == identity.ActorID {
			continue
		}
		s.notify(ctx, recipient, target, KindProposalWithdrawn,
			"Prompt proposal withdrawn",
			fmt.Sprintf("%s withdrew a change to %s.", s.actorLabel(ctx, identity.ActorID), label))
	}
}

func (s *Service) notify(
	ctx context.Context,
	recipient ids.ActorID,
	target promptaccess.Target,
	kind, title, body string,
) {
	if s.Notifications == nil || recipient == 0 {
		return
	}
	err := s.Notifications.Create(ctx, notifications.Notification{
		AgentID:          target.AgentID,
		ProjectID:        target.ProjectID,
		RecipientActorID: recipient,
		Kind:             kind,
		Severity:         "info",
		Title:            title,
		Body:             body,
	})
	if err != nil {
		log.Printf("prompt proposal notification: %v", err)
	}
}

func (s *Service) actorLabel(ctx context.Context, actorID ids.ActorID) string {
	if s.Accounts == nil || actorID == 0 {
		return "An actor"
	}
	actor, err := s.Accounts.GetActor(ctx, actorID)
	if err != nil {
		return "An actor"
	}
	if actor.Name != "" {
		return actor.Name
	}
	return actor.Username
}

func (s *Service) targetLabel(ctx context.Context, target promptaccess.Target) string {
	switch target.Scope {
	case prompts.ScopeAgent:
		if s.Agents != nil {
			if agent, err := s.Agents.Get(ctx, target.AgentID); err == nil {
				return agent.DisplayName + "'s prompt"
			}
		}
		return "an agent prompt"
	case prompts.ScopeAgentProject:
		agentName, projectName := "an agent", "a project"
		if s.Agents != nil {
			if agent, err := s.Agents.Get(ctx, target.AgentID); err == nil {
				agentName = agent.DisplayName
			}
		}
		if s.Projects != nil {
			if project, err := s.Projects.Get(ctx, target.ProjectID); err == nil {
				projectName = project.Name
			}
		}
		return agentName + " × " + projectName + " prompt"
	case prompts.ScopeProject:
		if s.Projects != nil {
			if project, err := s.Projects.Get(ctx, target.ProjectID); err == nil {
				return project.Name + " project prompt"
			}
		}
		return "a project prompt"
	case prompts.ScopeTeam, prompts.ScopeTeamProject:
		if s.Teams != nil {
			if name, err := s.Teams.Name(ctx, target.TeamID); err == nil && name != "" {
				return name + " team prompt"
			}
		}
		return "a team prompt"
	default:
		return "the organization prompt"
	}
}

func targetOf(p prompts.Proposal) promptaccess.Target {
	return promptaccess.Target{
		Scope:     p.Scope,
		AgentID:   p.AgentID,
		TeamID:    p.TeamID,
		ProjectID: p.ProjectID,
	}
}

func sameTarget(p prompts.Proposal, target promptaccess.Target) bool {
	return p.Scope == target.Scope &&
		p.AgentID == target.AgentID &&
		p.TeamID == target.TeamID &&
		p.ProjectID == target.ProjectID
}

func decisionKind(status prompts.ProposalStatus) string {
	if status == prompts.ProposalApproved {
		return KindProposalApproved
	}
	return KindProposalRejected
}

func truncate(value string) string {
	const max = 4000
	if len(value) <= max {
		return value
	}
	return value[:max] + "…"
}
