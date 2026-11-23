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

package knowledge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"github.com/google/uuid"
)

const (
	SourceTaskPrompt = "task_prompt"

	SourceFollowUp = "follow_up"

	SourceQuestionnaire = "questionnaire"

	SourceSystemPrompt = "system_prompt"

	LinkRelated = "related"

	LinkSupersedes = "supersedes"

	maxContentRunes = 20000
	maxTitleRunes   = 120
)

type BitSourceInput struct {
	Kind       string
	TaskID     string
	MessageID  string
	RequestID  string
	QuestionID string
	PromptID   int64
}

type BitInput struct {
	ProjectID     *string
	Title         string
	Content       string
	Tags          []string
	Sources       []BitSourceInput
	RelatedIDs    []string
	SupersedesIDs []string

	OriginTaskID string
	Summary      string
}

type BitPatch struct {
	Title         *string
	Content       *string
	Tags          *[]string
	Sources       *[]BitSourceInput
	RelatedIDs    *[]string
	SupersedesIDs *[]string
	BaseRevision  int64
	Summary       string
}

type BitView struct {
	Bit         store.KnowledgeBitRow
	Tags        []string
	Sources     []store.KnowledgeBitSourceRow
	Links       []store.KnowledgeBitLinkRow
	Backlinks   []store.KnowledgeBitLinkRow
	Revisions   []store.KnowledgeBitRevisionRow
	ProjectSlug string
	CanEdit     bool
	CanReview   bool
	CanDelete   bool
}

type BitFilter struct {
	Scope         string
	ProjectID     string
	Tag           string
	Status        string
	Query         string
	TaskID        string
	AuthorActorID string
	Limit         int
	Offset        int
}

type BitTag struct {
	Tag   string
	Count int64
}

func (s *Service) Add(ctx context.Context, actor Actor, in BitInput) (BitView, error) {
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return BitView{}, fmt.Errorf("%w: content is empty", ErrInvalidInput)
	}
	if len([]rune(content)) > maxContentRunes {
		return BitView{}, fmt.Errorf("%w: content is too long", ErrInvalidInput)
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = deriveTitle(content)
	}
	if len([]rune(title)) > maxTitleRunes {
		title = string([]rune(title)[:maxTitleRunes])
	}
	_, tags, err := normalizeTags(in.Tags)
	if err != nil {
		return BitView{}, err
	}
	if len(tags) > 32 {
		return BitView{}, fmt.Errorf("%w: at most 32 tags", ErrInvalidInput)
	}

	originTaskID := strings.TrimSpace(in.OriginTaskID)
	var originTask tasks.Task
	if originTaskID != "" {
		originTask, err = s.requireTask(ctx, actor, originTaskID, actor.Kind == accounts.KindAgent)
		if err != nil {
			return BitView{}, err
		}
	}
	if actor.Kind == accounts.KindAgent && originTaskID == "" {
		return BitView{}, ErrOriginTaskRequired
	}

	projectID := ""
	switch {
	case in.ProjectID != nil:
		projectID = strings.TrimSpace(*in.ProjectID)
	case originTask.ID != "":
		projectID = originTask.ProjectID
	}
	ref := ScopeRef{Scope: ScopeOrg}
	if projectID != "" {
		ref = ScopeRef{Scope: ScopeProject, ScopeID: projectID}
	}
	if err := s.requireWrite(ctx, actor, ref); err != nil {
		return BitView{}, err
	}

	sources, err := s.buildSources(ctx, actor, projectID, originTaskID, in.Sources)
	if err != nil {
		return BitView{}, err
	}

	now := s.now().UTC()
	bit := store.KnowledgeBitRow{
		ID:               uuid.NewString(),
		Title:            title,
		Content:          content,
		Status:           StatusPending,
		OriginTaskID:     originTaskID,
		OriginTaskRowID:  sql.NullInt64{},
		Revision:         1,
		CreatedByActorID: nullActor(actor.ID),
		CreatedBy:        actor.Ref(),
		CreatedAt:        now.UnixMilli(),
		UpdatedAt:        now.UnixMilli(),
	}
	if projectID != "" {
		bit.ProjectID = sql.NullString{String: projectID, Valid: true}
	}
	if originTask.ID != "" {
		bit.OriginTaskID = originTask.ID
		bit.OriginTaskRowID = sql.NullInt64{Int64: originTask.RowID, Valid: true}
	}
	if actor.IsHuman() {
		bit.Status = StatusApproved
		bit.ReviewedByActorID = nullActor(actor.ID)
		bit.ReviewedBy = actor.Ref()
		bit.ReviewedAt = sql.NullInt64{Int64: now.UnixMilli(), Valid: true}
	}

	links, supersedes, err := s.linkTargets(ctx, actor, bit.ID, in.RelatedIDs, in.SupersedesIDs)
	if err != nil {
		return BitView{}, err
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		summary = "add knowledge"
	}
	rev := store.KnowledgeBitRevisionRow{
		Title:     bit.Title,
		Content:   bit.Content,
		TagsJSON:  mustTagsJSON(tags),
		Status:    bit.Status,
		ActorID:   nullActor(actor.ID),
		ActorRef:  actor.Ref(),
		Summary:   summary,
		CreatedAt: now.UnixMilli(),
	}
	created, err := s.store.CreateKnowledgeBit(
		ctx, bit, tags, sources, links, rev,
	)
	if err != nil {
		return BitView{}, err
	}

	if bit.Status == StatusApproved {
		if err := s.retireTargets(ctx, created.ID, supersedes); err != nil {
			return BitView{}, err
		}
	}
	s.signal()
	return s.view(ctx, actor, created.ID)
}

func (s *Service) Get(ctx context.Context, actor Actor, id string) (BitView, error) {
	bit, err := s.loadBit(ctx, id)
	if err != nil {
		return BitView{}, err
	}
	if err := s.requireRead(ctx, actor, bitScope(bit)); err != nil {
		return BitView{}, err
	}
	return s.view(ctx, actor, bit.ID)
}

func (s *Service) List(ctx context.Context, actor Actor, filter BitFilter) ([]BitView, error) {
	storeFilter, err := s.bitFilter(ctx, actor, filter)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListKnowledgeBits(ctx, storeFilter)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	tags, err := s.store.KnowledgeBitTags(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]BitView, len(rows))
	for i, row := range rows {
		out[i] = s.lightView(ctx, actor, row, tags[row.ID])
	}
	return out, nil
}

func (s *Service) Search(ctx context.Context, actor Actor, filter BitFilter) ([]BitView, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	return s.List(ctx, actor, filter)
}

func (s *Service) Tags(ctx context.Context, actor Actor, filter BitFilter) ([]BitTag, error) {
	storeFilter, err := s.bitFilter(ctx, actor, filter)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListKnowledgeTagCounts(ctx, storeFilter)
	if err != nil {
		return nil, err
	}
	out := make([]BitTag, len(rows))
	for i, row := range rows {
		out[i] = BitTag{Tag: row.Tag, Count: row.Count}
	}
	return out, nil
}

func (s *Service) Update(
	ctx context.Context,
	actor Actor,
	id string,
	patch BitPatch,
) (BitView, error) {
	bit, err := s.loadBit(ctx, id)
	if err != nil {
		return BitView{}, err
	}
	if !s.canEdit(ctx, actor, bit) {
		return BitView{}, ErrForbidden
	}
	if bit.Status == StatusSuperseded {
		return BitView{}, fmt.Errorf("%w: a superseded bit cannot be edited", ErrConflict)
	}
	now := s.now().UTC()

	next := bit
	contentChanged := false
	if patch.Title != nil {
		next.Title = strings.TrimSpace(*patch.Title)
		if next.Title == "" {
			next.Title = deriveTitle(next.Content)
		}
		if len([]rune(next.Title)) > maxTitleRunes {
			next.Title = string([]rune(next.Title)[:maxTitleRunes])
		}
		contentChanged = true
	}
	if patch.Content != nil {
		content := strings.TrimSpace(*patch.Content)
		if content == "" {
			return BitView{}, fmt.Errorf("%w: content is empty", ErrInvalidInput)
		}
		if len([]rune(content)) > maxContentRunes {
			return BitView{}, fmt.Errorf("%w: content is too long", ErrInvalidInput)
		}
		next.Content = content
		contentChanged = true
	}
	tags := mustTags(ctx, s.store, bit.ID)
	if patch.Tags != nil {
		_, normalized, err := normalizeTags(*patch.Tags)
		if err != nil {
			return BitView{}, err
		}
		if len(normalized) > 32 {
			return BitView{}, fmt.Errorf("%w: at most 32 tags", ErrInvalidInput)
		}
		tags = normalized
		contentChanged = true
	}
	sources := mustSources(ctx, s.store, bit.ID)
	if patch.Sources != nil {
		projectID := ""
		if bit.ProjectID.Valid {
			projectID = bit.ProjectID.String
		}
		sources, err = s.buildSources(ctx, actor, projectID, bit.OriginTaskID, *patch.Sources)
		if err != nil {
			return BitView{}, err
		}
		contentChanged = true
	}
	links := existingLinks(ctx, s.store, bit.ID)
	oldSupersedes := supersedingIDs(ctx, s.store, bit.ID)
	var supersedes []string
	if patch.RelatedIDs != nil || patch.SupersedesIDs != nil {
		related, supersedeInputs := splitLinks(ctx, s.store, bit.ID)
		if patch.RelatedIDs != nil {
			related = *patch.RelatedIDs
		}
		if patch.SupersedesIDs != nil {
			supersedeInputs = *patch.SupersedesIDs
		}
		links, supersedes, err = s.linkTargets(ctx, actor, bit.ID, related, supersedeInputs)
		if err != nil {
			return BitView{}, err
		}
	}

	if !actor.IsHuman() && bit.Status == StatusApproved &&
		!sameIDs(oldSupersedes, supersedes) {
		contentChanged = true
	}
	if contentChanged {
		next.Status = StatusPending
		next.ReviewedByActorID = sql.NullInt64{}
		next.ReviewedBy = ""
		next.ReviewedAt = sql.NullInt64{}
		next.ReviewNote = ""
	}
	next.UpdatedAt = now.UnixMilli()
	summary := strings.TrimSpace(patch.Summary)
	if summary == "" {
		summary = "update knowledge"
	}
	rev := store.KnowledgeBitRevisionRow{
		Title:     next.Title,
		Content:   next.Content,
		TagsJSON:  mustTagsJSON(tags),
		Status:    next.Status,
		ActorID:   nullActor(actor.ID),
		ActorRef:  actor.Ref(),
		Summary:   summary,
		CreatedAt: now.UnixMilli(),
	}
	updated, err := s.store.UpdateKnowledgeBit(
		ctx, bit.ID, patch.BaseRevision, next, tags, sources, links, rev,
	)
	if errors.Is(err, store.ErrKnowledgeConflict) {
		return BitView{}, ErrConflict
	}
	if err != nil {
		return BitView{}, err
	}
	if updated.Status == StatusApproved {
		if err := s.retireTargets(ctx, bit.ID, supersedes); err != nil {
			return BitView{}, err
		}
	}
	s.signal()
	return s.view(ctx, actor, updated.ID)
}

func (s *Service) Approve(
	ctx context.Context,
	actor Actor,
	id, note string,
) (BitView, error) {
	bit, err := s.loadBit(ctx, id)
	if err != nil {
		return BitView{}, err
	}
	if err := s.requireReview(ctx, actor, bit); err != nil {
		return BitView{}, err
	}
	supersedes := supersedingIDs(ctx, s.store, bit.ID)
	now := s.now().UTC()
	updated, err := s.store.ReviewKnowledgeBit(
		ctx, bit.ID, StatusApproved, nullActor(actor.ID), actor.Ref(),
		strings.TrimSpace(note), now.UnixMilli(), supersedes,
	)
	if err != nil {
		return BitView{}, err
	}
	s.signal()
	return s.view(ctx, actor, updated.ID)
}

func (s *Service) Reject(
	ctx context.Context,
	actor Actor,
	id, note string,
) (BitView, error) {
	bit, err := s.loadBit(ctx, id)
	if err != nil {
		return BitView{}, err
	}
	if err := s.requireReview(ctx, actor, bit); err != nil {
		return BitView{}, err
	}
	now := s.now().UTC()
	updated, err := s.store.ReviewKnowledgeBit(
		ctx, bit.ID, StatusRejected, nullActor(actor.ID), actor.Ref(),
		strings.TrimSpace(note), now.UnixMilli(), nil,
	)
	if err != nil {
		return BitView{}, err
	}
	s.signal()
	return s.view(ctx, actor, updated.ID)
}

func (s *Service) Delete(ctx context.Context, actor Actor, id string) error {
	bit, err := s.loadBit(ctx, id)
	if err != nil {
		return err
	}
	if !s.canDelete(ctx, actor, bit) {
		return ErrForbidden
	}
	if err := s.store.SoftDeleteKnowledgeBit(ctx, bit.ID, s.now().UTC().UnixMilli()); err != nil {
		return err
	}
	s.signal()
	return nil
}

func (s *Service) Revisions(
	ctx context.Context,
	actor Actor,
	id string,
) ([]store.KnowledgeBitRevisionRow, error) {
	bit, err := s.loadBit(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireRead(ctx, actor, bitScope(bit)); err != nil {
		return nil, err
	}
	return s.store.ListKnowledgeBitRevisions(ctx, bit.ID)
}

func (s *Service) view(ctx context.Context, actor Actor, id string) (BitView, error) {
	bit, err := s.loadBit(ctx, id)
	if err != nil {
		return BitView{}, err
	}
	tags := mustTags(ctx, s.store, bit.ID)
	sources := mustSources(ctx, s.store, bit.ID)
	links := mustLinks(ctx, s.store, bit.ID, false)
	backlinks := mustLinks(ctx, s.store, bit.ID, true)
	revisions, err := s.store.ListKnowledgeBitRevisions(ctx, bit.ID)
	if err != nil {
		return BitView{}, err
	}
	view := s.lightView(ctx, actor, bit, tags)
	view.Sources = sources
	view.Links = links
	view.Backlinks = backlinks
	view.Revisions = revisions
	return view, nil
}

func (s *Service) lightView(
	ctx context.Context,
	actor Actor,
	bit store.KnowledgeBitRow,
	tags []string,
) BitView {
	view := BitView{Bit: bit, Tags: tags}
	if slug, err := s.ProjectSlug(ctx, projectOf(bit)); err == nil {
		view.ProjectSlug = slug
	}
	canWrite, _ := s.CanWrite(ctx, actor, bitScope(bit))
	view.CanEdit = canWrite && bit.Status != StatusSuperseded &&
		(actor.ID == ids.ActorID(bit.CreatedByActorID.Int64) || actor.IsHuman())
	view.CanReview = actor.IsHuman() && canWrite && bit.Status == StatusPending
	view.CanDelete = view.CanEdit
	return view
}

func (s *Service) buildSources(
	ctx context.Context,
	actor Actor,
	projectID, defaultTaskID string,
	inputs []BitSourceInput,
) ([]store.KnowledgeBitSourceRow, error) {
	out := make([]store.KnowledgeBitSourceRow, 0, len(inputs))
	seen := map[string]struct{}{}
	for _, in := range inputs {
		source, err := s.buildSource(ctx, actor, defaultTaskID, in)
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf(
			"%s|%s|%s|%s|%s|%d",
			source.Kind, source.TaskID, source.MessageID, source.RequestID,
			source.QuestionID, source.PromptID.Int64,
		)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		source.Position = int64(len(out))
		out = append(out, source)
	}
	return out, nil
}

func (s *Service) buildSource(
	ctx context.Context,
	actor Actor,
	defaultTaskID string,
	in BitSourceInput,
) (store.KnowledgeBitSourceRow, error) {
	kind := strings.TrimSpace(in.Kind)
	taskID := strings.TrimSpace(in.TaskID)
	if taskID == "" {
		taskID = defaultTaskID
	}
	source := store.KnowledgeBitSourceRow{Kind: kind}
	switch kind {
	case SourceTaskPrompt:
		if taskID == "" {
			return source, fmt.Errorf("%w: a task prompt needs a task", ErrSourceInvalid)
		}
		task, err := s.requireTask(ctx, actor, taskID, false)
		if err != nil {
			return source, err
		}
		source.TaskID = task.ID
		source.TaskRowID = sql.NullInt64{Int64: task.RowID, Valid: true}
		source.Label = "Task prompt · " + task.ID
		source.Href = "/tasks/" + task.ID
		source.At = sql.NullInt64{Int64: task.CreatedAt.UnixMilli(), Valid: true}
		return source, nil
	case SourceFollowUp:
		if taskID == "" {
			return source, fmt.Errorf("%w: a follow-up needs a task", ErrSourceInvalid)
		}
		task, err := s.requireTask(ctx, actor, taskID, false)
		if err != nil {
			return source, err
		}
		messages, err := s.store.ListTaskMessages(ctx, task.ID)
		if err != nil {
			return source, err
		}
		messageID := strings.TrimSpace(in.MessageID)
		var message *store.ListTaskMessagesRow
		for i := range messages {
			if messageID == "" || messages[i].ID == messageID {
				message = &messages[i]
				if messageID != "" {
					break
				}
			}
		}
		if message == nil {
			return source, fmt.Errorf("%w: no such follow-up message", ErrSourceInvalid)
		}
		who, _ := s.actorAttribution(ctx, ids.ActorID(message.ActorID.Int64))
		source.TaskID = task.ID
		source.TaskRowID = sql.NullInt64{Int64: task.RowID, Valid: true}
		source.MessageID = message.ID
		source.Label = "Follow-up message"
		if who != "" {
			source.Label += " · " + who
		}
		source.Href = "/tasks/" + task.ID + "?message=" + message.ID
		source.At = sql.NullInt64{Int64: message.CreatedAt, Valid: true}
		return source, nil
	case SourceQuestionnaire:
		if taskID == "" {
			return source, fmt.Errorf("%w: a questionnaire needs a task", ErrSourceInvalid)
		}
		requestID := strings.TrimSpace(in.RequestID)
		if requestID == "" {
			return source, fmt.Errorf("%w: a questionnaire needs a request id", ErrSourceInvalid)
		}
		task, err := s.requireTask(ctx, actor, taskID, false)
		if err != nil {
			return source, err
		}
		responses, err := s.store.ListQuestionnaireResponses(ctx, task.ID)
		if err != nil {
			return source, err
		}
		var response *store.ListQuestionnaireResponsesRow
		for i := range responses {
			if responses[i].RequestID == requestID {
				response = &responses[i]
				break
			}
		}
		if response == nil {
			return source, fmt.Errorf("%w: no such questionnaire answer", ErrSourceInvalid)
		}
		questionID := strings.TrimSpace(in.QuestionID)
		if questionID != "" && !tagPattern.MatchString(strings.ToLower(questionID)) {
			return source, fmt.Errorf("%w: invalid question id", ErrSourceInvalid)
		}
		answeredBy := ids.ActorID(response.ActorID.Int64)
		who, ref := s.actorAttribution(ctx, answeredBy)
		source.TaskID = task.ID
		source.TaskRowID = sql.NullInt64{Int64: task.RowID, Valid: true}
		source.RequestID = requestID
		source.QuestionID = questionID
		source.AnsweredByActorID = nullActor(answeredBy)
		source.AnsweredBy = ref
		source.Label = "Questionnaire answer"
		if who != "" {
			source.Label += " · " + who
		}
		source.Href = "/tasks/" + task.ID + "?request=" + requestID
		if questionID != "" {
			source.Href += "&question=" + questionID
		}
		source.At = sql.NullInt64{Int64: response.At, Valid: true}
		return source, nil
	case SourceSystemPrompt:
		if in.PromptID <= 0 {
			return source, fmt.Errorf("%w: a system prompt needs a prompt id", ErrSourceInvalid)
		}
		prompt, err := s.store.GetPrompt(ctx, in.PromptID)
		if err != nil {
			return source, fmt.Errorf("%w: no such prompt version", ErrSourceInvalid)
		}
		href, target := s.promptHref(ctx, prompt)
		source.PromptID = sql.NullInt64{Int64: prompt.ID, Valid: true}
		source.PromptScope = prompt.Scope
		source.PromptTargetID = target
		source.Label = promptScopeLabel(prompt.Scope) + " prompt v" +
			fmt.Sprint(prompt.Version)
		source.Href = href
		source.At = sql.NullInt64{Int64: prompt.CreatedAt, Valid: true}
		return source, nil
	default:
		return source, fmt.Errorf("%w: unknown source kind %q", ErrSourceInvalid, kind)
	}
}

func (s *Service) requireTask(
	ctx context.Context,
	actor Actor,
	id string,
	origin bool,
) (tasks.Task, error) {
	if s.tasks == nil {
		return tasks.Task{}, ErrSourceInvalid
	}
	task, err := s.tasks.Get(ctx, id)
	if err != nil {
		return tasks.Task{}, fmt.Errorf("%w: no such task", ErrSourceInvalid)
	}
	if task.ProjectID != "" {
		if err := s.requireRead(ctx, actor, ScopeRef{
			Scope: ScopeProject, ScopeID: task.ProjectID,
		}); err != nil {
			return tasks.Task{}, err
		}
	}
	if origin && actor.Kind == accounts.KindAgent {
		if task.AssigneeActorID != actor.ID && !s.sessionAgent(ctx, actor, task) {
			return tasks.Task{}, ErrForbidden
		}
	}
	return task, nil
}

func (s *Service) sessionAgent(
	ctx context.Context,
	actor Actor,
	task tasks.Task,
) bool {
	if s.agents == nil || task.SessionAgentID == 0 {
		return false
	}
	agent, err := s.agents.GetByActor(ctx, actor.ID)
	return err == nil && agent.ID == task.SessionAgentID
}

func (s *Service) actorAttribution(
	ctx context.Context,
	id ids.ActorID,
) (string, string) {
	if s.accounts == nil || id == 0 {
		return "", ""
	}
	account, err := s.accounts.GetActor(ctx, id)
	if err != nil {
		return "", ""
	}
	ref := Actor{
		ID:       account.ID,
		Kind:     account.Kind,
		Username: account.Username,
		Name:     account.Name,
	}.Ref()
	name := account.Name
	if name == "" {
		name = account.Username
	}
	return name, ref
}

func (s *Service) promptHref(
	ctx context.Context,
	prompt store.Prompt,
) (string, string) {
	target := ""
	switch prompt.Scope {
	case "org":
		return "/settings", ""
	case "project":
		target = prompt.ProjectID.String
		if slug, err := s.ProjectSlug(ctx, target); err == nil && slug != "" {
			return "/projects/" + slug + "/settings/tasks", target
		}
	case "agent_project":
		target = prompt.ProjectID.String
		if slug, err := s.ProjectSlug(ctx, target); err == nil && slug != "" {
			return "/projects/" + slug + "/settings/tasks", target
		}
	case "agent":
		target = actorid.Format(ids.ActorID(prompt.AgentID.Int64))
		if s.agents != nil {
			if agent, err := s.agents.Get(ctx, ids.AgentID(prompt.AgentID.Int64)); err == nil {
				target = agent.Username
				return "/agents/" + agent.Username + "/settings", target
			}
		}
	case "group":
		target = fmt.Sprint(prompt.GroupID.Int64)
		if agentGroup, err := s.store.GetAgentGroup(ctx, prompt.GroupID.Int64); err == nil {
			target = agentGroup.Slug
			return "/groups/" + agentGroup.Slug + "/prompts", target
		}
	case "team":
		target = fmt.Sprint(prompt.TeamID.Int64)
		return "/teams/" + target, target
	}
	return "", target
}

func promptScopeLabel(scope string) string {
	switch scope {
	case "org":
		return "Organization"
	case "project":
		return "Project"
	case "agent":
		return "Agent"
	case "agent_project":
		return "Agent project"
	case "group":
		return "Group"
	case "group_project":
		return "Group project"
	case "group_team":
		return "Group team"
	case "team":
		return "Team"
	case "team_project":
		return "Team project"
	default:
		return "Prompt"
	}
}

func (s *Service) linkTargets(
	ctx context.Context,
	actor Actor,
	selfID string,
	relatedIDs, supersedesIDs []string,
) ([]store.KnowledgeBitLinkWrite, []string, error) {
	links := make([]store.KnowledgeBitLinkWrite, 0, len(relatedIDs)+len(supersedesIDs))
	supersedes, err := s.linkKind(ctx, actor, selfID, relatedIDs, LinkRelated, &links)
	if err != nil {
		return nil, nil, err
	}
	retired, err := s.linkKind(ctx, actor, selfID, supersedesIDs, LinkSupersedes, &links)
	if err != nil {
		return nil, nil, err
	}
	return links, append(supersedes, retired...), nil
}

func (s *Service) linkKind(
	ctx context.Context,
	actor Actor,
	selfID string,
	targetIDs []string,
	kind string,
	links *[]store.KnowledgeBitLinkWrite,
) ([]string, error) {
	out := make([]string, 0, len(targetIDs))
	seen := map[string]struct{}{}
	for _, targetID := range targetIDs {
		targetID = strings.TrimSpace(targetID)
		if targetID == "" || targetID == selfID {
			continue
		}
		if _, ok := seen[targetID]; ok {
			continue
		}
		seen[targetID] = struct{}{}
		target, err := s.loadBit(ctx, targetID)
		if err != nil {
			return nil, fmt.Errorf("%w: no such knowledge bit %s", ErrSourceInvalid, targetID)
		}
		if err := s.requireRead(ctx, actor, bitScope(target)); err != nil {
			return nil, err
		}
		if kind == LinkSupersedes && target.Status == StatusSuperseded {
			return nil, fmt.Errorf(
				"%w: %s is already superseded", ErrSourceInvalid, targetID,
			)
		}
		*links = append(*links, store.KnowledgeBitLinkWrite{
			BitID:    selfID,
			TargetID: targetID,
			Kind:     kind,
			Position: int64(len(*links)),
		})
		if kind == LinkSupersedes {
			out = append(out, targetID)
		}
	}
	return out, nil
}

func (s *Service) retireTargets(ctx context.Context, byID string, targetIDs []string) error {
	if len(targetIDs) == 0 {
		return nil
	}

	return s.store.RetireKnowledgeBits(
		ctx, byID, targetIDs, s.now().UTC().UnixMilli(),
	)
}

func splitLinks(
	ctx context.Context,
	st store.Store,
	bitID string,
) ([]string, []string) {
	links, err := st.ListKnowledgeBitLinks(ctx, bitID)
	if err != nil {
		return nil, nil
	}
	related := make([]string, 0, len(links))
	supersedes := make([]string, 0, len(links))
	for _, link := range links {
		switch link.Kind {
		case LinkSupersedes:
			supersedes = append(supersedes, link.LinkedID)
		default:
			related = append(related, link.LinkedID)
		}
	}
	return related, supersedes
}

func existingLinks(
	ctx context.Context,
	st store.Store,
	bitID string,
) []store.KnowledgeBitLinkWrite {
	links, err := st.ListKnowledgeBitLinks(ctx, bitID)
	if err != nil {
		return nil
	}
	out := make([]store.KnowledgeBitLinkWrite, 0, len(links))
	for i, link := range links {
		out = append(out, store.KnowledgeBitLinkWrite{
			BitID:    bitID,
			TargetID: link.LinkedID,
			Kind:     link.Kind,
			Position: int64(i),
		})
	}
	return out
}

func supersedingIDs(
	ctx context.Context,
	st store.Store,
	bitID string,
) []string {
	links, err := st.ListKnowledgeBitLinks(ctx, bitID)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(links))
	for _, link := range links {
		if link.Kind == LinkSupersedes {
			out = append(out, link.LinkedID)
		}
	}
	return out
}

func bitScope(bit store.KnowledgeBitRow) ScopeRef {
	if !bit.ProjectID.Valid || bit.ProjectID.String == "" {
		return ScopeRef{Scope: ScopeOrg}
	}
	return ScopeRef{Scope: ScopeProject, ScopeID: bit.ProjectID.String}
}

func projectOf(bit store.KnowledgeBitRow) string {
	if !bit.ProjectID.Valid {
		return ""
	}
	return bit.ProjectID.String
}

func (s *Service) canEdit(ctx context.Context, actor Actor, bit store.KnowledgeBitRow) bool {
	canWrite, _ := s.CanWrite(ctx, actor, bitScope(bit))
	if !canWrite || bit.Status == StatusSuperseded {
		return false
	}
	return actor.ID == ids.ActorID(bit.CreatedByActorID.Int64) || actor.IsHuman()
}

func (s *Service) canDelete(ctx context.Context, actor Actor, bit store.KnowledgeBitRow) bool {
	return s.canEdit(ctx, actor, bit)
}

func (s *Service) requireReview(
	ctx context.Context,
	actor Actor,
	bit store.KnowledgeBitRow,
) error {
	if !actor.IsHuman() {
		return ErrReviewNotAllowed
	}
	if err := s.requireWrite(ctx, actor, bitScope(bit)); err != nil {
		return err
	}
	if bit.Status != StatusPending {
		return fmt.Errorf("%w: this bit is not pending", ErrConflict)
	}
	return nil
}

func (s *Service) bitFilter(
	ctx context.Context,
	actor Actor,
	filter BitFilter,
) (store.KnowledgeBitFilter, error) {
	out := store.KnowledgeBitFilter{
		Status: strings.TrimSpace(filter.Status),
		Tag:    strings.ToLower(strings.TrimSpace(filter.Tag)),
		Query:  strings.TrimSpace(filter.Query),
		TaskID: strings.TrimSpace(filter.TaskID),
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}
	if filter.AuthorActorID != "" {
		out.AuthorID = sql.NullInt64{Int64: parseActorID(filter.AuthorActorID), Valid: true}
	}
	switch strings.TrimSpace(filter.Scope) {
	case ScopeOrg:
		out.OrgOnly = true
		return out, nil
	case ScopeProject:
		if filter.ProjectID == "" {
			return out, ErrInvalidInput
		}
		if err := s.requireRead(ctx, actor, ScopeRef{
			Scope: ScopeProject, ScopeID: filter.ProjectID,
		}); err != nil {
			return out, err
		}
		projectID := filter.ProjectID
		out.ProjectID = &projectID
		return out, nil
	default:
		if filter.ProjectID != "" {
			if err := s.requireRead(ctx, actor, ScopeRef{
				Scope: ScopeProject, ScopeID: filter.ProjectID,
			}); err != nil {
				return out, err
			}
			projectID := filter.ProjectID
			out.ProjectID = &projectID
			return out, nil
		}
		visible, err := s.readableProjectIDs(ctx, actor)
		if err != nil {
			return out, err
		}
		out.ProjectIDs = visible
		return out, nil
	}
}

func (s *Service) readableProjectIDs(
	ctx context.Context,
	actor Actor,
) ([]string, error) {
	if s.projects == nil {
		return nil, nil
	}
	if actor.IsHuman() && actor.Admin {
		all, err := s.projects.List(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]string, len(all))
		for i, project := range all {
			out[i] = project.ID
		}
		return out, nil
	}
	mine, err := s.projects.ProjectsForActor(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(mine))
	for i, project := range mine {
		out[i] = project.ID
	}
	return out, nil
}

func (s *Service) loadBit(ctx context.Context, id string) (store.KnowledgeBitRow, error) {
	bit, err := s.store.GetKnowledgeBit(ctx, strings.TrimSpace(id))
	if errors.Is(err, sql.ErrNoRows) {
		return store.KnowledgeBitRow{}, ErrNotFound
	}
	if err != nil {
		return store.KnowledgeBitRow{}, err
	}
	return bit, nil
}

func mustTags(ctx context.Context, st store.Store, bitID string) []string {
	tags, err := st.KnowledgeBitTags(ctx, []string{bitID})
	if err != nil {
		return nil
	}
	return tags[bitID]
}

func mustSources(
	ctx context.Context,
	st store.Store,
	bitID string,
) []store.KnowledgeBitSourceRow {
	sources, err := st.ListKnowledgeBitSources(ctx, bitID)
	if err != nil {
		return nil
	}
	return sources
}

func mustLinks(
	ctx context.Context,
	st store.Store,
	bitID string,
	backlinks bool,
) []store.KnowledgeBitLinkRow {
	var links []store.KnowledgeBitLinkRow
	var err error
	if backlinks {
		links, err = st.ListKnowledgeBitBacklinks(ctx, bitID)
	} else {
		links, err = st.ListKnowledgeBitLinks(ctx, bitID)
	}
	if err != nil {
		return nil
	}
	return links
}

func mustTagsJSON(tags []string) string {
	raw, err := tagsJSONFor(tags)
	if err != nil {
		return "[]"
	}
	return raw
}

func sameIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, id := range a {
		seen[id]++
	}
	for _, id := range b {
		seen[id]--
		if seen[id] < 0 {
			return false
		}
	}
	return true
}

// deriveTitle uses the first non-empty line of the content as the title.
func deriveTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#>*- "))
		if line == "" {
			continue
		}
		if len([]rune(line)) > maxTitleRunes {
			return string([]rune(line)[:maxTitleRunes])
		}
		return line
	}
	return "Knowledge"
}
