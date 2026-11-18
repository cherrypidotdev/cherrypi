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

package tasks

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
)

type SessionState struct {
	Revision      int64
	ModelProvider string
	ModelID       string
	ThinkingLevel string

	EntriesJSON string
	UpdatedAt   int64
}

type TaskEventRow struct {
	Seq         int64
	At          int64
	Kind        string
	PayloadJSON string
}

var ErrSessionStateConflict = store.ErrTaskSessionStateConflict

func (s *Service) SessionState(
	ctx context.Context,
	taskID string,
) (SessionState, bool, error) {
	row, err := s.store.GetTaskSessionState(ctx, taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return SessionState{}, false, nil
	}
	if err != nil {
		return SessionState{}, false, err
	}
	return SessionState{
		Revision:      row.Revision,
		ModelProvider: row.ModelProvider.String,
		ModelID:       row.ModelID.String,
		ThinkingLevel: row.ThinkingLevel.String,
		EntriesJSON:   row.EntriesJson,
		UpdatedAt:     row.UpdatedAt,
	}, true, nil
}

func (s *Service) PutSessionState(
	ctx context.Context,
	taskID string,
	state SessionState,
	expectedRevision int64,
) (int64, error) {
	return s.store.PutTaskSessionState(ctx, store.PutTaskSessionStateParams{
		TaskID:           taskID,
		ExpectedRevision: expectedRevision,
		ModelProvider: sql.NullString{
			String: state.ModelProvider,
			Valid:  state.ModelProvider != "",
		},
		ModelID: sql.NullString{String: state.ModelID, Valid: state.ModelID != ""},
		ThinkingLevel: sql.NullString{
			String: state.ThinkingLevel,
			Valid:  state.ThinkingLevel != "",
		},
		EntriesJSON: state.EntriesJSON,
		UpdatedAt:   time.Now().UnixMilli(),
	})
}

func (s *Service) AppendTaskEvents(
	ctx context.Context,
	taskID string,
	events []TaskEventRow,
) (int64, error) {
	params := make([]store.AppendTaskEventParams, 0, len(events))
	for _, event := range events {
		if event.Seq <= 0 || event.Kind == "" || event.PayloadJSON == "" {
			continue
		}
		params = append(params, store.AppendTaskEventParams{
			TaskID:      taskID,
			Seq:         event.Seq,
			At:          event.At,
			Kind:        event.Kind,
			PayloadJson: event.PayloadJSON,
		})
	}
	return s.store.AppendTaskEvents(ctx, params)
}

func (s *Service) TaskEventPage(
	ctx context.Context,
	taskID string,
	before int64,
	limit int,
) ([]TaskEventRow, error) {
	rows, err := s.store.PageTaskEvents(ctx, taskID, before, limit)
	if err != nil {
		return nil, err
	}
	out := make([]TaskEventRow, len(rows))
	for i, row := range rows {
		out[i] = TaskEventRow{
			Seq:         row.Seq,
			At:          row.At,
			Kind:        row.Kind,
			PayloadJSON: row.PayloadJson,
		}
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (s *Service) TaskEvents(ctx context.Context, taskID string) ([]TaskEventRow, error) {
	rows, err := s.store.ListTaskEvents(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]TaskEventRow, len(rows))
	for i, row := range rows {
		out[i] = TaskEventRow{
			Seq:         row.Seq,
			At:          row.At,
			Kind:        row.Kind,
			PayloadJSON: row.PayloadJson,
		}
	}
	return out, nil
}

func (s *Service) LastTaskEventSeq(ctx context.Context, taskID string) (int64, error) {
	return s.store.LastTaskEventSeq(ctx, taskID)
}

type SessionHistoryEntry struct {
	Seq      int64
	EntryID  string
	ParentID string
	Type     string
	Raw      string
}

func (s *Service) AppendSessionHistory(
	ctx context.Context,
	taskID string,
	entries []SessionHistoryEntry,
) (int64, error) {
	now := time.Now().UnixMilli()
	params := make([]store.AppendTaskSessionEntryParams, 0, len(entries))
	for _, entry := range entries {
		if entry.EntryID == "" || entry.Raw == "" {
			continue
		}
		params = append(params, store.AppendTaskSessionEntryParams{
			TaskID:    taskID,
			EntryID:   entry.EntryID,
			ParentID:  entry.ParentID,
			Type:      entry.Type,
			Raw:       entry.Raw,
			CreatedAt: now,
		})
	}
	return s.store.AppendTaskSessionEntries(ctx, params)
}

func (s *Service) SessionHistory(
	ctx context.Context,
	taskID string,
) ([]SessionHistoryEntry, error) {
	rows, err := s.store.ListTaskSessionEntries(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]SessionHistoryEntry, len(rows))
	for i, row := range rows {
		out[i] = SessionHistoryEntry{
			Seq:      row.Seq,
			EntryID:  row.EntryID,
			ParentID: row.ParentID,
			Type:     row.Type,
			Raw:      row.Raw,
		}
	}
	return out, nil
}

func (s *Service) SessionHistoryTip(ctx context.Context, taskID string) (string, int64, error) {
	row, err := s.store.TaskSessionEntryTip(ctx, taskID)
	if err != nil {
		return "", 0, err
	}
	count, err := s.store.CountTaskSessionEntries(ctx, taskID)
	if err != nil {
		return "", 0, err
	}
	return row.EntryID, count, nil
}
