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
	"strings"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

type HoldInput struct {
	Key    string
	Reason string
}

type ReleaseOptions struct {
	Mode   string
	Prompt string
}

func (s *Service) HoldTask(
	ctx context.Context,
	pluginID, taskID string,
	in HoldInput,
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
	return s.Tasks.Hold(ctx, taskID, p.ID, tasks.HoldInput{
		Key:    in.Key,
		Reason: in.Reason,
	})
}

func (s *Service) ReleaseTask(
	ctx context.Context,
	pluginID, taskID, key string,
	opts ReleaseOptions,
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
	return s.Tasks.Release(ctx, taskID, p.ID, key, tasks.ReleaseOptions{
		Mode:    tasks.ReleaseMode(strings.ToLower(strings.TrimSpace(opts.Mode))),
		Prompt:  opts.Prompt,
		ActorID: ids.ActorID(p.ActorID),
	})
}

func (s *Service) TaskHolds(ctx context.Context, taskID string) ([]tasks.TaskHold, error) {
	if s.Tasks == nil {
		return nil, errors.New("tasks are unavailable")
	}
	return s.Tasks.Holds(ctx, taskID)
}

func (s *Service) releasePluginHolds(ctx context.Context, pluginID string) error {
	if s.Tasks == nil {
		return nil
	}
	releases, err := s.Tasks.ReleasePluginHolds(ctx, pluginID)
	for _, release := range releases {
		ev := HoldReleaseEvent(release)

		ev.IncludeDisabled = pluginID
		if _, derr := s.DispatchEventWait(ctx, ev); derr != nil {
			log.Printf(
				"[plugins] announce hold release %s/%s on %s: %v",
				release.PluginID, release.Key, release.TaskID, derr,
			)
		}
	}
	return err
}

func HoldReleaseEvent(release tasks.HoldRelease) Event {
	return Event{
		Name:   "task_hold_released",
		TaskID: release.TaskID,
		Extra: map[string]any{
			"plugin":      release.PluginID,
			"plugin_name": release.PluginName,
			"key":         release.Key,
			"reason":      release.Reason,
			"mode":        string(release.Mode),
			"applied":     release.Applied,
			"cause":       string(release.Cause),
		},
	}
}
