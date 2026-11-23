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
	"log"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
	"github.com/google/uuid"
)

var ErrNotCancelable = errors.New(
	"that message is not cancelable; it was sent by a plugin or the agency",
)

var ErrLastPendingMessage = errors.New(
	"the task is queued by this message; dequeue the task instead of cancelling it",
)

type PendingMessage struct {
	ID string

	MessageID  string
	Text       string
	ActorID    ids.ActorID
	Cancelable bool
	CreatedAt  time.Time
}

type QueueParams struct {
	Text    string
	ActorID ids.ActorID

	MessageID string

	Cancelable bool
}

type FollowUpParams struct {
	TaskID     string
	Text       string
	ActorID    ids.ActorID
	MessageID  string
	Cancelable bool

	Force bool
}

type FollowUpResult struct {
	Task Task

	Queued bool
}

func (s *Service) FollowUp(
	ctx context.Context,
	p FollowUpParams,
) (FollowUpResult, error) {
	p.Text = strings.TrimSpace(p.Text)
	if p.Text == "" {
		return FollowUpResult{}, ErrInvalidPrompt
	}
	cur, err := s.Get(ctx, p.TaskID)
	if err != nil {
		return FollowUpResult{}, err
	}
	switch cur.State {
	case StateBacklog:
		if cur.StartedAt.IsZero() {
			return FollowUpResult{}, ErrBacklogNotStarted
		}
	case StateSettled, StateFailed, StateCompleted, StateCancelled:

		if err := s.validateRequeue(ctx, cur); err != nil {
			return FollowUpResult{}, err
		}
	case StateWaiting:
		if p.Force {

			woken, err := s.ForceWake(ctx, cur, p.Text, p.ActorID)
			if err != nil {
				return FollowUpResult{}, err
			}
			if woken {
				after, err := s.Get(ctx, cur.ID)
				if err != nil {
					return FollowUpResult{}, err
				}
				return FollowUpResult{Task: after}, nil
			}
		}
	}
	messages, err := s.queuePendingMessage(ctx, cur.ID, QueueParams{
		Text:       p.Text,
		ActorID:    p.ActorID,
		MessageID:  p.MessageID,
		Cancelable: p.Cancelable,
	})
	if err != nil {
		return FollowUpResult{}, err
	}
	delivered, err := s.deliverQueued(ctx, cur, messages)
	if err != nil {
		return FollowUpResult{}, err
	}
	after, err := s.Get(ctx, cur.ID)
	if err != nil {
		return FollowUpResult{}, err
	}
	return FollowUpResult{Task: after, Queued: !delivered}, nil
}

func (s *Service) deliverQueued(
	ctx context.Context,
	cur Task,
	messages []PendingMessage,
) (bool, error) {
	switch cur.State {
	case StateSettled, StateFailed, StateCompleted, StateCancelled:
		return false, s.requeueQueued(ctx, cur, messages)
	case StateBacklog:

		return false, s.requeueQueued(ctx, cur, messages)
	case StateRunning:
		if s.DeliverPendingPrompt == nil || cur.SessionID == "" {
			return false, nil
		}
		after, err := s.Get(ctx, cur.ID)
		if err != nil {
			return false, err
		}
		return s.DeliverPendingPrompt(ctx, after), nil
	default:

		return false, nil
	}
}

func (s *Service) requeueQueued(
	ctx context.Context,
	cur Task,
	messages []PendingMessage,
) error {
	text, actorID := joinPending(messages)
	if err := s.requeueWithPrompt(ctx, cur, text, actorID); err != nil {
		return err
	}
	if s.OnPendingPromptQueued != nil {
		s.OnPendingPromptQueued(ctx, cur.ID)
	}
	return nil
}

func (s *Service) queuePendingMessage(
	ctx context.Context,
	taskID string,
	p QueueParams,
) ([]PendingMessage, error) {
	text := strings.TrimSpace(p.Text)
	if text == "" {
		return nil, nil
	}
	id := p.MessageID
	if id == "" {
		id = uuid.NewString()
	}
	if err := s.store.CreateTaskPendingMessage(ctx, store.CreateTaskPendingMessageParams{
		ID:         id,
		TaskID:     taskID,
		MessageID:  nullString(p.MessageID),
		Text:       text,
		ActorID:    nullID(p.ActorID),
		Cancelable: boolInt(p.Cancelable),
		CreatedAt:  time.Now().UnixMilli(),
	}); err != nil {
		return nil, err
	}
	messages, err := s.PendingMessages(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if err := s.writePendingPrompt(ctx, taskID, messages); err != nil {
		return nil, err
	}
	s.logQueuedMessage(ctx, taskID, text, p.ActorID)
	return messages, nil
}

func (s *Service) writePendingPrompt(
	ctx context.Context,
	taskID string,
	messages []PendingMessage,
) error {
	text, actorID := joinPending(messages)
	if _, err := s.store.SetTaskPendingPromptAny(ctx, store.SetTaskPendingPromptAnyParams{
		PendingPrompt:        text,
		PendingPromptActorID: nullID(actorID),
		UpdatedAt:            time.Now().UnixMilli(),
		ID:                   taskID,
	}); err != nil {
		return err
	}
	s.changed(ctx, taskID)
	return nil
}

func (s *Service) PendingMessages(
	ctx context.Context,
	taskID string,
) ([]PendingMessage, error) {
	rows, err := s.store.ListTaskPendingMessages(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]PendingMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, PendingMessage{
			ID:         row.ID,
			MessageID:  row.MessageID.String,
			Text:       row.Text,
			ActorID:    ids.ActorID(row.ActorID.Int64),
			Cancelable: row.Cancelable != 0,
			CreatedAt:  time.UnixMilli(row.CreatedAt),
		})
	}
	return out, nil
}

func (s *Service) RemovePendingMessage(
	ctx context.Context,
	taskID, messageID string,
) (Task, error) {
	cur, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	messages, err := s.PendingMessages(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	var target *PendingMessage
	for i := range messages {
		if messages[i].ID == messageID {
			target = &messages[i]
			break
		}
	}
	if target == nil {

		return cur, nil
	}
	if !CanCancelPendingMessage(cur.State, messages, *target) {
		if !target.Cancelable {
			return Task{}, ErrNotCancelable
		}
		return Task{}, ErrLastPendingMessage
	}
	if _, err := s.store.DeleteTaskPendingMessage(ctx, store.DeleteTaskPendingMessageParams{
		ID:     messageID,
		TaskID: taskID,
	}); err != nil {
		return Task{}, err
	}
	remaining := make([]PendingMessage, 0, len(messages)-1)
	for _, message := range messages {
		if message.ID != messageID {
			remaining = append(remaining, message)
		}
	}
	if err := s.writePendingPrompt(ctx, taskID, remaining); err != nil {
		return Task{}, err
	}
	s.recordLog(
		ctx, actorFrom(ctx), time.Now().UnixMilli(), taskID,
		LogMessage, "cancelled", "", "", "Queued message cancelled",
		messagePreview(target.Text),
	)
	return s.Get(ctx, taskID)
}

func CanCancelPendingMessage(
	state State,
	messages []PendingMessage,
	message PendingMessage,
) bool {
	if !message.Cancelable {
		return false
	}
	switch state {
	case StateQueued, StateWaiting:
		return len(messages) > 1
	default:
		return true
	}
}

func (s *Service) ClearPendingPrompt(ctx context.Context, id string) error {

	pending := ""
	if before, err := s.snapshot(ctx, id); err == nil {
		pending = before.PendingPrompt
	}
	if err := s.discardPendingMessages(ctx, id); err != nil {
		return err
	}
	_, err := s.store.ClearTaskPendingPrompt(ctx, store.ClearTaskPendingPromptParams{
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	if err == nil {
		s.logDeliveredMessage(ctx, id, pending)
		s.changed(ctx, id)
	}
	return err
}

func (s *Service) discardPendingMessages(ctx context.Context, id string) error {
	if _, err := s.store.DeleteTaskPendingMessages(ctx, id); err != nil {
		return err
	}
	return nil
}

func (s *Service) RequeueAfterSessionLost(
	ctx context.Context,
	id string,
) (bool, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return false, err
	}
	if cur.State != StateRunning && cur.State != StateInput {
		return false, nil
	}
	if strings.TrimSpace(cur.PendingPrompt) == "" ||
		s.pendingPromptDelivered(ctx, id, cur.PendingPrompt) {
		return false, nil
	}
	if len(cur.Holds) > 0 {

		return false, nil
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskRequeueLostSession,
		QueuedAt:  sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return true, err
	}
	s.logChanges(ctx, cur, after)
	if err := s.ReconcileWaiting(ctx, id); err != nil {
		log.Printf("[tasks] reconcile waiting after lost-session requeue %s: %v", id, err)
	}
	s.changed(ctx, id)
	return true, nil
}

func joinPending(messages []PendingMessage) (string, ids.ActorID) {
	if len(messages) == 0 {
		return "", 0
	}
	texts := make([]string, 0, len(messages))
	actor := ids.ActorID(0)
	for _, message := range messages {
		if text := strings.TrimSpace(message.Text); text != "" {
			texts = append(texts, text)
		}
		if actor == 0 {
			actor = message.ActorID
		}
	}
	return strings.Join(texts, "\n\n"), actor
}
