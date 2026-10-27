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
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrTaskHistoryNotFound = errors.New("task history: task not found")

var ErrTaskHistoryUnauthorized = errors.New("task history: unauthorized")

const (
	historyRetryBase        = time.Second
	historyRetryMax         = 2 * time.Minute
	historyAuthRefreshLimit = 3
	historyRetryJitter      = 4
)

func historyBackoff(failures int) time.Duration {
	if failures < 1 {
		return 0
	}
	delay := historyRetryBase
	for i := 1; i < failures && delay < historyRetryMax; i++ {
		delay *= 2
	}
	if delay > historyRetryMax {
		delay = historyRetryMax
	}
	jitter := time.Duration(rand.Int63n(int64(delay/historyRetryJitter) + 1))
	return delay + jitter
}

type SessionEntry struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Type     string `json:"type"`
	Raw      string `json:"raw"`
}

func (s *Session) historyEnabled() bool {
	return s.cfg.FetchTaskHistory != nil && s.cfg.PushTaskHistory != nil
}

func (s *Session) resetHistorySync() {
	s.histMu.Lock()
	s.syncedTip = ""
	s.historyReconciled = false
	s.historyDisabled = false
	s.noteHistorySuccessLocked()
	s.histMu.Unlock()
}

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

	s.syncHistory(ctx, true)
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
			s.syncHistory(ctx, false)
		}
	}
}

func (s *Session) syncHistory(ctx context.Context, force bool) {
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
	if !force && time.Now().Before(s.historyRetryAt) {
		return
	}
	local := s.readSessionEntries()

	if !s.historyReconciled {
		remote, err := s.cfg.FetchTaskHistory(ctx, projectID, seq)
		if err != nil {
			s.noteHistoryError(ctx, err)
			return
		}
		s.noteHistorySuccessLocked()
		switch {
		case len(remote) == 0:

		case len(local) == 0:

			if err := s.writeSessionEntries(remote); err != nil {
				log.Printf("[session %s] reconstruct session: %v", s.ID, err)
				return
			}
			local = s.readSessionEntries()
		default:
			if findEntry(local, remote[len(remote)-1].ID) < 0 {

				if err := s.writeSessionEntries(remote); err != nil {
					log.Printf("[session %s] reconstruct session: %v", s.ID, err)
					return
				}
				local = s.readSessionEntries()
			}

		}

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
		s.noteHistoryError(ctx, err)
		return
	}
	s.syncedTip = tail[len(tail)-1].ID
	s.noteHistorySuccessLocked()
}

func (s *Session) noteHistoryError(ctx context.Context, err error) {
	if errors.Is(err, ErrTaskHistoryNotFound) {
		s.historyDisabled = true
		return
	}
	if errors.Is(err, ErrTaskHistoryUnauthorized) &&
		s.cfg.RefreshTaskHistoryAuth != nil &&
		s.historyAuthAttempts < historyAuthRefreshLimit {
		s.historyAuthAttempts++
		if refreshErr := s.cfg.RefreshTaskHistoryAuth(ctx); refreshErr != nil {
			log.Printf("[session %s] refresh task history auth: %v", s.ID, refreshErr)
		} else {
			log.Printf("[session %s] refreshed task history auth", s.ID)
		}
	}
	s.historyFailures++
	delay := historyBackoff(s.historyFailures)
	s.historyRetryAt = time.Now().Add(delay)
	log.Printf(
		"[session %s] sync task history (attempt %d, retry in %s): %v",
		s.ID,
		s.historyFailures,
		delay.Round(time.Millisecond),
		err,
	)
}

func (s *Session) noteHistorySuccessLocked() {
	s.historyFailures = 0
	s.historyRetryAt = time.Time{}
	s.historyAuthAttempts = 0
}

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
