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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	DecisionTaskCreated      = "task_created"
	DecisionFollowUp         = "follow_up"
	DecisionQuestionAnswered = "question_answered"
	DecisionTaskSettled      = "task_settled"
	DecisionRecorded         = "recorded"
)

const decisionSummaryCap = 600

var ErrDecisionNotFound = errors.New("decision not found")

type Decision struct {
	Row          store.KnowledgeDecision
	ConceptPaths []string

	ProjectSlug string
}

type DecisionInput struct {
	Scope        ScopeRef
	TaskID       string
	Summary      string
	Rationale    string
	ConceptPaths []string
}

type CapturedDecision struct {
	Kind string

	EventID      string
	TaskID       string
	ProjectID    string
	ActorID      ids.ActorID
	ActorRef     string
	Summary      string
	Payload      map[string]any
	CreatedAt    time.Time
	ConceptPaths []string
}

type DecisionFilter struct {
	Scope    string
	ScopeID  string
	TaskID   string
	Kind     string
	BeforeID int64
	Limit    int
}

func (s *Service) RecordDecision(
	ctx context.Context,
	actor Actor,
	in DecisionInput,
) (Decision, error) {
	if strings.TrimSpace(in.Summary) == "" {
		return Decision{}, fmt.Errorf("%w: a decision needs a summary", ErrInvalidInput)
	}
	ref := normalizeRef(actor, in.Scope)
	if ref.Scope == "" {
		ref = ScopeRef{Scope: ScopeOrg}
	}
	if err := s.requireWrite(ctx, actor, ref); err != nil {
		return Decision{}, err
	}

	var projectID sql.NullString
	if ref.Scope == ScopeProject {
		projectID = sql.NullString{String: ref.ScopeID, Valid: true}
	}
	if ref.Scope == ScopeAgent {
		return Decision{}, fmt.Errorf("%w: decisions live in a wiki scope", ErrInvalidInput)
	}

	conceptIDs, err := s.resolveDecisionConcepts(ctx, actor, ref, in.ConceptPaths)
	if err != nil {
		return Decision{}, err
	}

	payload := map[string]any{}
	if strings.TrimSpace(in.Rationale) != "" {
		payload["rationale"] = strings.TrimSpace(in.Rationale)
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return Decision{}, err
	}
	now := s.now().UTC()
	row, created, err := s.store.CreateKnowledgeDecision(ctx, store.KnowledgeDecision{
		ProjectID:   projectID,
		TaskID:      strings.TrimSpace(in.TaskID),
		Kind:        DecisionRecorded,
		ActorID:     nullActor(actor.ID),
		ActorRef:    actor.Ref(),
		Summary:     truncateSummary(in.Summary),
		PayloadJson: string(rawPayload),
		EventID:     fmt.Sprintf("recorded:%d", now.UnixNano()),
		CreatedAt:   now.UnixMilli(),
	})
	if err != nil {
		return Decision{}, err
	}
	if !created {
		return Decision{}, fmt.Errorf("%w: the decision was already recorded", ErrConflict)
	}
	if err := s.store.LinkKnowledgeDecisionConcepts(ctx, row.ID, conceptIDs); err != nil {
		return Decision{}, err
	}
	s.signal()
	return s.decisionView(ctx, row, s.projectSlugs(ctx))
}

func (s *Service) CaptureTaskDecision(
	ctx context.Context,
	in CapturedDecision,
) error {
	kind := in.Kind
	switch kind {
	case DecisionTaskCreated, DecisionFollowUp, DecisionQuestionAnswered,
		DecisionTaskSettled:
	default:
		return fmt.Errorf("%w: %q is not a captured decision", ErrInvalidInput, kind)
	}
	payload := "{}"
	if len(in.Payload) > 0 {
		raw, err := json.Marshal(in.Payload)
		if err != nil {
			return err
		}
		payload = string(raw)
	}
	var projectID sql.NullString
	if in.ProjectID != "" {
		projectID = sql.NullString{String: in.ProjectID, Valid: true}
	}
	created := in.CreatedAt
	if created.IsZero() {
		created = s.now().UTC()
	}
	eventID := in.EventID
	if eventID == "" {
		eventID = fmt.Sprintf("%d", created.UnixNano())
	}
	actorRef := in.ActorRef
	if actorRef == "" {
		actorRef = s.actorRefFor(ctx, in.ActorID)
	}
	row, inserted, err := s.store.CreateKnowledgeDecision(ctx, store.KnowledgeDecision{
		ProjectID:   projectID,
		TaskID:      in.TaskID,
		Kind:        kind,
		ActorID:     nullActor(in.ActorID),
		ActorRef:    actorRef,
		Summary:     truncateSummary(in.Summary),
		PayloadJson: payload,
		EventID:     eventID,
		CreatedAt:   created.UnixMilli(),
	})
	if err != nil || !inserted {
		return err
	}
	conceptIDs, err := s.resolveDecisionConcepts(ctx, Actor{}, ScopeRef{
		Scope: scopeForProject(projectID), ScopeID: projectID.String,
	}, in.ConceptPaths)
	if err != nil {
		return err
	}
	if len(conceptIDs) > 0 {
		if err := s.store.LinkKnowledgeDecisionConcepts(ctx, row.ID, conceptIDs); err != nil {
			return err
		}
	}
	s.signal()
	return nil
}

func (s *Service) actorRefFor(ctx context.Context, id ids.ActorID) string {
	if id == 0 {
		return "process:cherrypi"
	}
	if s.accounts == nil {
		return fmt.Sprintf("human:%d", id)
	}
	actor, err := s.accounts.GetActor(ctx, id)
	if err != nil {
		return fmt.Sprintf("human:%d", id)
	}
	return Actor{
		ID:       id,
		Kind:     actor.Kind,
		Username: actor.Username,
		Name:     actor.Name,
	}.Ref()
}

func scopeForProject(projectID sql.NullString) string {
	if projectID.Valid && projectID.String != "" {
		return ScopeProject
	}
	return ScopeOrg
}

func (s *Service) resolveDecisionConcepts(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	paths []string,
) ([]string, error) {
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		path = strings.TrimSpace(strings.TrimSuffix(path, ".md"))
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, path)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		if err != nil {
			return nil, err
		}
		seen[path] = struct{}{}
		out = append(out, concept.ID)
	}
	return out, nil
}

func (s *Service) Decisions(
	ctx context.Context,
	actor Actor,
	f DecisionFilter,
) ([]Decision, error) {
	storeFilter := store.DecisionFilter{
		TaskID:   strings.TrimSpace(f.TaskID),
		Kind:     strings.TrimSpace(f.Kind),
		BeforeID: f.BeforeID,
		Limit:    f.Limit,
	}
	switch f.Scope {
	case "", ScopeOrg:

		storeFilter.OrgWide = true
	case ScopeProject:
		ref := ScopeRef{Scope: ScopeProject, ScopeID: f.ScopeID}
		if err := s.requireRead(ctx, actor, ref); err != nil {
			return nil, err
		}
		projectID := f.ScopeID
		storeFilter.ProjectID = &projectID
	case ScopeAgent:
		return nil, fmt.Errorf("%w: decisions live in a wiki scope", ErrInvalidInput)
	default:
		return nil, ErrInvalidScope
	}

	if storeFilter.TaskID != "" {
		storeFilter.OrgWide = true
	}
	if storeFilter.OrgWide {
		readable, err := s.readableProjectIDs(ctx, actor)
		if err != nil {
			return nil, err
		}
		storeFilter.ReadableProjectIDs = readable
	}

	rows, err := s.store.ListKnowledgeDecisions(ctx, storeFilter)
	if err != nil {
		return nil, err
	}

	slugs := s.projectSlugs(ctx)
	out := make([]Decision, 0, len(rows))
	for _, row := range rows {
		view, err := s.decisionView(ctx, row, slugs)
		if err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *Service) readableProjectIDs(
	ctx context.Context,
	actor Actor,
) ([]string, error) {
	if s.projects == nil {
		return nil, nil
	}
	var (
		list []projects.Project
		err  error
	)
	if actor.IsHuman() && actor.Admin {
		list, err = s.projects.List(ctx)
	} else {
		list, err = s.projects.ProjectsForActor(ctx, actor.ID)
	}
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, project := range list {
		ids = append(ids, project.ID)
	}
	return ids, nil
}

func (s *Service) ProjectSlug(ctx context.Context, projectID string) (string, error) {
	if s.projects == nil {
		return "", nil
	}
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return project.Slug, nil
}

func (s *Service) projectSlugs(ctx context.Context) map[string]string {
	if s.projects == nil {
		return nil
	}
	list, err := s.projects.List(ctx)
	if err != nil {
		return nil
	}
	out := make(map[string]string, len(list))
	for _, project := range list {
		out[project.ID] = project.Slug
	}
	return out
}

func (s *Service) decisionView(
	ctx context.Context,
	row store.KnowledgeDecision,
	slugs map[string]string,
) (Decision, error) {
	paths, err := s.store.ListKnowledgeDecisionConceptPaths(ctx, row.ID)
	if err != nil {
		return Decision{}, err
	}
	view := Decision{Row: row, ConceptPaths: paths}
	if row.ProjectID.Valid {
		view.ProjectSlug = slugs[row.ProjectID.String]
	}
	return view, nil
}

func truncateSummary(text string) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= decisionSummaryCap {
		return text
	}
	runes := []rune(text)
	return string(runes[:decisionSummaryCap-1]) + "…"
}
