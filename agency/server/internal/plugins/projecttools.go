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

package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

type ProjectTool struct {
	PluginID   string
	PluginName string
	Tool       Tool
}

func (s *Service) ProjectTools(ctx context.Context, projectID string) []ProjectTool {
	return s.ProjectToolsForTask(ctx, "", projectID)
}

func (s *Service) ProjectToolsForTask(
	ctx context.Context,
	taskID, projectID string,
) []ProjectTool {
	return s.projectToolsForTask(ctx, taskID, projectID, "")
}

func (s *Service) projectToolsForTask(
	ctx context.Context,
	taskID, projectID, onlyTool string,
) []ProjectTool {
	if s == nil || s.Store == nil || !s.Store.ready() || projectID == "" {
		return nil
	}
	list, err := s.Store.List(ctx)
	if err != nil {
		log.Printf("[plugins] list for project tools: %v", err)
		return nil
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	claimed := map[string]string{}
	var out []ProjectTool
	for _, p := range list {
		if !p.Enabled || len(p.Manifest.ProjectTools) == 0 {
			continue
		}
		if onlyTool != "" && !p.Manifest.declaresProjectTool(onlyTool) {
			continue
		}
		row, err := s.Store.ProjectSettings(ctx, p.ID, projectID)
		if err != nil || !row.Enabled {
			continue
		}
		if s.Projects != nil {
			role, ok, err := s.Projects.EffectiveRole(
				ctx, projectID, ids.ActorID(p.ActorID), false,
			)
			if err != nil || !ok || role == "" {
				continue
			}
		}
		for _, name := range s.taskToolNames(ctx, p, taskID) {
			tool, ok := p.Manifest.Tool(name)
			if !ok {
				continue
			}
			if owner, taken := claimed[name]; taken {
				log.Printf(
					"[plugins] project tool %q of %s is shadowed by %s in project %s",
					name, p.ID, owner, projectID,
				)
				continue
			}
			claimed[name] = p.ID
			out = append(out, ProjectTool{
				PluginID:   p.ID,
				PluginName: p.Name,
				Tool:       tool,
			})
		}
	}
	return out
}

func (s *Service) taskToolNames(
	ctx context.Context,
	p Plugin,
	taskID string,
) []string {
	declared := p.Manifest.ProjectTools
	if taskID == "" || !p.Manifest.HasHandler(HandlerTaskTools) {
		return declared
	}
	inv := Invocation{
		ID:     newInvocationID(),
		Kind:   HandlerTaskTools,
		Caller: Caller{Kind: "agency"},
		TaskID: taskID,
	}
	names, err := s.taskToolsFromHook(ctx, p, inv)
	if err == nil {
		return names
	}

	level, message := LevelDebug, "handle_task_tools failed again; every declared project tool is attached"
	if s.markTaskToolWarning(p.ID, taskID) {
		level = LevelWarn
		message = "handle_task_tools failed; every declared project tool is attached"
		log.Printf(
			"[plugins] %s handle_task_tools for task %s: %v", p.ID, taskID, err,
		)
	}
	if s.Store != nil {
		s.Store.Log(
			ctx,
			p.ID,
			level,
			"task_tools",
			inv.ID,
			message,
			map[string]any{"task_id": taskID, "error": err.Error()},
		)
	}
	return declared
}

func (s *Service) markTaskToolWarning(pluginID, taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.taskToolWarned == nil {
		s.taskToolWarned = map[string]string{}
	}
	if s.taskToolWarned[pluginID] == taskID {
		return false
	}
	s.taskToolWarned[pluginID] = taskID
	return true
}

func (s *Service) taskToolsFromHook(
	ctx context.Context,
	p Plugin,
	inv Invocation,
) ([]string, error) {
	resp, err := s.Invoke(ctx, p, inv)
	if err != nil {
		return nil, err
	}
	return parseTaskToolNames(resp.Body, p.Manifest.ProjectTools)
}

func parseTaskToolNames(body []byte, declared []string) ([]string, error) {
	if len(body) == 0 {
		return declared, nil
	}
	var shape any
	if err := json.Unmarshal(body, &shape); err != nil {
		return nil, fmt.Errorf(
			"handle_task_tools must return a list of tool names: %w", err,
		)
	}
	switch value := shape.(type) {
	case nil:
		return declared, nil
	case map[string]any:
		if len(value) == 0 {
			return []string{}, nil
		}
	case []any:
		declaredSet := make(map[string]bool, len(declared))
		for _, name := range declared {
			declaredSet[name] = true
		}
		allowed := make(map[string]bool, len(value))
		for _, item := range value {
			name, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf(
					"handle_task_tools returned a non-string tool name: %v",
					item,
				)
			}
			if !declaredSet[name] {
				return nil, fmt.Errorf(
					"handle_task_tools returned %q, which is not a project tool",
					name,
				)
			}
			allowed[name] = true
		}
		out := make([]string, 0, len(allowed))
		for _, name := range declared {
			if allowed[name] {
				out = append(out, name)
			}
		}
		return out, nil
	}
	return nil, errors.New("handle_task_tools must return a list of tool names")
}

func (s *Service) ResolveTool(ctx context.Context, taskID, tool string) (string, error) {
	if s == nil || s.Store == nil || !s.Store.ready() {
		return "", errors.New("plugins are unavailable")
	}
	row, ok, err := s.Store.TaskPlugin(ctx, taskID)
	if err != nil {
		return "", err
	}
	if ok {
		for _, attached := range row.Tools {
			if attached.Name == tool {
				return row.PluginID, nil
			}
		}
	}
	if s.Tasks == nil {
		return "", errors.New("the tool is not attached to this task")
	}
	t, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return "", err
	}
	for _, projectTool := range s.projectToolsForTask(
		ctx, taskID, t.ProjectID, tool,
	) {
		if projectTool.Tool.Name == tool {
			return projectTool.PluginID, nil
		}
	}

	if len(s.projectToolsForTask(ctx, "", t.ProjectID, tool)) > 0 {
		return "", errors.New("the tool is not available to this task")
	}
	return "", errors.New("the tool is not attached to this task")
}
