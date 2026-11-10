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
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var ErrTickInFlight = errors.New("the previous tick is still running")

type TickClock interface {
	Now() time.Time
	NewTicker(interval time.Duration) TickTicker
}

type TickTicker interface {
	C() <-chan time.Time
	Stop()
}

type realTickClock struct{}

func (realTickClock) Now() time.Time { return time.Now() }

func (realTickClock) NewTicker(interval time.Duration) TickTicker {
	return realTickTicker{ticker: time.NewTicker(interval)}
}

type realTickTicker struct{ ticker *time.Ticker }

func (r realTickTicker) C() <-chan time.Time { return r.ticker.C }
func (r realTickTicker) Stop()               { r.ticker.Stop() }

type tickLoop struct {
	cancel context.CancelFunc
	done   chan struct{}
	wg     sync.WaitGroup
}

func (s *Service) startTicker(p Plugin) {
	if s.Config.DisableTicks {
		s.stopTicker(p.ID)
		return
	}
	interval := p.Manifest.TickInterval()
	if interval <= 0 || !p.Enabled {
		s.stopTicker(p.ID)
		return
	}
	s.stopTicker(p.ID)

	loopCtx, cancel := context.WithCancel(s.serviceContext())
	loop := &tickLoop{cancel: cancel, done: make(chan struct{})}
	s.mu.Lock()
	s.tickers[p.ID] = loop
	s.mu.Unlock()

	go func() {
		defer close(loop.done)
		s.runTickLoop(loopCtx, p, interval, loop)
	}()
}

func (s *Service) stopTicker(id string) {
	s.mu.Lock()
	loop := s.tickers[id]
	delete(s.tickers, id)
	s.mu.Unlock()
	if loop == nil {
		return
	}
	loop.cancel()
	<-loop.done
	loop.wg.Wait()
}

func (s *Service) stopAllTickers() {
	s.mu.Lock()
	ids := make([]string, 0, len(s.tickers))
	for id := range s.tickers {
		ids = append(ids, id)
	}
	s.mu.Unlock()
	for _, id := range ids {
		s.stopTicker(id)
	}
}

func (s *Service) runTickLoop(
	ctx context.Context,
	p Plugin,
	interval time.Duration,
	loop *tickLoop,
) {
	if jitter := s.tickJitter(interval); jitter > 0 {
		timer := time.NewTimer(jitter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
	ticker := s.clock().NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case at := <-ticker.C():
			if !s.beginTick(p.ID) {
				// One line per in-flight tick, not per skipped interval: a

				if s.noteSkipped(p.ID) {
					_ = s.Store.LogRuntime(
						ctx,
						p.ID,
						"",
						LevelWarn,
						"previous tick is still running; skipped its interval",
						nil,
						nil,
					)
				}
				continue
			}
			loop.wg.Add(1)
			go func(at time.Time) {
				defer loop.wg.Done()
				defer s.endTick(p.ID)
				s.runTickSafely(ctx, p, at)
			}(at)
		}
	}
}

func (s *Service) runTickSafely(ctx context.Context, p Plugin, at time.Time) {
	s.recoverTick(ctx, p.ID, func() {
		s.runTick(ctx, p, at)
	})
}

func (s *Service) recoverTick(ctx context.Context, id string, run func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.Store.Log(
				ctx,
				id,
				"error",
				"tick",
				"",
				"tick panicked",
				map[string]any{"panic": fmt.Sprint(recovered)},
			)
		}
	}()
	run()
}

func (s *Service) runTick(ctx context.Context, p Plugin, at time.Time) {
	current, err := s.Store.Get(ctx, p.ID)
	if err != nil {
		return
	}
	if !current.Enabled || current.Manifest.TickInterval() <= 0 {
		return
	}
	_ = s.invokeTick(ctx, current, at.UnixMilli())
}

func (s *Service) invokeTick(ctx context.Context, p Plugin, nowMs int64) error {
	if nowMs == 0 {
		nowMs = s.clock().Now().UnixMilli()
	}
	inv := Invocation{
		ID:     newInvocationID(),
		Kind:   HandlerTick,
		Caller: Caller{Kind: "agency"},
		NowMs:  nowMs,
	}
	timeout := defaultTimeout
	if p.Manifest.Limits.TimeoutSeconds > 0 {
		timeout = time.Duration(p.Manifest.Limits.TimeoutSeconds) * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := s.Invoke(callCtx, p, inv)
	if err != nil {
		s.Store.Log(
			ctx,
			p.ID,
			"error",
			"tick",
			inv.ID,
			"tick failed",
			map[string]any{"error": err.Error()},
		)
	}
	return err
}

func (s *Service) InvokeTick(ctx context.Context, id string, nowMs int64) error {
	p, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if p.Manifest.TickInterval() <= 0 {
		return errors.New("plugin has no schedule")
	}
	if !p.Enabled {
		return errors.New("plugin is disabled")
	}
	if !s.beginTick(id) {
		return ErrTickInFlight
	}
	defer s.endTick(id)
	return s.invokeTick(ctx, p, nowMs)
}

func (s *Service) beginTick(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tickBusy[id] {
		return false
	}
	s.tickBusy[id] = true
	s.tickSkipLogged[id] = false
	return true
}

func (s *Service) endTick(id string) {
	s.mu.Lock()
	delete(s.tickBusy, id)
	delete(s.tickSkipLogged, id)
	s.mu.Unlock()
}

func (s *Service) noteSkipped(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tickBusy[id] || s.tickSkipLogged[id] {
		return false
	}
	s.tickSkipLogged[id] = true
	return true
}

func (s *Service) clock() TickClock {
	if s.Clock != nil {
		return s.Clock
	}
	return realTickClock{}
}

func (s *Service) tickJitter(interval time.Duration) time.Duration {
	if s.Jitter != nil {
		return s.Jitter(interval)
	}
	if interval <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(interval)))
}

func (s *Service) serviceContext() context.Context {
	ctx, _ := s.root()
	return ctx
}

func (s *Service) root() (context.Context, context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rootCtx == nil {
		s.rootCtx, s.rootCancel = context.WithCancel(context.Background())
	}
	return s.rootCtx, s.rootCancel
}
