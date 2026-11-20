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

package variables

import (
	"context"
	"sort"
	"strconv"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

func rank(kind string) int {
	switch kind {
	case ScopeOrg:
		return 0
	case ScopeProject:
		return 1
	case ScopeTeam:
		return 2
	case ScopeGroup:
		return 3
	case ScopeAgent:
		return 4
	default:
		return -1
	}
}

func scopeRank(scope Scope) (int, int64, string) {
	id, err := strconv.ParseInt(scope.ID, 10, 64)
	if err != nil {
		return rank(scope.Kind), 0, scope.ID
	}
	return rank(scope.Kind), id, ""
}

func less(a, b Scope) bool {
	aRank, aID, aText := scopeRank(a)
	bRank, bID, bText := scopeRank(b)
	if aRank != bRank {
		return aRank < bRank
	}
	if aID != bID {
		return aID < bID
	}
	return aText < bText
}

func (s *Service) Resolve(ctx context.Context, caller Caller) ([]Resolved, error) {
	scopes := []Scope{OrgScope()}
	if caller.ProjectID != "" {
		scopes = append(scopes, ProjectScope(caller.ProjectID))
	}
	if caller.AgentActorID != 0 {
		teams, err := s.store.ListTeamIDsForAgentActor(ctx, caller.AgentActorID)
		if err != nil {
			return nil, err
		}
		for _, teamID := range teams {
			scopes = append(scopes, TeamScope(teamID))
		}
		groups, err := s.store.ListAgentGroupIDsForAgentActor(ctx, caller.AgentActorID)
		if err != nil {
			return nil, err
		}
		for _, groupID := range groups {
			scopes = append(scopes, GroupScope(groupID))
		}
		scopes = append(scopes, AgentScope(actorIDOf(caller.AgentActorID)))
	}
	return s.resolveScopes(ctx, scopes)
}

func (s *Service) Effective(
	ctx context.Context,
	agentActorID int64,
	projectID string,
) ([]Resolved, error) {
	resolved, err := s.Resolve(ctx, Caller{
		AgentActorID: agentActorID,
		ProjectID:    projectID,
	})
	if err != nil {
		return nil, err
	}
	own := AgentScope(actorIDOf(agentActorID))
	for i := range resolved {
		if resolved[i].Secret || resolved[i].Source != own {
			resolved[i].Value = ""
		}
	}
	return resolved, nil
}

func (s *Service) resolveScopes(ctx context.Context, scopes []Scope) ([]Resolved, error) {
	all, err := s.store.ListVariables(ctx)
	if err != nil {
		return nil, err
	}
	wanted := make(map[Scope]bool, len(scopes))
	for _, scope := range scopes {
		wanted[scope] = true
	}

	byName := map[string][]Variable{}
	for _, row := range all {
		variable := variableOf(row)
		if !wanted[variable.Scope] {
			continue
		}
		byName[variable.Name] = append(byName[variable.Name], variable)
	}

	out := make([]Resolved, 0, len(byName))
	for name, definitions := range byName {
		sort.Slice(definitions, func(i, j int) bool {
			return less(definitions[i].Scope, definitions[j].Scope)
		})
		winner := definitions[len(definitions)-1]
		resolved := Resolved{
			Name:   name,
			Note:   winner.Note,
			Secret: winner.Secret,
			Source: winner.Scope,
		}
		if winner.Secret {
			if resolved.Value, err = s.sealer.Open(winner.sealed); err != nil {
				return nil, err
			}
		} else {
			resolved.Value = winner.Value
		}
		for _, shadowed := range definitions[:len(definitions)-1] {
			resolved.Shadowed = append(resolved.Shadowed, shadowed.Scope)
		}
		out = append(out, resolved)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func actorIDOf(actorID int64) ids.ActorID { return ids.ActorID(actorID) }
