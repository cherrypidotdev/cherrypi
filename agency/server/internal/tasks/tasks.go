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

// Package tasks owns the agency-side task lifecycle: the work item that a
// user drafts or queues and the scheduler later runs on an agent. It is the
// authority for state transitions and for the concurrency caps the scheduler
// checks; the tunnel forwarding lives in the scheduler and the gRPC layer.
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

// State is a task's agency-side lifecycle. The string values match the
// `tasks.state` CHECK constraint and are never renumbered.
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

// HoldsSlot reports whether a task in the given state occupies an agent's
// concurrency slot. Running, input, settled, and failed tasks do until they
// are closed out or returned to backlog/draft. Archived tasks never hold a
// slot (the running-count queries exclude them).
func HoldsSlot(state State) bool {
	return state == StateRunning || state == StateInput ||
		state == StateSettled || state == StateFailed
}

// Resolution is the close-out a user picks for a task that is not already
// completed or cancelled.
type Resolution string

const (
	ResolutionCompleted Resolution = "completed"
	ResolutionCancelled Resolution = "cancelled"
)

// Priority orders the scheduler's queue. The stored values are the ones the
// schema documents: -1 low, 0 normal, 1 high, 2 critical.
const (
	PriorityLow      int64 = -1
	PriorityNormal   int64 = 0
	PriorityHigh     int64 = 1
	PriorityCritical int64 = 2
)

// Size is a task's manually estimated scope. It is informational only: the
// scheduler ignores it, and it can change in any state. The stored values
// mirror the schema's CHECK constraint.
const (
	SizeSmall  = "small"
	SizeMedium = "medium"
	SizeBig    = "big"
	SizeEpic   = "epic"
)

var (
	// ErrNotFound is returned for an unknown task id.
	ErrNotFound = errors.New("task not found")
	// ErrInvalidPrompt is returned for an empty prompt.
	ErrInvalidPrompt = errors.New("prompt must be non-empty")
	// ErrProjectRequired is returned when a task is created or edited without
	// a project. Every task belongs to one.
	ErrProjectRequired = errors.New("project is required")
	// ErrNotDraft is returned when editing a non-draft task.
	ErrNotDraft = errors.New("task is not a draft")
	// ErrWrongState is returned when a transition does not apply.
	ErrWrongState = errors.New("task is not in a valid state for this action")
	// ErrRunning is returned when a close-out targets a running or input task,
	// which must be stopped or settled first.
	ErrRunning = errors.New("task is running; stop it or let it settle first")
	// ErrTagMissing is returned when a tag id does not resolve.
	ErrTagMissing = errors.New("tag not found")
	// ErrNotMember is returned when an agent is not a member of a project.
	ErrNotMember = errors.New("agent is not a member of the project")
	// ErrProjectHasNoAgents is returned when a task would be queued into a
	// project with no member agents, where the scheduler could never place it.
	ErrProjectHasNoAgents = errors.New(
		"project has no member agents; add one or save the task as a draft",
	)
	// ErrSelfReference is returned when a task is made to reference itself.
	ErrSelfReference = errors.New("a task cannot reference itself")
	// ErrReferenceCycle is returned when a blocking reference would make the
	// graph cyclic.
	ErrReferenceCycle = errors.New("blocking reference would create a cycle")
	// ErrReferenceNotAllowed is returned when a cross-project reference does
	// not follow the project hierarchy: a task may reference tasks in its own
	// project, a descendant project, or a sibling project.
	ErrReferenceNotAllowed = errors.New(
		"a task may only reference tasks in its own, a descendant, or a sibling project",
	)
	// ErrMergeTooFew is returned when a merge names fewer than two tasks.
	ErrMergeTooFew = errors.New("select at least two tasks to merge")
	// ErrMergeProjectMismatch is returned when merged tasks span projects.
	ErrMergeProjectMismatch = errors.New("tasks to merge must share one project")
	// ErrMergeAssigneeConflict is returned when merged tasks have different
	// assignees.
	ErrMergeAssigneeConflict = errors.New("tasks to merge have conflicting assignees")
	// ErrMergeState is returned when a merged task is in a state that cannot be
	// merged (running, input, settled, completed, or cancelled).
	ErrMergeState = errors.New("only draft, backlog, queued, or failed tasks can be merged")
	// ErrTeamsRequired is returned when a new task names no owning team and the
	// creator has none attached to the project either.
	ErrTeamsRequired = errors.New("at least one owning team is required")
	// ErrTeamNotInProject is returned when an owning team is not attached to
	// the task's project.
	ErrTeamNotInProject = errors.New("owning team is not attached to the project")
	// ErrLeadNotAllowed is returned when a lead is neither a project member nor
	// a member of one of the task's owning teams.
	ErrLeadNotAllowed = errors.New(
		"a lead must be a project member or belong to one of the task's owning teams",
	)
	// ErrCannotPrompt is returned when the caller is not allowed to prompt the
	// task.
	ErrCannotPrompt = errors.New("not allowed to prompt this task")
	// ErrActorRequired is returned when an action needs an authenticated actor
	// but the context carries none.
	ErrActorRequired = errors.New("an authenticated actor is required")
)

// Tag is one label attached to a task. Archived tags still render on
// settled tasks.
type Tag struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	Color       string
	Archived    bool
}

// Link is a labeled URL an agent attached to its task, e.g. a PR or an issue.
type Link struct {
	URL   string
	Label string
}

// ReferenceKind is the meaning of a reference edge. `blocking` gates queuing;
// `informational` is a plain relation; `supersedes` cancels the referenced
// task when it is in draft, backlog, queued, or failed.
type ReferenceKind string

const (
	ReferenceBlocking      ReferenceKind = "blocking"
	ReferenceInformational ReferenceKind = "informational"
	ReferenceSupersedes    ReferenceKind = "supersedes"
)

// Reference is a lightweight edge to another task, enough for a list to render
// the referenced task's title, state, and the edge's kind.
type Reference struct {
	ID     string
	Name   string
	Prompt string
	State  State
	Kind   ReferenceKind
}

// Blocking reports whether the edge gates queuing.
func (r Reference) Blocking() bool { return r.Kind == ReferenceBlocking }

// ReferenceInput identifies a task a task references and the edge's kind.
type ReferenceInput struct {
	TaskID string
	Kind   ReferenceKind
}

// Task is the public view of a work item.
type Task struct {
	// RowID is the task's stable, table-unique integer key. Unlike ID it never
	// changes, so it is what the Cleanup tab orders by.
	RowID           int64
	ID              string
	Seq             int64
	Prompt          string
	Name            string
	ProjectID       string
	AssigneeActorID ids.ActorID
	// CreatorActorID is the actor that created the task. Zero for tasks that
	// predate phase 6 or were created without an authenticated actor.
	CreatorActorID ids.ActorID
	// TeamIDs are the teams that own the task. A member of any owning team may
	// prompt it. Empty on a legacy task, which falls back to the project rule.
	TeamIDs []int64
	// LeadActorIDs are the actors accountable for the task, each a member of an
	// owning team. They are shown on the task and notified of its updates.
	LeadActorIDs []ids.ActorID
	// Subscribed reports whether the caller is subscribed to the task's
	// updates. Only Get sets it: a list does not know the caller.
	Subscribed     bool
	AssigneePinned bool
	Priority       int64
	Size           string
	State          State
	SessionID      string
	// SessionAgentID is the agent that owns SessionID. It survives a later
	// reassignment, so the scheduler can tell a same-agent resume from a
	// cross-agent continuation.
	SessionAgentID ids.AgentID
	Error          string
	ModelProvider  string
	ModelID        string
	ThinkingLevel  string
	Interactive    bool
	Workspace      string
	AutoTitle      bool
	PromptIDs      prompts.VersionIDs
	// TeamPromptIDs pins each owning team's general and per-project prompt
	// versions, captured on the first start.
	TeamPromptIDs []prompts.TeamVersionIDs
	Tags          []Tag
	Links         []Link
	References    []Reference
	// ReferencedBy is the incoming side: the tasks that reference this one.
	// Only Get enriches it, so listing tasks stays cheap.
	ReferencedBy []Reference
	// Blockers is the number of incomplete blocking references holding this
	// task's queuing back. Only meaningful for a queued task.
	Blockers int
	// PendingPrompt is a follow-up queued on a resting task that the scheduler
	// delivers when the task is next started (empty otherwise).
	PendingPrompt string
	// PendingPromptActorID is the actor that sent PendingPrompt, so the
	// scheduler can attribute the delivered prompt.
	PendingPromptActorID ids.ActorID
	QueuedAt             time.Time
	StartedAt            time.Time
	SettledAt            time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	// StateUpdatedAt is when the effective state last changed; unlike UpdatedAt
	// a plain edit does not move it.
	StateUpdatedAt time.Time
	// SessionDeletedAt is when the task's agent-side Pi session and workspace
	// were cleaned up. Zero means the session still exists. The agency keeps
	// the conversation, so a follow-up starts a fresh Pi session that replays
	// it; a new session clears this mark.
	SessionDeletedAt time.Time
	// Archived hides the task from the active lists and releases its agent
	// slot. Archiving also cleans up the agent-side session; an archived task
	// whose session still exists (the agent was offline) is highlighted so the
	// cleanup can be retried.
	Archived   bool
	ArchivedAt time.Time
}

// Filter narrows List. Every field is optional.
type Filter struct {
	ProjectID string
	AgentID   ids.AgentID
	State     State
	TagID     string
	Search    string
	// Size narrows to one manually set size; empty applies no filter.
	Size string
	// Archived selects the archive (true) instead of the active list (the
	// default false).
	Archived bool
}

// CreateParams carries a new task's fields. Priority is a stored value
// (PriorityLow/Normal/High/Critical); State selects the initial state and must
// be StateDraft, StateBacklog, or StateQueued (empty queues it).
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
	Interactive     bool
	Workspace       string
	AutoTitle       bool
	State           State
	// TeamIDs are the teams that own the task. At least one is required, and
	// each must be attached to the project. When empty, the creator's teams on
	// the project are used.
	TeamIDs []int64
	// LeadActorIDs are the task's initial leads. Each must be a project
	// member or a member of one of TeamIDs.
	LeadActorIDs []ids.ActorID
}

// UpdateParams carries a draft task's new fields.
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
	Interactive     bool
	Workspace       string
	AutoTitle       bool
}

// AttributeSet is a partial update of a task's mutable attributes, shared by
// the single- and bulk-update paths. A nil field leaves that attribute
// unchanged; a non-nil AssigneeActorID (even an empty string) sets the pin,
// so an empty value clears it and lets the scheduler pick an agent.
//
// Nothing here requires a draft: priority, size, and tag changes are safe in
// any state (an added tag is validated against the task's project). A live run
// (running or input) cannot be reassigned, and [Service.applyAttributes]
// rejects it so a task never reports an assignee other than the one running it.
type AttributeSet struct {
	Priority        *int64
	Size            *string
	AssigneeActorID *ids.ActorID
	// Prompt, when non-nil, replaces a task's prompt. It is only editable
	// before the task is queued (draft or backlog): a queued or started task's
	// prompt is pinned.
	Prompt *string
	// PendingPrompt, when non-nil, replaces a draft task's stashed follow-up.
	// It is draft-only: the pending message is what a re-queue delivers, so it
	// cannot change while a task is queued or running.
	PendingPrompt *string
	// AddTagIDs and RemoveTagIDs attach and detach project tags. Removing a tag
	// a task does not carry, or attaching one it already has, is a no-op.
	AddTagIDs    []string
	RemoveTagIDs []string
}

// Empty reports whether no attribute is set.
func (a AttributeSet) Empty() bool {
	return a.Priority == nil && a.Size == nil && a.AssigneeActorID == nil &&
		a.Prompt == nil &&
		a.PendingPrompt == nil &&
		len(a.AddTagIDs) == 0 && len(a.RemoveTagIDs) == 0
}

// changesTags reports whether attrs touches the task's tag set.
func (a AttributeSet) changesTags() bool {
	return len(a.AddTagIDs) > 0 || len(a.RemoveTagIDs) > 0
}

// AttributeResult is one task's outcome in a bulk attribute update. Exactly
// one of Task or Error is meaningful: Task is set on success, Error on failure.
type AttributeResult struct {
	ID    string
	Task  Task
	Error error
}

// Transition is a task landing in a new lifecycle state, announced to the
// inbox through [Service.OnTransition].
type Transition struct {
	TaskID string
	State  State
}

// Service reads and writes tasks.
type Service struct {
	store    store.Store
	projects *projects.Service
	// OnTransition, when set, is called after a task lands in a new lifecycle
	// state (started, settled, completed, cancelled). It runs synchronously, so
	// implementations must stay cheap; a nil hook disables the callback. The
	// input and failed states are deliberately not announced here: the events
	// consumer owns those richer notifications (dialog title, error message).
	OnTransition func(ctx context.Context, t Transition)
}

// New builds the service.
func New(st store.Store, projectsSvc *projects.Service) *Service {
	return &Service{store: st, projects: projectsSvc}
}

// transitioned invokes the lifecycle hook, if any.
func (s *Service) transitioned(ctx context.Context, id string, state State) {
	if s.OnTransition != nil {
		s.OnTransition(ctx, Transition{TaskID: id, State: state})
	}
}

// Create inserts a task. It always writes the row as a draft first, attaches
// its dependencies, then queues it (as queued or blocked) unless `Draft` is
// set, so a new task can never be drained before its dependencies are on
// record. A pinned assignee in a project must be a member of it.
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
	// Resolve the references up front so a bad id never leaves a half-created
	// task behind.
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
		Interactive:     boolInt(p.Interactive),
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
	// The creator follows their own task by default, so it does not need a
	// separate subscription.
	if creator := actorIDFrom(ctx); creator != 0 {
		if err := s.store.AddTaskSubscription(ctx, store.AddTaskSubscriptionParams{
			TaskID:    row.ID,
			ActorID:   int64(creator),
			CreatedAt: now,
		}); err != nil {
			return Task{}, err
		}
	}
	// Creation is one log entry; the individual tag/reference/queue writes are
	// not logged separately, so the log does not replay the create form.
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
		// Draft is never queued directly: promotion to backlog is the ladder's
		// only path to the queue.
		if _, err := s.Promote(internal, row.ID); err != nil {
			return Task{}, err
		}
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

// normalizeCreateState maps a create request's state onto an initial lifecycle
// state. An empty state queues the task immediately (the pre-backlog default).
func normalizeCreateState(state State) State {
	switch state {
	case StateDraft, StateBacklog, StateQueued:
		return state
	default:
		return StateQueued
	}
}

// resolveReferences trims, dedupes, and validates reference inputs for a new
// task.
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

// checkReferenceProject enforces the hierarchy's reference shape: same
// project, a descendant, or a sibling. Orphan tasks (no project) are exempt so
// legacy rows keep working.
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

// normalizeReferenceKind maps an unspecified edge kind onto informational.
func normalizeReferenceKind(kind ReferenceKind) ReferenceKind {
	switch kind {
	case ReferenceBlocking, ReferenceSupersedes:
		return kind
	default:
		return ReferenceInformational
	}
}

// Get resolves one task with its tags.
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
	}
	return t, nil
}

// GetByProjectSeq resolves a task by its stable reference: the project it
// belongs to and its per-project sequence. Unlike the rendered task id, the
// pair survives a project prefix rename, so the agent-tool mint route can
// resolve a task from the reference an agent persisted long ago.
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

// snapshot reads a task's own row only (no tags, links, or dependencies). It
// is enough to log a state or single-field change without the child-table
// queries, and keeps before/after comparisons consistent.
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

// List returns tasks matching the filter, newest-first, with their tags.
func (s *Service) List(ctx context.Context, f Filter) ([]Task, error) {
	rows, err := s.store.ListTasks(ctx, store.ListTasksParams{
		ProjectID: nullString(f.ProjectID),
		AgentID:   nullID(f.AgentID),
		State:     nullString(string(f.State)),
		Archived:  boolInt(f.Archived),
		TagID:     nullString(f.TagID),
		Search:    nullString(f.Search),
		Size:      nullString(f.Size),
	})
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
	// Float the best matches above the rest: an exact task id (`PROJECT-61`),
	// then the task-number ranking for a numeric search (exact before partial).
	prioritizeSearchMatches(out, f.Search)
	return out, nil
}

// prioritizeSearchMatches stable-sorts a list so a task whose human-readable
// id equals the search string sorts first, and, for a purely numeric search,
// tasks whose number matches sort next (exact before partial). Any other
// query keeps the store's newest-first order.
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

// isDigits reports whether s is a non-empty run of ASCII digits, the shape
// that enables task-number search.
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

// ListQueued returns the scheduler's queue: priority DESC, then oldest first.
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

// ListRunning returns every in-flight task (any agent), which the scheduler's
// reconcile uses to materialize settlement.
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

// Update edits a draft task. Any other state is rejected.
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

	// A task that moves to another project takes that project's prefix and a
	// fresh number, so its id keeps the `<prefix>-<number>` shape.
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
		Interactive:     boolInt(p.Interactive),
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

// Queue moves a backlog task to queued. A queued task with incomplete
// blocking references stays queued but is reported as blocked; the scheduler
// skips it until every blocking reference is completed.
func (s *Service) Queue(ctx context.Context, id string) (Task, error) {
	cur, err := s.requireState(ctx, id, StateBacklog)
	if err != nil {
		return Task{}, err
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

// Promote moves a draft task to backlog: very likely to be run, but not queued
// yet. Draft is never queued directly.
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

// Demote moves a backlog task back to draft.
func (s *Service) Demote(ctx context.Context, id string) (Task, error) {
	cur, err := s.requireState(ctx, id, StateBacklog)
	if err != nil {
		return Task{}, err
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

// Dequeue moves a queued task back to backlog without starting it. It is the
// only way a queued task leaves the queue besides being claimed.
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

// References returns the tasks a task references (outgoing edges), newest
// first.
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

// AddReference upserts a reference from taskID to referencedID. A blocking
// edge gates queuing: a self-reference and a cycle are rejected; a queued task
// that gains one stays queued but is reported as blocked. A supersedes edge
// cancels the referenced task when it is in draft, backlog, queued, or failed.
// References are editable in any state.
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

// cancelSuperseded cancels a task a supersedes edge now points at, but only
// when it is in draft, backlog, queued, or failed. Any other state is left
// untouched.
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

// RemoveReference drops an edge from a task in any state.
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

// SetReferences replaces a task's reference set in any state. It resolves and
// validates the whole set first, so a bad id leaves the current set untouched;
// the individual add/remove calls then apply the diff.
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

// hasBlockingReferencePath reports whether `from` transitively blocking-
// references `to`, following blocking edges. Adding `to -> from` would create a
// cycle when this is true.
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

// Archive hides a completed or cancelled task; its lifecycle state is
// preserved. The caller cleans up the agent-side session (which may be
// deferred when the agent is offline). Only a closed-out task can be archived.
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

// Unarchive restores an archived task to the active lists.
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

// mergePromptSeparator divides consecutive mergee prompts in the merged task,
// so the origin of each section stays legible.
const mergePromptSeparator = "\n\n---\n\n"

// Merge creates a new draft task from the given tasks: their prompts are
// concatenated in order, separated by a `---` rule, their tags unioned, their
// size the largest, and their attachments, links, and references folded in.
// The merged task supersedes every mergee, which the supersedes edge cancels.
// Every mergee must share one project and one assignee, and be in draft,
// backlog, queued, or failed.
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
	// Ownership is the union of the mergees', so the merged task keeps its
	// owning teams and leads.
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
		// Leave the title empty: the merged draft auto-titles once it runs,
		// rather than freezing the mergee ids into a permanent name.
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

	// Fold each mergee's attachments, links, and references into the merged
	// task. Attachments share the stored blob; links dedupe by URL and
	// references by target and kind. References that point at a mergee are
	// skipped, since the merged task supersedes it instead.
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
	// The merged task supersedes every mergee; the edge cancels each one that
	// is in draft, backlog, queued, or failed.
	for _, t := range mergees {
		if _, err := s.AddReference(ctx, merged.ID, t.ID, ReferenceSupersedes); err != nil {
			return Task{}, err
		}
	}
	return s.Get(ctx, merged.ID)
}

// sizeRankOf maps a size to its ascending rank.
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

// sizeName maps a size rank back to its stored value.
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

// Settle marks a running or input task settled: its Pi run finished and it is
// idle, awaiting a follow-up or a close-out. It reports false when the task
// already left those states (a concurrent update won).
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

// MarkInput records that a running task is blocked on an extension dialog.
// It reports false when the task was not running.
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

// ClearInput records that a task blocked on a dialog is running again.
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

// Resume flips a settled or failed task back to running after a follow-up
// prompt was accepted. It reports false when the task was not in either state.
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

// Complete closes out a non-running task as done.
func (s *Service) Complete(ctx context.Context, id string) (Task, error) {
	return s.closeOut(ctx, id, ResolutionCompleted)
}

// Cancel closes out a non-running task as abandoned.
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

// RequeueWithPrompt stores a follow-up to a resting (settled/failed/completed/
// cancelled) task as its pending prompt and queues it. The scheduler delivers
// the prompt when the task is next started, respecting the agent's slot caps.
// A queued task with incomplete blocking references stays queued but blocked.
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
	now := time.Now().UnixMilli()
	n, err := s.store.TransitionTask(ctx, store.TransitionTaskParams{
		Event:                store.TaskRequeueWithPrompt,
		PendingPrompt:        prompt,
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

// ClearPendingPrompt drops a pending prompt once the scheduler has delivered
// it. Delivery already happened, so a failed clear is best-effort.
func (s *Service) ClearPendingPrompt(ctx context.Context, id string) error {
	_, err := s.store.ClearTaskPendingPrompt(ctx, store.ClearTaskPendingPromptParams{
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	return err
}

// Fail marks a running or input task failed. The slot stays held so a retry or
// follow-up can resume the warm Pi session; there is no auto-retry.
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

// Reopen returns a settled, failed, completed, or cancelled task to backlog
// (or straight to queued when `queue` is set). The Pi session and prompt pins
// survive, so the scheduler resumes a warm session when it next starts.
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

// Rename sets a task's display name.
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

// SetPriority sets a task's scheduler priority. Unlike Update it does not
// require a draft: priority only orders the queue, so it is safe to change in
// any state (a running task keeps its slot and picks the value up on a
// re-queue).
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

// SetSize sets a task's manually estimated size. Like SetPriority it does not
// require a draft: size is informational, so it is safe to change in any
// state.
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

// UpdateAttributes applies a partial attribute update to one task. It shares
// its checks with [Service.UpdateManyAttributes].
func (s *Service) UpdateAttributes(
	ctx context.Context,
	id string,
	attrs AttributeSet,
) (Task, error) {
	cur, err := s.snapshot(ctx, id)
	if err != nil {
		return Task{}, err
	}
	// A snapshot omits child rows, but the change log diffs tags, links, and
	// references, so load the full task when the update touches a field that
	// returns one (a tag change or a prompt edit, which is logged with the
	// task's child rows).
	if attrs.changesTags() || attrs.Prompt != nil {
		if cur, err = s.Get(ctx, id); err != nil {
			return Task{}, err
		}
	}
	return s.applyAttributes(ctx, cur, attrs)
}

// UpdateManyAttributes applies the same attributes to each id, checking every
// task independently so one failure (a missing id, a live run, an assignee
// outside the project) does not block the rest. The returned slice follows
// the input order and always has one result per id.
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

// applyAttributes writes the non-nil attributes of attrs to one task. A live
// run owns its agent, so reassigning it is rejected; priority and size stay
// editable in any state.
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
		// A prompt is only editable before the task is queued; queued and
		// running tasks have pinned their prompt into the Pi session.
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
		// The pending follow-up is the next message a re-queue delivers, so it
		// is only editable while the task is a plain draft.
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
		// A tag must belong to the task's project; ValidateTags rejects an
		// unknown or cross-project id so the task is reported, not silently
		// mis-tagged. Removals accept any id (detaching a foreign tag is a
		// no-op) and run first so a tag in both lists ends up attached.
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

// Delete removes a task from any state. Its tags cascade.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return s.store.DeleteTask(ctx, id)
}

// Claim atomically flips a queued task to running and assigns it. It reports
// false when the task was already claimed. Prompt pins are only written on
// the first run (StartedAt zero), so a re-queue keeps them.
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
		// Team prompt pins are written on the first start only, alongside the
		// other scopes. A failure here leaves the claimed task without pins; the
		// next fresh session falls back to the latest team prompts.
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

// SetSession records the agent-side Pi session id a task started and the
// agent that owns it. session_agent_id is kept even after a reassignment so
// the scheduler can tell a same-agent resume from a cross-agent continuation.
// A new session also clears the cleanup mark, so the task has a live session
// again.
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
	return err
}

// MarkSessionDeleted records that a task's agent-side Pi session and workspace
// were cleaned up. It keeps the task's lifecycle state and is idempotent: a
// second call leaves the first timestamp in place.
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
		// Already marked (or the task vanished); snapshot reports the latter.
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

// SetError records an error string without changing the state.
func (s *Service) SetError(ctx context.Context, id, errMsg string) error {
	_, err := s.store.SetTaskError(ctx, store.SetTaskErrorParams{
		Error:     errMsg,
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	})
	return err
}

// RunningCounts aggregates an agent's running tasks for the scheduler's caps.
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

// IsMember reports whether an agent belongs to a task's project. Orphan tasks
// have no membership requirement.
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
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if cur.State != want {
		return Task{}, ErrWrongState
	}
	return cur, nil
}

// validateScope resolves a project (by id or slug) and, when an assignee is
// pinned, enforces membership. It returns the resolved project.
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

// agentIDByActor resolves an assignee actor to its agent row, if any.
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

// TaskID formats a task id from a project's prefix and sequence number.
func TaskID(prefix string, number int64) string {
	return prefix + "-" + strconv.FormatInt(number, 10)
}

// requireExecutor rejects a task that would be queued into a project with no
// member agents, where the scheduler could never place it. A pinned assignee
// or an orphan task is always fine.
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

// ValidateTags rejects unknown tags and tags that belong to another project.
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

// SetTags replaces a task's tags. The caller is responsible for validating
// that the tags belong to the task's project.
//
// ponytail: delete+insert without a transaction; the agency is a single writer
// and a tag set is tiny. Wrap in a store transaction if concurrent edits land.
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

// SetLinksBySession replaces the links of the task that owns an agent's Pi
// session. An unknown session is a no-op, so a link that races completion is
// not an error. Only absolute http(s) URLs are kept, deduplicated by URL (the
// last label wins) and order-preserving.
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

// SetLinks replaces a task's labeled links. Only absolute http(s) URLs are
// kept, deduplicated by URL (the last label wins) and order-preserving.
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

// allReferences maps each task id to its lightweight outgoing references, so a
// list renders every row's references with one query.
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

// allReferencedBy maps each task id to its lightweight incoming references, so
// a task detail renders "referenced by" with one query.
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

// allTaskTeams maps each task id to its owning team ids, so a list fills them
// in one query.
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

// allTaskLeads maps each task id to its lead actor ids, so a list fills them in
// one query.
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

// actorIDsFromInt64s widens stored actor ids to the service's type.
func actorIDsFromInt64s(in []int64) []ids.ActorID {
	out := make([]ids.ActorID, len(in))
	for i, id := range in {
		out[i] = ids.ActorID(id)
	}
	return out
}

// dedupeActorIDs returns the ids in order with zero and duplicates removed.
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

// dedupeInt64 returns the ids in order with duplicates removed.
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

// validateTaskTeams requires at least one owning team and checks every team is
// attached to the project. It rejects before anything is written, so a bad id
// never half-clears the set.
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

// resolveTaskTeams returns the owning teams a new task should get: the explicit
// list when present, else the creator's teams attached to the project, else the
// project's teams. A task in a project that has teams is therefore never
// teamless; a project with no teams yields a legacy teamless task.
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

// setTaskTeams replaces a task's owning teams without logging. Create calls it
// before the task is announced; SetTaskTeams wraps it with a log entry.
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

// SetTaskTeams replaces a task's owning teams and prunes the leads that are no
// longer owning-team members or project members. A task keeps at least one
// owning team.
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

// setTaskLeads replaces a task's leads without logging. It validates that every
// lead is a project member or a member of an owning team.
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

// SetTaskLeads replaces a task's leads.
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

// validateTaskLeads rejects a lead who is neither a project member nor a member
// of an owning team. The owning-team check runs first: a viewer-role team may
// still own a task, and its members may lead it.
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

// taskTeamMembers returns every actor in a task's owning teams.
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

// SetSubscribed subscribes or unsubscribes actorID to a task's updates. The
// transport decides whether the actor may view the task; this only records the
// choice.
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
	// The caller subscribed explicitly, so its own flag is authoritative even
	// when Get could not read the actor from the context.
	t.Subscribed = subscribed
	return t, nil
}

// CanCreateInProject reports whether actorID may create a task in a project:
// any explicit or inherited member/owner role. A viewer (including the viewer
// role inherited from a parent project) cannot prompt.
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

// CanPrompt reports whether actorID may send a prompt to the task. An
// administrator and the creator may always prompt; otherwise the actor must be
// in one of the task's owning teams (or be one of its leads). A legacy task
// with no owning teams falls back to the project member rule, so it keeps its
// pre-ownership behaviour.
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

// Recipients returns the actors a task's inbox entries target: its leads plus
// its subscribers.
func (s *Service) Recipients(ctx context.Context, id string) ([]ids.ActorID, error) {
	rows, err := s.store.ListTaskNotificationRecipients(ctx, id)
	if err != nil {
		return nil, err
	}
	return actorIDsFromInt64s(rows), nil
}

// HasTeams reports whether a task has any owning team. A teamless task is a
// legacy task and falls back to the project prompt rule.
func (s *Service) HasTeams(ctx context.Context, id string) (bool, error) {
	teams, err := s.store.ListTaskTeamsByTask(ctx, id)
	if err != nil {
		return false, err
	}
	return len(teams) > 0, nil
}

// Sender is the actor a prompt is attributed to.
type Sender struct {
	ActorID  ids.ActorID
	Username string
	Name     string
	Kind     string
}

// Sender resolves the actor info a prompt envelope carries. An unknown or
// zero actor becomes a system placeholder so a legacy task can still prompt.
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

// countBlockers counts incomplete blocking references: a queued task is
// reported as blocked while this is greater than zero.
func countBlockers(refs []Reference) int {
	n := 0
	for _, ref := range refs {
		if ref.Kind == ReferenceBlocking && ref.State != StateCompleted {
			n++
		}
	}
	return n
}

// setTeamPromptPins replaces the team prompt versions a task pinned on its
// first start. It is idempotent and runs after the claim transition.
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

// normalizeSize maps the protobuf's UNSPECIFIED (which arrives as an empty
// string) onto the stored default. Every persisted task has a size.
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

// TaskIDBySession resolves the agency task id that owns an agent's Pi session.
// It returns ErrNotFound when no task matches (for example a session created
// before the agency tracked the task).
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

// FailBySession marks the running task that owns an agent's Pi session as
// failed. It is a no-op when no running task owns that session (for example a
// task created before the agency tracked it).
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

// Enrich fills a task's name and prompt from its agent's record. The update is
// a no-op unless the agency never recorded them (legacy rows backfilled from
// session_context), so it never overwrites a task's own prompt or name.
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
