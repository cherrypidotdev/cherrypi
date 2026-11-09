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
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

const maxPluginWakes = 20

func wakeCapError(plugin string, limit int) string {
	return fmt.Sprintf(
		"plugin wake cap reached (%d): %s keeps waking this task",
		limit, plugin,
	)
}

func (s *Service) AddTaskMessage(
	ctx context.Context,
	pluginID, taskID, text string,
) error {
	p, err := s.Get(ctx, pluginID)
	if err != nil {
		return err
	}
	if !p.Enabled {
		return errors.New("plugin is disabled")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("the message is empty")
	}
	if s.Tasks == nil {
		return errors.New("tasks are unavailable")
	}
	t, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return err
	}

	state, err := s.Tasks.WakeState(ctx, t.ID)
	if err != nil {
		return err
	}
	if state.PluginWakes[p.ID] >= maxPluginWakes {
		message := wakeCapError(p.Name, maxPluginWakes)
		if err := s.Tasks.FailForWakeCap(ctx, t.ID, message); err != nil {
			return err
		}
		return errors.New(message)
	}
	actorID := ids.ActorID(p.ActorID)
	if s.FollowUp != nil {
		if err := s.FollowUp(ctx, t, text, actorID); err != nil {
			return err
		}
	} else {
		switch t.State {
		case tasks.StateSettled, tasks.StateFailed,
			tasks.StateCompleted, tasks.StateCancelled:
			if _, err := s.Tasks.RequeueWithPrompt(ctx, t.ID, text, actorID); err != nil {
				return err
			}
		case tasks.StateQueued, tasks.StateWaiting:
			if err := s.Tasks.AppendPendingPrompt(ctx, t.ID, text, actorID); err != nil {
				return err
			}
		case tasks.StateInput:
			return errors.New(
				"the task is waiting for input; answer the dialog first",
			)
		default:
			return fmt.Errorf(
				"a task in state %q does not accept a follow-up", t.State,
			)
		}
	}
	if state.PluginWakes == nil {
		state.PluginWakes = map[string]int{}
	}
	state.PluginWakes[p.ID]++
	if err := s.Tasks.SetWakeState(ctx, t.ID, state); err != nil {
		return err
	}
	if s.RecordMessage != nil {
		s.RecordMessage(ctx, t.ID, newInvocationID(), text, actorID, time.Now().UnixMilli())
	}
	return nil
}

func (s *Service) CloseTask(
	ctx context.Context,
	pluginID, taskID string,
	outcome tasks.Resolution,
) error {
	p, err := s.Get(ctx, pluginID)
	if err != nil {
		return err
	}
	if !p.Enabled {
		return errors.New("plugin is disabled")
	}
	if s.Tasks == nil {
		return errors.New("tasks are unavailable")
	}
	ctx = tasks.WithActor(ctx, tasks.ActorPlugin)
	ctx = tasks.WithActorID(ctx, ids.ActorID(p.ActorID))
	switch outcome {
	case tasks.ResolutionCompleted:
		_, err = s.Tasks.Complete(ctx, taskID)
	case tasks.ResolutionCancelled:
		_, err = s.Tasks.Cancel(ctx, taskID)
	default:
		return fmt.Errorf("unknown close-out outcome %q", outcome)
	}
	if errors.Is(err, tasks.ErrRunning) {
		return errors.New("the task is running; close it out once it settles")
	}
	if errors.Is(err, tasks.ErrPendingPrompt) {
		return errors.New(
			"the task has an undelivered follow-up; " +
				"close it out once it has run",
		)
	}
	return err
}
