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
	"errors"
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
		for _, name := range p.Manifest.ProjectTools {
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
	for _, projectTool := range s.ProjectTools(ctx, t.ProjectID) {
		if projectTool.Tool.Name == tool {
			return projectTool.PluginID, nil
		}
	}
	return "", errors.New("the tool is not attached to this task")
}
