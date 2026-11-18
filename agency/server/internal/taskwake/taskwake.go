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

// summary line to the blocked task and, once the blocked task has no unresolved

// person, so its line is stashed and the task is not started. A settled blocker

package taskwake

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"cherrypi.vladfaust.com/agency/internal/tasks"
)

const DefaultMaxConsecutive = 20

const DefaultCoalesceWindow = 15 * time.Second

type Steer func(ctx context.Context, t tasks.Task, prompt string) error

type Waker struct {
	Tasks *tasks.Service

	Steer Steer

	Wake func()

	MaxConsecutive int

	CoalesceWindow time.Duration

	Now func() time.Time
}

func (w *Waker) maxConsecutive() int {
	if w.MaxConsecutive > 0 {
		return w.MaxConsecutive
	}
	return DefaultMaxConsecutive
}

func (w *Waker) window() time.Duration {
	if w.CoalesceWindow > 0 {
		return w.CoalesceWindow
	}
	return DefaultCoalesceWindow
}

func (w *Waker) now() int64 {
	if w.Now != nil {
		return w.Now().UnixMilli()
	}
	return time.Now().UnixMilli()
}

func (w *Waker) OnTaskFinished(
	ctx context.Context,
	blocker tasks.Task,
	outcome tasks.FinishOutcome,
) {
	if w.Tasks == nil {
		return
	}

	// reason, so its line is stashed for the next run.
	if outcome == tasks.OutcomeSettled {
		return
	}
	edges, err := w.Tasks.BlockedEdges(ctx, blocker.ID)
	if err != nil {
		log.Printf("[taskwake] list edges blocked by %s: %v", blocker.ID, err)
		return
	}
	resolved := outcome == tasks.OutcomeCompleted ||
		outcome == tasks.OutcomeCancelled
	for _, edge := range edges {
		blocked, err := w.Tasks.Get(ctx, edge.BlockedTaskID)
		if err != nil {
			log.Printf("[taskwake] load blocked %s: %v", edge.BlockedTaskID, err)
			continue
		}

		switch blocked.State {
		case tasks.StateCompleted, tasks.StateCancelled, tasks.StateBacklog:
			continue
		}
		key := wakeKey(blocker.ID)
		state, err := w.Tasks.WakeState(ctx, blocked.ID)
		if err != nil {
			log.Printf("[taskwake] read wake state %s: %v", blocked.ID, err)
			continue
		}

		// must not append the same line twice.
		if previous, ok := state.Keys[key]; ok && previous == string(outcome) {
			continue
		}
		if state.Keys == nil {
			state.Keys = map[string]string{}
		}
		state.Keys[key] = string(outcome)
		state.Lines = append(state.Lines, wakeLine(blocker, outcome))

		if err := w.Tasks.ReconcileWaiting(ctx, blocked.ID); err != nil {
			log.Printf("[taskwake] reconcile blocked %s: %v", blocked.ID, err)
		}
		refreshed, err := w.Tasks.Get(ctx, blocked.ID)
		if err != nil {
			continue
		}
		// Store the line before delivering: a delivery that cannot happen yet

		remaining := len(tasks.IncompleteBlockerIDs(refreshed.References))
		startEarly := resolved && edge.WakeOnResolveWhenMany && remaining > 0
		if startEarly {

			if state.WakeAfter == 0 {
				state.WakeAfter = w.now() + w.window().Milliseconds()
			}
		}
		if err := w.Tasks.SetWakeState(ctx, blocked.ID, state); err != nil {
			log.Printf("[taskwake] store wake state %s: %v", blocked.ID, err)
			continue
		}

		// coalescing window, and a task still parked takes the line on the run

		if remaining == 0 {
			w.deliver(ctx, refreshed, state, deliverSummary)
		}
	}
}

func (w *Waker) OnInputCleared(ctx context.Context, t tasks.Task) {
	state, err := w.Tasks.WakeState(ctx, t.ID)
	if err != nil || len(state.Lines) == 0 {
		return
	}
	w.deliver(ctx, t, state, deliverSummary)
}

func (w *Waker) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.FlushDue(ctx)
		}
	}
}

func (w *Waker) FlushDue(ctx context.Context) {
	if w.Tasks == nil {
		return
	}
	rows, err := w.Tasks.PendingSteeringTasks(ctx)
	if err != nil {
		log.Printf("[taskwake] list pending steering: %v", err)
		return
	}
	now := w.now()
	for _, row := range rows {
		state, ok := tasks.ParseWakeStateRow(row.PendingSteering)
		if !ok || state.WakeAfter == 0 || state.WakeAfter > now {
			continue
		}
		t, err := w.Tasks.Get(ctx, row.ID)
		if err != nil {
			continue
		}
		w.deliver(ctx, t, state, deliverStart)
	}
}

type delivery int

const (
	deliverSummary delivery = iota

	deliverStart
)

func (w *Waker) deliver(
	ctx context.Context,
	task tasks.Task,
	state tasks.WakeState,
	reason delivery,
) {
	if len(state.Lines) == 0 {
		return
	}
	summary := strings.Join(state.Lines, "\n")
	if state.Count >= w.maxConsecutive() {
		w.failTask(ctx, task, state)
		return
	}

	switch task.State {
	case tasks.StateRunning:
		if w.Steer == nil {
			return
		}
		if err := w.Steer(ctx, task, summary); err != nil {
			log.Printf("[taskwake] steer %s: %v", task.ID, err)
			return
		}
	case tasks.StateInput:

		return
	case tasks.StateQueued:

		if err := w.Tasks.AppendPendingPrompt(ctx, task.ID, summary, 0); err != nil {
			log.Printf("[taskwake] append pending prompt %s: %v", task.ID, err)
			return
		}
	case tasks.StateWaiting:

		if reason == deliverStart {
			_, err := w.Tasks.WakeWaitingWithPrompt(ctx, task.ID, summary, 0)
			if err == nil {
				break
			}

			if err != tasks.ErrHeld && err != tasks.ErrWrongState {
				log.Printf("[taskwake] wake waiting task %s: %v", task.ID, err)
				return
			}
		}
		if err := w.Tasks.AppendPendingPrompt(ctx, task.ID, summary, 0); err != nil {
			log.Printf("[taskwake] append pending prompt %s: %v", task.ID, err)
			return
		}
	case tasks.StateSettled, tasks.StateFailed:
		if _, err := w.Tasks.RequeueWithPrompt(ctx, task.ID, summary, 0); err != nil {
			log.Printf("[taskwake] requeue task %s: %v", task.ID, err)
			return
		}
	default:
		return
	}

	state.Lines = nil
	state.WakeAfter = 0
	state.Count++
	if err := w.Tasks.SetWakeState(ctx, task.ID, state); err != nil {
		log.Printf("[taskwake] store wake state %s: %v", task.ID, err)
	}
	if w.Wake != nil {
		w.Wake()
	}
}

func (w *Waker) failTask(ctx context.Context, task tasks.Task, state tasks.WakeState) {
	state.Lines = nil
	state.WakeAfter = 0
	if err := w.Tasks.SetWakeState(ctx, task.ID, state); err != nil {
		log.Printf("[taskwake] store wake state %s: %v", task.ID, err)
	}
	msg := wakeError(w.maxConsecutive())
	if err := w.Tasks.FailForWakeCap(ctx, task.ID, msg); err != nil {
		log.Printf("[taskwake] fail task %s on wake cap: %v", task.ID, err)
		return
	}
	log.Printf("[taskwake] task %s failed: %s", task.ID, msg)
}

func wakeError(cap int) string {
	return "wake cap reached (" + strconv.Itoa(cap) +
		"): the task keeps being woken by its blockers"
}

func wakeKey(blockerID string) string {
	return "block:" + blockerID
}

func wakeLine(blocker tasks.Task, outcome tasks.FinishOutcome) string {
	label := taskLabel(blocker)
	switch outcome {
	case tasks.OutcomeFailed:
		detail := strings.TrimSpace(blocker.Error)
		if detail == "" {
			detail = "no error details"
		}
		return "Blocking task " + label + " failed: " + detail +
			" — it keeps blocking this task; retry it, drop the edge, or ask an operator."
	case tasks.OutcomeCompleted:
		return "Blocking task " + label + " completed — that edge is resolved."
	case tasks.OutcomeCancelled:
		return "Blocking task " + label + " was cancelled — that edge is resolved; " +
			"drop it or point it at the work that replaces it."
	default:
		return "Blocking task " + label + " settled — review it with getTask " +
			blocker.ID + "."
	}
}

// taskLabel names a task the way a wake line refers to it: its id, plus its

func taskLabel(task tasks.Task) string {
	name := strings.TrimSpace(task.Name)
	if name == "" {
		name = firstLine(task.Prompt)
	}
	if name == "" || name == task.ID {
		return task.ID
	}
	return task.ID + " (" + name + ")"
}

// firstLine returns the first non-empty line of a prompt, trimmed to a short

func firstLine(prompt string) string {
	for _, line := range strings.Split(prompt, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#"))
		if line != "" {
			if len(line) > 80 {

				cut := 77
				for cut > 0 && !utf8.RuneStart(line[cut]) {
					cut--
				}
				line = line[:cut] + "…"
			}
			return line
		}
	}
	return ""
}
