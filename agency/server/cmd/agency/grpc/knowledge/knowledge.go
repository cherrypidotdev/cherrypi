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
	"encoding/json"
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

func (s *Server) ListKnowledge(
	ctx context.Context,
	req *agencyv1.ListKnowledgeRequest,
) (*agencyv1.ListKnowledgeResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	views, err := s.Knowledge.List(ctx, actor, knowledge.BitFilter{
		Scope:         req.Scope,
		ProjectID:     req.ProjectId,
		Tag:           req.Tag,
		Status:        req.Status,
		Query:         req.Query,
		TaskID:        req.TaskId,
		AuthorActorID: req.AuthorActorId,
		Limit:         int(req.Limit),
		Offset:        int(req.Offset),
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.KnowledgeBit, len(views))
	for i, view := range views {
		out[i] = s.bitProto(ctx, view)
	}
	return &agencyv1.ListKnowledgeResponse{Bits: out}, nil
}

func (s *Server) GetKnowledge(
	ctx context.Context,
	req *agencyv1.GetKnowledgeRequest,
) (*agencyv1.GetKnowledgeResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	view, err := s.Knowledge.Get(ctx, actor, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.GetKnowledgeResponse{Bit: s.bitProto(ctx, view)}, nil
}

func (s *Server) AddKnowledge(
	ctx context.Context,
	req *agencyv1.AddKnowledgeRequest,
) (*agencyv1.AddKnowledgeResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	view, err := s.Knowledge.Add(ctx, actor, knowledge.BitInput{
		ProjectID:     req.ProjectId,
		Title:         req.Title,
		Content:       req.Content,
		Tags:          req.Tags,
		Sources:       sourceInputs(req.Sources),
		RelatedIDs:    req.RelatedIds,
		SupersedesIDs: req.SupersedesIds,
		OriginTaskID:  req.OriginTaskId,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.AddKnowledgeResponse{Bit: s.bitProto(ctx, view)}, nil
}

func (s *Server) UpdateKnowledge(
	ctx context.Context,
	req *agencyv1.UpdateKnowledgeRequest,
) (*agencyv1.UpdateKnowledgeResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	patch := knowledge.BitPatch{
		Title:        req.Title,
		Content:      req.Content,
		BaseRevision: req.BaseRevision,
		Summary:      req.Summary,
	}
	if req.ReplaceTags {
		tags := req.Tags
		patch.Tags = &tags
	}
	if req.ReplaceSources {
		sources := sourceInputs(req.Sources)
		patch.Sources = &sources
	}
	if req.ReplaceRelated {
		related := req.RelatedIds
		patch.RelatedIDs = &related
	}
	if req.ReplaceSupersedes {
		supersedes := req.SupersedesIds
		patch.SupersedesIDs = &supersedes
	}
	view, err := s.Knowledge.Update(ctx, actor, req.Id, patch)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateKnowledgeResponse{Bit: s.bitProto(ctx, view)}, nil
}

func (s *Server) ApproveKnowledge(
	ctx context.Context,
	req *agencyv1.ApproveKnowledgeRequest,
) (*agencyv1.ApproveKnowledgeResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	view, err := s.Knowledge.Approve(ctx, actor, req.Id, req.Note)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.ApproveKnowledgeResponse{Bit: s.bitProto(ctx, view)}, nil
}

func (s *Server) RejectKnowledge(
	ctx context.Context,
	req *agencyv1.RejectKnowledgeRequest,
) (*agencyv1.RejectKnowledgeResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	view, err := s.Knowledge.Reject(ctx, actor, req.Id, req.Note)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RejectKnowledgeResponse{Bit: s.bitProto(ctx, view)}, nil
}

func (s *Server) DeleteKnowledge(
	ctx context.Context,
	req *agencyv1.DeleteKnowledgeRequest,
) (*agencyv1.DeleteKnowledgeResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	if err := s.Knowledge.Delete(ctx, actor, req.Id); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DeleteKnowledgeResponse{}, nil
}

func (s *Server) ListKnowledgeRevisions(
	ctx context.Context,
	req *agencyv1.ListKnowledgeRevisionsRequest,
) (*agencyv1.ListKnowledgeRevisionsResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	rows, err := s.Knowledge.Revisions(ctx, actor, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.KnowledgeRevision, len(rows))
	for i, row := range rows {
		out[i] = &agencyv1.KnowledgeRevision{
			Id:        row.ID,
			Revision:  row.Revision,
			Title:     row.Title,
			Content:   row.Content,
			Tags:      tagsFromJSON(row.TagsJSON),
			Status:    row.Status,
			ActorId:   actorid.Format(ids.ActorID(row.ActorID.Int64)),
			ActorRef:  row.ActorRef,
			Summary:   row.Summary,
			CreatedAt: timestamppb.New(timeFromMillis(row.CreatedAt)),
		}
	}
	return &agencyv1.ListKnowledgeRevisionsResponse{Revisions: out}, nil
}

func (s *Server) SearchKnowledge(
	ctx context.Context,
	req *agencyv1.SearchKnowledgeRequest,
) (*agencyv1.SearchKnowledgeResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	views, err := s.Knowledge.Search(ctx, actor, knowledge.BitFilter{
		Scope:     req.Scope,
		ProjectID: req.ProjectId,
		Tag:       req.Tag,
		Status:    req.Status,
		Query:     req.Query,
		Limit:     int(req.Limit),
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.KnowledgeBit, len(views))
	for i, view := range views {
		out[i] = s.bitProto(ctx, view)
	}
	return &agencyv1.SearchKnowledgeResponse{Bits: out}, nil
}

func (s *Server) ListKnowledgeTags(
	ctx context.Context,
	req *agencyv1.ListKnowledgeTagsRequest,
) (*agencyv1.ListKnowledgeTagsResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	tags, err := s.Knowledge.Tags(ctx, actor, knowledge.BitFilter{
		Scope:     req.Scope,
		ProjectID: req.ProjectId,
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.KnowledgeTag, len(tags))
	for i, tag := range tags {
		out[i] = &agencyv1.KnowledgeTag{Tag: tag.Tag, Count: int32(tag.Count)}
	}
	return &agencyv1.ListKnowledgeTagsResponse{Tags: out}, nil
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
	return memoryContextProto(snapshot), nil
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
	files := make([]*agencyv1.MemoryFile, 0, len(status.Files))
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

func (s *Server) SearchMemory(
	ctx context.Context,
	req *agencyv1.SearchMemoryRequest,
) (*agencyv1.SearchMemoryResponse, error) {
	actor, ok := caller(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, knowledge.ErrForbidden)
	}
	rows, err := s.Knowledge.SearchMemory(ctx, actor, req.Query, int(req.Limit))
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.MemoryFile, len(rows))
	for i, row := range rows {
		out[i] = memoryFileProto(knowledge.MemoryFile{
			Path:      row.Path,
			Body:      row.Body,
			Title:     row.Title,
			UpdatedAt: timeFromMillis(row.UpdatedAt),
			Revision:  row.Revision,
		})
	}
	return &agencyv1.SearchMemoryResponse{Files: out}, nil
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
	out := make([]*agencyv1.MemoryFile, 0, len(files))
	for _, file := range files {
		summary := memoryFileProto(file)
		summary.BodySize = int32(utf8.RuneCountInString(summary.Body))
		summary.Body = ""
		out = append(out, summary)
	}
	return &agencyv1.ListAgentMemoryResponse{Files: out}, nil
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
	return &agencyv1.GetAgentMemoryFileResponse{File: memoryFileProto(file)}, nil
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

func (s *Server) bitProto(
	ctx context.Context,
	view knowledge.BitView,
) *agencyv1.KnowledgeBit {
	bit := view.Bit
	out := &agencyv1.KnowledgeBit{
		Id:                bit.ID,
		ProjectId:         projectOf(bit),
		ProjectSlug:       view.ProjectSlug,
		Title:             bit.Title,
		Content:           bit.Content,
		Tags:              view.Tags,
		Status:            bit.Status,
		OriginTaskId:      bit.OriginTaskID,
		CreatedByActorId:  actorid.Format(ids.ActorID(bit.CreatedByActorID.Int64)),
		CreatedBy:         bit.CreatedBy,
		CreatedAt:         timestamppb.New(timeFromMillis(bit.CreatedAt)),
		UpdatedAt:         timestamppb.New(timeFromMillis(bit.UpdatedAt)),
		ReviewedByActorId: actorid.Format(ids.ActorID(bit.ReviewedByActorID.Int64)),
		ReviewedBy:        bit.ReviewedBy,
		ReviewNote:        bit.ReviewNote,
		Revision:          bit.Revision,
		ContentSize:       int32(utf8.RuneCountInString(bit.Content)),
		SupersededById:    bit.SupersededByID.String,
		CanEdit:           view.CanEdit,
		CanReview:         view.CanReview,
		CanDelete:         view.CanDelete,
	}
	if bit.ReviewedAt.Valid {
		out.ReviewedAt = timestamppb.New(timeFromMillis(bit.ReviewedAt.Int64))
	}
	for _, source := range view.Sources {
		out.Sources = append(out.Sources, &agencyv1.KnowledgeSource{
			Id:                source.ID,
			Kind:              source.Kind,
			TaskId:            source.TaskID,
			MessageId:         source.MessageID,
			RequestId:         source.RequestID,
			QuestionId:        source.QuestionID,
			AnsweredByActorId: actorid.Format(ids.ActorID(source.AnsweredByActorID.Int64)),
			AnsweredBy:        source.AnsweredBy,
			PromptId:          source.PromptID.Int64,
			PromptScope:       source.PromptScope,
			PromptTargetId:    source.PromptTargetID,
			Label:             source.Label,
			Href:              source.Href,
			At:                timestampOrNil(source.At),
		})
	}
	for _, link := range view.Links {
		out.Links = append(out.Links, linkProto(link))
	}
	for _, link := range view.Backlinks {
		out.Backlinks = append(out.Backlinks, linkProto(link))
	}
	for _, rev := range view.Revisions {
		out.Revisions = append(out.Revisions, &agencyv1.KnowledgeRevision{
			Id:        rev.ID,
			Revision:  rev.Revision,
			Title:     rev.Title,
			Content:   rev.Content,
			Tags:      tagsFromJSON(rev.TagsJSON),
			Status:    rev.Status,
			ActorId:   actorid.Format(ids.ActorID(rev.ActorID.Int64)),
			ActorRef:  rev.ActorRef,
			Summary:   rev.Summary,
			CreatedAt: timestamppb.New(timeFromMillis(rev.CreatedAt)),
		})
	}
	return out
}

func linkProto(link store.KnowledgeBitLinkRow) *agencyv1.KnowledgeLink {
	return &agencyv1.KnowledgeLink{
		KnowledgeId: link.LinkedID,
		Title:       link.Title,
		ProjectId:   link.ProjectID.String,
		Status:      link.Status,
		Kind:        link.Kind,
	}
}

func sourceInputs(items []*agencyv1.KnowledgeSourceInput) []knowledge.BitSourceInput {
	out := make([]knowledge.BitSourceInput, 0, len(items))
	for _, item := range items {
		out = append(out, knowledge.BitSourceInput{
			Kind:       item.Kind,
			TaskID:     item.TaskId,
			MessageID:  item.MessageId,
			RequestID:  item.RequestId,
			QuestionID: item.QuestionId,
			PromptID:   item.PromptId,
		})
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

func memoryFileProto(file knowledge.MemoryFile) *agencyv1.MemoryFile {
	return &agencyv1.MemoryFile{
		Path:      file.Path,
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

func projectOf(bit store.KnowledgeBitRow) string {
	if !bit.ProjectID.Valid {
		return ""
	}
	return bit.ProjectID.String
}

func tagsFromJSON(raw string) []string {
	var tags []string
	_ = jsonUnmarshal(raw, &tags)
	return tags
}

func jsonUnmarshal(raw string, out any) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), out)
}

func timestampOrNil(ms sql.NullInt64) *timestamppb.Timestamp {
	if !ms.Valid {
		return nil
	}
	return timestamppb.New(timeFromMillis(ms.Int64))
}

func timeFromMillis(ms int64) time.Time {
	return time.UnixMilli(ms).UTC()
}

func mapError(err error) *connect.Error {
	switch {
	case errors.Is(err, knowledge.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, knowledge.ErrForbidden),
		errors.Is(err, knowledge.ErrAgentMemoryOnly),
		errors.Is(err, knowledge.ErrReviewNotAllowed):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, knowledge.ErrProtected),
		errors.Is(err, knowledge.ErrOriginTaskRequired):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, knowledge.ErrConflict),
		errors.Is(err, store.ErrKnowledgeConflict):
		return connect.NewError(connect.CodeAborted, err)
	case errors.Is(err, knowledge.ErrInvalidPath),
		errors.Is(err, knowledge.ErrInvalidInput),
		errors.Is(err, knowledge.ErrInvalidScope),
		errors.Is(err, knowledge.ErrSourceInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:

		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "sql") {
			msg = "knowledge operation failed"
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("%s", msg))
	}
}
