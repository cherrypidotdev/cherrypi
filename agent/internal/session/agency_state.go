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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"github.com/sky-valley/pi/agent"
	"google.golang.org/protobuf/encoding/protojson"
)

type TaskSessionState struct {
	Revision      int64
	ModelProvider string
	ModelID       string
	ThinkingLevel string
	EntriesJSON   string
	LastEventSeq  int64
}

type TaskEventRecord struct {
	Seq         int64
	At          int64
	Kind        string
	PayloadJSON string
}

var ErrTaskSessionStateConflict = errors.New("task session state revision conflict")

func (s *Session) stateEnabled() bool {
	return s.cfg.GetTaskSessionState != nil && s.cfg.PutTaskSessionState != nil
}

func (s *Session) eventsEnabled() bool {
	return s.cfg.AppendTaskEvents != nil
}

func (s *Session) agencyRef() (projectID string, seq int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agencyProjectID, s.agencySeq
}

func (s *Session) restoreState(ctx context.Context) {
	if !s.stateEnabled() {
		return
	}
	projectID, seq := s.agencyRef()
	if projectID == "" || seq <= 0 {
		return
	}
	taskID := s.ensureAgencyTaskID(ctx)
	if taskID == "" {
		return
	}

	state, found, err := s.cfg.GetTaskSessionState(ctx, taskID)
	if err != nil {
		slog.Warn("restore task session state", "session_id", s.ID, "error", err)
		return
	}

	s.mu.Lock()
	if state.LastEventSeq+1 > s.eventSeq {
		s.eventSeq = state.LastEventSeq + 1
	}
	s.mu.Unlock()
	if found {
		s.adoptStateSelection(state)
	}
	if found && state.EntriesJSON != "" {
		s.applyRestoredState(state)
		return
	}
	s.migrateLegacyHistory(ctx, taskID, projectID, seq, state.LastEventSeq)
}

func (s *Session) adoptStateSelection(state TaskSessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.modelProvider == "" && s.model == "" {
		s.modelProvider = state.ModelProvider
		s.model = state.ModelID
	}
	if s.thinkingLevel == "" {
		s.thinkingLevel = state.ThinkingLevel
	}
}

func (s *Session) adoptStoredSelection(ctx context.Context) {
	if s.probe || !s.stateEnabled() {
		return
	}
	s.mu.Lock()
	empty := s.modelProvider == "" && s.model == ""
	s.mu.Unlock()
	if !empty {
		return
	}
	taskID := s.ensureAgencyTaskID(ctx)
	if taskID == "" {
		return
	}
	state, found, err := s.cfg.GetTaskSessionState(ctx, taskID)
	if err != nil {
		slog.Warn("read task session state", "session_id", s.ID, "error", err)
		return
	}
	if found {
		s.adoptStateSelection(state)
	}
}

func (s *Session) applyRestoredState(state TaskSessionState) {
	projection, err := decodeProjection(state.EntriesJSON)
	if err != nil {
		slog.Warn("decode task session state", "session_id", s.ID, "error", err)
		return
	}
	s.adoptStateSelection(state)
	s.mu.Lock()
	s.restoredProjection = projection
	s.hasRestoredProjection = true
	s.historyRaw = historyLinesFromMessages(projection.Messages)
	s.pushed = len(projection.Messages)
	s.stateRevision = state.Revision
	if state.LastEventSeq+1 > s.eventSeq {
		s.eventSeq = state.LastEventSeq + 1
	}
	s.mu.Unlock()
	if len(projection.Messages) > 0 {
		slog.Info("restored task session state",
			"session_id", s.ID,
			"messages", len(projection.Messages),
			"compacted", hasCompaction(projection),
			"revision", state.Revision,
		)
	}
}

func (s *Session) migrateLegacyHistory(
	ctx context.Context,
	taskID, projectID string,
	seq, lastEventSeq int64,
) {
	if s.cfg.FetchTaskHistory == nil {
		return
	}
	entries, err := s.cfg.FetchTaskHistory(ctx, projectID, seq)
	if err != nil {
		slog.Debug("legacy history fetch", "session_id", s.ID, "error", err)
		return
	}
	if len(entries) == 0 {
		return
	}
	projection, err := projectionFromPiEntries(entries)
	if err != nil {
		slog.Warn("migrate legacy task history", "session_id", s.ID, "error", err)
		return
	}
	entriesJSON, err := encodeProjection(projection)
	if err != nil {
		slog.Warn("encode migrated task history", "session_id", s.ID, "error", err)
		return
	}

	s.mu.Lock()
	s.restoredProjection = projection
	s.hasRestoredProjection = true
	s.historyRaw = historyLinesFromMessages(projection.Messages)
	s.pushed = len(projection.Messages)
	s.stateRevision = 0
	s.mu.Unlock()

	if lastEventSeq == 0 {
		if err := s.backfillLegacyEvents(entries); err != nil {
			slog.Warn("backfill legacy task events", "session_id", s.ID, "error", err)
			return
		}
	}

	state := TaskSessionState{
		ModelProvider: string(projection.Provider),
		ModelID:       projection.ModelID,
		ThinkingLevel: projection.ThinkingLevel,
		EntriesJSON:   entriesJSON,
	}
	revision, err := s.cfg.PutTaskSessionState(ctx, taskID, state, 0)
	if errors.Is(err, ErrTaskSessionStateConflict) {

		if current, found, getErr := s.cfg.GetTaskSessionState(ctx, taskID); getErr == nil &&
			found {
			s.applyRestoredState(current)
			return
		}
	}
	if err != nil {
		slog.Warn("put migrated task session state", "session_id", s.ID, "error", err)
		return
	}
	state.Revision = revision
	s.applyRestoredState(state)
	slog.Info("migrated legacy task history", "session_id", s.ID, "entries", len(entries))
}

func (s *Session) backfillLegacyEvents(entries []SessionEntry) error {
	if !s.eventsEnabled() {
		return nil
	}
	for _, entry := range entries {
		if entry.Raw == "" {
			continue
		}
		for _, event := range parseHistoryLine([]byte(entry.Raw)) {
			s.enqueueEvent(TaskEventProto(s.ID, event), eventKind(event))
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return s.flushEvents(ctx)
}

func (s *Session) checkpointState(ctx context.Context) {
	if !s.stateEnabled() {
		return
	}

	s.checkpointMu.Lock()
	defer s.checkpointMu.Unlock()
	s.mu.Lock()
	stoodDown := s.stateStoodDown
	eng := s.eng
	revision := s.stateRevision
	s.mu.Unlock()
	if stoodDown {

		return
	}
	if eng == nil {
		return
	}
	s.recordContextSize(ctx, eng)
	taskID := s.ensureAgencyTaskID(ctx)
	if taskID == "" {
		return
	}

	projection, err := eng.exportProjection()
	if err != nil {
		slog.Warn("export task session projection", "session_id", s.ID, "error", err)
		return
	}
	entriesJSON, err := encodeProjection(projection)
	if err != nil {
		slog.Warn("encode task session projection", "session_id", s.ID, "error", err)
		return
	}
	s.mu.Lock()
	provider, modelID, thinking := s.modelProvider, s.model, s.thinkingLevel
	s.mu.Unlock()
	if provider == "" {
		provider = string(projection.Provider)
	}
	if modelID == "" {
		modelID = projection.ModelID
	}

	if thinking == "" {
		thinking = projection.ThinkingLevel
	}
	state := TaskSessionState{
		ModelProvider: provider,
		ModelID:       modelID,
		ThinkingLevel: thinking,
		EntriesJSON:   entriesJSON,
	}

	newRevision, err := s.cfg.PutTaskSessionState(ctx, taskID, state, revision)
	if errors.Is(err, ErrTaskSessionStateConflict) {

		current, found, getErr := s.cfg.GetTaskSessionState(ctx, taskID)
		if getErr != nil || !found {
			slog.Warn("reload task session state", "session_id", s.ID, "error", getErr)
			return
		}

		slog.Warn(
			"task session state conflict",
			"session_id", s.ID,
			"expected_revision", revision,
			"current_revision", current.Revision,
		)
		s.mu.Lock()
		s.stateRevision = current.Revision
		s.stateStoodDown = true
		s.mu.Unlock()
		return
	}
	if err != nil {
		slog.Warn("put task session state", "session_id", s.ID, "error", err)
		return
	}

	s.mu.Lock()
	s.stateRevision = newRevision
	s.historyRaw = historyLinesFromMessages(eng.sess.History())
	s.mu.Unlock()
}

func (s *Session) recordContextSize(ctx context.Context, eng *engine) {
	window := 0
	if eng.sess.Model != nil {
		window = eng.sess.Model.ContextWindow
	}
	if window <= 0 {
		return
	}
	tokens := estimateContextTokens(eng)
	now := time.Now().UnixMilli()
	if err := s.store.SetSessionContext(ctx, sqlitedb.SetSessionContextParams{
		ContextTokens:    int64(tokens),
		ContextWindow:    int64(window),
		ContextUpdatedAt: now,
		UpdatedAt:        now,
		ID:               s.ID,
	}); err != nil {
		slog.Debug("record context size", "session_id", s.ID, "error", err)
	}
}

func (s *Session) enqueueEvent(event *agentv1.TaskEvent, kind string) {
	if event == nil || kind == "" || !s.eventsEnabled() {
		return
	}
	payload, err := protojson.Marshal(event)
	if err != nil {
		slog.Warn("marshal task event", "session_id", s.ID, "error", err)
		return
	}
	at := time.Now().UnixMilli()
	if event.At != nil {
		at = event.At.AsTime().UnixMilli()
	}

	s.eventsMu.Lock()
	if s.eventSeq <= 0 {
		s.eventSeq = 1
	}
	s.pendingEvents = append(s.pendingEvents, TaskEventRecord{
		Seq:         s.eventSeq,
		At:          at,
		Kind:        kind,
		PayloadJSON: string(payload),
	})
	s.eventSeq++
	s.eventsMu.Unlock()
}

func (s *Session) queueTranscriptEvents() {
	if !s.eventsEnabled() {
		return
	}

	s.transcriptMu.Lock()
	defer s.transcriptMu.Unlock()
	s.mu.Lock()
	eng := s.eng
	pushed := s.pushed
	s.mu.Unlock()
	if eng == nil {
		return
	}

	messages := eng.sess.History()
	if pushed > len(messages) {
		pushed = 0
	}
	if pushed == len(messages) {
		return
	}
	for index := pushed; index < len(messages); index++ {
		for _, event := range messageEvents(messages[index]) {
			s.enqueueEvent(TaskEventProto(s.ID, event), eventKind(event))
		}
	}
	s.mu.Lock()
	s.pushed = len(messages)
	s.mu.Unlock()
}

func (s *Session) flushEvents(ctx context.Context) error {
	if !s.eventsEnabled() {
		return nil
	}
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	s.eventsMu.Lock()
	if len(s.pendingEvents) == 0 {
		s.eventsMu.Unlock()
		return nil
	}
	if time.Now().Before(s.eventRetryAt) {
		s.eventsMu.Unlock()
		return nil
	}
	batch := append([]TaskEventRecord(nil), s.pendingEvents...)
	s.eventsMu.Unlock()

	taskID := s.ensureAgencyTaskID(ctx)
	if taskID == "" {
		return nil
	}
	if err := s.cfg.AppendTaskEvents(ctx, taskID, batch); err != nil {
		s.eventsMu.Lock()
		if s.eventBackoff == 0 {
			s.eventBackoff = historyRetryMin
		} else {
			s.eventBackoff *= 2
		}
		if s.eventBackoff > historyRetryMax {
			s.eventBackoff = historyRetryMax
		}
		s.eventRetryAt = time.Now().Add(s.eventBackoff)
		s.eventsMu.Unlock()
		slog.Warn("append task events", "session_id", s.ID, "error", err)
		return err
	}

	s.eventsMu.Lock()

	acked := batch[len(batch)-1].Seq
	kept := s.pendingEvents[:0]
	for _, event := range s.pendingEvents {
		if event.Seq > acked {
			kept = append(kept, event)
		}
	}
	s.pendingEvents = kept
	s.eventBackoff = 0
	s.eventRetryAt = time.Time{}
	s.eventsMu.Unlock()
	return nil
}

// flight. A checkpoint failure is a log line and never fails the task.
func (s *Session) syncLoop(ctx context.Context) {
	if !s.eventsEnabled() && !s.stateEnabled() {
		return
	}
	eventTicker := time.NewTicker(historySyncInterval)
	defer eventTicker.Stop()
	checkpointTicker := time.NewTicker(stateSyncInterval)
	defer checkpointTicker.Stop()

	checkpoint := func() {
		s.queueTranscriptEvents()
		if err := s.flushEvents(ctx); err != nil {
			return
		}
		s.checkpointState(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			s.queueTranscriptEvents()
			_ = s.flushEvents(flushCtx)
			s.checkpointState(flushCtx)
			cancel()
			return
		case <-eventTicker.C:
			s.queueTranscriptEvents()
			_ = s.flushEvents(ctx)
		case <-checkpointTicker.C:
			s.mu.Lock()
			running := s.running
			s.mu.Unlock()
			if running {
				checkpoint()
			}
		}
	}
}

type historyLine struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	ParentID  *string         `json:"parentId"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
}

func historyLinesFromMessages(messages []agent.AgentMessage) []string {
	lines := make([]string, 0, len(messages))
	for index, message := range messages {
		raw, err := json.Marshal(message)
		if err != nil {
			continue
		}
		line := historyLine{
			Type:      "message",
			ID:        sessionEntryID(index),
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Message:   raw,
		}
		if index > 0 {
			parent := sessionEntryID(index - 1)
			line.ParentID = &parent
		}
		encoded, err := json.Marshal(line)
		if err != nil {
			continue
		}
		lines = append(lines, string(encoded))
	}
	return lines
}

func sessionEntryID(index int) string {
	return fmt.Sprintf("m%d", index)
}
