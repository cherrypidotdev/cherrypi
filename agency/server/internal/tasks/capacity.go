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

package tasks

import (
	"strings"

	"cherrypi.vladfaust.com/agency/internal/limits"
)

// Counts is an agent's currently running task tally, split by the dimensions
// the scheduler's caps use.
type Counts struct {
	Total int
	// ByTag maps a lowercased tag name to its count, matching the agent's
	// global by-name tag caps.
	ByTag map[string]int
	// ByProject and ByProjectTag stay keyed by ids: a project tag id is
	// already unique to one project.
	ByProject    map[string]int
	ByProjectTag map[string]map[string]int
}

// NewCounts builds an empty tally.
func NewCounts() Counts {
	return Counts{
		ByTag:        map[string]int{},
		ByProject:    map[string]int{},
		ByProjectTag: map[string]map[string]int{},
	}
}

// add records one running task.
func (c *Counts) add(projectID string, tags []Tag) {
	c.Total++
	if projectID != "" {
		c.ByProject[projectID]++
	}
	for _, tag := range tags {
		c.ByTag[strings.ToLower(tag.Name)]++
		if projectID != "" {
			if c.ByProjectTag[projectID] == nil {
				c.ByProjectTag[projectID] = map[string]int{}
			}
			c.ByProjectTag[projectID][tag.ID]++
		}
	}
}

// Fits reports whether one more task with the given project and tags may run
// on an agent. Every applicable cap must have room: the total, the project
// (explicit override or the default), each tag name with an explicit global
// cap, and each project×tag with an explicit limit. A tag or project-tag
// without a stored limit imposes no extra cap.
func Fits(l limits.Limits, c Counts, projectID string, tags []Tag) bool {
	if c.Total+1 > l.MaxConcurrent {
		return false
	}

	if projectID != "" {
		projectCap := l.MaxConcurrentPerProject
		if override, ok := l.Projects[projectID]; ok {
			projectCap = override
		}
		if c.ByProject[projectID]+1 > projectCap {
			return false
		}
	}

	for _, tag := range tags {
		name := strings.ToLower(tag.Name)
		if cap, ok := l.Tags[name]; ok && c.ByTag[name]+1 > cap {
			return false
		}
		if projectID == "" {
			continue
		}
		if cap, ok := l.ProjectTags[projectID][tag.ID]; ok &&
			c.ByProjectTag[projectID][tag.ID]+1 > cap {
			return false
		}
	}

	return true
}
