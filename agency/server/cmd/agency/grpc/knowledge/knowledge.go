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

package knowledgegrpc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/knowledge"
	"cherrypi.vladfaust.com/agency/internal/store"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	Knowledge *knowledge.Service
	Agents    *agents.Registry
}

func (s *Server) ListConcepts(
	ctx context.Context,
	req *agencyv1.ListConceptsRequest,
) (*agencyv1.ListConceptsResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	ref := knowledge.ScopeRef{Scope: req.Scope, ScopeID: req.ScopeId}
	rows, err := s.Knowledge.List(
		ctx,
		actor,
		ref,
		req.Query,
		req.PathPrefix,
		req.Status,
		req.Tag,
		int(req.Limit),
	)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.KnowledgeConcept, len(rows))
	for i, row := range rows {
		out[i] = s.toProto(ctx, actor, knowledge.ConceptView{Concept: row}, false)
	}
	return &agencyv1.ListConceptsResponse{Concepts: out}, nil
}

func (s *Server) GetConcept(
	ctx context.Context,
	req *agencyv1.GetConceptRequest,
) (*agencyv1.GetConceptResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	ref := knowledge.ScopeRef{Scope: req.Scope, ScopeID: req.ScopeId}
	view, err := s.Knowledge.Get(ctx, actor, ref, req.Path)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.GetConceptResponse{Concept: s.toProto(ctx, actor, view, true)}, nil
}

func (s *Server) CreateConcept(
	ctx context.Context,
	req *agencyv1.CreateConceptRequest,
) (*agencyv1.CreateConceptResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	view, err := s.Knowledge.Create(ctx, actor, knowledge.CreateInput{
		Scope:                knowledge.ScopeRef{Scope: req.Scope, ScopeID: req.ScopeId},
		Path:                 req.Path,
		Type:                 req.Type,
		Title:                req.Title,
		Description:          req.Description,
		Resource:             req.Resource,
		Body:                 req.Body,
		Status:               req.Status,
		Tags:                 req.Tags,
		StaleAfter:           timePtr(req.StaleAfter),
		ExtraFrontmatterJSON: req.ExtraFrontmatterJson,
		Sources:              sourceInputs(req.Sources),
		Links:                linkInputs(req.Links),
		DecisionIDs:          req.DecisionIds,
		Summary:              req.Summary,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.CreateConceptResponse{Concept: s.toProto(ctx, actor, view, true)}, nil
}

func (s *Server) UpdateConcept(
	ctx context.Context,
	req *agencyv1.UpdateConceptRequest,
) (*agencyv1.UpdateConceptResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	patch := knowledge.Patch{
		Path:         req.NewPath,
		Type:         req.Type,
		Title:        req.Title,
		Description:  req.Description,
		Resource:     req.Resource,
		Body:         req.Body,
		Status:       req.Status,
		StaleAfter:   timePtr(req.StaleAfter),
		ClearStale:   req.ClearStaleAfter,
		BaseRevision: req.BaseRevision,
		Summary:      req.Summary,
	}
	if req.ReplaceTags {
		tags := req.Tags
		patch.Tags = &tags
	}
	if req.ExtraFrontmatterJson != nil {
		patch.ExtraFrontmatterJSON = req.ExtraFrontmatterJson
	}
	if req.ReplaceSources {
		sources := sourceInputs(req.Sources)
		patch.Sources = &sources
	}
	if req.ReplaceLinks {
		links := linkInputs(req.Links)
		patch.Links = &links
	}
	if req.ReplaceDecisions {
		decisions := req.DecisionIds
		patch.DecisionIDs = &decisions
	}
	view, err := s.Knowledge.Update(ctx, actor, knowledge.ScopeRef{
		Scope: req.Scope, ScopeID: req.ScopeId,
	}, req.Path, patch)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateConceptResponse{Concept: s.toProto(ctx, actor, view, true)}, nil
}

func (s *Server) DeleteConcept(
	ctx context.Context,
	req *agencyv1.DeleteConceptRequest,
) (*agencyv1.DeleteConceptResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	err := s.Knowledge.Delete(
		ctx,
		actor,
		knowledge.ScopeRef{Scope: req.Scope, ScopeID: req.ScopeId},
		req.Path,
		req.BaseRevision,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DeleteConceptResponse{}, nil
}

func (s *Server) ListRevisions(
	ctx context.Context,
	req *agencyv1.ListRevisionsRequest,
) (*agencyv1.ListRevisionsResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	revisions, err := s.Knowledge.Revisions(ctx, actor, knowledge.ScopeRef{
		Scope: req.Scope, ScopeID: req.ScopeId,
	}, req.Path)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.KnowledgeRevision, len(revisions))
	for i, revision := range revisions {
		out[i] = &agencyv1.KnowledgeRevision{
			Id:        revision.ID,
			Revision:  revision.Revision,
			Title:     revision.Title,
			Body:      revision.Body,
			Status:    revision.Status,
			ActorId:   actorid.Format(revision.ActorID.Int64),
			ActorRef:  revision.ActorRef,
			Summary:   revision.Summary,
			CreatedAt: timestamppb.New(timeFromMillis(revision.CreatedAt)),
		}
	}
	return &agencyv1.ListRevisionsResponse{Revisions: out}, nil
}

func (s *Server) VerifyConcept(
	ctx context.Context,
	req *agencyv1.VerifyConceptRequest,
) (*agencyv1.VerifyConceptResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	view, err := s.Knowledge.Verify(ctx, actor, knowledge.ScopeRef{
		Scope: req.Scope, ScopeID: req.ScopeId,
	}, req.Path, req.Note)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.VerifyConceptResponse{Concept: s.toProto(ctx, actor, view, true)}, nil
}

func (s *Server) RevokeVerification(
	ctx context.Context,
	req *agencyv1.RevokeVerificationRequest,
) (*agencyv1.RevokeVerificationResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	view, err := s.Knowledge.RevokeVerification(ctx, actor, knowledge.ScopeRef{
		Scope: req.Scope, ScopeID: req.ScopeId,
	}, req.Path, req.VerificationId)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RevokeVerificationResponse{
		Concept: s.toProto(ctx, actor, view, true),
	}, nil
}

func (s *Server) SearchKnowledge(
	ctx context.Context,
	req *agencyv1.SearchKnowledgeRequest,
) (*agencyv1.SearchKnowledgeResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	var scope *knowledge.ScopeRef
	if req.Scope != "" {
		scope = &knowledge.ScopeRef{Scope: req.Scope, ScopeID: req.ScopeId}
	}
	rows, err := s.Knowledge.Search(ctx, actor, req.Query, scope, int(req.Limit))
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.KnowledgeConcept, len(rows))
	for i, row := range rows {
		out[i] = s.toProto(ctx, actor, knowledge.ConceptView{Concept: row}, false)
	}
	return &agencyv1.SearchKnowledgeResponse{Concepts: out}, nil
}

func (s *Server) ListDecisions(
	ctx context.Context,
	req *agencyv1.ListDecisionsRequest,
) (*agencyv1.ListDecisionsResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	decisions, err := s.Knowledge.Decisions(ctx, actor, knowledge.DecisionFilter{
		Scope:    req.Scope,
		ScopeID:  req.ScopeId,
		TaskID:   req.TaskId,
		Kind:     req.Kind,
		BeforeID: req.BeforeId,
		Limit:    int(req.Limit),
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.Decision, len(decisions))
	for i, decision := range decisions {
		out[i] = s.decisionProto(ctx, decision)
	}
	return &agencyv1.ListDecisionsResponse{Decisions: out}, nil
}

func (s *Server) RecordDecision(
	ctx context.Context,
	req *agencyv1.RecordDecisionRequest,
) (*agencyv1.RecordDecisionResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	decision, err := s.Knowledge.RecordDecision(ctx, actor, knowledge.DecisionInput{
		Scope:        knowledge.ScopeRef{Scope: req.Scope, ScopeID: req.ScopeId},
		TaskID:       req.TaskId,
		Summary:      req.Summary,
		Rationale:    req.Rationale,
		ConceptPaths: req.ConceptPaths,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RecordDecisionResponse{
		Decision: s.decisionProto(ctx, decision),
	}, nil
}

func (s *Server) decisionProto(
	ctx context.Context,
	decision knowledge.Decision,
) *agencyv1.Decision {
	row := decision.Row
	out := &agencyv1.Decision{
		Id:           row.ID,
		TaskId:       row.TaskID,
		Kind:         row.Kind,
		ActorId:      actorid.Format(row.ActorID.Int64),
		ActorRef:     row.ActorRef,
		Summary:      row.Summary,
		PayloadJson:  row.PayloadJson,
		ConceptPaths: decision.ConceptPaths,
		ProjectSlug:  decision.ProjectSlug,
		CreatedAt:    timestamppb.New(timeFromMillis(row.CreatedAt)),
	}
	if row.ProjectID.Valid {
		out.ProjectId = row.ProjectID.String
	}
	return out
}

func (s *Server) GetMemoryContext(
	ctx context.Context,
	req *agencyv1.GetMemoryContextRequest,
) (*agencyv1.GetMemoryContextResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	snapshot, err := s.Knowledge.Context(ctx, actor, req.ProjectId)
	if err != nil {
		return nil, mapError(err)
	}
	layers := make([]*agencyv1.MemoryLayer, 0, len(snapshot.Layers))
	for _, layer := range snapshot.Layers {
		layers = append(layers, &agencyv1.MemoryLayer{
			Name:      layer.Name,
			Path:      layer.Path,
			Chars:     int32(layer.Chars),
			Truncated: layer.Truncated,
		})
	}
	out := &agencyv1.GetMemoryContextResponse{
		Text:       snapshot.Text,
		Hash:       snapshot.Hash,
		SnapshotAt: timestamppb.New(snapshot.SnapshotAt),
		ProjectId:  snapshot.ProjectID,
		Layers:     layers,
	}
	return out, nil
}

func (s *Server) WriteMemory(
	ctx context.Context,
	req *agencyv1.WriteMemoryRequest,
) (*agencyv1.WriteMemoryResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	file, dirty, err := s.Knowledge.WriteMemory(ctx, actor, knowledge.MemoryWriteInput{
		ProjectID: req.ProjectId,
		Target:    req.Target,
		Path:      req.Path,
		Content:   req.Content,
		Replace:   req.Replace,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.WriteMemoryResponse{
		File:          memoryFileProto(file),
		SnapshotDirty: dirty,
	}, nil
}

func (s *Server) ForgetMemory(
	ctx context.Context,
	req *agencyv1.ForgetMemoryRequest,
) (*agencyv1.ForgetMemoryResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	result, err := s.Knowledge.Forget(ctx, actor, req.Query)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.ForgetMemoryResponse{
		RecoveryId: result.RecoveryID,
		Files:      int32(result.Files),
		Entries:    int32(result.Entries),
	}, nil
}

func (s *Server) RestoreMemory(
	ctx context.Context,
	req *agencyv1.RestoreMemoryRequest,
) (*agencyv1.RestoreMemoryResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	files, err := s.Knowledge.RestoreMemory(ctx, actor, req.RecoveryId)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RestoreMemoryResponse{RestoredFiles: int32(files)}, nil
}

func (s *Server) Scratchpad(
	ctx context.Context,
	req *agencyv1.ScratchpadRequest,
) (*agencyv1.ScratchpadResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	body, err := s.Knowledge.Scratchpad(ctx, actor, knowledge.ScratchpadInput{
		Action: req.Action,
		Text:   req.Text,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.ScratchpadResponse{Content: body}, nil
}

func (s *Server) GetMemoryStatus(
	ctx context.Context,
	req *agencyv1.GetMemoryStatusRequest,
) (*agencyv1.GetMemoryStatusResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	status, err := s.Knowledge.Status(ctx, actor, req.ProjectId)
	if err != nil {
		return nil, mapError(err)
	}
	files := make([]*agencyv1.KnowledgeConcept, 0, len(status.Files))
	for _, file := range status.Files {
		files = append(files, memoryFileProto(file))
	}
	return &agencyv1.GetMemoryStatusResponse{
		ProjectId:  status.ProjectID,
		TodayPath:  status.TodayPath,
		TotalChars: int32(status.TotalChars),
		Files:      files,
		Context:    memoryContextProto(status.Context),
	}, nil
}

func (s *Server) ListAgentMemory(
	ctx context.Context,
	req *agencyv1.ListAgentMemoryRequest,
) (*agencyv1.ListAgentMemoryResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	agentActorID, err := s.agentActor(ctx, req.AgentId)
	if err != nil {
		return nil, mapError(err)
	}
	files, err := s.Knowledge.MemoryFiles(ctx, actor, agentActorID)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.KnowledgeConcept, 0, len(files))
	for _, file := range files {
		summary := memoryFileProto(file)
		summary.ScopeId = actorid.Format(agentActorID)
		summary.Body = ""
		out = append(out, summary)
	}
	return &agencyv1.ListAgentMemoryResponse{Concepts: out}, nil
}

func (s *Server) GetAgentMemoryFile(
	ctx context.Context,
	req *agencyv1.GetAgentMemoryFileRequest,
) (*agencyv1.GetAgentMemoryFileResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	agentActorID, err := s.agentActor(ctx, req.AgentId)
	if err != nil {
		return nil, mapError(err)
	}
	file, err := s.Knowledge.MemoryFileAt(ctx, actor, agentActorID, req.Path)
	if err != nil {
		return nil, mapError(err)
	}
	concept := memoryFileProto(file)
	concept.ScopeId = actorid.Format(agentActorID)
	return &agencyv1.GetAgentMemoryFileResponse{Concept: concept}, nil
}

func (s *Server) agentActor(ctx context.Context, raw string) (ids.ActorID, error) {
	if raw == "" {
		return 0, knowledge.ErrInvalidInput
	}
	if s.Agents == nil {
		return 0, knowledge.ErrNotFound
	}
	if agent, err := s.Agents.Get(ctx, actorid.ParseAgent(raw)); err == nil {
		return agent.ActorID, nil
	}
	if agent, err := s.Agents.GetByActor(ctx, actorid.ParseActor(raw)); err == nil {
		return agent.ActorID, nil
	}
	return 0, knowledge.ErrNotFound
}

func (s *Server) toProto(
	ctx context.Context,
	actor knowledge.Actor,
	view knowledge.ConceptView,
	full bool,
) *agencyv1.KnowledgeConcept {
	c := view.Concept
	out := &agencyv1.KnowledgeConcept{
		Id:                 c.ID,
		Scope:              c.Scope,
		ScopeId:            c.ScopeID,
		Path:               c.Path,
		Type:               c.Type,
		Title:              c.Title,
		Description:        c.Description,
		Resource:           c.Resource,
		Tags:               view.Tags,
		BodySize:           int32(utf8.RuneCountInString(c.Body)),
		Status:             c.Status,
		GeneratedByActorId: actorid.Format(c.GeneratedByActorID.Int64),
		GeneratedBy:        c.GeneratedBy,
		GeneratedAt:        timestamppb.New(timeFromMillis(c.GeneratedAt)),
		TrustTier:          c.TrustTier,
		Revision:           c.Revision,
		Protected:          c.Protected != 0,
		CreatedAt:          timestamppb.New(timeFromMillis(c.CreatedAt)),
		UpdatedAt:          timestamppb.New(timeFromMillis(c.UpdatedAt)),
	}
	if full {
		out.Body = c.Body
		out.FrontmatterJson = c.FrontmatterJson
	}
	if c.StaleAfter.Valid {
		out.StaleAfter = timestamppb.New(timeFromMillis(c.StaleAfter.Int64))
	}
	ref := knowledge.ScopeRef{Scope: c.Scope, ScopeID: c.ScopeID}
	if ok, err := s.Knowledge.CanWrite(ctx, actor, ref); err == nil {
		out.CanEdit = ok
	}

	projectSlug := ""
	if len(view.Decisions) > 0 && c.Scope == knowledge.ScopeProject {
		if slug, err := s.Knowledge.ProjectSlug(ctx, c.ScopeID); err == nil {
			projectSlug = slug
		}
	}
	out.CanVerify = actor.IsHuman() && out.CanEdit
	for _, source := range view.Sources {
		item := &agencyv1.KnowledgeSource{
			Id:       source.SourceID,
			Resource: source.Resource,
			Title:    source.Title,
			Author:   source.AuthorRef,
		}
		if source.UsageCount.Valid {
			v := source.UsageCount.Int64
			item.UsageCount = &v
		}
		if source.LastModified.Valid {
			v := source.LastModified.Int64
			item.LastModified = &v
		}
		out.Sources = append(out.Sources, item)
	}
	for _, link := range view.Links {
		out.Links = append(out.Links, &agencyv1.KnowledgeLink{
			TargetPath: link.TargetPath,
			Kind:       link.Kind,
		})
	}
	for _, backlink := range view.Backlinks {
		out.Backlinks = append(out.Backlinks, &agencyv1.KnowledgeBacklink{
			Id:      backlink.ID,
			Scope:   backlink.Scope,
			ScopeId: backlink.ScopeID,
			Path:    backlink.Path,
			Title:   backlink.Title,
		})
	}
	for _, decision := range view.Decisions {
		out.Decisions = append(out.Decisions, s.decisionProto(ctx, knowledge.Decision{
			Row:         decision,
			ProjectSlug: projectSlug,
		}))
	}
	for _, verification := range view.Verifications {
		item := &agencyv1.KnowledgeVerification{
			Id:               verification.ID,
			ActorId:          actorid.Format(verification.ActorID.Int64),
			ActorRef:         verification.ActorRef,
			Note:             verification.Note,
			CreatedAt:        timestamppb.New(timeFromMillis(verification.CreatedAt)),
			RevokedByActorId: actorid.Format(verification.RevokedByActorID.Int64),
		}
		if verification.RevokedAt.Valid {
			item.RevokedAt = timestamppb.New(timeFromMillis(verification.RevokedAt.Int64))
		}
		out.Verifications = append(out.Verifications, item)
	}
	return out
}

func caller(ctx context.Context) (knowledge.Actor, bool) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return knowledge.Actor{}, false
	}
	return knowledge.ActorFromIdentity(identity), true
}

func sourceInputs(items []*agencyv1.KnowledgeSourceInput) []knowledge.SourceInput {
	out := make([]knowledge.SourceInput, 0, len(items))
	for _, item := range items {
		out = append(out, knowledge.SourceInput{
			ID:        item.Id,
			Resource:  item.Resource,
			Title:     item.Title,
			AuthorRef: item.Author,
		})
	}
	return out
}

func linkInputs(items []*agencyv1.KnowledgeLinkInput) []knowledge.LinkInput {
	out := make([]knowledge.LinkInput, 0, len(items))
	for _, item := range items {
		out = append(out, knowledge.LinkInput{
			TargetPath: item.TargetPath,
			Kind:       item.Kind,
		})
	}
	return out
}

func memoryFileProto(file knowledge.MemoryFile) *agencyv1.KnowledgeConcept {
	return &agencyv1.KnowledgeConcept{
		Scope:     knowledge.ScopeAgent,
		Path:      file.Path,
		Type:      "Memory",
		Title:     file.Title,
		Body:      file.Body,
		BodySize:  int32(utf8.RuneCountInString(file.Body)),
		Revision:  file.Revision,
		UpdatedAt: timestamppb.New(file.UpdatedAt),
		TrustTier: knowledge.TrustUnverified,
		Status:    knowledge.StatusStable,
	}
}

func memoryContextProto(snapshot knowledge.MemoryContext) *agencyv1.GetMemoryContextResponse {
	layers := make([]*agencyv1.MemoryLayer, 0, len(snapshot.Layers))
	for _, layer := range snapshot.Layers {
		layers = append(layers, &agencyv1.MemoryLayer{
			Name:      layer.Name,
			Path:      layer.Path,
			Chars:     int32(layer.Chars),
			Truncated: layer.Truncated,
		})
	}
	return &agencyv1.GetMemoryContextResponse{
		Text:       snapshot.Text,
		Hash:       snapshot.Hash,
		SnapshotAt: timestamppb.New(snapshot.SnapshotAt),
		ProjectId:  snapshot.ProjectID,
		Layers:     layers,
	}
}

func timeFromMillis(ms int64) time.Time {
	return time.UnixMilli(ms).UTC()
}

func timePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func mapError(err error) *connect.Error {
	switch {
	case errors.Is(err, knowledge.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, knowledge.ErrForbidden),
		errors.Is(err, knowledge.ErrAgentMemoryOnly),
		errors.Is(err, knowledge.ErrVerifyNotAllowed):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, knowledge.ErrProtected):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, knowledge.ErrConflict),
		errors.Is(err, store.ErrKnowledgeConflict):
		return connect.NewError(connect.CodeAborted, err)
	case errors.Is(err, knowledge.ErrInvalidPath),
		errors.Is(err, knowledge.ErrInvalidInput),
		errors.Is(err, knowledge.ErrInvalidScope):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:

		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "sql") {
			msg = "knowledge operation failed"
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("%s", msg))
	}
}
