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

package integrations

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"strconv"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

func errNoRows() error { return sql.ErrNoRows }

func base64Std(raw []byte) string { return base64.StdEncoding.EncodeToString(raw) }

func base64Decode(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

type Authorizer struct {
	IsAdmin func(ctx context.Context, actorID int64) bool

	ProjectRole func(
		ctx context.Context,
		projectID string,
		actorID ids.ActorID,
		isAdmin bool,
	) (string, bool, error)

	TeamRole func(ctx context.Context, teamID int64, actorID ids.ActorID) (string, bool, error)

	GroupOwner func(ctx context.Context, groupID int64, actorID ids.ActorID) (bool, error)

	AgentOwner func(ctx context.Context, agentActorID int64, actorID ids.ActorID) (bool, error)
}

func (a *Authorizer) CanAuthor(ctx context.Context, actorID int64, scope Scope) error {
	if a == nil {
		return ErrForbidden
	}
	if actorID != 0 && a.IsAdmin != nil && a.IsAdmin(ctx, actorID) {
		return nil
	}
	scope = normalizeScope(scope)
	switch scope.Kind {
	case KindOrg:

		return ErrForbidden
	case KindProject:
		role, ok, err := a.projectRole(ctx, scope.ID, actorID)
		if err != nil {
			return err
		}
		if ok && role == "owner" {
			return nil
		}
		return ErrForbidden
	case KindTeam:
		role, ok, err := a.teamRole(ctx, scope.ID, actorID)
		if err != nil {
			return err
		}
		if ok && role == "owner" {
			return nil
		}
		return ErrForbidden
	case KindGroup:
		owner, err := a.groupOwner(ctx, scope.ID, actorID)
		if err != nil {
			return err
		}
		if owner {
			return nil
		}
		return ErrForbidden
	case KindAgent:
		owner, err := a.agentOwner(ctx, scope.ID, actorID)
		if err != nil {
			return err
		}
		if owner {
			return nil
		}
		return ErrForbidden
	default:
		return ErrForbidden
	}
}

func (a *Authorizer) CanAdminister(ctx context.Context, actorID int64, scope Scope) error {
	if a == nil {
		return ErrForbidden
	}
	if actorID != 0 && a.IsAdmin != nil && a.IsAdmin(ctx, actorID) {
		return nil
	}
	scope = normalizeScope(scope)
	if scope.Kind == KindOrg {
		return ErrForbidden
	}
	return a.CanAuthor(ctx, actorID, scope)
}

func (a *Authorizer) CanSee(ctx context.Context, actorID int64, scope Scope) (bool, error) {
	if a == nil {
		return false, nil
	}
	if actorID != 0 && a.IsAdmin != nil && a.IsAdmin(ctx, actorID) {
		return true, nil
	}
	scope = normalizeScope(scope)
	switch scope.Kind {
	case KindOrg:
		return true, nil
	case KindProject:
		_, ok, err := a.projectRole(ctx, scope.ID, actorID)
		return ok, err
	case KindTeam:
		_, ok, err := a.teamRole(ctx, scope.ID, actorID)
		return ok, err
	case KindGroup:

		return true, nil
	case KindAgent:
		owner, err := a.agentOwner(ctx, scope.ID, actorID)
		if err != nil {
			return false, err
		}
		if owner {
			return true, nil
		}

		return false, nil
	default:
		return false, nil
	}
}

func (a *Authorizer) projectRole(
	ctx context.Context,
	projectID string,
	actorID int64,
) (string, bool, error) {
	if a.ProjectRole == nil {
		return "", false, nil
	}
	return a.ProjectRole(ctx, projectID, ids.ActorID(actorID), false)
}

func (a *Authorizer) teamRole(
	ctx context.Context,
	rawID string,
	actorID int64,
) (string, bool, error) {
	if a.TeamRole == nil {
		return "", false, nil
	}
	teamID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return "", false, err
	}
	return a.TeamRole(ctx, teamID, ids.ActorID(actorID))
}

func (a *Authorizer) groupOwner(
	ctx context.Context,
	rawID string,
	actorID int64,
) (bool, error) {
	if a.GroupOwner == nil {
		return false, nil
	}
	groupID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return false, err
	}
	return a.GroupOwner(ctx, groupID, ids.ActorID(actorID))
}

func (a *Authorizer) agentOwner(
	ctx context.Context,
	rawID string,
	actorID int64,
) (bool, error) {
	if a.AgentOwner == nil {
		return false, nil
	}
	agentActorID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return false, err
	}
	return a.AgentOwner(ctx, agentActorID, ids.ActorID(actorID))
}

func IsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound)
}
