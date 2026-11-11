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
	"encoding/json"

	"cherrypi.vladfaust.com/agency/internal/tasks"
)

type Tool struct {
	Name        string
	Description string

	ParametersJSON string
}

type toolDeclaration struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type toolLoadoutMessage struct {
	Role         string            `json:"role"`
	ToolsAdded   []toolDeclaration `json:"toolsAdded"`
	ToolsRemoved []toolDeclaration `json:"toolsRemoved"`
}

// systemEntry is the envelope of a persisted Pi session line as far as the tool

type systemEntry struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

func CollectToolLoadout(entries []tasks.SessionHistoryEntry) ([]Tool, bool) {
	order := make([]string, 0, 16)
	byName := map[string]Tool{}
	recorded := false
	for _, entry := range entries {
		var envelope systemEntry
		if json.Unmarshal([]byte(entry.Raw), &envelope) != nil {
			continue
		}
		if envelope.Type != "message" || len(envelope.Message) == 0 {
			continue
		}
		var message toolLoadoutMessage
		if json.Unmarshal(envelope.Message, &message) != nil {
			continue
		}
		if message.Role != "system" {
			continue
		}
		if len(message.ToolsAdded) == 0 && len(message.ToolsRemoved) == 0 {
			continue
		}
		recorded = true
		for _, removed := range message.ToolsRemoved {
			if removed.Name == "" {
				continue
			}
			if _, ok := byName[removed.Name]; !ok {
				continue
			}
			delete(byName, removed.Name)
			order = withoutName(order, removed.Name)
		}
		for _, added := range message.ToolsAdded {
			if added.Name == "" {
				continue
			}
			if _, ok := byName[added.Name]; !ok {
				order = append(order, added.Name)
			}
			byName[added.Name] = Tool{
				Name:           added.Name,
				Description:    added.Description,
				ParametersJSON: string(added.Parameters),
			}
		}
	}
	tools := make([]Tool, 0, len(order))
	for _, name := range order {
		tools = append(tools, byName[name])
	}
	return tools, recorded
}

func withoutName(order []string, name string) []string {
	for i, entry := range order {
		if entry != name {
			continue
		}
		return append(order[:i:i], order[i+1:]...)
	}
	return order
}
