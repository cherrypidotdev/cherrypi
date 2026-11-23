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
	"log"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"

	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/limits"
	"cherrypi.vladfaust.com/agency/internal/modelrouting"
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

	ModelRouting *modelrouting.Service

	Interval time.Duration

	SettleGrace time.Duration

	StartRefusalCooldown time.Duration

	Connected  func(agentID ids.AgentID) bool
	CreateTask func(
		ctx context.Context,
		agentID ids.AgentID,
		req *agentv1.CreateTaskRequest,
	) (sessionID string, err error)

	RetryTask func(
		ctx context.Context,
		agentID ids.AgentID, sessionID string,
		ref *agentv1.TaskRef,
		requiredTools []string,
		model ModelSelection,
	) error

	AddToTask func(
		ctx context.Context,
		agentID ids.AgentID, sessionID, prompt string,
		ref *agentv1.TaskRef,
		sender *agentv1.PromptSender,
		atts []*agentv1.AttachmentRef,
		requiredTools []string,
		model ModelSelection,
	) error

	AgentTasks func(ctx context.Context, agentID ids.AgentID) ([]*agentv1.Task, error)

	Rand func(n int) int

	Now func() time.Time

	wake chan struct{}
	mu   sync.Mutex

	waiting map[string]string

	coolMu  sync.Mutex
	cooling map[ids.AgentID]cooldown
}

type cooldown struct {
	until  time.Time
	window time.Duration
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
		cooling:  map[ids.AgentID]cooldown{},
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
			requiredTools []string,
			model ModelSelection,
		) error {
			conn, err := hub.Connection(agentID)
			if err != nil {
				return err
			}
			_, err = conn.Tasks.RetryTask(ctx, &agentv1.RetryTaskRequest{
				Id:            sessionID,
				Ref:           ref,
				RequiredTools: requiredTools,
				ModelProvider: model.Provider,
				ModelId:       model.ModelID,
				ThinkingLevel: model.ThinkingLevel,
			})
			return err
		}
		s.AddToTask = func(
			ctx context.Context,
			agentID ids.AgentID, sessionID, prompt string,
			ref *agentv1.TaskRef,
			sender *agentv1.PromptSender,
			atts []*agentv1.AttachmentRef,
			requiredTools []string,
			model ModelSelection,
		) error {
			conn, err := hub.Connection(agentID)
			if err != nil {
				return err
			}
			_, err = conn.Tasks.AddToTask(ctx, &agentv1.AddToTaskRequest{
				Id:            sessionID,
				Prompt:        prompt,
				Attachments:   atts,
				Ref:           ref,
				Sender:        sender,
				RequiredTools: requiredTools,
				ModelProvider: model.Provider,
				ModelId:       model.ModelID,
				ThinkingLevel: model.ThinkingLevel,
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
	force bool,
) error {
	if strings.TrimSpace(prompt) == "" {
		return errors.New("the message is empty")
	}

	if _, err := s.Tasks.FollowUp(ctx, tasks.FollowUpParams{
		TaskID:  t.ID,
		Text:    prompt,
		ActorID: actorID,
		Force:   force,
	}); err != nil {
		return err
	}
	s.Wake()
	return nil
}

func (s *Scheduler) DeliverPendingPrompt(ctx context.Context, t tasks.Task) bool {
	if strings.TrimSpace(t.PendingPrompt) == "" {
		return false
	}
	if t.SessionID == "" || s.AddToTask == nil {
		return false
	}
	err := s.AddToTask(
		ctx, t.SessionAgentID, t.SessionID, t.PendingPrompt,
		taskRef(t), s.PendingSender(ctx, t),
		s.pendingAttachments(ctx, t), t.RequiredTools,
		s.ResumeModel(ctx, t, t.SessionAgentID),
	)
	if err != nil {
		log.Printf("[scheduler] deliver pending prompt %s: %v", t.ID, err)
		return false
	}
	s.clearDeliveredPrompt(ctx, t.ID, t.PendingPrompt)
	return true
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

func (s *Scheduler) settleGrace() time.Duration {
	if s.SettleGrace != 0 {
		return s.SettleGrace
	}
	return tasks.SettleGrace
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
					errMsg = tasks.RunFailedReason
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

				if task.RunMayBeStarting(time.Now(), s.settleGrace()) {
					continue
				}
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
	if task.State != tasks.StateRunning && task.State != tasks.StateInput {
		return
	}

	requeued, err := s.Tasks.RequeueAfterSessionLost(ctx, task.ID)
	if err != nil {
		log.Printf("[scheduler] requeue lost session %s: %v", task.ID, err)
	}
	if requeued {
		log.Printf(
			"[scheduler] re-queued %s after its session was lost, with a "+
				"follow-up waiting", task.ID,
		)
		s.Wake()
		return
	}
	failReason := sessionLostReason +
		"; re-queue or follow up to recreate it"
	if _, err := s.Tasks.Fail(ctx, task.ID, failReason); err != nil {
		log.Printf("[scheduler] fail lost session %s: %v", task.ID, err)
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

	agentCooling

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
	if s.coolingDown(agent.ID) {
		return agentCheck{verdict: agentCooling}, nil
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

	if owner, ok := warmSessionOwner(task); ok {
		for i, agent := range candidates {
			if agent.ID != owner {
				continue
			}
			check, err := s.checkAgent(ctx, task, agent, view)
			if err != nil {
				log.Printf("[scheduler] check agent %d for %s: %v", agent.ID, task.ID, err)
			} else if check.verdict == agentReady {
				return agent, true
			}

			candidates = append(candidates[:i], candidates[i+1:]...)
			break
		}
	}

	s.shuffle(candidates)
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

func warmSessionOwner(task tasks.Task) (ids.AgentID, bool) {
	if task.SessionID == "" || task.SessionAgentID == 0 ||
		!task.SessionDeletedAt.IsZero() {
		return 0, false
	}
	return task.SessionAgentID, true
}

func (s *Scheduler) shuffle(list []agents.Agent) {
	for i := len(list) - 1; i > 0; i-- {
		j := s.randomIndex(i + 1)
		list[i], list[j] = list[j], list[i]
	}
}

func (s *Scheduler) randomIndex(n int) int {
	if s.Rand != nil {
		return s.Rand(n)
	}
	return rand.IntN(n)
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
			"Waiting for blocking tasks to complete or be cancelled: "+
				strings.Join(blockers, ", "),
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
	offline, cooling := false, false
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
		case agentCooling:
			cooling = true
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
	if cooling {
		return "Waiting for an agent that is restarting"
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
	return AttachmentRefs(s.Attachments, ctx, taskID)
}

func AttachmentRefs(
	svc *attachments.Service,
	ctx context.Context,
	taskID string,
) []*agentv1.AttachmentRef {
	if svc == nil {
		return nil
	}
	list, err := svc.TaskAttachments(ctx, taskID)
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
	queued, err := s.Tasks.PendingMessages(ctx, task.ID)
	if err != nil {
		return nil
	}
	messageIDs := map[string]bool{}
	for _, message := range queued {
		if message.MessageID != "" {
			messageIDs[message.MessageID] = true
		}
	}
	all, err := s.Attachments.TaskAttachments(ctx, task.ID)
	if err != nil {
		return nil
	}
	refs := make([]*agentv1.AttachmentRef, 0, len(all))
	for _, a := range all {
		if !messageIDs[a.MessageID] {
			continue
		}
		refs = append(refs, &agentv1.AttachmentRef{
			Id:          a.ID,
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Size:        a.Size,
			Url:         attachments.URLFor(a.ID),
		})
	}
	if len(refs) > 0 {
		return refs
	}
	return s.pendingAttachmentsByText(ctx, task)
}

func (s *Scheduler) pendingAttachmentsByText(
	ctx context.Context,
	task tasks.Task,
) []*agentv1.AttachmentRef {
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
	task tasks.Task,
	prompt string,
	actorID ids.ActorID,
	now int64,
) {
	if s.Attachments == nil {
		return
	}
	existing, err := s.Attachments.TaskMessages(ctx, task.ID)
	if err != nil {
		return
	}
	for _, message := range existing {
		if strings.TrimSpace(message.Text) == strings.TrimSpace(prompt) {
			return
		}
	}
	if task.StartedAt.IsZero() && !task.CreatedAt.IsZero() {
		now = task.CreatedAt.UnixMilli()
	}

	list, err := s.Attachments.TaskAttachments(ctx, task.ID)
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
		ctx, task.ID, uuid.NewString(), prompt, actorID, ids, now,
	); err != nil {
		log.Printf("[scheduler] record initial message %s: %v", task.ID, err)
	}
}

func (s *Scheduler) PendingSender(
	ctx context.Context,
	task tasks.Task,
) *agentv1.PromptSender {
	sender := s.sender(ctx, task.PendingPromptActorID)
	if queued, err := s.Tasks.PendingMessages(ctx, task.ID); err == nil &&
		len(queued) == 1 {
		sender.MessageId = queued[0].MessageID
	}
	return sender
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

const startRefusalCooldown = 10 * time.Second

const maxStartRefusalCooldown = time.Minute

func (s *Scheduler) coolingDown(agentID ids.AgentID) bool {
	s.coolMu.Lock()
	defer s.coolMu.Unlock()
	cool, ok := s.cooling[agentID]
	if !ok {
		return false
	}
	return s.now().Before(cool.until)
}

func (s *Scheduler) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Scheduler) coolDown(agentID ids.AgentID) {
	base := s.StartRefusalCooldown
	if base <= 0 {
		base = startRefusalCooldown
	}
	base = min(base, maxStartRefusalCooldown)
	s.coolMu.Lock()
	if s.cooling == nil {
		s.cooling = map[ids.AgentID]cooldown{}
	}
	window := base
	if prev := s.cooling[agentID]; prev.window > 0 {
		window = min(prev.window*2, maxStartRefusalCooldown)
	}
	s.cooling[agentID] = cooldown{
		until:  s.now().Add(window),
		window: window,
	}
	s.coolMu.Unlock()
}

func (s *Scheduler) WarmAgent(agentID ids.AgentID) {
	s.coolMu.Lock()
	delete(s.cooling, agentID)
	s.coolMu.Unlock()
}

func (s *Scheduler) failStart(ctx context.Context, taskID string, cause error) {
	if _, err := s.Tasks.Fail(ctx, taskID, cause.Error()); err != nil {
		log.Printf("[scheduler] fail %s: %v", taskID, err)
	}
}

func (s *Scheduler) deferStart(
	ctx context.Context,
	taskID string,
	agentID ids.AgentID,
	cause error,
) bool {
	if connect.CodeOf(cause) != connect.CodeUnavailable {
		return false
	}
	log.Printf(
		"[scheduler] start %s refused by agent %d, retrying later: %v",
		taskID, agentID, cause,
	)
	s.coolDown(agentID)
	if err := s.Tasks.RequeueAfterStartFailure(
		ctx, taskID, cause.Error(),
	); err != nil {
		log.Printf("[scheduler] requeue deferred start %s: %v", taskID, err)
	}
	return true
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
		"`getTaskHistory`, `getTaskInput`, `updateTask`, and `listProjects` tools " +
		"talk to the CherryPi agency; use them to coordinate work when the task " +
		"calls for it. `listTasks` and `getTask` never return a task's " +
		"conversation; read it with `getTaskHistory`. Create work with " +
		"`queue_child_task` when this task must wait for it, or " +
		"`create_backlog_task` when it must not. A task you lead may be blocked " +
		"on a question: read it with `getTaskInput` and answer it with " +
		"`respondToTaskInput`. An automated reaction answers its origin task " +
		"with `answer_origin_task_questions`, `complete_origin_task`, or " +
		"`cancel_origin_task`. You cannot delete tasks, and you can only see " +
		"tasks in a project you are a member of with the `can_list_tasks` " +
		"permission. You can always see and update your own tasks.\n\n" +
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

func routePriority(priority int64) int64 {
	switch priority {
	case tasks.PriorityLow:
		return 1
	case tasks.PriorityNormal:
		return 2
	case tasks.PriorityHigh:
		return 3
	case tasks.PriorityCritical:
		return 4
	default:
		return 0
	}
}

type ModelSelection struct {
	Provider      string
	ModelID       string
	ThinkingLevel string
}

func (s *Scheduler) taskModel(
	ctx context.Context,
	task tasks.Task,
	agentID ids.AgentID,
) ModelSelection {
	sel := ModelSelection{
		Provider:      task.ModelProvider,
		ModelID:       task.ModelID,
		ThinkingLevel: task.ThinkingLevel,
	}
	if sel.Provider != "" && sel.ModelID != "" && sel.ThinkingLevel != "" {
		return sel
	}

	if sel.Provider == "" && sel.ModelID == "" && s.ModelRouting != nil {
		tags := make([]string, len(task.Tags))
		for i, tag := range task.Tags {
			tags[i] = tag.Name
		}
		selection, ok, err := s.ModelRouting.Resolve(ctx, modelrouting.ResolveInput{
			ProjectID: task.ProjectID,
			Priority:  routePriority(task.Priority),
			Size:      task.Size,
			Tags:      tags,
		})
		if err != nil {
			log.Printf("[scheduler] model routing for task %s: %v", task.ID, err)
		} else if ok {
			sel.Provider = selection.Provider
			sel.ModelID = selection.ModelID
			if sel.ThinkingLevel == "" {
				sel.ThinkingLevel = selection.ThinkingLevel
			}
		}
	}

	defaults, err := s.Agents.Defaults(ctx, agentID)
	if err != nil {
		log.Printf("[scheduler] agent defaults %d: %v", agentID, err)
		return sel
	}
	if sel.Provider == "" && sel.ModelID == "" {
		sel.Provider = defaults.ModelProvider
		sel.ModelID = defaults.ModelID
	}
	if sel.ThinkingLevel == "" {
		sel.ThinkingLevel = defaults.ThinkingLevel
	}
	return sel
}

func (s *Scheduler) ResumeModel(
	ctx context.Context,
	task tasks.Task,
	agentID ids.AgentID,
) ModelSelection {
	state, ok, err := s.Tasks.SessionState(ctx, task.ID)
	if err != nil {
		log.Printf("[scheduler] resume model for %s: %v", task.ID, err)
	} else if ok && state.ModelProvider != "" && state.ModelID != "" {
		sel := ModelSelection{
			Provider:      state.ModelProvider,
			ModelID:       state.ModelID,
			ThinkingLevel: state.ThinkingLevel,
		}
		if sel.ThinkingLevel == "" {

			sel.ThinkingLevel = s.taskModel(ctx, task, agentID).ThinkingLevel
		}
		return sel
	}
	return s.taskModel(ctx, task, agentID)
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
			s.PendingSender(ctx, task),
			s.pendingAttachments(ctx, task),
			task.RequiredTools,
			s.ResumeModel(ctx, task, agent.ID),
		)
		if err == nil {
			s.WarmAgent(agent.ID)
			s.clearDeliveredPrompt(ctx, task.ID, task.PendingPrompt)
			return
		}
		if !s.sessionGone(ctx, task, err) {
			log.Printf("[scheduler] add prompt %s on %d: %v", task.ID, agent.ID, err)
			if !s.deferStart(ctx, task.ID, agent.ID, err) {
				s.failStart(ctx, task.ID, err)
			}
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
		err := s.RetryTask(
			ctx, agent.ID, task.SessionID, taskRef(task), task.RequiredTools,
			s.ResumeModel(ctx, task, agent.ID),
		)
		if err == nil {
			s.WarmAgent(agent.ID)
			return
		}
		if !s.sessionGone(ctx, task, err) {
			log.Printf("[scheduler] retry %s on %d: %v", task.ID, agent.ID, err)
			if !s.deferStart(ctx, task.ID, agent.ID, err) {
				s.failStart(ctx, task.ID, err)
			}
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
	deliveringQueue := false
	if task.PendingPrompt != "" && !task.StartedAt.IsZero() {
		prompt = task.PendingPrompt
		promptActorID = task.PendingPromptActorID
		deliveringQueue = true
	}

	s.recordInitialMessage(ctx, task, prompt, promptActorID, now)

	var model ModelSelection
	if task.StartedAt.IsZero() {
		model = s.taskModel(ctx, task, agent.ID)
	} else {
		model = s.ResumeModel(ctx, task, agent.ID)
	}

	interactive := !task.Automated

	sender := s.sender(ctx, promptActorID)
	if deliveringQueue {
		sender = s.PendingSender(ctx, task)
	}
	sessionID, err := s.CreateTask(ctx, agent.ID, &agentv1.CreateTaskRequest{
		Prompt:             prompt,
		Workspace:          task.Workspace,
		Interactive:        &interactive,
		ModelProvider:      model.Provider,
		ModelId:            model.ModelID,
		ThinkingLevel:      model.ThinkingLevel,
		Title:              task.Name,
		AutoTitle:          task.AutoTitle,
		AppendSystemPrompt: appendPrompt,
		Attachments:        s.attachmentRefs(ctx, task.ID),
		TaskId:             task.ID,
		Ref:                taskRef(task),
		Sender:             sender,
		Tools:              s.pluginTools(ctx, task),
		RequiredTools:      task.RequiredTools,
	})
	if err != nil {
		log.Printf("[scheduler] start %s on %d: %v", task.ID, agent.ID, err)
		if !s.deferStart(ctx, task.ID, agent.ID, err) {
			s.failStart(ctx, task.ID, err)
		}
		return
	}
	if err := s.Tasks.SetSession(ctx, task.ID, agent.ID, sessionID); err != nil {
		log.Printf("[scheduler] store session %s: %v", task.ID, err)
	}
	s.WarmAgent(agent.ID)
	log.Printf("[scheduler] started %s on agent %d session=%s", task.ID, agent.ID, sessionID)
	if deliveringQueue {
		s.clearDeliveredPrompt(ctx, task.ID, task.PendingPrompt)
	}
}
