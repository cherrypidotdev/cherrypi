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

package realtimegrpc

import (
	"context"
	"errors"
	"time"

	"cherrypi.vladfaust.com/agency/internal/realtime"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
)

const coalesceWindow = 40 * time.Millisecond

type Server struct {
	Bus *realtime.Bus
}

func (s *Server) Watch(
	ctx context.Context,
	_ *agencyv1.WatchRequest,
	stream *connect.ServerStream[agencyv1.WatchResponse],
) error {
	if s.Bus == nil {
		return connect.NewError(
			connect.CodeUnavailable,
			errors.New("realtime stream is not configured"),
		)
	}

	signals, unsubscribe := s.Bus.Subscribe()
	defer unsubscribe()

	if err := writeEvent(stream, agencyv1.RealtimeKind_REALTIME_KIND_SYNC); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case kind, ok := <-signals:
			if !ok {
				return nil
			}

			pending := map[agencyv1.RealtimeKind]struct{}{kind: {}}
			timer := time.NewTimer(coalesceWindow)
		drain:
			for {
				select {
				case next, ok := <-signals:
					if !ok {
						timer.Stop()
						break drain
					}
					pending[next] = struct{}{}
				case <-timer.C:
					break drain
				case <-ctx.Done():
					timer.Stop()
					return nil
				}
			}
			for next := range pending {
				if err := writeEvent(stream, next); err != nil {
					return err
				}
			}
		}
	}
}

func writeEvent(
	stream *connect.ServerStream[agencyv1.WatchResponse],
	kind agencyv1.RealtimeKind,
) error {
	return stream.Send(&agencyv1.WatchResponse{
		Event: &agencyv1.RealtimeEvent{Kind: kind},
	})
}
