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

package store

import (
	"context"

	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

func (s *PgStore) InsertAgentHealthSample(
	ctx context.Context,
	p InsertAgentHealthSampleParams,
	before int64,
) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.InsertAgentHealthSample(
			ctx,
			pgdb.InsertAgentHealthSampleParams(p),
		); err != nil {
			return err
		}
		return q.PruneAgentHealthSamples(
			ctx,
			pgdb.PruneAgentHealthSamplesParams(PruneAgentHealthSamplesParams{
				AgentID: p.AgentID,
				Before:  before,
			}),
		)
	})
}

func (s *PgStore) GetLatestAgentHealthSample(
	ctx context.Context,
	agentID ids.AgentID,
) (AgentHealthSample, error) {
	row, err := s.q.GetLatestAgentHealthSample(ctx, int64(agentID))
	if err != nil {
		return AgentHealthSample{}, err
	}
	return fromPgAgentHealthSample(row), nil
}

func (s *PgStore) ListAgentHealthSamples(
	ctx context.Context,
	p ListAgentHealthSamplesParams,
) ([]AgentHealthSample, error) {
	rows, err := s.q.ListAgentHealthSamples(ctx, pgdb.ListAgentHealthSamplesParams(p))
	if err != nil {
		return nil, err
	}
	return fromPgAgentHealthSampleSlice(rows), nil
}

func (s *PgStore) DeleteAgentHealthSamples(
	ctx context.Context,
	agentID ids.AgentID,
) error {
	return s.q.DeleteAgentHealthSamples(ctx, int64(agentID))
}
