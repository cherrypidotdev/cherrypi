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

package agents

import (
	"cherrypi.vladfaust.com/agency/internal/store"
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

var (
	// ErrOwnerMissing is returned when removing an owner that is not set.
	ErrOwnerMissing = errors.New("agent owner not found")
	// ErrOwnerUnknownActor is returned when the owner username is unknown.
	ErrOwnerUnknownActor = errors.New("owner actor not found")
)

// Owner is one actor that may manage an agent.
type Owner struct {
	ActorID  ids.ActorID
	Kind     string
	Username string
	Name     string
	AddedAt  time.Time
}

// Owners lists an agent's owners.
func (r *Registry) Owners(ctx context.Context, agentID ids.AgentID) ([]Owner, error) {
	agent, err := r.store.GetAgent(ctx, agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAgentMissing
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.store.ListAgentOwners(ctx, agent.ActorID)
	if err != nil {
		return nil, err
	}
	out := make([]Owner, len(rows))
	for i, row := range rows {
		out[i] = Owner{
			ActorID:  ids.ActorID(row.OwnerActorID),
			Kind:     row.ActorKind,
			Username: row.ActorUsername,
			Name:     row.ActorName,
			AddedAt:  time.UnixMilli(row.CreatedAt),
		}
	}
	return out, nil
}

// AddOwner makes ownerActorID an owner of the agent. Adding an existing owner
// is a no-op (idempotent).
func (r *Registry) AddOwner(
	ctx context.Context,
	agentID ids.AgentID,
	ownerActorID ids.ActorID,
) error {
	agent, err := r.store.GetAgent(ctx, agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAgentMissing
	}
	if err != nil {
		return err
	}
	if _, err := r.store.GetActorByID(ctx, ownerActorID); errors.Is(err, sql.ErrNoRows) {
		return ErrOwnerUnknownActor
	} else if err != nil {
		return err
	}
	if err := r.store.AddAgentOwner(ctx, store.AddAgentOwnerParams{
		AgentActorID: int64(agent.ActorID),
		OwnerActorID: int64(ownerActorID),
		CreatedAt:    time.Now().UnixMilli(),
	}); err != nil {
		return err
	}
	r.logAgent(ctx, agent.ActorID, audit.KindOwnerAdd, "owner_actor_id", "",
		actorIDLabel(ownerActorID), "Added an owner", nil)
	return nil
}

// AddOwnerByUsername resolves an owner by its globally-unique username.
func (r *Registry) AddOwnerByUsername(
	ctx context.Context,
	agentID ids.AgentID,
	username string,
) (Owner, error) {
	actor, err := r.store.GetActorByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return Owner{}, ErrOwnerUnknownActor
	}
	if err != nil {
		return Owner{}, err
	}
	if err := r.AddOwner(ctx, agentID, ids.ActorID(actor.ID)); err != nil {
		return Owner{}, err
	}
	return Owner{
		ActorID:  ids.ActorID(actor.ID),
		Kind:     actor.Kind,
		Username: actor.Username,
		Name:     actor.Name,
		AddedAt:  time.Now(),
	}, nil
}

// RemoveOwner drops an owner. Removing the last owner is allowed: the agent
// then falls back to admin-only management.
func (r *Registry) RemoveOwner(
	ctx context.Context,
	agentID ids.AgentID,
	ownerActorID ids.ActorID,
) error {
	agent, err := r.store.GetAgent(ctx, agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAgentMissing
	}
	if err != nil {
		return err
	}
	n, err := r.store.RemoveAgentOwner(ctx, store.RemoveAgentOwnerParams{
		AgentActorID: int64(agent.ActorID),
		OwnerActorID: int64(ownerActorID),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrOwnerMissing
	}
	r.logAgent(ctx, agent.ActorID, audit.KindOwnerRemove, "owner_actor_id",
		actorIDLabel(ownerActorID), "", "Removed an owner", nil)
	return nil
}

func actorIDLabel(id ids.ActorID) string {
	return strconv.FormatInt(int64(id), 10)
}

// IsOwner reports whether actorID owns the agent.
func (r *Registry) IsOwner(
	ctx context.Context,
	agentID ids.AgentID,
	actorID ids.ActorID,
) (bool, error) {
	agent, err := r.store.GetAgent(ctx, agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrAgentMissing
	}
	if err != nil {
		return false, err
	}
	return r.IsActorOwner(ctx, agent.ActorID, actorID)
}

// IsActorOwner reports ownership keyed on the agent's actor id.
func (r *Registry) IsActorOwner(
	ctx context.Context,
	agentActorID, actorID ids.ActorID,
) (bool, error) {
	return r.store.IsAgentOwner(ctx, agentActorID, actorID)
}

// OwnerActorIDs lists the agent actor ids an actor owns.
func (r *Registry) OwnerActorIDs(
	ctx context.Context,
	ownerActorID ids.ActorID,
) ([]ids.ActorID, error) {
	rows, err := r.store.ListAgentsOwnedBy(ctx, ownerActorID)
	if err != nil {
		return nil, err
	}
	out := make([]ids.ActorID, len(rows))
	for i, id := range rows {
		out[i] = ids.ActorID(id)
	}
	return out, nil
}
