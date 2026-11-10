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

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"

	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/limits"
	pluginhost "cherrypi.vladfaust.com/agency/internal/plugins"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"connectrpc.com/connect"
	"github.com/google/uuid"
)

const DefaultInterval = 5 * time.Second

type Scheduler struct {
	Tasks    *tasks.Service
	Projects *projects.Service
	Limits   *limits.Service
	Prompts  *prompts.Service
	Agents   *agents.Registry

	Attachments *attachments.Service

	Plugins *pluginhost.Service

	Interval time.Duration

	Connected  func(agentID ids.AgentID) bool
	CreateTask func(
		ctx context.Context,
		agentID ids.AgentID,
		req *agentv1.CreateTaskRequest,
	) (sessionID string, err error)
	RetryTask func(ctx context.Context, agentID ids.AgentID, sessionID string, ref *agentv1.TaskRef) error

	AddToTask func(
		ctx context.Context,
		agentID ids.AgentID, sessionID, prompt string,
		ref *agentv1.TaskRef,
		sender *agentv1.PromptSender,
		atts []*agentv1.AttachmentRef,
	) error

	AgentTasks func(ctx context.Context, agentID ids.AgentID) ([]*agentv1.Task, error)

	wake chan struct{}
	mu   sync.Mutex

	waiting map[string]string
}

func New(
	tasksSvc *tasks.Service,
	projectsSvc *projects.Service,
	limitsSvc *limits.Service,
	promptsSvc *prompts.Service,
	agentsReg *agents.Registry,
	hub *tunnel.Hub,
) *Scheduler {
	s := &Scheduler{
		Tasks:    tasksSvc,
		Projects: projectsSvc,
		Limits:   limitsSvc,
		Prompts:  promptsSvc,
		Agents:   agentsReg,
		Interval: DefaultInterval,
		wake:     make(chan struct{}, 1),
	}

	if hub != nil {
		s.Connected = func(agentID ids.AgentID) bool {
			_, err := hub.Connection(agentID)
			return err == nil
		}
		s.CreateTask = func(
			ctx context.Context,
			agentID ids.AgentID,
			req *agentv1.CreateTaskRequest,
		) (string, error) {
			conn, err := hub.Connection(agentID)
			if err != nil {
				return "", err
			}
			res, err := conn.Tasks.CreateTask(ctx, req)
			if err != nil {
				return "", err
			}
			if res.Task == nil || res.Task.GetId() == "" {
				return "", errors.New("agent returned no task id")
			}
			return res.Task.GetId(), nil
		}
		s.RetryTask = func(
			ctx context.Context,
			agentID ids.AgentID, sessionID string,
			ref *agentv1.TaskRef,
		) error {
			conn, err := hub.Connection(agentID)
			if err != nil {
				return err
			}
			_, err = conn.Tasks.RetryTask(ctx, &agentv1.RetryTaskRequest{
				Id:  sessionID,
				Ref: ref,
			})
			return err
		}
		s.AddToTask = func(
			ctx context.Context,
			agentID ids.AgentID, sessionID, prompt string,
			ref *agentv1.TaskRef,
			sender *agentv1.PromptSender,
			atts []*agentv1.AttachmentRef,
		) error {
			conn, err := hub.Connection(agentID)
			if err != nil {
				return err
			}
			_, err = conn.Tasks.AddToTask(ctx, &agentv1.AddToTaskRequest{
				Id:          sessionID,
				Prompt:      prompt,
				Attachments: atts,
				Ref:         ref,
				Sender:      sender,
			})
			return err
		}
		s.AgentTasks = func(ctx context.Context, agentID ids.AgentID) ([]*agentv1.Task, error) {
			conn, err := hub.Connection(agentID)
			if err != nil {
				return nil, err
			}
			res, err := conn.Tasks.ListTasks(ctx, &agentv1.ListTasksRequest{})
			if err != nil {
				return nil, err
			}
			return res.Tasks, nil
		}
	}

	return s
}

func (s *Scheduler) Run(ctx context.Context) {
	interval := s.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
			s.Drain(ctx)
			s.Reconcile(ctx)
		case <-ticker.C:
			s.Drain(ctx)
			s.Reconcile(ctx)
		}
	}
}

func (s *Scheduler) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Scheduler) FollowUp(
	ctx context.Context,
	t tasks.Task,
	prompt string,
	actorID ids.ActorID,
) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return errors.New("the message is empty")
	}
	switch t.State {
	case tasks.StateSettled, tasks.StateFailed,
		tasks.StateCompleted, tasks.StateCancelled:
		if _, err := s.Tasks.RequeueWithPrompt(ctx, t.ID, prompt, actorID); err != nil {
			return err
		}
		s.Wake()
		return nil
	case tasks.StateQueued, tasks.StateWaiting:
		if err := s.Tasks.AppendPendingPrompt(ctx, t.ID, prompt, actorID); err != nil {
			return err
		}
		s.Wake()
		return nil
	case tasks.StateInput:

		return s.Tasks.AppendPendingPrompt(ctx, t.ID, prompt, actorID)
	case tasks.StateRunning:
		if t.SessionID == "" || s.AddToTask == nil {
			return errors.New("the task has no warm session to steer")
		}
		return s.AddToTask(
			ctx, t.SessionAgentID, t.SessionID, prompt,
			taskRef(t), s.sender(ctx, actorID), nil,
		)
	default:
		return fmt.Errorf(
			"a task in state %q does not accept a follow-up", t.State,
		)
	}
}

func (s *Scheduler) DeliverPendingPrompt(ctx context.Context, t tasks.Task) {
	if strings.TrimSpace(t.PendingPrompt) == "" {
		return
	}
	if t.SessionID == "" || s.AddToTask == nil {
		return
	}
	err := s.AddToTask(
		ctx, t.SessionAgentID, t.SessionID, t.PendingPrompt,
		taskRef(t), s.sender(ctx, t.PendingPromptActorID),
		s.pendingAttachments(ctx, t),
	)
	if err != nil {
		log.Printf("[scheduler] deliver pending prompt %s: %v", t.ID, err)
		return
	}
	s.clearDeliveredPrompt(ctx, t.ID, t.PendingPrompt)
}

func (s *Scheduler) clearDeliveredPrompt(
	ctx context.Context,
	taskID, prompt string,
) {
	if err := s.Tasks.ClearPendingPrompt(ctx, taskID); err != nil {
		log.Printf("[scheduler] clear pending prompt %s: %v", taskID, err)
		if err := s.Tasks.MarkPendingPromptDelivered(
			ctx, taskID, prompt,
		); err != nil {
			log.Printf(
				"[scheduler] mark pending prompt delivered %s: %v",
				taskID, err,
			)
		}
	}
}

func (s *Scheduler) Drain(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	queued, err := s.Tasks.ListQueued(ctx)
	if err != nil {
		log.Printf("[scheduler] list queued: %v", err)
		return
	}

	view := s.newFitView()
	for _, task := range queued {
		if ctx.Err() != nil {
			return
		}
		agent, ok := s.pickAgent(ctx, task, view)
		if !ok {
			s.noteWaiting(ctx, task, view)
			continue
		}
		s.start(ctx, task, agent)
	}

	if len(s.waiting) > 0 {
		queuedIDs := make(map[string]struct{}, len(queued))
		for _, task := range queued {
			queuedIDs[task.ID] = struct{}{}
		}
		for id := range s.waiting {
			if _, ok := queuedIDs[id]; !ok {
				delete(s.waiting, id)
			}
		}
	}
}

func (s *Scheduler) noteWaiting(
	ctx context.Context,
	task tasks.Task,
	view *fitView,
) {
	reason := s.waitingReason(ctx, task, view)
	if reason == "" {
		return
	}
	if s.waiting == nil {
		s.waiting = map[string]string{}
	}
	if s.waiting[task.ID] == reason {
		return
	}
	s.waiting[task.ID] = reason
	log.Printf("[scheduler] %s (%s) not started: %s", task.ID, task.State, reason)
}

func (s *Scheduler) Reconcile(ctx context.Context) {
	if s.AgentTasks == nil {
		return
	}
	held, err := s.Tasks.ListSessionTasks(ctx)
	if err != nil {
		log.Printf("[scheduler] list session tasks: %v", err)
		return
	}

	byAgent := map[ids.AgentID][]tasks.Task{}
	for _, task := range held {
		if task.SessionID == "" || task.SessionAgentID == 0 {
			continue
		}
		byAgent[task.SessionAgentID] = append(byAgent[task.SessionAgentID], task)
	}

	for agentID, agentTasks := range byAgent {
		if ctx.Err() != nil {
			return
		}
		live, err := s.AgentTasks(ctx, agentID)
		if err != nil {
			continue
		}
		bySession := make(map[string]*agentv1.Task, len(live))
		for _, at := range live {
			bySession[at.GetId()] = at
		}
		for _, task := range agentTasks {
			at := bySession[task.SessionID]
			if at == nil {
				s.sessionLost(ctx, task)
				continue
			}

			if task.State != tasks.StateRunning && task.State != tasks.StateInput {
				continue
			}

			switch {
			case at.GetStatus() == agentv1.TaskStatus_TASK_STATUS_FAILED:
				errMsg := at.GetError()
				if errMsg == "" {
					errMsg = "the agent's run failed"
				}
				if err := s.Tasks.FailBySession(
					ctx,
					agentID,
					task.SessionID,
					errMsg,
				); err != nil {
					log.Printf("[scheduler] fail %s: %v", task.ID, err)
				}
			case at.GetWaitingForInput():
				if task.State != tasks.StateInput {
					if _, err := s.Tasks.MarkInput(ctx, task.ID); err != nil {
						log.Printf("[scheduler] mark input %s: %v", task.ID, err)
					} else {
						logReconciled(task, task.State, tasks.StateInput, "waiting for input")
					}
				}
			case at.GetRunInProgress():
				if task.State == tasks.StateInput {
					if _, err := s.Tasks.ClearInput(ctx, task.ID); err != nil {
						log.Printf("[scheduler] clear input %s: %v", task.ID, err)
					} else {
						logReconciled(task, task.State, tasks.StateRunning, "run in flight")
					}
				}
			default:
				if settled, err := s.Tasks.Settle(ctx, task.ID); err != nil {
					log.Printf("[scheduler] settle %s: %v", task.ID, err)
				} else if settled {

					state := tasks.StateSettled
					if after, err := s.Tasks.Get(ctx, task.ID); err == nil {
						state = after.State
					}
					logReconciled(task, task.State, state, "run finished")
				}
			}
		}
	}
}

func (s *Scheduler) sessionLost(ctx context.Context, task tasks.Task) {
	log.Printf(
		"[scheduler] session lost %s on agent %d (session=%s, state=%s)",
		task.ID, task.SessionAgentID, task.SessionID, task.State,
	)
	if _, err := s.Tasks.MarkSessionLost(ctx, task.ID, sessionLostReason); err != nil {
		log.Printf("[scheduler] mark session lost %s: %v", task.ID, err)
		return
	}
	if task.State == tasks.StateRunning || task.State == tasks.StateInput {
		failReason := sessionLostReason +
			"; re-queue or follow up to recreate it"
		if _, err := s.Tasks.Fail(ctx, task.ID, failReason); err != nil {
			log.Printf("[scheduler] fail lost session %s: %v", task.ID, err)
		}
	}
}

const sessionLostReason = "the agent no longer has this task's session " +
	"(it may have been restarted on another machine with fresh state)"

func logReconciled(task tasks.Task, from, to tasks.State, why string) {
	log.Printf(
		"[scheduler] reconciled %s: %s -> %s (%s, session=%s)",
		task.ID, from, to, why, task.SessionID,
	)
}

type agentVerdict int

const (
	agentReady agentVerdict = iota

	agentOffline

	agentForeign

	agentViewer

	agentUnqualified

	agentFull

	agentReserved

	agentUnknown
)

type agentCheck struct {
	verdict agentVerdict
	class   string
	missing []tasks.Tag
}

func (s *Scheduler) checkAgent(
	ctx context.Context,
	task tasks.Task,
	agent agents.Agent,
	view *fitView,
) (agentCheck, error) {
	member, canPrompt, err := s.Tasks.CanRunTask(ctx, task.ProjectID, agent.ID)
	if err != nil {
		return agentCheck{verdict: agentUnknown}, err
	}
	if !member {
		return agentCheck{verdict: agentForeign}, nil
	}

	if !canPrompt {
		return agentCheck{verdict: agentViewer}, nil
	}
	lim, err := view.agentLimits(ctx, agent.ID)
	if err != nil {
		return agentCheck{verdict: agentUnknown}, err
	}

	if missing := tasks.MissingTagDeclarations(
		lim,
		task.ProjectID,
		task.Tags,
	); len(missing) > 0 {
		return agentCheck{verdict: agentUnqualified, missing: missing}, nil
	}
	if s.Connected == nil || !s.Connected(agent.ID) {
		return agentCheck{verdict: agentOffline}, nil
	}
	counts, err := s.Tasks.RunningCounts(ctx, agent.ID)
	if err != nil {
		return agentCheck{verdict: agentUnknown}, err
	}
	if !tasks.Fits(lim, counts, task.ProjectID, task.Tags) {
		return agentCheck{verdict: agentFull}, nil
	}

	blocked, class, err := view.agentBlocks(ctx, agent.ID, lim.MaxConcurrent, task.Tags)
	if err != nil {
		return agentCheck{verdict: agentUnknown}, err
	}
	if blocked {
		return agentCheck{verdict: agentReserved, class: class}, nil
	}
	if task.ProjectID != "" {
		blocked, class, err = view.projectBlocks(ctx, task.ProjectID, task.Tags)
		if err != nil {
			return agentCheck{verdict: agentUnknown}, err
		}
		if blocked {
			return agentCheck{verdict: agentReserved, class: class}, nil
		}
	}
	return agentCheck{verdict: agentReady}, nil
}

func (s *Scheduler) pickAgent(
	ctx context.Context,
	task tasks.Task,
	view *fitView,
) (agents.Agent, bool) {
	candidates := s.candidates(ctx, task)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })

	for _, agent := range candidates {
		check, err := s.checkAgent(ctx, task, agent, view)
		if err != nil {
			log.Printf("[scheduler] check agent %d for %s: %v", agent.ID, task.ID, err)
			continue
		}
		if check.verdict == agentReady {
			return agent, true
		}
	}
	return agents.Agent{}, false
}

func (s *Scheduler) WaitingReason(ctx context.Context, task tasks.Task) string {
	if task.State != tasks.StateQueued && task.State != tasks.StateWaiting {
		return ""
	}
	return s.waitingReason(ctx, task, s.newFitView())
}

func (s *Scheduler) waitingReason(
	ctx context.Context,
	task tasks.Task,
	view *fitView,
) string {
	reasons := make([]string, 0, 3)

	if len(task.Holds) > 0 {
		holds := make([]string, 0, len(task.Holds))
		for _, hold := range task.Holds {
			holds = append(holds, "Waiting for "+hold.Summary())
		}
		reasons = append(reasons, strings.Join(holds, "; "))
	}
	if blockers := tasks.IncompleteBlockerIDs(task.References); len(blockers) > 0 {
		sort.SliceStable(blockers, func(i, j int) bool {
			return taskIDLess(blockers[i], blockers[j])
		})
		reasons = append(
			reasons,
			"Waiting for tasks to become completed: "+strings.Join(blockers, ", "),
		)
	}
	if children, err := s.Tasks.IncompleteChildIDs(ctx, task.ID); err == nil && len(children) > 0 {
		sort.SliceStable(children, func(i, j int) bool {
			return taskIDLess(children[i], children[j])
		})
		reasons = append(
			reasons,
			"Waiting for sub-tasks to finish: "+strings.Join(children, ", "),
		)
	}
	if len(reasons) > 0 {
		return strings.Join(reasons, "; ")
	}
	if task.State == tasks.StateWaiting {
		return "Waiting for the scheduler"
	}

	candidates := s.candidates(ctx, task)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	if len(candidates) == 0 {
		return "Waiting for an eligible agent"
	}

	full, foreign, viewer, unknown, reserved := false, false, false, false, ""
	offline := false
	missingTags := map[string]string{}
	for _, agent := range candidates {
		check, err := s.checkAgent(ctx, task, agent, view)
		if err != nil {
			unknown = true
			continue
		}
		switch check.verdict {
		case agentReady:
			return "Waiting for the scheduler"
		case agentFull:
			full = true
		case agentReserved:
			if reserved == "" {
				reserved = check.class
			}
		case agentUnqualified:
			for _, tag := range check.missing {
				missingTags[tag.ID] = tag.Name
			}
		case agentForeign:
			foreign = true
		case agentViewer:
			viewer = true
		case agentOffline:
			offline = true
		}
	}
	if reserved != "" {
		return "Waiting for a slot reserved for #" + reserved
	}
	if full {
		return "Waiting for free agent slot"
	}
	if offline {
		return "Waiting for an agent to connect"
	}
	if len(missingTags) > 0 {
		names := make([]string, 0, len(missingTags))
		for _, name := range missingTags {
			names = append(names, "#"+name)
		}
		sort.Strings(names)
		return "Waiting for an agent with required tags: " + strings.Join(names, ", ")
	}
	if viewer {
		return "Waiting for an agent with member access"
	}
	if foreign || unknown {
		return "Waiting for an eligible agent"
	}
	return "Waiting for an agent to connect"
}

func taskIDLess(a, b string) bool {
	ap, an, aok := splitTaskID(a)
	bp, bn, bok := splitTaskID(b)
	if aok && bok {
		if ap != bp {
			return ap < bp
		}
		return an < bn
	}
	return a < b
}

func splitTaskID(id string) (string, int64, bool) {
	i := strings.LastIndexByte(id, '-')
	if i <= 0 || i == len(id)-1 {
		return "", 0, false
	}
	n, err := strconv.ParseInt(id[i+1:], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return id[:i], n, true
}

func (s *Scheduler) candidates(ctx context.Context, task tasks.Task) []agents.Agent {
	if task.AssigneeActorID != 0 {
		agent, err := s.Agents.GetByActor(ctx, task.AssigneeActorID)
		if err != nil {
			log.Printf("[scheduler] pinned assignee %d: %v", task.AssigneeActorID, err)
			return nil
		}
		return []agents.Agent{agent}
	}
	if task.ProjectID != "" {
		members, err := s.Projects.ListAgents(ctx, task.ProjectID)
		if err != nil {
			log.Printf("[scheduler] project members %s: %v", task.ProjectID, err)
			return nil
		}
		want := make(map[ids.AgentID]bool, len(members))
		for _, m := range members {
			want[m.AgentID] = true
		}
		all, err := s.Agents.List(ctx)
		if err != nil {
			log.Printf("[scheduler] list agents: %v", err)
			return nil
		}
		out := make([]agents.Agent, 0, len(all))
		for _, a := range all {
			if want[a.ID] {
				out = append(out, a)
			}
		}
		return out
	}
	list, err := s.Agents.List(ctx)
	if err != nil {
		log.Printf("[scheduler] list agents: %v", err)
		return nil
	}
	return list
}

func (s *Scheduler) attachmentRefs(ctx context.Context, taskID string) []*agentv1.AttachmentRef {
	if s.Attachments == nil {
		return nil
	}
	list, err := s.Attachments.TaskAttachments(ctx, taskID)
	if err != nil {
		return nil
	}
	refs := make([]*agentv1.AttachmentRef, 0, len(list))
	for _, a := range list {
		refs = append(refs, &agentv1.AttachmentRef{
			Id:          a.ID,
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Size:        a.Size,
			Url:         attachments.URLFor(a.ID),
		})
	}
	return refs
}

func (s *Scheduler) pendingAttachments(
	ctx context.Context,
	task tasks.Task,
) []*agentv1.AttachmentRef {
	if s.Attachments == nil || task.PendingPrompt == "" {
		return nil
	}
	messages, err := s.Attachments.TaskMessages(ctx, task.ID)
	if err != nil {
		return nil
	}

	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if strings.TrimSpace(m.Text) != task.PendingPrompt {
			continue
		}
		if task.PendingPromptActorID != 0 && m.ActorID != task.PendingPromptActorID {
			continue
		}
		refs := make([]*agentv1.AttachmentRef, 0, len(m.Attachments))
		for _, a := range m.Attachments {
			refs = append(refs, &agentv1.AttachmentRef{
				Id:          a.ID,
				Filename:    a.Filename,
				ContentType: a.ContentType,
				Size:        a.Size,
				Url:         attachments.URLFor(a.ID),
			})
		}
		return refs
	}
	return nil
}

func (s *Scheduler) recordInitialMessage(
	ctx context.Context,
	taskID, prompt string,
	actorID ids.ActorID,
	now int64,
) {
	if s.Attachments == nil {
		return
	}

	existing, err := s.Attachments.TaskMessages(ctx, taskID)
	if err != nil || len(existing) > 0 {
		return
	}

	list, err := s.Attachments.TaskAttachments(ctx, taskID)
	if err != nil {
		return
	}
	var ids []int64
	for _, a := range list {
		if a.MessageID == "" {
			ids = append(ids, a.ID)
		}
	}

	if err := s.Attachments.RecordMessage(
		ctx, taskID, uuid.NewString(), prompt, actorID, ids, now,
	); err != nil {
		log.Printf("[scheduler] record initial message %s: %v", taskID, err)
	}
}

func (s *Scheduler) sender(ctx context.Context, actorID ids.ActorID) *agentv1.PromptSender {
	sender, err := s.Tasks.Sender(ctx, actorID)
	if err != nil {
		log.Printf("[scheduler] resolve sender %d: %v", actorID, err)
		return &agentv1.PromptSender{Kind: "system", Username: "system", Name: "CherryPi"}
	}
	return &agentv1.PromptSender{
		ActorId:  int64(sender.ActorID),
		Username: sender.Username,
		Name:     sender.Name,
		Kind:     sender.Kind,
	}
}

func (s *Scheduler) failStart(ctx context.Context, taskID string, cause error) {
	if _, err := s.Tasks.Fail(ctx, taskID, cause.Error()); err != nil {
		log.Printf("[scheduler] fail %s: %v", taskID, err)
	}
}

func (s *Scheduler) freshPrompt(
	ctx context.Context,
	task tasks.Task,
	agent agents.Agent,
) (appendPrompt string, pins prompts.VersionIDs, ok bool) {
	resolved, err := s.Prompts.Resolve(ctx, agent.ID, task.ProjectID, task.TeamIDs...)
	if err != nil {
		log.Printf("[scheduler] resolve prompts %s: %v", task.ID, err)
		_ = s.Tasks.SetError(ctx, task.ID, err.Error())
		return "", prompts.VersionIDs{}, false
	}
	appendPrompt = resolved.Text + s.contextBlock(ctx, task, agent.ID)
	if task.StartedAt.IsZero() {
		pins = resolved.IDs
	}
	return appendPrompt, pins, true
}

func (s *Scheduler) sessionGone(
	ctx context.Context,
	task tasks.Task,
	err error,
) bool {
	if connect.CodeOf(err) != connect.CodeNotFound {
		return false
	}
	log.Printf(
		"[scheduler] agent %d no longer has session %s for %s, recreating: %v",
		task.SessionAgentID, task.SessionID, task.ID, err,
	)
	if _, markErr := s.Tasks.MarkSessionLost(
		ctx,
		task.ID,
		sessionLostReason,
	); markErr != nil {
		log.Printf("[scheduler] mark session lost %s: %v", task.ID, markErr)
	}
	return true
}

func (s *Scheduler) contextBlock(ctx context.Context, task tasks.Task, agentID ids.AgentID) string {
	var b strings.Builder
	b.WriteString("\n\n## CherryPi context\n\n")
	b.WriteString("This session is a CherryPi task. The `listTasks`, `getTask`, " +
		"`getTaskHistory`, `getTaskInput`, `respondToTaskInput`, `createTask`, " +
		"`updateTask`, and `listProjects` tools talk to the CherryPi agency; use " +
		"them to coordinate work when the task calls for it. `listTasks` and " +
		"`getTask` never return a task's conversation; read it with " +
		"`getTaskHistory`. A task you lead may be blocked on a question: read it " +
		"with `getTaskInput` and answer it with `respondToTaskInput`. You cannot " +
		"delete tasks, and you can only see tasks in a project you are a member " +
		"of with the `can_list_tasks` permission. You can always see and update " +
		"your own tasks.\n\n" +
		"Your prompts are versioned. Use `getPrompt` and `listPromptVersions` to " +
		"read your own agent prompt or a project prompt, and `proposePromptChange` " +
		"to suggest a complete replacement; an owner approves it (or your " +
		"self-prompt auto-approve setting applies it immediately). " +
		"`listPromptProposals` shows the status of your proposals.\n\n")

	b.WriteString("### Current task\n\n")
	writeField(&b, "Id", task.ID)
	if task.Name != "" {
		writeField(&b, "Name", task.Name)
	}
	writeField(&b, "State", string(task.State))
	writeField(&b, "Priority", priorityLabel(task.Priority))
	writeField(&b, "Size", task.Size)
	writeField(&b, "Assignee agent", strconv.FormatInt(int64(agentID), 10))
	if names := tagNames(task.Tags); len(names) > 0 {
		writeField(&b, "Tags", strings.Join(names, ", "))
	}
	if len(task.Links) > 0 {
		links := make([]string, len(task.Links))
		for i, l := range task.Links {
			links[i] = l.URL
			if l.Label != "" {
				links[i] = l.Label + " (" + l.URL + ")"
			}
		}
		writeField(&b, "Links", strings.Join(links, ", "))
	}
	if len(task.References) > 0 {
		refs := make([]string, len(task.References))
		for i, r := range task.References {
			suffix := string(r.State)
			if r.Kind == tasks.ReferenceBlocking {
				suffix = "blocking, " + suffix
			}
			refs[i] = r.ID + " (" + suffix + ")"
		}
		writeField(&b, "References", strings.Join(refs, ", "))
	}

	if task.ProjectID == "" {
		return b.String()
	}
	project, err := s.Projects.Get(ctx, task.ProjectID)
	if err != nil {
		return b.String()
	}

	b.WriteString("\n### Current project\n\n")
	writeField(&b, "Id", project.ID)
	writeField(&b, "Name", project.Name)
	writeField(&b, "Slug", project.Slug)
	writeField(&b, "Task prefix", project.Prefix)
	if project.Description != "" {
		writeField(&b, "Description", project.Description)
	}
	if !project.ArchivedAt.IsZero() {
		writeField(&b, "Archived", "yes")
	}
	if perms, member, err := s.Projects.Permissions(ctx, project.ID, agentID); err == nil {
		writeField(&b, "Your permissions", permissionsLabel(member, perms))
	}
	return b.String()
}

func writeField(b *strings.Builder, name, value string) {
	b.WriteString("- **")
	b.WriteString(name)
	b.WriteString(":** ")
	b.WriteString(value)
	b.WriteString("\n")
}

func tagNames(tags []tasks.Tag) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, tag.Name)
	}
	return out
}

func priorityLabel(priority int64) string {
	switch priority {
	case tasks.PriorityLow:
		return "low"
	case tasks.PriorityHigh:
		return "high"
	case tasks.PriorityCritical:
		return "critical"
	default:
		return "normal"
	}
}

func permissionsLabel(member bool, perms projects.Permissions) string {
	if !member {
		return "not a member"
	}
	var out []string
	if perms.CanListTasks {
		out = append(out, "can_list_tasks")
	}
	if perms.CanCreateTasks {
		out = append(out, "can_create_tasks")
	}
	if perms.CanUpdateTasks {
		out = append(out, "can_update_tasks")
	}
	if len(out) == 0 {
		return "member (own tasks only)"
	}
	return "member: " + strings.Join(out, ", ")
}

func taskRef(task tasks.Task) *agentv1.TaskRef {
	if task.ProjectID == "" || task.Seq == 0 {
		return nil
	}
	return &agentv1.TaskRef{ProjectId: task.ProjectID, Seq: task.Seq}
}

func (s *Scheduler) taskModel(
	ctx context.Context,
	task tasks.Task,
	agent agents.Agent,
) (provider, modelID, thinkingLevel string) {
	provider = task.ModelProvider
	modelID = task.ModelID
	thinkingLevel = task.ThinkingLevel
	if provider != "" && modelID != "" && thinkingLevel != "" {
		return provider, modelID, thinkingLevel
	}
	defaults, err := s.Agents.Defaults(ctx, agent.ID)
	if err != nil {
		log.Printf("[scheduler] agent defaults %d: %v", agent.ID, err)
		return provider, modelID, thinkingLevel
	}
	if provider == "" && modelID == "" {
		provider = defaults.ModelProvider
		modelID = defaults.ModelID
	}
	if thinkingLevel == "" {
		thinkingLevel = defaults.ThinkingLevel
	}
	return provider, modelID, thinkingLevel
}

func (s *Scheduler) pluginTools(ctx context.Context, task tasks.Task) []*agentv1.PluginTool {
	if s.Plugins == nil {
		return nil
	}
	var out []*agentv1.PluginTool
	seen := map[string]bool{}
	if row, ok := s.Plugins.TaskPlugin(ctx, task.ID); ok {
		for _, tool := range row.Tools {
			if seen[tool.Name] {
				continue
			}
			seen[tool.Name] = true
			out = append(out, &agentv1.PluginTool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: string(tool.InputSchema),
			})
		}
	}
	for _, projectTool := range s.Plugins.ProjectToolsForTask(
		ctx, task.ID, task.ProjectID,
	) {
		if seen[projectTool.Tool.Name] {
			continue
		}
		seen[projectTool.Tool.Name] = true
		out = append(out, &agentv1.PluginTool{
			Name:        projectTool.Tool.Name,
			Description: projectTool.Tool.Description,
			InputSchema: string(projectTool.Tool.InputSchema),
		})
	}
	return out
}

func (s *Scheduler) start(ctx context.Context, task tasks.Task, agent agents.Agent) {
	now := time.Now().UnixMilli()

	resume := task.SessionID != "" &&
		task.SessionAgentID == agent.ID &&
		task.SessionDeletedAt.IsZero()

	var pins prompts.VersionIDs
	appendPrompt := ""

	if !resume {
		var ok bool
		appendPrompt, pins, ok = s.freshPrompt(ctx, task, agent)
		if !ok {
			return
		}
	}

	claimed, err := s.Tasks.Claim(ctx, task, agent.ActorID, pins, now)
	if err != nil {
		log.Printf("[scheduler] claim %s: %v", task.ID, err)
		return
	}
	if !claimed {
		return
	}
	log.Printf(
		"[scheduler] claimed %s on agent %d (resume=%t session=%q)",
		task.ID, agent.ID, resume, task.SessionID,
	)

	if resume && task.PendingPrompt != "" {
		err := s.AddToTask(
			ctx,
			agent.ID,
			task.SessionID,
			task.PendingPrompt,
			taskRef(task),
			s.sender(ctx, task.PendingPromptActorID),
			s.pendingAttachments(ctx, task),
		)
		if err == nil {
			s.clearDeliveredPrompt(ctx, task.ID, task.PendingPrompt)
			return
		}
		if !s.sessionGone(ctx, task, err) {
			log.Printf("[scheduler] add prompt %s on %d: %v", task.ID, agent.ID, err)
			s.failStart(ctx, task.ID, err)
			return
		}
		resume = false
		var ok bool
		appendPrompt, _, ok = s.freshPrompt(ctx, task, agent)
		if !ok {
			return
		}
	}

	if resume {
		err := s.RetryTask(ctx, agent.ID, task.SessionID, taskRef(task))
		if err == nil {
			return
		}
		if !s.sessionGone(ctx, task, err) {
			log.Printf("[scheduler] retry %s on %d: %v", task.ID, agent.ID, err)
			s.failStart(ctx, task.ID, err)
			return
		}
		var ok bool
		appendPrompt, _, ok = s.freshPrompt(ctx, task, agent)
		if !ok {
			return
		}
	}

	prompt := task.Prompt
	promptActorID := task.CreatorActorID
	if task.PendingPrompt != "" {
		prompt = task.PendingPrompt
		promptActorID = task.PendingPromptActorID
	}

	s.recordInitialMessage(ctx, task.ID, prompt, promptActorID, now)

	modelProvider, modelID, thinkingLevel := s.taskModel(ctx, task, agent)

	interactive := task.Interactive
	sessionID, err := s.CreateTask(ctx, agent.ID, &agentv1.CreateTaskRequest{
		Prompt:             prompt,
		Workspace:          task.Workspace,
		Interactive:        &interactive,
		ModelProvider:      modelProvider,
		ModelId:            modelID,
		ThinkingLevel:      thinkingLevel,
		Title:              task.Name,
		AutoTitle:          task.AutoTitle,
		AppendSystemPrompt: appendPrompt,
		Attachments:        s.attachmentRefs(ctx, task.ID),
		TaskId:             task.ID,
		Ref:                taskRef(task),
		Sender:             s.sender(ctx, promptActorID),
		Tools:              s.pluginTools(ctx, task),
	})
	if err != nil {
		log.Printf("[scheduler] start %s on %d: %v", task.ID, agent.ID, err)
		s.failStart(ctx, task.ID, err)
		return
	}
	if err := s.Tasks.SetSession(ctx, task.ID, agent.ID, sessionID); err != nil {
		log.Printf("[scheduler] store session %s: %v", task.ID, err)
	}
	log.Printf("[scheduler] started %s on agent %d session=%s", task.ID, agent.ID, sessionID)
	if task.PendingPrompt != "" {
		s.clearDeliveredPrompt(ctx, task.ID, task.PendingPrompt)
	}
}
