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

// SessionHistoryEntry is one persisted Pi session entry: everything but the
// session header. `Raw` is the verbatim JSONL line, kept so a fresh agent can
// write a resumable session file, and `EntryID`/`ParentID` are Pi's tree keys.
type SessionHistoryEntry struct {
	// Seq is the agency's 1-based order; it doubles as the history page cursor.
	Seq      int64
	EntryID  string
	ParentID string
	Type     string
	Raw      string
}

// AppendSessionHistory adds entries a task's agent reported. Pi entry ids are
// unique per task, so a replayed push (after a lost response) is a no-op. It
// returns how many entries were newly stored. Entries are appended in order;
// a partial failure is safe because the agent retries the tail.
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

// SessionHistory returns a task's canonical conversation, oldest first.
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

// HistoryMigrated reports whether the agency holds any of a task's session
// history, i.e. whether deleting the agent-side Pi session would keep the
// conversation.
func (s *Service) HistoryMigrated(ctx context.Context, taskID string) (bool, error) {
	count, err := s.store.CountTaskSessionEntries(ctx, taskID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// MigratedTaskIDs returns the set of task ids with stored history, so a list
// can mark migrated tasks in one query.
func (s *Service) MigratedTaskIDs(ctx context.Context) (map[string]bool, error) {
	ids, err := s.store.ListTaskHistoryTaskIDs(ctx)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// SessionHistoryTip returns the newest stored entry's Pi id (empty when the
// task has no history) and the stored entry count.
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
