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

type projectedEntry struct {
	SystemMessage json.RawMessage   `json:"system_message"`
	Messages      []json.RawMessage `json:"messages"`
}

type loadoutFold struct {
	order    []string
	byName   map[string]Tool
	recorded bool
}

func newLoadoutFold() *loadoutFold {
	return &loadoutFold{order: make([]string, 0, 16), byName: map[string]Tool{}}
}

func (f *loadoutFold) apply(raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var message toolLoadoutMessage
	if json.Unmarshal(raw, &message) != nil {
		return
	}
	if message.Role != "system" {
		return
	}
	if len(message.ToolsAdded) == 0 && len(message.ToolsRemoved) == 0 {
		return
	}
	f.recorded = true
	for _, removed := range message.ToolsRemoved {
		if removed.Name == "" {
			continue
		}
		if _, ok := f.byName[removed.Name]; !ok {
			continue
		}
		delete(f.byName, removed.Name)
		f.order = withoutName(f.order, removed.Name)
	}
	for _, added := range message.ToolsAdded {
		if added.Name == "" {
			continue
		}
		if _, ok := f.byName[added.Name]; !ok {
			f.order = append(f.order, added.Name)
		}
		f.byName[added.Name] = Tool{
			Name:           added.Name,
			Description:    added.Description,
			ParametersJSON: string(added.Parameters),
		}
	}
}

func (f *loadoutFold) tools() []Tool {
	tools := make([]Tool, 0, len(f.order))
	for _, name := range f.order {
		tools = append(tools, f.byName[name])
	}
	return tools
}

func CollectToolLoadout(entries []tasks.SessionHistoryEntry) ([]Tool, bool) {
	fold := newLoadoutFold()
	for _, entry := range entries {
		var envelope systemEntry
		if json.Unmarshal([]byte(entry.Raw), &envelope) != nil {
			continue
		}
		if envelope.Type != "message" {
			continue
		}
		fold.apply(envelope.Message)
	}
	return fold.tools(), fold.recorded
}

func CollectProjectedToolLoadout(entriesJSON string) ([]Tool, bool) {
	var entries []projectedEntry
	if json.Unmarshal([]byte(entriesJSON), &entries) != nil {
		return nil, false
	}
	fold := newLoadoutFold()
	for _, entry := range entries {
		fold.apply(entry.SystemMessage)
		for _, message := range entry.Messages {
			fold.apply(message)
		}
	}
	return fold.tools(), fold.recorded
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
