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

package tasktemplatesgrpc

import (
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/templates"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	Templates *templates.Service
}

func (s *Server) ListTaskTemplates(
	ctx context.Context,
	req *agencyv1.ListTaskTemplatesRequest,
) (*agencyv1.ListTaskTemplatesResponse, error) {
	list, err := s.Templates.List(ctx, req.ProjectId, req.IncludeArchived)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.TaskTemplate, len(list))
	for i, t := range list {
		out[i] = toProto(t)
	}
	return &agencyv1.ListTaskTemplatesResponse{Templates: out}, nil
}

func (s *Server) GetTaskTemplate(
	ctx context.Context,
	req *agencyv1.GetTaskTemplateRequest,
) (*agencyv1.GetTaskTemplateResponse, error) {
	t, err := s.Templates.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.GetTaskTemplateResponse{Template: toProto(t)}, nil
}

func (s *Server) CreateTaskTemplate(
	ctx context.Context,
	req *agencyv1.CreateTaskTemplateRequest,
) (*agencyv1.CreateTaskTemplateResponse, error) {
	t, validation, err := s.Templates.Create(
		ctx, req.ProjectId, req.Name, req.Description, contentFromProto(req.Content),
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.CreateTaskTemplateResponse{
		Template: toProto(t),
		Warnings: validation.Warnings,
	}, nil
}

func (s *Server) UpdateTaskTemplate(
	ctx context.Context,
	req *agencyv1.UpdateTaskTemplateRequest,
) (*agencyv1.UpdateTaskTemplateResponse, error) {
	var content *templates.Content
	if req.Content != nil {
		converted := contentFromProto(req.Content)
		content = &converted
	}
	t, versionCreated, validation, err := s.Templates.Update(
		ctx, req.Id, req.Name, req.Description, content,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateTaskTemplateResponse{
		Template:       toProto(t),
		VersionCreated: versionCreated,
		Warnings:       validation.Warnings,
	}, nil
}

func (s *Server) ArchiveTaskTemplate(
	ctx context.Context,
	req *agencyv1.ArchiveTaskTemplateRequest,
) (*agencyv1.ArchiveTaskTemplateResponse, error) {
	t, err := s.Templates.SetArchived(ctx, req.Id, true)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.ArchiveTaskTemplateResponse{Template: toProto(t)}, nil
}

func (s *Server) UnarchiveTaskTemplate(
	ctx context.Context,
	req *agencyv1.UnarchiveTaskTemplateRequest,
) (*agencyv1.UnarchiveTaskTemplateResponse, error) {
	t, err := s.Templates.SetArchived(ctx, req.Id, false)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UnarchiveTaskTemplateResponse{Template: toProto(t)}, nil
}

func (s *Server) ListTaskTemplateVersions(
	ctx context.Context,
	req *agencyv1.ListTaskTemplateVersionsRequest,
) (*agencyv1.ListTaskTemplateVersionsResponse, error) {
	versions, err := s.Templates.Versions(ctx, req.TemplateId)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.TaskTemplateVersion, len(versions))
	for i, v := range versions {
		out[i] = versionToProto(v)
	}
	return &agencyv1.ListTaskTemplateVersionsResponse{Versions: out}, nil
}

func (s *Server) RestoreTaskTemplateVersion(
	ctx context.Context,
	req *agencyv1.RestoreTaskTemplateVersionRequest,
) (*agencyv1.RestoreTaskTemplateVersionResponse, error) {
	t, created, err := s.Templates.RestoreVersion(ctx, req.TemplateId, req.Version)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RestoreTaskTemplateVersionResponse{
		Template:       toProto(t),
		CreatedVersion: created,
	}, nil
}

func (s *Server) ValidateTaskTemplate(
	ctx context.Context,
	req *agencyv1.ValidateTaskTemplateRequest,
) (*agencyv1.ValidateTaskTemplateResponse, error) {
	validation, err := s.Templates.Validate(ctx, req.ProjectId, contentFromProto(req.Content))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.ValidateTaskTemplateResponse{
		Valid:               validation.Valid(),
		Errors:              validation.Errors,
		Warnings:            validation.Warnings,
		UndeclaredVariables: validation.UndeclaredVariables,
		UnusedArguments:     validation.UnusedArguments,
	}, nil
}

func (s *Server) RenderTaskTemplate(
	ctx context.Context,
	req *agencyv1.RenderTaskTemplateRequest,
) (*agencyv1.RenderTaskTemplateResponse, error) {
	var (
		rendered templates.Rendered
		err      error
	)
	if req.Content != nil {
		rendered, err = s.Templates.RenderContent(
			ctx, req.ProjectId, contentFromProto(req.Content), req.ArgsJson,
		)
	} else {
		rendered, err = s.Templates.Render(ctx, req.TemplateId, req.Version, req.ArgsJson)
	}
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RenderTaskTemplateResponse{
		TemplateId:   rendered.TemplateID,
		TemplateName: rendered.TemplateName,
		Version:      rendered.Version,
		Prompt:       rendered.Prompt,
		Title:        rendered.Title,
		Priority:     priorityToProto(rendered.Priority),
		Size:         sizeToProto(rendered.Size),
		TagIds:       rendered.TagIDs,
		TeamIds:      int64sToInt32s(rendered.TeamIDs),
		LeadActorIds: actorIDStrings(rendered.LeadActorIDs),
		ArgsJson:     rendered.ArgsJSON,
	}, nil
}

func contentFromProto(in *agencyv1.TaskTemplateContent) templates.Content {
	if in == nil {
		return templates.Content{}
	}
	out := templates.Content{
		Body:                in.Body,
		TitleTemplate:       in.TitleTemplate,
		DefaultPriority:     priorityFromProto(in.DefaultPriority),
		DefaultSize:         sizeFromProto(in.DefaultSize),
		DefaultTagIDs:       in.DefaultTagIds,
		DefaultTeamIDs:      int32sToInt64s(in.DefaultTeamIds),
		DefaultLeadActorIDs: actorIDsFromStrings(in.DefaultLeadActorIds),
	}
	for _, arg := range in.Args {
		options := make([]templates.ArgOption, len(arg.Options))
		for i, option := range arg.Options {
			options[i] = templates.ArgOption{Value: option.Value, Label: option.Label}
		}
		out.Args = append(out.Args, templates.Arg{
			Key:          arg.Key,
			Label:        arg.Label,
			Description:  arg.Description,
			Type:         argTypeFromProto(arg.Type),
			Required:     arg.Required,
			DefaultValue: arg.DefaultValue,
			Options:      options,
		})
	}
	return out
}

func toProto(t templates.Template) *agencyv1.TaskTemplate {
	out := &agencyv1.TaskTemplate{
		Id:            t.ID,
		ProjectId:     t.ProjectID,
		Name:          t.Name,
		Description:   t.Description,
		CreatedAt:     timestamppb.New(t.CreatedAt),
		UpdatedAt:     timestamppb.New(t.UpdatedAt),
		LatestVersion: t.LatestVersion,
	}
	if !t.ArchivedAt.IsZero() {
		out.ArchivedAt = timestamppb.New(t.ArchivedAt)
	}
	if t.Latest != nil {
		out.Latest = versionToProto(*t.Latest)
	}
	return out
}

func versionToProto(v templates.Version) *agencyv1.TaskTemplateVersion {
	args := make([]*agencyv1.TaskTemplateArg, len(v.Args))
	for i, arg := range v.Args {
		options := make([]*agencyv1.TaskTemplateArgOption, len(arg.Options))
		for j, option := range arg.Options {
			options[j] = &agencyv1.TaskTemplateArgOption{Value: option.Value, Label: option.Label}
		}
		args[i] = &agencyv1.TaskTemplateArg{
			Key:          arg.Key,
			Label:        arg.Label,
			Description:  arg.Description,
			Type:         argTypeToProto(arg.Type),
			Required:     arg.Required,
			DefaultValue: arg.DefaultValue,
			Options:      options,
		}
	}
	out := &agencyv1.TaskTemplateVersion{
		Version: v.Version,
		Content: &agencyv1.TaskTemplateContent{
			Body:                v.Body,
			TitleTemplate:       v.TitleTemplate,
			Args:                args,
			DefaultPriority:     priorityToProto(v.DefaultPriority),
			DefaultSize:         sizeToProto(v.DefaultSize),
			DefaultTagIds:       v.DefaultTagIDs,
			DefaultTeamIds:      int64sToInt32s(v.DefaultTeamIDs),
			DefaultLeadActorIds: actorIDStrings(v.DefaultLeadActorIDs),
		},
		CreatedAt:        timestamppb.New(v.CreatedAt),
		CreatedByActorId: actorid.Format(v.CreatedByActorID),
	}
	return out
}

func argTypeFromProto(in agencyv1.TaskTemplateArgType) templates.ArgType {
	switch in {
	case agencyv1.TaskTemplateArgType_TASK_TEMPLATE_ARG_TYPE_MULTILINE:
		return templates.ArgMultiline
	case agencyv1.TaskTemplateArgType_TASK_TEMPLATE_ARG_TYPE_NUMBER:
		return templates.ArgNumber
	case agencyv1.TaskTemplateArgType_TASK_TEMPLATE_ARG_TYPE_BOOLEAN:
		return templates.ArgBoolean
	case agencyv1.TaskTemplateArgType_TASK_TEMPLATE_ARG_TYPE_SELECT:
		return templates.ArgSelect
	default:
		return templates.ArgText
	}
}

func argTypeToProto(in templates.ArgType) agencyv1.TaskTemplateArgType {
	switch in {
	case templates.ArgMultiline:
		return agencyv1.TaskTemplateArgType_TASK_TEMPLATE_ARG_TYPE_MULTILINE
	case templates.ArgNumber:
		return agencyv1.TaskTemplateArgType_TASK_TEMPLATE_ARG_TYPE_NUMBER
	case templates.ArgBoolean:
		return agencyv1.TaskTemplateArgType_TASK_TEMPLATE_ARG_TYPE_BOOLEAN
	case templates.ArgSelect:
		return agencyv1.TaskTemplateArgType_TASK_TEMPLATE_ARG_TYPE_SELECT
	default:
		return agencyv1.TaskTemplateArgType_TASK_TEMPLATE_ARG_TYPE_TEXT
	}
}

func priorityFromProto(in agencyv1.TaskPriority) int64 {
	switch in {
	case agencyv1.TaskPriority_TASK_PRIORITY_LOW:
		return tasks.PriorityLow
	case agencyv1.TaskPriority_TASK_PRIORITY_HIGH:
		return tasks.PriorityHigh
	case agencyv1.TaskPriority_TASK_PRIORITY_CRITICAL:
		return tasks.PriorityCritical
	default:
		return tasks.PriorityNormal
	}
}

func priorityToProto(in int64) agencyv1.TaskPriority {
	switch in {
	case tasks.PriorityLow:
		return agencyv1.TaskPriority_TASK_PRIORITY_LOW
	case tasks.PriorityHigh:
		return agencyv1.TaskPriority_TASK_PRIORITY_HIGH
	case tasks.PriorityCritical:
		return agencyv1.TaskPriority_TASK_PRIORITY_CRITICAL
	default:
		return agencyv1.TaskPriority_TASK_PRIORITY_NORMAL
	}
}

func sizeFromProto(in agencyv1.TaskSize) string {
	switch in {
	case agencyv1.TaskSize_TASK_SIZE_SMALL:
		return tasks.SizeSmall
	case agencyv1.TaskSize_TASK_SIZE_BIG:
		return tasks.SizeBig
	case agencyv1.TaskSize_TASK_SIZE_EPIC:
		return tasks.SizeEpic
	default:
		return tasks.SizeMedium
	}
}

func sizeToProto(in string) agencyv1.TaskSize {
	switch in {
	case tasks.SizeSmall:
		return agencyv1.TaskSize_TASK_SIZE_SMALL
	case tasks.SizeBig:
		return agencyv1.TaskSize_TASK_SIZE_BIG
	case tasks.SizeEpic:
		return agencyv1.TaskSize_TASK_SIZE_EPIC
	default:
		return agencyv1.TaskSize_TASK_SIZE_MEDIUM
	}
}

func int32sToInt64s(in []int32) []int64 {
	out := make([]int64, len(in))
	for i, value := range in {
		out[i] = int64(value)
	}
	return out
}

func int64sToInt32s(in []int64) []int32 {
	out := make([]int32, len(in))
	for i, value := range in {
		out[i] = int32(value)
	}
	return out
}

func actorIDsFromStrings(in []string) []ids.ActorID {
	out := make([]ids.ActorID, 0, len(in))
	for _, value := range in {
		if id := actorid.Parse(value); id != 0 {
			out = append(out, ids.ActorID(id))
		}
	}
	return out
}

func actorIDStrings(in []ids.ActorID) []string {
	out := make([]string, len(in))
	for i, id := range in {
		out[i] = actorid.Format(id)
	}
	return out
}

func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, templates.ErrTemplateMissing),
		errors.Is(err, templates.ErrVersionMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, templates.ErrNameTaken):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, templates.ErrArchived):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, templates.ErrVersionConflict):
		return connect.NewError(connect.CodeAborted, err)
	case errors.Is(err, projects.ErrProjectMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, templates.ErrNameRequired),
		errors.Is(err, templates.ErrBodyRequired),
		errors.Is(err, templates.ErrInvalidContent),
		errors.Is(err, templates.ErrProjectMismatch):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, projects.ErrForbidden),
		errors.Is(err, projects.ErrNotMember):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
