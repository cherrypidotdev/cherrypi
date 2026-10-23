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

// Package audit is the agency's domain-log surface (auth phase 7). There is no
// central event table: each object scope keeps its own log (task, agent,
// project, team, user) and this package unions them for the admin explorer,
// prunes them on retention, and records the two scopes the services write to
// directly (user and agent).
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

// Scope names one of the five domain logs.
type Scope string

// The five scopes. They are the explorer's `scope` filter values.
const (
	ScopeTask    Scope = "task"
	ScopeAgent   Scope = "agent"
	ScopeProject Scope = "project"
	ScopeTeam    Scope = "team"
	ScopeUser    Scope = "user"
)

// SettingKeyRetentionDays stores the audit retention in the settings table.
const SettingKeyRetentionDays = "audit.retention_days"

// DefaultRetentionDays is the locked default (docs/auth.md).
const DefaultRetentionDays = 90

// Lifecycle kinds are visible to a plain member even when another actor made
// the entry; the locked scoped-visibility rule keeps every other entry to its
// author.
var lifecycleKinds = map[string]bool{
	"created":    true,
	"deleted":    true,
	"archived":   true,
	"unarchived": true,
}

// IsLifecycle reports whether a kind is always visible in a scoped log.
func IsLifecycle(kind string) bool { return lifecycleKinds[kind] }

// ScopedVisible applies the locked scoped-visibility rule to one entry: an
// owner or administrator sees everything; a plain member sees their own entries
// plus lifecycle events.
func ScopedVisible(actorID, caller ids.ActorID, full bool, kind string) bool {
	if full {
		return true
	}
	if actorID != 0 && actorID == caller {
		return true
	}
	return IsLifecycle(kind)
}

// Entry is one domain-log row from any scope.
type Entry struct {
	ID            int64
	Scope         Scope
	ScopeID       string
	ScopeLabel    string
	At            time.Time
	ActorID       ids.ActorID
	ActorKind     string
	ActorUsername string
	ActorName     string
	Kind          string
	Field         string
	OldValue      string
	NewValue      string
	Message       string
	DataJSON      string
}

// Service reads and writes the domain logs.
type Service struct {
	store store.Store
}

// New builds the audit service over the agency store.
func New(st store.Store) *Service { return &Service{store: st} }

// LogUser appends one entry to a user's domain log. userActorID is the user the
// entry is about; actorID is who acted (0 falls back to the request identity).
// A write failure is logged and never fails the mutation it describes.
func (s *Service) LogUser(
	ctx context.Context,
	userActorID, actorID ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	if actorID == 0 {
		actorID = ActorID(ctx)
	}
	params := store.InsertUserLogParams{
		UserActorID: int64(userActorID),
		At:          time.Now().UnixMilli(),
		ActorID:     nullableActor(actorID),
		Kind:        kind,
		Field:       field,
		OldValue:    oldValue,
		NewValue:    newValue,
		Message:     message,
		DataJson:    marshalData(data),
	}
	if err := s.store.InsertUserLog(ctx, params); err != nil {
		log.Printf("failed to write user log for actor %d: %v", userActorID, err)
	}
}

// LogAgent appends one entry to an agent's domain log.
func (s *Service) LogAgent(
	ctx context.Context,
	agentActorID, actorID ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	if actorID == 0 {
		actorID = ActorID(ctx)
	}
	params := store.InsertAgentDomainLogParams{
		AgentActorID: int64(agentActorID),
		At:           time.Now().UnixMilli(),
		ActorID:      nullableActor(actorID),
		Kind:         kind,
		Field:        field,
		OldValue:     oldValue,
		NewValue:     newValue,
		Message:      message,
		DataJson:     marshalData(data),
	}
	if err := s.store.InsertAgentDomainLog(ctx, params); err != nil {
		log.Printf("failed to write agent log for actor %d: %v", agentActorID, err)
	}
}

// ActorID returns the acting actor from the request context, or 0.
func ActorID(ctx context.Context) ids.ActorID {
	if id, ok := auth.ActorID(ctx); ok {
		return id
	}
	return 0
}

// RetentionDays reads the configured retention; a missing or invalid value is
// the 90-day default.
func (s *Service) RetentionDays(ctx context.Context) int {
	rows, err := s.store.ListSettings(ctx)
	if err != nil {
		return DefaultRetentionDays
	}
	for _, row := range rows {
		if row.Key != SettingKeyRetentionDays {
			continue
		}
		days, err := strconv.Atoi(row.Value)
		if err == nil && days > 0 {
			return days
		}
	}
	return DefaultRetentionDays
}

// SetRetentionDays persists the retention in days. It must be positive.
func (s *Service) SetRetentionDays(ctx context.Context, days int) error {
	if days <= 0 {
		days = DefaultRetentionDays
	}
	return s.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   SettingKeyRetentionDays,
		Value: strconv.Itoa(days),
	})
}

// Prune deletes every domain-log entry older than the configured retention and
// reports how many days it used.
func (s *Service) Prune(ctx context.Context, now time.Time) (int, error) {
	days := s.RetentionDays(ctx)
	before := now.AddDate(0, 0, -days).UnixMilli()
	if err := s.store.PruneDomainLogs(ctx, before); err != nil {
		return days, err
	}
	return days, nil
}

func nullableActor(id ids.ActorID) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(id), Valid: id != 0}
}

func marshalData(data any) string {
	if data == nil {
		return ""
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(raw)
}
