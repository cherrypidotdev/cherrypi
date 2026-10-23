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

// Package agents is the agency's agent registry: the fleet's metadata,
// per-agent tunnel credentials, and in-memory presence. Presence is derived
// from the tunnel hub (Phase C); until then every agent reports offline.
package agents

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
	sharedprotocol "cherrypi.vladfaust.com/shared/protocol"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidUsername is returned for empty or unsafe usernames.
	ErrInvalidUsername = errors.New(
		"username must be non-empty and contain no whitespace, ':' or ','",
	)
	// ErrAgentExists is returned when the username is taken.
	ErrAgentExists = errors.New("agent username already exists")
	// ErrAgentMissing is returned when an agent id/username is unknown.
	ErrAgentMissing = errors.New("agent not found")
	// ErrCredentialExists is returned when a credential username is taken.
	ErrCredentialExists = errors.New("credential already exists")
	// ErrCredentialMissing is returned when revoking an unknown credential.
	ErrCredentialMissing = errors.New("credential not found")
)

// Status mirrors agency.v1.AgentStatus. Never renumber.
type Status int64

const (
	StatusOffline Status = 1
	StatusOnline  Status = 2
)

// VersionStatus mirrors agency.v1.AgentVersionStatus. Never renumber.
type VersionStatus int64

const (
	// VersionStatusUnspecified means the agent has never reported a version
	// (it is offline and was not refused).
	VersionStatusUnspecified VersionStatus = 0
	// VersionStatusCurrent means the agent is on the agency's compatibility
	// line and at or ahead of the agency's version.
	VersionStatusCurrent VersionStatus = 1
	// VersionStatusOutdated means the same compatibility line but an older
	// version.
	VersionStatusOutdated VersionStatus = 2
	// VersionStatusIncompatible means a different compatibility line or no
	// version.
	VersionStatusIncompatible VersionStatus = 3
)

// Agent is the public view of a registered worker.
type Agent struct {
	ID              ids.AgentID
	ActorID         ids.ActorID
	Username        string
	DisplayName     string
	Description     string
	Status          Status
	Version         string
	ProtocolVersion string
	VersionStatus   VersionStatus
	RemoteAddr      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastSeenAt      time.Time
}

// Credential is an agent tunnel credential (password omitted).
type Credential struct {
	Username   string
	CreatedAt  time.Time
	LastUsedAt time.Time
}

// presenceInfo is the in-memory state of an agent's most recent connection
// attempt: the versions and address it reported, and whether the tunnel was
// accepted. A refused attempt is kept (online=false, versions as reported) so
// the UI can say why the agent is offline.
type presenceInfo struct {
	online          bool
	buildVersion    string
	protocolVersion string
	remoteAddr      string
}

// Registry owns the agent table, its credentials, and presence.
type Registry struct {
	store store.Store

	// Audit records agent mutations in the agent domain log. Optional: nil in
	// tests.
	Audit *audit.Service

	mu       sync.RWMutex
	presence map[ids.AgentID]presenceInfo
}

// NewRegistry builds the registry.
func NewRegistry(st store.Store) *Registry {
	return &Registry{store: st, presence: map[ids.AgentID]presenceInfo{}}
}

// List returns every agent, sorted by display name.
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

// Get returns one agent by its agent id.
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

// GetByActor returns the agent that owns an actor, or ErrAgentMissing.
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

// Create registers an agent and generates its first credential. The plaintext
// password is returned exactly once. It is CreateOwned with no owner.
func (r *Registry) Create(
	ctx context.Context,
	username, displayName, description string,
) (Agent, string, error) {
	return r.CreateOwned(ctx, username, displayName, description, 0)
}

// CreateOwned registers an agent, generates its first credential and makes
// ownerActorID its first owner (when non-zero). The plaintext password is
// returned exactly once. The ownership row and the agent share a transaction,
// so a newly created agent is never ownerless when the caller had an owner.
func (r *Registry) CreateOwned(
	ctx context.Context,
	username, displayName, description string,
	ownerActorID ids.ActorID,
) (Agent, string, error) {
	username = strings.TrimSpace(username)
	if username == "" || strings.ContainsAny(username, ":, \t\r\n") {
		return Agent{}, "", ErrInvalidUsername
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = username
	}

	if _, err := r.store.GetAgentByUsername(ctx, username); err == nil {
		return Agent{}, "", ErrAgentExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Agent{}, "", err
	}

	password, err := generatePassword()
	if err != nil {
		return Agent{}, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Agent{}, "", err
	}

	now := time.Now()
	row, err := r.store.RegisterAgent(ctx, store.RegisterAgentParams{
		Username:               username,
		DisplayName:            displayName,
		Description:            strings.TrimSpace(description),
		OwnerActorID:           ownerActorID,
		CredentialUsername:     username,
		CredentialPasswordHash: string(hash),
		CreatedAt:              now.UnixMilli(),
		UpdatedAt:              now.UnixMilli(),
	})
	if err != nil {
		return Agent{}, "", err
	}

	r.logAgent(ctx, row.ActorID, audit.KindCreated, "username", "", username,
		"Created agent "+username, nil)
	return r.toAgent(row), password, nil
}

// Update changes an agent's display name and description.
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

// Delete removes an agent and its credentials (cascade).
func (r *Registry) Delete(ctx context.Context, id ids.AgentID) error {
	agent, err := r.store.GetAgent(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAgentMissing
	} else if err != nil {
		return err
	}
	// Agent logs are not foreign-keyed, so drop them with the agent. The
	// domain log is scoped to the agent actor and goes too.
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

// ListCredentials lists an agent's tunnel credentials.
func (r *Registry) ListCredentials(ctx context.Context, agentID ids.AgentID) ([]Credential, error) {
	rows, err := r.store.ListAgentCredentials(ctx, agentID)
	if err != nil {
		return nil, err
	}
	out := make([]Credential, len(rows))
	for i, row := range rows {
		out[i] = toCredential(row)
	}
	return out, nil
}

// CreateCredential adds a tunnel credential and returns its plaintext password
// exactly once.
func (r *Registry) CreateCredential(
	ctx context.Context,
	agentID ids.AgentID, username string,
) (Credential, string, error) {
	agent, err := r.store.GetAgent(ctx, agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return Credential{}, "", ErrAgentMissing
	}
	if err != nil {
		return Credential{}, "", err
	}

	username = strings.TrimSpace(username)
	if username == "" {
		// Derive a free username from the agent's (base, base-2, base-3, …) so
		// adding a second credential does not collide with the first.
		username, err = r.freeCredentialUsername(ctx, agent.Username)
		if err != nil {
			return Credential{}, "", err
		}
	}
	if strings.ContainsAny(username, ":, \t\r\n") {
		return Credential{}, "", ErrInvalidUsername
	}
	if _, err := r.store.GetAgentCredential(ctx, username); err == nil {
		return Credential{}, "", ErrCredentialExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Credential{}, "", err
	}

	password, err := generatePassword()
	if err != nil {
		return Credential{}, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Credential{}, "", err
	}

	row, err := r.store.CreateAgentCredential(ctx, store.CreateAgentCredentialParams{
		AgentID:      int64(agentID),
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UnixMilli(),
	})
	if err != nil {
		return Credential{}, "", err
	}
	r.logAgent(ctx, agent.ActorID, audit.KindCredentialCreate, "username", "",
		username, "Added tunnel credential "+username, nil)
	return toCredential(row), password, nil
}

// freeCredentialUsername returns base if unused, otherwise base-2, base-3, ….
// It is the default when adding a credential without naming one.
func (r *Registry) freeCredentialUsername(ctx context.Context, base string) (string, error) {
	for i := 1; i <= 100; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		if _, err := r.store.GetAgentCredential(ctx, candidate); errors.Is(err, sql.ErrNoRows) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", ErrCredentialExists
}

// RevokeCredential removes a tunnel credential.
func (r *Registry) RevokeCredential(
	ctx context.Context,
	agentID ids.AgentID,
	username string,
) error {
	n, err := r.store.RevokeAgentCredential(ctx, store.RevokeAgentCredentialParams{
		AgentID:  int64(agentID),
		Username: username,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrCredentialMissing
	}
	if agent, err := r.store.GetAgent(ctx, agentID); err == nil {
		r.logAgent(ctx, agent.ActorID, audit.KindCredentialRevoke, "username",
			username, "", "Revoked tunnel credential "+username, nil)
	}
	return nil
}

// Authenticate validates a tunnel credential and returns the agent id. It also
// records last-used and last-seen timestamps.
func (r *Registry) Authenticate(
	ctx context.Context,
	username, password string,
) (ids.AgentID, bool) {
	row, err := r.store.GetAgentCredential(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	if err != nil {
		return 0, false
	}
	if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)) != nil {
		return 0, false
	}

	now := time.Now().UnixMilli()
	_ = r.store.TouchAgentCredentialUsed(ctx, store.TouchAgentCredentialUsedParams{
		LastUsedAt: sql.NullInt64{Int64: now, Valid: true},
		Username:   username,
	})
	agentID := ids.AgentID(row.AgentID)
	_ = r.store.TouchAgentSeen(ctx, agentID, now)

	// Presence is derived solely from the tunnel hub, which records the
	// version report: a REST request must not make a refused agent look online.
	return agentID, true
}

// EnsureCredential creates an agent (if missing) and ensures a tunnel
// credential with the given plaintext password exists, updating the password if
// it differs. It backs the CHERRYPI_AGENCY_SEED_AGENT dev bootstrap, where the
// credential must be known before the agent connects.
func (r *Registry) EnsureCredential(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || strings.ContainsAny(username, ":, \t\r\n") {
		return ErrInvalidUsername
	}
	if password == "" {
		return ErrInvalidUsername
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()

	agent, err := r.store.GetAgentByUsername(ctx, username)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Register the agent and its first credential atomically: a failure
		// must not leave an agent that the seed credential cannot reach.
		_, err = r.store.RegisterAgent(ctx, store.RegisterAgentParams{
			Username:               username,
			DisplayName:            username,
			CredentialUsername:     username,
			CredentialPasswordHash: string(hash),
			CreatedAt:              now,
			UpdatedAt:              now,
		})
		return err
	case err != nil:
		return err
	}

	cred, err := r.store.GetAgentCredential(ctx, username)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = r.store.CreateAgentCredential(ctx, store.CreateAgentCredentialParams{
			AgentID:      int64(agent.ID),
			Username:     username,
			PasswordHash: string(hash),
			CreatedAt:    now,
		})
		return err
	case err != nil:
		return err
	}

	if bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password)) == nil {
		return nil
	}
	return r.store.UpdateAgentCredentialPassword(ctx, store.UpdateAgentCredentialPasswordParams{
		PasswordHash: string(hash),
		Username:     username,
	})
}

// SetOnline records an accepted agent connection and the versions it reported
// in the tunnel hello.
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

// Reject records a refused agent connection and the versions it reported, so
// the UI can explain the incompatibility. The agent is not marked online.
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

// SetOffline marks an agent as disconnected.
func (r *Registry) SetOffline(agentID ids.AgentID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.presence, agentID)
}

// presenceOf returns the tracked state for an agent, if any.
func (r *Registry) presenceOf(agentID ids.AgentID) (presenceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.presence[agentID]
	return p, ok
}

// versionStatus derives the compatibility verdict for a tracked presence.
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

func toCredential(row store.AgentCredential) Credential {
	c := Credential{
		Username:  row.Username,
		CreatedAt: time.UnixMilli(row.CreatedAt),
	}
	if row.LastUsedAt.Valid {
		c.LastUsedAt = time.UnixMilli(row.LastUsedAt.Int64)
	}
	return c
}

// generatePassword returns 32 hex characters (128 bits) from crypto/rand.
func generatePassword() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
