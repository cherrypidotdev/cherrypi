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

package taskhistory

import (
	"sort"
	"strings"
	"unicode/utf8"

	"cherrypi.vladfaust.com/agency/internal/tasks"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ToolCall struct {
	CallID string
	Name   string

	ArgsJSON string

	ResultJSON string

	DetailsJSON string
	IsError     bool
	StartedAt   *timestamppb.Timestamp
	EndedAt     *timestamppb.Timestamp
}

func (c ToolCall) Interrupted() bool { return c.EndedAt == nil }

func (c ToolCall) DurationMs() int64 {
	if c.StartedAt == nil || c.EndedAt == nil {
		return 0
	}
	ms := c.EndedAt.AsTime().Sub(c.StartedAt.AsTime()).Milliseconds()
	if ms < 0 {
		return 0
	}
	return ms
}

type ToolCallFilter struct {
	Names []string

	ErrorsOnly bool

	InterruptedOnly bool

	Search string

	Limit int

	Offset int
}

const (
	defaultToolCallPage = 100
	maxToolCallPage     = 500

	toolCallArgsPreview = 2048
)

func CollectToolCalls(entries []tasks.SessionHistoryEntry) []ToolCall {
	return CollectToolCallsFromEvents(Events("", entries))
}

func CollectToolCallsFromEvents(events []*agentv1.TaskEvent) []ToolCall {
	var calls []ToolCall
	index := map[string]int{}
	for _, event := range events {
		tool := event.GetTool()
		if tool == nil || tool.ToolCallId == "" {
			continue
		}
		i, ok := index[tool.ToolCallId]
		if !ok {
			index[tool.ToolCallId] = len(calls)
			calls = append(calls, ToolCall{
				CallID:    tool.ToolCallId,
				Name:      tool.ToolName,
				ArgsJSON:  tool.ArgsJson,
				StartedAt: event.At,
			})
			i = len(calls) - 1
		}
		call := &calls[i]
		if call.Name == "" {
			call.Name = tool.ToolName
		}
		if tool.ArgsJson != "" {
			call.ArgsJSON = tool.ArgsJson
		}
		if tool.Phase == agentv1.ToolPhase_TOOL_PHASE_FINISHED {
			call.ResultJSON = tool.ResultJson
			call.DetailsJSON = tool.DetailsJson
			call.IsError = tool.IsError
			call.EndedAt = event.At
		} else if call.StartedAt == nil {
			call.StartedAt = event.At
		}
	}
	return calls
}

func ToolCallNames(calls []ToolCall) []string {
	seen := map[string]struct{}{}
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		if call.Name == "" {
			continue
		}
		if _, ok := seen[call.Name]; ok {
			continue
		}
		seen[call.Name] = struct{}{}
		names = append(names, call.Name)
	}
	sort.Strings(names)
	return names
}

func FilterToolCalls(calls []ToolCall, filter ToolCallFilter) ([]ToolCall, int) {
	names := map[string]struct{}{}
	for _, name := range filter.Names {
		if name != "" {
			names[name] = struct{}{}
		}
	}
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	matched := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		if len(names) > 0 {
			if _, ok := names[call.Name]; !ok {
				continue
			}
		}
		if filter.ErrorsOnly && !call.IsError {
			continue
		}
		if filter.InterruptedOnly && !call.Interrupted() {
			continue
		}
		if search != "" &&
			!strings.Contains(strings.ToLower(call.Name), search) &&
			!strings.Contains(strings.ToLower(call.ArgsJSON), search) &&
			!strings.Contains(strings.ToLower(call.ResultJSON), search) {
			continue
		}
		matched = append(matched, call)
	}

	sort.SliceStable(matched, func(i, j int) bool {
		return startedAtMs(matched[i]) > startedAtMs(matched[j])
	})
	total := len(matched)
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultToolCallPage
	}
	if limit > maxToolCallPage {
		limit = maxToolCallPage
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(matched) {
		return []ToolCall{}, total
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	return matched[offset:end], total
}

func startedAtMs(call ToolCall) int64 {
	if call.StartedAt == nil {
		return 0
	}
	return call.StartedAt.AsTime().UnixMilli()
}

func (c ToolCall) ArgsPreview() string {
	return preview(c.ArgsJSON, toolCallArgsPreview)
}

func (c ToolCall) ResultBytes() int { return len(c.ResultJSON) }

const ToolCallBodyLimit = 256 << 10

func PreviewBody(s string, limit int) (string, bool) {
	if limit <= 0 || len(s) <= limit {
		return s, false
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "\n\n…(truncated)", true
}

func preview(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
