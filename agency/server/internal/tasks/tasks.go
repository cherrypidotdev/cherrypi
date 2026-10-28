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
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/store"
)

type State string

const (
	StateDraft     State = "draft"
	StateBacklog   State = "backlog"
	StateQueued    State = "queued"
	StateRunning   State = "running"
	StateInput     State = "input"
	StateSettled   State = "settled"
	StateFailed    State = "failed"
	StateCompleted State = "completed"
	StateCancelled State = "cancelled"
)

func HoldsSlot(state State) bool {
	return state == StateRunning || state == StateInput ||
		state == StateSettled || state == StateFailed
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

	ErrProjectRequired = errors.New("project is required")

	ErrNotDraft = errors.New("task is not a draft")

	ErrWrongState = errors.New("task is not in a valid state for this action")

	ErrRunning = errors.New("task is running; stop it or let it settle first")

	ErrTagMissing = errors.New("tag not found")

	ErrNotMember = errors.New("agent is not a member of the project")

	ErrProjectHasNoAgents = errors.New(
		"project has no member agents; add one or save the task as a draft",
	)

	ErrSelfReference = errors.New("a task cannot reference itself")

	ErrReferenceCycle = errors.New("blocking reference would create a cycle")

	ErrReferenceNotAllowed = errors.New(
		"a task may only reference tasks in its own, a descendant, or a sibling project",
	)

	ErrMergeTooFew = errors.New("select at least two tasks to merge")

	ErrMergeProjectMismatch = errors.New("tasks to merge must share one project")

	ErrMergeAssigneeConflict = errors.New("tasks to merge have conflicting assignees")

	ErrMergeState = errors.New("only draft, backlog, queued, or failed tasks can be merged")

	ErrTeamsRequired = errors.New("at least one owning team is required")

	ErrTeamNotInProject = errors.New("owning team is not attached to the project")

	ErrLeadNotAllowed = errors.New(
		"a lead must be a project member or belong to one of the task's owning teams",
	)

	ErrCannotPrompt = errors.New("not allowed to prompt this task")

	ErrFollowUpRequired = errors.New(
		"task has already started; send a follow-up prompt to queue it",
	)

	ErrPostponedNotDraft = errors.New(
		"a task that has already started cannot return to draft",
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
)

type Reference struct {
	ID     string
	Name   string
	Prompt string
	State  State
	Kind   ReferenceKind
}

func (r Reference) Blocking() bool { return r.Kind == ReferenceBlocking }

type ReferenceInput struct {
	TaskID string
	Kind   ReferenceKind
}

type Task struct {
	RowID           int64
	ID              string
	Seq             int64
	Prompt          string
	Name            string
	ProjectID       string
	AssigneeActorID ids.ActorID

	CreatorActorID ids.ActorID

	TeamIDs []int64

	LeadActorIDs []ids.ActorID

	Subscribed bool

	Favorite       bool
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
	Interactive    bool
	Workspace      string
	AutoTitle      bool
	PromptIDs      prompts.VersionIDs

	TeamPromptIDs []prompts.TeamVersionIDs
	Tags          []Tag
	Links         []Link
	References    []Reference

	ReferencedBy []Reference

	Blockers int

	PendingPrompt string

	PendingPromptActorID ids.ActorID
	QueuedAt             time.Time
	StartedAt            time.Time
	SettledAt            time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time

	StateUpdatedAt time.Time

	SessionDeletedAt time.Time

	Archived   bool
	ArchivedAt time.Time
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

	ReferenceProjectID string

	Limit  int64
	Offset int64

	VisibleToActorID ids.ActorID

	VisibleToAdmin bool
}

type CreateParams struct {
	Prompt          string
	Name            string
	ProjectID       string
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

	LeadActorIDs []ids.ActorID
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

type AttributeSet struct {
	Priority        *int64
	Size            *string
	AssigneeActorID *ids.ActorID

	Prompt *string

	PendingPrompt *string

	AddTagIDs    []string
	RemoveTagIDs []string
}

func (a AttributeSet) Empty() bool {
	return a.Priority == nil && a.Size == nil && a.AssigneeActorID == nil &&
		a.Prompt == nil &&
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

type Transition struct {
	TaskID string
	State  State
}

type Service struct {
	store    store.Store
	projects *projects.Service

	OnTransition func(ctx context.Context, t Transition)

	OnChange func(ctx context.Context, taskID string)
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
	teamIDs, err := s.resolveTaskTeams(ctx, project.ID, p.TeamIDs)
	if err != nil {
		return Task{}, err
	}
	initial := normalizeCreateState(p.State)
	if initial != StateDraft {
		if err := s.requireExecutor(ctx, project.ID, p.AssigneeActorID); err != nil {
			return Task{}, err
		}
	}

	refs, err := s.resolveReferences(ctx, project.ID, p.References)
	if err != nil {
		return Task{}, err
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
		AssigneeActorID: nullID(p.AssigneeActorID),
		AssigneePinned:  boolInt(p.AssigneeActorID != 0),
		Priority:        p.Priority,
		CreatorActorID:  nullID(actorIDFrom(ctx)),
		Size:            normalizeSize(p.Size),
		State:           string(StateDraft),
		ModelProvider:   p.ModelProvider,
		ModelID:         p.ModelID,
		ThinkingLevel:   p.ThinkingLevel,
		Interactive:     1,
		Workspace:       p.Workspace,
		AutoTitle:       boolInt(p.AutoTitle),
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return Task{}, err
	}
	if err := s.SetTags(ctx, row.ID, p.TagIDs); err != nil {
		return Task{}, err
	}
	if err := s.setTaskTeams(ctx, row.ID, teamIDs); err != nil {
		return Task{}, err
	}
	if err := s.setTaskLeads(ctx, project.ID, row.ID, p.LeadActorIDs); err != nil {
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
		if _, err := s.AddReference(internal, row.ID, ref.TaskID, ref.Kind); err != nil {
			return Task{}, err
		}
	}
	switch initial {
	case StateBacklog:
		if _, err := s.Promote(internal, row.ID); err != nil {
			return Task{}, err
		}
	case StateQueued:

		if _, err := s.Queue(internal, row.ID); err != nil {
			return Task{}, err
		}
	}
	t, err := s.Get(ctx, row.ID)
	if err != nil {
		return Task{}, err
	}
	s.logCreated(ctx, t)
	return t, nil
}

func normalizeCreateState(state State) State {
	switch state {
	case StateDraft, StateBacklog, StateQueued:
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
		if err := s.checkReferenceProject(
			ctx,
			fromProjectID,
			target.ProjectID.String,
		); err != nil {
			return nil, err
		}
		seen[id] = true
		out = append(out, ReferenceInput{TaskID: id, Kind: normalizeReferenceKind(ref.Kind)})
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
	case ReferenceBlocking, ReferenceSupersedes:
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
	refs, err := s.allReferences(ctx)
	if err != nil {
		return Task{}, err
	}
	referencedBy, err := s.allReferencedBy(ctx)
	if err != nil {
		return Task{}, err
	}
	t := toTask(row, tags, links, refs[id], referencedBy[id])
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
		t.TeamPromptIDs = append(t.TeamPromptIDs, prompts.TeamVersionIDs{
			TeamID:        pin.TeamID,
			Prompt:        pin.PromptID.Int64,
			ProjectPrompt: pin.ProjectPromptID.Int64,
		})
	}
	leads, err := s.store.ListTaskLeadsByTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	t.LeadActorIDs = actorIDsFromInt64s(leads)
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
	}
	return t, nil
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
	return toTask(row, nil, nil, nil, nil), nil
}

func (s *Service) List(ctx context.Context, f Filter) ([]Task, error) {
	rows, err := s.store.ListTasks(ctx, store.ListTasksParams{
		ProjectID:          nullString(f.ProjectID),
		AgentID:            nullID(f.AgentID),
		State:              nullString(string(f.State)),
		Archived:           boolInt(f.Archived),
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
			if !row.ProjectID.Valid || row.ProjectID.String == "" || visible[row.ProjectID.String] {
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
	refsByTask, err := s.allReferences(ctx)
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
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(row, tagsByTask[row.ID], linksByTask[row.ID], refsByTask[row.ID], nil)
		out[i].TeamIDs = teamsByTask[row.ID]
		out[i].LeadActorIDs = actorIDsFromInt64s(leadsByTask[row.ID])
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
	}

	prioritizeSearchMatches(out, f.Search)
	return out, nil
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
	tagsByTask, err := s.allTags(ctx)
	if err != nil {
		return nil, err
	}
	linksByTask, err := s.allLinks(ctx)
	if err != nil {
		return nil, err
	}
	refsByTask, err := s.allReferences(ctx)
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
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(row, tagsByTask[row.ID], linksByTask[row.ID], refsByTask[row.ID], nil)
		out[i].TeamIDs = teamsByTask[row.ID]
		out[i].LeadActorIDs = actorIDsFromInt64s(leadsByTask[row.ID])
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
	refsByTask, err := s.allReferences(ctx)
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
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(row, tagsByTask[row.ID], linksByTask[row.ID], refsByTask[row.ID], nil)
		out[i].TeamIDs = teamsByTask[row.ID]
		out[i].LeadActorIDs = actorIDsFromInt64s(leadsByTask[row.ID])
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (Task, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if cur.State != StateDraft {
		return Task{}, ErrNotDraft
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

	newID := cur.ID
	number := cur.Seq
	if project.ID != cur.ProjectID {
		number, err = s.store.AllocateTaskNumber(ctx, project.ID)
		if err != nil {
			return Task{}, err
		}
		newID = TaskID(project.Prefix, number)
	}

	if _, err := s.store.UpdateDraftTask(ctx, store.UpdateDraftTaskParams{
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
		Interactive:     1,
		Workspace:       p.Workspace,
		AutoTitle:       boolInt(p.AutoTitle),
		UpdatedAt:       time.Now().UnixMilli(),
		ID:              id,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotDraft
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
	cur, err := s.requireStates(ctx, id, StateDraft, StateBacklog)
	if err != nil {
		return Task{}, err
	}
	if !cur.StartedAt.IsZero() {
		return Task{}, ErrFollowUpRequired
	}
	if err := s.requireExecutor(ctx, cur.ProjectID, cur.AssigneeActorID); err != nil {
		return Task{}, err
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskQueue,
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

func (s *Service) Promote(ctx context.Context, id string) (Task, error) {
	cur, err := s.requireState(ctx, id, StateDraft)
	if err != nil {
		return Task{}, err
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(
		ctx,
		store.TransitionTaskParams{Event: store.TaskPromote, UpdatedAt: now, ID: id},
	)
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

func (s *Service) Demote(ctx context.Context, id string) (Task, error) {
	cur, err := s.requireState(ctx, id, StateBacklog)
	if err != nil {
		return Task{}, err
	}
	if !cur.StartedAt.IsZero() {
		return Task{}, ErrPostponedNotDraft
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(
		ctx,
		store.TransitionTaskParams{Event: store.TaskDemote, UpdatedAt: now, ID: id},
	)
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

func (s *Service) Dequeue(ctx context.Context, id string) (Task, error) {
	cur, err := s.requireState(ctx, id, StateQueued)
	if err != nil {
		return Task{}, err
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
	refsByTask, err := s.allReferences(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Task, len(rows))
	for i, row := range rows {
		out[i] = toTask(row, tagsByTask[row.ID], linksByTask[row.ID], refsByTask[row.ID], nil)
	}
	return out, nil
}

func (s *Service) AddReference(
	ctx context.Context,
	taskID, referencedID string,
	kind ReferenceKind,
) (Task, error) {
	kind = normalizeReferenceKind(kind)
	cur, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	if taskID == referencedID {
		return Task{}, ErrSelfReference
	}
	referenced, err := s.Get(ctx, referencedID)
	if err != nil {
		return Task{}, err
	}
	if err := s.checkReferenceProject(ctx, cur.ProjectID, referenced.ProjectID); err != nil {
		return Task{}, err
	}
	if kind == ReferenceBlocking {
		cycle, err := s.hasBlockingReferencePath(ctx, referencedID, taskID)
		if err != nil {
			return Task{}, err
		}
		if cycle {
			return Task{}, ErrReferenceCycle
		}
	}
	if err := s.store.AddTaskReference(ctx, store.UpsertTaskReferenceParams{
		TaskID:           taskID,
		ReferencedTaskID: referencedID,
		Kind:             string(kind),
		CreatedAt:        time.Now().UnixMilli(),
	}); err != nil {
		return Task{}, err
	}
	if kind == ReferenceSupersedes {
		if err := s.cancelSuperseded(ctx, referencedID); err != nil {
			return Task{}, err
		}
	}
	after, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) cancelSuperseded(ctx context.Context, id string) error {
	t, err := s.snapshot(ctx, id)
	if err != nil {
		return err
	}
	switch t.State {
	case StateDraft, StateBacklog, StateQueued, StateFailed:
		_, err = s.Cancel(ctx, id)
		return err
	default:
		return nil
	}
}

func (s *Service) RemoveReference(ctx context.Context, taskID, referencedID string) (Task, error) {
	cur, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	if _, err := s.store.RemoveTaskReference(ctx, store.RemoveTaskReferenceParams{
		TaskID:           taskID,
		ReferencedTaskID: referencedID,
	}); err != nil {
		return Task{}, err
	}
	after, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
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
	existing, err := s.store.ListTaskReferenceIDs(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	for _, id := range existing {
		if _, ok := want[id]; !ok {
			if _, err := s.RemoveReference(ctx, taskID, id); err != nil {
				return Task{}, err
			}
		}
	}
	for _, ref := range target {
		if _, err := s.AddReference(ctx, taskID, ref.TaskID, ref.Kind); err != nil {
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
	leadSeen := make(map[ids.ActorID]bool)
	leadIDs := make([]ids.ActorID, 0)
	for _, t := range mergees {
		if t.Archived {
			return Task{}, ErrMergeState
		}
		if t.ProjectID != projectID {
			return Task{}, ErrMergeProjectMismatch
		}
		switch t.State {
		case StateDraft, StateBacklog, StateQueued, StateFailed:
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
		for _, leadID := range t.LeadActorIDs {
			if !leadSeen[leadID] {
				leadSeen[leadID] = true
				leadIDs = append(leadIDs, leadID)
			}
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
		State:           StateDraft,
		TeamIDs:         teamIDs,
		LeadActorIDs:    leadIDs,
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
		if _, err := s.AddReference(ctx, merged.ID, t.ID, ReferenceSupersedes); err != nil {
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

func (s *Service) Settle(ctx context.Context, id string) (bool, error) {
	before, beforeErr := s.snapshot(ctx, id)
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskSettle,
		SettledAt: sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return false, err
	}
	if n == 1 {
		s.transitioned(ctx, id, StateSettled)
		if after, err := s.snapshot(ctx, id); err == nil && beforeErr == nil {
			s.logChanges(ctx, before, after)
		}
	}
	return n == 1, nil
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
	if n == 1 && beforeErr == nil {
		if after, err := s.snapshot(ctx, id); err == nil {
			s.logChanges(ctx, before, after)
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
	return after, nil
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
	pending := prompt
	if undelivered := strings.TrimSpace(cur.PendingPrompt); undelivered != "" {
		pending = undelivered + "\n\n" + prompt
	}
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:                store.TaskRequeueWithPrompt,
		PendingPrompt:        pending,
		PendingPromptActorID: nullID(actorID),
		QueuedAt:             sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt:            now,
		ID:                   id,
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

func (s *Service) ClearPendingPrompt(ctx context.Context, id string) error {
	_, err := s.store.ClearTaskPendingPrompt(ctx, store.ClearTaskPendingPromptParams{
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	if err == nil {
		s.changed(ctx, id)
	}
	return err
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
	after, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logChanges(ctx, cur, after)
	return after, nil
}

func (s *Service) Reopen(ctx context.Context, id string, queue bool) (Task, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	switch cur.State {
	case StateSettled, StateFailed, StateCompleted, StateCancelled:
	default:
		return Task{}, ErrWrongState
	}
	state := StateBacklog
	if queue {

		if !cur.StartedAt.IsZero() && cur.State != StateFailed {
			return Task{}, ErrFollowUpRequired
		}
		if err := s.requireExecutor(ctx, cur.ProjectID, cur.AssigneeActorID); err != nil {
			return Task{}, err
		}
		state = StateQueued
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

	if attrs.changesTags() || attrs.Prompt != nil {
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
	now := time.Now().UnixMilli()
	if attrs.Prompt != nil {

		if cur.State != StateDraft && cur.State != StateBacklog {
			return Task{}, ErrNotDraft
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
			return Task{}, ErrNotDraft
		}
	}
	if attrs.PendingPrompt != nil {

		if cur.State != StateDraft {
			return Task{}, ErrNotDraft
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
			return Task{}, ErrNotDraft
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
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.store.DeleteTask(ctx, id); err != nil {
		return err
	}
	s.changed(ctx, id)
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

		if task.StartedAt.IsZero() {
			if err := s.setTeamPromptPins(ctx, task.ID, pins.Teams); err != nil {
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
	if err == nil {
		s.changed(ctx, id)
	}
	return err
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

func (s *Service) IsMember(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
) (bool, error) {
	if projectID == "" {
		return true, nil
	}
	return s.projects.IsMember(ctx, projectID, agentID)
}

func (s *Service) requireState(ctx context.Context, id string, want State) (Task, error) {
	return s.requireStates(ctx, id, want)
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
		member, err := s.projects.IsMember(ctx, project.ID, agentID)
		if err != nil {
			return projects.Project{}, err
		}
		if !member {
			return projects.Project{}, ErrNotMember
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
	if len(agents) == 0 {
		return ErrProjectHasNoAgents
	}
	return nil
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

func (s *Service) allReferences(ctx context.Context) (map[string][]Reference, error) {
	rows, err := s.store.ListTaskReferenceRefs(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]Reference{}
	for _, row := range rows {
		out[row.TaskID] = append(out[row.TaskID], Reference{
			ID:     row.ReferenceID,
			Name:   row.ReferenceName,
			Prompt: row.ReferencePrompt,
			State:  State(row.ReferenceState),
			Kind:   ReferenceKind(row.Kind),
		})
	}
	return out, nil
}

func (s *Service) allReferencedBy(ctx context.Context) (map[string][]Reference, error) {
	rows, err := s.store.ListTaskReferencedByRefs(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]Reference{}
	for _, row := range rows {
		out[row.TaskID] = append(out[row.TaskID], Reference{
			ID:     row.ReferenceID,
			Name:   row.ReferenceName,
			Prompt: row.ReferencePrompt,
			State:  State(row.ReferenceState),
			Kind:   ReferenceKind(row.Kind),
		})
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

func actorIDsFromInt64s(in []int64) []ids.ActorID {
	out := make([]ids.ActorID, len(in))
	for i, id := range in {
		out[i] = ids.ActorID(id)
	}
	return out
}

func dedupeActorIDs(in []ids.ActorID) []ids.ActorID {
	out := make([]ids.ActorID, 0, len(in))
	seen := make(map[ids.ActorID]bool, len(in))
	for _, id := range in {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
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
		if _, err := s.store.GetProjectTeamMembership(
			ctx,
			store.GetProjectTeamMembershipParams{ProjectID: projectID, TeamID: id},
		); errors.Is(err, sql.ErrNoRows) {
			return ErrTeamNotInProject
		} else if err != nil {
			return err
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
		rows, err := s.store.ListProjectTeamMemberships(ctx, projectID)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			teamIDs = append(teamIDs, row.TeamID)
		}
		teamIDs = dedupeInt64(teamIDs)
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

func (s *Service) SetTaskTeams(ctx context.Context, id string, teamIDs []int64) (Task, error) {
	before, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	teamIDs = dedupeInt64(teamIDs)
	if err := s.validateTaskTeams(ctx, before.ProjectID, teamIDs); err != nil {
		return Task{}, err
	}
	if err := s.setTaskTeams(ctx, id, teamIDs); err != nil {
		return Task{}, err
	}
	allowed, err := s.taskTeamMembers(ctx, id)
	if err != nil {
		return Task{}, err
	}
	kept := make([]ids.ActorID, 0, len(before.LeadActorIDs))
	for _, lead := range before.LeadActorIDs {
		if allowed[lead] {
			kept = append(kept, lead)
			continue
		}
		isMember, err := s.projects.ActorCanPrompt(ctx, before.ProjectID, lead)
		if err != nil {
			return Task{}, err
		}
		if isMember {
			kept = append(kept, lead)
		}
	}
	if err := s.setTaskLeads(ctx, before.ProjectID, id, kept); err != nil {
		return Task{}, err
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logOwnership(ctx, before, after)
	return after, nil
}

func (s *Service) setTaskLeads(
	ctx context.Context,
	projectID string,
	id string,
	leadActorIDs []ids.ActorID,
) error {
	if err := s.validateTaskLeads(ctx, projectID, id, leadActorIDs); err != nil {
		return err
	}
	if err := s.store.DeleteTaskLeads(ctx, id); err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for _, actorID := range dedupeActorIDs(leadActorIDs) {
		if err := s.store.AddTaskLead(ctx, store.AddTaskLeadParams{
			TaskID:  id,
			ActorID: int64(actorID),
			AddedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) SetTaskLeads(
	ctx context.Context,
	id string,
	leadActorIDs []ids.ActorID,
) (Task, error) {
	before, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if err := s.setTaskLeads(ctx, before.ProjectID, id, leadActorIDs); err != nil {
		return Task{}, err
	}
	after, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.logOwnership(ctx, before, after)
	return after, nil
}

func (s *Service) validateTaskLeads(
	ctx context.Context,
	projectID string,
	id string,
	leadActorIDs []ids.ActorID,
) error {
	leadActorIDs = dedupeActorIDs(leadActorIDs)
	if len(leadActorIDs) == 0 {
		return nil
	}
	allowed, err := s.taskTeamMembers(ctx, id)
	if err != nil {
		return err
	}
	for _, actorID := range leadActorIDs {
		if allowed[actorID] {
			continue
		}
		isMember, err := s.projects.ActorCanPrompt(ctx, projectID, actorID)
		if err != nil {
			return err
		}
		if !isMember {
			return ErrLeadNotAllowed
		}
	}
	return nil
}

func (s *Service) taskTeamMembers(ctx context.Context, id string) (map[ids.ActorID]bool, error) {
	members, err := s.store.ListTaskTeamMemberActorIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make(map[ids.ActorID]bool, len(members))
	for _, member := range members {
		out[ids.ActorID(member)] = true
	}
	return out, nil
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
		if strings.TrimSpace(t.ProjectID) == "" {
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

func (s *Service) Recipients(ctx context.Context, id string) ([]ids.ActorID, error) {
	rows, err := s.store.ListTaskNotificationRecipients(ctx, id)
	if err != nil {
		return nil, err
	}
	return actorIDsFromInt64s(rows), nil
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
		RowID:           row.Rowid,
		ID:              row.ID,
		Seq:             row.Seq,
		Prompt:          row.Prompt,
		Name:            row.Name,
		ProjectID:       row.ProjectID.String,
		AssigneeActorID: ids.ActorID(row.AssigneeActorID.Int64),
		AssigneePinned:  row.AssigneePinned != 0,
		Priority:        row.Priority,
		Size:            normalizeSize(row.Size),
		State:           state,
		SessionID:       row.SessionID,
		SessionAgentID:  ids.AgentID(row.SessionAgentID.Int64),
		Error:           row.Error,
		ModelProvider:   row.ModelProvider,
		ModelID:         row.ModelID,
		ThinkingLevel:   row.ThinkingLevel,
		Interactive:     row.Interactive != 0,
		Workspace:       row.Workspace,
		AutoTitle:       row.AutoTitle != 0,
		PromptIDs: prompts.VersionIDs{
			Org:          row.OrgPromptID.Int64,
			Agent:        row.AgentPromptID.Int64,
			Project:      row.ProjectPromptID.Int64,
			AgentProject: row.AgentProjectPromptID.Int64,
		},
		Tags:                 tags,
		Links:                links,
		References:           refs,
		ReferencedBy:         referencedBy,
		Blockers:             countBlockers(refs),
		CreatorActorID:       ids.ActorID(row.CreatorActorID.Int64),
		PendingPrompt:        row.PendingPrompt,
		PendingPromptActorID: ids.ActorID(row.PendingPromptActorID.Int64),
		QueuedAt:             timeOrZero(row.QueuedAt),
		StartedAt:            timeOrZero(row.StartedAt),
		SettledAt:            timeOrZero(row.SettledAt),
		CreatedAt:            time.UnixMilli(row.CreatedAt),
		UpdatedAt:            time.UnixMilli(row.UpdatedAt),
		StateUpdatedAt:       time.UnixMilli(row.StateUpdatedAt),
		SessionDeletedAt:     timeOrZero(row.SessionDeletedAt),
		Archived:             row.ArchivedAt.Valid,
		ArchivedAt:           timeOrZero(row.ArchivedAt),
	}
}

func IncompleteBlockerIDs(refs []Reference) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Kind == ReferenceBlocking && ref.State != StateCompleted {
			out = append(out, ref.ID)
		}
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
	_, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:     store.TaskFailBySession,
		Error:     errMsg,
		SettledAt: sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		AgentID:   agentID,
		SessionID: sessionID,
	})
	if err != nil || idErr != nil {
		return err
	}
	if after, err := s.snapshot(ctx, id); err == nil {
		s.logChanges(ctx, before, after)
	}
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
