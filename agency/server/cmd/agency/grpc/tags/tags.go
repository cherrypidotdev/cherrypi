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

// Package tagsgrpc implements agency.v1.TagService over the tags store.
package tagsgrpc

import (
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/tags"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
)

// TagServer implements agency.v1.TagService.
type TagServer struct {
	Tags *tags.Service
}

func (s *TagServer) ListTags(
	ctx context.Context,
	req *agencyv1.ListTagsRequest,
) (*agencyv1.ListTagsResponse, error) {
	list, err := s.Tags.List(ctx, req.ProjectId, req.IncludeArchived)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.Tag, len(list))
	for i, tag := range list {
		out[i] = toProto(tag)
	}
	return &agencyv1.ListTagsResponse{Tags: out}, nil
}

func (s *TagServer) CreateTag(
	ctx context.Context,
	req *agencyv1.CreateTagRequest,
) (*agencyv1.CreateTagResponse, error) {
	tag, err := s.Tags.Create(
		ctx,
		req.ProjectId,
		req.Name,
		req.Description,
		req.Color,
		req.IsDefault,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.CreateTagResponse{Tag: toProto(tag)}, nil
}

func (s *TagServer) UpdateTag(
	ctx context.Context,
	req *agencyv1.UpdateTagRequest,
) (*agencyv1.UpdateTagResponse, error) {
	tag, err := s.Tags.Update(
		ctx,
		req.Id,
		req.Name,
		req.Description,
		req.Color,
		req.IsDefault,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateTagResponse{Tag: toProto(tag)}, nil
}

func (s *TagServer) ArchiveTag(
	ctx context.Context,
	req *agencyv1.ArchiveTagRequest,
) (*agencyv1.ArchiveTagResponse, error) {
	tag, err := s.Tags.SetArchived(ctx, req.Id, true)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.ArchiveTagResponse{Tag: toProto(tag)}, nil
}

func (s *TagServer) RestoreTag(
	ctx context.Context,
	req *agencyv1.RestoreTagRequest,
) (*agencyv1.RestoreTagResponse, error) {
	tag, err := s.Tags.SetArchived(ctx, req.Id, false)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RestoreTagResponse{Tag: toProto(tag)}, nil
}

func toProto(tag tags.Tag) *agencyv1.Tag {
	return &agencyv1.Tag{
		Id:          tag.ID,
		ProjectId:   tag.ProjectID,
		Name:        tag.Name,
		Description: tag.Description,
		Color:       tag.Color,
		Archived:    tag.Archived,
		IsDefault:   tag.IsDefault,
		TaskCount:   int32(tag.TaskCount),
	}
}

func mapError(err error) *connect.Error {
	switch {
	case errors.Is(err, tags.ErrTagMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, tags.ErrInvalidName), errors.Is(err, tags.ErrProjectRequired):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, tags.ErrTagExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
