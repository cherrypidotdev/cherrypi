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
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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
}

// historyEntry is one line of Pi's session JSONL.
type historyEntry struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
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
	path := s.sessionFilePath()
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var out []historyEvent
	for offset := int64(0); offset < int64(len(data)); {
		rest := data[offset:]
		next := bytes.IndexByte(rest, '\n')
		var line []byte
		var size int64
		if next < 0 {
			line = rest
			size = int64(len(rest))
		} else {
			line = rest[:next]
			size = int64(next) + 1
		}
		for _, ev := range parseHistoryLine(line) {
			out = append(out, historyEvent{event: ev, offset: offset})
		}
		offset += size
	}
	return out
}

// parseHistoryLine decodes one JSONL line into zero or more events.
func parseHistoryLine(line []byte) []Event {
	if len(line) == 0 {
		return nil
	}

	var entry historyEntry
	if json.Unmarshal(line, &entry) != nil || entry.Type != "message" {
		return nil
	}
	var msg historyMessage
	if json.Unmarshal(entry.Message, &msg) != nil {
		return nil
	}
	at := parseHistoryTime(entry.Timestamp)

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
					events = append(
						events,
						Event{Kind: EventThinking, Delta: block.Thinking, At: at},
					)
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

func (s *Session) sessionFilePath() string {
	dir := s.sessionDir()

	matches, err := filepath.Glob(filepath.Join(dir, "*_"+s.PiSessionID+".jsonl"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 {
		return matches[0]
	}

	sort.Slice(matches, func(i, j int) bool {
		left, err := os.Stat(matches[i])
		if err != nil {
			return false
		}
		right, err := os.Stat(matches[j])
		if err != nil {
			return true
		}
		return left.ModTime().After(right.ModTime())
	})
	return matches[0]
}

func (s *Session) sessionDir() string {
	if s.cfg.PiSessionDir != "" {
		return s.cfg.PiSessionDir
	}
	agentDir := s.cfg.AgentDir
	if agentDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			agentDir = filepath.Join(home, ".pi", "agent")
		}
	}
	return filepath.Join(agentDir, "sessions", encodeSessionCwd(s.Workspace))
}

func encodeSessionCwd(cwd string) string {
	trimmed := strings.TrimPrefix(cwd, "/")
	trimmed = strings.TrimPrefix(trimmed, "\\")
	trimmed = strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(trimmed)
	return "--" + trimmed + "--"
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
