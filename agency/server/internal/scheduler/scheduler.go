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
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"errors"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/limits"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
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

func (s *Scheduler) Drain(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	queued, err := s.Tasks.ListQueued(ctx)
	if err != nil {
		log.Printf("[scheduler] list queued: %v", err)
		return
	}

	for _, task := range queued {
		if ctx.Err() != nil {
			return
		}
		agent, ok := s.pickAgent(ctx, task)
		if !ok {
			continue
		}
		s.start(ctx, task, agent)
	}
}

func (s *Scheduler) Reconcile(ctx context.Context) {
	if s.AgentTasks == nil {
		return
	}
	running, err := s.Tasks.ListRunning(ctx)
	if err != nil {
		log.Printf("[scheduler] list running: %v", err)
		return
	}

	byActor := map[ids.ActorID][]tasks.Task{}
	for _, task := range running {
		if task.SessionID == "" || task.AssigneeActorID == 0 {
			continue
		}
		byActor[task.AssigneeActorID] = append(byActor[task.AssigneeActorID], task)
	}

	for actorID, agentTasks := range byActor {
		if ctx.Err() != nil {
			return
		}
		agent, err := s.Agents.GetByActor(ctx, actorID)
		if err != nil {
			continue
		}
		live, err := s.AgentTasks(ctx, agent.ID)
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
				continue
			}

			switch {
			case at.GetWaitingForInput():
				if task.State != tasks.StateInput {
					if _, err := s.Tasks.MarkInput(ctx, task.ID); err != nil {
						log.Printf("[scheduler] mark input %s: %v", task.ID, err)
					}
				}
			case at.GetRunInProgress():
				if task.State == tasks.StateInput {
					if _, err := s.Tasks.ClearInput(ctx, task.ID); err != nil {
						log.Printf("[scheduler] clear input %s: %v", task.ID, err)
					}
				}
			default:
				if _, err := s.Tasks.Settle(ctx, task.ID); err != nil {
					log.Printf("[scheduler] settle %s: %v", task.ID, err)
				}
			}
		}
	}
}

func (s *Scheduler) pickAgent(ctx context.Context, task tasks.Task) (agents.Agent, bool) {
	candidates := s.candidates(ctx, task)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })

	for _, agent := range candidates {
		if s.Connected == nil || !s.Connected(agent.ID) {
			continue
		}
		member, err := s.Tasks.IsMember(ctx, task.ProjectID, agent.ID)
		if err != nil {
			log.Printf("[scheduler] membership %s/%d: %v", task.ProjectID, agent.ID, err)
			continue
		}
		if !member {
			continue
		}
		lim, err := s.Limits.Get(ctx, agent.ID)
		if err != nil {
			log.Printf("[scheduler] limits %d: %v", agent.ID, err)
			continue
		}
		counts, err := s.Tasks.RunningCounts(ctx, agent.ID)
		if err != nil {
			log.Printf("[scheduler] running counts %d: %v", agent.ID, err)
			continue
		}
		if tasks.Fits(lim, counts, task.ProjectID, task.Tags) {
			return agent, true
		}
	}
	return agents.Agent{}, false
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

func (s *Scheduler) contextBlock(ctx context.Context, task tasks.Task, agentID ids.AgentID) string {
	var b strings.Builder
	b.WriteString("\n\n## CherryPi context\n\n")
	b.WriteString("This session is a CherryPi task. The `listTasks`, `getTask`, " +
		"`getTaskHistory`, `createTask`, `updateTask`, and `listProjects` tools " +
		"talk to the CherryPi agency; use them to coordinate work when the task " +
		"calls for it. `listTasks` and `getTask` never return a task's " +
		"conversation; read it with `getTaskHistory`. You cannot delete tasks, and " +
		"you can only see tasks in a project you are a member of with the " +
		"`can_list_tasks` permission. You can always see and update your own tasks.\n\n")

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

func (s *Scheduler) start(ctx context.Context, task tasks.Task, agent agents.Agent) {
	now := time.Now().UnixMilli()

	resume := task.SessionID != "" &&
		task.SessionAgentID == agent.ID &&
		task.SessionDeletedAt.IsZero()

	var pins prompts.VersionIDs
	appendPrompt := ""

	if !resume {
		resolved, err := s.Prompts.Resolve(ctx, agent.ID, task.ProjectID, task.TeamIDs...)
		if err != nil {
			log.Printf("[scheduler] resolve prompts %s: %v", task.ID, err)
			_ = s.Tasks.SetError(ctx, task.ID, err.Error())
			return
		}
		appendPrompt = resolved.Text + s.contextBlock(ctx, task, agent.ID)
		if task.StartedAt.IsZero() {
			pins = resolved.IDs
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

	if resume && task.PendingPrompt != "" {
		if err := s.AddToTask(
			ctx,
			agent.ID,
			task.SessionID,
			task.PendingPrompt,
			taskRef(task),
			s.sender(ctx, task.PendingPromptActorID),
			s.pendingAttachments(ctx, task),
		); err != nil {
			log.Printf("[scheduler] add prompt %s on %d: %v", task.ID, agent.ID, err)
			s.failStart(ctx, task.ID, err)
			return
		}
		if err := s.Tasks.ClearPendingPrompt(ctx, task.ID); err != nil {
			log.Printf("[scheduler] clear pending prompt %s: %v", task.ID, err)
		}
		return
	}

	if resume {
		if err := s.RetryTask(ctx, agent.ID, task.SessionID, taskRef(task)); err != nil {
			log.Printf("[scheduler] retry %s on %d: %v", task.ID, agent.ID, err)
			s.failStart(ctx, task.ID, err)
		}
		return
	}

	prompt := task.Prompt
	promptActorID := task.CreatorActorID
	if task.PendingPrompt != "" {
		prompt = task.PendingPrompt
		promptActorID = task.PendingPromptActorID
	}

	s.recordInitialMessage(ctx, task.ID, prompt, promptActorID, now)

	modelProvider, modelID, thinkingLevel := s.taskModel(ctx, task, agent)

	sessionID, err := s.CreateTask(ctx, agent.ID, &agentv1.CreateTaskRequest{
		Prompt:             prompt,
		Workspace:          task.Workspace,
		Interactive:        true,
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
	})
	if err != nil {
		log.Printf("[scheduler] start %s on %d: %v", task.ID, agent.ID, err)
		s.failStart(ctx, task.ID, err)
		return
	}
	if err := s.Tasks.SetSession(ctx, task.ID, agent.ID, sessionID); err != nil {
		log.Printf("[scheduler] store session %s: %v", task.ID, err)
	}
	if task.PendingPrompt != "" {
		if err := s.Tasks.ClearPendingPrompt(ctx, task.ID); err != nil {
			log.Printf("[scheduler] clear pending prompt %s: %v", task.ID, err)
		}
	}
}
