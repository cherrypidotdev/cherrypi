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

package variablesgrpc

import (
	"context"
	"errors"
	"strconv"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/promptaccess"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/variables"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	Variables *variables.Service

	Access *promptaccess.Authorizer

	Actors ActorStore
}

type ActorStore interface {
	ListActorsByIDs(ctx context.Context, actorIDs []ids.ActorID) ([]store.Actor, error)
}

func (s *Server) ListVariables(
	ctx context.Context,
	req *agencyv1.ListVariablesRequest,
) (*agencyv1.ListVariablesResponse, error) {
	scope, err := scopeOf(req.GetTarget())
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, scope); err != nil {
		return nil, err
	}
	list, err := s.Variables.List(ctx, scope)
	if err != nil {
		return nil, mapError(err)
	}
	names := s.actorNames(ctx, list)
	out := make([]*agencyv1.Variable, 0, len(list))
	for _, variable := range list {
		out = append(out, toProto(variable, names[variable.UpdatedBy]))
	}
	return &agencyv1.ListVariablesResponse{Variables: out}, nil
}

func (s *Server) CreateVariable(
	ctx context.Context,
	req *agencyv1.CreateVariableRequest,
) (*agencyv1.CreateVariableResponse, error) {
	scope, err := scopeOf(req.GetTarget())
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, scope); err != nil {
		return nil, err
	}
	variable, err := s.Variables.Create(
		ctx, scope, req.GetName(), req.GetValue(), req.GetNote(), req.GetSecret(),
		authActorID(ctx),
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.CreateVariableResponse{
		Variable: toProto(
			variable,
			s.actorNames(ctx, []variables.Variable{variable})[variable.UpdatedBy],
		),
	}, nil
}

func (s *Server) UpdateVariable(
	ctx context.Context,
	req *agencyv1.UpdateVariableRequest,
) (*agencyv1.UpdateVariableResponse, error) {
	scope, err := scopeOf(req.GetTarget())
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, scope); err != nil {
		return nil, err
	}
	variable, err := s.Variables.Update(
		ctx, scope, req.GetName(), req.Note, req.Value, authActorID(ctx),
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateVariableResponse{
		Variable: toProto(
			variable,
			s.actorNames(ctx, []variables.Variable{variable})[variable.UpdatedBy],
		),
	}, nil
}

func (s *Server) PromoteVariable(
	ctx context.Context,
	req *agencyv1.PromoteVariableRequest,
) (*agencyv1.PromoteVariableResponse, error) {
	scope, err := scopeOf(req.GetTarget())
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, scope); err != nil {
		return nil, err
	}
	variable, err := s.Variables.Promote(ctx, scope, req.GetName(), authActorID(ctx))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.PromoteVariableResponse{
		Variable: toProto(
			variable,
			s.actorNames(ctx, []variables.Variable{variable})[variable.UpdatedBy],
		),
	}, nil
}

func (s *Server) DeleteVariable(
	ctx context.Context,
	req *agencyv1.DeleteVariableRequest,
) (*agencyv1.DeleteVariableResponse, error) {
	scope, err := scopeOf(req.GetTarget())
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, scope); err != nil {
		return nil, err
	}
	if err := s.Variables.Delete(ctx, scope, req.GetName()); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DeleteVariableResponse{}, nil
}

func (s *Server) ListEffectiveVariables(
	ctx context.Context,
	req *agencyv1.ListEffectiveVariablesRequest,
) (*agencyv1.ListEffectiveVariablesResponse, error) {
	actorID, err := strconv.ParseInt(req.GetAgentId(), 10, 64)
	if err != nil || actorID == 0 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("agent_id is required"),
		)
	}

	if err := s.requireAccess(ctx, variables.AgentScope(ids.ActorID(actorID))); err != nil {
		return nil, err
	}
	list, err := s.Variables.Effective(ctx, actorID, req.GetProjectId())
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.EffectiveVariable, 0, len(list))
	for _, resolved := range list {
		entry := &agencyv1.EffectiveVariable{
			Name:   resolved.Name,
			Note:   resolved.Note,
			Secret: resolved.Secret,
			Value:  resolved.Value,
			Source: toProtoTarget(resolved.Source),
		}
		for _, shadowed := range resolved.Shadowed {
			entry.Shadowed = append(entry.Shadowed, toProtoTarget(shadowed))
		}
		out = append(out, entry)
	}
	return &agencyv1.ListEffectiveVariablesResponse{Variables: out}, nil
}

func (s *Server) GetVariableCapabilities(
	_ context.Context,
	_ *agencyv1.GetVariableCapabilitiesRequest,
) (*agencyv1.GetVariableCapabilitiesResponse, error) {
	return &agencyv1.GetVariableCapabilitiesResponse{
		SecretsEnabled: s.Variables.SecretsEnabled(),
	}, nil
}

func (s *Server) requireAccess(ctx context.Context, scope variables.Scope) error {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if s.Access == nil {
		if identity.IsAdmin() {
			return nil
		}
		return connect.NewError(connect.CodePermissionDenied, variables.ErrForbidden)
	}
	allowed, err := s.Access.CanEdit(ctx, promptTarget(scope), identity)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if !allowed {
		return connect.NewError(connect.CodePermissionDenied, variables.ErrForbidden)
	}
	return nil
}

func (s *Server) actorNames(
	ctx context.Context,
	list []variables.Variable,
) map[ids.ActorID]store.Actor {
	seen := make([]ids.ActorID, 0, len(list))
	unique := map[ids.ActorID]bool{}
	for _, variable := range list {
		if variable.UpdatedBy == 0 || unique[variable.UpdatedBy] {
			continue
		}
		unique[variable.UpdatedBy] = true
		seen = append(seen, variable.UpdatedBy)
	}
	if len(seen) == 0 || s.Actors == nil {
		return nil
	}
	actors, err := s.Actors.ListActorsByIDs(ctx, seen)
	if err != nil {
		return nil
	}
	out := make(map[ids.ActorID]store.Actor, len(actors))
	for _, actor := range actors {
		out[ids.ActorID(actor.ID)] = actor
	}
	return out
}

func promptTarget(scope variables.Scope) promptaccess.Target {
	target := promptaccess.Target{}
	switch scope.Kind {
	case variables.ScopeOrg:
		target.Scope = prompts.ScopeOrg
	case variables.ScopeProject:
		target.Scope = prompts.ScopeProject
		target.ProjectID = scope.ID
	case variables.ScopeTeam:
		target.Scope = prompts.ScopeTeam
		target.TeamID, _ = strconv.ParseInt(scope.ID, 10, 64)
	case variables.ScopeGroup:
		target.Scope = prompts.ScopeGroup
		target.GroupID, _ = strconv.ParseInt(scope.ID, 10, 64)
	case variables.ScopeAgent:
		target.Scope = prompts.ScopeAgent
		actorID, _ := strconv.ParseInt(scope.ID, 10, 64)
		target.AgentID = ids.AgentID(actorID)
	}
	return target
}

func scopeOf(target *agencyv1.VariableTarget) (variables.Scope, error) {
	if target == nil {
		return variables.Scope{}, connect.NewError(
			connect.CodeInvalidArgument, errors.New("a target is required"),
		)
	}
	scope := variables.Scope{}
	switch target.GetScope() {
	case agencyv1.VariableScope_VARIABLE_SCOPE_ORG:
		scope = variables.OrgScope()
	case agencyv1.VariableScope_VARIABLE_SCOPE_PROJECT:
		scope = variables.ProjectScope(target.GetProjectId())
	case agencyv1.VariableScope_VARIABLE_SCOPE_TEAM:
		scope = variables.TeamScope(target.GetTeamId())
	case agencyv1.VariableScope_VARIABLE_SCOPE_GROUP:
		scope = variables.GroupScope(target.GetGroupId())
	case agencyv1.VariableScope_VARIABLE_SCOPE_AGENT:
		actorID, err := strconv.ParseInt(target.GetAgentId(), 10, 64)
		if err != nil || actorID == 0 {
			return variables.Scope{}, connect.NewError(
				connect.CodeInvalidArgument, errors.New("agent_id is required"),
			)
		}
		scope = variables.AgentScope(ids.ActorID(actorID))
	default:
		return variables.Scope{}, connect.NewError(
			connect.CodeInvalidArgument, errors.New("unknown variable scope"),
		)
	}
	if err := scope.Validate(); err != nil {
		return variables.Scope{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return scope, nil
}

func toProtoTarget(scope variables.Scope) *agencyv1.VariableTarget {
	target := &agencyv1.VariableTarget{}
	switch scope.Kind {
	case variables.ScopeOrg:
		target.Scope = agencyv1.VariableScope_VARIABLE_SCOPE_ORG
	case variables.ScopeProject:
		target.Scope = agencyv1.VariableScope_VARIABLE_SCOPE_PROJECT
		target.ProjectId = scope.ID
	case variables.ScopeTeam:
		target.Scope = agencyv1.VariableScope_VARIABLE_SCOPE_TEAM
		target.TeamId, _ = strconv.ParseInt(scope.ID, 10, 64)
	case variables.ScopeGroup:
		target.Scope = agencyv1.VariableScope_VARIABLE_SCOPE_GROUP
		target.GroupId, _ = strconv.ParseInt(scope.ID, 10, 64)
	case variables.ScopeAgent:
		target.Scope = agencyv1.VariableScope_VARIABLE_SCOPE_AGENT
		target.AgentId = scope.ID
	}
	return target
}

func toProto(variable variables.Variable, actor store.Actor) *agencyv1.Variable {
	out := &agencyv1.Variable{
		Name:      variable.Name,
		Note:      variable.Note,
		Secret:    variable.Secret,
		Target:    toProtoTarget(variable.Scope),
		CreatedAt: timestamppb.New(variable.CreatedAt),
		UpdatedAt: timestamppb.New(variable.UpdatedAt),
	}
	if !variable.Secret {
		out.Value = variable.Value
	}
	if actor.ID != 0 {
		out.UpdatedByUsername = actor.Username
		out.UpdatedByName = actor.Name
	}
	return out
}

func authActorID(ctx context.Context) ids.ActorID {
	if id, ok := auth.ActorID(ctx); ok {
		return id
	}
	return 0
}

func mapError(err error) error {
	switch {
	case errors.Is(err, variables.ErrInvalidName),
		errors.Is(err, variables.ErrReservedName),
		errors.Is(err, variables.ErrInvalidValue),
		errors.Is(err, variables.ErrNoteTooLong),
		errors.Is(err, variables.ErrInvalidScope),
		errors.Is(err, variables.ErrAlreadySecret):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, variables.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, variables.ErrExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, variables.ErrForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
