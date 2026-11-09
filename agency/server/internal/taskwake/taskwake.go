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

// line to the parent and, once the parent has no incomplete child left, hands

package taskwake

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/tasks"
)

const DefaultMaxConsecutive = 20

func wakeError(cap int) string {
	return fmt.Sprintf(
		"child wake cap reached (%d): the task keeps spawning sub-tasks", cap,
	)
}

type Steer func(ctx context.Context, t tasks.Task, prompt string) error

type Waker struct {
	Tasks *tasks.Service

	Steer Steer

	Wake func()

	MaxConsecutive int
}

func (w *Waker) maxConsecutive() int {
	if w.MaxConsecutive > 0 {
		return w.MaxConsecutive
	}
	return DefaultMaxConsecutive
}

func (w *Waker) OnChildFinished(
	ctx context.Context,
	child tasks.Task,
	outcome tasks.ChildOutcome,
) {
	if w.Tasks == nil || child.ParentTaskID == "" {
		return
	}
	parent, err := w.Tasks.Get(ctx, child.ParentTaskID)
	if err != nil {
		log.Printf("[taskwake] load parent %s: %v", child.ParentTaskID, err)
		return
	}

	switch parent.State {
	case tasks.StateCompleted, tasks.StateCancelled,
		tasks.StateBacklog:
		return
	}

	state, err := w.Tasks.WakeState(ctx, parent.ID)
	if err != nil {
		log.Printf("[taskwake] read wake state %s: %v", parent.ID, err)
		return
	}

	if previous, ok := state.Keys[child.ID]; ok {
		if previous == string(outcome) ||
			(outcome == tasks.ChildCompleted || outcome == tasks.ChildCancelled) {
			return
		}
	}
	if state.Keys == nil {
		state.Keys = map[string]string{}
	}
	state.Keys[child.ID] = string(outcome)
	state.Lines = append(state.Lines, wakeLine(child, outcome))
	if err := w.Tasks.SetWakeState(ctx, parent.ID, state); err != nil {
		log.Printf("[taskwake] store wake state %s: %v", parent.ID, err)
		return
	}

	if err := w.Tasks.ReconcileWaiting(ctx, parent.ID); err != nil {
		log.Printf("[taskwake] reconcile parent %s: %v", parent.ID, err)
	}
	parent, err = w.Tasks.Get(ctx, parent.ID)
	if err != nil {
		log.Printf("[taskwake] reload parent %s: %v", parent.ID, err)
		return
	}
	if parent.PendingChildren > 0 {
		return
	}
	w.deliver(ctx, parent, state)
}

func (w *Waker) OnInputCleared(ctx context.Context, t tasks.Task) {
	state, err := w.Tasks.WakeState(ctx, t.ID)
	if err != nil || len(state.Lines) == 0 {
		return
	}
	w.deliver(ctx, t, state)
}

func (w *Waker) deliver(ctx context.Context, parent tasks.Task, state tasks.WakeState) {
	if len(state.Lines) == 0 {
		return
	}
	summary := strings.Join(state.Lines, "\n")
	if state.Count >= w.maxConsecutive() {
		w.failParent(ctx, parent, state)
		return
	}

	switch parent.State {
	case tasks.StateRunning:
		if w.Steer == nil {
			return
		}
		if err := w.Steer(ctx, parent, summary); err != nil {
			log.Printf("[taskwake] steer %s: %v", parent.ID, err)
			return
		}
	case tasks.StateInput:

		return
	case tasks.StateQueued, tasks.StateWaiting:

		if err := w.Tasks.AppendPendingPrompt(ctx, parent.ID, summary, 0); err != nil {
			log.Printf("[taskwake] append pending prompt %s: %v", parent.ID, err)
			return
		}
	case tasks.StateSettled, tasks.StateFailed:
		if _, err := w.Tasks.RequeueWithPrompt(ctx, parent.ID, summary, 0); err != nil {
			log.Printf("[taskwake] requeue parent %s: %v", parent.ID, err)
			return
		}
	default:
		return
	}

	state.Lines = nil
	state.Count++
	if err := w.Tasks.SetWakeState(ctx, parent.ID, state); err != nil {
		log.Printf("[taskwake] store wake state %s: %v", parent.ID, err)
	}
	if w.Wake != nil {
		w.Wake()
	}
}

func (w *Waker) failParent(ctx context.Context, parent tasks.Task, state tasks.WakeState) {
	state.Lines = nil
	if err := w.Tasks.SetWakeState(ctx, parent.ID, state); err != nil {
		log.Printf("[taskwake] store wake state %s: %v", parent.ID, err)
	}
	msg := wakeError(w.maxConsecutive())
	if err := w.Tasks.FailForWakeCap(ctx, parent.ID, msg); err != nil {
		log.Printf("[taskwake] fail parent %s on wake cap: %v", parent.ID, err)
		return
	}
	log.Printf("[taskwake] parent %s failed: %s", parent.ID, msg)
}

func wakeLine(child tasks.Task, outcome tasks.ChildOutcome) string {
	name := strings.TrimSpace(child.Name)
	if name == "" {
		name = firstLine(child.Prompt)
	}
	label := child.ID
	if name != "" && name != child.ID {
		label = fmt.Sprintf("%s (%s)", child.ID, name)
	}
	switch outcome {
	case tasks.ChildFailed:
		detail := strings.TrimSpace(child.Error)
		if detail == "" {
			detail = "no error details"
		}
		return fmt.Sprintf(
			"Sub-task %s failed: %s — retry, cancel, or reassign it.",
			label, detail,
		)
	case tasks.ChildCompleted:
		return fmt.Sprintf("Sub-task %s completed — no action needed.", label)
	case tasks.ChildCancelled:
		return fmt.Sprintf("Sub-task %s was cancelled — no action needed.", label)
	default:
		return fmt.Sprintf(
			"Sub-task %s settled — review it with getTask %s.",
			label, child.ID,
		)
	}
}

// firstLine returns the first non-empty line of a prompt, trimmed to a short

func firstLine(prompt string) string {
	for _, line := range strings.Split(prompt, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#"))
		if line != "" {
			if len(line) > 80 {
				line = line[:77] + "…"
			}
			return line
		}
	}
	return ""
}
