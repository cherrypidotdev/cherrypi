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
	"cherrypi.vladfaust.com/agency/internal/store"
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

const (
	DefaultLimit = 100
	MaxLimit     = 500
	ExportLimit  = 10000
)

type Filter struct {
	Actor string
	Scope Scope
	Kind  string
	Since time.Time
	Until time.Time

	Search string
	Limit  int64

	Cursor string
}

type Page struct {
	Entries    []Entry
	NextCursor string
}

type cursor struct {
	at    int64
	scope string
	id    int64
}

func (s *Service) List(ctx context.Context, filter Filter) (Page, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	params := store.ListDomainLogsParams{
		Actor:    nullableString(filter.Actor),
		Scope:    nullableScope(filter.Scope),
		Kind:     nullableString(filter.Kind),
		Since:    nullableTime(filter.Since),
		Until:    nullableTime(filter.Until),
		Search:   nullableString(filter.Search),
		MaxCount: limit,
	}
	if c, ok := parseCursor(filter.Cursor); ok {
		params.CursorAt = sql.NullInt64{Int64: c.at, Valid: true}
		params.CursorScope = sql.NullString{String: c.scope, Valid: true}
		params.CursorID = sql.NullInt64{Int64: c.id, Valid: true}
	}

	rows, err := s.store.ListDomainLogs(ctx, params)
	if err != nil {
		return Page{}, err
	}
	entries := make([]Entry, len(rows))
	for i, row := range rows {
		entries[i] = Entry{
			ID:            row.EntryID,
			Scope:         Scope(row.Scope),
			ScopeID:       row.ScopeID,
			ScopeLabel:    row.ScopeLabel,
			At:            time.UnixMilli(row.At),
			ActorID:       ids.ActorID(row.ActorID.Int64),
			ActorKind:     row.ActorKind.String,
			ActorUsername: row.ActorUsername.String,
			ActorName:     row.ActorName.String,
			Kind:          row.Kind,
			Field:         row.Field,
			OldValue:      row.OldValue,
			NewValue:      row.NewValue,
			Message:       row.Message,
			DataJSON:      row.DataJson,
		}
	}

	page := Page{Entries: entries}
	if int64(len(entries)) == limit && len(entries) > 0 {
		last := entries[len(entries)-1]
		page.NextCursor = formatCursor(cursor{
			at:    last.At.UnixMilli(),
			scope: string(last.Scope),
			id:    last.ID,
		})
	}
	return page, nil
}

func (s *Service) ListUser(
	ctx context.Context,
	userActorID ids.ActorID,
	beforeID int64,
	limit int64,
) ([]Entry, error) {
	limit = clampLimit(limit)
	if beforeID <= 0 {
		beforeID = maxInt64
	}
	rows, err := s.store.ListUserLogs(ctx, store.ListUserLogsParams{
		UserActorID: int64(userActorID),
		BeforeID:    beforeID,
		MaxCount:    limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Entry, len(rows))
	for i, row := range rows {
		out[i] = Entry{
			ID:            row.ID,
			Scope:         ScopeUser,
			ScopeID:       strconv.FormatInt(row.UserActorID, 10),
			At:            time.UnixMilli(row.At),
			ActorID:       ids.ActorID(row.ActorID.Int64),
			ActorKind:     row.ActorKind.String,
			ActorUsername: row.ActorUsername.String,
			ActorName:     row.ActorName.String,
			Kind:          row.Kind,
			Field:         row.Field,
			OldValue:      row.OldValue,
			NewValue:      row.NewValue,
			Message:       row.Message,
			DataJSON:      row.DataJson,
		}
	}
	return out, nil
}

func (s *Service) ListAgent(
	ctx context.Context,
	agentActorID ids.ActorID,
	beforeID int64,
	limit int64,
) ([]Entry, error) {
	limit = clampLimit(limit)
	if beforeID <= 0 {
		beforeID = maxInt64
	}
	rows, err := s.store.ListAgentDomainLogs(ctx,
		store.ListAgentDomainLogsParams{
			AgentActorID: int64(agentActorID),
			BeforeID:     beforeID,
			MaxCount:     limit,
		})
	if err != nil {
		return nil, err
	}
	out := make([]Entry, len(rows))
	for i, row := range rows {
		out[i] = Entry{
			ID:            row.ID,
			Scope:         ScopeAgent,
			ScopeID:       strconv.FormatInt(row.AgentActorID, 10),
			At:            time.UnixMilli(row.At),
			ActorID:       ids.ActorID(row.ActorID.Int64),
			ActorKind:     row.ActorKind.String,
			ActorUsername: row.ActorUsername.String,
			ActorName:     row.ActorName.String,
			Kind:          row.Kind,
			Field:         row.Field,
			OldValue:      row.OldValue,
			NewValue:      row.NewValue,
			Message:       row.Message,
			DataJSON:      row.DataJson,
		}
	}
	return out, nil
}

const maxInt64 = int64(^uint64(0) >> 1)

func clampLimit(limit int64) int64 {
	switch {
	case limit <= 0:
		return DefaultLimit
	case limit > MaxLimit:
		return MaxLimit
	default:
		return limit
	}
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func nullableScope(scope Scope) sql.NullString {
	if scope == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(scope), Valid: true}
}

func nullableTime(value time.Time) sql.NullInt64 {
	if value.IsZero() {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value.UnixMilli(), Valid: true}
}

func formatCursor(c cursor) string {
	return strconv.FormatInt(c.at, 10) + ":" + c.scope + ":" +
		strconv.FormatInt(c.id, 10)
}

func parseCursor(value string) (cursor, bool) {
	if value == "" {
		return cursor{}, false
	}
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return cursor{}, false
	}
	at, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return cursor{}, false
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return cursor{}, false
	}
	return cursor{at: at, scope: parts[1], id: id}, true
}
