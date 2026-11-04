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

package promptaccess

import (
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/teams"
)

type Target struct {
	Scope     prompts.Scope
	AgentID   ids.AgentID
	TeamID    int64
	ProjectID string
}

type Authorizer struct {
	Agents   *agents.Registry
	Projects *projects.Service
	Teams    *teams.Service
}

func (a *Authorizer) CanEdit(
	ctx context.Context,
	target Target,
	identity auth.Identity,
) (bool, error) {
	if identity.IsAdmin() {
		return true, nil
	}
	switch target.Scope {
	case prompts.ScopeOrg:
		return false, nil
	case prompts.ScopeAgent, prompts.ScopeAgentProject:
		if a.Agents == nil || target.AgentID == 0 {
			return false, nil
		}
		return a.Agents.IsOwner(ctx, target.AgentID, identity.ActorID)
	case prompts.ScopeProject:
		return a.projectOwner(ctx, target.ProjectID, identity)
	case prompts.ScopeTeam, prompts.ScopeTeamProject:
		if a.Teams == nil || target.TeamID == 0 {
			return false, nil
		}
		return a.Teams.IsOwner(ctx, target.TeamID, identity.ActorID)
	default:
		return false, nil
	}
}

func (a *Authorizer) CanPropose(
	ctx context.Context,
	target Target,
	identity auth.Identity,
) (bool, error) {
	switch target.Scope {
	case prompts.ScopeAgent:
		return a.isSelfAgent(ctx, target.AgentID, identity)
	case prompts.ScopeAgentProject:
		self, err := a.isSelfAgent(ctx, target.AgentID, identity)
		if err != nil || !self {
			return false, err
		}
		return a.projectMember(ctx, target.ProjectID, identity)
	case prompts.ScopeProject:
		return a.projectMember(ctx, target.ProjectID, identity)
	default:
		return false, nil
	}
}

func (a *Authorizer) CanApprove(
	ctx context.Context,
	target Target,
	identity auth.Identity,
) (bool, error) {
	switch target.Scope {
	case prompts.ScopeAgent, prompts.ScopeAgentProject:
		return a.CanEdit(ctx, target, identity)
	case prompts.ScopeProject:
		return a.CanEdit(ctx, target, identity)
	default:
		return a.CanEdit(ctx, target, identity)
	}
}

func (a *Authorizer) Owners(
	ctx context.Context,
	target Target,
) ([]ids.ActorID, error) {
	switch target.Scope {
	case prompts.ScopeAgent, prompts.ScopeAgentProject:
		if a.Agents == nil || target.AgentID == 0 {
			return nil, nil
		}
		owners, err := a.Agents.Owners(ctx, target.AgentID)
		if err != nil {
			return nil, err
		}
		out := make([]ids.ActorID, 0, len(owners))
		for _, owner := range owners {
			out = append(out, owner.ActorID)
		}
		return out, nil
	case prompts.ScopeProject:
		if a.Projects == nil || target.ProjectID == "" {
			return nil, nil
		}
		return a.Projects.OwnerActorIDs(ctx, target.ProjectID)
	default:
		return nil, nil
	}
}

func (a *Authorizer) AgentForActor(
	ctx context.Context,
	actorID ids.ActorID,
) (agents.Agent, error) {
	if a.Agents == nil {
		return agents.Agent{}, agents.ErrAgentMissing
	}
	return a.Agents.GetByActor(ctx, actorID)
}

func (a *Authorizer) isSelfAgent(
	ctx context.Context,
	agentID ids.AgentID,
	identity auth.Identity,
) (bool, error) {
	if identity.Kind != "agent" || agentID == 0 {
		return false, nil
	}
	agent, err := a.AgentForActor(ctx, identity.ActorID)
	if errors.Is(err, agents.ErrAgentMissing) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return agent.ID == agentID, nil
}

func (a *Authorizer) projectMember(
	ctx context.Context,
	projectID string,
	identity auth.Identity,
) (bool, error) {
	if a.Projects == nil || projectID == "" {
		return false, nil
	}
	role, ok, err := a.Projects.EffectiveRole(
		ctx,
		projectID,
		identity.ActorID,
		identity.IsAdmin(),
	)
	if err != nil || !ok {
		return false, err
	}
	return role == projects.RoleOwner || role == projects.RoleMember, nil
}

func (a *Authorizer) projectOwner(
	ctx context.Context,
	projectID string,
	identity auth.Identity,
) (bool, error) {
	if a.Projects == nil || projectID == "" {
		return false, nil
	}
	role, ok, err := a.Projects.EffectiveRole(
		ctx,
		projectID,
		identity.ActorID,
		identity.IsAdmin(),
	)
	if err != nil || !ok {
		return false, err
	}
	return role == projects.RoleOwner, nil
}
