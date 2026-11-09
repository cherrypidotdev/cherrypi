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

package scheduler

import (
	"context"
	"sort"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/limits"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

type fitView struct {
	s            *Scheduler
	limits       map[ids.AgentID]limits.Limits
	reservations map[ids.AgentID]map[string]int
	projects     map[string]projectFit
}

type projectFit struct {
	reserved map[string]int
	capacity int
}

func (s *Scheduler) newFitView() *fitView {
	return &fitView{
		s:            s,
		limits:       map[ids.AgentID]limits.Limits{},
		reservations: map[ids.AgentID]map[string]int{},
		projects:     map[string]projectFit{},
	}
}

func (v *fitView) agentLimits(ctx context.Context, agentID ids.AgentID) (limits.Limits, error) {
	if lim, ok := v.limits[agentID]; ok {
		return lim, nil
	}
	lim, err := v.s.Limits.Get(ctx, agentID)
	if err != nil {
		return limits.Limits{}, err
	}
	v.limits[agentID] = lim
	return lim, nil
}

func (v *fitView) agentReservations(
	ctx context.Context,
	agentID ids.AgentID,
) (map[string]int, error) {
	if reserved, ok := v.reservations[agentID]; ok {
		return reserved, nil
	}
	reserved, err := v.s.Limits.AgentReservations(ctx, agentID)
	if err != nil {
		return nil, err
	}
	v.reservations[agentID] = reserved
	return reserved, nil
}

func (v *fitView) agentBlocks(
	ctx context.Context,
	agentID ids.AgentID,
	cap int,
	tags []tasks.Tag,
) (bool, string, error) {
	reserved, err := v.agentReservations(ctx, agentID)
	if err != nil {
		return false, "", err
	}
	if len(reserved) == 0 {
		return false, "", nil
	}
	counts, err := v.s.Tasks.AgentClassCounts(ctx, agentID, reserved)
	if err != nil {
		return false, "", err
	}
	return floorBlocks(reserved, counts, cap, tags), blockingClass(reserved, tags), nil
}

func (v *fitView) project(ctx context.Context, projectID string) (projectFit, error) {
	if fit, ok := v.projects[projectID]; ok {
		return fit, nil
	}
	reserved, err := v.s.Limits.ProjectReservationMap(ctx, projectID)
	if err != nil {
		return projectFit{}, err
	}
	fit := projectFit{reserved: reserved}
	if len(reserved) > 0 {
		capacity, err := v.projectCapacity(ctx, projectID)
		if err != nil {
			return projectFit{}, err
		}
		fit.capacity = capacity
	}
	v.projects[projectID] = fit
	return fit, nil
}

func (v *fitView) projectBlocks(
	ctx context.Context,
	projectID string,
	tags []tasks.Tag,
) (bool, string, error) {
	fit, err := v.project(ctx, projectID)
	if err != nil {
		return false, "", err
	}
	if len(fit.reserved) == 0 {
		return false, "", nil
	}
	counts, err := v.s.Tasks.ProjectClassCounts(ctx, projectID, fit.reserved)
	if err != nil {
		return false, "", err
	}
	return floorBlocks(fit.reserved, counts, fit.capacity, tags),
		blockingClass(fit.reserved, tags), nil
}

func (v *fitView) projectCapacity(ctx context.Context, projectID string) (int, error) {
	members, err := v.s.Projects.ListAgents(ctx, projectID)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, member := range members {
		if !member.CanCreateTasks {
			continue
		}
		lim, err := v.agentLimits(ctx, member.AgentID)
		if err != nil {
			return 0, err
		}
		total += projectCap(lim, projectID)
	}
	return total, nil
}

func projectCap(lim limits.Limits, projectID string) int {
	if override, ok := lim.Projects[projectID]; ok {
		return override
	}
	return lim.MaxConcurrentPerProject
}

func floorBlocks(
	reserved map[string]int,
	counts tasks.ClassCounts,
	cap int,
	tags []tasks.Tag,
) bool {
	if len(reserved) == 0 {
		return false
	}
	return !tasks.FitsClass(counts, cap, reserved, tags)
}

func blockingClass(reserved map[string]int, tags []tasks.Tag) string {
	names := make([]string, 0, len(reserved))
	for name, slots := range reserved {
		if slots <= 0 || taskHasTag(tags, name) {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		for name, slots := range reserved {
			if slots > 0 {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Slice(names, func(i, j int) bool {
		if reserved[names[i]] != reserved[names[j]] {
			return reserved[names[i]] > reserved[names[j]]
		}
		return names[i] < names[j]
	})
	return names[0]
}

func taskHasTag(tags []tasks.Tag, name string) bool {
	for _, tag := range tags {
		if strings.ToLower(tag.Name) == name {
			return true
		}
	}
	return false
}
