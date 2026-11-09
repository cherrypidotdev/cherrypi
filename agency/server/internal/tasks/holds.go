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

package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	MaxHoldsPerPlugin = 8

	MaxHoldReasonLength = 200
)

var holdKeyPattern = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

var (
	ErrHeld = errors.New("task is held")

	ErrInvalidHoldKey = errors.New(
		"hold key must be 1-32 characters of a-z, 0-9, '_' or '-'",
	)

	ErrHoldReasonTooLong = fmt.Errorf(
		"hold reason must be at most %d characters", MaxHoldReasonLength,
	)

	ErrHoldLimit = fmt.Errorf(
		"a plugin may hold a task for at most %d conditions", MaxHoldsPerPlugin,
	)

	ErrInvalidReleaseMode = errors.New(
		`release mode must be "settle", "queue", or "complete"`,
	)
)

type TaskHold struct {
	TaskID     string
	PluginID   string
	PluginName string
	Key        string
	Reason     string
	CreatedAt  time.Time
}

func (h TaskHold) Summary() string {
	name := h.PluginName
	if name == "" {
		name = h.PluginID
	}
	if h.Reason == "" {
		return name
	}
	return name + ": " + h.Reason
}

type HoldInput struct {
	Key    string
	Reason string
}

type ReleaseMode string

const (
	ReleaseSettle ReleaseMode = "settle"

	ReleaseQueue ReleaseMode = "queue"

	ReleaseComplete ReleaseMode = "complete"
)

type ReleaseOptions struct {
	Mode ReleaseMode

	Prompt string

	ActorID ids.ActorID
}

type HoldReleaseCause string

const (
	HoldReleasedByOperator HoldReleaseCause = "operator"

	HoldClearedByCloseOut HoldReleaseCause = "closed_out"

	HoldClearedByDequeue HoldReleaseCause = "dequeued"

	HoldClearedByArchive HoldReleaseCause = "archived"

	HoldReleasedByPluginDisable HoldReleaseCause = "plugin_disabled"
)

type HoldRelease struct {
	TaskID     string
	PluginID   string
	PluginName string
	Key        string

	Reason string

	Mode ReleaseMode

	Applied bool
	Cause   HoldReleaseCause
}

func (s *Service) Holds(ctx context.Context, taskID string) ([]TaskHold, error) {
	rows, err := s.store.ListTaskHolds(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return toTaskHolds(rows), nil
}

func (s *Service) Hold(ctx context.Context, taskID, pluginID string, in HoldInput) error {
	key, err := normalizeHoldKey(in.Key)
	if err != nil {
		return err
	}
	reason, err := normalizeHoldReason(in.Reason)
	if err != nil {
		return err
	}
	if strings.TrimSpace(pluginID) == "" {
		return errors.New("plugin_id is required")
	}
	before, err := s.snapshot(ctx, taskID)
	if err != nil {
		return err
	}
	switch {
	case before.Archived:
		return fmt.Errorf("%w: task is archived", ErrWrongState)
	case before.State == StateCompleted, before.State == StateCancelled:
		return fmt.Errorf("%w: task is %s", ErrWrongState, before.State)
	}

	held, err := s.Holds(ctx, taskID)
	if err != nil {
		return err
	}
	mine := 0
	existing := false
	for _, hold := range held {
		if hold.PluginID != pluginID {
			continue
		}
		mine++
		if hold.Key == key {
			existing = true
		}
	}
	if !existing && mine >= MaxHoldsPerPlugin {
		return ErrHoldLimit
	}
	now := time.Now().UnixMilli()
	if err := s.store.UpsertTaskHold(ctx, store.UpsertTaskHoldParams{
		TaskID:    taskID,
		PluginID:  pluginID,
		Key:       key,
		Reason:    reason,
		CreatedAt: now,
	}); err != nil {
		return err
	}

	if before.State == StateSettled || before.State == StateFailed {
		n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
			Event:     store.TaskWait,
			QueuedAt:  sql.NullInt64{Int64: now, Valid: true},
			UpdatedAt: now,
			ID:        taskID,
		})
		if err != nil {
			return err
		}
		if n == 1 {
			after, err := s.snapshot(ctx, taskID)
			if err != nil {
				return err
			}
			s.logChanges(ctx, before, after)
			if holds, err := s.Holds(ctx, taskID); err == nil {
				s.transitionedHeld(ctx, taskID, holds)
			} else {
				s.transitioned(ctx, taskID, StateWaiting)
			}
			s.changed(ctx, taskID)
			return nil
		}
	}
	s.changed(ctx, taskID)
	return nil
}

func (s *Service) Release(
	ctx context.Context,
	taskID, pluginID, key string,
	opts ReleaseOptions,
) error {
	_, err := s.release(ctx, taskID, pluginID, key, opts)
	return err
}

func (s *Service) release(
	ctx context.Context,
	taskID, pluginID, key string,
	opts ReleaseOptions,
) (bool, error) {
	key, err := normalizeHoldKey(key)
	if err != nil {
		return false, err
	}
	mode, prompt, err := normalizeRelease(opts)
	if err != nil {
		return false, err
	}

	holds, err := s.Holds(ctx, taskID)
	if err != nil {
		return false, err
	}
	var released *TaskHold
	for i := range holds {
		if holds[i].PluginID == pluginID && holds[i].Key == key {
			released = &holds[i]
			break
		}
	}
	if released == nil {

		return false, nil
	}
	n, err := s.store.DeleteTaskHold(ctx, store.DeleteTaskHoldParams{
		TaskID:   taskID,
		PluginID: pluginID,
		Key:      key,
	})
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	s.changed(ctx, taskID)

	remaining, err := s.store.CountTaskHolds(ctx, taskID)
	if err != nil {
		s.restoreHold(ctx, *released)
		return false, err
	}
	if remaining > 0 {
		return false, nil
	}
	applied, err := s.applyRelease(ctx, taskID, mode, prompt, opts.ActorID)
	if err != nil {
		s.restoreHold(ctx, *released)
		return false, err
	}
	return applied, nil
}

func (s *Service) ReleaseHold(
	ctx context.Context,
	taskID, pluginID, key string,
	opts ReleaseOptions,
) (Task, error) {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return Task{}, errors.New("plugin_id is required")
	}
	key, err := normalizeHoldKey(key)
	if err != nil {
		return Task{}, err
	}
	mode, prompt, err := normalizeRelease(opts)
	if err != nil {
		return Task{}, err
	}
	holds, err := s.Holds(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	var released *TaskHold
	for i := range holds {
		if holds[i].PluginID == pluginID && holds[i].Key == key {
			released = &holds[i]
			break
		}
	}
	if released == nil {
		return Task{}, fmt.Errorf("%w: %s/%s", ErrHoldNotFound, pluginID, key)
	}
	applied, err := s.release(ctx, taskID, pluginID, key, ReleaseOptions{
		Mode:    mode,
		Prompt:  prompt,
		ActorID: opts.ActorID,
	})
	if err != nil {
		return Task{}, err
	}
	s.recordHoldRelease(ctx, *released, mode)
	task, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}

	s.announceHoldRelease(ctx, HoldRelease{
		TaskID:     released.TaskID,
		PluginID:   released.PluginID,
		PluginName: released.PluginName,
		Key:        released.Key,
		Reason:     released.Reason,
		Mode:       mode,
		Applied:    applied,
		Cause:      HoldReleasedByOperator,
	})
	return task, nil
}

func (s *Service) recordHoldRelease(
	ctx context.Context,
	hold TaskHold,
	mode ReleaseMode,
) {
	name := hold.PluginName
	if name == "" {
		name = hold.PluginID
	}
	s.recordLog(
		ctx, actorFrom(ctx), time.Now().UnixMilli(), hold.TaskID,
		LogHold, "hold", hold.Key, string(mode),
		fmt.Sprintf("Released the %s hold %q (%s)", name, hold.Key, mode),
		map[string]any{
			"plugin":     hold.PluginID,
			"pluginName": name,
			"key":        hold.Key,
			"reason":     hold.Reason,
			"mode":       string(mode),
		},
	)
}

func (s *Service) restoreHold(ctx context.Context, hold TaskHold) {
	if err := s.store.UpsertTaskHold(ctx, store.UpsertTaskHoldParams{
		TaskID:    hold.TaskID,
		PluginID:  hold.PluginID,
		Key:       hold.Key,
		Reason:    hold.Reason,
		CreatedAt: hold.CreatedAt.UnixMilli(),
	}); err != nil {
		log.Printf(
			"[tasks] restore hold %s/%s on %s: %v",
			hold.PluginID, hold.Key, hold.TaskID, err,
		)
		return
	}
	s.changed(ctx, hold.TaskID)
}

func (s *Service) applyRelease(
	ctx context.Context,
	taskID string,
	mode ReleaseMode,
	prompt string,
	actorID ids.ActorID,
) (bool, error) {
	cur, err := s.Get(ctx, taskID)
	if err != nil {
		return false, err
	}
	if cur.State != StateWaiting {
		return false, nil
	}

	if len(cur.Holds) > 0 {
		return false, nil
	}
	waiting, err := s.hasIncompleteDependencies(ctx, cur)
	if err != nil {
		return false, err
	}
	if waiting {
		return false, nil
	}
	now := time.Now().UnixMilli()
	switch mode {
	case ReleaseQueue:
		applied, err := s.releaseQueue(ctx, cur, prompt, actorID, now)
		if err != nil {
			return false, err
		}
		return applied, nil
	case ReleaseComplete:
		if _, err := s.Complete(ctx, taskID); err != nil {
			return false, err
		}
		return true, nil
	default:
		n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
			Event:     store.TaskReleaseSettle,
			SettledAt: sql.NullInt64{Int64: now, Valid: true},
			UpdatedAt: now,
			ID:        taskID,
		})
		if err != nil {
			return false, err
		}
		if n == 0 {
			return false, nil
		}
		s.transitioned(ctx, taskID, StateSettled)
		after, err := s.Get(ctx, taskID)
		if err != nil {
			return false, err
		}
		s.logChanges(ctx, cur, after)
		s.changed(ctx, taskID)

		if requeued, err := s.requeuePendingPrompt(ctx, taskID); err != nil {
			log.Printf("[tasks] requeue pending prompt %s: %v", taskID, err)
		} else if requeued && s.OnPendingPromptQueued != nil {
			s.OnPendingPromptQueued(ctx, taskID)
		}
		s.childFinished(ctx, taskID, ChildSettled)
		return true, nil
	}
}

func (s *Service) releaseQueue(
	ctx context.Context,
	cur Task,
	prompt string,
	actorID ids.ActorID,
	now int64,
) (bool, error) {
	if err := s.requireLead(ctx, cur.ID, StateQueued); err != nil {
		return false, err
	}
	if cur.ParentTaskID != "" {
		s.clearChildWakeKey(ctx, cur.ParentTaskID, cur.ID)
	}
	pendingActor := nullID(actorID)
	if stashed := strings.TrimSpace(cur.PendingPrompt); stashed != "" {
		prompt = stashed + "\n\n" + prompt
		pendingActor = nullID(cur.PendingPromptActorID)
	}
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:                store.TaskReleaseQueue,
		PendingPrompt:        prompt,
		PendingPromptActorID: pendingActor,
		QueuedAt:             sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt:            now,
		ID:                   cur.ID,
	})
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	after, err := s.Get(ctx, cur.ID)
	if err != nil {
		return false, err
	}
	s.logChanges(ctx, cur, after)
	s.changed(ctx, cur.ID)
	return true, nil
}

func (s *Service) ReleasePluginHolds(
	ctx context.Context,
	pluginID string,
) ([]HoldRelease, error) {
	taskIDs, err := s.store.ListTaskHoldTaskIDsByPlugin(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	var (
		releases []HoldRelease
		firstErr error
	)
	for _, taskID := range taskIDs {
		holds, err := s.Holds(ctx, taskID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, hold := range holds {
			if hold.PluginID != pluginID {
				continue
			}

			applied, err := s.release(ctx, taskID, pluginID, hold.Key, ReleaseOptions{
				Mode: ReleaseSettle,
			})
			if err != nil {
				log.Printf("[tasks] release hold %s on %s: %v", hold.Key, taskID, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			releases = append(releases, HoldRelease{
				TaskID:     hold.TaskID,
				PluginID:   hold.PluginID,
				PluginName: hold.PluginName,
				Key:        hold.Key,
				Reason:     hold.Reason,
				Mode:       ReleaseSettle,
				Applied:    applied,
				Cause:      HoldReleasedByPluginDisable,
			})
		}
	}
	return releases, firstErr
}

func (s *Service) ClearHolds(
	ctx context.Context,
	taskID string,
	cause HoldReleaseCause,
) error {
	holds, err := s.Holds(ctx, taskID)
	if err != nil {
		return err
	}
	n, err := s.store.DeleteTaskHolds(ctx, taskID)
	if err != nil {
		return err
	}
	if n > 0 {
		s.changed(ctx, taskID)
	}
	for _, hold := range holds {
		s.announceHoldRelease(ctx, HoldRelease{
			TaskID:     hold.TaskID,
			PluginID:   hold.PluginID,
			PluginName: hold.PluginName,
			Key:        hold.Key,
			Reason:     hold.Reason,
			Cause:      cause,
		})
	}
	return nil
}

func (s *Service) announceHoldRelease(ctx context.Context, r HoldRelease) {
	if s.OnHoldReleased == nil || r.TaskID == "" || r.PluginID == "" {
		return
	}
	s.OnHoldReleased(ctx, r)
}

func (s *Service) transitionedHeld(ctx context.Context, id string, holds []TaskHold) {
	if s.OnTransition == nil {
		return
	}
	s.OnTransition(ctx, Transition{
		TaskID: id,
		State:  StateWaiting,
		Held:   true,
		Holds:  holds,
	})
}

func heldError(holds []TaskHold) error {
	return fmt.Errorf("%w by %s", ErrHeld, holds[0].Summary())
}

func toTaskHolds(rows []store.ListTaskHoldsRow) []TaskHold {
	out := make([]TaskHold, len(rows))
	for i, row := range rows {
		out[i] = TaskHold{
			TaskID:     row.TaskID,
			PluginID:   row.PluginID,
			PluginName: row.PluginName,
			Key:        row.Key,
			Reason:     row.Reason,
			CreatedAt:  time.UnixMilli(row.CreatedAt),
		}
	}
	return out
}

func normalizeHoldKey(raw string) (string, error) {
	key := strings.TrimSpace(raw)
	if !holdKeyPattern.MatchString(key) {
		return "", ErrInvalidHoldKey
	}
	return key, nil
}

func normalizeHoldReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if utf8.RuneCountInString(reason) > MaxHoldReasonLength {
		return "", ErrHoldReasonTooLong
	}
	return reason, nil
}

func normalizeRelease(opts ReleaseOptions) (ReleaseMode, string, error) {
	mode := opts.Mode
	switch mode {
	case "":
		mode = ReleaseSettle
	case ReleaseSettle, ReleaseQueue, ReleaseComplete:
	default:
		return "", "", ErrInvalidReleaseMode
	}
	prompt := strings.TrimSpace(opts.Prompt)
	if mode == ReleaseQueue && prompt == "" {
		return "", "", fmt.Errorf("%w: a queued release needs a prompt", ErrInvalidPrompt)
	}
	if mode != ReleaseQueue {
		prompt = ""
	}
	return mode, prompt, nil
}
