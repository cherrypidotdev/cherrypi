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
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/limits"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/textutil"
)

type State string

const (
	StateBacklog   State = "backlog"
	StateWaiting   State = "waiting"
	StateQueued    State = "queued"
	StateRunning   State = "running"
	StateInput     State = "input"
	StateSettled   State = "settled"
	StateFailed    State = "failed"
	StateCompleted State = "completed"
	StateCancelled State = "cancelled"
)

func HoldsSlot(state State) bool {
	return state == StateRunning
}

func CountsTtc(state State) bool {
	return state != StateBacklog && state != StateCompleted &&
		state != StateCancelled
}

type Resolution string

const (
	ResolutionCompleted Resolution = "completed"
	ResolutionCancelled Resolution = "cancelled"
)

const (
	PriorityLow      int64 = -1
	PriorityNormal   int64 = 0
	PriorityHigh     int64 = 1
	PriorityCritical int64 = 2
)

const (
	SizeSmall  = "small"
	SizeMedium = "medium"
	SizeBig    = "big"
	SizeEpic   = "epic"
)

var (
	ErrNotFound = errors.New("task not found")

	ErrInvalidPrompt = errors.New("prompt must be non-empty")

	ErrInvalidModel = errors.New("model provider and id must be set together")

	ErrProjectRequired = errors.New("project is required")

	ErrNotEditable = errors.New("only a backlog task that has not started can be edited")

	ErrWrongState = errors.New("task is not in a valid state for this action")

	ErrRunning = errors.New("task is running; stop it or let it settle first")

	ErrPendingPrompt = errors.New(
		"task has an undelivered follow-up; close it out once it has run",
	)

	ErrTagMissing = errors.New("tag not found")

	ErrBacklogNotStarted = errors.New(
		"a backlog task that has not started does not accept a follow-up; edit its prompt or queue it first",
	)

	ErrNotMember = errors.New("agent is not a member of the project")

	ErrAssigneeViewer = errors.New(
		"assignee is a viewer of the project and cannot run tasks; " +
			"grant it the member role first",
	)

	ErrAssigneeMissingTag = errors.New(
		"assignee does not declare required agent tags",
	)

	ErrProjectHasNoAgents = errors.New(
		"project has no member agents; grant one the member role before queueing the task",
	)

	ErrSelfReference = errors.New("a task cannot reference itself")

	ErrReferenceCycle = errors.New("blocking reference would create a cycle")

	ErrReferenceNotAllowed = errors.New(
		"a task may only reference tasks in its own, an ancestor, a descendant, or a sibling project",
	)

	ErrMergeTooFew = errors.New("select at least two tasks to merge")

	ErrMergeProjectMismatch = errors.New("tasks to merge must share one project")

	ErrMergeAssigneeConflict = errors.New("tasks to merge have conflicting assignees")

	ErrMergeState = errors.New("only backlog, queued, or failed tasks can be merged")

	ErrTeamsRequired = errors.New("at least one owning team is required")

	ErrTeamNotInProject = errors.New("owning team is not attached to the project")

	ErrLeadNotAllowed = errors.New(
		"a leader must be a project member or belong to one of the task's " +
			"owning teams",
	)

	ErrLeadRequired = errors.New(
		"a task needs a leader who is a project member or a member of an " +
			"owning team",
	)

	ErrAutomatedLeader = errors.New("an automated task cannot name a leader")

	ErrAutomatedBacklog = errors.New("an automated task cannot be in backlog")

	ErrCannotPrompt = errors.New("not allowed to prompt this task")

	ErrCannotCloseOut = errors.New(
		"only the leader or an administrator may close out or archive this task",
	)

	ErrHoldNotFound = errors.New("task hold not found")

	ErrFollowUpRequired = errors.New(
		"task has already started; send a follow-up prompt to queue it",
	)

	ErrActorRequired = errors.New("an authenticated actor is required")
)

type Tag struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	Color       string
	Archived    bool

	Required bool
}

type Link struct {
	URL   string
	Label string
}

type ReferenceKind string

const (
	ReferenceBlocking      ReferenceKind = "blocking"
	ReferenceInformational ReferenceKind = "informational"
	ReferenceSupersedes    ReferenceKind = "supersedes"

	ReferenceBlockedBy ReferenceKind = "blocked_by"

	ReferenceBlocks ReferenceKind = "blocks"

	ReferenceSupersededBy ReferenceKind = "superseded_by"
)

func referenceEdge(
	kind ReferenceKind,
	taskID, referencedID string,
) (sourceID, targetID string, stored ReferenceKind) {
	switch kind {
	case ReferenceBlocks:
		return referencedID, taskID, ReferenceBlocking
	case ReferenceSupersededBy:
		return referencedID, taskID, ReferenceSupersedes
	case ReferenceBlocking, ReferenceBlockedBy:
		return taskID, referencedID, ReferenceBlocking
	case ReferenceSupersedes:
		return taskID, referencedID, ReferenceSupersedes
	default:
		return taskID, referencedID, ReferenceInformational
	}
}

func isInverseReferenceKind(kind ReferenceKind) bool {
	return kind == ReferenceBlocks || kind == ReferenceSupersededBy
}

func perspectiveKind(kind ReferenceKind, incoming bool) ReferenceKind {
	switch kind {
	case ReferenceBlocking:
		if incoming {
			return ReferenceBlocks
		}
		return ReferenceBlockedBy
	case ReferenceSupersedes:
		if incoming {
			return ReferenceSupersededBy
		}
		return ReferenceSupersedes
	default:
		return ReferenceInformational
	}
}

type Reference struct {
	ID     string
	Name   string
	Prompt string
	State  State
	Kind   ReferenceKind

	Priority int64
	Size     string

	Tags []Tag

	WakeOnResolveWhenMany bool
}

func (r Reference) Blocking() bool { return r.Kind == ReferenceBlocking }

type ReferenceInput struct {
	TaskID string
	Kind   ReferenceKind

	WakeOnResolveWhenMany bool
}

type Task struct {
	RowID     int64
	ID        string
	Seq       int64
	Prompt    string
	Name      string
	ProjectID string

	OriginTaskID    string
	AssigneeActorID ids.ActorID

	CreatorActorID ids.ActorID

	TeamIDs []int64

	LeaderActorID ids.ActorID

	Automated bool

	Reaction bool

	RequiredTools []string

	NeedsOperator string

	ReactionReminders int

	Subscribed bool

	Favorite bool

	UnreadComments int
	AssigneePinned bool
	Priority       int64
	Size           string
	State          State
	SessionID      string

	SessionAgentID ids.AgentID
	Error          string
	ModelProvider  string
	ModelID        string
	ThinkingLevel  string
	Workspace      string
	AutoTitle      bool
	PromptIDs      prompts.VersionIDs
	Tags           []Tag
	Links          []Link
	Progress       Progress
	References     []Reference

	ReferencedBy []Reference

	PendingSteering string

	CanPrompt bool

	Blockers int

	Holds []TaskHold

	PendingPrompt string

	PendingPromptActorID ids.ActorID
	QueuedAt             time.Time
	StartedAt            time.Time
	SettledAt            time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time

	StateUpdatedAt time.Time

	TtcMs int64

	SessionDeletedAt time.Time

	Archived   bool
	ArchivedAt time.Time

	Template *TemplateApplication
}

type TemplateApplication struct {
	TemplateID      string
	TemplateName    string
	TemplateVersion int64
	ArgsJSON        string
}

type Filter struct {
	ProjectID string
	AgentID   ids.AgentID
	State     State
	TagID     string
	Search    string

	Size string

	Priority *int64

	Archived bool

	IncludeArchived bool

	Automated *bool

	ReferenceProjectID string

	Limit  int64
	Offset int64

	VisibleToActorID ids.ActorID

	VisibleToAdmin bool
}

type CreateParams struct {
	Prompt    string
	Name      string
	ProjectID string

	OriginTaskID    string
	AssigneeActorID ids.ActorID
	TagIDs          []string
	References      []ReferenceInput
	Priority        int64
	Size            string
	ModelProvider   string
	ModelID         string
	ThinkingLevel   string
	Workspace       string
	AutoTitle       bool
	State           State

	TeamIDs []int64

	LeaderActorID ids.ActorID

	Automated bool

	Reaction bool

	Template *TemplateApplication
}

type UpdateParams struct {
	Prompt          string
	Name            string
	ProjectID       string
	AssigneeActorID ids.ActorID
	TagIDs          []string
	Priority        int64
	Size            string
	ModelProvider   string
	ModelID         string
	ThinkingLevel   string
	Workspace       string
	AutoTitle       bool
}

type ModelOverride struct {
	Provider      string
	ModelID       string
	ThinkingLevel string
}

type AttributeSet struct {
	Priority        *int64
	Size            *string
	AssigneeActorID *ids.ActorID

	Prompt *string

	Model *ModelOverride

	PendingPrompt *string

	AddTagIDs    []string
	RemoveTagIDs []string
}

func (a AttributeSet) Empty() bool {
	return a.Priority == nil && a.Size == nil && a.AssigneeActorID == nil &&
		a.Prompt == nil &&
		a.Model == nil &&
		a.PendingPrompt == nil &&
		len(a.AddTagIDs) == 0 && len(a.RemoveTagIDs) == 0
}

func (a AttributeSet) changesTags() bool {
	return len(a.AddTagIDs) > 0 || len(a.RemoveTagIDs) > 0
}

type AttributeResult struct {
	ID    string
	Task  Task
	Error error
}

type FinishOutcome string

const (
	OutcomeSettled   FinishOutcome = "settled"
	OutcomeFailed    FinishOutcome = "failed"
	OutcomeCompleted FinishOutcome = "completed"
	OutcomeCancelled FinishOutcome = "cancelled"
)

type WakeState struct {
	Lines []string          `json:"lines,omitempty"`
	Keys  map[string]string `json:"keys,omitempty"`
	Count int               `json:"count,omitempty"`

	PluginWakes map[string]int `json:"plugin_wakes,omitempty"`

	DeliveredPrompt string `json:"delivered_prompt,omitempty"`

	ContinuedAt int64 `json:"continued_at,omitempty"`

	InputWakes []InputWake `json:"input_wakes,omitempty"`

	WakeAfter int64 `json:"wake_after,omitempty"`
}

type InputWake struct {
	ChildID   string `json:"child_id"`
	RequestID string `json:"request_id,omitempty"`
}

const MaxInputWakes = 8

func (w WakeState) SeenInputWake(childID, requestID string) bool {
	for _, seen := range w.InputWakes {
		if seen.ChildID == childID && seen.RequestID == requestID {
			return true
		}
	}
	return false
}

func (w *WakeState) RememberInputWake(childID, requestID string) {
	w.InputWakes = append(w.InputWakes, InputWake{
		ChildID:   childID,
		RequestID: requestID,
	})
	if over := len(w.InputWakes) - MaxInputWakes; over > 0 {
		w.InputWakes = w.InputWakes[over:]
	}
}

func parseWakeState(raw string) WakeState {
	var st WakeState
	if strings.TrimSpace(raw) == "" {
		return WakeState{Keys: map[string]string{}}
	}
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		return WakeState{Keys: map[string]string{}}
	}
	if st.Keys == nil {
		st.Keys = map[string]string{}
	}
	return st
}

func (w WakeState) JSON() string {
	raw, err := json.Marshal(w)
	if err != nil {
		return ""
	}
	return string(raw)
}

type Transition struct {
	TaskID string
	State  State

	Held bool

	Holds []TaskHold
}

type Service struct {
	store    store.Store
	projects *projects.Service

	Limits *limits.Service

	OnTransition func(ctx context.Context, t Transition)

	OnChange func(ctx context.Context, taskID string)

	OnComment func(ctx context.Context, taskID string, comment Comment)

	OnMention func(ctx context.Context, taskID string, comment Comment)

	OnCreated func(ctx context.Context, taskID string)

	OnTaskFinished func(ctx context.Context, t Task, outcome FinishOutcome)

	OnInputCleared func(ctx context.Context, t Task)

	DeliverPendingPrompt func(ctx context.Context, t Task) bool

	OnPendingPromptQueued func(ctx context.Context, taskID string)

	OnHoldReleased func(ctx context.Context, release HoldRelease)

	DeleteAgentSession func(
		ctx context.Context,
		agentID ids.AgentID,
		sessionID string,
	) error

	handoffLocks [handoffLockStripes]sync.Mutex
}

func New(st store.Store, projectsSvc *projects.Service) *Service {
	return &Service{store: st, projects: projectsSvc}
}

func (s *Service) transitioned(ctx context.Context, id string, state State) {
	if s.OnTransition != nil {
		s.OnTransition(ctx, Transition{TaskID: id, State: state})
	}
}

func (s *Service) changed(ctx context.Context, id string) {
	if s.OnChange != nil && id != "" {
		s.OnChange(ctx, id)
	}
}

func (s *Service) Create(ctx context.Context, p CreateParams) (Task, error) {

	if p.Reaction {
		p.Automated = true
	}
	prompt := strings.TrimSpace(p.Prompt)
	if prompt == "" {
		return Task{}, ErrInvalidPrompt
	}
	if strings.TrimSpace(p.ProjectID) == "" {
		return Task{}, ErrProjectRequired
	}

	project, err := s.validateScope(ctx, p.ProjectID, p.AssigneeActorID)
	if err != nil {
		return Task{}, err
	}
	if err := s.ValidateTags(ctx, project.ID, p.TagIDs); err != nil {
		return Task{}, err
	}
	if err := s.validateAssigneeTagDeclarations(
		ctx,
		project.ID,
		p.AssigneeActorID,
		p.TagIDs,
	); err != nil {
		return Task{}, err
	}
	teamIDs, err := s.resolveTaskTeams(ctx, project.ID, p.TeamIDs)
	if err != nil {
		return Task{}, err
	}

	if p.Automated && p.LeaderActorID != 0 {
		return Task{}, ErrAutomatedLeader
	}
	initial := normalizeCreateState(p.State)

	if p.Automated {
		initial = StateQueued
	}
	if initial == StateQueued {
		if err := s.requireExecutor(ctx, project.ID, p.AssigneeActorID); err != nil {
			return Task{}, err
		}
	}

	leaderActorID, err := s.resolveCreateLeader(
		ctx, project.ID, teamIDs, p.Automated, p.LeaderActorID,
	)
	if err != nil {
		return Task{}, err
	}
	if !p.Automated && leadRequired(initial) &&
		actorIDFrom(ctx) != 0 && leaderActorID == 0 {
		return Task{}, ErrLeadRequired
	}

	refs, err := s.resolveReferences(ctx, project.ID, p.References)
	if err != nil {
		return Task{}, err
	}

	originID := strings.TrimSpace(p.OriginTaskID)
	if originID != "" {
		if _, err := s.store.GetTask(ctx, originID); errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		} else if err != nil {
			return Task{}, err
		}
	}
	number, err := s.store.AllocateTaskNumber(ctx, project.ID)
	if err != nil {
		return Task{}, err
	}

	now := time.Now().UnixMilli()
	row, err := s.store.CreateTask(ctx, store.CreateTaskParams{
		ID:              TaskID(project.Prefix, number),
		Seq:             number,
		Prompt:          prompt,
		Name:            strings.TrimSpace(p.Name),
		ProjectID:       project.ID,
		OriginTaskID:    originID,
		AssigneeActorID: nullID(p.AssigneeActorID),
		AssigneePinned:  boolInt(p.AssigneeActorID != 0),
		Priority:        p.Priority,
		CreatorActorID:  nullID(actorIDFrom(ctx)),
		Size:            normalizeSize(p.Size),
		State:           string(StateBacklog),
		ModelProvider:   p.ModelProvider,
		ModelID:         p.ModelID,
		ThinkingLevel:   p.ThinkingLevel,
		Automated:       boolInt(p.Automated),
		Reaction:        boolInt(p.Reaction),
		Workspace:       p.Workspace,
		AutoTitle:       boolInt(p.AutoTitle),
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return Task{}, err
	}

	if p.Template != nil {
		if _, err := s.store.CreateTaskTemplateApplication(
			ctx,
			store.TaskTemplateApplicationParams{
				TaskID:          row.ID,
				TemplateID:      p.Template.TemplateID,
				TemplateVersion: p.Template.TemplateVersion,
				TemplateName:    p.Template.TemplateName,
				ArgsJSON:        p.Template.ArgsJSON,
				CreatedAt:       now,
			},
		); err != nil {
			return Task{}, err
		}
	}
	if err := s.SetTags(ctx, row.ID, p.TagIDs); err != nil {
		return Task{}, err
	}
	if err := s.setTaskTeams(ctx, row.ID, teamIDs); err != nil {
		return Task{}, err
	}
	if err := s.setTaskLeader(ctx, project.ID, row.ID, leaderActorID); err != nil {
		return Task{}, err
	}

	if creator := actorIDFrom(ctx); creator != 0 {
		if err := s.store.AddTaskSubscription(ctx, store.AddTaskSubscriptionParams{
			TaskID:    row.ID,
			ActorID:   int64(creator),
			CreatedAt: now,
		}); err != nil {
			return Task{}, err
		}
	}

	internal := withoutLogging(ctx)
	for _, ref := range refs {
		if _, err := s.addReference(
			internal, row.ID, ref.TaskID, ref.Kind, ref.WakeOnResolveWhenMany, true,
		); err != nil {
			return Task{}, err
		}
	}

	switch initial {
	case StateQueued:
		created, err := s.snapshot(ctx, row.ID)
		if err != nil {
			return Task{}, err
		}
		if created.State == StateBacklog {
			if _, err := s.Queue(internal, row.ID); err != nil {
				return Task{}, err
			}
		}
	}
	t, err := s.Get(ctx, row.ID)
	if err != nil {
		return Task{}, err
	}
	if t.State == StateWaiting {
		ctx = WithStateReason(ctx, s.blockedReason(ctx, t))
	}
	s.logCreated(ctx, t)
	if s.OnCreated != nil {
		s.OnCreated(ctx, t.ID)
	}
	return t, nil
}

func normalizeCreateState(state State) State {
	switch state {
	case StateBacklog, StateQueued:
		return state
	default:
		return StateQueued
	}
}

func (s *Service) resolveReferences(
	ctx context.Context,
	fromProjectID string,
	in []ReferenceInput,
) ([]ReferenceInput, error) {
	out := make([]ReferenceInput, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, ref := range in {
		id := strings.TrimSpace(ref.TaskID)
		if id == "" || seen[id] {
			continue
		}
		target, err := s.store.GetTask(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		} else if err != nil {
			return nil, err
		}
		kind := normalizeReferenceKind(ref.Kind)

		if err := s.checkReferenceProject(
			ctx,
			fromProjectID,
			target.ProjectID.String,
		); err != nil {
			return nil, err
		}
		seen[id] = true
		out = append(out, ReferenceInput{
			TaskID:                id,
			Kind:                  kind,
			WakeOnResolveWhenMany: ref.WakeOnResolveWhenMany,
		})
	}
	return out, nil
}

func (s *Service) checkReferenceProject(
	ctx context.Context,
	fromProjectID, toProjectID string,
) error {
	if fromProjectID == "" || toProjectID == "" || fromProjectID == toProjectID {
		return nil
	}
	ok, err := s.projects.ReferenceAllowed(ctx, fromProjectID, toProjectID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %s cannot reference %s", ErrReferenceNotAllowed,
			fromProjectID, toProjectID)
	}
	return nil
}

func normalizeReferenceKind(kind ReferenceKind) ReferenceKind {
	switch kind {
	case ReferenceBlocking, ReferenceBlockedBy, ReferenceBlocks,
		ReferenceSupersedes, ReferenceSupersededBy, ReferenceInformational:
		return kind
	default:
		return ReferenceInformational
	}
}

func (s *Service) Get(ctx context.Context, id string) (Task, error) {
	row, err := s.store.GetTask(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}
	tags, err := s.tagsForTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	links, err := s.linksForTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	refs, err := s.referencesForTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	referencedBy, err := s.referencedByForTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	progress, err := s.progressForTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	t := toTask(row, tags, links, refs, referencedBy, progress)
	holds, err := s.Holds(ctx, id)
	if err != nil {
		return Task{}, err
	}
	t.Holds = holds
	application, err := s.store.GetTaskTemplateApplication(ctx, id)
	if err == nil {
		t.Template = &TemplateApplication{
			TemplateID:      application.TemplateID.String,
			TemplateName:    application.TemplateName,
			TemplateVersion: application.TemplateVersion,
			ArgsJSON:        application.ArgsJson,
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Task{}, err
	}
	teams, err := s.store.ListTaskTeamsByTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	t.TeamIDs = teams
	teamPrompts, err := s.store.ListTaskTeamPromptsByTask(ctx, id)
	if err != nil {
		return Task{}, err
	}

	for _, pin := range teamPrompts {
		t.PromptIDs.Teams = append(t.PromptIDs.Teams, prompts.TeamVersionIDs{
			TeamID:        pin.TeamID,
			Prompt:        pin.PromptID.Int64,
			ProjectPrompt: pin.ProjectPromptID.Int64,
		})
	}
	groupPrompts, err := s.store.ListTaskGroupPromptsByTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	groupTeamPrompts, err := s.store.ListTaskGroupTeamPromptsByTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	for _, pin := range groupPrompts {
		t.PromptIDs.Groups = append(t.PromptIDs.Groups, prompts.GroupVersionIDs{
			GroupID:       pin.GroupID,
			Prompt:        pin.PromptID.Int64,
			ProjectPrompt: pin.ProjectPromptID.Int64,
		})
	}
	for _, pin := range groupTeamPrompts {
		for i := range t.PromptIDs.Groups {
			if t.PromptIDs.Groups[i].GroupID != pin.GroupID {
				continue
			}
			t.PromptIDs.Groups[i].Teams = append(t.PromptIDs.Groups[i].Teams,
				prompts.GroupTeamVersionIDs{
					TeamID: pin.TeamID,
					Prompt: pin.PromptID.Int64,
				})
		}
	}
	leaders, err := s.store.ListTaskLeadsByTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if len(leaders) > 0 {
		t.LeaderActorID = ids.ActorID(leaders[0])
	}
	if actorID := actorIDFrom(ctx); actorID != 0 {
		subscribed, err := s.store.IsActorSubscribedToTask(
			ctx,
			store.IsActorSubscribedToTaskParams{
				TaskID:  id,
				ActorID: int64(actorID),
			},
		)
		if err != nil {
			return Task{}, err
		}
		t.Subscribed = subscribed
		favorite, err := s.store.IsActorFavoriteTask(
			ctx,
			store.IsActorFavoriteTaskParams{
				TaskID:  id,
				ActorID: int64(actorID),
			},
		)
		if err != nil {
			return Task{}, err
		}
		t.Favorite = favorite
		unread, err := s.UnreadComments(ctx, id, actorID)
		if err != nil {
			return Task{}, err
		}
		t.UnreadComments = unread
	}
	return t, nil
}

func (s *Service) OriginChain(ctx context.Context, id string) ([]Task, error) {
	seen := map[string]struct{}{id: {}}
	next := id
	var chain []Task
	for {
		row, err := s.store.GetTask(ctx, next)
		if errors.Is(err, sql.ErrNoRows) {
			return chain, nil
		}
		if err != nil {
			return nil, err
		}
		origin := row.OriginTaskID.String
		if origin == "" {
			break
		}
		if _, ok := seen[origin]; ok {
			break
		}
		seen[origin] = struct{}{}
		originRow, err := s.store.GetTask(ctx, origin)
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return nil, err
		}
		chain = append(chain, toTask(originRow, nil, nil, nil, nil, Progress{}))
		next = origin
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

func (s *Service) OriginVisible(
	ctx context.Context,
	id string,
	actorID ids.ActorID,
	isAdmin bool,
) (bool, error) {
	row, err := s.store.GetTask(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return s.CanView(ctx, row.ProjectID.String, actorID, isAdmin)
}

func (s *Service) GetByProjectSeq(
	ctx context.Context,
	projectID string,
	seq int64,
) (Task, error) {
	row, err := s.store.GetTaskByProjectSeq(ctx, projectID, seq)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}
	return s.Get(ctx, row.ID)
}

func (s *Service) snapshot(ctx context.Context, id string) (Task, error) {
	row, err := s.store.GetTask(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}
	return toTask(row, nil, nil, nil, nil, emptyProgress()), nil
}

func (s *Service) List(ctx context.Context, f Filter) ([]Task, error) {

	if strings.TrimSpace(f.ProjectID) != "" {
		if id, err := s.ResolveProjectID(ctx, f.ProjectID); err == nil {
			f.ProjectID = id
		}
	}
	rows, err := s.store.ListTasks(ctx, store.ListTasksParams{
		ProjectID:          nullString(f.ProjectID),
		AgentID:            nullID(f.AgentID),
		State:              nullString(string(f.State)),
		Archived:           boolInt(f.Archived),
		IncludeArchived:    boolInt(f.IncludeArchived),
		Automated:          nullBoolInt64Ptr(f.Automated),
		TagID:              nullString(f.TagID),
		Search:             nullString(f.Search),
		Size:               nullString(f.Size),
		Priority:           nullInt64Ptr(f.Priority),
		ReferenceProjectID: nullString(f.ReferenceProjectID),
		PageLimit:          nullPositive(f.Limit),
		PageOffset:         f.Offset,
	})
	if err != nil {
		return nil, err
	}
	if f.VisibleToActorID != 0 && !f.VisibleToAdmin && s.projects != nil {
		visibleProjects, err := s.projects.ProjectsForActor(ctx, f.VisibleToActorID)
		if err != nil {
			return nil, err
		}
		visible := make(map[string]bool, len(visibleProjects))
		for _, project := range visibleProjects {
			visible[project.ID] = true
		}
		kept := rows[:0]
		for _, row := range rows {

			own := row.AssigneeActorID.Valid &&
				ids.ActorID(row.AssigneeActorID.Int64) == f.VisibleToActorID
			if !own {
				own = row.CreatorActorID.Valid &&
					ids.ActorID(row.CreatorActorID.Int64) == f.VisibleToActorID
			}
			if !row.ProjectID.Valid || row.ProjectID.String == "" ||
				visible[row.ProjectID.String] || own {
				kept = append(kept, row)
			}
		}
		rows = kept
	}
	tagsByTask, err := s.allTags(ctx)
	if err != nil {
		return nil, err
	}
	linksByTask, err := s.allLinks(ctx)
	if err != nil {
		return nil, err
	}
	progressByTask, err := s.allProgress(ctx)
	if err != nil {
		return nil, err
	}
	refsByTask, err := s.allReferences(ctx, tagsByTask)
	if err != nil {
		return nil, err
	}
	teamsByTask, err := s.allTaskTeams(ctx)
	if err != nil {
		return nil, err
	}
	leadsByTask, err := s.allTaskLeads(ctx)
	if err != nil {
		return nil, err
	}
	holdsByTask, err := s.allHolds(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(
			row,
			tagsByTask[row.ID],
			linksByTask[row.ID],
			refsByTask[row.ID],
			nil,
			progressByTask[row.ID],
		)
		out[i].TeamIDs = teamsByTask[row.ID]
		out[i].LeaderActorID = firstActorID(leadsByTask[row.ID])
		out[i].Holds = holdsByTask[row.ID]
	}

	if actorID := actorIDFrom(ctx); actorID != 0 {
		favoriteIDs, err := s.store.ListFavoriteTaskIDs(ctx, actorID)
		if err != nil {
			return nil, err
		}
		favorites := make(map[string]struct{}, len(favoriteIDs))
		for _, id := range favoriteIDs {
			favorites[id] = struct{}{}
		}
		for i := range out {
			_, out[i].Favorite = favorites[out[i].ID]
		}
		if len(out) > 0 {
			unread, err := s.store.ListUnreadTaskCommentCounts(ctx, actorID)
			if err != nil {
				return nil, err
			}
			counts := make(map[string]int, len(unread))
			for _, row := range unread {
				counts[row.TaskID] = int(row.Unread)
			}
			for i := range out {
				out[i].UnreadComments = counts[out[i].ID]
			}
		}
	}
	if err := s.fillPromptPolicy(
		ctx, out, f.VisibleToActorID, f.VisibleToAdmin, teamsByTask, leadsByTask,
	); err != nil {
		return nil, err
	}

	prioritizeSearchMatches(out, f.Search)
	return out, nil
}

func (s *Service) fillPromptPolicy(
	ctx context.Context,
	list []Task,
	actorID ids.ActorID,
	isAdmin bool,
	teamsByTask map[string][]int64,
	leadsByTask map[string][]int64,
) error {
	if len(list) == 0 {
		return nil
	}
	if isAdmin {
		for i := range list {
			list[i].CanPrompt = true
		}
		return nil
	}
	if actorID == 0 {
		return nil
	}
	rows, err := s.store.ListTeamsForActor(ctx, actorID)
	if err != nil {
		return err
	}
	member := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		member[row.ID] = struct{}{}
	}
	projectAllowed := map[string]bool{}
	for i := range list {
		task := &list[i]
		if task.CreatorActorID == actorID {
			task.CanPrompt = true
			continue
		}
		teams := teamsByTask[task.ID]
		if len(teams) == 0 {
			if s.projects == nil || strings.TrimSpace(task.ProjectID) == "" {
				continue
			}
			allowed, ok := projectAllowed[task.ProjectID]
			if !ok {
				allowed, err = s.projects.ActorCanPrompt(
					ctx, task.ProjectID, actorID,
				)
				if err != nil {
					return err
				}
				projectAllowed[task.ProjectID] = allowed
			}
			task.CanPrompt = allowed
			continue
		}
		inTeam := false
		for _, team := range teams {
			if _, ok := member[team]; ok {
				inTeam = true
				break
			}
		}
		if inTeam {
			task.CanPrompt = true
			continue
		}
		for _, lead := range leadsByTask[task.ID] {
			if ids.ActorID(lead) == actorID {
				task.CanPrompt = true
				break
			}
		}
	}
	return nil
}

func prioritizeSearchMatches(list []Task, search string) {
	if search == "" {
		return
	}
	digits := isDigits(search)
	rank := func(t Task) int {
		if strings.EqualFold(t.ID, search) {
			return 0
		}
		if !digits {
			return 3
		}
		number := strconv.FormatInt(t.Seq, 10)
		switch {
		case number == search:
			return 1
		case strings.Contains(number, search):
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		return rank(list[i]) < rank(list[j])
	})
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *Service) ListQueued(ctx context.Context) ([]Task, error) {
	rows, err := s.store.ListQueuedTasks(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []Task{}, nil
	}
	tagsByTask, err := s.allTags(ctx)
	if err != nil {
		return nil, err
	}
	linksByTask, err := s.allLinks(ctx)
	if err != nil {
		return nil, err
	}
	refsByTask, err := s.allReferences(ctx, tagsByTask)
	if err != nil {
		return nil, err
	}
	teamsByTask, err := s.allTaskTeams(ctx)
	if err != nil {
		return nil, err
	}
	leadsByTask, err := s.allTaskLeads(ctx)
	if err != nil {
		return nil, err
	}
	holdsByTask, err := s.allHolds(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(
			row,
			tagsByTask[row.ID],
			linksByTask[row.ID],
			refsByTask[row.ID],
			nil,
			emptyProgress(),
		)
		out[i].TeamIDs = teamsByTask[row.ID]
		out[i].LeaderActorID = firstActorID(leadsByTask[row.ID])
		out[i].Holds = holdsByTask[row.ID]
	}
	return out, nil
}

func (s *Service) ListRunning(ctx context.Context) ([]Task, error) {
	rows, err := s.store.ListRunningTasks(ctx)
	if err != nil {
		return nil, err
	}
	tagsByTask, err := s.allTags(ctx)
	if err != nil {
		return nil, err
	}
	linksByTask, err := s.allLinks(ctx)
	if err != nil {
		return nil, err
	}
	refsByTask, err := s.allReferences(ctx, tagsByTask)
	if err != nil {
		return nil, err
	}
	teamsByTask, err := s.allTaskTeams(ctx)
	if err != nil {
		return nil, err
	}
	leadsByTask, err := s.allTaskLeads(ctx)
	if err != nil {
		return nil, err
	}
	holdsByTask, err := s.allHolds(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(
			row,
			tagsByTask[row.ID],
			linksByTask[row.ID],
			refsByTask[row.ID],
			nil,
			emptyProgress(),
		)
		out[i].TeamIDs = teamsByTask[row.ID]
		out[i].LeaderActorID = firstActorID(leadsByTask[row.ID])
		out[i].Holds = holdsByTask[row.ID]
	}
	return out, nil
}

func (s *Service) ListSessionTasks(ctx context.Context) ([]Task, error) {
	rows, err := s.store.ListSessionTasks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(row, nil, nil, nil, nil, emptyProgress())
	}
	return out, nil
}

func (s *Service) AutoArchivable(
	ctx context.Context,
	orgDays int,
	now time.Time,
) ([]Task, error) {
	var rows []store.Task
	if orgDays > 0 {
		inherited, err := s.store.ListInheritedAutoArchivableTasks(
			ctx,
			now.AddDate(0, 0, -orgDays).UnixMilli(),
		)
		if err != nil {
			return nil, err
		}
		rows = inherited
	}
	overridden, err := s.store.ListOverriddenAutoArchivableTasks(ctx, now.UnixMilli())
	if err != nil {
		return nil, err
	}
	rows = append(rows, overridden...)
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(row, nil, nil, nil, nil, emptyProgress())
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (Task, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if cur.State != StateBacklog || !cur.StartedAt.IsZero() {
		return Task{}, ErrNotEditable
	}
	prompt := strings.TrimSpace(p.Prompt)
	if prompt == "" {
		return Task{}, ErrInvalidPrompt
	}
	if strings.TrimSpace(p.ProjectID) == "" {
		return Task{}, ErrProjectRequired
	}

	project, err := s.validateScope(ctx, p.ProjectID, p.AssigneeActorID)
	if err != nil {
		return Task{}, err
	}
	if err := s.ValidateTags(ctx, project.ID, p.TagIDs); err != nil {
		return Task{}, err
	}
	if err := s.validateAssigneeTagDeclarations(
		ctx,
		project.ID,
		p.AssigneeActorID,
		p.TagIDs,
	); err != nil {
		return Task{}, err
	}

	newID := cur.ID
	number := cur.Seq
	if project.ID != cur.ProjectID {
		number, err = s.store.AllocateTaskNumber(ctx, project.ID)
		if err != nil {
			return Task{}, err
		}
		newID = TaskID(project.Prefix, number)
	}

	if _, err := s.store.UpdateBacklogTask(ctx, store.UpdateBacklogTaskParams{
		NewID:           newID,
		Seq:             number,
		Prompt:          prompt,
		Name:            strings.TrimSpace(p.Name),
		ProjectID:       project.ID,
		AssigneeActorID: nullID(p.AssigneeActorID),
		AssigneePinned:  boolInt(p.AssigneeActorID != 0),
		Priority:        p.Priority,
		Size:            normalizeSize(p.Size),
		ModelProvider:   p.ModelProvider,
		ModelID:         p.ModelID,
		ThinkingLevel:   p.ThinkingLevel,
		Workspace:       p.Workspace,
		AutoTitle:       boolInt(p.AutoTitle),
		UpdatedAt:       time.Now().UnixMilli(),
		ID:              id,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotEditable
		}
		return Task{}, err
	}
	if err := s.SetTags(ctx, newID, p.TagIDs); err != nil {
		return Task{}, err
	}
	after, err := s.Get(ctx, newID)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) Queue(ctx context.Context, id string) (Task, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}

	if len(cur.Holds) > 0 {
		return Task{}, heldError(cur.Holds)
	}
	if cur.State == StateWaiting {
		return cur, nil
	}
	waiting, err := s.hasIncompleteDependencies(ctx, cur)
	if err != nil {
		return Task{}, err
	}
	switch cur.State {
	case StateBacklog:
		if !cur.StartedAt.IsZero() {
			return Task{}, ErrFollowUpRequired
		}
		if err := s.requireExecutor(ctx, cur.ProjectID, cur.AssigneeActorID); err != nil {
			return Task{}, err
		}
	case StateRunning, StateInput, StateSettled, StateFailed:
		if !waiting {
			return Task{}, ErrFollowUpRequired
		}
	default:
		return Task{}, ErrWrongState
	}
	if err := s.validateAssigneeTagDeclarations(
		ctx,
		cur.ProjectID,
		cur.AssigneeActorID,
		tagIDsOf(cur.Tags),
	); err != nil {
		return Task{}, err
	}
	if err := s.requireLeader(ctx, id, StateQueued); err != nil {
		return Task{}, err
	}
	now := time.Now().UnixMilli()
	event := store.TaskQueue
	if waiting {
		event = store.TaskWait

		ctx = WithStateReason(ctx, s.blockedReason(ctx, cur))
	}
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     event,
		QueuedAt:  sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return Task{}, err
	}
	if n == 0 {
		return Task{}, ErrWrongState
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) hasIncompleteDependencies(ctx context.Context, t Task) (bool, error) {
	return len(IncompleteBlockerIDs(t.References)) > 0, nil
}

func (s *Service) blockedReason(ctx context.Context, t Task) string {
	blockers := IncompleteBlockerIDs(t.References)
	if len(blockers) == 0 {
		return "Blocked"
	}
	failed := make([]string, 0, len(blockers))
	for _, ref := range t.References {
		if ref.Kind == ReferenceBlocking && ref.State == StateFailed {
			failed = append(failed, ref.ID)
		}
	}
	reason := "Blocked on blocking task" + plural(len(blockers)) + ": " +
		strings.Join(blockers, ", ")
	if len(failed) > 0 {
		reason += " (" + strings.Join(failed, ", ") +
			" failed and will not resolve on its own; retry it or drop the edge)"
	}
	return reason
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func (s *Service) ReconcileWaiting(ctx context.Context, id string) error {
	before, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	switch before.State {
	case StateRunning, StateInput, StateBacklog,
		StateCompleted, StateCancelled:
		return nil
	}
	if before.State == StateWaiting && len(before.Holds) > 0 {
		return nil
	}
	waiting, err := s.hasIncompleteDependencies(ctx, before)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	var event store.TaskTransition
	var reason string
	switch {
	case waiting && (before.State == StateQueued ||
		before.State == StateSettled || before.State == StateFailed):
		event = store.TaskWait
		reason = s.blockedReason(ctx, before)
	case !waiting && before.State == StateWaiting:
		event = store.TaskUnwait
		reason = "Woke up: every blocking task resolved"
	default:
		return nil
	}
	ctx = WithStateReason(ctx, reason)
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     event,
		QueuedAt:  sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil || n == 0 {
		return err
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	s.logChanges(ctx, before, after)
	return nil
}

func (s *Service) Dequeue(ctx context.Context, id string) (Task, error) {
	cur, err := s.requireStates(ctx, id, StateQueued, StateWaiting)
	if err != nil {
		return Task{}, err
	}
	if cur.Automated {
		return Task{}, ErrAutomatedBacklog
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(
		ctx,
		store.TransitionTaskParams{Event: store.TaskDequeue, UpdatedAt: now, ID: id},
	)
	if err != nil {
		return Task{}, err
	}
	if n == 0 {
		return Task{}, ErrWrongState
	}
	if err := s.ClearHolds(ctx, id, HoldClearedByDequeue); err != nil {
		log.Printf("[tasks] clear holds after dequeue %s: %v", id, err)
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) References(ctx context.Context, taskID string) ([]Task, error) {
	if _, err := s.Get(ctx, taskID); err != nil {
		return nil, err
	}
	rows, err := s.store.ListTaskReferences(ctx, taskID)
	if err != nil {
		return nil, err
	}
	tagsByTask, err := s.allTags(ctx)
	if err != nil {
		return nil, err
	}
	linksByTask, err := s.allLinks(ctx)
	if err != nil {
		return nil, err
	}
	refsByTask, err := s.allReferences(ctx, tagsByTask)
	if err != nil {
		return nil, err
	}
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(
			row,
			tagsByTask[row.ID],
			linksByTask[row.ID],
			refsByTask[row.ID],
			nil,
			emptyProgress(),
		)
	}
	return out, nil
}

func (s *Service) AddReference(
	ctx context.Context,
	taskID, referencedID string,
	kind ReferenceKind,
) (Task, error) {
	return s.addReference(ctx, taskID, referencedID, kind, false, true)
}

func (s *Service) addReference(
	ctx context.Context,
	taskID, referencedID string,
	kind ReferenceKind,
	wakeOnResolveWhenMany bool,
	rewire bool,
) (Task, error) {
	kind = normalizeReferenceKind(kind)
	cur, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	if taskID == referencedID {
		return Task{}, ErrSelfReference
	}
	sourceID, targetID, stored := referenceEdge(kind, taskID, referencedID)
	source, err := s.Get(ctx, sourceID)
	if err != nil {
		return Task{}, err
	}
	target, err := s.Get(ctx, targetID)
	if err != nil {
		return Task{}, err
	}
	if err := s.checkReferenceProject(ctx, source.ProjectID, target.ProjectID); err != nil {
		return Task{}, err
	}
	if stored == ReferenceBlocking {
		cycle, err := s.hasBlockingReferencePath(ctx, targetID, sourceID)
		if err != nil {
			return Task{}, err
		}
		if cycle {
			return Task{}, ErrReferenceCycle
		}
	}
	if err := s.store.AddTaskReference(ctx, store.UpsertTaskReferenceParams{
		TaskID:                sourceID,
		ReferencedTaskID:      targetID,
		Kind:                  string(stored),
		WakeOnResolveWhenMany: boolInt(wakeOnResolveWhenMany && stored == ReferenceBlocking),
		CreatedAt:             time.Now().UnixMilli(),
	}); err != nil {
		return Task{}, err
	}
	if stored == ReferenceSupersedes {
		if err := s.cancelSuperseded(ctx, targetID); err != nil {
			return Task{}, err
		}
	}
	after, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	if isInverseReferenceKind(kind) {

		s.logIncomingReference(ctx, taskID, Reference{
			ID:     source.ID,
			Name:   source.Name,
			Prompt: source.Prompt,
			State:  source.State,
		}, stored, true)
	}
	if stored == ReferenceSupersedes && rewire {
		if err := s.rewireBlockers(ctx, targetID, sourceID); err != nil {
			return Task{}, err
		}
	}
	if err := s.ReconcileWaiting(ctx, sourceID); err != nil {
		return Task{}, err
	}
	return s.Get(ctx, taskID)
}

func (s *Service) rewireBlockers(
	ctx context.Context,
	supersededID, replacementID string,
) error {
	if supersededID == replacementID {
		return nil
	}
	edges, err := s.store.ListAllBlockingTaskReferences(ctx)
	if err != nil {
		return err
	}
	for _, edge := range edges {
		if edge.ReferencedTaskID != supersededID || edge.TaskID == replacementID {
			continue
		}
		referrer, err := s.snapshot(ctx, edge.TaskID)
		if err != nil {
			return err
		}
		if referrer.Archived {
			continue
		}
		switch referrer.State {
		case StateCompleted, StateCancelled:
			continue
		}
		if _, err := s.RemoveReference(ctx, edge.TaskID, supersededID); err != nil {
			return err
		}
		if _, err := s.AddReference(
			ctx, edge.TaskID, replacementID, ReferenceBlocking,
		); err != nil {
			if errors.Is(err, ErrReferenceCycle) {
				continue
			}
			return err
		}
	}
	return nil
}

func (s *Service) cancelSuperseded(ctx context.Context, id string) error {
	t, err := s.snapshot(ctx, id)
	if err != nil {
		return err
	}
	switch t.State {
	case StateBacklog, StateQueued, StateFailed:
		_, err = s.Cancel(ctx, id)
		return err
	default:
		return nil
	}
}

func (s *Service) RemoveReference(ctx context.Context, taskID, referencedID string) (Task, error) {
	return s.removeReferenceEdge(ctx, taskID, referencedID, taskID)
}

func (s *Service) RemoveIncomingReference(
	ctx context.Context,
	taskID, referencedID string,
) (Task, error) {
	return s.removeReferenceEdge(ctx, referencedID, taskID, taskID)
}

func (s *Service) removeReferenceEdge(
	ctx context.Context,
	sourceID, targetID, viewID string,
) (Task, error) {
	cur, err := s.Get(ctx, viewID)
	if err != nil {
		return Task{}, err
	}
	var removed *Reference
	if viewID != sourceID {
		for _, ref := range cur.ReferencedBy {
			if ref.ID == sourceID {
				peer := ref
				removed = &peer
				break
			}
		}
	}
	if _, err := s.store.RemoveTaskReference(ctx, store.RemoveTaskReferenceParams{
		TaskID:           sourceID,
		ReferencedTaskID: targetID,
	}); err != nil {
		return Task{}, err
	}
	after, err := s.Get(ctx, viewID)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	if removed != nil {

		s.logIncomingReference(ctx, viewID, *removed, removed.Kind, false)
	}
	if err := s.ReconcileWaiting(ctx, sourceID); err != nil {
		return Task{}, err
	}
	return s.Get(ctx, viewID)
}

func (s *Service) SetReferences(
	ctx context.Context,
	taskID string,
	in []ReferenceInput,
) (Task, error) {
	cur, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	target, err := s.resolveReferences(ctx, cur.ProjectID, in)
	if err != nil {
		return Task{}, err
	}
	want := make(map[string]ReferenceInput, len(target))
	for _, ref := range target {
		want[ref.TaskID] = ref
	}
	for _, ref := range cur.References {
		if _, ok := want[ref.ID]; !ok {
			if _, err := s.RemoveReference(ctx, taskID, ref.ID); err != nil {
				return Task{}, err
			}
		}
	}
	for _, ref := range target {
		if _, err := s.addReference(
			ctx, taskID, ref.TaskID, ref.Kind, ref.WakeOnResolveWhenMany, true,
		); err != nil {
			return Task{}, err
		}
	}
	return s.Get(ctx, taskID)
}

func (s *Service) hasBlockingReferencePath(ctx context.Context, from, to string) (bool, error) {
	edges, err := s.store.ListAllBlockingTaskReferences(ctx)
	if err != nil {
		return false, err
	}
	adj := make(map[string][]string, len(edges))
	for _, e := range edges {
		adj[e.TaskID] = append(adj[e.TaskID], e.ReferencedTaskID)
	}
	seen := map[string]bool{from: true}
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			return true, nil
		}
		for _, next := range adj[cur] {
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false, nil
}

func (s *Service) Archive(ctx context.Context, id string) (Task, error) {
	cur, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if cur.State != StateCompleted && cur.State != StateCancelled {
		return Task{}, ErrWrongState
	}
	if cur.Archived {
		return s.snapshot(ctx, id)
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:      store.TaskArchive,
		ArchivedAt: sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt:  now,
		ID:         id,
	})
	if err != nil {
		return Task{}, err
	}
	if n == 0 {
		return Task{}, ErrWrongState
	}
	if err := s.ClearHolds(ctx, id, HoldClearedByArchive); err != nil {
		log.Printf("[tasks] clear holds after archive %s: %v", id, err)
	}
	if err := s.clearRequiredTools(ctx, cur); err != nil {
		log.Printf("[tasks] clear required tools after archive %s: %v", id, err)
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.recordLog(
		ctx, actorFrom(ctx), now, after.ID, LogState, "archive",
		"", "", "Task archived", nil,
	)
	return after, nil
}

func (s *Service) DeleteSessionData(ctx context.Context, id string) error {
	return s.store.DeleteTaskSessionData(ctx, id)
}

func (s *Service) Unarchive(ctx context.Context, id string) (Task, error) {
	cur, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if !cur.Archived {
		return cur, nil
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(
		ctx,
		store.TransitionTaskParams{Event: store.TaskUnarchive, UpdatedAt: now, ID: id},
	)
	if err != nil {
		return Task{}, err
	}
	if n == 0 {
		return Task{}, ErrWrongState
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.recordLog(
		ctx, actorFrom(ctx), now, after.ID, LogState, "archive",
		"", "", "Task unarchived", nil,
	)
	return after, nil
}

const mergePromptSeparator = "\n\n---\n\n"

func (s *Service) Merge(ctx context.Context, rawIDs []string) (Task, error) {
	mergees := make([]Task, 0, len(rawIDs))
	seen := make(map[string]bool, len(rawIDs))
	for _, raw := range rawIDs {
		id := strings.TrimSpace(raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		t, err := s.Get(ctx, id)
		if err != nil {
			return Task{}, err
		}
		mergees = append(mergees, t)
	}
	if len(mergees) < 2 {
		return Task{}, ErrMergeTooFew
	}
	projectID := mergees[0].ProjectID
	if projectID == "" {
		return Task{}, ErrProjectRequired
	}
	assignee := mergees[0].AssigneeActorID
	priority := mergees[0].Priority
	sizeRank := sizeRankOf(mergees[0].Size)
	tags := make(map[string]bool)
	tagIDs := make([]string, 0)
	prompts := make([]string, 0, len(mergees))

	teamSeen := make(map[int64]bool)
	teamIDs := make([]int64, 0)
	var leaderID ids.ActorID
	for _, t := range mergees {
		if t.Archived {
			return Task{}, ErrMergeState
		}
		if t.ProjectID != projectID {
			return Task{}, ErrMergeProjectMismatch
		}
		switch t.State {
		case StateBacklog, StateQueued, StateFailed:
		default:
			return Task{}, ErrMergeState
		}
		if t.AssigneeActorID != assignee {
			return Task{}, ErrMergeAssigneeConflict
		}
		if t.Priority > priority {
			priority = t.Priority
		}
		if r := sizeRankOf(t.Size); r > sizeRank {
			sizeRank = r
		}
		for _, tag := range t.Tags {
			if !tags[tag.ID] {
				tags[tag.ID] = true
				tagIDs = append(tagIDs, tag.ID)
			}
		}
		for _, teamID := range t.TeamIDs {
			if !teamSeen[teamID] {
				teamSeen[teamID] = true
				teamIDs = append(teamIDs, teamID)
			}
		}
		if leaderID == 0 {
			leaderID = t.LeaderActorID
		}
		prompts = append(prompts, t.Prompt)
	}

	merged, err := s.Create(ctx, CreateParams{
		Prompt: strings.Join(prompts, mergePromptSeparator),

		Name:            "",
		ProjectID:       projectID,
		AssigneeActorID: assignee,
		TagIDs:          tagIDs,
		Priority:        priority,
		Size:            sizeName(sizeRank),
		State:           StateBacklog,
		TeamIDs:         teamIDs,
		LeaderActorID:   leaderID,
	})
	if err != nil {
		return Task{}, err
	}

	now := time.Now().UnixMilli()
	linkSeen := make(map[string]bool)
	refSeen := make(map[string]bool)
	links := make([]Link, 0)
	refs := make([]ReferenceInput, 0)
	for _, t := range mergees {
		attIDs, err := s.store.ListTaskAttachmentIDs(ctx, t.ID)
		if err != nil {
			return Task{}, err
		}
		for _, attID := range attIDs {
			if err := s.store.LinkTaskAttachment(ctx, store.LinkTaskAttachmentParams{
				TaskID:       merged.ID,
				AttachmentID: attID,
				CreatedAt:    now,
			}); err != nil {
				return Task{}, err
			}
		}
		for _, link := range t.Links {
			if linkSeen[link.URL] {
				continue
			}
			linkSeen[link.URL] = true
			links = append(links, link)
		}
		for _, ref := range t.References {
			if seen[ref.ID] {
				continue
			}
			key := ref.ID + "|" + string(ref.Kind)
			if refSeen[key] {
				continue
			}
			refSeen[key] = true
			refs = append(refs, ReferenceInput{TaskID: ref.ID, Kind: ref.Kind})
		}
	}
	if len(links) > 0 {
		if _, err := s.SetLinks(ctx, merged.ID, links); err != nil {
			return Task{}, err
		}
	}
	for _, ref := range refs {
		if _, err := s.AddReference(ctx, merged.ID, ref.TaskID, ref.Kind); err != nil {
			return Task{}, err
		}
	}

	for _, t := range mergees {
		if _, err := s.addReference(
			ctx, merged.ID, t.ID, ReferenceSupersedes, false, false,
		); err != nil {
			return Task{}, err
		}
	}

	for _, t := range mergees {
		if err := s.rewireBlockers(ctx, t.ID, merged.ID); err != nil {
			return Task{}, err
		}
	}
	return s.Get(ctx, merged.ID)
}

func sizeRankOf(size string) int {
	switch normalizeSize(size) {
	case SizeSmall:
		return 0
	case SizeBig:
		return 2
	case SizeEpic:
		return 3
	default:
		return 1
	}
}

func sizeName(rank int) string {
	switch {
	case rank <= 0:
		return SizeSmall
	case rank == 2:
		return SizeBig
	case rank >= 3:
		return SizeEpic
	default:
		return SizeMedium
	}
}

const SettleGrace = 10 * time.Second

func (t Task) RunMayBeStarting(now time.Time, grace time.Duration) bool {
	if grace <= 0 {
		return false
	}
	if t.State != StateRunning && t.State != StateInput {
		return false
	}
	if t.StateUpdatedAt.IsZero() {
		return false
	}
	return now.Sub(t.StateUpdatedAt) < grace
}

func (s *Service) Settle(ctx context.Context, id string) (bool, error) {

	defer s.lockHandoff(id)()
	before, beforeErr := s.snapshot(ctx, id)
	now := time.Now().UnixMilli()

	ctx = WithStateReason(ctx, "run finished")
	if beforeErr == nil && before.AssigneeActorID != 0 {
		ctx = WithActorID(ctx, before.AssigneeActorID)
		ctx = WithActor(ctx, ActorAgent)
	}

	trigger, stopped := s.pendingStop(ctx, id)
	if stopped {
		ctx = WithStopTrigger(ctx, trigger)
	}

	if !stopped && beforeErr == nil && before.State == StateRunning {
		continued, err := s.continueRun(ctx, before, now)
		if err != nil {
			log.Printf("[tasks] continue run %s: %v", id, err)
		}
		if continued {
			return false, nil
		}
	}
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskSettle,
		SettledAt: sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return false, err
	}
	if n != 1 {
		return false, nil
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return true, nil
	}
	if beforeErr == nil {
		s.logChanges(ctx, before, after)
	}
	if after.State == StateWaiting {

		holds, err := s.Holds(ctx, id)
		if err != nil {
			return true, nil
		}
		s.transitionedHeld(ctx, id, holds)
		return true, nil
	}
	s.transitioned(ctx, id, StateSettled)
	if err := s.ReconcileWaiting(ctx, id); err != nil {
		log.Printf("[tasks] reconcile waiting after settle %s: %v", id, err)
	}

	if requeued, err := s.requeuePendingPrompt(ctx, id); err != nil {
		log.Printf("[tasks] requeue pending prompt %s: %v", id, err)
	} else if requeued && s.OnPendingPromptQueued != nil {
		s.OnPendingPromptQueued(ctx, id)
	}
	s.taskFinished(ctx, id, OutcomeSettled)
	return true, nil
}

const SettleHandoffTimeout = 10 * time.Second

func (s *Service) continueRun(ctx context.Context, before Task, now int64) (bool, error) {
	if s.DeliverPendingPrompt == nil {
		return false, nil
	}
	prompt := strings.TrimSpace(before.PendingPrompt)

	delivered := prompt != "" && s.pendingPromptDelivered(ctx, before.ID, prompt)
	if prompt == "" || delivered {

		return s.continuedRecently(ctx, before.ID, now), nil
	}

	cur, err := s.Get(ctx, before.ID)
	if err != nil {
		return false, err
	}
	if len(cur.Holds) > 0 {
		return false, nil
	}
	waits, err := s.hasIncompleteDependencies(ctx, cur)
	if err != nil {
		return false, err
	}
	if waits {
		return false, nil
	}
	handoffCtx, cancel := context.WithTimeout(ctx, SettleHandoffTimeout)
	defer cancel()
	if !s.DeliverPendingPrompt(handoffCtx, cur) {

		return false, nil
	}

	if err := s.MarkRunContinued(ctx, before.ID); err != nil {
		log.Printf("[tasks] mark run continued %s: %v", before.ID, err)
	}
	return true, nil
}

func (s *Service) continuedRecently(ctx context.Context, id string, now int64) bool {
	st, err := s.WakeState(ctx, id)
	if err != nil || st.ContinuedAt == 0 {
		return false
	}
	return now-st.ContinuedAt < SettleGrace.Milliseconds()
}

func (s *Service) MarkRunContinued(ctx context.Context, id string) error {
	st, err := s.WakeState(ctx, id)
	if err != nil {
		return err
	}
	st.ContinuedAt = time.Now().UnixMilli()
	return s.SetWakeState(ctx, id, st)
}

const handoffLockStripes = 64

func (s *Service) lockHandoff(id string) func() {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	mu := &s.handoffLocks[h.Sum32()%handoffLockStripes]
	mu.Lock()
	return mu.Unlock
}

func isFailedStop(dataJSON string) bool {
	if dataJSON == "" {
		return false
	}
	var data struct {
		Failed bool `json:"failed"`
	}
	return json.Unmarshal([]byte(dataJSON), &data) == nil && data.Failed
}

func (s *Service) pendingStop(ctx context.Context, id string) (StopTrigger, bool) {
	logs, err := s.Logs(ctx, id)
	if err != nil {
		return StopTrigger{}, false
	}
	for _, entry := range logs {
		switch entry.Kind {
		case LogStop:

			if isFailedStop(entry.DataJSON) {
				return StopTrigger{}, false
			}
			return StopTrigger{
				Actor:   Actor(entry.Actor),
				ActorID: entry.ActorID,
				At:      entry.At.UnixMilli(),
			}, true
		case LogState:

			return StopTrigger{}, false
		}
	}
	return StopTrigger{}, false
}

func (s *Service) requeuePendingPrompt(ctx context.Context, id string) (bool, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return false, err
	}
	if cur.State != StateSettled || strings.TrimSpace(cur.PendingPrompt) == "" {
		return false, nil
	}

	if s.pendingPromptDelivered(ctx, id, cur.PendingPrompt) {
		return false, nil
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:                store.TaskRequeueWithPrompt,
		PendingPrompt:        cur.PendingPrompt,
		PendingPromptActorID: nullID(cur.PendingPromptActorID),
		QueuedAt:             sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt:            now,
		ID:                   id,
	})
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return true, err
	}
	s.logChanges(ctx, cur, after)
	if err := s.ReconcileWaiting(ctx, id); err != nil {
		log.Printf("[tasks] reconcile waiting after pending-prompt requeue %s: %v", id, err)
	}
	s.changed(ctx, id)
	return true, nil
}

func (s *Service) pendingPromptDelivered(
	ctx context.Context,
	id, prompt string,
) bool {
	st, err := s.WakeState(ctx, id)
	if err != nil {
		return false
	}
	return st.DeliveredPrompt != "" &&
		st.DeliveredPrompt == strings.TrimSpace(prompt)
}

func (s *Service) MarkInput(ctx context.Context, id string) (bool, error) {
	before, beforeErr := s.snapshot(ctx, id)
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskInput,
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	if err != nil {
		return false, err
	}
	if n == 1 && beforeErr == nil {
		if after, err := s.snapshot(ctx, id); err == nil {
			s.logChanges(ctx, before, after)
		}
	}
	return n == 1, nil
}

func (s *Service) ClearInput(ctx context.Context, id string) (bool, error) {
	before, beforeErr := s.snapshot(ctx, id)
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskClearInput,
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	if err != nil {
		return false, err
	}
	if n == 1 {
		if beforeErr == nil {
			if after, err := s.snapshot(ctx, id); err == nil {
				s.logChanges(ctx, before, after)
			}
		}
		if s.OnInputCleared != nil {
			if t, err := s.Get(ctx, id); err == nil {

				unlock := s.lockHandoff(id)
				defer unlock()
				s.OnInputCleared(ctx, t)
			}
		}
	}
	return n == 1, nil
}

func (s *Service) Resume(ctx context.Context, id string) (bool, error) {
	before, beforeErr := s.snapshot(ctx, id)
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskResume,
		SettledAt: sql.NullInt64{},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return false, err
	}
	if n == 1 {
		s.transitioned(ctx, id, StateRunning)
		if after, err := s.snapshot(ctx, id); err == nil && beforeErr == nil {
			s.logChanges(ctx, before, after)
		}
	}
	return n == 1, nil
}

func (s *Service) Complete(ctx context.Context, id string) (Task, error) {
	return s.closeOut(ctx, id, ResolutionCompleted)
}

func (s *Service) Cancel(ctx context.Context, id string) (Task, error) {
	return s.closeOut(ctx, id, ResolutionCancelled)
}

func (s *Service) closeOut(ctx context.Context, id string, r Resolution) (Task, error) {
	cur, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if cur.State == StateRunning || cur.State == StateInput {
		return Task{}, ErrRunning
	}

	if r == ResolutionCompleted &&
		strings.TrimSpace(cur.PendingPrompt) != "" &&
		!s.pendingPromptDelivered(ctx, id, cur.PendingPrompt) {
		return Task{}, ErrPendingPrompt
	}
	if r == ResolutionCompleted {
		if err := s.requireLeader(ctx, id, StateCompleted); err != nil {
			return Task{}, err
		}
	}
	now := time.Now().UnixMilli()
	var n int64
	switch r {
	case ResolutionCompleted:
		n, err = s.store.TransitionTask(ctx, store.TransitionTaskParams{
			Event:     store.TaskComplete,
			SettledAt: sql.NullInt64{Int64: now, Valid: true},
			UpdatedAt: now,
			ID:        id,
		})
	case ResolutionCancelled:
		n, err = s.store.TransitionTask(ctx, store.TransitionTaskParams{
			Event:     store.TaskCancel,
			SettledAt: sql.NullInt64{Int64: now, Valid: true},
			UpdatedAt: now,
			ID:        id,
		})
	default:
		return Task{}, ErrWrongState
	}
	if err != nil {
		return Task{}, err
	}
	if n == 0 {
		return Task{}, ErrWrongState
	}

	if err := s.discardPendingMessages(ctx, id); err != nil {
		log.Printf("[tasks] discard queued messages after close-out %s: %v", id, err)
	}

	if err := s.ClearHolds(ctx, id, HoldClearedByCloseOut); err != nil {
		log.Printf("[tasks] clear holds after close-out %s: %v", id, err)
	}
	if err := s.clearRequiredTools(ctx, cur); err != nil {
		log.Printf("[tasks] clear required tools after close-out %s: %v", id, err)
	}
	state := StateCompleted
	if r == ResolutionCancelled {
		state = StateCancelled
	}
	s.transitioned(ctx, id, state)
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	if err := s.ReconcileWaiting(ctx, id); err != nil {
		log.Printf("[tasks] reconcile waiting after close-out %s: %v", id, err)
	}
	s.reconcileReferrers(ctx, id)
	if r == ResolutionCompleted {
		s.taskFinished(ctx, id, OutcomeCompleted)
	} else {
		s.taskFinished(ctx, id, OutcomeCancelled)
	}
	return after, nil
}

func (s *Service) reconcileReferrers(ctx context.Context, id string) {
	referrers, err := s.store.ListTaskReferencedBy(ctx, id)
	if err != nil {
		log.Printf("[tasks] list referrers of %s: %v", id, err)
		return
	}
	for _, referrer := range referrers {
		if err := s.ReconcileWaiting(ctx, referrer.ID); err != nil {
			log.Printf("[tasks] reconcile referrer %s: %v", referrer.ID, err)
		}
	}
}

func (s *Service) RequeueWithPrompt(
	ctx context.Context,
	id, prompt string,
	actorID ids.ActorID,
) (Task, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return Task{}, ErrInvalidPrompt
	}
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	switch cur.State {
	case StateSettled, StateFailed, StateCompleted, StateCancelled:
	case StateBacklog:
		if cur.StartedAt.IsZero() {
			return Task{}, ErrWrongState
		}
	default:
		return Task{}, ErrWrongState
	}
	messages, err := s.queuePendingMessage(ctx, id, QueueParams{
		Text:    prompt,
		ActorID: actorID,
	})
	if err != nil {
		return Task{}, err
	}
	text, actor := joinPending(messages)
	if err := s.requeueWithPrompt(ctx, cur, text, actor); err != nil {
		return Task{}, err
	}
	return s.Get(ctx, id)
}

func (s *Service) requeueWithPrompt(
	ctx context.Context,
	cur Task,
	pending string,
	actorID ids.ActorID,
) error {
	if err := s.validateRequeue(ctx, cur); err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:                store.TaskRequeueWithPrompt,
		PendingPrompt:        pending,
		PendingPromptActorID: nullID(actorID),
		QueuedAt:             sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt:            now,
		ID:                   cur.ID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrWrongState
	}
	after, err := s.Get(ctx, cur.ID)
	if err != nil {
		return err
	}
	s.logChanges(ctx, cur, after)

	if cur.Archived {
		s.recordLog(
			ctx, actorFrom(ctx), now, cur.ID, LogState, "archive",
			"", "", "Task unarchived by a follow-up", nil,
		)
	}
	if err := s.ReconcileWaiting(ctx, cur.ID); err != nil {
		log.Printf("[tasks] reconcile waiting after requeue %s: %v", cur.ID, err)
	}
	return nil
}

func (s *Service) validateRequeue(ctx context.Context, cur Task) error {
	if err := s.validateAssigneeTagDeclarations(
		ctx,
		cur.ProjectID,
		cur.AssigneeActorID,
		tagIDsOf(cur.Tags),
	); err != nil {
		return err
	}
	return s.requireLeader(ctx, cur.ID, StateQueued)
}

func (s *Service) RequeueAfterStartFailure(
	ctx context.Context,
	id, errMsg string,
) error {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskRequeue,
		Error:     strings.TrimSpace(errMsg),
		QueuedAt:  sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}
	if n == 0 {

		return nil
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	s.logChanges(ctx, cur, after)
	s.transitioned(ctx, id, StateQueued)

	if err := s.ReconcileWaiting(ctx, id); err != nil {
		log.Printf("[tasks] reconcile waiting after requeue %s: %v", id, err)
	}
	return nil
}

func (s *Service) WakeWaitingWithPrompt(
	ctx context.Context,
	id, prompt string,
	actorID ids.ActorID,
) (Task, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return Task{}, ErrInvalidPrompt
	}
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if cur.State != StateWaiting {
		return Task{}, ErrWrongState
	}
	if len(cur.Holds) > 0 {
		return Task{}, heldError(cur.Holds)
	}

	messages, err := s.queuePendingMessage(ctx, id, QueueParams{
		Text:    prompt,
		ActorID: actorID,
	})
	if err != nil {
		return Task{}, err
	}
	pending, sender := joinPending(messages)
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:                store.TaskWakeWaiting,
		PendingPrompt:        pending,
		PendingPromptActorID: nullID(sender),
		QueuedAt:             sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt:            now,
		ID:                   id,
	})
	if err != nil {
		return Task{}, err
	}
	if n == 0 {

		if now, err := s.Get(ctx, id); err == nil && len(now.Holds) > 0 {
			return Task{}, heldError(now.Holds)
		}
		return Task{}, ErrWrongState
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) ForceWake(
	ctx context.Context,
	t Task,
	prompt string,
	actorID ids.ActorID,
) (bool, error) {
	if t.State != StateWaiting {
		return false, nil
	}
	_, err := s.WakeWaitingWithPrompt(ctx, t.ID, prompt, actorID)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, ErrHeld), errors.Is(err, ErrWrongState):
		return false, nil
	default:
		return false, err
	}
}

func (s *Service) Fail(ctx context.Context, id, errMsg string) (Task, error) {
	cur, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if cur.State != StateRunning && cur.State != StateInput {
		return Task{}, ErrWrongState
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskFail,
		Error:     errMsg,
		SettledAt: sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return Task{}, err
	}
	if n == 0 {
		return Task{}, ErrWrongState
	}

	if err := s.clearRequiredTools(ctx, cur); err != nil {
		log.Printf("[tasks] clear required tools after fail %s: %v", id, err)
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	if err := s.ReconcileWaiting(ctx, id); err != nil {
		log.Printf("[tasks] reconcile waiting after fail %s: %v", id, err)
	}
	s.taskFinished(ctx, id, OutcomeFailed)
	return after, nil
}

func (s *Service) Reopen(ctx context.Context, id string, queue bool) (Task, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	switch cur.State {
	case StateSettled, StateFailed, StateCompleted, StateCancelled, StateWaiting:
	default:
		return Task{}, ErrWrongState
	}
	if !queue && cur.Automated {
		return Task{}, ErrAutomatedBacklog
	}
	state := StateBacklog
	if queue {

		if len(cur.Holds) > 0 {
			if cur.State != StateFailed {
				return Task{}, heldError(cur.Holds)
			}
			state = StateWaiting
		} else {

			if !cur.StartedAt.IsZero() && cur.State != StateFailed {
				return Task{}, ErrFollowUpRequired
			}
			if err := s.requireExecutor(ctx, cur.ProjectID, cur.AssigneeActorID); err != nil {
				return Task{}, err
			}

			if err := s.validateAssigneeTagDeclarations(
				ctx,
				cur.ProjectID,
				cur.AssigneeActorID,
				tagIDsOf(cur.Tags),
			); err != nil {
				return Task{}, err
			}
			if err := s.requireLeader(ctx, id, StateQueued); err != nil {
				return Task{}, err
			}
			state = StateQueued
			waiting, err := s.hasIncompleteDependencies(ctx, cur)
			if err != nil {
				return Task{}, err
			}
			if waiting {
				state = StateWaiting
			}
		}
	}
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskReopen,
		State:     string(state),
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	if err != nil {
		return Task{}, err
	}
	if n == 0 {
		return Task{}, ErrWrongState
	}

	s.clearBlockerWakeKeys(ctx, id)

	if err := s.discardPendingMessages(ctx, id); err != nil {
		log.Printf("[tasks] discard queued messages after reopen %s: %v", id, err)
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) Rename(ctx context.Context, id, name string) (Task, error) {
	cur, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if _, err := s.store.RenameTask(ctx, store.RenameTaskParams{
		Name:      strings.TrimSpace(name),
		UpdatedAt: time.Now().UnixMilli(),
		ID:        cur.ID,
	}); err != nil {
		return Task{}, err
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) SetPriority(ctx context.Context, id string, priority int64) (Task, error) {
	cur, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if _, err := s.store.SetTaskPriority(ctx, store.SetTaskPriorityParams{
		Priority:  priority,
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	}); err != nil {
		return Task{}, err
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) SetSize(ctx context.Context, id string, size string) (Task, error) {
	cur, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if _, err := s.store.SetTaskSize(ctx, store.SetTaskSizeParams{
		Size:      normalizeSize(size),
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	}); err != nil {
		return Task{}, err
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) UpdateAttributes(
	ctx context.Context,
	id string,
	attrs AttributeSet,
) (Task, error) {
	cur, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}

	if attrs.changesTags() || attrs.Prompt != nil || attrs.AssigneeActorID != nil {
		if cur, err = s.Get(ctx, id); err != nil {
			return Task{}, err
		}
	}
	return s.applyAttributes(ctx, cur, attrs)
}

func (s *Service) UpdateManyAttributes(
	ctx context.Context,
	ids []string,
	attrs AttributeSet,
) []AttributeResult {
	out := make([]AttributeResult, 0, len(ids))
	for _, id := range ids {
		t, err := s.UpdateAttributes(ctx, id, attrs)
		out = append(out, AttributeResult{ID: id, Task: t, Error: err})
	}
	return out
}

func (s *Service) applyAttributes(
	ctx context.Context,
	cur Task,
	attrs AttributeSet,
) (Task, error) {
	if attrs.Empty() {
		return cur, nil
	}
	if attrs.AssigneeActorID != nil {
		if cur.State == StateRunning || cur.State == StateInput {
			return Task{}, ErrRunning
		}
		if *attrs.AssigneeActorID != 0 {
			if _, err := s.validateScope(
				ctx,
				cur.ProjectID,
				*attrs.AssigneeActorID,
			); err != nil {
				return Task{}, err
			}
		}
	}

	finalAssignee := cur.AssigneeActorID
	if attrs.AssigneeActorID != nil {
		finalAssignee = *attrs.AssigneeActorID
	}
	if finalAssignee != 0 && (attrs.AssigneeActorID != nil || attrs.changesTags()) {
		if err := s.validateAssigneeTagDeclarations(
			ctx,
			cur.ProjectID,
			finalAssignee,
			resultingTagIDs(tagIDsOf(cur.Tags), attrs.AddTagIDs, attrs.RemoveTagIDs),
		); err != nil {
			return Task{}, err
		}
	}
	editable := cur.State == StateBacklog && cur.StartedAt.IsZero()
	now := time.Now().UnixMilli()
	if attrs.Prompt != nil {

		if !editable {
			return Task{}, ErrNotEditable
		}
		prompt := strings.TrimSpace(*attrs.Prompt)
		if prompt == "" {
			return Task{}, ErrInvalidPrompt
		}
		n, err := s.store.SetTaskPrompt(ctx, store.SetTaskPromptParams{
			Prompt:    prompt,
			UpdatedAt: now,
			ID:        cur.ID,
		})
		if err != nil {
			return Task{}, err
		}
		if n == 0 {
			return Task{}, ErrNotEditable
		}
	}
	if attrs.PendingPrompt != nil &&
		strings.TrimSpace(*attrs.PendingPrompt) == "" &&
		(cur.State == StateInput || cur.State == StateRunning) {

		if err := s.discardPendingMessages(ctx, cur.ID); err != nil {
			return Task{}, err
		}
		_, err := s.store.ClearTaskPendingPrompt(
			ctx,
			store.ClearTaskPendingPromptParams{
				UpdatedAt: now,
				ID:        cur.ID,
			},
		)
		if err != nil {
			return Task{}, err
		}
	} else if attrs.PendingPrompt != nil {

		if !editable {
			return Task{}, ErrNotEditable
		}
		n, err := s.store.SetTaskPendingPrompt(ctx, store.SetTaskPendingPromptParams{
			PendingPrompt:        *attrs.PendingPrompt,
			PendingPromptActorID: nullID(actorIDFrom(ctx)),
			UpdatedAt:            now,
			ID:                   cur.ID,
		})
		if err != nil {
			return Task{}, err
		}
		if n == 0 {
			return Task{}, ErrNotEditable
		}

		if err := s.discardPendingMessages(ctx, cur.ID); err != nil {
			return Task{}, err
		}
		if _, err := s.queuePendingMessage(ctx, cur.ID, QueueParams{
			Text:       *attrs.PendingPrompt,
			ActorID:    actorIDFrom(ctx),
			Cancelable: true,
		}); err != nil {
			return Task{}, err
		}
	}
	if attrs.Model != nil {

		if !editable {
			return Task{}, ErrNotEditable
		}
		model := *attrs.Model
		if (model.Provider == "") != (model.ModelID == "") {
			return Task{}, ErrInvalidModel
		}

		n, err := s.store.SetTaskModel(ctx, store.SetTaskModelParams{
			ModelProvider: model.Provider,
			ModelID:       model.ModelID,
			ThinkingLevel: model.ThinkingLevel,
			UpdatedAt:     now,
			ID:            cur.ID,
		})
		if err != nil {
			return Task{}, err
		}
		if n == 0 {
			return Task{}, ErrNotEditable
		}
	}
	if attrs.Priority != nil {
		if _, err := s.store.SetTaskPriority(ctx, store.SetTaskPriorityParams{
			Priority:  *attrs.Priority,
			UpdatedAt: now,
			ID:        cur.ID,
		}); err != nil {
			return Task{}, err
		}
	}
	if attrs.Size != nil {
		if _, err := s.store.SetTaskSize(ctx, store.SetTaskSizeParams{
			Size:      normalizeSize(*attrs.Size),
			UpdatedAt: now,
			ID:        cur.ID,
		}); err != nil {
			return Task{}, err
		}
	}
	if attrs.AssigneeActorID != nil {
		if _, err := s.store.SetTaskAssignee(ctx, store.SetTaskAssigneeParams{
			AssigneeActorID: nullID(*attrs.AssigneeActorID),
			AssigneePinned:  boolInt(*attrs.AssigneeActorID != 0),
			UpdatedAt:       now,
			ID:              cur.ID,
		}); err != nil {
			return Task{}, err
		}
	}
	if attrs.changesTags() {

		if err := s.ValidateTags(ctx, cur.ProjectID, attrs.AddTagIDs); err != nil {
			return Task{}, err
		}
		for _, tagID := range attrs.RemoveTagIDs {
			if err := s.store.DeleteTaskTag(ctx, store.DeleteTaskTagParams{
				TaskID: cur.ID,
				TagID:  tagID,
			}); err != nil {
				return Task{}, err
			}
		}
		for _, tagID := range attrs.AddTagIDs {
			if err := s.store.AddTaskTag(ctx, store.AddTaskTagParams{
				TaskID: cur.ID,
				TagID:  tagID,
			}); err != nil {
				return Task{}, err
			}
		}
	}
	var (
		after Task
		err   error
	)
	if attrs.changesTags() || attrs.Prompt != nil {
		after, err = s.Get(ctx, cur.ID)
	} else {
		after, err = s.snapshot(ctx, cur.ID)
	}
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	referrers, _ := s.store.ListTaskReferencedBy(ctx, id)
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.store.DeleteTask(ctx, id); err != nil {
		return err
	}
	s.changed(ctx, id)

	for _, referrer := range referrers {
		if err := s.ReconcileWaiting(ctx, referrer.ID); err != nil {
			log.Printf("[tasks] reconcile referrer %s after delete: %v", referrer.ID, err)
		}
	}
	return nil
}

func (s *Service) Claim(
	ctx context.Context,
	task Task,
	actorID ids.ActorID,
	pins prompts.VersionIDs,
	now int64,
) (bool, error) {
	org, agent, project, agentProject := nullPromptIDs(pins)
	if !task.StartedAt.IsZero() {
		org = sql.NullInt64{}
		agent = sql.NullInt64{}
		project = sql.NullInt64{}
		agentProject = sql.NullInt64{}
	}
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:                store.TaskClaim,
		AssigneeActorID:      nullID(actorID),
		AssigneePinned:       boolInt(task.AssigneePinned),
		StartedAt:            sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt:            now,
		OrgPromptID:          org,
		AgentPromptID:        agent,
		ProjectPromptID:      project,
		AgentProjectPromptID: agentProject,
		ID:                   task.ID,
	})
	if err != nil {
		return false, err
	}
	if n == 1 {

		if err := s.ResetWakeState(ctx, task.ID); err != nil {
			log.Printf("[tasks] reset wake state %s: %v", task.ID, err)
		}

		if task.StartedAt.IsZero() {
			if err := s.setTeamPromptPins(ctx, task.ID, pins.Teams); err != nil {
				return false, err
			}
			if err := s.setGroupPromptPins(ctx, task.ID, pins.Groups); err != nil {
				return false, err
			}
		}
		s.transitioned(ctx, task.ID, StateRunning)
		if after, err := s.Get(ctx, task.ID); err == nil {
			s.logChanges(ctx, task, after)
		}
	}
	return n == 1, nil
}

func (s *Service) SetSession(
	ctx context.Context,
	id string,
	agentID ids.AgentID,
	sessionID string,
) error {
	_, err := s.store.SetTaskSession(ctx, store.SetTaskSessionParams{
		SessionID:      sessionID,
		SessionAgentID: sql.NullInt64{Int64: int64(agentID), Valid: true},
		UpdatedAt:      time.Now().UnixMilli(),
		ID:             id,
	})
	if err != nil {
		return err
	}

	if err := s.store.RecordTaskSession(ctx, store.RecordTaskSessionParams{
		TaskID:    id,
		AgentID:   int64(agentID),
		SessionID: sessionID,
		CreatedAt: time.Now().UnixMilli(),
	}); err != nil {
		log.Printf("[tasks] record session %s for %s: %v", sessionID, id, err)
	}
	s.changed(ctx, id)
	return nil
}

const CleanupSessionTimeout = 30 * time.Second

func (s *Service) CleanupSession(ctx context.Context, id string) error {
	t, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.DeleteAgentSession == nil {
		return nil
	}
	sessions, err := s.store.ListLiveTaskSessions(ctx, t.ID)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {

		if t.SessionID != "" && t.SessionDeletedAt.IsZero() {
			if _, err := s.MarkSessionDeleted(ctx, t.ID); err != nil {
				return err
			}
		}
		return nil
	}

	var firstErr error
	cleaned := 0
	for _, session := range sessions {
		callCtx, cancel := context.WithTimeout(ctx, CleanupSessionTimeout)
		err := s.DeleteAgentSession(
			callCtx,
			ids.AgentID(session.AgentID),
			session.SessionID,
		)
		cancel()
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"delete agent session %s on agent %d: %w",
					session.SessionID, session.AgentID, err,
				)
			}
			continue
		}
		if err := s.store.MarkTaskSessionCleaned(
			ctx,
			store.MarkTaskSessionCleanedParams{
				DeletedAt: sql.NullInt64{
					Int64: time.Now().UnixMilli(),
					Valid: true,
				},
				TaskID:    t.ID,
				AgentID:   session.AgentID,
				SessionID: session.SessionID,
			},
		); err != nil {
			return err
		}
		cleaned++
	}

	if firstErr != nil {
		return firstErr
	}
	if cleaned > 0 && t.SessionDeletedAt.IsZero() {
		if _, err := s.MarkSessionDeleted(ctx, t.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) MarkSessionDeleted(ctx context.Context, id string) (Task, error) {
	now := time.Now().UnixMilli()
	if _, err := s.store.MarkTaskSessionDeleted(
		ctx,
		store.MarkTaskSessionDeletedParams{
			SessionDeletedAt: sql.NullInt64{Int64: now, Valid: true},
			UpdatedAt:        now,
			ID:               id,
		},
	); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return Task{}, err
		}

		return s.snapshot(ctx, id)
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.recordLog(
		ctx, actorFrom(ctx), now, after.ID, LogSession, "session",
		"", "", "Pi session deleted", nil,
	)
	return after, nil
}

func (s *Service) MarkSessionLost(ctx context.Context, id, reason string) (Task, error) {
	now := time.Now().UnixMilli()
	if _, err := s.store.MarkTaskSessionDeleted(
		ctx,
		store.MarkTaskSessionDeletedParams{
			SessionDeletedAt: sql.NullInt64{Int64: now, Valid: true},
			UpdatedAt:        now,
			ID:               id,
		},
	); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return Task{}, err
		}

		return s.snapshot(ctx, id)
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.recordLog(
		ctx, actorFrom(ctx), now, after.ID, LogSessionLost, "session",
		"", "", "Pi session lost on the agent",
		map[string]any{"reason": reason},
	)
	return after, nil
}

func (s *Service) SetError(ctx context.Context, id, errMsg string) error {
	_, err := s.store.SetTaskError(ctx, store.SetTaskErrorParams{
		Error:     errMsg,
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	if err == nil {
		s.changed(ctx, id)
	}
	return err
}

func (s *Service) RunningCounts(ctx context.Context, agentID ids.AgentID) (Counts, error) {
	rows, err := s.store.ListRunningTasksByAgent(ctx, agentID)
	if err != nil {
		return Counts{}, err
	}
	links, err := s.store.ListRunningTaskTagsByAgent(ctx, agentID)
	if err != nil {
		return Counts{}, err
	}
	tagsByTask := map[string][]Tag{}
	for _, link := range links {
		tagsByTask[link.TaskID] = append(tagsByTask[link.TaskID], Tag{
			ID:   link.TagID,
			Name: link.Name,
		})
	}

	out := NewCounts()
	for _, row := range rows {
		out.add(row.ProjectID.String, tagsByTask[row.ID])
	}
	return out, nil
}

func (s *Service) AgentClassCounts(
	ctx context.Context,
	agentID ids.AgentID,
	reserved map[string]int,
) (ClassCounts, error) {
	rows, err := s.store.ListRunningTasksByAgent(ctx, agentID)
	if err != nil {
		return ClassCounts{}, err
	}
	links, err := s.store.ListRunningTaskTagsByAgent(ctx, agentID)
	if err != nil {
		return ClassCounts{}, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	tagsByTask := map[string][]Tag{}
	for _, link := range links {
		tagsByTask[link.TaskID] = append(tagsByTask[link.TaskID], Tag{
			ID:   link.TagID,
			Name: link.Name,
		})
	}
	return ClassCountsOf(ids, tagsByTask, reserved), nil
}

func (s *Service) ProjectClassCounts(
	ctx context.Context,
	projectID string,
	reserved map[string]int,
) (ClassCounts, error) {
	rows, err := s.store.ListRunningTasksByProject(ctx, projectID)
	if err != nil {
		return ClassCounts{}, err
	}
	links, err := s.store.ListRunningTaskTagsByProject(ctx, projectID)
	if err != nil {
		return ClassCounts{}, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	tagsByTask := map[string][]Tag{}
	for _, link := range links {
		tagsByTask[link.TaskID] = append(tagsByTask[link.TaskID], Tag{
			ID:   link.TagID,
			Name: link.Name,
		})
	}
	return ClassCountsOf(ids, tagsByTask, reserved), nil
}

func (s *Service) CanRunTask(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
) (member, canPrompt bool, err error) {
	if projectID == "" {
		return true, true, nil
	}
	perms, ok, err := s.projects.Permissions(ctx, projectID, agentID)
	if err != nil {
		return false, false, err
	}
	return ok, ok && perms.CanCreateTasks, nil
}

func (s *Service) IsMember(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
) (bool, error) {
	member, _, err := s.CanRunTask(ctx, projectID, agentID)
	return member, err
}

func (s *Service) requireStates(
	ctx context.Context,
	id string,
	want ...State,
) (Task, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	for _, state := range want {
		if cur.State == state {
			return cur, nil
		}
	}
	return Task{}, ErrWrongState
}

func (s *Service) validateScope(
	ctx context.Context,
	projectID string, assigneeActorID ids.ActorID,
) (projects.Project, error) {
	if projectID == "" {
		return projects.Project{}, nil
	}
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return projects.Project{}, err
	}
	if agentID, ok := s.agentIDByActor(ctx, assigneeActorID); ok {
		member, canPrompt, err := s.CanRunTask(ctx, project.ID, agentID)
		if err != nil {
			return projects.Project{}, err
		}
		if !member {
			return projects.Project{}, ErrNotMember
		}
		if !canPrompt {
			return projects.Project{}, ErrAssigneeViewer
		}
	}
	return project, nil
}

func (s *Service) agentIDByActor(ctx context.Context, actorID ids.ActorID) (ids.AgentID, bool) {
	if actorID == 0 {
		return 0, false
	}
	row, err := s.store.GetAgentByActor(ctx, actorID)
	if err != nil {
		return 0, false
	}
	return row.ID, true
}

func TaskID(prefix string, number int64) string {
	return prefix + "-" + strconv.FormatInt(number, 10)
}

func (s *Service) requireExecutor(
	ctx context.Context,
	projectID string,
	assigneeActorID ids.ActorID,
) error {
	if projectID == "" || assigneeActorID != 0 {
		return nil
	}
	agents, err := s.projects.ListAgents(ctx, projectID)
	if err != nil {
		return err
	}
	for _, agent := range agents {

		if agent.CanCreateTasks {
			return nil
		}
	}
	return ErrProjectHasNoAgents
}

func (s *Service) validateAssigneeTagDeclarations(
	ctx context.Context,
	projectID string,
	assigneeActorID ids.ActorID,
	tagIDs []string,
) error {
	if s.Limits == nil || projectID == "" || assigneeActorID == 0 ||
		len(tagIDs) == 0 {
		return nil
	}
	agentID, ok := s.agentIDByActor(ctx, assigneeActorID)
	if !ok {
		return nil
	}
	required := make([]Tag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		row, err := s.store.GetTag(ctx, tagID)
		if err != nil {
			return err
		}
		if row.IsRequired == 0 {
			continue
		}
		required = append(required, Tag{
			ID:        row.ID,
			ProjectID: row.ProjectID.String,
			Name:      row.Name,
			Required:  true,
		})
	}
	if len(required) == 0 {
		return nil
	}
	lim, err := s.Limits.Get(ctx, agentID)
	if err != nil {
		return err
	}
	missing := MissingTagDeclarations(lim, projectID, required)
	if len(missing) == 0 {
		return nil
	}
	names := make([]string, len(missing))
	for i, tag := range missing {
		names[i] = "#" + tag.Name
	}
	return fmt.Errorf("%w: %s", ErrAssigneeMissingTag, strings.Join(names, ", "))
}

func tagIDsOf(tags []Tag) []string {
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = tag.ID
	}
	return out
}

func resultingTagIDs(current, add, remove []string) []string {
	seen := make(map[string]bool, len(current)+len(add))
	out := make([]string, 0, len(current)+len(add))
	for _, id := range current {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range remove {
		delete(seen, id)
	}
	kept := out[:0]
	for _, id := range out {
		if seen[id] {
			kept = append(kept, id)
		}
	}
	for _, id := range add {
		if seen[id] {
			continue
		}
		seen[id] = true
		kept = append(kept, id)
	}
	return kept
}

func (s *Service) ValidateTags(ctx context.Context, projectID string, tagIDs []string) error {
	for _, tagID := range tagIDs {
		row, err := s.store.GetTag(ctx, tagID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTagMissing
		} else if err != nil {
			return err
		}
		if row.ProjectID.String != projectID {
			return ErrTagMissing
		}
	}
	return nil
}

func (s *Service) SetTags(ctx context.Context, taskID string, tagIDs []string) error {
	cur, err := s.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if err := s.validateAssigneeTagDeclarations(
		ctx,
		cur.ProjectID,
		cur.AssigneeActorID,
		tagIDs,
	); err != nil {
		return err
	}
	if err := s.store.DeleteTaskTags(ctx, taskID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if err := s.store.AddTaskTag(ctx, store.AddTaskTagParams{
			TaskID: taskID,
			TagID:  tagID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) tagsForTask(ctx context.Context, taskID string) ([]Tag, error) {
	rows, err := s.store.ListTaskTagsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, row := range rows {
		out[i] = Tag{
			ID:          row.ID,
			ProjectID:   row.ProjectID.String,
			Name:        row.Name,
			Description: row.Description,
			Color:       row.Color,
			Archived:    row.ArchivedAt.Valid,
			Required:    row.IsRequired != 0,
		}
	}
	return out, nil
}

func (s *Service) SetLinksBySession(
	ctx context.Context,
	agentID ids.AgentID, sessionID string,
	links []Link,
) error {
	taskID, err := s.store.TaskIDBySession(ctx, store.TaskIDBySessionParams{
		AgentID:   int64(agentID),
		SessionID: sessionID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.SetLinks(ctx, taskID, links)
	return err
}

func (s *Service) SetLinks(ctx context.Context, taskID string, links []Link) (Task, error) {
	before, beforeErr := s.Get(ctx, taskID)
	clean := normalizeLinks(links)
	if err := s.store.DeleteTaskLinks(ctx, taskID); err != nil {
		return Task{}, err
	}
	for i, link := range clean {
		if err := s.store.InsertTaskLink(ctx, store.InsertTaskLinkParams{
			TaskID:   taskID,
			Url:      link.URL,
			Label:    link.Label,
			Position: int64(i),
		}); err != nil {
			return Task{}, err
		}
	}
	after, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	if beforeErr == nil {
		s.logChanges(ctx, before, after)
	}
	return after, nil
}

func normalizeLinks(in []Link) []Link {
	out := make([]Link, 0, len(in))
	seen := make(map[string]int, len(in))
	for _, link := range in {
		raw := strings.TrimSpace(link.URL)
		if !isHTTPURL(raw) {
			continue
		}
		label := strings.TrimSpace(link.Label)
		if i, ok := seen[raw]; ok {
			out[i].Label = label
			continue
		}
		seen[raw] = len(out)
		out = append(out, Link{URL: raw, Label: label})
	}
	return out
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (s *Service) linksForTask(ctx context.Context, taskID string) ([]Link, error) {
	rows, err := s.store.ListTaskLinksByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return toLinks(rows), nil
}

func (s *Service) allLinks(ctx context.Context) (map[string][]Link, error) {
	rows, err := s.store.ListTaskLinks(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]Link{}
	for _, row := range rows {
		out[row.TaskID] = append(out[row.TaskID], Link{URL: row.Url, Label: row.Label})
	}
	return out, nil
}

func toLinks(rows []store.ListTaskLinksByTaskRow) []Link {
	out := make([]Link, len(rows))
	for i, row := range rows {
		out[i] = Link{URL: row.Url, Label: row.Label}
	}
	return out
}

func (s *Service) allReferences(
	ctx context.Context,
	tagsByTask map[string][]Tag,
) (map[string][]Reference, error) {
	rows, err := s.store.ListTaskReferenceRefs(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]Reference{}
	for _, row := range rows {
		out[row.TaskID] = append(out[row.TaskID], Reference{
			ID:                    row.ReferenceID,
			Name:                  row.ReferenceName,
			Prompt:                row.ReferencePrompt,
			State:                 State(row.ReferenceState),
			Kind:                  ReferenceKind(row.Kind),
			Priority:              row.ReferencePriority,
			Size:                  row.ReferenceSize,
			Tags:                  tagsByTask[row.ReferenceID],
			WakeOnResolveWhenMany: row.WakeOnResolveWhenMany != 0,
		})
	}
	return out, nil
}

func (s *Service) referencesForTask(
	ctx context.Context,
	taskID string,
) ([]Reference, error) {
	rows, err := s.store.ListTaskReferenceRefsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]Reference, len(rows))
	for i, row := range rows {
		out[i] = Reference{
			ID:       row.ReferenceID,
			Name:     row.ReferenceName,
			Prompt:   row.ReferencePrompt,
			State:    State(row.ReferenceState),
			Kind:     ReferenceKind(row.Kind),
			Priority: row.ReferencePriority,
			Size:     row.ReferenceSize,
		}
	}
	return out, nil
}

func (s *Service) referencedByForTask(
	ctx context.Context,
	taskID string,
) ([]Reference, error) {
	rows, err := s.store.ListTaskReferencedByRefsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]Reference, len(rows))
	for i, row := range rows {
		out[i] = Reference{
			ID:       row.ReferenceID,
			Name:     row.ReferenceName,
			Prompt:   row.ReferencePrompt,
			State:    State(row.ReferenceState),
			Kind:     ReferenceKind(row.Kind),
			Priority: row.ReferencePriority,
			Size:     row.ReferenceSize,
		}
	}
	return out, nil
}

func (s *Service) allTags(ctx context.Context) (map[string][]Tag, error) {
	rows, err := s.store.ListTaskTags(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]Tag{}
	for _, row := range rows {
		out[row.TaskID] = append(out[row.TaskID], Tag{
			ID:          row.ID,
			ProjectID:   row.ProjectID.String,
			Name:        row.Name,
			Description: row.Description,
			Color:       row.Color,
			Archived:    row.ArchivedAt.Valid,
			Required:    row.IsRequired != 0,
		})
	}
	return out, nil
}

func (s *Service) allTaskTeams(ctx context.Context) (map[string][]int64, error) {
	rows, err := s.store.ListTaskTeams(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]int64{}
	for _, row := range rows {
		out[row.TaskID] = append(out[row.TaskID], row.TeamID)
	}
	return out, nil
}

func (s *Service) allTaskLeads(ctx context.Context) (map[string][]int64, error) {
	rows, err := s.store.ListTaskLeads(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]int64{}
	for _, row := range rows {
		out[row.TaskID] = append(out[row.TaskID], row.ActorID)
	}
	return out, nil
}

func (s *Service) allHolds(ctx context.Context) (map[string][]TaskHold, error) {
	rows, err := s.store.ListAllTaskHolds(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]TaskHold{}
	for _, row := range rows {
		out[row.TaskID] = append(out[row.TaskID], TaskHold{
			TaskID:     row.TaskID,
			PluginID:   row.PluginID,
			PluginName: row.PluginName,
			Key:        row.Key,
			Reason:     row.Reason,
			CreatedAt:  time.UnixMilli(row.CreatedAt),
		})
	}
	return out, nil
}

func actorIDsFromInt64s(in []int64) []ids.ActorID {
	out := make([]ids.ActorID, len(in))
	for i, id := range in {
		out[i] = ids.ActorID(id)
	}
	return out
}

func firstActorID(in []int64) ids.ActorID {
	if len(in) == 0 {
		return 0
	}
	return ids.ActorID(in[0])
}

func dedupeInt64(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func (s *Service) validateTaskTeams(
	ctx context.Context,
	projectID string,
	teamIDs []int64,
) error {
	teamIDs = dedupeInt64(teamIDs)
	if len(teamIDs) == 0 {
		return ErrTeamsRequired
	}
	for _, id := range teamIDs {
		_, ok, err := s.projects.TeamRole(ctx, projectID, id)
		if err != nil {
			return err
		}
		if !ok {
			return ErrTeamNotInProject
		}
	}
	return nil
}

func (s *Service) resolveTaskTeams(
	ctx context.Context,
	projectID string,
	teamIDs []int64,
) ([]int64, error) {
	teamIDs = dedupeInt64(teamIDs)
	if len(teamIDs) == 0 {
		if creator := actorIDFrom(ctx); creator != 0 {
			derived, err := s.store.ListActorProjectTeams(
				ctx,
				store.ListActorProjectTeamsParams{
					ProjectID: projectID,
					ActorID:   int64(creator),
				},
			)
			if err != nil {
				return nil, err
			}
			teamIDs = dedupeInt64(derived)
		}
	}
	if len(teamIDs) == 0 {
		effective, err := s.projects.TeamIDs(ctx, projectID)
		if err != nil {
			return nil, err
		}
		teamIDs = dedupeInt64(effective)
	}
	if len(teamIDs) == 0 {
		return nil, nil
	}
	if err := s.validateTaskTeams(ctx, projectID, teamIDs); err != nil {
		return nil, err
	}
	return teamIDs, nil
}

func (s *Service) setTaskTeams(ctx context.Context, id string, teamIDs []int64) error {
	if err := s.store.DeleteTaskTeams(ctx, id); err != nil {
		return err
	}
	for _, teamID := range teamIDs {
		if err := s.store.AddTaskTeam(ctx, store.AddTaskTeamParams{
			TaskID: id,
			TeamID: teamID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) setTaskLeader(
	ctx context.Context,
	projectID string,
	id string,
	leaderActorID ids.ActorID,
) error {
	if err := s.validateTaskLeader(ctx, projectID, id, leaderActorID); err != nil {
		return err
	}
	if err := s.store.DeleteTaskLeads(ctx, id); err != nil {
		return err
	}
	if leaderActorID == 0 {
		return nil
	}
	return s.store.AddTaskLead(ctx, store.AddTaskLeadParams{
		TaskID:  id,
		ActorID: int64(leaderActorID),
		AddedAt: time.Now().UnixMilli(),
	})
}

func (s *Service) SetTaskLeader(
	ctx context.Context,
	id string,
	leaderActorID ids.ActorID,
) (Task, error) {
	before, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if before.Automated && leaderActorID != 0 {
		return Task{}, ErrAutomatedLeader
	}
	if !before.Automated && leadRequired(before.State) &&
		actorIDFrom(ctx) != 0 && leaderActorID == 0 {
		return Task{}, ErrLeadRequired
	}
	if err := s.setTaskLeader(ctx, before.ProjectID, id, leaderActorID); err != nil {
		return Task{}, err
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logOwnership(ctx, before, after)
	return after, nil
}

func (s *Service) SetTaskTeams(ctx context.Context, id string, teamIDs []int64) (Task, error) {
	before, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	teamIDs = dedupeInt64(teamIDs)
	if err := s.validateTaskTeams(ctx, before.ProjectID, teamIDs); err != nil {
		return Task{}, err
	}
	if before.LeaderActorID != 0 {

		eligible, err := s.canLeadWithTeams(
			ctx, before.ProjectID, teamIDs, before.LeaderActorID,
		)
		if err != nil {
			return Task{}, err
		}
		if !eligible {
			return Task{}, ErrLeadNotAllowed
		}
	}
	if err := s.setTaskTeams(ctx, id, teamIDs); err != nil {
		return Task{}, err
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logOwnership(ctx, before, after)
	return after, nil
}

func leadRequired(state State) bool {
	return state != StateBacklog && state != StateCancelled
}

func (s *Service) canLead(
	ctx context.Context,
	projectID, taskID string,
	actorID ids.ActorID,
) (bool, error) {
	if actorID == 0 {
		return false, nil
	}
	inTeam, err := s.store.IsActorInTaskTeam(ctx, store.IsActorInTaskTeamParams{
		TaskID:  taskID,
		ActorID: int64(actorID),
	})
	if err != nil {
		return false, err
	}
	if inTeam {
		return true, nil
	}
	if strings.TrimSpace(projectID) == "" {
		return false, nil
	}
	return s.projects.ActorCanPrompt(ctx, projectID, actorID)
}

func (s *Service) canLeadWithTeams(
	ctx context.Context,
	projectID string,
	teamIDs []int64,
	actorID ids.ActorID,
) (bool, error) {
	if actorID == 0 {
		return false, nil
	}
	for _, teamID := range teamIDs {
		members, err := s.store.ListEffectiveTeamMemberActorIDs(ctx, teamID)
		if err != nil {
			return false, err
		}
		for _, member := range members {
			if ids.ActorID(member) == actorID {
				return true, nil
			}
		}
	}
	if strings.TrimSpace(projectID) == "" {
		return false, nil
	}
	return s.projects.ActorCanPrompt(ctx, projectID, actorID)
}

func (s *Service) resolveCreateLeader(
	ctx context.Context,
	projectID string,
	teamIDs []int64,
	automated bool,
	explicit ids.ActorID,
) (ids.ActorID, error) {
	if automated {
		return 0, nil
	}
	actorID := explicit
	if actorID == 0 {
		actorID = actorIDFrom(ctx)
	}
	if actorID == 0 {
		return 0, nil
	}
	eligible, err := s.canLeadWithTeams(ctx, projectID, teamIDs, actorID)
	if err != nil {
		return 0, err
	}
	if !eligible {
		if explicit != 0 {
			return 0, ErrLeadNotAllowed
		}
		return 0, nil
	}
	return actorID, nil
}

func (s *Service) requireLeader(ctx context.Context, id string, state State) error {
	if actorIDFrom(ctx) == 0 {
		return nil
	}
	t, err := s.store.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if t.Automated != 0 {
		return nil
	}
	if !leadRequired(state) {
		return nil
	}
	leaders, err := s.store.ListTaskLeadsByTask(ctx, id)
	if err != nil {
		return err
	}
	if len(leaders) == 0 {
		return ErrLeadRequired
	}
	return nil
}

func (s *Service) validateTaskLeader(
	ctx context.Context,
	projectID string,
	id string,
	leaderActorID ids.ActorID,
) error {
	if leaderActorID == 0 {
		return nil
	}
	eligible, err := s.canLead(ctx, projectID, id, leaderActorID)
	if err != nil {
		return err
	}
	if !eligible {
		return ErrLeadNotAllowed
	}
	return nil
}

func (s *Service) SetSubscribed(
	ctx context.Context,
	id string,
	actorID ids.ActorID,
	subscribed bool,
) (Task, error) {
	if actorID == 0 {
		return Task{}, ErrActorRequired
	}
	if _, err := s.Get(ctx, id); err != nil {
		return Task{}, err
	}
	if subscribed {
		if err := s.store.AddTaskSubscription(ctx, store.AddTaskSubscriptionParams{
			TaskID:    id,
			ActorID:   int64(actorID),
			CreatedAt: time.Now().UnixMilli(),
		}); err != nil {
			return Task{}, err
		}
	} else {
		if err := s.store.DeleteTaskSubscription(ctx, store.DeleteTaskSubscriptionParams{
			TaskID:  id,
			ActorID: int64(actorID),
		}); err != nil {
			return Task{}, err
		}
	}
	t, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}

	t.Subscribed = subscribed
	return t, nil
}

func (s *Service) SetFavorite(
	ctx context.Context,
	id string,
	actorID ids.ActorID,
	favorite bool,
) (Task, error) {
	if actorID == 0 {
		return Task{}, ErrActorRequired
	}
	if _, err := s.Get(ctx, id); err != nil {
		return Task{}, err
	}
	if favorite {
		if err := s.store.AddTaskFavorite(ctx, store.AddTaskFavoriteParams{
			TaskID:    id,
			ActorID:   int64(actorID),
			CreatedAt: time.Now().UnixMilli(),
		}); err != nil {
			return Task{}, err
		}
	} else {
		if err := s.store.DeleteTaskFavorite(ctx, store.DeleteTaskFavoriteParams{
			TaskID:  id,
			ActorID: int64(actorID),
		}); err != nil {
			return Task{}, err
		}
	}
	t, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}

	t.Favorite = favorite
	return t, nil
}

func (s *Service) ResolveProjectID(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	if s.projects == nil {
		return ref, nil
	}
	project, err := s.projects.Get(ctx, ref)
	if err == nil {
		return project.ID, nil
	}

	if lower := strings.ToLower(ref); lower != ref {
		if project, lowerErr := s.projects.Get(ctx, lower); lowerErr == nil {
			return project.ID, nil
		}
	}
	return "", err
}

func (s *Service) CanView(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) (bool, error) {
	if isAdmin || actorID == 0 || strings.TrimSpace(projectID) == "" {
		return true, nil
	}
	if s.projects == nil {
		return false, nil
	}
	_, ok, err := s.projects.EffectiveRole(ctx, projectID, actorID, false)
	return ok, err
}

func (s *Service) CanCreateInProject(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
) (bool, error) {
	if projectID == "" || actorID == 0 {
		return true, nil
	}
	return s.projects.ActorCanPrompt(ctx, projectID, actorID)
}

func (s *Service) CanUpdate(
	ctx context.Context,
	t Task,
	actorID ids.ActorID,
	isAdmin bool,
) (bool, error) {
	if isAdmin {
		return true, nil
	}
	if actorID == 0 {
		return false, nil
	}
	if t.AssigneeActorID != 0 && t.AssigneeActorID == actorID {
		return true, nil
	}
	if t.CreatorActorID != 0 && t.CreatorActorID == actorID {
		return true, nil
	}
	if strings.TrimSpace(t.ProjectID) == "" || s.projects == nil {
		return false, nil
	}
	return s.projects.ActorCanPrompt(ctx, t.ProjectID, actorID)
}

func (s *Service) CanPrompt(
	ctx context.Context,
	t Task,
	actorID ids.ActorID,
	isAdmin bool,
) (bool, error) {
	if isAdmin {
		return true, nil
	}
	if actorID == 0 {
		return false, nil
	}
	if t.CreatorActorID != 0 && actorID == t.CreatorActorID {
		return true, nil
	}
	teams, err := s.store.ListTaskTeamsByTask(ctx, t.ID)
	if err != nil {
		return false, err
	}
	if len(teams) == 0 {
		if s.projects == nil || strings.TrimSpace(t.ProjectID) == "" {
			return false, nil
		}
		return s.projects.ActorCanPrompt(ctx, t.ProjectID, actorID)
	}
	in, err := s.store.IsActorInTaskTeam(ctx, store.IsActorInTaskTeamParams{
		TaskID:  t.ID,
		ActorID: int64(actorID),
	})
	if err != nil {
		return false, err
	}
	if in {
		return true, nil
	}
	return s.store.IsActorTaskLead(ctx, store.IsActorTaskLeadParams{
		TaskID:  t.ID,
		ActorID: int64(actorID),
	})
}

func (s *Service) WakeState(ctx context.Context, id string) (WakeState, error) {
	row, err := s.store.GetTask(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return WakeState{}, ErrNotFound
	}
	if err != nil {
		return WakeState{}, err
	}
	return parseWakeState(row.PendingSteering), nil
}

func (s *Service) SetWakeState(ctx context.Context, id string, st WakeState) error {
	if _, err := s.store.SetTaskPendingSteering(ctx, store.SetTaskPendingSteeringParams{
		PendingSteering: st.JSON(),
		UpdatedAt:       time.Now().UnixMilli(),
		ID:              id,
	}); err != nil {
		return err
	}
	s.changed(ctx, id)
	return nil
}

func (s *Service) MarkPendingPromptDelivered(ctx context.Context, id, text string) error {
	st, err := s.WakeState(ctx, id)
	if err != nil {
		return err
	}
	st.DeliveredPrompt = strings.TrimSpace(text)
	return s.SetWakeState(ctx, id, st)
}

func (s *Service) ResetWakeState(ctx context.Context, id string) error {
	return s.SetWakeState(ctx, id, WakeState{Keys: map[string]string{}})
}

func (s *Service) clearBlockerWakeKeys(ctx context.Context, blockerID string) {
	blocked, err := s.store.ListBlockedEdgesByBlocker(ctx, blockerID)
	if err != nil {
		log.Printf("[tasks] list blocked edges of %s: %v", blockerID, err)
		return
	}
	key := blockerWakeKey(blockerID)
	for _, edge := range blocked {
		st, err := s.WakeState(ctx, edge.BlockedTaskID)
		if err != nil {
			continue
		}
		if _, ok := st.Keys[key]; !ok {
			continue
		}
		delete(st.Keys, key)
		if err := s.SetWakeState(ctx, edge.BlockedTaskID, st); err != nil {
			log.Printf(
				"[tasks] clear blocker wake key %s/%s: %v",
				edge.BlockedTaskID, blockerID, err,
			)
		}
	}
}

func blockerWakeKey(blockerID string) string {
	return "block:" + blockerID
}

func (s *Service) SetAgentLastOnline(
	ctx context.Context,
	agentID ids.AgentID,
	at time.Time,
) error {
	stamp := at.UnixMilli()
	if at.IsZero() {
		stamp = 0
	}
	return s.store.SetAgentLastOnline(ctx, agentID, stamp)
}

func (s *Service) LiveReactions(ctx context.Context, originID string) ([]Task, error) {
	rows, err := s.store.ListLiveReactionTasksByOrigin(ctx, originID)
	if err != nil {
		return nil, err
	}
	out := make([]Task, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTask(row, nil, nil, nil, nil, emptyProgress()))
	}
	return out, nil
}

func (s *Service) StalledReactions(ctx context.Context, cutoff time.Time) ([]Task, error) {
	rows, err := s.store.ListStalledReactionTasks(ctx, cutoff.UnixMilli())
	if err != nil {
		return nil, err
	}
	out := make([]Task, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTask(row, nil, nil, nil, nil, emptyProgress()))
	}
	return out, nil
}

func (s *Service) BumpReactionReminders(ctx context.Context, id string) (int, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return 0, err
	}
	next := cur.ReactionReminders + 1
	if _, err := s.store.SetTaskReactionReminders(ctx, store.SetTaskReactionRemindersParams{
		Count:     int64(next),
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	}); err != nil {
		return 0, err
	}
	return next, nil
}

func (s *Service) ResetReactionReminders(ctx context.Context, id string) error {
	_, err := s.store.SetTaskReactionReminders(ctx, store.SetTaskReactionRemindersParams{
		Count:     0,
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	return err
}

func (s *Service) SetNeedsOperator(ctx context.Context, id, reason string) error {
	_, err := s.store.SetTaskNeedsOperator(ctx, store.SetTaskNeedsOperatorParams{
		Reason:    strings.TrimSpace(reason),
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	if err != nil {
		return err
	}
	s.changed(ctx, id)
	return nil
}

func (s *Service) FailNeedsOperator(ctx context.Context, id, reason string) error {
	if err := s.SetNeedsOperator(ctx, id, reason); err != nil {
		return err
	}
	return s.FailForWakeCap(ctx, id, reason)
}

type BlockedEdge struct {
	BlockedTaskID         string
	WakeOnResolveWhenMany bool
}

func (s *Service) BlockedEdges(
	ctx context.Context,
	blockerID string,
) ([]BlockedEdge, error) {
	rows, err := s.store.ListBlockedEdgesByBlocker(ctx, blockerID)
	if err != nil {
		return nil, err
	}
	out := make([]BlockedEdge, 0, len(rows))
	for _, row := range rows {
		out = append(out, BlockedEdge{
			BlockedTaskID:         row.BlockedTaskID,
			WakeOnResolveWhenMany: row.WakeOnResolveWhenMany != 0,
		})
	}
	return out, nil
}

type PendingSteeringRow struct {
	ID              string
	PendingSteering string
}

func (s *Service) PendingSteeringTasks(
	ctx context.Context,
) ([]PendingSteeringRow, error) {
	rows, err := s.store.ListPendingSteeringTasks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PendingSteeringRow, len(rows))
	for i, row := range rows {
		out[i] = PendingSteeringRow{ID: row.ID, PendingSteering: row.PendingSteering}
	}
	return out, nil
}

func ParseWakeStateRow(raw string) (WakeState, bool) {
	if strings.TrimSpace(raw) == "" {
		return WakeState{}, false
	}
	return parseWakeState(raw), true
}

func (s *Service) AppendPendingPrompt(
	ctx context.Context, id, text string, actorID ids.ActorID,
) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	_, err := s.queuePendingMessage(ctx, id, QueueParams{
		Text:    text,
		ActorID: actorID,
	})
	return err
}

func (s *Service) FailForWakeCap(ctx context.Context, id, msg string) error {
	t, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if t.State == StateRunning || t.State == StateInput {
		_, err = s.Fail(ctx, id, msg)
		return err
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskFailResting,
		Error:     msg,
		SettledAt: sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrWrongState
	}

	if err := s.clearRequiredTools(ctx, t); err != nil {
		log.Printf("[tasks] clear required tools after wake cap %s: %v", id, err)
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return err
	}
	s.transitioned(ctx, id, StateFailed)
	s.logChanges(ctx, t, after)
	return nil
}

func (s *Service) taskFinished(ctx context.Context, id string, outcome FinishOutcome) {
	if s.OnTaskFinished == nil {
		return
	}
	t, err := s.Get(ctx, id)
	if err != nil {
		return
	}
	s.OnTaskFinished(ctx, t, outcome)
}

func (s *Service) CanCloseOut(
	ctx context.Context,
	t Task,
	actorID ids.ActorID,
	isAdmin bool,
) (bool, error) {
	if isAdmin {
		return true, nil
	}
	leader, err := s.IsLead(ctx, t.ID, actorID)
	if err != nil || leader {
		return leader, err
	}

	return s.IsReactionFor(ctx, t.ID, actorID)
}

func (s *Service) IsReactionFor(
	ctx context.Context,
	taskID string,
	actorID ids.ActorID,
) (bool, error) {
	if actorID == 0 {
		return false, nil
	}
	rows, err := s.store.ListLiveReactionTasksByOrigin(ctx, taskID)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.AssigneeActorID.Valid && ids.ActorID(row.AssigneeActorID.Int64) == actorID {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) IsLead(
	ctx context.Context,
	taskID string,
	actorID ids.ActorID,
) (bool, error) {
	if actorID == 0 {
		return false, nil
	}
	return s.store.IsActorTaskLead(ctx, store.IsActorTaskLeadParams{
		TaskID:  taskID,
		ActorID: int64(actorID),
	})
}

func (s *Service) CanAnswerInput(
	ctx context.Context,
	t Task,
	actorID ids.ActorID,
	isAdmin bool,
) (bool, error) {
	return s.CanCloseOut(ctx, t, actorID, isAdmin)
}

func (s *Service) Recipients(ctx context.Context, id string) ([]ids.ActorID, error) {
	rows, err := s.store.ListTaskNotificationRecipients(ctx, id)
	if err != nil {
		return nil, err
	}
	return actorIDsFromInt64s(rows), nil
}

func (s *Service) NotificationRecipients(
	ctx context.Context,
	id string,
) ([]ids.ActorID, bool, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}
	recipients, err := s.Recipients(ctx, id)
	if err != nil {
		return nil, false, err
	}
	return recipients, len(recipients) == 0 && !t.Automated, nil
}

func (s *Service) HasTeams(ctx context.Context, id string) (bool, error) {
	teams, err := s.store.ListTaskTeamsByTask(ctx, id)
	if err != nil {
		return false, err
	}
	return len(teams) > 0, nil
}

type Sender struct {
	ActorID  ids.ActorID
	Username string
	Name     string
	Kind     string
}

func (s *Service) Sender(ctx context.Context, actorID ids.ActorID) (Sender, error) {
	if actorID == 0 {
		return Sender{Kind: "system", Username: "system", Name: "CherryPi"}, nil
	}
	actor, err := s.store.GetActorByID(ctx, actorID)
	if errors.Is(err, sql.ErrNoRows) {
		return Sender{Kind: "system", Username: "system", Name: "CherryPi"}, nil
	}
	if err != nil {
		return Sender{}, err
	}
	name := actor.Name
	if strings.TrimSpace(name) == "" {
		name = actor.Username
	}
	return Sender{
		ActorID:  actorID,
		Username: actor.Username,
		Name:     name,
		Kind:     actor.Kind,
	}, nil
}

func toTask(
	row store.Task,
	tags []Tag,
	links []Link,
	refs, referencedBy []Reference,
	progress Progress,
) Task {
	if tags == nil {
		tags = []Tag{}
	}
	if links == nil {
		links = []Link{}
	}
	if refs == nil {
		refs = []Reference{}
	}
	if referencedBy == nil {
		referencedBy = []Reference{}
	}
	state := State(row.State)
	return Task{
		RowID:             row.Rowid,
		ID:                row.ID,
		Seq:               row.Seq,
		Prompt:            textutil.Clean(row.Prompt),
		Name:              textutil.Clean(row.Name),
		ProjectID:         row.ProjectID.String,
		OriginTaskID:      row.OriginTaskID.String,
		AssigneeActorID:   ids.ActorID(row.AssigneeActorID.Int64),
		AssigneePinned:    row.AssigneePinned != 0,
		Priority:          row.Priority,
		Size:              normalizeSize(row.Size),
		State:             state,
		SessionID:         row.SessionID,
		SessionAgentID:    ids.AgentID(row.SessionAgentID.Int64),
		Error:             textutil.Clean(row.Error),
		ModelProvider:     row.ModelProvider,
		ModelID:           row.ModelID,
		ThinkingLevel:     row.ThinkingLevel,
		Automated:         row.Automated != 0,
		Reaction:          row.Reaction != 0,
		RequiredTools:     parseRequiredTools(row.RequiredTools),
		NeedsOperator:     textutil.Clean(row.NeedsOperator),
		ReactionReminders: int(row.ReactionReminders),
		Workspace:         row.Workspace,
		AutoTitle:         row.AutoTitle != 0,
		PromptIDs: prompts.VersionIDs{
			Org:          row.OrgPromptID.Int64,
			Agent:        row.AgentPromptID.Int64,
			Project:      row.ProjectPromptID.Int64,
			AgentProject: row.AgentProjectPromptID.Int64,
		},
		Tags:                 tags,
		Links:                links,
		Progress:             progressOrEmpty(progress),
		References:           refs,
		ReferencedBy:         referencedBy,
		PendingSteering:      strings.Join(parseWakeState(row.PendingSteering).Lines, "\n"),
		Blockers:             countBlockers(refs),
		CreatorActorID:       ids.ActorID(row.CreatorActorID.Int64),
		PendingPrompt:        textutil.Clean(row.PendingPrompt),
		PendingPromptActorID: ids.ActorID(row.PendingPromptActorID.Int64),
		QueuedAt:             timeOrZero(row.QueuedAt),
		StartedAt:            timeOrZero(row.StartedAt),
		SettledAt:            timeOrZero(row.SettledAt),
		CreatedAt:            time.UnixMilli(row.CreatedAt),
		UpdatedAt:            time.UnixMilli(row.UpdatedAt),
		StateUpdatedAt:       time.UnixMilli(row.StateUpdatedAt),
		TtcMs:                ttcMs(row.State, row.StateUpdatedAt, row.TtcMs),
		SessionDeletedAt:     timeOrZero(row.SessionDeletedAt),
		Archived:             row.ArchivedAt.Valid,
		ArchivedAt:           timeOrZero(row.ArchivedAt),
	}
}

func ttcMs(state string, stateUpdatedAt, accumulated int64) int64 {
	if !CountsTtc(State(state)) || stateUpdatedAt <= 0 {
		return accumulated
	}
	if live := time.Now().UnixMilli() - stateUpdatedAt; live > 0 {
		return accumulated + live
	}
	return accumulated
}

func IncompleteBlockerIDs(refs []Reference) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Kind != ReferenceBlocking {
			continue
		}
		if ref.State == StateCompleted || ref.State == StateCancelled {
			continue
		}
		out = append(out, ref.ID)
	}
	return out
}

func countBlockers(refs []Reference) int {
	return len(IncompleteBlockerIDs(refs))
}

func (s *Service) setTeamPromptPins(
	ctx context.Context,
	id string,
	pins []prompts.TeamVersionIDs,
) error {
	if err := s.store.DeleteTaskTeamPrompts(ctx, id); err != nil {
		return err
	}
	for _, pin := range pins {
		if err := s.store.AddTaskTeamPrompt(ctx, store.AddTaskTeamPromptParams{
			TaskID:          id,
			TeamID:          pin.TeamID,
			PromptID:        nullID(pin.Prompt),
			ProjectPromptID: nullID(pin.ProjectPrompt),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) setGroupPromptPins(
	ctx context.Context,
	id string,
	pins []prompts.GroupVersionIDs,
) error {
	if err := s.store.DeleteTaskGroupPrompts(ctx, id); err != nil {
		return err
	}
	if err := s.store.DeleteTaskGroupTeamPrompts(ctx, id); err != nil {
		return err
	}
	for _, pin := range pins {
		if err := s.store.AddTaskGroupPrompt(ctx, store.AddTaskGroupPromptParams{
			TaskID:          id,
			GroupID:         pin.GroupID,
			PromptID:        nullID(pin.Prompt),
			ProjectPromptID: nullID(pin.ProjectPrompt),
		}); err != nil {
			return err
		}
		for _, team := range pin.Teams {
			if err := s.store.AddTaskGroupTeamPrompt(
				ctx,
				store.AddTaskGroupTeamPromptParams{
					TaskID:   id,
					GroupID:  pin.GroupID,
					TeamID:   team.TeamID,
					PromptID: nullID(team.Prompt),
				},
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func nullPromptIDs(
	p prompts.VersionIDs,
) (sql.NullInt64, sql.NullInt64, sql.NullInt64, sql.NullInt64) {
	return nullID(p.Org), nullID(p.Agent), nullID(p.Project), nullID(p.AgentProject)
}

func nullID[T ~int64](id T) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(id), Valid: id != 0}
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullPositive(n int64) sql.NullInt64 {
	return sql.NullInt64{Int64: n, Valid: n > 0}
}

func nullBoolInt64Ptr(b *bool) sql.NullInt64 {
	if b == nil {
		return sql.NullInt64{}
	}
	if *b {
		return sql.NullInt64{Int64: 1, Valid: true}
	}
	return sql.NullInt64{Int64: 0, Valid: true}
}

func nullInt64Ptr(n *int64) sql.NullInt64 {
	if n == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *n, Valid: true}
}

func normalizeSize(size string) string {
	if size == "" {
		return SizeMedium
	}
	return size
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func timeOrZero(t sql.NullInt64) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return time.UnixMilli(t.Int64)
}

func (s *Service) TaskIDBySession(
	ctx context.Context,
	agentID ids.AgentID,
	sessionID string,
) (string, error) {
	id, err := s.store.TaskIDBySession(ctx, store.TaskIDBySessionParams{
		AgentID:   int64(agentID),
		SessionID: sessionID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

const RunFailedReason = "the agent's run failed"

func (s *Service) FailBySession(
	ctx context.Context,
	agentID ids.AgentID,
	sessionID, errMsg string,
) error {
	id, idErr := s.store.TaskIDBySession(ctx, store.TaskIDBySessionParams{
		AgentID:   int64(agentID),
		SessionID: sessionID,
	})
	var before Task
	if idErr == nil {
		before, _ = s.snapshot(ctx, id)
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskFailBySession,
		Error:     errMsg,
		SettledAt: sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		AgentID:   agentID,
		SessionID: sessionID,
		NotBefore: sql.NullInt64{
			Int64: now - SettleGrace.Milliseconds(),
			Valid: true,
		},
	})
	if err != nil {
		return err
	}
	if n == 0 || idErr != nil {
		return nil
	}
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return nil
	}
	s.logChanges(ctx, before, after)
	return nil
}

func (s *Service) Enrich(ctx context.Context, id, name, prompt string) error {
	name = strings.TrimSpace(name)
	prompt = strings.TrimSpace(prompt)
	if name == "" && prompt == "" {
		return nil
	}
	return s.store.SetTaskNamePrompt(ctx, store.SetTaskNamePromptParams{
		Name:   name,
		Prompt: prompt,
		ID:     id,
	})
}
