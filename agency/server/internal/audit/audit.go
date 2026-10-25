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

type Scope string

const (
	ScopeTask    Scope = "task"
	ScopeAgent   Scope = "agent"
	ScopeProject Scope = "project"
	ScopeTeam    Scope = "team"
	ScopeUser    Scope = "user"
)

const SettingKeyRetentionDays = "audit.retention_days"

const DefaultRetentionDays = 90

var lifecycleKinds = map[string]bool{
	"created":    true,
	"deleted":    true,
	"archived":   true,
	"unarchived": true,
}

func IsLifecycle(kind string) bool { return lifecycleKinds[kind] }

func ScopedVisible(actorID, caller ids.ActorID, full bool, kind string) bool {
	if full {
		return true
	}
	if actorID != 0 && actorID == caller {
		return true
	}
	return IsLifecycle(kind)
}

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

type Service struct {
	store store.Store
}

func New(st store.Store) *Service { return &Service{store: st} }

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

func ActorID(ctx context.Context) ids.ActorID {
	if id, ok := auth.ActorID(ctx); ok {
		return id
	}
	return 0
}

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

func (s *Service) SetRetentionDays(ctx context.Context, days int) error {
	if days <= 0 {
		days = DefaultRetentionDays
	}
	return s.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   SettingKeyRetentionDays,
		Value: strconv.Itoa(days),
	})
}

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
