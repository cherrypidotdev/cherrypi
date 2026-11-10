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
	"sort"
	"time"

	"cherrypi.vladfaust.com/agency/internal/realtime"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
)

const coalesceWindow = 40 * time.Millisecond

const maxPushedTasks = 8

type Server struct {
	Bus *realtime.Bus

	Task func(ctx context.Context, id string) *agencyv1.Task
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

	if err := writeEvent(stream, &agencyv1.RealtimeEvent{
		Kind: agencyv1.RealtimeKind_REALTIME_KIND_SYNC,
	}); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case signal, ok := <-signals:
			if !ok {
				return nil
			}

			pending := map[agencyv1.RealtimeKind]map[string]struct{}{
				signal.Kind: {signal.TaskID: {}},
			}
			timer := time.NewTimer(coalesceWindow)
		drain:
			for {
				select {
				case next, ok := <-signals:
					if !ok {
						timer.Stop()
						break drain
					}
					ids, seen := pending[next.Kind]
					if !seen {
						ids = map[string]struct{}{}
						pending[next.Kind] = ids
					}
					ids[next.TaskID] = struct{}{}
				case <-timer.C:
					break drain
				case <-ctx.Done():
					timer.Stop()
					return nil
				}
			}
			for kind, ids := range pending {
				for _, event := range s.events(ctx, kind, ids) {
					if err := writeEvent(stream, event); err != nil {
						return err
					}
				}
			}
		}
	}
}

func (s *Server) events(
	ctx context.Context,
	kind agencyv1.RealtimeKind,
	ids map[string]struct{},
) []*agencyv1.RealtimeEvent {
	bare := func() []*agencyv1.RealtimeEvent {
		return []*agencyv1.RealtimeEvent{{Kind: kind}}
	}
	if kind != agencyv1.RealtimeKind_REALTIME_KIND_TASKS || s.Task == nil {
		return bare()
	}
	if _, ok := ids[""]; ok || len(ids) > maxPushedTasks {
		return bare()
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	out := make([]*agencyv1.RealtimeEvent, 0, len(ordered))
	for _, id := range ordered {
		out = append(out, &agencyv1.RealtimeEvent{
			Kind: kind,
			Task: s.Task(ctx, id),
		})
	}
	return out
}

func writeEvent(
	stream *connect.ServerStream[agencyv1.WatchResponse],
	event *agencyv1.RealtimeEvent,
) error {
	return stream.Send(&agencyv1.WatchResponse{Event: event})
}
