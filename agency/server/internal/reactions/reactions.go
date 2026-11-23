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

package reactions

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

const (
	DefaultMaxReminders = 3

	DefaultOfflineAfter = 15 * time.Minute
)

type Service struct {
	Tasks *tasks.Service

	IsAgent func(ctx context.Context, actorID ids.ActorID) bool

	MaxReminders int

	OfflineAfter time.Duration

	Now func() time.Time

	Wake func()
	// Steer delivers a line into a running reaction's live session, so a

	Steer func(ctx context.Context, t tasks.Task, prompt string) error
}

func (s *Service) maxReminders() int {
	if s.MaxReminders > 0 {
		return s.MaxReminders
	}
	return DefaultMaxReminders
}

func (s *Service) offlineAfter() time.Duration {
	if s.OfflineAfter > 0 {
		return s.OfflineAfter
	}
	return DefaultOfflineAfter
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) OnEvent(ctx context.Context, taskID string) {
	if s.Tasks == nil || s.IsAgent == nil {
		return
	}
	t, err := s.Tasks.Get(ctx, taskID)
	if err != nil || t.Automated {
		return
	}
	if t.State != tasks.StateInput && t.State != tasks.StateSettled {
		return
	}
	if t.LeaderActorID == 0 {

		return
	}
	if !s.IsAgent(ctx, t.LeaderActorID) {

		return
	}
	live, err := s.Tasks.LiveReactions(ctx, t.ID)
	if err != nil {
		log.Printf("[reactions] live reactions of %s: %v", t.ID, err)
		return
	}
	if len(live) > 0 {

		s.join(ctx, live[0], t)
		return
	}
	if _, err := s.Tasks.Create(ctx, tasks.CreateParams{
		Prompt: reactionPrompt(t),
		Name:   reactionName(t),

		ProjectID:       t.ProjectID,
		TeamIDs:         t.TeamIDs,
		TagIDs:          tagIDsOf(t),
		AssigneeActorID: t.LeaderActorID,
		OriginTaskID:    t.ID,
		Automated:       true,

		Reaction: true,
		State:    tasks.StateQueued,

		AutoTitle: false,
	}); err != nil {
		log.Printf("[reactions] create reaction for %s: %v", t.ID, err)
		return
	}
}

func (s *Service) join(ctx context.Context, reaction, origin tasks.Task) {
	line := fmt.Sprintf(
		"The origin task %s changed again: it is now %s. Resolve it before "+
			"finishing.",
		origin.ID, origin.State,
	)
	switch reaction.State {
	case tasks.StateRunning:
		if s.Steer == nil {
			log.Printf(
				"[reactions] %s is running and cannot be steered; the new event "+
					"on %s is left to its current turn",
				reaction.ID, origin.ID,
			)
			return
		}
		if err := s.Steer(ctx, reaction, line); err != nil {
			log.Printf("[reactions] steer %s: %v", reaction.ID, err)
		}
	case tasks.StateQueued, tasks.StateWaiting, tasks.StateInput:
		if err := s.Tasks.AppendPendingPrompt(ctx, reaction.ID, line, 0); err != nil {
			log.Printf("[reactions] join %s: %v", reaction.ID, err)
			return
		}
		if s.Wake != nil {
			s.Wake()
		}
	case tasks.StateSettled, tasks.StateFailed:
		if _, err := s.Tasks.RequeueWithPrompt(ctx, reaction.ID, line, 0); err != nil {
			log.Printf("[reactions] requeue %s: %v", reaction.ID, err)
		}
	}
}

func (s *Service) OnFinished(
	ctx context.Context,
	t tasks.Task,
	outcome tasks.FinishOutcome,
) {
	if s.Tasks == nil || !t.Reaction || t.OriginTaskID == "" {
		return
	}
	switch outcome {
	case tasks.OutcomeCompleted, tasks.OutcomeCancelled:
		return
	case tasks.OutcomeFailed:

		return
	}
	origin, err := s.Tasks.Get(ctx, t.OriginTaskID)
	if err != nil {

		s.closeOut(ctx, t)
		return
	}
	if !Unresolved(origin.State) {
		s.closeOut(ctx, t)
		return
	}
	reminders, err := s.Tasks.BumpReactionReminders(ctx, t.ID)
	if err != nil {
		log.Printf("[reactions] bump reminders %s: %v", t.ID, err)
		return
	}
	if reminders >= s.maxReminders() {
		s.needsOperator(ctx, t, origin,
			fmt.Sprintf(
				"reaction for %s gave up after %d unresolved turns: the origin "+
					"is still %s; answer it, close it out, or re-queue it, then "+
					"re-queue this task",
				origin.ID, reminders, origin.State,
			),
		)
		return
	}
	reminder := fmt.Sprintf(
		"Reminder %d/%d: the origin task %s is still %s. Resolve it before "+
			"finishing — answer its question with answer_origin_task_questions, "+
			"close it out with complete_origin_task or cancel_origin_task, or "+
			"re-queue it with queueTask once you have added what it needs.",
		reminders, s.maxReminders(), origin.ID, origin.State,
	)
	if _, err := s.Tasks.RequeueWithPrompt(ctx, t.ID, reminder, 0); err != nil {
		log.Printf("[reactions] reminder %s: %v", t.ID, err)
	}
}

func Unresolved(state tasks.State) bool {
	return state == tasks.StateInput || state == tasks.StateSettled
}

func (s *Service) OnResolved(ctx context.Context, originID string) {
	if s.Tasks == nil || originID == "" {
		return
	}
	origin, err := s.Tasks.Get(ctx, originID)
	if err != nil || Unresolved(origin.State) {
		return
	}
	s.Reconcile(ctx, origin)
}

func (s *Service) Reconcile(ctx context.Context, origin tasks.Task) {
	if s.Tasks == nil || Unresolved(origin.State) {
		return
	}
	live, err := s.Tasks.LiveReactions(ctx, origin.ID)
	if err != nil {
		log.Printf("[reactions] live reactions of %s: %v", origin.ID, err)
		return
	}
	for _, reaction := range live {
		s.closeOut(ctx, reaction)
	}
}

func (s *Service) Sweep(ctx context.Context) {
	if s.Tasks == nil {
		return
	}
	cutoff := s.now().Add(-s.offlineAfter())
	stalled, err := s.Tasks.StalledReactions(ctx, cutoff)
	if err != nil {
		log.Printf("[reactions] list stalled reactions: %v", err)
		return
	}
	for _, reaction := range stalled {
		origin, err := s.Tasks.Get(ctx, reaction.OriginTaskID)
		if err != nil || !Unresolved(origin.State) {
			s.closeOut(ctx, reaction)
			continue
		}
		s.needsOperator(ctx, reaction, origin, fmt.Sprintf(
			"reaction for %s is stalled: its assignee has been offline for "+
				"over %s and the origin is still %s; reassign and re-queue it",
			origin.ID, s.offlineAfter(), origin.State,
		))
	}
}

func (s *Service) closeOut(ctx context.Context, reaction tasks.Task) {
	if reaction.State == tasks.StateCompleted || reaction.State == tasks.StateCancelled {
		return
	}
	if err := s.Tasks.ResetReactionReminders(ctx, reaction.ID); err != nil {
		log.Printf("[reactions] reset reminders %s: %v", reaction.ID, err)
	}
	if _, err := s.Tasks.Complete(ctx, reaction.ID); err != nil {
		log.Printf("[reactions] close out %s: %v", reaction.ID, err)
		return
	}
	if err := s.Tasks.SetNeedsOperator(ctx, reaction.ID, ""); err != nil {
		log.Printf("[reactions] clear needs_operator %s: %v", reaction.ID, err)
	}
}

func (s *Service) needsOperator(
	ctx context.Context,
	reaction, origin tasks.Task,
	reason string,
) {
	if err := s.Tasks.FailNeedsOperator(ctx, reaction.ID, reason); err != nil {
		log.Printf("[reactions] fail %s for an operator: %v", reaction.ID, err)
		return
	}
	log.Printf("[reactions] %s needs an operator: %s", reaction.ID, reason)
}

func reactionName(origin tasks.Task) string {
	subject := origin.ID
	if name := strings.TrimSpace(origin.Name); name != "" {
		subject = origin.ID + " " + name
	}
	return "Resolve " + subject
}

func reactionPrompt(origin tasks.Task) string {
	var event string
	switch origin.State {
	case tasks.StateInput:
		event = "is waiting for an answer to a question"
	case tasks.StateSettled:
		event = "has settled and needs its outcome accepted"
	default:
		event = "needs attention"
	}
	question := ""
	if origin.State == tasks.StateInput {
		question = "\nRead the pending dialog with getTaskInput and answer it " +
			"with answer_origin_task_questions.\n"
	}
	return fmt.Sprintf(`You are the automated reaction for %s, which %s.

This task is not work of its own: it exists so the agent accountable for %s can
resolve that event. You may create and block on other tasks to do it, and you
must not finish until %s is resolved:

- answer its pending question with answer_origin_task_questions (read it first
  with getTaskInput), or
- accept its outcome with complete_origin_task, or abandon it with
  cancel_origin_task, or
- re-queue it with queueTask once you have added the blocking tasks it needs.

Use getTask and getTaskHistory to see what it did. The agency closes this task
out automatically once %s is resolved; settling without resolving it only
queues it again with a reminder, and after a few turns it is failed and handed
to a person.
%s
Origin task: %s
Prompt of the origin task:

%s`, origin.ID, event, origin.ID, origin.ID, origin.ID, question,
		origin.ID, origin.Prompt)
}

func tagIDsOf(t tasks.Task) []string {
	out := make([]string, 0, len(t.Tags))
	for _, tag := range t.Tags {
		out = append(out, tag.ID)
	}
	return out
}
