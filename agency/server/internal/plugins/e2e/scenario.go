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

package e2e

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Scenario struct {
	Agency   AgencySpec        `yaml:"agency"`
	Vars     map[string]string `yaml:"vars"`
	Settings SettingsSpec      `yaml:"settings"`
	Project  ProjectSpec       `yaml:"project"`
	Steps    []Step            `yaml:"steps"`
}

type AgencySpec struct {
	Project ProjectSeed `yaml:"project"`
	Agent   AgentSeed   `yaml:"agent"`
	Tags    []string    `yaml:"tags"`
}

type ProjectSeed struct {
	Name   string `yaml:"name"`
	Slug   string `yaml:"slug"`
	Prefix string `yaml:"prefix"`
}

type AgentSeed struct {
	Username string `yaml:"username"`
	Name     string `yaml:"name"`

	ToolArgs map[string]any `yaml:"tool_args"`
}

type SettingsSpec struct {
	Org     map[string]any `yaml:"org"`
	Project map[string]any `yaml:"project"`
	Agent   map[string]any `yaml:"agent"`
	User    map[string]any `yaml:"user"`
}

type ProjectSpec struct {
	Enabled           *bool `yaml:"enabled"`
	AllowTaskCreation *bool `yaml:"allow_task_creation"`
	MaxPriority       *int  `yaml:"max_priority"`
}

type Step struct {
	Name   string     `yaml:"name"`
	HTTP   *HTTPStep  `yaml:"http"`
	Tool   *ToolStep  `yaml:"tool"`
	Wait   *WaitStep  `yaml:"wait"`
	Task   *TaskStep  `yaml:"task"`
	Event  *EventStep `yaml:"event"`
	Sleep  string     `yaml:"sleep"`
	Expect Expect     `yaml:"expect"`
}

type HTTPStep struct {
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	URL     string            `yaml:"url"`
	Body    string            `yaml:"body"`
	JSON    map[string]any    `yaml:"json"`
	Headers map[string]string `yaml:"headers"`
}

type ToolStep struct {
	Name string         `yaml:"name"`
	Args map[string]any `yaml:"args"`
	Task string         `yaml:"task"`
}

type WaitStep struct {
	Task    string `yaml:"task"`
	State   string `yaml:"state"`
	Timeout string `yaml:"timeout"`
}

type TaskStep struct {
	Project string `yaml:"project"`
	Name    string `yaml:"name"`
	Prompt  string `yaml:"prompt"`

	State string `yaml:"state"`

	Assignee string `yaml:"assignee"`
}

type EventStep struct {
	Name string `yaml:"name"`

	Task    string         `yaml:"task"`
	Payload map[string]any `yaml:"payload"`
}

func (e *EventStep) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		return node.Decode(&e.Name)
	}
	if err := checkKeys(node, "name", "task", "payload"); err != nil {
		return err
	}
	type plain EventStep
	return node.Decode((*plain)(e))
}

func checkKeys(node *yaml.Node, allowed ...string) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a mapping, got %s", node.ShortTag())
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		known := false
		for _, name := range allowed {
			if name == key {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("unknown field %q (want %s)", key, strings.Join(allowed, ", "))
		}
	}
	return nil
}

type Expect struct {
	Status       int            `yaml:"status"`
	JSON         map[string]any `yaml:"json"`
	TextContains string         `yaml:"text_contains"`
	Tasks        []TaskExpect   `yaml:"tasks"`
}

type TaskExpect struct {
	Project        string   `yaml:"project"`
	State          string   `yaml:"state"`
	Assignee       string   `yaml:"assignee"`
	Tags           []string `yaml:"tags"`
	PromptContains string   `yaml:"prompt_contains"`
	NameContains   string   `yaml:"name_contains"`
}

func LoadScenario(path string) (Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, err
	}
	var scenario Scenario
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)
	if err := dec.Decode(&scenario); err != nil {
		return Scenario{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if scenario.Agency.Project.Name == "" {
		return Scenario{}, fmt.Errorf("%s: agency.project.name is required", path)
	}
	if len(scenario.Steps) == 0 {
		return Scenario{}, fmt.Errorf("%s: at least one step is required", path)
	}
	for i, step := range scenario.Steps {
		actions := 0
		if step.HTTP != nil {
			actions++
		}
		if step.Tool != nil {
			actions++
		}
		if step.Wait != nil {
			actions++
		}
		if step.Task != nil {
			actions++
		}
		if step.Event != nil {
			actions++
		}
		if step.Sleep != "" {
			actions++
		}
		if actions != 1 {
			return Scenario{}, fmt.Errorf(
				"%s: step %d (%s) must have exactly one action",
				path, i+1, step.Name,
			)
		}
		if step.Task != nil && strings.TrimSpace(step.Task.Prompt) == "" {
			return Scenario{}, fmt.Errorf(
				"%s: step %d (%s): task.prompt is required",
				path, i+1, step.Name,
			)
		}
		if step.Event != nil && strings.TrimSpace(step.Event.Name) == "" {
			return Scenario{}, fmt.Errorf(
				"%s: step %d (%s): event.name is required",
				path, i+1, step.Name,
			)
		}
	}
	return scenario, nil
}

func expand(value string, vars map[string]string) string {
	for {
		start := strings.Index(value, "${")
		if start < 0 {
			return value
		}
		end := strings.Index(value[start:], "}")
		if end < 0 {
			return value
		}
		end += start
		key := value[start+2 : end]
		replacement, ok := vars[key]
		if !ok {
			if name, found := strings.CutPrefix(key, "env."); found {
				replacement = os.Getenv(name)
			} else {
				replacement = value[start : end+1]
			}
		}
		value = value[:start] + replacement + value[end+1:]
	}
}

func expandValue(value any, vars map[string]string) any {
	switch v := value.(type) {
	case string:
		return expand(v, vars)
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = expandValue(item, vars)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = expandValue(item, vars)
		}
		return out
	default:
		return value
	}
}
