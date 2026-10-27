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
	"cherrypi.vladfaust.com/agency/internal/store"
	"context"
	"time"
)

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
