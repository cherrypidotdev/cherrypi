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
	"strings"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
)

const defaultBashTimeoutSeconds = 300

func (s *Session) wrapTools(tools []agent.AgentTool) []agent.AgentTool {
	out := make([]agent.AgentTool, len(tools))
	for i, tool := range tools {
		out[i] = s.wrapTool(tool)
	}
	return out
}

func (s *Session) wrapTool(tool agent.AgentTool) agent.AgentTool {
	if tool.Name == "bash" {
		tool.Description = strings.Replace(
			tool.Description,
			"Optionally provide a timeout in seconds.",
			"Optionally provide a timeout in seconds; the default is 300 (5 minutes).",
			1,
		)
		tool.Parameters = withBashTimeoutDefault(tool.Parameters)
		prepare := tool.PrepareArguments
		tool.PrepareArguments = func(raw map[string]any) map[string]any {
			args := raw
			if prepare != nil {
				if next := prepare(raw); next != nil {
					args = next
				}
			}

			if value, ok := args["timeout"]; ok && value != nil {
				return args
			}
			out := make(map[string]any, len(args)+1)
			for key, value := range args {
				out[key] = value
			}
			out["timeout"] = defaultBashTimeoutSeconds
			return out
		}
	}
	execute := tool.Execute
	if execute == nil {
		return tool
	}

	redactUpdate := func(update agent.ToolUpdateFunc) agent.ToolUpdateFunc {
		if update == nil {
			return nil
		}
		return func(result agent.AgentToolResult) {
			update(s.redactResult(result))
		}
	}
	tool.Execute = func(
		ctx context.Context,
		toolCallID string,
		params map[string]any,
		onUpdate agent.ToolUpdateFunc,
	) (agent.AgentToolResult, error) {
		callCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		s.registerToolCall(toolCallID, cancel)
		defer s.unregisterToolCall(toolCallID)

		result, err := execute(callCtx, toolCallID, params, redactUpdate(onUpdate))
		result = s.redactResult(result)

		if err != nil {
			err = s.redactError(err)
		}
		if actor := s.takeToolInterrupt(toolCallID); actor != "" {
			return interruptedToolResult(actor), nil
		}
		if actor := s.stopActorName(); actor != "" && callCtx.Err() != nil {
			return interruptedToolResult(actor), nil
		}

		if err == nil && !result.IsError && s.markRequiredToolCalled(tool.Name) {
			s.notify("task_required_tool", map[string]any{
				"tool":           tool.Name,
				"required_tools": s.requiredToolsSnapshot(),
			})
		}
		return result, err
	}
	return tool
}

func withBashTimeoutDefault(schema *ai.Schema) *ai.Schema {
	if schema == nil {
		return schema
	}
	timeout, ok := schema.Properties["timeout"]
	if !ok {
		return schema
	}
	patched := *timeout
	patched.Description = "Timeout in seconds (optional; default 300, i.e. 5 minutes)"
	properties := make(map[string]*ai.Schema, len(schema.Properties))
	for name, property := range schema.Properties {
		properties[name] = property
	}
	properties["timeout"] = &patched
	copied := *schema
	copied.Properties = properties
	return &copied
}

func (s *Session) registerToolCall(callID string, cancel context.CancelFunc) {
	s.toolMu.Lock()
	defer s.toolMu.Unlock()
	if s.toolCancels == nil {
		s.toolCancels = map[string]context.CancelFunc{}
	}
	s.toolCancels[callID] = cancel
}

func (s *Session) unregisterToolCall(callID string) {
	s.toolMu.Lock()
	defer s.toolMu.Unlock()
	delete(s.toolCancels, callID)
}

func (s *Session) takeToolInterrupt(callID string) string {
	s.toolMu.Lock()
	defer s.toolMu.Unlock()
	actor := s.toolInterrupts[callID]
	delete(s.toolInterrupts, callID)
	return actor
}

func (s *Session) stopActorName() string {
	s.toolMu.Lock()
	defer s.toolMu.Unlock()
	return s.stopActor
}

func interruptedToolResult(actor string) agent.AgentToolResult {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "an operator"
	}
	return agent.AgentToolResult{
		Content: ai.ContentList{ai.TextContent{Text: "Interrupted by " + actor}},
		Details: map[string]any{"interruptedBy": actor},
		IsError: true,
	}
}
