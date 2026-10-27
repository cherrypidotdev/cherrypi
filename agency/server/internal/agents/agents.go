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
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
	sharedprotocol "cherrypi.vladfaust.com/shared/protocol"
)

var (
	ErrInvalidUsername = errors.New(
		"username must be non-empty and contain no whitespace, ':' or ','",
	)

	ErrAgentExists = errors.New("agent username already exists")

	ErrAgentMissing = errors.New("agent not found")

	ErrInvalidRole = errors.New("agent role must be 'member' or 'admin'")
)

const (
	RoleMember = "member"
	RoleAdmin  = "admin"
)

type Status int64

const (
	StatusOffline Status = 1
	StatusOnline  Status = 2
)

type VersionStatus int64

const (
	VersionStatusUnspecified VersionStatus = 0

	// line and at or ahead of the agency's version.
	VersionStatusCurrent VersionStatus = 1
	// VersionStatusOutdated means the same compatibility line but an older

	VersionStatusOutdated VersionStatus = 2
	// VersionStatusIncompatible means a different compatibility line or no

	VersionStatusIncompatible VersionStatus = 3
)

type Agent struct {
	ID          ids.AgentID
	ActorID     ids.ActorID
	Username    string
	DisplayName string
	Description string

	Role            string
	Status          Status
	Version         string
	ProtocolVersion string
	VersionStatus   VersionStatus
	RemoteAddr      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastSeenAt      time.Time
}

type presenceInfo struct {
	online          bool
	buildVersion    string
	protocolVersion string
	remoteAddr      string
}

type Registry struct {
	store store.Store

	Audit *audit.Service

	mu       sync.RWMutex
	presence map[ids.AgentID]presenceInfo
}

func NewRegistry(st store.Store) *Registry {
	return &Registry{store: st, presence: map[ids.AgentID]presenceInfo{}}
}

func (r *Registry) List(ctx context.Context) ([]Agent, error) {
	rows, err := r.store.ListAgents(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Agent, len(rows))
	for i, row := range rows {
		out[i] = r.toAgent(row)
	}
	return out, nil
}

func (r *Registry) Get(ctx context.Context, id ids.AgentID) (Agent, error) {
	row, err := r.store.GetAgent(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Agent{}, ErrAgentMissing
	}
	if err != nil {
		return Agent{}, err
	}
	return r.toAgent(row), nil
}

func (r *Registry) GetByActor(ctx context.Context, actorID ids.ActorID) (Agent, error) {
	row, err := r.store.GetAgentByActor(ctx, actorID)
	if errors.Is(err, sql.ErrNoRows) {
		return Agent{}, ErrAgentMissing
	}
	if err != nil {
		return Agent{}, err
	}
	return r.toAgent(row), nil
}

func (r *Registry) Create(
	ctx context.Context,
	username, displayName, description string,
) (Agent, error) {
	return r.CreateOwned(ctx, username, displayName, description, 0)
}

func (r *Registry) CreateOwned(
	ctx context.Context,
	username, displayName, description string,
	ownerActorID ids.ActorID,
) (Agent, error) {
	username = strings.TrimSpace(username)
	if username == "" || strings.ContainsAny(username, ":, \t\r\n") {
		return Agent{}, ErrInvalidUsername
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = username
	}

	if _, err := r.store.GetAgentByUsername(ctx, username); err == nil {
		return Agent{}, ErrAgentExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Agent{}, err
	}

	now := time.Now()
	row, err := r.store.RegisterAgent(ctx, store.RegisterAgentParams{
		Username:     username,
		DisplayName:  displayName,
		Description:  strings.TrimSpace(description),
		OwnerActorID: ownerActorID,
		CreatedAt:    now.UnixMilli(),
		UpdatedAt:    now.UnixMilli(),
	})
	if err != nil {
		return Agent{}, err
	}

	r.logAgent(ctx, row.ActorID, audit.KindCreated, "username", "", username,
		"Created agent "+username, nil)
	return r.toAgent(row), nil
}

func (r *Registry) SetRole(
	ctx context.Context,
	id ids.AgentID,
	role string,
) (Agent, error) {
	if role != RoleMember && role != RoleAdmin {
		return Agent{}, ErrInvalidRole
	}
	row, err := r.store.GetAgent(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Agent{}, ErrAgentMissing
	}
	if err != nil {
		return Agent{}, err
	}
	if err := r.store.SetAgentRole(ctx, id, role); err != nil {
		return Agent{}, err
	}
	if row.Role != role {
		r.logAgent(ctx, row.ActorID, audit.KindRole, "role", row.Role, role,
			"Changed "+row.Username+" role to "+role, nil)
	}
	return r.Get(ctx, id)
}

func (r *Registry) Update(
	ctx context.Context,
	id ids.AgentID,
	displayName, description string,
) (Agent, error) {
	displayName = strings.TrimSpace(displayName)
	row, err := r.store.GetAgent(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Agent{}, ErrAgentMissing
	}
	if err != nil {
		return Agent{}, err
	}
	if displayName == "" {
		displayName = row.Name
	}

	updated, err := r.store.UpdateAgent(
		ctx, id, displayName, strings.TrimSpace(description), time.Now().UnixMilli(),
	)
	if err != nil {
		return Agent{}, err
	}
	r.logAgent(ctx, updated.ActorID, audit.KindUpdated, "name", row.Name,
		displayName, "Updated agent "+updated.Username, nil)
	return r.toAgent(updated), nil
}

func (r *Registry) Delete(ctx context.Context, id ids.AgentID) error {
	agent, err := r.store.GetAgent(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAgentMissing
	} else if err != nil {
		return err
	}

	if err := r.store.DeleteAgentLogs(ctx, id); err != nil {
		return err
	}
	if err := r.store.DeleteAgentDomainLogs(ctx, int64(agent.ActorID)); err != nil {
		return err
	}
	if err := r.store.DeleteAgent(ctx, id); err != nil {
		return err
	}
	r.SetOffline(id)
	return nil
}

func (r *Registry) SetOnline(
	agentID ids.AgentID,
	buildVersion, protocolVersion, remoteAddr string,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.presence[agentID] = presenceInfo{
		online:          true,
		buildVersion:    buildVersion,
		protocolVersion: protocolVersion,
		remoteAddr:      remoteAddr,
	}
}

func (r *Registry) Reject(
	agentID ids.AgentID,
	buildVersion, protocolVersion, remoteAddr string,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.presence[agentID] = presenceInfo{
		buildVersion:    buildVersion,
		protocolVersion: protocolVersion,
		remoteAddr:      remoteAddr,
	}
}

func (r *Registry) SetOffline(agentID ids.AgentID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.presence, agentID)
}

func (r *Registry) presenceOf(agentID ids.AgentID) (presenceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.presence[agentID]
	return p, ok
}

func versionStatus(p presenceInfo) VersionStatus {
	switch sharedprotocol.Compare(p.protocolVersion, sharedprotocol.Version) {
	case sharedprotocol.Current:
		return VersionStatusCurrent
	case sharedprotocol.Outdated:
		return VersionStatusOutdated
	default:
		return VersionStatusIncompatible
	}
}

func (r *Registry) toAgent(row store.AgentRow) Agent {
	a := Agent{
		ID:          row.ID,
		ActorID:     row.ActorID,
		Username:    row.Username,
		DisplayName: row.Name,
		Description: row.Description,
		Role:        row.Role,
		Status:      StatusOffline,
		CreatedAt:   time.UnixMilli(row.CreatedAt),
		UpdatedAt:   time.UnixMilli(row.UpdatedAt),
	}
	if row.LastSeenAt.Valid {
		a.LastSeenAt = time.UnixMilli(row.LastSeenAt.Int64)
	}
	if p, ok := r.presenceOf(row.ID); ok {
		if p.online {
			a.Status = StatusOnline
		}
		a.Version = p.buildVersion
		a.ProtocolVersion = p.protocolVersion
		a.VersionStatus = versionStatus(p)
		a.RemoteAddr = p.remoteAddr
	}
	return a
}
