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

package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrTaskHistoryNotFound reports that the agency does not know the task a
// session is syncing history for (for example a session created outside the
// agency task lifecycle, or one whose task was just deleted). History sync is
// disabled for such a session so it does not retry on every flush.
var ErrTaskHistoryNotFound = errors.New("task history: task not found")

// SessionEntry is one persisted Pi session entry: everything but the session
// header. Raw is the verbatim JSONL line, stored by the agency and handed back
// to reconstruct a resumable session file.
type SessionEntry struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Type     string `json:"type"`
	Raw      string `json:"raw"`
}

// historyEnabled reports whether the agent has an agency history surface.
func (s *Session) historyEnabled() bool {
	return s.cfg.FetchTaskHistory != nil && s.cfg.PushTaskHistory != nil
}

// resetHistorySync arms a fresh reconcile for a newly spawned Pi process (or
// an explicit migration request). It re-enables sync in case the agency did
// not know the task on an earlier attempt.
func (s *Session) resetHistorySync() {
	s.histMu.Lock()
	s.syncedTip = ""
	s.historyReconciled = false
	s.historyDisabled = false
	s.histMu.Unlock()
}

// startHistoryWatch starts the per-session flush loop. It is a no-op when
// history sync is unconfigured or a watcher is already running.
func (s *Session) startHistoryWatch() {
	if !s.historyEnabled() {
		return
	}
	s.mu.Lock()
	if s.historyCancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.historyCancel = cancel
	s.mu.Unlock()

	go s.watchHistory(ctx)
}

// stopHistoryWatch stops the flush loop and does one final synchronous flush,
// so entries written between ticks are not lost when Pi exits.
func (s *Session) stopHistoryWatch() {
	s.mu.Lock()
	cancel := s.historyCancel
	s.historyCancel = nil
	s.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()

	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	s.syncHistory(ctx)
}

func (s *Session) watchHistory(ctx context.Context) {
	interval := s.cfg.HistorySyncInterval
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncHistory(ctx)
		}
	}
}

// syncHistory keeps the agency's canonical copy of this session in step with
// the local Pi session file. On the first call of a run it reconciles: if the
// local file is missing or diverged, it is rebuilt from the agency's history
// (a fresh agent continuing the task, or a lost workspace); if the agency's tip
// is an ancestor of the local tip, the local tail is pushed (a fast-forward
// merge of entries the agency missed). Later calls for a run only push the
// entries written since the last flush.
func (s *Session) syncHistory(ctx context.Context) {
	if !s.historyEnabled() {
		return
	}
	projectID, seq := s.agencyRef()
	if projectID == "" || seq == 0 {
		return
	}

	s.histMu.Lock()
	defer s.histMu.Unlock()

	if s.historyDisabled {
		return
	}
	local := s.readSessionEntries()

	if !s.historyReconciled {
		remote, err := s.cfg.FetchTaskHistory(ctx, projectID, seq)
		if err != nil {
			s.noteHistoryError(err)
			return
		}
		switch {
		case len(remote) == 0:
			// Nothing stored yet: this is the start of the conversation.
		case len(local) == 0:
			// A fresh agent (or a lost workspace) continuing the task:
			// materialize the agency's conversation under this Pi session id.
			if err := s.writeSessionEntries(remote); err != nil {
				log.Printf("[session %s] reconstruct session: %v", s.ID, err)
				return
			}
			local = s.readSessionEntries()
		default:
			if findEntry(local, remote[len(remote)-1].ID) < 0 {
				// Diverged (the agency does not have a prefix of the local
				// tree): the agency is canonical, rebuild from it.
				if err := s.writeSessionEntries(remote); err != nil {
					log.Printf("[session %s] reconstruct session: %v", s.ID, err)
					return
				}
				local = s.readSessionEntries()
			}
			// Otherwise the agency is behind and the tail push below
			// fast-forwards it.
		}
		// Everything the agency already has is now present locally.
		if len(remote) > 0 {
			if i := findEntry(local, remote[len(remote)-1].ID); i >= 0 {
				s.syncedTip = local[i].ID
			}
		}
		s.historyReconciled = true
	}

	start := 0
	if s.syncedTip != "" {
		i := findEntry(local, s.syncedTip)
		if i < 0 {
			// The anchor vanished (for example the file was replaced); force a
			// full reconcile on the next tick.
			s.historyReconciled = false
			return
		}
		start = i + 1
	}
	tail := local[start:]
	if len(tail) == 0 {
		return
	}
	if err := s.cfg.PushTaskHistory(ctx, projectID, seq, tail); err != nil {
		s.noteHistoryError(err)
		return
	}
	s.syncedTip = tail[len(tail)-1].ID
}

// noteHistoryError disables sync when the agency reports the task is unknown,
// so a session created outside the task lifecycle does not retry every flush.
// Other failures are logged, matching the session package's style.
func (s *Session) noteHistoryError(err error) {
	if errors.Is(err, ErrTaskHistoryNotFound) {
		s.historyDisabled = true
		return
	}
	log.Printf("[session %s] sync task history: %v", s.ID, err)
}

// readSessionEntries parses the local Pi session file into ordered entries,
// skipping the header and any malformed line. It returns nil when no file
// exists yet (Pi only writes one once the session has a message).
func (s *Session) readSessionEntries() []SessionEntry {
	path := s.sessionFilePath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var out []SessionEntry
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var base struct {
			Type     string  `json:"type"`
			ID       string  `json:"id"`
			ParentID *string `json:"parentId"`
		}
		if json.Unmarshal(line, &base) != nil {
			continue
		}
		if base.Type == "" || base.Type == "session" || base.ID == "" {
			continue
		}
		parentID := ""
		if base.ParentID != nil {
			parentID = *base.ParentID
		}
		out = append(out, SessionEntry{
			ID:       base.ID,
			ParentID: parentID,
			Type:     base.Type,
			Raw:      string(line),
		})
	}
	return out
}

// writeSessionEntries replaces this session's Pi session file with a new
// header plus the given entries. Pi discovers it by the header id, so the
// filename only needs the conventional timestamp_id shape. Existing files for
// the same id are removed so Pi cannot pick a stale one.
func (s *Session) writeSessionEntries(entries []SessionEntry) error {
	dir := s.sessionDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*_"+s.PiSessionID+".jsonl"))
	for _, path := range matches {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	timestamp := time.Now().UTC()
	header := map[string]any{
		"type":      "session",
		"version":   3,
		"id":        s.PiSessionID,
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"cwd":       s.Workspace,
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.Write(headerJSON)
	buf.WriteByte('\n')
	for _, entry := range entries {
		if entry.Raw == "" {
			continue
		}
		buf.WriteString(entry.Raw)
		buf.WriteByte('\n')
	}

	name := strings.NewReplacer(":", "-", ".", "-").Replace(
		timestamp.Format("2006-01-02T15:04:05.000Z"),
	) + "_" + s.PiSessionID + ".jsonl"
	return os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644)
}

// findEntry returns the index of the entry with the given id, or -1.
func findEntry(entries []SessionEntry, id string) int {
	if id == "" {
		return -1
	}
	for i, entry := range entries {
		if entry.ID == id {
			return i
		}
	}
	return -1
}
