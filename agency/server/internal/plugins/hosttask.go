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
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/tasks"

	lua "github.com/yuin/gopher-lua"
)

func (h *Host) taskForWrite(taskID string) (tasks.Task, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return tasks.Task{}, errors.New("task_id is required")
	}
	t, err := h.Bridge.Task(h.Ctx, taskID)
	if err != nil {
		return tasks.Task{}, err
	}
	allowed, err := h.taskAllowed(t)
	if err != nil {
		return tasks.Task{}, err
	}
	if !allowed {
		return tasks.Task{}, errors.New("the plugin may not act on this task")
	}
	return t, nil
}

func (h *Host) addTaskMessage(taskID, text string) error {
	if _, err := h.taskForWrite(taskID); err != nil {
		return err
	}
	return h.Bridge.AddTaskMessage(h.Ctx, strings.TrimSpace(taskID), text)
}

func (h *Host) closeTask(taskID, outcome string) error {
	if _, err := h.taskForWrite(taskID); err != nil {
		return err
	}
	var resolution tasks.Resolution
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "completed", "complete", "done":
		resolution = tasks.ResolutionCompleted
	case "cancelled", "canceled", "cancel":
		resolution = tasks.ResolutionCancelled
	default:
		return fmt.Errorf(
			"unknown outcome %q; use \"completed\" or \"cancelled\"", outcome,
		)
	}
	return h.Bridge.CloseTask(h.Ctx, strings.TrimSpace(taskID), resolution)
}

func (h *Host) setTaskBadges(taskID string, value lua.LValue) error {
	if _, err := h.taskForWrite(taskID); err != nil {
		return err
	}
	if value == nil || value == lua.LNil {
		return h.Bridge.SetTaskBadges(h.Ctx, strings.TrimSpace(taskID), nil)
	}
	raw, err := json.Marshal(fromLua(value))
	if err != nil {
		return err
	}
	var badges []TaskBadge
	if err := json.Unmarshal(raw, &badges); err != nil {
		return errors.New(
			"badges must be a list of {key, label, value, tone, url} objects",
		)
	}
	return h.Bridge.SetTaskBadges(h.Ctx, strings.TrimSpace(taskID), badges)
}

func (h *Host) holdTask(taskID string, opts *lua.LTable) error {
	if _, err := h.taskForWrite(taskID); err != nil {
		return err
	}
	key := stringField(opts, "key")
	if key == "" {
		return errors.New("hold_task needs a key")
	}
	return h.Bridge.HoldTask(h.Ctx, strings.TrimSpace(taskID), HoldInput{
		Key:    key,
		Reason: stringField(opts, "reason"),
	})
}

func (h *Host) releaseTask(taskID, key string, opts *lua.LTable) error {
	if _, err := h.taskForWrite(taskID); err != nil {
		return err
	}
	return h.Bridge.ReleaseTask(h.Ctx, strings.TrimSpace(taskID), key, ReleaseOptions{
		Mode:   stringOpt(opts, "mode", "settle"),
		Prompt: stringOpt(opts, "prompt", ""),
	})
}

func (h *Host) listTaskHolds(taskID string) ([]map[string]any, error) {
	t, err := h.Bridge.Task(h.Ctx, taskID)
	if err != nil {
		return nil, err
	}
	allowed, err := h.taskAllowed(t)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("the plugin may not read this task")
	}
	holds, err := h.Bridge.TaskHolds(h.Ctx, t.ID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(holds))
	for i, hold := range holds {
		out[i] = map[string]any{
			"plugin":      hold.PluginID,
			"plugin_name": hold.PluginName,
			"key":         hold.Key,
			"reason":      hold.Reason,
			"created_at":  hold.CreatedAt.UnixMilli(),
		}
	}
	return out, nil
}
