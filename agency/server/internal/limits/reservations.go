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
	"context"
	"errors"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

var ErrInvalidReservation = errors.New("reserved concurrency must not be negative")

type Reservation struct {
	TagName  string
	Reserved int
}

func (s *Service) ProjectReservations(
	ctx context.Context,
	projectID string,
) ([]Reservation, error) {
	rows, err := s.store.ListProjectTagReservations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]Reservation, 0, len(rows))
	for _, row := range rows {
		out = append(out, Reservation{
			TagName:  row.TagName,
			Reserved: int(row.Reserved),
		})
	}
	return out, nil
}

func (s *Service) ProjectReservationMap(
	ctx context.Context,
	projectID string,
) (map[string]int, error) {
	list, err := s.ProjectReservations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return reservationMap(list), nil
}

func (s *Service) UpdateProjectReservations(
	ctx context.Context,
	projectID string,
	in []Reservation,
) ([]Reservation, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("project is required")
	}
	merged := map[string]int{}
	var order []string
	for _, res := range in {
		name := strings.ToLower(strings.TrimSpace(res.TagName))
		if res.Reserved < 0 {
			return nil, ErrInvalidReservation
		}
		if name == "" {
			continue
		}
		if _, seen := merged[name]; !seen {
			order = append(order, name)
		}
		if res.Reserved > merged[name] {
			merged[name] = res.Reserved
		}
	}

	now := time.Now().UnixMilli()
	rows := make([]store.InsertProjectTagReservationParams, 0, len(order))
	for _, name := range order {
		if merged[name] == 0 {
			continue
		}
		rows = append(rows, store.InsertProjectTagReservationParams{
			ProjectID: projectID,
			TagName:   name,
			Reserved:  int64(merged[name]),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	if err := s.store.ReplaceProjectTagReservations(ctx, projectID, rows); err != nil {
		return nil, err
	}
	return s.ProjectReservations(ctx, projectID)
}

func (s *Service) AgentReservations(
	ctx context.Context,
	agentID ids.AgentID,
) (map[string]int, error) {
	rows, err := s.store.ListAgentProjectTagReservations(ctx, agentID)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, row := range rows {
		name := strings.ToLower(row.TagName)
		if int(row.Reserved) > out[name] {
			out[name] = int(row.Reserved)
		}
	}
	return out, nil
}

func reservationMap(list []Reservation) map[string]int {
	out := make(map[string]int, len(list))
	for _, res := range list {
		if res.Reserved <= 0 || res.TagName == "" {
			continue
		}
		out[strings.ToLower(res.TagName)] = res.Reserved
	}
	return out
}
