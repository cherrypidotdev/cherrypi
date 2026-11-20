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
	"errors"
	"sort"
	"strings"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
	"github.com/sky-valley/pi/coding"
)

const secretRedactionMinLength = 8

const redactionMarker = "[secret:"

func (s *Session) refreshVariables(ctx context.Context) {
	resolve := s.refreshFn
	if resolve == nil {
		resolve = s.catalog
	}
	catalog := resolve(ctx)
	if catalog == nil {
		return
	}
	s.mu.Lock()
	s.variables = catalog.variables
	s.mu.Unlock()
}

func (s *Session) variablesSnapshot() []AgencyVariable {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.variables) == 0 {
		return nil
	}
	out := make([]AgencyVariable, len(s.variables))
	copy(out, s.variables)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Session) variablesPrompt() string {
	variables := s.variablesSnapshot()
	if len(variables) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Your shell receives these environment variables:\n")
	for _, variable := range variables {
		b.WriteString("- ")
		b.WriteString(variable.Name)
		if variable.Secret {
			b.WriteString(" (secret; never print, echo or write its value)")
		}
		if note := strings.TrimSpace(variable.Note); note != "" {
			b.WriteString(": ")
			b.WriteString(note)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (s *Session) bashSpawnHook(spawn coding.BashSpawnContext) coding.BashSpawnContext {
	variables := s.variablesSnapshot()
	s.mu.Lock()
	provider, model, level := s.modelProvider, s.model, s.thinkingLevel
	s.mu.Unlock()
	if level == "" {
		level = string(coding.DefaultThinkingLevel)
	}

	owned := make(map[string]bool, len(variables))
	for _, variable := range variables {
		owned[variable.Name] = true
	}
	env := make([]string, 0, len(spawn.Env)+len(variables)+4)
	for _, entry := range spawn.Env {
		name, _, ok := strings.Cut(entry, "=")
		if ok && owned[name] {
			continue
		}
		env = append(env, entry)
	}

	if provider != "" {
		env = append(env, "PI_PROVIDER="+provider)
	}
	if model != "" {
		env = append(env, "PI_MODEL="+model)
	}
	env = append(env, "PI_REASONING_LEVEL="+level)
	for _, variable := range variables {
		env = append(env, variable.Name+"="+variable.Value)
	}
	spawn.Env = env
	return spawn
}

func (s *Session) withVariableShell(tools []agent.AgentTool) []agent.AgentTool {
	out := make([]agent.AgentTool, len(tools))
	copy(out, tools)
	for i, tool := range out {
		if tool.Name != "bash" {
			continue
		}
		out[i] = coding.CreateBashTool(s.Workspace, &coding.BashToolOptions{
			SpawnHook: s.bashSpawnHook,
		})
	}
	return out
}

func (s *Session) redactResult(result agent.AgentToolResult) agent.AgentToolResult {
	secrets := s.secretValues()
	if len(secrets) == 0 {
		return result
	}
	if len(result.Content) > 0 {
		content := make(ai.ContentList, len(result.Content))
		copy(content, result.Content)
		for i, block := range content {
			text, ok := block.(ai.TextContent)
			if !ok {
				continue
			}
			text.Text = redactValues(text.Text, secrets)
			content[i] = text
		}
		result.Content = content
	}
	if result.Details != nil {
		result.Details = redactValue(result.Details, secrets)
	}
	return result
}

func (s *Session) redactError(err error) error {
	secrets := s.secretValues()
	if len(secrets) == 0 {
		return err
	}
	text := err.Error()
	redacted := redactValues(text, secrets)
	if redacted == text {
		return err
	}
	return errors.New(redacted)
}

func (s *Session) secretValues() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]string{}
	for _, variable := range s.variables {
		if !variable.Secret || len(variable.Value) < secretRedactionMinLength {
			continue
		}
		out[variable.Value] = variable.Name
	}
	return out
}

func redactValues(text string, secrets map[string]string) string {
	if text == "" {
		return text
	}
	for value, name := range secrets {
		if strings.Contains(text, value) {
			text = strings.ReplaceAll(text, value, redactionMarker+name+"]")
		}
	}
	return text
}

func redactValue(value any, secrets map[string]string) any {
	switch typed := value.(type) {
	case string:
		return redactValues(typed, secrets)
	case []string:
		out := make([]string, len(typed))
		for i, entry := range typed {
			out[i] = redactValues(entry, secrets)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, entry := range typed {
			out[i] = redactValue(entry, secrets)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, entry := range typed {
			out[key] = redactValue(entry, secrets)
		}
		return out
	default:
		return value
	}
}
