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
	"github.com/sky-valley/pi/ai"
)

func (s *Session) forcePlanToolChoice(payload any, model *ai.Model) (any, error) {
	body, ok := payload.(map[string]any)
	if !ok {
		return payload, nil
	}

	s.mu.Lock()
	forced := s.progressForceToolChoice
	allowed := s.forcePlanAllowed

	compacting := !s.compactionStartedAt.IsZero()
	s.mu.Unlock()
	if !allowed || !forced || compacting {
		return payload, nil
	}

	if !setNamedToolChoice(body, model.Api, planToolName) {

		return payload, nil
	}

	s.mu.Lock()
	s.progressForceToolChoice = false
	s.mu.Unlock()
	return body, nil
}

func setNamedToolChoice(body map[string]any, api ai.Api, name string) bool {
	switch api {
	case ai.APIOpenAICompletions:

		body["tool_choice"] = map[string]any{
			"type":     "function",
			"function": map[string]any{"name": name},
		}
	case ai.APIOpenAIResponses, ai.APIAzureOpenAIResponses,
		ai.APIOpenAICodexResponses:

		body["tool_choice"] = map[string]any{
			"type": "function",
			"name": name,
		}
	case ai.APIAnthropicMessages:
		body["tool_choice"] = map[string]any{
			"type": "tool",
			"name": name,
		}
	case ai.APIGoogleGenerativeAI, ai.APIGoogleVertex:

		config, _ := body["toolConfig"].(map[string]any)
		if config == nil {
			config = map[string]any{}
			body["toolConfig"] = config
		}
		calling, _ := config["functionCallingConfig"].(map[string]any)
		if calling == nil {
			calling = map[string]any{}
			config["functionCallingConfig"] = calling
		}
		calling["mode"] = "ANY"
		calling["allowedFunctionNames"] = []string{name}
	case ai.APIPiMessages:

		options, _ := body["options"].(map[string]any)
		if options == nil {
			options = map[string]any{}
			body["options"] = options
		}
		options["toolChoice"] = map[string]any{
			"type":     "function",
			"function": map[string]any{"name": name},
		}
	default:
		return false
	}
	return true
}
