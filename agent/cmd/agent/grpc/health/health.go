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

package healthgrpc

import (
	"context"
	"errors"
	"time"

	agenthealth "cherrypi.vladfaust.com/agent/internal/health"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"connectrpc.com/connect"
)

type HealthServer struct {
	Sampler *agenthealth.Sampler

	Default time.Duration
}

func (s *HealthServer) SetHealthInterval(
	_ context.Context,
	req *agentv1.SetHealthIntervalRequest,
) (*agentv1.SetHealthIntervalResponse, error) {
	if s.Sampler == nil {
		return nil, connect.NewError(
			connect.CodeUnavailable,
			errors.New("health sampling is not configured"),
		)
	}

	interval := s.Default
	inherited := true
	if req.IntervalSeconds != nil {
		seconds := *req.IntervalSeconds
		if seconds < 0 {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("interval_seconds must not be negative"),
			)
		}
		interval = time.Duration(seconds) * time.Second
		inherited = false
	}
	if interval < 0 {
		interval = 0
	}

	s.Sampler.SetInterval(interval)
	return &agentv1.SetHealthIntervalResponse{
		IntervalSeconds: int64(interval / time.Second),
		Inherited:       inherited,
	}, nil
}
