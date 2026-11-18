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
	"slices"
	"strings"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
)

const (
	maxRequiredGateBlocks = 2

	maxRequiredSettleNudges = 1
)

func (s *Session) setRequiredTools(tools []string) {
	normalized := normalizeToolNames(tools)
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.Equal(normalized, s.requiredTools) {
		return
	}
	s.requiredTools = normalized
	s.requiredSatisfied = len(normalized) == 0
	s.requiredGateBlocks = 0
	s.requiredBatch = nil
	s.requiredBatchBlocked = false
	s.requiredSettleNudges = 0
}

func (s *Session) requiredToolsSnapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requiredTools...)
}

func (s *Session) requiredToolsState() (tools []string, satisfied bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requiredTools...), s.requiredSatisfied
}

func normalizeToolNames(tools []string) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool)
		if name != "" && !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

func batchCallsRequiredTool(message *ai.AssistantMessage, tools []string) bool {
	if message == nil || len(tools) == 0 {
		return false
	}
	for _, content := range message.Content {
		call, ok := content.(ai.ToolCall)
		if ok && slices.Contains(tools, call.Name) {
			return true
		}
	}
	return false
}

func (s *Session) requiredToolGate(
	_ context.Context,
	c agent.BeforeToolCallContext,
) *agent.BeforeToolCallResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requiredTools) == 0 || s.requiredSatisfied {
		return nil
	}

	if c.AssistantMessage == nil {
		return nil
	}

	if c.AssistantMessage == s.requiredBatch {
		if s.requiredBatchBlocked {
			return &agent.BeforeToolCallResult{
				Block:  true,
				Reason: requiredToolReminder(s.requiredTools),
			}
		}
		return nil
	}
	s.requiredBatch = c.AssistantMessage
	if batchCallsRequiredTool(c.AssistantMessage, s.requiredTools) {
		s.requiredBatchBlocked = false
		return nil
	}
	if s.requiredGateBlocks >= maxRequiredGateBlocks {
		s.requiredBatchBlocked = false
		return nil
	}
	s.requiredGateBlocks++
	s.requiredBatchBlocked = true
	return &agent.BeforeToolCallResult{
		Block:  true,
		Reason: requiredToolReminder(s.requiredTools),
	}
}

func (s *Session) markRequiredToolCalled(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requiredTools) == 0 || s.requiredSatisfied {
		return false
	}
	if !slices.Contains(s.requiredTools, name) {
		return false
	}
	s.requiredSatisfied = true
	return true
}

func (s *Session) shouldContinueWithoutRequiredTool() (ai.Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requiredTools) == 0 || s.requiredSatisfied {
		return nil, false
	}
	if s.requiredSettleNudges >= maxRequiredSettleNudges {
		return nil, false
	}
	s.requiredSettleNudges++
	return progressReminder(
		"Required tool reminder: this task must call " +
			requiredToolPhrase(s.requiredTools) +
			" before it can settle, and this run has not. Call it now with the " +
			"arguments the task needs." + carryOnRequiredTool,
	), true
}

func requiredToolReminder(tools []string) string {
	return "Required tool reminder: this task must call " +
		requiredToolPhrase(tools) +
		" before it can settle. Call it now instead of another tool; the rest " +
		"of the work follows it." + carryOnRequiredTool
}

const carryOnRequiredTool = " Then carry on with the work immediately in the " +
	"same turn: the requirement is part of the task, not a turn of its own."

func requiredToolPhrase(tools []string) string {
	quoted := make([]string, len(tools))
	for i, tool := range tools {
		quoted[i] = "`" + tool + "`"
	}
	if len(quoted) == 1 {
		return quoted[0]
	}
	return strings.Join(quoted[:len(quoted)-1], ", ") + " or " +
		quoted[len(quoted)-1]
}
