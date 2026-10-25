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

package limits

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"database/sql"
	"errors"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/store"
)

var ErrInvalidLimit = errors.New("concurrency limit must not be negative")

const (
	DefaultMaxConcurrent           = 1
	DefaultMaxConcurrentPerProject = 1
)

type Limits struct {
	MaxConcurrent           int
	MaxConcurrentPerProject int

	Tags map[string]int

	Projects map[string]int

	ProjectTags map[string]map[string]int
}

type Service struct {
	store store.Store
}

func New(st store.Store) *Service { return &Service{store: st} }

func (s *Service) Get(ctx context.Context, agentID ids.AgentID) (Limits, error) {
	out := Limits{
		MaxConcurrent:           DefaultMaxConcurrent,
		MaxConcurrentPerProject: DefaultMaxConcurrentPerProject,
		Tags:                    map[string]int{},
		Projects:                map[string]int{},
		ProjectTags:             map[string]map[string]int{},
	}

	row, err := s.store.GetAgentTaskLimits(ctx, agentID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Limits{}, err
	}
	if err == nil {
		out.MaxConcurrent = int(row.MaxConcurrent)
		out.MaxConcurrentPerProject = int(row.MaxConcurrentPerProject)
	}

	tagRows, err := s.store.ListAgentTagLimits(ctx, agentID)
	if err != nil {
		return Limits{}, err
	}
	for _, r := range tagRows {
		out.Tags[strings.ToLower(r.TagName)] = int(r.MaxConcurrent)
	}

	projectRows, err := s.store.ListAgentProjectLimits(ctx, agentID)
	if err != nil {
		return Limits{}, err
	}
	for _, r := range projectRows {
		out.Projects[r.ProjectID] = int(r.MaxConcurrent)
	}

	projectTagRows, err := s.store.ListAgentProjectTagLimits(ctx, agentID)
	if err != nil {
		return Limits{}, err
	}
	for _, r := range projectTagRows {
		if out.ProjectTags[r.ProjectID] == nil {
			out.ProjectTags[r.ProjectID] = map[string]int{}
		}
		out.ProjectTags[r.ProjectID][r.TagID] = int(r.MaxConcurrent)
	}

	return out, nil
}

func (s *Service) Update(ctx context.Context, agentID ids.AgentID, in Limits) (Limits, error) {
	if in.MaxConcurrent < 0 || in.MaxConcurrentPerProject < 0 {
		return Limits{}, ErrInvalidLimit
	}
	for _, cap := range in.Tags {
		if cap < 0 {
			return Limits{}, ErrInvalidLimit
		}
	}
	for _, cap := range in.Projects {
		if cap < 0 {
			return Limits{}, ErrInvalidLimit
		}
	}
	for _, byTag := range in.ProjectTags {
		for _, cap := range byTag {
			if cap < 0 {
				return Limits{}, ErrInvalidLimit
			}
		}
	}

	if err := s.store.UpsertAgentTaskLimits(ctx, store.UpsertAgentTaskLimitsParams{
		AgentID:                 int64(agentID),
		MaxConcurrent:           int64(in.MaxConcurrent),
		MaxConcurrentPerProject: int64(in.MaxConcurrentPerProject),
	}); err != nil {
		return Limits{}, err
	}

	if err := s.store.DeleteAgentTagLimits(ctx, agentID); err != nil {
		return Limits{}, err
	}
	for tagName, cap := range in.Tags {
		name := strings.ToLower(strings.TrimSpace(tagName))
		if name == "" {
			continue
		}
		if err := s.store.InsertAgentTagLimit(ctx, store.InsertAgentTagLimitParams{
			AgentID:       int64(agentID),
			TagName:       name,
			MaxConcurrent: int64(cap),
		}); err != nil {
			return Limits{}, err
		}
	}

	if err := s.store.DeleteAgentProjectLimits(ctx, agentID); err != nil {
		return Limits{}, err
	}
	for projectID, cap := range in.Projects {
		if err := s.store.InsertAgentProjectLimit(ctx, store.InsertAgentProjectLimitParams{
			AgentID:       int64(agentID),
			ProjectID:     projectID,
			MaxConcurrent: int64(cap),
		}); err != nil {
			return Limits{}, err
		}
	}

	if err := s.store.DeleteAgentProjectTagLimits(ctx, agentID); err != nil {
		return Limits{}, err
	}
	for projectID, byTag := range in.ProjectTags {
		for tagID, cap := range byTag {
			if err := s.store.InsertAgentProjectTagLimit(
				ctx,
				store.InsertAgentProjectTagLimitParams{
					AgentID:       int64(agentID),
					ProjectID:     projectID,
					TagID:         tagID,
					MaxConcurrent: int64(cap),
				},
			); err != nil {
				return Limits{}, err
			}
		}
	}

	return s.Get(ctx, agentID)
}
