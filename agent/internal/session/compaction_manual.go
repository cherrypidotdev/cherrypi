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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sky-valley/pi/ai"
	"github.com/sky-valley/pi/coding"
)

var (
	ErrNothingToCompact = errors.New("nothing to compact")

	ErrCompactionDisabled = errors.New("compaction is disabled")
)

func (s *Session) Compact(ctx context.Context) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if s.abandonedForShutdown() {
		return ErrShuttingDown
	}
	if waiting, _ := s.waitingForInput(); waiting {
		return ErrSessionBusy
	}
	s.mu.Lock()
	running := s.running
	eng := s.eng
	s.mu.Unlock()
	if running && eng != nil {
		if eng.compaction == nil || !eng.compaction.Enabled {
			return ErrCompactionDisabled
		}

		eng.queuedCompaction.Store(true)
		return nil
	}
	if s.busy() {
		return ErrSessionBusy
	}

	s.cancelIdle()
	if err := s.ensureEngineLocked(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	eng = s.eng
	s.mu.Unlock()
	if eng == nil {
		return fmt.Errorf("session is not running")
	}
	if err := eng.compactNow(ctx); err != nil {
		s.scheduleIdleReap()
		return err
	}
	s.checkpointState(ctx)
	s.scheduleIdleReap()
	return nil
}

func (e *engine) compactNow(ctx context.Context) error {
	if e.compaction == nil || !e.compaction.Enabled {
		return ErrCompactionDisabled
	}
	if e.sess.Model == nil || e.sess.Model.ContextWindow <= 0 {
		return fmt.Errorf("the model has no context window")
	}
	transform := e.sess.Agent.TransformContext
	if transform == nil {
		return ErrCompactionDisabled
	}

	before, err := e.exportProjection()
	if err != nil {
		return err
	}
	beforeStamp := checkpointStamp(before)

	forced := *e.compaction

	forced.ReserveTokens = max(forced.ReserveTokens, e.sess.Model.ContextWindow)

	probe := &compactionProbe{}
	restoreProbe := e.probeSummaries(probe)
	defer restoreProbe()

	e.manualCompaction.Store(true)
	defer e.manualCompaction.Store(false)
	e.sess.EnableCompaction(forced)
	defer e.sess.EnableCompaction(*e.compaction)

	transform(ctx, e.sess.History())

	after, err := e.exportProjection()
	if err != nil {
		return err
	}
	if !hasCompaction(after) || ctx.Err() != nil {
		return ErrNothingToCompact
	}

	advanced := !hasCompaction(before) || checkpointStamp(after) != beforeStamp
	complete := probe.successes > 0 && probe.successes == probe.requests
	if !advanced && !complete {
		return ErrNothingToCompact
	}
	return nil
}

type compactionProbe struct {
	requests  int
	successes int
	empty     int
}

func (e *engine) probeSummaries(probe *compactionProbe) func() {
	original := e.sess.Agent.StreamFn
	e.sess.Agent.StreamFn = func(
		ctx context.Context,
		model *ai.Model,
		req ai.TranscriptContext,
		opts *ai.SimpleStreamOptions,
	) *ai.AssistantMessageEventStream {
		if original == nil || !isSummarizationRequest(req) {
			return original(ctx, model, req, opts)
		}
		probe.requests++
		message := original(ctx, model, req, opts).Result()
		switch {
		case message == nil || strings.TrimSpace(ai.ContentText(message.Content)) == "":
			probe.empty++
			message = ai.NewErrorMessage(
				model.Api,
				model.Provider,
				model.ID,
				ai.StopError,
				"the summarization response was empty",
			)
		case summarizationAccepted(message):
			probe.successes++
		}
		stream := ai.NewAssistantMessageEventStream()
		stream.End(message)
		return stream
	}
	return func() { e.sess.Agent.StreamFn = original }
}

func isSummarizationRequest(req ai.TranscriptContext) bool {
	system, ok := ai.GetInitialSystemMessage(req.Messages)
	return ok && strings.Contains(
		ai.GetSystemMessageText(system),
		"context summarization assistant",
	)
}

func summarizationAccepted(message *ai.AssistantMessage) bool {
	if message == nil || message.StopReason == ai.StopError || message.StopReason == ai.StopLength {
		return false
	}
	for _, block := range message.Content {
		if _, ok := block.(ai.ToolCall); ok {
			return false
		}
	}
	return true
}

func checkpointStamp(p coding.BranchProjection) string {
	if !hasCompaction(p) {
		return ""
	}
	hash := sha256.New()
	encoder := json.NewEncoder(hash)
	for _, entry := range p.Entries {
		source := entry.SourceEntry
		if source == nil {
			continue
		}
		if source.Type == "compaction" {
			if source.SystemMessage != nil {
				_ = encoder.Encode(source.SystemMessage)
			}
			continue
		}
		for _, message := range entry.Messages {
			_ = encoder.Encode(message)
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}
