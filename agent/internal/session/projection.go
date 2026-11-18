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
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
	"github.com/sky-valley/pi/coding"
)

const (
	compactionSummaryPrefix = "The conversation history before this point was compacted into the following summary:\n\n<summary>\n"
	compactionSummarySuffix = "\n</summary>"
)

type projectedEntryJSON struct {
	Type             string            `json:"type"`
	ID               string            `json:"id"`
	ParentID         string            `json:"parent_id,omitempty"`
	Timestamp        string            `json:"timestamp,omitempty"`
	Summary          string            `json:"summary,omitempty"`
	FirstKeptEntryID string            `json:"first_kept_entry_id,omitempty"`
	SystemMessage    json.RawMessage   `json:"system_message,omitempty"`
	Provider         string            `json:"provider,omitempty"`
	ModelID          string            `json:"model_id,omitempty"`
	ThinkingLevel    string            `json:"thinking_level,omitempty"`
	Details          json.RawMessage   `json:"details,omitempty"`
	FromHook         bool              `json:"from_hook,omitempty"`
	Messages         []json.RawMessage `json:"messages"`
}

func encodeProjection(p coding.BranchProjection) (string, error) {
	entries := make([]projectedEntryJSON, 0, len(p.Entries))
	for _, entry := range p.Entries {
		source := entry.SourceEntry
		if source == nil {
			continue
		}
		out := projectedEntryJSON{
			Type:             source.Type,
			ID:               source.ID,
			ParentID:         source.ParentID,
			Timestamp:        source.Timestamp,
			Summary:          source.Summary,
			FirstKeptEntryID: source.FirstKeptEntryID,
			Provider:         source.Provider,
			ModelID:          source.ModelID,
			ThinkingLevel:    source.ThinkingLevel,
			Details:          source.Details,
			FromHook:         source.FromHook,
		}
		if source.SystemMessage != nil {
			raw, err := json.Marshal(source.SystemMessage)
			if err != nil {
				return "", fmt.Errorf("marshal compaction system message: %w", err)
			}
			out.SystemMessage = raw
		}
		for _, message := range entry.Messages {
			raw, err := json.Marshal(message)
			if err != nil {
				return "", fmt.Errorf("marshal projection message: %w", err)
			}
			out.Messages = append(out.Messages, raw)
		}
		if out.Messages == nil {
			out.Messages = []json.RawMessage{}
		}
		entries = append(entries, out)
	}
	raw, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("marshal projection entries: %w", err)
	}
	return string(raw), nil
}

func decodeProjection(entriesJSON string) (coding.BranchProjection, error) {
	var entries []projectedEntryJSON
	if err := json.Unmarshal([]byte(entriesJSON), &entries); err != nil {
		return coding.BranchProjection{}, fmt.Errorf("decode projection entries: %w", err)
	}

	projection := coding.BranchProjection{
		BranchContext: coding.BranchContext{ThinkingLevel: "off"},
	}
	for _, entry := range entries {
		source := &coding.SessionEntry{
			Type:             entry.Type,
			ID:               entry.ID,
			ParentID:         entry.ParentID,
			Timestamp:        entry.Timestamp,
			Summary:          entry.Summary,
			FirstKeptEntryID: entry.FirstKeptEntryID,
			Provider:         entry.Provider,
			ModelID:          entry.ModelID,
			ThinkingLevel:    entry.ThinkingLevel,
			Details:          entry.Details,
			FromHook:         entry.FromHook,
		}
		if len(entry.SystemMessage) > 0 {
			var system ai.SystemMessage
			if err := json.Unmarshal(entry.SystemMessage, &system); err != nil {
				return coding.BranchProjection{},
					fmt.Errorf("decode compaction system message: %w", err)
			}
			source.SystemMessage = &system
		}
		messages := make([]agent.AgentMessage, 0, len(entry.Messages))
		for _, raw := range entry.Messages {
			message, err := ai.UnmarshalMessage(raw)
			if err != nil {
				return coding.BranchProjection{}, fmt.Errorf("decode projection message: %w", err)
			}
			messages = append(messages, message)
		}
		projection.Entries = append(
			projection.Entries,
			coding.ProjectedSessionEntry{SourceEntry: source, Messages: messages},
		)
		projection.Messages = append(projection.Messages, messages...)
		if source.ThinkingLevel != "" {
			projection.ThinkingLevel = source.ThinkingLevel
		}
		if source.Provider != "" || source.ModelID != "" {
			projection.Provider, projection.ModelID = source.Provider, source.ModelID
		}
	}
	return projection, nil
}

func projectionTrustedFrom(p coding.BranchProjection) int {
	if !hasCompaction(p) {
		return 0
	}
	compactionID := p.Entries[0].SourceEntry.ID
	offset := len(p.Entries[0].Messages)
	for _, entry := range p.Entries[1:] {
		if entry.SourceEntry.ParentID == compactionID {
			return offset
		}
		offset += len(entry.Messages)
	}
	return len(p.Messages)
}

func hasCompaction(p coding.BranchProjection) bool {
	return len(p.Entries) > 0 && p.Entries[0].SourceEntry != nil &&
		p.Entries[0].SourceEntry.Type == "compaction"
}

func projectionFromView(
	view []agent.AgentMessage,
	raw []agent.AgentMessage,
	rawLen int,
) (coding.BranchProjection, bool) {
	headLen, summary, systemMessage, ok := compactedHead(view)
	if !ok {
		return coding.BranchProjection{}, false
	}
	if rawLen < 0 || rawLen > len(raw) {
		rawLen = len(raw)
	}
	projected := slices.Clone(view)
	projected = append(projected, raw[rawLen:]...)

	compaction := &coding.SessionEntry{
		Type:          "compaction",
		ID:            "compaction",
		Summary:       summary,
		SystemMessage: systemMessage,
	}
	projection := coding.BranchProjection{
		BranchContext: coding.BranchContext{ThinkingLevel: "off"},
	}
	projection.Entries = append(projection.Entries, coding.ProjectedSessionEntry{
		SourceEntry: compaction,
		Messages:    slices.Clone(projected[:headLen]),
	})
	projection.Messages = append(projection.Messages, projected[:headLen]...)
	for index, message := range projected[headLen:] {
		parent := "compaction"
		if index > 0 {
			parent = fmt.Sprintf("m%d", index-1)
		}
		source := &coding.SessionEntry{
			Type:     "message",
			ID:       fmt.Sprintf("m%d", index),
			ParentID: parent,
		}
		projection.Entries = append(
			projection.Entries,
			coding.ProjectedSessionEntry{
				SourceEntry: source,
				Messages:    []agent.AgentMessage{message},
			},
		)
		projection.Messages = append(projection.Messages, message)
	}
	return projection, true
}

func projectionFromMessages(messages []agent.AgentMessage) coding.BranchProjection {
	projection := coding.BranchProjection{
		BranchContext: coding.BranchContext{ThinkingLevel: "off"},
	}
	for index, message := range messages {
		source := &coding.SessionEntry{
			Type: "message",
			ID:   fmt.Sprintf("m%d", index),
		}
		if index > 0 {
			source.ParentID = fmt.Sprintf("m%d", index-1)
		}
		projection.Entries = append(
			projection.Entries,
			coding.ProjectedSessionEntry{
				SourceEntry: source,
				Messages:    []agent.AgentMessage{message},
			},
		)
		projection.Messages = append(projection.Messages, message)
	}
	return projection
}

func compactedHead(
	view []agent.AgentMessage,
) (int, string, *ai.SystemMessage, bool) {
	index := 0
	var systemMessage *ai.SystemMessage
	if index < len(view) && view[index].MessageRole() == ai.RoleSystem {
		switch message := view[index].(type) {
		case ai.SystemMessage:
			systemMessage = &message
		case *ai.SystemMessage:
			systemMessage = message
		}
		index++
	}
	if index >= len(view) {
		return 0, "", nil, false
	}
	summary, ok := compactionSummaryText(view[index])
	if !ok {
		return 0, "", nil, false
	}
	return index + 1, summary, systemMessage, true
}

func compactionSummaryText(message agent.AgentMessage) (string, bool) {
	text := userMessageText(message)
	if len(text) < len(compactionSummaryPrefix) ||
		text[:len(compactionSummaryPrefix)] != compactionSummaryPrefix {
		return "", false
	}
	rest := text[len(compactionSummaryPrefix):]
	if len(rest) >= len(compactionSummarySuffix) &&
		rest[len(rest)-len(compactionSummarySuffix):] == compactionSummarySuffix {
		rest = rest[:len(rest)-len(compactionSummarySuffix)]
	}
	return rest, true
}

func projectionFromPiEntries(entries []SessionEntry) (coding.BranchProjection, error) {
	if len(entries) == 0 {
		return coding.BranchProjection{}, fmt.Errorf("no legacy session entries")
	}
	file, err := os.CreateTemp("", "cherrypi-session-migrate-*.jsonl")
	if err != nil {
		return coding.BranchProjection{}, err
	}
	path := file.Name()
	defer func() {
		_ = os.Remove(path)
	}()

	for _, entry := range entries {
		if entry.Raw == "" {
			continue
		}
		if _, err := file.WriteString(entry.Raw + "\n"); err != nil {
			_ = file.Close()
			return coding.BranchProjection{}, err
		}
	}
	if err := file.Close(); err != nil {
		return coding.BranchProjection{}, err
	}

	tree, err := coding.LoadSessionTree(path)
	if err != nil {
		return coding.BranchProjection{}, err
	}
	return tree.BuildProjection(), nil
}

func (e *engine) observeProjection(raw, transformed []agent.AgentMessage) {
	e.projectionMu.Lock()
	defer e.projectionMu.Unlock()
	e.projectionStamp++
	if _, _, _, ok := compactedHead(transformed); !ok {
		e.projected = nil
		e.projectedRawLen = 0
		e.projectedTrusted = 0
		return
	}
	e.projected = slices.Clone(transformed)
	e.projectedRawLen = len(raw)

	e.projectedTrusted = len(transformed)
}

func (e *engine) modelView() (view []agent.AgentMessage, stamp uint64, trustedFrom int) {
	raw := e.sess.History()
	e.projectionMu.Lock()
	captured := e.projected
	rawLen := e.projectedRawLen
	trusted := e.projectedTrusted
	stamp = e.projectionStamp
	e.projectionMu.Unlock()
	if len(captured) == 0 || rawLen < 0 || rawLen > len(raw) {
		return raw, stamp, 0
	}
	view = make([]agent.AgentMessage, 0, len(captured)+len(raw)-rawLen)
	view = append(view, captured...)
	view = append(view, raw[rawLen:]...)
	return view, stamp, trusted
}

func (e *engine) needsCompaction(view []agent.AgentMessage, trustedFrom int) bool {
	if e.compaction == nil || !e.compaction.Enabled || e.sess.Model == nil {
		return false
	}
	window := e.sess.Model.ContextWindow
	if window <= 0 {
		return false
	}
	return estimateViewTokens(view, trustedFrom) > window-e.compaction.ReserveTokens
}

func (e *engine) exportProjection() (coding.BranchProjection, error) {
	raw := e.sess.History()
	e.projectionMu.Lock()
	view := slices.Clone(e.projected)
	rawLen := e.projectedRawLen
	e.projectionMu.Unlock()

	if len(view) == 0 {
		return projectionFromMessages(raw), nil
	}
	if projection, ok := projectionFromView(view, raw, rawLen); ok {
		return projection, nil
	}
	return projectionFromMessages(raw), nil
}
