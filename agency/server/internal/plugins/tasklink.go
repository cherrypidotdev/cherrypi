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
	"log"
	"slices"
	"sort"
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

type AddMessageOptions struct {
	RequiredTools []string

	Force bool
}

func (s *Service) AddTaskMessage(
	ctx context.Context,
	pluginID, taskID, text string,
	opts AddMessageOptions,
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

	requirementChanged := false
	previousRequirement := t.RequiredTools
	attemptedRequirement := t.RequiredTools
	if opts.RequiredTools != nil {
		names, err := s.attachedToolNames(ctx, t)
		if err != nil {
			return err
		}
		for _, tool := range opts.RequiredTools {
			if !names[tool] {
				return fmt.Errorf(
					"required tool %q is not attached to task %s; it can call: %s",
					tool, t.ID, toolList(names),
				)
			}
		}
		t, err = s.Tasks.SetRequiredTools(ctx, t.ID, opts.RequiredTools)
		if err != nil {
			return err
		}
		requirementChanged = true
		attemptedRequirement = t.RequiredTools
		s.logRequiredTools(ctx, p, t.ID, previousRequirement, t.RequiredTools)
	}
	actorID := ids.ActorID(p.ActorID)
	if err := s.deliverFollowUp(ctx, t, text, actorID, opts.Force); err != nil {
		s.rollBackRequirement(
			ctx, p, t.ID, requirementChanged,
			attemptedRequirement, previousRequirement,
		)
		return err
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

func (s *Service) deliverFollowUp(
	ctx context.Context,
	t tasks.Task,
	text string,
	actorID ids.ActorID,
	force bool,
) error {
	if s.FollowUp != nil {
		return s.FollowUp(ctx, t, text, actorID, force)
	}
	_, err := s.Tasks.FollowUp(ctx, tasks.FollowUpParams{
		TaskID:  t.ID,
		Text:    text,
		ActorID: actorID,
		Force:   force,
	})
	return err
}

func (s *Service) rollBackRequirement(
	ctx context.Context,
	p Plugin,
	taskID string,
	changed bool,
	attempted, previous []string,
) {
	if !changed {
		return
	}
	if _, err := s.Tasks.SetRequiredTools(ctx, taskID, previous); err != nil {
		log.Printf(
			"[plugins] restore required tools on %s: %v", taskID, err,
		)
		return
	}

	s.logRequiredTools(ctx, p, taskID, attempted, previous)
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

func (s *Service) attachedToolNames(
	ctx context.Context,
	t tasks.Task,
) (map[string]bool, error) {
	names := map[string]bool{}
	if row, ok := s.TaskPlugin(ctx, t.ID); ok {
		for _, tool := range row.Tools {
			names[tool.Name] = true
		}
	}
	for _, tool := range s.ProjectToolsForTask(ctx, t.ID, t.ProjectID) {
		names[tool.Tool.Name] = true
	}
	return names, nil
}

func toolList(names map[string]bool) string {
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	sort.Strings(list)
	if len(list) == 0 {
		return "no tools"
	}
	return strings.Join(list, ", ")
}

func (s *Service) logRequiredTools(
	ctx context.Context,
	p Plugin,
	taskID string,
	before, after []string,
) {
	if s.Tasks == nil || slices.Equal(before, after) {
		return
	}
	logCtx := tasks.WithActor(ctx, tasks.ActorPlugin)
	logCtx = tasks.WithActorID(logCtx, ids.ActorID(p.ActorID))
	message := "Cleared the required tools of " + p.Name
	if len(after) > 0 {
		message = "Required " + p.Name + " tools: " + strings.Join(after, ", ")
	}
	s.Tasks.AppendLog(
		logCtx, taskID, tasks.LogPlugin, "required_tools",
		strings.Join(before, ", "), strings.Join(after, ", "),
		message, map[string]any{"plugin_id": p.ID, "plugin": p.Name, "tools": after},
	)
}
