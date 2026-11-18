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
	"strings"
	"time"
)

type historyBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type historyMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Details    json.RawMessage `json:"details"`
	IsError    bool            `json:"isError"`

	Timestamp  int64 `json:"timestamp"`
	DurationMs int64 `json:"durationMs"`
}

// historyEntry is one line of Pi's session JSONL.
type historyEntry struct {
	Type       string          `json:"type"`
	Timestamp  string          `json:"timestamp"`
	Message    json.RawMessage `json:"message"`
	CustomType string          `json:"customType"`
	Data       json.RawMessage `json:"data"`
}

type historyEvent struct {
	event  Event
	offset int64
}

func (s *Session) loadHistory() []Event {
	all := s.readHistory()
	events := make([]Event, 0, len(all))
	for _, h := range all {
		events = append(events, h.event)
	}
	return events
}

func (s *Session) loadHistoryPage(before int64, limit int) ([]Event, int64) {
	all := s.readHistory()

	older := all
	if before > 0 {
		older = nil
		for _, h := range all {
			if h.offset < before {
				older = append(older, h)
			}
		}
	}

	var nextBefore int64
	if limit > 0 && len(older) > limit {
		older = older[len(older)-limit:]
		nextBefore = older[0].offset
	}

	events := make([]Event, 0, len(older))
	for _, h := range older {
		events = append(events, h.event)
	}
	return events, nextBefore
}

// readHistory parses every message line of the session file in order, keeping

func (s *Session) readHistory() []historyEvent {

	s.mu.Lock()
	lines := append([]string(nil), s.historyRaw...)
	s.mu.Unlock()

	var out []historyEvent
	var offset int64
	parser := &historyParser{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		for _, event := range parser.parseLine([]byte(trimmed)) {
			out = append(out, historyEvent{event: event, offset: offset})
		}
		offset += int64(len(line)) + 1
	}
	return out
}

func parseCustomHistoryEntry(entry historyEntry, at time.Time) []Event {
	switch entry.CustomType {
	case modelWaitCustomType:
		wait, ok := parseModelWait(entry.Data)
		if !ok {
			return nil
		}
		return []Event{{Kind: EventModelWait, At: at, ModelWait: wait}}
	case thinkingSpanCustomType:
		span, ok := parseThinkingSpan(entry.Data)
		if !ok {
			return nil
		}
		return []Event{{Kind: EventThinkingSpan, At: at, ThinkingSpan: span}}
	}
	return nil
}

type historyParser struct {
	lastWaitEnd time.Time
}

// parseHistoryLine decodes one JSONL line on its own, without the timing context

func parseHistoryLine(line []byte) []Event {
	return new(historyParser).parseLine(line)
}

func (p *historyParser) legacyThinkingSpan(
	chars int32,
	msg historyMessage,
	at time.Time,
) *ThinkingSpanEvent {
	end := int64(0)
	if !at.IsZero() {
		end = at.UnixMilli()
	} else if msg.Timestamp > 0 && msg.DurationMs > 0 {
		end = msg.Timestamp + msg.DurationMs
	}
	if end <= 0 {
		return nil
	}
	start := msg.Timestamp
	if start <= 0 && msg.DurationMs > 0 {
		start = end - msg.DurationMs
	}
	if start <= 0 {
		return nil
	}
	if !p.lastWaitEnd.IsZero() {
		waitEnd := p.lastWaitEnd.UnixMilli()
		if waitEnd > start && waitEnd <= end {
			start = waitEnd
		}
	}
	duration := end - start
	if duration <= 0 || chars <= 0 {
		return nil
	}
	return &ThinkingSpanEvent{DurationMs: duration, Chars: chars, Estimated: true}
}

// parseLine decodes one JSONL line into zero or more events.
func (p *historyParser) parseLine(line []byte) []Event {
	if len(line) == 0 {
		return nil
	}

	var entry historyEntry
	if json.Unmarshal(line, &entry) != nil {
		return nil
	}
	at := parseHistoryTime(entry.Timestamp)
	if entry.Type == "custom" {
		events := parseCustomHistoryEntry(entry, at)
		if entry.CustomType == modelWaitCustomType && len(events) > 0 {
			p.lastWaitEnd = at
		}
		return events
	}
	if entry.Type != "message" {
		return nil
	}
	var msg historyMessage
	if json.Unmarshal(entry.Message, &msg) != nil {
		return nil
	}

	switch msg.Role {
	case "user":
		if text := contentText(msg.Content); text != "" {
			return []Event{{Kind: EventUser, Text: text, At: at}}
		}
	case "assistant":
		var blocks []historyBlock
		if json.Unmarshal(msg.Content, &blocks) != nil {
			return nil
		}
		var events []Event
		for _, block := range blocks {
			switch block.Type {
			case "text":
				if block.Text != "" {
					events = append(events, Event{Kind: EventText, Delta: block.Text, At: at})
				}
			case "thinking":
				if block.Thinking != "" {
					ev := Event{Kind: EventThinking, Delta: block.Thinking, At: at}
					ev.ThinkingSpan = p.legacyThinkingSpan(
						int32(len([]rune(block.Thinking))), msg, at,
					)
					events = append(events, ev)
				}
			case "toolCall":
				events = append(events, Event{Kind: EventTool, At: at, Tool: &ToolEvent{
					CallID:   block.ID,
					Name:     block.Name,
					Phase:    ToolStarted,
					ArgsJSON: string(block.Arguments),
				}})
			}
		}
		return events
	case "toolResult":
		return []Event{{Kind: EventTool, At: at, Tool: &ToolEvent{
			CallID:      msg.ToolCallID,
			Name:        msg.ToolName,
			Phase:       ToolFinished,
			ResultJSON:  contentText(msg.Content),
			DetailsJSON: rawString(msg.Details),
			IsError:     msg.IsError,
		}}}
	}

	return nil
}

func parseHistoryTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}
