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
	"log"
	"sync"
	"time"
)

type Event struct {
	Name      string
	TaskID    string
	ProjectID string
	State     string

	Extra map[string]any

	IncludeDisabled string
}

var ErrEventQueueFull = errors.New("event queue full; dropped")

type eventJob struct {
	plugin Plugin
	inv    Invocation

	batch *eventBatch
}

type eventBatch struct {
	mu       sync.Mutex
	pending  int
	firstErr error
	done     chan struct{}
}

func newEventBatch(pending int) *eventBatch {
	batch := &eventBatch{pending: pending, done: make(chan struct{})}
	if pending == 0 {
		close(batch.done)
	}
	return batch
}

func (b *eventBatch) complete(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil && b.firstErr == nil {
		b.firstErr = err
	}
	b.pending--
	if b.pending == 0 {
		close(b.done)
	}
}

func (b *eventBatch) wait(ctx context.Context) error {
	select {
	case <-b.done:
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.firstErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) DispatchEvent(ctx context.Context, ev Event) {
	_, _ = s.dispatchEvent(ctx, ev, false)
}

func (s *Service) DispatchEventWait(ctx context.Context, ev Event) (int, error) {
	return s.dispatchEvent(ctx, ev, true)
}

func (s *Service) dispatchEvent(ctx context.Context, ev Event, wait bool) (int, error) {
	if s == nil || !s.Store.ready() || ev.Name == "" {
		return 0, nil
	}
	s.Start(ctx)

	payload := map[string]any{
		"event":      ev.Name,
		"task_id":    ev.TaskID,
		"project_id": ev.ProjectID,
		"state":      ev.State,
		"at":         time.Now().UnixMilli(),
	}
	for key, value := range ev.Extra {
		payload[key] = value
	}
	if ev.TaskID != "" && s.Tasks != nil {
		t, err := s.Tasks.Get(ctx, ev.TaskID)
		if err == nil {
			if payload["project_id"] == "" {
				payload["project_id"] = t.ProjectID
			}
			if payload["state"] == "" {
				payload["state"] = string(t.State)
			}
			payload["task"] = taskMap(t)
		}
	}

	list, err := s.Store.List(ctx)
	if err != nil {
		log.Printf("[plugins] list for event %s: %v", ev.Name, err)
		return 0, err
	}
	jobs := make([]eventJob, 0, len(list))
	for _, p := range list {
		if !p.Enabled && p.ID != ev.IncludeDisabled {
			continue
		}
		if !p.Manifest.SubscribesTo(ev.Name) {
			continue
		}
		inv := Invocation{
			ID:     newInvocationID(),
			Kind:   HandlerEvent,
			Caller: Caller{Kind: "agency"},
			TaskID: ev.TaskID,
			Event:  ev.Name,
			Body:   payload,
		}
		if !p.Enabled {

			inv.allowDisabled = true
		}
		jobs = append(jobs, eventJob{plugin: p, inv: inv})
	}
	var batch *eventBatch
	if wait {
		batch = newEventBatch(len(jobs))
		for i := range jobs {
			jobs[i].batch = batch
		}
	}
	delivered := 0
	for _, job := range jobs {
		select {
		case s.events <- job:
			delivered++
		default:
			s.Store.Log(ctx, job.plugin.ID, "warn", "event", "", "event queue full; dropped",
				map[string]any{"event": ev.Name, "task_id": ev.TaskID})
			if job.batch != nil {
				job.batch.complete(ErrEventQueueFull)
			}
		}
	}
	if batch != nil {
		return delivered, batch.wait(ctx)
	}
	return delivered, nil
}

func (s *Service) eventWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-s.events:
			callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, err := s.Invoke(callCtx, job.plugin, job.inv)
			cancel()
			if err != nil {
				s.Store.Log(ctx, job.plugin.ID, "error", "event", job.inv.ID,
					"event delivery failed",
					map[string]any{"event": job.inv.Event, "error": err.Error()})
				log.Printf("[plugins] %s event %s: %v", job.plugin.ID, job.inv.Event, err)
			}
			if job.batch != nil {
				job.batch.complete(err)
			}
		}
	}
}
