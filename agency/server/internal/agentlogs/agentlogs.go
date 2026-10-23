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

// Package agentlogs persists the operational log lines agents push over their
// event streams, with a per-agent retention cap, and serves them back to the
// dashboard.
package agentlogs

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"database/sql"
	"encoding/json"

	"cherrypi.vladfaust.com/agency/internal/store"
)

// DefaultRetention is the number of newest lines kept per agent. Older lines
// are pruned on insert, so an agent that logs forever cannot grow the table.
const DefaultRetention = 2000

// Severity levels, ordered so `>= min` is a severity filter. They match the
// agent_logs.level column.
const (
	LevelDebug = 0
	LevelInfo  = 1
	LevelWarn  = 2
	LevelError = 3
)

// Log is one line to persist.
type Log struct {
	AgentID   ids.AgentID
	At        int64
	Level     int
	Message   string
	TaskID    string
	SessionID string
	Attrs     map[string]any
}

// Row is one persisted line.
type Row struct {
	ID        int64
	AgentID   ids.AgentID
	At        int64
	Level     int
	Message   string
	TaskID    string
	SessionID string
	AttrsJSON string
}

// Service reads and writes agent logs.
type Service struct {
	store store.Store
	// Retention is the per-agent line cap applied on insert. Tests lower it.
	Retention int
}

// New builds the service over the agency store.
func New(st store.Store) *Service {
	return &Service{store: st, Retention: DefaultRetention}
}

// ParseLevel maps a wire level name ("info", "warn", "error", "debug") to its
// severity. Unknown names are info.
func ParseLevel(name string) int {
	switch name {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Insert appends one line and prunes the agent's history to Retention.
func (s *Service) Insert(ctx context.Context, log Log) error {
	attrs := ""
	if len(log.Attrs) > 0 {
		if b, err := json.Marshal(log.Attrs); err == nil {
			attrs = string(b)
		}
	}
	return s.store.InsertAgentLog(ctx, store.InsertAgentLogParams{
		AgentID:   sql.NullInt64{Int64: int64(log.AgentID), Valid: log.AgentID != 0},
		At:        log.At,
		Level:     int64(log.Level),
		Message:   log.Message,
		TaskID:    log.TaskID,
		SessionID: log.SessionID,
		AttrsJson: attrs,
	}, int64(s.Retention))
}

// List returns an agent's newest lines with severity >= minLevel, newest
// first, older than beforeID (0 starts at the newest).
func (s *Service) List(
	ctx context.Context,
	agentID ids.AgentID,
	minLevel int,
	beforeID int64,
	limit int64,
) ([]Row, error) {
	rows, err := s.store.ListAgentLogs(ctx, store.ListAgentLogsParams{
		AgentID:  sql.NullInt64{Int64: int64(agentID), Valid: true},
		MinLevel: int64(minLevel),
		BeforeID: beforeID,
		MaxCount: limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Row, len(rows))
	for i, r := range rows {
		out[i] = Row{
			ID:        r.ID,
			AgentID:   ids.AgentID(r.AgentID.Int64),
			At:        r.At,
			Level:     int(r.Level),
			Message:   r.Message,
			TaskID:    r.TaskID,
			SessionID: r.SessionID,
			AttrsJSON: r.AttrsJson,
		}
	}
	return out, nil
}
