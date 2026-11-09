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

type Counts struct {
	Total int

	ByTag map[string]int

	ByProject    map[string]int
	ByProjectTag map[string]map[string]int
}

func NewCounts() Counts {
	return Counts{
		ByTag:        map[string]int{},
		ByProject:    map[string]int{},
		ByProjectTag: map[string]map[string]int{},
	}
}

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

type ClassCounts struct {
	General int

	ByClass map[string]int
}

func FitsClass(
	c ClassCounts,
	cap int,
	reserved map[string]int,
	tags []Tag,
) bool {
	if len(reserved) == 0 {
		return true
	}
	reservedTotal := 0
	spill := 0
	own := 0
	for name, slots := range reserved {
		if slots <= 0 {
			continue
		}
		reservedTotal += slots
		if extra := c.ByClass[name] - slots; extra > 0 {
			spill += extra
		}
		if !hasTag(tags, name) {
			continue
		}

		if free := slots - c.ByClass[name]; free > own {
			own = free
		}
	}
	generalFree := cap - reservedTotal - c.General - spill
	return own+generalFree >= 1
}

func hasTag(tags []Tag, name string) bool {
	for _, tag := range tags {
		if strings.ToLower(tag.Name) == name {
			return true
		}
	}
	return false
}

func ClassCountsOf(
	rows []string,
	tagsByTask map[string][]Tag,
	reserved map[string]int,
) ClassCounts {
	out := ClassCounts{ByClass: make(map[string]int, len(reserved))}
	for _, id := range rows {
		tags := tagsByTask[id]
		classed := false
		for _, tag := range tags {
			name := strings.ToLower(tag.Name)
			if _, ok := reserved[name]; !ok {
				continue
			}
			out.ByClass[name]++
			classed = true
		}
		if !classed {
			out.General++
		}
	}
	return out
}

func RequiredTags(tags []Tag) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, tag := range tags {
		if tag.Required {
			out = append(out, tag)
		}
	}
	return out
}

func MissingTagDeclarations(l limits.Limits, projectID string, tags []Tag) []Tag {
	var missing []Tag
	for _, tag := range tags {
		if !tag.Required {
			continue
		}
		if !l.Declares(projectID, tag.Name, tag.ID) {
			missing = append(missing, tag)
		}
	}
	return missing
}
