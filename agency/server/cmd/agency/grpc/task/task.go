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

package taskgrpc

import (
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/artifacts"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/integrations"
	"cherrypi.vladfaust.com/agency/internal/plugins"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/scheduler"
	"cherrypi.vladfaust.com/agency/internal/tags"
	"cherrypi.vladfaust.com/agency/internal/taskhistory"
	"cherrypi.vladfaust.com/agency/internal/taskinput"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/templates"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	agentv1connect "cherrypi.vladfaust.com/shared/gen/buf/agent/v1/agentv1connect"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AgentConnector interface {
	Connection(agentID ids.AgentID) (*tunnel.Connection, error)
}

type ActorInfo struct {
	Username string
	Name     string
	Kind     string

	PictureURL string
}

type TaskServer struct {
	Tasks       *tasks.Service
	Tags        *tags.Service
	Agents      *agents.Registry
	Hub         AgentConnector
	Scheduler   *scheduler.Scheduler
	Attachments *attachments.Service
	Artifacts   *artifacts.Service

	Templates *templates.Service

	Actors func(ctx context.Context, id ids.ActorID) (ActorInfo, error)

	ActorsByIDs func(ctx context.Context, ids []ids.ActorID) (map[ids.ActorID]ActorInfo, error)

	TeamName func(ctx context.Context, id int64) (string, error)

	TeamNamesByIDs func(ctx context.Context, ids []int64) (map[int64]string, error)

	Plugins *plugins.Service

	Integrations *integrations.Service

	Input *taskinput.Service

	EnrichTimeout time.Duration

	SettleGrace time.Duration

	toolCallMu    sync.Mutex
	toolCallCache map[string]cachedToolCalls

	toolLoadoutCache map[string]cachedToolLoadout
}

type cachedToolLoadout struct {
	source   string
	tools    []taskhistory.Tool
	recorded bool
}

type pluginToolOwner struct {
	ID   string
	Name string
}

type cachedToolCalls struct {
	tip   string
	calls []taskhistory.ToolCall
}

const maxToolCallCache = 64

const defaultEnrichTimeout = 2 * time.Second

func (s *TaskServer) enrichTimeout() time.Duration {
	if s.EnrichTimeout > 0 {
		return s.EnrichTimeout
	}
	return defaultEnrichTimeout
}

func (s *TaskServer) ListTasks(
	ctx context.Context,
	req *agencyv1.ListTasksRequest,
) (*agencyv1.ListTasksResponse, error) {
	var priority *int64
	if req.Priority != agencyv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED {
		p := priorityFromProto(req.Priority)
		priority = &p
	}
	filter := tasks.Filter{
		ProjectID:          req.ProjectId,
		AgentID:            actorid.ParseAgent(req.AgentId),
		State:              stateFromProto(req.State),
		TagID:              req.TagId,
		Search:             req.Search,
		Size:               sizeFromProto(req.Size),
		Priority:           priority,
		Archived:           req.Archived,
		IncludeArchived:    req.IncludeArchived,
		Automated:          req.Automated,
		ReferenceProjectID: req.ReferenceProjectId,
		Limit:              int64(req.Limit),
		Offset:             int64(req.Offset),
	}

	if actorID, ok := auth.ActorID(ctx); ok && actorID != 0 {
		filter.VisibleToActorID = actorID
		filter.VisibleToAdmin = auth.IsAdmin(ctx)
	}
	list, err := s.Tasks.List(ctx, filter)
	if err != nil {
		return nil, mapError(err)
	}
	s.enrichNames(ctx, list)
	out := make([]*agencyv1.Task, len(list))
	for i, t := range list {
		out[i] = toProtoTask(t)
		out[i].CanCloseOut = s.canCloseOut(ctx, t)
		out[i].CanAnswerInput = s.canAnswerInput(ctx, t)
		out[i].CanPrompt = t.CanPrompt
	}
	stripProgressItems(out)

	resolvers := s.attributionResolvers(ctx)
	resolvers.prefetch(out)
	for _, task := range out {
		s.enrichAttributionWith(resolvers, task)
	}
	s.enrichRuntime(ctx, list, out)
	s.enrichWaitingReasons(ctx, list, out)
	s.enrichBadges(ctx, out)
	s.enrichHoldIcons(ctx, out)
	return &agencyv1.ListTasksResponse{Tasks: out}, nil
}

func (s *TaskServer) visibleTask(ctx context.Context, id string) (tasks.Task, error) {
	t, err := s.Tasks.Get(ctx, id)
	if err != nil {
		return tasks.Task{}, mapError(err)
	}
	if actorID, ok := auth.ActorID(ctx); ok && actorID != 0 && !auth.IsAdmin(ctx) {
		if t.AssigneeActorID == actorID || t.CreatorActorID == actorID {
			return t, nil
		}
		allowed, err := s.Tasks.CanView(ctx, t.ProjectID, actorID, false)
		if err != nil {
			return tasks.Task{}, mapError(err)
		}
		if !allowed {
			return tasks.Task{}, connect.NewError(connect.CodeNotFound, tasks.ErrNotFound)
		}
	}
	return t, nil
}

func (s *TaskServer) GetTask(
	ctx context.Context,
	req *agencyv1.GetTaskRequest,
) (*agencyv1.GetTaskResponse, error) {
	t, err := s.visibleTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := s.protoTask(ctx, t)
	if at, ok := s.runtime(ctx, t); ok {
		rt := &agencyv1.TaskRuntime{
			RunInProgress:   at.GetRunInProgress(),
			WaitingForInput: at.GetWaitingForInput(),
			ProcessRunning:  at.GetProcessRunning(),
		}
		out.Runtime = rt

		if state, changed := s.materializeRuntime(ctx, t, at); changed {
			out.State = state
			if refreshed, err := s.Tasks.Get(ctx, t.ID); err == nil {
				out = s.protoTask(ctx, refreshed)
				out.Runtime = rt
			}
		}
	}
	return &agencyv1.GetTaskResponse{Task: out}, nil
}

func (s *TaskServer) ProtoTask(ctx context.Context, id string) *agencyv1.Task {
	t, err := s.visibleTask(ctx, id)
	if err != nil {
		return nil
	}

	enrichCtx, cancel := context.WithTimeout(ctx, s.enrichTimeout())
	defer cancel()
	out := s.protoTask(enrichCtx, t)
	s.enrichRuntime(enrichCtx, []tasks.Task{t}, []*agencyv1.Task{out})
	return out
}

func (s *TaskServer) protoTask(ctx context.Context, t tasks.Task) *agencyv1.Task {
	list := []tasks.Task{t}
	s.enrichNames(ctx, list)
	t = list[0]
	out := toProtoTask(t)
	out.CanCloseOut = s.canCloseOut(ctx, t)
	out.CanAnswerInput = s.canAnswerInput(ctx, t)
	out.CanPrompt = s.canPrompt(ctx, t)
	s.enrichWaitingReasons(ctx, list, []*agencyv1.Task{out})
	s.enrichAttachments(ctx, t.ID, t.State, out)
	s.enrichAttribution(ctx, out)
	s.enrichBadges(ctx, []*agencyv1.Task{out})
	s.enrichHoldIcons(ctx, []*agencyv1.Task{out})
	out.OriginChain = s.originChain(ctx, t)
	return out
}

func (s *TaskServer) originChain(
	ctx context.Context,
	t tasks.Task,
) []*agencyv1.TaskReference {
	chain, err := s.Tasks.OriginChain(ctx, t.ID)
	if err != nil {

		// timeout), not a fault worth a line per pushed row.
		if !errors.Is(err, context.Canceled) &&
			!errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[task %s] origin chain: %v", t.ID, err)
		}
		return nil
	}
	actorID, signedIn := auth.ActorID(ctx)
	admin := auth.IsAdmin(ctx)
	out := make([]*agencyv1.TaskReference, 0, len(chain))
	for _, ancestor := range chain {
		ref := &agencyv1.TaskReference{Id: ancestor.ID}
		visible := true
		if signedIn && actorID != 0 && !admin {
			allowed, err := s.Tasks.CanView(ctx, ancestor.ProjectID, actorID, false)
			if err != nil {
				log.Printf("[task %s] origin visibility %s: %v", t.ID, ancestor.ID, err)
				visible = false
			} else {
				visible = allowed
			}
		}
		if visible {
			ref.Name = ancestor.Name
			ref.State = stateToProto(ancestor.State)
		}
		out = append(out, ref)
	}
	return out
}

func (s *TaskServer) enrichBadges(ctx context.Context, tasks []*agencyv1.Task) {
	if s.Plugins == nil || len(tasks) == 0 {
		return
	}
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if task != nil && task.Id != "" {
			ids = append(ids, task.Id)
		}
	}
	if len(ids) == 0 {
		return
	}
	byTask := s.Plugins.TaskBadgesFor(ctx, ids)
	for _, task := range tasks {
		badges := byTask[task.Id]
		if len(badges) == 0 {
			continue
		}
		task.Badges = make([]*agencyv1.TaskBadge, len(badges))
		for i, badge := range badges {
			task.Badges[i] = &agencyv1.TaskBadge{
				PluginId:      badge.PluginID,
				PluginName:    badge.PluginName,
				PluginIconUrl: badge.PluginIconURL,
				Key:           badge.Key,
				Label:         badge.Label,
				Value:         badge.Value,
				Tone:          badge.Tone,
				Color:         badge.Color,
				Url:           badge.URL,
			}
		}
	}
}

func (s *TaskServer) enrichHoldIcons(ctx context.Context, tasks []*agencyv1.Task) {
	if s.Plugins == nil || len(tasks) == 0 {
		return
	}
	icons := map[string]string{}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		for _, hold := range task.Holds {
			if hold == nil || hold.Plugin == "" {
				continue
			}
			icon, ok := icons[hold.Plugin]
			if !ok {
				icon = s.Plugins.PluginIconURL(ctx, hold.Plugin)
				icons[hold.Plugin] = icon
			}
			hold.PluginIconUrl = icon
		}
	}
}

func (s *TaskServer) enrichAttachments(
	ctx context.Context,
	taskID string,
	state tasks.State,
	out *agencyv1.Task,
) {
	if s.Attachments == nil {
		return
	}

	if list, err := s.Attachments.TaskAttachments(ctx, taskID); err == nil {
		out.Attachments = make([]*agencyv1.Attachment, len(list))
		for i, a := range list {
			out.Attachments[i] = attachments.ToProto(a)
		}
	}

	messages, err := s.Attachments.TaskMessages(ctx, taskID)
	if err != nil {
		return
	}
	out.Messages = make([]*agencyv1.TaskMessage, len(messages))
	for i, message := range messages {
		tm := &agencyv1.TaskMessage{
			Id:        message.ID,
			Text:      message.Text,
			ActorId:   actorid.Format(message.ActorID),
			CreatedAt: timestamppb.New(message.CreatedAt),
		}
		for _, a := range message.Attachments {
			tm.Attachments = append(tm.Attachments, attachments.ToProto(a))
		}
		out.Messages[i] = tm
	}

	s.enrichPendingMessages(ctx, taskID, state, out)

	if s.Artifacts != nil {
		list, err := s.Artifacts.List(ctx, taskID)
		if err != nil {
			return
		}
		out.Artifacts = make([]*agencyv1.Artifact, len(list))
		for i, artifact := range list {
			out.Artifacts[i] = artifactToProto(artifact)
		}
	}

	logs, err := s.Tasks.Logs(ctx, taskID)
	if err != nil {
		return
	}
	out.Logs = make([]*agencyv1.TaskLog, len(logs))
	for i, entry := range logs {
		out.Logs[i] = toProtoLog(entry)
	}
}

func (s *TaskServer) enrichAttribution(ctx context.Context, out *agencyv1.Task) {
	s.enrichAttributionWith(s.attributionResolvers(ctx), out)
}

type attributionResolvers struct {
	ctx       context.Context
	server    *TaskServer
	actors    map[ids.ActorID]ActorInfo
	teamNames map[int64]string
}

func (s *TaskServer) attributionResolvers(ctx context.Context) *attributionResolvers {
	return &attributionResolvers{
		ctx:       ctx,
		server:    s,
		actors:    map[ids.ActorID]ActorInfo{},
		teamNames: map[int64]string{},
	}
}

func (r *attributionResolvers) actor(raw string) (ActorInfo, bool) {
	id := actorid.ParseActor(raw)
	if id == 0 || r.server.Actors == nil {
		return ActorInfo{}, false
	}
	if info, ok := r.actors[id]; ok {
		return info, true
	}
	info, err := r.server.Actors(r.ctx, id)
	if err != nil {
		return ActorInfo{}, false
	}
	r.actors[id] = info
	return info, true
}

func (r *attributionResolvers) teamName(id int64) (string, bool) {
	if r.server.TeamName == nil {
		return "", false
	}
	if name, ok := r.teamNames[id]; ok {
		return name, true
	}
	name, err := r.server.TeamName(r.ctx, id)
	if err != nil {
		return "", false
	}
	r.teamNames[id] = name
	return name, true
}

func (r *attributionResolvers) prefetch(page []*agencyv1.Task) {
	actorIDs := map[ids.ActorID]struct{}{}
	teamIDs := map[int64]struct{}{}
	addActor := func(raw string) {
		if id := actorid.ParseActor(raw); id != 0 {
			actorIDs[id] = struct{}{}
		}
	}
	for _, task := range page {
		addActor(task.CreatorActorId)
		addActor(task.PendingPromptActorId)
		if task.Leader != nil {
			addActor(task.Leader.ActorId)
		}
		for _, message := range task.Messages {
			addActor(message.ActorId)
		}
		for _, message := range task.PendingMessages {
			addActor(message.ActorId)
		}
		for _, team := range task.Teams {
			teamIDs[int64(team.Id)] = struct{}{}
		}
	}
	if r.server.ActorsByIDs != nil && len(actorIDs) > 0 {
		list := make([]ids.ActorID, 0, len(actorIDs))
		for id := range actorIDs {
			list = append(list, id)
		}
		if found, err := r.server.ActorsByIDs(r.ctx, list); err == nil {
			for id, info := range found {
				r.actors[id] = info
			}
		}
	}
	if r.server.TeamNamesByIDs != nil && len(teamIDs) > 0 {
		list := make([]int64, 0, len(teamIDs))
		for id := range teamIDs {
			list = append(list, id)
		}
		if found, err := r.server.TeamNamesByIDs(r.ctx, list); err == nil {
			for id, name := range found {
				r.teamNames[id] = name
			}
		}
	}
}

func (s *TaskServer) enrichAttributionWith(
	resolvers *attributionResolvers,
	out *agencyv1.Task,
) {
	actor, teamName := resolvers.actor, resolvers.teamName
	if info, ok := actor(out.CreatorActorId); ok {
		out.CreatorName = displayActorName(info)
		out.CreatorPictureUrl = info.PictureURL
		out.CreatorUsername = info.Username
		out.CreatorKind = info.Kind
	}
	if info, ok := actor(out.PendingPromptActorId); ok {
		out.PendingPromptActorName = displayActorName(info)
	}
	for _, team := range out.Teams {
		if name, ok := teamName(int64(team.Id)); ok {
			team.Name = name
		}
	}
	if out.Leader != nil {
		if info, ok := actor(out.Leader.ActorId); ok {
			out.Leader.Username = info.Username
			out.Leader.Name = displayActorName(info)
			out.Leader.Kind = info.Kind
			out.Leader.PictureUrl = info.PictureURL
		}
	}
	for _, message := range out.Messages {
		if info, ok := actor(message.ActorId); ok {
			message.ActorName = displayActorName(info)
			message.ActorUsername = info.Username
			message.ActorKind = info.Kind
		}
	}
	for _, message := range out.PendingMessages {
		if info, ok := actor(message.ActorId); ok {
			message.ActorName = displayActorName(info)
			message.ActorUsername = info.Username
			message.ActorKind = info.Kind
		}
	}
	for _, entry := range out.Logs {
		if info, ok := actor(entry.ActorId); ok {
			entry.ActorName = displayActorName(info)
			entry.ActorUsername = info.Username
		}
	}
}

func displayActorName(info ActorInfo) string {
	if info.Name != "" {
		return info.Name
	}
	return info.Username
}
func artifactToProto(a artifacts.Artifact) *agencyv1.Artifact {
	out := &agencyv1.Artifact{
		ArtifactId:  a.ID,
		Name:        a.Name,
		Description: a.Description,
		Streaming:   a.Streaming,
		Metadata:    metadataStruct(a.MetadataJSON),
		CreatedAt:   timestamppb.New(a.CreatedAt),
		UpdatedAt:   timestamppb.New(a.UpdatedAt),
	}
	for _, p := range a.Parts {
		part := &agencyv1.ArtifactPart{
			Filename:  p.Filename,
			MediaType: p.MediaType,
			Metadata:  metadataStruct(p.MetadataJSON),
		}
		switch p.Kind {
		case artifacts.KindFile:
			file := &agencyv1.ArtifactFile{AttachmentId: p.AttachmentID, Raw: p.AttachmentID > 0}
			if p.AttachmentID > 0 {
				file.Url = attachments.URLFor(p.AttachmentID)
			} else {
				file.Url = p.URL
			}
			part.Content = &agencyv1.ArtifactPart_File{File: file}
		case artifacts.KindData:
			part.Content = &agencyv1.ArtifactPart_DataJson{DataJson: p.DataJSON}
		default:
			part.Content = &agencyv1.ArtifactPart_Text{Text: p.Text}
		}
		out.Parts = append(out.Parts, part)
	}
	return out
}

func metadataStruct(raw string) *structpb.Struct {
	if raw == "" {
		return nil
	}
	var m map[string]any
	if json.Unmarshal([]byte(raw), &m) != nil {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}

func (s *TaskServer) CreateTask(
	ctx context.Context,
	req *agencyv1.CreateTaskRequest,
) (*agencyv1.CreateTaskResponse, error) {
	if err := s.checkAssignee(ctx, actorid.ParseActor(req.AssigneeActorId)); err != nil {
		return nil, err
	}

	if actorID, ok := auth.ActorID(ctx); ok && actorID != 0 && !auth.IsAdmin(ctx) {
		allowed, err := s.Tasks.CanCreateInProject(ctx, req.ProjectId, actorID)
		if err != nil {
			return nil, mapError(err)
		}
		if !allowed {
			return nil, connect.NewError(
				connect.CodePermissionDenied,
				errors.New("not allowed to create tasks in this project"),
			)
		}
	}

	if origin := strings.TrimSpace(req.OriginTaskId); origin != "" {
		if actorID, ok := auth.ActorID(ctx); ok && actorID != 0 && !auth.IsAdmin(ctx) {
			visible, err := s.Tasks.OriginVisible(ctx, origin, actorID, false)
			if err != nil {
				return nil, mapError(err)
			}
			if !visible {
				return nil, connect.NewError(connect.CodeNotFound, tasks.ErrNotFound)
			}
		}
	}
	params := tasks.CreateParams{
		Prompt:          req.Prompt,
		Name:            req.Name,
		ProjectID:       req.ProjectId,
		AssigneeActorID: actorid.ParseActor(req.AssigneeActorId),
		TagIDs:          req.TagIds,
		References:      referenceInputsFromProto(req.References),
		Priority:        priorityFromProto(req.Priority),
		Size:            sizeFromProto(req.Size),
		ModelProvider:   req.ModelProvider,
		ModelID:         req.ModelId,
		ThinkingLevel:   req.ThinkingLevel,
		Automated:       req.Automated != nil && req.GetAutomated(),
		Workspace:       req.Workspace,
		AutoTitle:       req.AutoTitle,
		State:           stateFromProto(req.State),
		TeamIDs:         int32sToInt64s(req.TeamIds),
		LeaderActorID:   actorid.ParseActor(req.LeaderActorId),
		OriginTaskID:    req.OriginTaskId,
	}
	if req.TemplateId != "" {
		if s.Templates == nil {
			return nil, connect.NewError(
				connect.CodeFailedPrecondition,
				errors.New("task templates are not available"),
			)
		}
		rendered, err := s.Templates.RenderForProject(
			ctx,
			req.ProjectId,
			req.TemplateId,
			req.TemplateVersion,
			req.TemplateArgsJson,
		)
		if err != nil {
			return nil, mapTemplateError(err)
		}
		params.Prompt = rendered.Prompt
		if req.Prompt != "" {
			if req.TemplatePromptOverride {
				params.Prompt = req.Prompt
			} else {
				params.Prompt = rendered.Prompt + "\n\n" + req.Prompt
			}
		}
		if req.Name == "" {
			params.Name = rendered.Title
		}
		if req.Priority == agencyv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED {
			params.Priority = rendered.Priority
		}
		if req.Size == agencyv1.TaskSize_TASK_SIZE_UNSPECIFIED {
			params.Size = rendered.Size
		}
		if len(req.TagIds) == 0 {
			params.TagIDs = rendered.TagIDs
		}
		if len(req.TeamIds) == 0 {
			params.TeamIDs = rendered.TeamIDs
		}
		if req.LeaderActorId == "" {
			params.LeaderActorID = rendered.LeaderActorID
		}
		params.Template = &tasks.TemplateApplication{
			TemplateID:      rendered.TemplateID,
			TemplateName:    rendered.TemplateName,
			TemplateVersion: rendered.Version,
			ArgsJSON:        rendered.ArgsJSON,
		}
	}
	t, err := s.Tasks.Create(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}
	if t.State == tasks.StateQueued {
		s.wake()
	}
	return &agencyv1.CreateTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) UpdateTask(
	ctx context.Context,
	req *agencyv1.UpdateTaskRequest,
) (*agencyv1.UpdateTaskResponse, error) {
	if _, err := s.requireCanEdit(ctx, req.Id); err != nil {
		return nil, err
	}
	if attrs := attributesFromProto(
		req.SetPriority,
		req.SetSize,
		actorid.ParseActorPtr(req.SetAssigneeActorId),
		req.SetPrompt,
		req.SetPendingPrompt,
		req.SetModel,
		nil,
		nil,
	); !attrs.Empty() {
		if req.Prompt != "" || req.ProjectId != "" {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				errors.New(
					"an attribute update must not include the task prompt or project fields",
				),
			)
		}
		if err := s.checkAssignee(
			ctx,
			actorid.ParseActor(req.GetSetAssigneeActorId()),
		); err != nil {
			return nil, err
		}

		if attrs.Model != nil {
			if err := s.requireTaskModelChange(
				ctx,
				req.Id,
				attrs.Model.Provider,
				attrs.Model.ModelID,
				attrs.Model.ThinkingLevel,
			); err != nil {
				return nil, err
			}
		}
		t, err := s.Tasks.UpdateAttributes(ctx, req.Id, attrs)
		if err != nil {
			return nil, mapError(err)
		}

		s.wake()
		return &agencyv1.UpdateTaskResponse{Task: s.protoTask(ctx, t)}, nil
	}
	if err := s.checkAssignee(ctx, actorid.ParseActor(req.AssigneeActorId)); err != nil {
		return nil, err
	}
	if err := s.requireTaskModelChange(
		ctx, req.Id, req.ModelProvider, req.ModelId, req.ThinkingLevel,
	); err != nil {
		return nil, err
	}
	t, err := s.Tasks.Update(ctx, req.Id, tasks.UpdateParams{
		Prompt:          req.Prompt,
		Name:            req.Name,
		ProjectID:       req.ProjectId,
		AssigneeActorID: actorid.ParseActor(req.AssigneeActorId),
		TagIDs:          req.TagIds,
		Priority:        priorityFromProto(req.Priority),
		Size:            sizeFromProto(req.Size),
		ModelProvider:   req.ModelProvider,
		ModelID:         req.ModelId,
		ThinkingLevel:   req.ThinkingLevel,
		Workspace:       req.Workspace,
		AutoTitle:       req.AutoTitle,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) QueueTask(
	ctx context.Context,
	req *agencyv1.QueueTaskRequest,
) (*agencyv1.QueueTaskResponse, error) {
	if _, err := s.requireCanEdit(ctx, req.Id); err != nil {
		return nil, err
	}
	t, err := s.Tasks.Queue(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	s.wake()
	return &agencyv1.QueueTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) ListTaskReferences(
	ctx context.Context,
	req *agencyv1.ListTaskReferencesRequest,
) (*agencyv1.ListTaskReferencesResponse, error) {
	if _, err := s.visibleTask(ctx, req.Id); err != nil {
		return nil, err
	}
	refs, err := s.Tasks.References(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	s.enrichNames(ctx, refs)
	out := make([]*agencyv1.Task, len(refs))
	for i, t := range refs {
		out[i] = toProtoTask(t)
	}
	stripProgressItems(out)
	s.enrichRuntime(ctx, refs, out)
	return &agencyv1.ListTaskReferencesResponse{Tasks: out}, nil
}

func (s *TaskServer) AddTaskReference(
	ctx context.Context,
	req *agencyv1.AddTaskReferenceRequest,
) (*agencyv1.AddTaskReferenceResponse, error) {
	if _, err := s.requireCanEdit(ctx, req.Id); err != nil {
		return nil, err
	}
	t, err := s.Tasks.AddReference(
		ctx,
		req.Id,
		req.ReferencedTaskId,
		referenceKindFromProto(req.Kind),
	)
	if err != nil {
		return nil, mapError(err)
	}

	s.wake()
	return &agencyv1.AddTaskReferenceResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) RemoveTaskReference(
	ctx context.Context,
	req *agencyv1.RemoveTaskReferenceRequest,
) (*agencyv1.RemoveTaskReferenceResponse, error) {
	if _, err := s.requireCanEdit(ctx, req.Id); err != nil {
		return nil, err
	}
	remove := s.Tasks.RemoveReference
	if req.GetReverse() {
		remove = s.Tasks.RemoveIncomingReference
	}
	t, err := remove(ctx, req.Id, req.ReferencedTaskId)
	if err != nil {
		return nil, mapError(err)
	}

	s.wake()
	return &agencyv1.RemoveTaskReferenceResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) DequeueTask(
	ctx context.Context,
	req *agencyv1.DequeueTaskRequest,
) (*agencyv1.DequeueTaskResponse, error) {
	t, err := s.Tasks.Dequeue(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DequeueTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) CloseOutTask(
	ctx context.Context,
	req *agencyv1.CloseOutTaskRequest,
) (*agencyv1.CloseOutTaskResponse, error) {
	cur, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if err := s.requireCanCloseOut(ctx, cur); err != nil {
		return nil, err
	}
	var t tasks.Task
	switch req.Outcome {
	case agencyv1.TaskState_TASK_STATE_COMPLETED:
		t, err = s.Tasks.Complete(ctx, req.Id)
	case agencyv1.TaskState_TASK_STATE_CANCELLED:
		t, err = s.Tasks.Cancel(ctx, req.Id)
	default:
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("outcome must be completed or cancelled"),
		)
	}
	if err != nil {
		return nil, mapError(err)
	}

	s.wake()
	return &agencyv1.CloseOutTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) ReopenTask(
	ctx context.Context,
	req *agencyv1.ReopenTaskRequest,
) (*agencyv1.ReopenTaskResponse, error) {
	t, err := s.Tasks.Reopen(ctx, req.Id, req.Queue)
	if err != nil {
		return nil, mapError(err)
	}
	if t.State == tasks.StateQueued {
		s.wake()
	}
	return &agencyv1.ReopenTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) ReleaseTaskHold(
	ctx context.Context,
	req *agencyv1.ReleaseTaskHoldRequest,
) (*agencyv1.ReleaseTaskHoldResponse, error) {
	cur, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}

	if err := s.requireCanCloseOut(ctx, cur); err != nil {
		return nil, err
	}
	actorID, _ := auth.ActorID(ctx)
	t, err := s.Tasks.ReleaseHold(ctx, req.Id, req.Plugin, req.Key, tasks.ReleaseOptions{
		Mode:    holdReleaseModeFromProto(req.Mode),
		Prompt:  req.Prompt,
		ActorID: actorID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	if t.State == tasks.StateQueued {
		s.wake()
	}
	return &agencyv1.ReleaseTaskHoldResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) DeleteTask(
	ctx context.Context,
	req *agencyv1.DeleteTaskRequest,
) (*agencyv1.DeleteTaskResponse, error) {
	cur, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}

	if err := s.cleanupTaskSession(ctx, cur); err != nil {
		log.Printf("[task %s] delete agent session: %v", cur.ID, err)
	}

	if err := s.Tasks.DeleteSessionData(ctx, req.Id); err != nil {
		return nil, mapError(err)
	}
	if err := s.Tasks.Delete(ctx, req.Id); err != nil {
		return nil, mapError(err)
	}
	s.wake()
	return &agencyv1.DeleteTaskResponse{}, nil
}

func (s *TaskServer) MergeTasks(
	ctx context.Context,
	req *agencyv1.MergeTasksRequest,
) (*agencyv1.MergeTasksResponse, error) {
	t, err := s.Tasks.Merge(ctx, req.Ids)
	if err != nil {
		return nil, mapError(err)
	}

	s.wake()
	return &agencyv1.MergeTasksResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) ArchiveTask(
	ctx context.Context,
	req *agencyv1.ArchiveTaskRequest,
) (*agencyv1.ArchiveTaskResponse, error) {
	cur, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if err := s.requireCanCloseOut(ctx, cur); err != nil {
		return nil, err
	}
	t, err := s.Tasks.Archive(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if err := s.cleanupTaskSession(ctx, t); err != nil {
		log.Printf("[task %s] archive cleanup deferred: %v", t.ID, err)
	}
	if refreshed, err := s.Tasks.Get(ctx, t.ID); err == nil {
		t = refreshed
	}
	s.wake()
	return &agencyv1.ArchiveTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) UnarchiveTask(
	ctx context.Context,
	req *agencyv1.UnarchiveTaskRequest,
) (*agencyv1.UnarchiveTaskResponse, error) {
	cur, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if err := s.requireCanCloseOut(ctx, cur); err != nil {
		return nil, err
	}
	t, err := s.Tasks.Unarchive(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	s.wake()
	return &agencyv1.UnarchiveTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) CleanupTaskSession(
	ctx context.Context,
	req *agencyv1.CleanupTaskSessionRequest,
) (*agencyv1.CleanupTaskSessionResponse, error) {
	t, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if !t.Archived {
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			errors.New("task is not archived"),
		)
	}
	if err := s.cleanupTaskSession(ctx, t); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if refreshed, err := s.Tasks.Get(ctx, t.ID); err == nil {
		t = refreshed
	}
	return &agencyv1.CleanupTaskSessionResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) cleanupTaskSession(ctx context.Context, t tasks.Task) error {
	return s.Tasks.CleanupSession(ctx, t.ID)
}

func (s *TaskServer) RenameTask(
	ctx context.Context,
	req *agencyv1.RenameTaskRequest,
) (*agencyv1.RenameTaskResponse, error) {
	if _, err := s.requireCanEdit(ctx, req.Id); err != nil {
		return nil, err
	}
	t, err := s.Tasks.Rename(ctx, req.Id, req.Name)
	if err != nil {
		return nil, mapError(err)
	}
	if (t.State == tasks.StateRunning || t.State == tasks.StateSettled) &&
		t.SessionID != "" {
		if conn, err := s.connectionForActor(ctx, t.AssigneeActorID); err == nil {
			if _, err := conn.Tasks.RenameTask(ctx, &agentv1.RenameTaskRequest{
				Id:   t.SessionID,
				Name: req.Name,
			}); err != nil && connect.CodeOf(err) != connect.CodeNotFound {
				log.Printf("[task %s] rename agent session: %v", t.ID, err)
			}
		}
	}
	return &agencyv1.RenameTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) SetTaskPriority(
	ctx context.Context,
	req *agencyv1.SetTaskPriorityRequest,
) (*agencyv1.SetTaskPriorityResponse, error) {
	if _, err := s.requireCanEdit(ctx, req.Id); err != nil {
		return nil, err
	}
	t, err := s.Tasks.SetPriority(ctx, req.Id, priorityFromProto(req.Priority))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.SetTaskPriorityResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) SetTaskSize(
	ctx context.Context,
	req *agencyv1.SetTaskSizeRequest,
) (*agencyv1.SetTaskSizeResponse, error) {
	if _, err := s.requireCanEdit(ctx, req.Id); err != nil {
		return nil, err
	}
	t, err := s.Tasks.SetSize(ctx, req.Id, sizeFromProto(req.Size))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.SetTaskSizeResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) SetTaskLinks(
	ctx context.Context,
	req *agencyv1.SetTaskLinksRequest,
) (*agencyv1.SetTaskLinksResponse, error) {
	links := make([]tasks.Link, len(req.GetLinks()))
	for i, link := range req.GetLinks() {
		links[i] = tasks.Link{URL: link.GetUrl(), Label: link.GetLabel()}
	}
	if _, err := s.requireCanEdit(ctx, req.Id); err != nil {
		return nil, err
	}
	t, err := s.Tasks.SetLinks(ctx, req.Id, links)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.SetTaskLinksResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) SetTaskTeams(
	ctx context.Context,
	req *agencyv1.SetTaskTeamsRequest,
) (*agencyv1.SetTaskTeamsResponse, error) {
	t, err := s.Tasks.SetTaskTeams(ctx, req.Id, int32sToInt64s(req.TeamIds))
	if err != nil {
		return nil, mapError(err)
	}
	out := s.protoTask(ctx, t)
	s.enrichAttribution(ctx, out)
	return &agencyv1.SetTaskTeamsResponse{Task: out}, nil
}

func (s *TaskServer) SetTaskLeader(
	ctx context.Context,
	req *agencyv1.SetTaskLeaderRequest,
) (*agencyv1.SetTaskLeaderResponse, error) {
	cur, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if err := s.requireCanPrompt(ctx, cur); err != nil {
		return nil, err
	}
	t, err := s.Tasks.SetTaskLeader(ctx, req.Id, actorid.ParseActor(req.LeaderActorId))
	if err != nil {
		return nil, mapError(err)
	}
	out := s.protoTask(ctx, t)
	s.enrichAttribution(ctx, out)
	return &agencyv1.SetTaskLeaderResponse{Task: out}, nil
}

func (s *TaskServer) SetTaskSubscription(
	ctx context.Context,
	req *agencyv1.SetTaskSubscriptionRequest,
) (*agencyv1.SetTaskSubscriptionResponse, error) {
	actorID, ok := auth.ActorID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, tasks.ErrActorRequired)
	}
	t, err := s.Tasks.SetSubscribed(ctx, req.Id, actorID, req.Subscribed)
	if err != nil {
		return nil, mapError(err)
	}
	out := s.protoTask(ctx, t)
	s.enrichAttribution(ctx, out)
	return &agencyv1.SetTaskSubscriptionResponse{Task: out}, nil
}

func (s *TaskServer) SetTaskFavorite(
	ctx context.Context,
	req *agencyv1.SetTaskFavoriteRequest,
) (*agencyv1.SetTaskFavoriteResponse, error) {
	actorID, ok := auth.ActorID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, tasks.ErrActorRequired)
	}
	t, err := s.Tasks.SetFavorite(ctx, req.Id, actorID, req.Favorite)
	if err != nil {
		return nil, mapError(err)
	}
	out := s.protoTask(ctx, t)
	s.enrichAttribution(ctx, out)
	return &agencyv1.SetTaskFavoriteResponse{Task: out}, nil
}

func (s *TaskServer) ListTaskComments(
	ctx context.Context,
	req *agencyv1.ListTaskCommentsRequest,
) (*agencyv1.ListTaskCommentsResponse, error) {
	t, err := s.visibleTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	actorID, ok := auth.ActorID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, tasks.ErrActorRequired)
	}
	list, unread, err := s.Tasks.ListComments(ctx, t.ID, actorID)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.ListTaskCommentsResponse{
		Comments: s.commentsToProto(ctx, t.ID, list),
		Unread:   int32(unread),
	}, nil
}

func (s *TaskServer) AddTaskComment(
	ctx context.Context,
	req *agencyv1.AddTaskCommentRequest,
) (*agencyv1.AddTaskCommentResponse, error) {
	t, err := s.visibleTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	actorID, ok := auth.ActorID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, tasks.ErrActorRequired)
	}
	if s.Attachments != nil {
		if max := s.Attachments.MaxPerMessage(); max > 0 && len(req.AttachmentIds) > max {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				fmt.Errorf("at most %d attachments per comment", max),
			)
		}
	}
	comment, err := s.Tasks.AddComment(ctx, t.ID, actorID, req.Body, req.AttachmentIds)
	if err != nil {
		return nil, mapError(err)
	}
	out := s.commentsToProto(ctx, t.ID, []tasks.Comment{comment})
	return &agencyv1.AddTaskCommentResponse{Comment: out[0]}, nil
}

func (s *TaskServer) UpdateTaskComment(
	ctx context.Context,
	req *agencyv1.UpdateTaskCommentRequest,
) (*agencyv1.UpdateTaskCommentResponse, error) {
	actorID, ok := auth.ActorID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, tasks.ErrActorRequired)
	}
	taskID, err := s.Tasks.CommentTaskID(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if _, err := s.visibleTask(ctx, taskID); err != nil {
		return nil, err
	}
	comment, err := s.Tasks.UpdateComment(
		ctx,
		req.Id,
		actorID,
		auth.IsAdmin(ctx),
		req.Body,
	)
	if err != nil {
		return nil, mapError(err)
	}
	out := s.commentsToProto(ctx, taskID, []tasks.Comment{comment})
	return &agencyv1.UpdateTaskCommentResponse{Comment: out[0]}, nil
}

func (s *TaskServer) DeleteTaskComment(
	ctx context.Context,
	req *agencyv1.DeleteTaskCommentRequest,
) (*agencyv1.DeleteTaskCommentResponse, error) {
	actorID, ok := auth.ActorID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, tasks.ErrActorRequired)
	}
	taskID, err := s.Tasks.CommentTaskID(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if _, err := s.visibleTask(ctx, taskID); err != nil {
		return nil, err
	}
	if _, err := s.Tasks.DeleteComment(ctx, req.Id, actorID, auth.IsAdmin(ctx)); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DeleteTaskCommentResponse{}, nil
}

func (s *TaskServer) commentsToProto(
	ctx context.Context,
	taskID string,
	list []tasks.Comment,
) []*agencyv1.TaskComment {
	var byComment map[int64][]attachments.Attachment
	if s.Attachments != nil {
		byComment, _ = s.Attachments.CommentAttachments(ctx, taskID)
	}
	actor := s.actorResolver(ctx)
	out := make([]*agencyv1.TaskComment, len(list))
	for i, comment := range list {
		item := &agencyv1.TaskComment{
			Id:               comment.ID,
			TaskId:           comment.TaskID,
			AuthorActorId:    actorid.Format(comment.AuthorActorID),
			Body:             comment.Body,
			Edited:           comment.Edited(),
			Deleted:          comment.Deleted(),
			DeletedByActorId: actorid.Format(comment.DeletedByID),
			CreatedAt:        timestamppb.New(comment.CreatedAt),
			UpdatedAt:        timestamppb.New(comment.UpdatedAt),
		}
		if info, ok := actor(item.AuthorActorId); ok {
			item.AuthorName = displayActorName(info)
			item.AuthorUsername = info.Username
			item.AuthorKind = info.Kind
			item.AuthorPictureUrl = info.PictureURL
		}
		if info, ok := actor(item.DeletedByActorId); ok {
			item.DeletedByName = displayActorName(info)
			item.DeletedByUsername = info.Username
		}
		if comment.Deleted() {
			item.DeletedAt = timestamppb.New(comment.DeletedAt)
		}
		for _, file := range byComment[comment.ID] {
			item.Attachments = append(item.Attachments, attachments.ToProto(file))
		}
		out[i] = item
	}
	return out
}

func (s *TaskServer) actorResolver(ctx context.Context) func(string) (ActorInfo, bool) {
	cache := map[ids.ActorID]ActorInfo{}
	return func(raw string) (ActorInfo, bool) {
		id := actorid.ParseActor(raw)
		if id == 0 || s.Actors == nil {
			return ActorInfo{}, false
		}
		if info, ok := cache[id]; ok {
			return info, true
		}
		info, err := s.Actors(ctx, id)
		if err != nil {
			return ActorInfo{}, false
		}
		cache[id] = info
		return info, true
	}
}

func (s *TaskServer) BulkUpdateTask(
	ctx context.Context,
	req *agencyv1.BulkUpdateTaskRequest,
) (*agencyv1.BulkUpdateTaskResponse, error) {
	attrs := attributesFromProto(
		req.SetPriority,
		req.SetSize,
		actorid.ParseActorPtr(req.SetAssigneeActorId),
		nil,
		nil,
		nil,
		req.GetAddTagIds(),
		req.GetRemoveTagIds(),
	)
	if attrs.Empty() {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("at least one attribute must be set"),
		)
	}
	if len(req.GetIds()) == 0 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("at least one task id is required"),
		)
	}
	if attrs.AssigneeActorID != nil && *attrs.AssigneeActorID != 0 {
		if err := s.checkAssignee(ctx, *attrs.AssigneeActorID); err != nil {
			return nil, err
		}
	}

	allowed := make([]string, 0, len(req.GetIds()))
	denied := make(map[string]string, len(req.GetIds()))
	for _, id := range req.GetIds() {
		if _, err := s.requireCanEdit(ctx, id); err != nil {
			denied[id] = "not allowed to edit this task"
			continue
		}
		allowed = append(allowed, id)
	}
	results := s.Tasks.UpdateManyAttributes(ctx, allowed, attrs)

	byID := make(map[string]tasks.AttributeResult, len(results))
	for _, result := range results {
		byID[result.ID] = result
	}
	merged := make([]tasks.AttributeResult, 0, len(req.GetIds()))
	for _, id := range req.GetIds() {
		if msg, ok := denied[id]; ok {
			merged = append(merged, tasks.AttributeResult{
				ID:    id,
				Error: errors.New(msg),
			})
			continue
		}
		if result, ok := byID[id]; ok {
			merged = append(merged, result)
		}
	}
	results = merged
	out := make([]*agencyv1.BulkUpdateTaskResult, 0, len(results))
	for _, r := range results {
		item := &agencyv1.BulkUpdateTaskResult{Id: r.ID}
		if r.Error != nil {
			item.Error = r.Error.Error()
		} else {
			item.Task = s.protoTask(ctx, r.Task)
		}
		out = append(out, item)
	}

	s.wake()
	return &agencyv1.BulkUpdateTaskResponse{Results: out}, nil
}

func (s *TaskServer) AddToTask(
	ctx context.Context,
	req *agencyv1.AddToTaskRequest,
) (*agencyv1.AddToTaskResponse, error) {
	t, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if t.State == tasks.StateInput {

		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			errors.New("task is waiting for input; answer the dialog first"),
		)
	}

	actorID, _ := auth.ActorID(ctx)
	allowed, err := s.Tasks.CanPrompt(ctx, t, actorID, auth.IsAdmin(ctx))
	if err != nil {
		return nil, mapError(err)
	}
	if !allowed {
		return nil, connect.NewError(connect.CodePermissionDenied, tasks.ErrCannotPrompt)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, tasks.ErrInvalidPrompt)
	}

	if t.State == tasks.StateBacklog && t.StartedAt.IsZero() {
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			tasks.ErrBacklogNotStarted,
		)
	}
	messageID := req.MessageId
	if messageID == "" {
		messageID = uuid.NewString()
	}

	s.recordMessage(ctx, t.ID, messageID, req.Prompt, actorID, req.AttachmentIds)

	result, err := s.Tasks.FollowUp(ctx, tasks.FollowUpParams{
		TaskID:     t.ID,
		Text:       req.Prompt,
		ActorID:    actorID,
		MessageID:  messageID,
		Cancelable: true,
	})
	if err != nil {
		return nil, mapError(err)
	}
	s.wake()
	return &agencyv1.AddToTaskResponse{
		MessageId: messageID,
		Queued:    result.Queued,
	}, nil
}

func (s *TaskServer) RemoveTaskPendingMessage(
	ctx context.Context,
	req *agencyv1.RemoveTaskPendingMessageRequest,
) (*agencyv1.RemoveTaskPendingMessageResponse, error) {
	t, err := s.promptableTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	updated, err := s.Tasks.RemovePendingMessage(ctx, t.ID, req.MessageId)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RemoveTaskPendingMessageResponse{
		Task: s.protoTask(ctx, updated),
	}, nil
}

func (s *TaskServer) promptableTask(
	ctx context.Context,
	id string,
) (tasks.Task, error) {
	t, err := s.Tasks.Get(ctx, id)
	if err != nil {
		return tasks.Task{}, mapError(err)
	}
	actorID, _ := auth.ActorID(ctx)
	allowed, err := s.Tasks.CanPrompt(ctx, t, actorID, auth.IsAdmin(ctx))
	if err != nil {
		return tasks.Task{}, mapError(err)
	}
	if !allowed {
		return tasks.Task{}, connect.NewError(
			connect.CodePermissionDenied,
			tasks.ErrCannotPrompt,
		)
	}
	return t, nil
}

func (s *TaskServer) resumeModel(
	ctx context.Context,
	t tasks.Task,
) scheduler.ModelSelection {
	if s.Scheduler == nil {
		return scheduler.ModelSelection{}
	}
	return s.Scheduler.ResumeModel(ctx, t, t.SessionAgentID)
}

func (s *TaskServer) recordMessage(
	ctx context.Context,
	taskID, messageID, text string,
	actorID ids.ActorID,
	attachmentIDs []int64,
) {
	if s.Attachments == nil {
		return
	}
	if err := s.Attachments.RecordMessage(
		ctx, taskID, messageID, text, actorID, attachmentIDs, time.Now().UnixMilli(),
	); err != nil {
		log.Printf("[task %s] record message: %v", taskID, err)
	}
}

func (s *TaskServer) StopTask(
	ctx context.Context,
	req *agencyv1.StopTaskRequest,
) (*agencyv1.StopTaskResponse, error) {

	if _, err := s.visibleTask(ctx, req.Id); err != nil {
		return nil, err
	}
	t, conn, err := s.activeTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	s.Tasks.AppendLog(
		ctx, t.ID, tasks.LogStop, "", "", "", "Stop requested",
		map[string]any{"requested": true},
	)
	if _, err := conn.Tasks.StopTask(ctx, &agentv1.StopTaskRequest{
		Id:    t.SessionID,
		Actor: stopActor(ctx),
	}); err != nil {

		s.Tasks.AppendLog(
			ctx, t.ID, tasks.LogStop, "", "", "", "Stop failed",
			map[string]any{"failed": true, "error": err.Error()},
		)
		return nil, err
	}
	return &agencyv1.StopTaskResponse{}, nil
}

func (s *TaskServer) InterruptTaskToolCall(
	ctx context.Context,
	req *agencyv1.InterruptTaskToolCallRequest,
) (*agencyv1.InterruptTaskToolCallResponse, error) {

	if strings.TrimSpace(req.ToolCallId) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("tool_call_id is required"))
	}
	if _, err := s.visibleTask(ctx, req.Id); err != nil {
		return nil, err
	}
	t, conn, err := s.activeTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Tasks.InterruptToolCall(ctx, &agentv1.InterruptToolCallRequest{
		Id:         t.SessionID,
		ToolCallId: req.ToolCallId,
		Actor:      stopActor(ctx),
	}); err != nil {
		return nil, err
	}
	return &agencyv1.InterruptTaskToolCallResponse{}, nil
}

func stopActor(ctx context.Context) string {
	if identity, ok := auth.IdentityFrom(ctx); ok {
		if name := strings.TrimSpace(identity.Name); name != "" {
			return name
		}
		if username := strings.TrimSpace(identity.Username); username != "" {
			return username
		}
	}
	return "an operator"
}

func (s *TaskServer) RetryTask(
	ctx context.Context,
	req *agencyv1.RetryTaskRequest,
) (*agencyv1.RetryTaskResponse, error) {
	t, conn, err := s.activeTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	model := s.resumeModel(ctx, t)
	if _, err := conn.Tasks.RetryTask(ctx, &agentv1.RetryTaskRequest{
		Id:            t.SessionID,
		Ref:           taskRef(t),
		RequiredTools: t.RequiredTools,
		ModelProvider: model.Provider,
		ModelId:       model.ModelID,
		ThinkingLevel: model.ThinkingLevel,
	}); err != nil {
		return nil, err
	}
	if t.State == tasks.StateSettled || t.State == tasks.StateFailed {
		if _, err := s.Tasks.Resume(ctx, t.ID); err != nil {
			log.Printf("[task %s] resume: %v", t.ID, err)
		}
		t, _ = s.Tasks.Get(ctx, t.ID)
	}
	return &agencyv1.RetryTaskResponse{Task: s.protoTask(ctx, t)}, nil
}

func (s *TaskServer) WatchTask(
	ctx context.Context,
	req *agencyv1.WatchTaskRequest,
	stream *connect.ServerStream[agencyv1.WatchTaskResponse],
) error {
	t, err := s.visibleTask(ctx, req.Id)
	if err != nil {
		return err
	}
	if !t.SessionDeletedAt.IsZero() {

		return nil
	}
	_, conn, err := s.sessionTask(ctx, req.Id)
	if err != nil {
		return err
	}
	inner, err := conn.Tasks.WatchTask(ctx, &agentv1.WatchTaskRequest{
		Id:            t.SessionID,
		ReplayHistory: req.ReplayHistory,
	})
	if err != nil {
		return err
	}
	for inner.Receive() {
		event := inner.Msg().Event

		switch {
		case event.GetRunStarted() != nil:
			s.materializeStatus(
				ctx,
				conn.ID,
				t,
				agentv1.TaskStatus_TASK_STATUS_IN_PROGRESS,
				"",
			)
		case event.GetRunSettled() != nil:

			if _, err := s.Tasks.Settle(ctx, t.ID); err != nil {
				log.Printf("[task %s] settle: %v", t.ID, err)
			}
		case event.GetStatusChanged() != nil:
			s.materializeStatus(
				ctx,
				conn.ID,
				t,
				event.GetStatusChanged().GetStatus(),
				event.GetStatusChanged().GetError(),
			)
		}
		if err := stream.Send(
			&agencyv1.WatchTaskResponse{Event: event},
		); err != nil {
			return err
		}
	}
	err = inner.Err()
	if connect.CodeOf(err) == connect.CodeNotFound {

		log.Printf("[task %s] watch lost session %s: %v", t.ID, t.SessionID, err)
		return nil
	}
	return err
}

func (s *TaskServer) readableTask(ctx context.Context, id string) (tasks.Task, error) {
	if identity, ok := auth.IdentityFrom(ctx); ok && identity.Kind == "agent" {
		return s.agentTask(ctx, id)
	}
	return s.visibleTask(ctx, id)
}

func (s *TaskServer) requireCanEdit(ctx context.Context, id string) (tasks.Task, error) {
	t, err := s.Tasks.Get(ctx, id)
	if err != nil {
		return tasks.Task{}, mapError(err)
	}
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return t, nil
	}
	allowed, err := s.Tasks.CanUpdate(ctx, t, actorID, auth.IsAdmin(ctx))
	if err != nil {
		return tasks.Task{}, mapError(err)
	}
	if !allowed {
		return tasks.Task{}, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("not allowed to edit this task"),
		)
	}
	return t, nil
}

func (s *TaskServer) writableTask(ctx context.Context, id string) (tasks.Task, error) {
	if auth.IsAdmin(ctx) {
		return s.visibleTask(ctx, id)
	}
	if _, ok := auth.ActorID(ctx); !ok {
		return tasks.Task{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("authentication required"))
	}
	return s.agentTask(ctx, id)
}

func (s *TaskServer) agentTask(ctx context.Context, id string) (tasks.Task, error) {
	t, err := s.Tasks.Get(ctx, id)
	if err != nil {
		return tasks.Task{}, mapError(err)
	}
	actorID, ok := auth.ActorID(ctx)
	if !ok {
		return tasks.Task{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("authentication required"))
	}
	if t.AssigneeActorID != 0 && t.AssigneeActorID == actorID {
		return t, nil
	}
	if agent, err := s.Agents.GetByActor(ctx, actorID); err == nil &&
		t.SessionAgentID != 0 && agent.ID == t.SessionAgentID {
		return t, nil
	}
	return tasks.Task{}, connect.NewError(connect.CodePermissionDenied,
		errors.New("task is not assigned to this agent"))
}

func (s *TaskServer) GetTaskSessionState(
	ctx context.Context,
	req *agencyv1.GetTaskSessionStateRequest,
) (*agencyv1.GetTaskSessionStateResponse, error) {
	t, err := s.readableTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	state, found, err := s.Tasks.SessionState(ctx, t.ID)
	if err != nil {
		return nil, mapError(err)
	}
	lastSeq, err := s.Tasks.LastTaskEventSeq(ctx, t.ID)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.GetTaskSessionStateResponse{
		Found: found,
		State: &agencyv1.TaskSessionState{
			TaskId:        t.ID,
			Revision:      state.Revision,
			ModelProvider: state.ModelProvider,
			ModelId:       state.ModelID,
			ThinkingLevel: state.ThinkingLevel,
			EntriesJson:   state.EntriesJSON,
			UpdatedAt:     state.UpdatedAt,
			LastEventSeq:  lastSeq,
		},
	}, nil
}

func (s *TaskServer) PutTaskSessionState(
	ctx context.Context,
	req *agencyv1.PutTaskSessionStateRequest,
) (*agencyv1.PutTaskSessionStateResponse, error) {
	t, err := s.writableTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.EntriesJson) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("entries_json is required"))
	}
	revision, err := s.Tasks.PutSessionState(ctx, t.ID, tasks.SessionState{
		ModelProvider: req.ModelProvider,
		ModelID:       req.ModelId,
		ThinkingLevel: req.ThinkingLevel,
		EntriesJSON:   req.EntriesJson,
	}, req.ExpectedRevision)
	if err != nil {
		if errors.Is(err, tasks.ErrSessionStateConflict) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, mapError(err)
	}
	return &agencyv1.PutTaskSessionStateResponse{Revision: revision}, nil
}

func (s *TaskServer) AppendTaskEvents(
	ctx context.Context,
	req *agencyv1.AppendTaskEventsRequest,
) (*agencyv1.AppendTaskEventsResponse, error) {
	t, err := s.writableTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	rows := make([]tasks.TaskEventRow, 0, len(req.Events))
	for _, event := range req.Events {
		if event == nil {
			continue
		}
		rows = append(rows, tasks.TaskEventRow{
			Seq:         event.Seq,
			At:          event.At,
			Kind:        event.Kind,
			PayloadJSON: event.PayloadJson,
		})
	}
	if _, err := s.Tasks.AppendTaskEvents(ctx, t.ID, rows); err != nil {
		return nil, mapError(err)
	}
	lastSeq, err := s.Tasks.LastTaskEventSeq(ctx, t.ID)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.AppendTaskEventsResponse{LastSeq: lastSeq}, nil
}

func (s *TaskServer) GetTaskInput(
	ctx context.Context,
	req *agencyv1.GetTaskInputRequest,
) (*agencyv1.GetTaskInputResponse, error) {
	task, err := s.visibleTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if s.Input == nil {
		return nil, connect.NewError(connect.CodeUnavailable,
			errors.New("task input is unavailable"))
	}

	dialog, waiting, err := s.Input.Pending(ctx, task)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	out := &agencyv1.TaskInput{
		TaskId:  task.ID,
		State:   string(task.State),
		Waiting: waiting,
	}
	if !waiting {
		out.Hint = "the task is not waiting for input right now; " +
			"read it again with getTask or getTaskHistory"
		return &agencyv1.GetTaskInputResponse{Input: out}, nil
	}
	out.RequestId = dialog.RequestID
	out.Method = dialog.Method
	out.Title = dialog.Title
	out.Message = dialog.Message
	out.Placeholder = dialog.Placeholder
	out.Options = dialog.Options
	out.Timeout = dialog.Timeout
	out.Raw = dialog.Raw
	if dialog.Questionnaire != nil {
		raw, err := json.Marshal(dialog.Questionnaire)
		if err == nil {
			out.QuestionnaireJson = string(raw)
		}
	}
	out.Hint = answerHint(dialog)
	return &agencyv1.GetTaskInputResponse{Input: out}, nil
}

func answerHint(dialog taskinput.Dialog) string {
	switch {
	case dialog.Questionnaire != nil:
		return "answer with \"answers\": one entry per question, each addressing " +
			"the question by its header (or 1-based index) and naming an option " +
			"label, or sending \"custom\""
	case dialog.Method == "confirm":
		return "answer with \"confirmed\": true or false"
	case dialog.Method == "select":
		return "answer with \"value\": exactly one of the listed options"
	case dialog.Method == "input", dialog.Method == "editor":
		return "answer with \"value\": the text to send"
	default:
		return "answer with \"value\""
	}
}

func (s *TaskServer) RespondToTaskInput(
	ctx context.Context,
	req *agencyv1.RespondToTaskInputRequest,
) (*agencyv1.RespondToTaskInputResponse, error) {
	task, err := s.visibleTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if !s.canAnswerInput(ctx, task) {
		return nil, connect.NewError(connect.CodePermissionDenied,
			errors.New("only a lead of this task may read or answer its input"))
	}
	if s.Input == nil {
		return nil, connect.NewError(connect.CodeUnavailable,
			errors.New("task input is unavailable"))
	}

	dialog, waiting, err := s.Input.Pending(ctx, task)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if !waiting {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("the task is not waiting for input right now (it may have "+
				"been answered already); read it again with GetTaskInput"))
	}
	in := req.GetAnswer()
	if in == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("an answer is required"))
	}
	if requestID := strings.TrimSpace(in.GetRequestId()); requestID != "" &&
		requestID != dialog.RequestID {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("request_id "+requestID+" does not match the pending dialog ("+
				dialog.RequestID+"); read it again with GetTaskInput"))
	}

	var answers []taskinput.LabelAnswer
	if raw := strings.TrimSpace(in.GetAnswersJson()); raw != "" {
		if err := json.Unmarshal([]byte(raw), &answers); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("answers_json: %w", err))
		}
	}
	answer, err := inputAnswer(dialog, in, answers)
	if err != nil {
		return nil, mapTaskInputError(err)
	}
	if err := s.Input.Answer(ctx, task, answer); err != nil {
		return nil, mapTaskInputError(err)
	}
	refreshed, err := s.Tasks.Get(ctx, task.ID)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RespondToTaskInputResponse{
		TaskId:    refreshed.ID,
		Answered:  true,
		Cancelled: answer.Cancelled,
		State:     string(refreshed.State),
		Hint:      answerHintAfter(refreshed, answer.Cancelled),
	}, nil
}

func answerHintAfter(_ tasks.Task, cancelled bool) string {
	if cancelled {
		return "the dialog was declined; the task continues without an answer"
	}
	return "the task continues from your answer; its run is in flight again"
}

func inputAnswer(
	dialog taskinput.Dialog,
	in *agencyv1.TaskInputAnswer,
	answers []taskinput.LabelAnswer,
) (taskinput.Answer, error) {
	answer := taskinput.Answer{RequestID: dialog.RequestID}
	confirmed := in.Confirmed
	value := in.GetValue()
	note := in.GetNote()
	if in.GetCancelled() {
		if len(answers) > 0 || value != "" || confirmed != nil {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "cancelled cannot be combined with an answer; " +
					"send either cancelled or the dialog's answer",
			}
		}
		answer.Cancelled = true
		return answer, nil
	}
	if dialog.Questionnaire != nil {
		if confirmed != nil || value != "" {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a questionnaire; answer it with " +
					"\"answers\" (question + option label, or custom), not " +
					"\"value\" or \"confirmed\"",
			}
		}
		resolved, err := dialog.Questionnaire.AnswerValue(answers, note)
		if err != nil {
			return taskinput.Answer{}, err
		}
		resolved.RequestID = dialog.RequestID
		return resolved, nil
	}
	switch dialog.Method {
	case "confirm":
		if confirmed == nil {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a confirm; answer it with " +
					"\"confirmed\": true or false",
			}
		}
		if len(answers) > 0 || value != "" || note != "" {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "a confirm dialog takes only \"confirmed\"; drop the " +
					"other fields",
			}
		}
		choice := *confirmed
		answer.Confirmed = &choice
		return answer, nil
	case "select", "input", "editor":
		if len(answers) > 0 {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a plain " + dialog.Method +
					"; answer it with \"value\", not \"answers\"",
			}
		}
		if confirmed != nil {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a plain " + dialog.Method +
					"; answer it with \"value\", not \"confirmed\"",
			}
		}
		if dialog.Method == "select" && strings.TrimSpace(value) == "" {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a select; answer it with \"value\" " +
					"set to one of: " + optionsForError(dialog.Options),
			}
		}
		answer.Value = value
		return answer, nil
	default:
		if len(answers) > 0 {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog does not take \"answers\"; answer it " +
					"with \"value\" or \"confirmed\"",
			}
		}
		answer.Value = value
		if confirmed != nil {
			choice := *confirmed
			answer.Confirmed = &choice
		}
		return answer, nil
	}
}

func optionsForError(options []string) string {
	if len(options) == 0 {
		return "no options"
	}
	quoted := make([]string, 0, len(options))
	for _, option := range options {
		quoted = append(quoted, "\""+option+"\"")
	}
	return strings.Join(quoted, ", ")
}

func mapTaskInputError(err error) error {
	switch {
	case errors.Is(err, taskinput.ErrNoPendingDialog):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case func() bool { _, ok := err.(*taskinput.ValidationError); return ok }():
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeUnavailable, err)
	}
}

func (s *TaskServer) ResolveTaskRef(
	ctx context.Context,
	req *agencyv1.ResolveTaskRefRequest,
) (*agencyv1.ResolveTaskRefResponse, error) {
	if strings.TrimSpace(req.ProjectId) == "" || req.Seq <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("project_id and seq are required"))
	}
	task, err := s.Tasks.GetByProjectSeq(ctx, req.ProjectId, req.Seq)
	if err != nil {
		return nil, mapError(err)
	}
	if _, err := s.agentTask(ctx, task.ID); err != nil {
		return nil, err
	}
	return &agencyv1.ResolveTaskRefResponse{TaskId: task.ID}, nil
}

func (s *TaskServer) GetTaskHistory(
	ctx context.Context,
	req *agencyv1.GetTaskHistoryRequest,
) (*agencyv1.GetTaskHistoryResponse, error) {
	t, err := s.visibleTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	var before int64
	if req.Before != "" {
		before, err = strconv.ParseInt(req.Before, 10, 64)
		if err != nil || before < 0 {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("invalid history cursor"),
			)
		}
	}

	lastSeq, err := s.Tasks.LastTaskEventSeq(ctx, t.ID)
	if err != nil {
		return nil, mapError(err)
	}
	if lastSeq > 0 {

		limit := taskhistory.PageLimit(int(req.Limit))

		stored, err := s.Tasks.TaskEventPage(ctx, t.ID, before, limit+1)
		if err != nil {
			return nil, mapError(err)
		}
		var (
			events []*agentv1.TaskEvent
			next   int64
		)
		if req.ModelFacing {

			events, next = taskhistory.PageEventsBounded(
				t.ID, stored, before, limit,
				taskhistory.ModelMessageBytes, taskhistory.ModelPageBytes,
			)
		} else {
			events, next = taskhistory.PageEvents(t.ID, stored, before, limit)
		}
		nextBefore := ""
		if next > 0 {
			nextBefore = strconv.FormatInt(next, 10)
		}
		return &agencyv1.GetTaskHistoryResponse{Events: events, NextBefore: nextBefore}, nil
	}

	entries, err := s.Tasks.SessionHistory(ctx, t.ID)
	if err != nil {
		return nil, mapError(err)
	}
	if len(entries) > 0 {

		var (
			events []*agentv1.TaskEvent
			next   int64
		)
		if req.ModelFacing {
			events, next = taskhistory.PageBounded(
				t.ID, entries, before, int(req.Limit),
				taskhistory.ModelMessageBytes, taskhistory.ModelPageBytes,
			)
		} else {
			events, next = taskhistory.Page(t.ID, entries, before, int(req.Limit))
		}
		nextBefore := ""
		if next > 0 {
			nextBefore = strconv.FormatInt(next, 10)
		}
		return &agencyv1.GetTaskHistoryResponse{Events: events, NextBefore: nextBefore}, nil
	}

	if !t.SessionDeletedAt.IsZero() {

		return &agencyv1.GetTaskHistoryResponse{}, nil
	}

	t, conn, err := s.sessionTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	res, err := conn.Tasks.GetTaskHistory(ctx, &agentv1.GetTaskHistoryRequest{
		Id:     t.SessionID,
		Before: req.Before,
		Limit:  req.Limit,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {

			log.Printf("[task %s] history of lost session %s: %v", t.ID, t.SessionID, err)
			return &agencyv1.GetTaskHistoryResponse{}, nil
		}
		return nil, err
	}
	if req.ModelFacing {

		taskhistory.TruncateForModel(res.Events, taskhistory.ModelMessageBytes)
	}
	return &agencyv1.GetTaskHistoryResponse{
		Events:     res.Events,
		NextBefore: res.NextBefore,
	}, nil
}

func (s *TaskServer) cachedToolCalls(id, tip string) ([]taskhistory.ToolCall, bool) {
	s.toolCallMu.Lock()
	defer s.toolCallMu.Unlock()
	entry, ok := s.toolCallCache[id]
	if !ok || entry.tip != tip {
		return nil, false
	}
	return entry.calls, true
}

func (s *TaskServer) storeToolCalls(
	id, tip string,
	calls []taskhistory.ToolCall,
) {
	s.toolCallMu.Lock()
	defer s.toolCallMu.Unlock()
	if s.toolCallCache == nil || len(s.toolCallCache) >= maxToolCallCache {
		s.toolCallCache = make(map[string]cachedToolCalls, maxToolCallCache)
	}
	s.toolCallCache[id] = cachedToolCalls{tip: tip, calls: calls}
}

func (s *TaskServer) ListTaskCompactions(
	ctx context.Context,
	req *agencyv1.ListTaskCompactionsRequest,
) (*agencyv1.ListTaskCompactionsResponse, error) {
	t, err := s.visibleTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	out := &agencyv1.ListTaskCompactionsResponse{}
	const pageSize = 500
	var before int64
	for {
		rows, err := s.Tasks.TaskEventPage(ctx, t.ID, before, pageSize)
		if err != nil {
			return nil, mapError(err)
		}
		for _, row := range rows {
			var event agentv1.TaskEvent
			if protojson.Unmarshal([]byte(row.PayloadJSON), &event) != nil {
				continue
			}
			wait := event.GetModelWait()
			if wait.GetKind() != agentv1.ModelWait_KIND_COMPACTION {
				continue
			}
			out.Compactions = append(out.Compactions, &agencyv1.TaskCompaction{
				At:                  timestamppb.New(time.UnixMilli(row.At)),
				DurationMs:          wait.GetDurationMs(),
				Reason:              wait.GetReason(),
				Model:               wait.GetModel(),
				Provider:            wait.GetProvider(),
				ContextTokensBefore: wait.GetContextTokensBefore(),
				ContextTokensAfter:  wait.GetContextTokensAfter(),
			})
		}
		if len(rows) < pageSize {
			break
		}
		before = rows[0].Seq
	}
	slices.SortFunc(out.Compactions, func(a, b *agencyv1.TaskCompaction) int {
		return b.GetAt().AsTime().Compare(a.GetAt().AsTime())
	})
	return out, nil
}

func (s *TaskServer) ListTaskToolCalls(
	ctx context.Context,
	req *agencyv1.ListTaskToolCallsRequest,
) (*agencyv1.ListTaskToolCallsResponse, error) {
	t, err := s.visibleTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	calls, err := s.taskToolCalls(ctx, t)
	if err != nil {
		return nil, err
	}
	page, total := taskhistory.FilterToolCalls(calls, taskhistory.ToolCallFilter{
		Names:           req.ToolNames,
		ErrorsOnly:      req.ErrorsOnly,
		InterruptedOnly: req.InterruptedOnly,
		Search:          req.Search,
		Limit:           int(req.Limit),
		Offset:          int(req.Offset),
	})
	out := &agencyv1.ListTaskToolCallsResponse{
		Total: int32(total),
		Names: taskhistory.ToolCallNames(calls),
	}
	for _, call := range page {
		out.Calls = append(out.Calls, &agencyv1.TaskToolCall{
			CallId:      call.CallID,
			Name:        call.Name,
			IsError:     call.IsError,
			Interrupted: call.Interrupted(),
			StartedAt:   call.StartedAt,
			EndedAt:     call.EndedAt,
			DurationMs:  call.DurationMs(),
			ArgsJson:    call.ArgsPreview(),
			ResultBytes: int64(call.ResultBytes()),
		})
	}
	return out, nil
}

func (s *TaskServer) GetTaskToolCall(
	ctx context.Context,
	req *agencyv1.GetTaskToolCallRequest,
) (*agencyv1.GetTaskToolCallResponse, error) {
	t, err := s.visibleTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	calls, err := s.taskToolCalls(ctx, t)
	if err != nil {
		return nil, err
	}
	for _, call := range calls {
		if call.CallID != req.CallId {
			continue
		}
		return &agencyv1.GetTaskToolCallResponse{
			Call: toolCallDetail(call, req.Full),
		}, nil
	}
	return nil, connect.NewError(
		connect.CodeNotFound,
		fmt.Errorf("task %s recorded no tool call %q", t.ID, req.CallId),
	)
}

func toolCallDetail(
	call taskhistory.ToolCall,
	full bool,
) *agencyv1.TaskToolCallDetail {
	limit := taskhistory.ToolCallBodyLimit
	if full {
		limit = 0
	}
	args, argsTruncated := taskhistory.PreviewBody(call.ArgsJSON, limit)
	result, resultTruncated := taskhistory.PreviewBody(call.ResultJSON, limit)
	details, detailsTruncated := taskhistory.PreviewBody(call.DetailsJSON, limit)
	return &agencyv1.TaskToolCallDetail{
		CallId:           call.CallID,
		Name:             call.Name,
		IsError:          call.IsError,
		Interrupted:      call.Interrupted(),
		StartedAt:        call.StartedAt,
		EndedAt:          call.EndedAt,
		DurationMs:       call.DurationMs(),
		ArgsJson:         args,
		ResultJson:       result,
		DetailsJson:      details,
		ArgsBytes:        int64(len(call.ArgsJSON)),
		ResultBytes:      int64(len(call.ResultJSON)),
		DetailsBytes:     int64(len(call.DetailsJSON)),
		ArgsTruncated:    argsTruncated,
		ResultTruncated:  resultTruncated,
		DetailsTruncated: detailsTruncated,
	}
}

func (s *TaskServer) ListTaskTools(
	ctx context.Context,
	req *agencyv1.ListTaskToolsRequest,
) (*agencyv1.ListTaskToolsResponse, error) {
	t, err := s.visibleTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	tools, recorded, err := s.taskToolLoadout(ctx, t)
	if err != nil {
		return nil, err
	}
	owners := s.pluginToolOwners(ctx, t)
	counts := map[string]int32{}
	calls, err := s.taskToolCalls(ctx, t)
	if err != nil {

		log.Printf("[task %s] tool call counts: %v", t.ID, err)
	}
	for _, call := range calls {
		counts[call.Name]++
	}
	out := &agencyv1.ListTaskToolsResponse{
		Recorded: recorded,
		Tools:    make([]*agencyv1.TaskTool, 0, len(tools)),
	}
	for _, tool := range tools {
		entry := &agencyv1.TaskTool{
			Name:           tool.Name,
			Description:    tool.Description,
			ParametersJson: tool.ParametersJSON,
			Origin:         agencyv1.TaskToolOrigin_TASK_TOOL_ORIGIN_BUILTIN,
			CallCount:      counts[tool.Name],
		}
		if owner, ok := owners[tool.Name]; ok {
			entry.Origin = agencyv1.TaskToolOrigin_TASK_TOOL_ORIGIN_PLUGIN
			entry.PluginId = owner.ID
			entry.PluginName = owner.Name
		}
		out.Tools = append(out.Tools, entry)
	}
	return out, nil
}

func (s *TaskServer) taskToolLoadout(
	ctx context.Context,
	t tasks.Task,
) ([]taskhistory.Tool, bool, error) {
	if tools, ok := s.reportedToolLoadout(ctx, t); ok {
		return tools, true, nil
	}
	tools, recorded, err := s.projectedToolLoadout(ctx, t)
	if err != nil {
		return nil, false, err
	}
	if recorded {
		return tools, true, nil
	}
	tools, recorded, err = s.legacyToolLoadout(ctx, t)
	if err != nil {
		return nil, false, err
	}
	if recorded {
		return tools, true, nil
	}
	return taskhistory.BuiltinBaseline(!t.Automated), false, nil
}

func (s *TaskServer) reportedToolLoadout(
	ctx context.Context,
	t tasks.Task,
) ([]taskhistory.Tool, bool) {
	if s.Integrations == nil {
		return nil, false
	}
	report, err := s.Integrations.TaskReport(ctx, t.ID)
	if err != nil {
		log.Printf("[task %s] tool report: %v", t.ID, err)
		return nil, false
	}
	tools := make([]taskhistory.Tool, 0, len(report.Report.Tools))
	for _, tool := range report.Report.Tools {
		if tool.Name == "" {
			continue
		}
		tools = append(tools, taskhistory.Tool{
			Name:           tool.Name,
			Description:    tool.Description,
			ParametersJSON: string(tool.Parameters),
		})
	}
	return tools, len(tools) > 0
}

func (s *TaskServer) projectedToolLoadout(
	ctx context.Context,
	t tasks.Task,
) ([]taskhistory.Tool, bool, error) {
	state, found, err := s.Tasks.SessionState(ctx, t.ID)
	if err != nil {
		return nil, false, mapError(err)
	}
	if !found || state.EntriesJSON == "" {
		return nil, false, nil
	}
	source := "state:" + strconv.FormatInt(state.Revision, 10)
	if tools, recorded, ok := s.cachedToolLoadout(t.ID, source); ok && recorded {
		return tools, true, nil
	}
	tools, recorded := taskhistory.CollectProjectedToolLoadout(state.EntriesJSON)
	if recorded {
		s.storeToolLoadout(t.ID, source, tools, recorded)
		return tools, true, nil
	}
	return nil, false, nil
}

func (s *TaskServer) legacyToolLoadout(
	ctx context.Context,
	t tasks.Task,
) ([]taskhistory.Tool, bool, error) {
	tip, _, err := s.Tasks.SessionHistoryTip(ctx, t.ID)
	if err != nil {
		return nil, false, mapError(err)
	}
	if tip != "" {
		if tools, recorded, ok := s.cachedToolLoadout(t.ID, tip); ok {
			return tools, recorded, nil
		}
	}
	entries, err := s.Tasks.SessionHistory(ctx, t.ID)
	if err != nil {
		return nil, false, mapError(err)
	}
	tools, recorded := taskhistory.CollectToolLoadout(entries)
	if len(entries) > 0 {
		s.storeToolLoadout(t.ID, tip, tools, recorded)
	}
	return tools, recorded, nil
}

func (s *TaskServer) cachedToolLoadout(
	id, source string,
) ([]taskhistory.Tool, bool, bool) {
	s.toolCallMu.Lock()
	defer s.toolCallMu.Unlock()
	entry, ok := s.toolLoadoutCache[id]
	if !ok || entry.source != source {
		return nil, false, false
	}
	return entry.tools, entry.recorded, true
}

func (s *TaskServer) storeToolLoadout(
	id, source string,
	tools []taskhistory.Tool,
	recorded bool,
) {
	s.toolCallMu.Lock()
	defer s.toolCallMu.Unlock()
	if s.toolLoadoutCache == nil || len(s.toolLoadoutCache) >= maxToolCallCache {
		s.toolLoadoutCache = make(map[string]cachedToolLoadout, maxToolCallCache)
	}
	s.toolLoadoutCache[id] = cachedToolLoadout{
		source:   source,
		tools:    tools,
		recorded: recorded,
	}
}

func (s *TaskServer) pluginToolOwners(
	ctx context.Context,
	t tasks.Task,
) map[string]pluginToolOwner {
	if s.Plugins == nil {
		return map[string]pluginToolOwner{}
	}
	projectTools := s.Plugins.ProjectTools(ctx, t.ProjectID)
	row, ok := s.Plugins.TaskPlugin(ctx, t.ID)
	if !ok {
		return toolOwners(projectTools, nil, "")
	}
	name := row.PluginID
	if plugin, err := s.Plugins.Get(ctx, row.PluginID); err == nil && plugin.Name != "" {
		name = plugin.Name
	}
	return toolOwners(projectTools, &row, name)
}

func toolOwners(
	projectTools []plugins.ProjectTool,
	attached *plugins.TaskPlugin,
	attachedName string,
) map[string]pluginToolOwner {
	owners := map[string]pluginToolOwner{}
	for _, tool := range projectTools {
		if _, taken := owners[tool.Tool.Name]; taken {
			continue
		}
		owners[tool.Tool.Name] = pluginToolOwner{
			ID:   tool.PluginID,
			Name: tool.PluginName,
		}
	}
	if attached == nil {
		return owners
	}
	for _, tool := range attached.Tools {
		if _, taken := owners[tool.Name]; taken {
			continue
		}
		owners[tool.Name] = pluginToolOwner{ID: attached.PluginID, Name: attachedName}
	}
	return owners
}

func (s *TaskServer) taskToolCalls(
	ctx context.Context,
	t tasks.Task,
) ([]taskhistory.ToolCall, error) {

	tip, _, err := s.Tasks.SessionHistoryTip(ctx, t.ID)
	if err != nil {
		return nil, mapError(err)
	}
	if tip != "" {
		if calls, ok := s.cachedToolCalls(t.ID, tip); ok {
			return calls, nil
		}
	}
	entries, err := s.Tasks.SessionHistory(ctx, t.ID)
	if err != nil {
		return nil, mapError(err)
	}
	if len(entries) > 0 {
		calls := taskhistory.CollectToolCalls(entries)
		s.storeToolCalls(t.ID, tip, calls)
		return calls, nil
	}
	if !t.SessionDeletedAt.IsZero() {
		return nil, nil
	}
	var events []*agentv1.TaskEvent
	before := ""
	for {
		page, err := s.GetTaskHistory(ctx, &agencyv1.GetTaskHistoryRequest{
			Id:     t.ID,
			Before: before,
			Limit:  1000,
		})
		if err != nil {
			return nil, err
		}
		events = append(page.Events, events...)
		if page.NextBefore == "" {
			break
		}
		before = page.NextBefore
	}
	return taskhistory.CollectToolCallsFromEvents(events), nil
}

func (s *TaskServer) GetTaskConfig(
	ctx context.Context,
	req *agencyv1.GetTaskConfigRequest,
) (*agencyv1.GetTaskConfigResponse, error) {
	if _, err := s.visibleTask(ctx, req.Id); err != nil {
		return nil, err
	}
	t, conn, err := s.sessionTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	res, err := conn.Tasks.GetTaskConfig(ctx, &agentv1.GetTaskConfigRequest{Id: t.SessionID})
	if err != nil {
		return nil, err
	}
	return &agencyv1.GetTaskConfigResponse{Config: res.Config}, nil
}

func (s *TaskServer) GetTaskDefaults(
	ctx context.Context,
	req *agencyv1.GetTaskDefaultsRequest,
) (*agencyv1.GetTaskDefaultsResponse, error) {
	conn, err := s.connection(actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, err
	}
	res, err := conn.GetTaskDefaults(ctx, &agentv1.GetTaskDefaultsRequest{})
	if err != nil {
		return nil, err
	}
	return &agencyv1.GetTaskDefaultsResponse{Defaults: res.Defaults}, nil
}

func (s *TaskServer) SetTaskModel(
	ctx context.Context,
	req *agencyv1.SetTaskModelRequest,
) (*agencyv1.SetTaskModelResponse, error) {
	t, conn, err := s.sessionTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if err := s.requireTaskModelSelection(ctx, t); err != nil {
		return nil, err
	}
	res, err := conn.Tasks.SetTaskModel(ctx, &agentv1.SetTaskModelRequest{
		Id:       t.SessionID,
		Provider: req.Provider,
		ModelId:  req.ModelId,
	})
	if err != nil {
		return nil, err
	}
	return &agencyv1.SetTaskModelResponse{Config: res.Config}, nil
}

func (s *TaskServer) SetTaskThinkingLevel(
	ctx context.Context,
	req *agencyv1.SetTaskThinkingLevelRequest,
) (*agencyv1.SetTaskThinkingLevelResponse, error) {
	t, conn, err := s.sessionTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if err := s.requireTaskModelSelection(ctx, t); err != nil {
		return nil, err
	}
	res, err := conn.Tasks.SetTaskThinkingLevel(
		ctx,
		&agentv1.SetTaskThinkingLevelRequest{Id: t.SessionID, Level: req.Level},
	)
	if err != nil {
		return nil, err
	}
	return &agencyv1.SetTaskThinkingLevelResponse{Config: res.Config}, nil
}

func (s *TaskServer) RespondToExtensionUi(
	ctx context.Context,
	req *agencyv1.RespondToExtensionUiRequest,
) (*agencyv1.RespondToExtensionUiResponse, error) {
	t, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	if err := s.requireAnswerInput(ctx, t); err != nil {
		return nil, err
	}
	answer := taskinput.Answer{RequestID: req.RequestId}
	switch resp := req.Response.(type) {
	case *agencyv1.RespondToExtensionUiRequest_Value:
		answer.Value = resp.Value
	case *agencyv1.RespondToExtensionUiRequest_Confirmed:
		confirmed := resp.Confirmed
		answer.Confirmed = &confirmed
	case *agencyv1.RespondToExtensionUiRequest_Cancelled:
		answer.Cancelled = resp.Cancelled
	}
	for _, qa := range req.Questionnaire {
		answer.Questionnaire = append(
			answer.Questionnaire,
			taskinput.QuestionnaireAnswer{
				Question: qa.Question,
				Answer:   qa.Answer,
			},
		)
	}
	if err := s.Input.Answer(ctx, t, answer); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RespondToExtensionUiResponse{}, nil
}

func (s *TaskServer) ClearTaskQueue(
	ctx context.Context,
	req *agencyv1.ClearTaskQueueRequest,
) (*agencyv1.ClearTaskQueueResponse, error) {
	t, conn, err := s.activeTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Tasks.ClearQueue(ctx, &agentv1.ClearQueueRequest{
		Id: t.SessionID,
	}); err != nil {
		return nil, err
	}
	return &agencyv1.ClearTaskQueueResponse{}, nil
}

func (s *TaskServer) RemoveTaskQueuedMessage(
	ctx context.Context,
	req *agencyv1.RemoveTaskQueuedMessageRequest,
) (*agencyv1.RemoveTaskQueuedMessageResponse, error) {
	t, conn, err := s.sessionTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Tasks.RemoveQueuedMessage(ctx, &agentv1.RemoveQueuedMessageRequest{
		Id:      t.SessionID,
		QueueId: req.QueueId,
	}); err != nil {
		return nil, err
	}
	return &agencyv1.RemoveTaskQueuedMessageResponse{}, nil
}

func (s *TaskServer) RestartTaskSession(
	ctx context.Context,
	req *agencyv1.RestartTaskSessionRequest,
) (*agencyv1.RestartTaskSessionResponse, error) {
	t, conn, err := s.sessionTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if err := s.requireTaskControl(
		ctx,
		t,
		"not allowed to restart this task's session",
	); err != nil {
		return nil, err
	}
	if _, err := conn.Tasks.RestartTaskSession(
		ctx,
		&agentv1.RestartTaskSessionRequest{Id: t.SessionID},
	); err != nil {
		return nil, err
	}
	return &agencyv1.RestartTaskSessionResponse{}, nil
}

func (s *TaskServer) CompactTask(
	ctx context.Context,
	req *agencyv1.CompactTaskRequest,
) (*agencyv1.CompactTaskResponse, error) {
	t, conn, err := s.sessionTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if err := s.requireTaskControl(
		ctx,
		t,
		"not allowed to compact this task's session",
	); err != nil {
		return nil, err
	}
	if _, err := conn.Tasks.CompactTask(
		ctx,
		&agentv1.CompactTaskRequest{Id: t.SessionID},
	); err != nil {
		return nil, err
	}
	return &agencyv1.CompactTaskResponse{}, nil
}

func (s *TaskServer) enrichNames(ctx context.Context, list []tasks.Task) {
	missing := map[ids.ActorID][]int{}
	for i := range list {
		t := list[i]
		if (t.Name != "" && t.Prompt != "") || t.SessionID == "" || t.AssigneeActorID == 0 {
			continue
		}
		missing[t.AssigneeActorID] = append(missing[t.AssigneeActorID], i)
	}
	for actorID, idxs := range missing {
		conn, err := s.connectionForActor(ctx, actorID)
		if err != nil {
			continue
		}
		res, err := conn.Tasks.ListTasks(ctx, &agentv1.ListTasksRequest{})
		if err != nil {
			continue
		}
		byID := make(map[string]*agentv1.Task, len(res.Tasks))
		for _, at := range res.Tasks {
			byID[at.GetId()] = at
		}
		for _, i := range idxs {
			at := byID[list[i].SessionID]
			if at == nil {
				continue
			}
			name, prompt := at.GetName(), at.GetPrompt()
			if name == "" && prompt == "" {
				continue
			}
			if err := s.Tasks.Enrich(ctx, list[i].ID, name, prompt); err != nil {
				log.Printf("[task %s] enrich name: %v", list[i].ID, err)
				continue
			}
			list[i].Name = name
			list[i].Prompt = prompt
		}
	}
}

func (s *TaskServer) requireTaskModelSelection(
	ctx context.Context,
	t tasks.Task,
) error {
	return s.requireTaskControl(ctx, t, "not allowed to change this task's model")
}

func (s *TaskServer) requireTaskControl(
	ctx context.Context,
	t tasks.Task,
	message string,
) error {
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return nil
	}
	allowed, err := s.Tasks.CanPrompt(ctx, t, actorID, auth.IsAdmin(ctx))
	if err != nil {
		return mapError(err)
	}
	if !allowed {
		return connect.NewError(connect.CodePermissionDenied, errors.New(message))
	}
	return nil
}

func (s *TaskServer) requireAnswerInput(
	ctx context.Context,
	t tasks.Task,
) error {
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return nil
	}
	allowed, err := s.Tasks.CanAnswerInput(ctx, t, actorID, auth.IsAdmin(ctx))
	if err != nil {
		return mapError(err)
	}
	if !allowed {
		return connect.NewError(
			connect.CodePermissionDenied,
			errors.New("only a lead or an administrator may answer this task"),
		)
	}
	return nil
}

func (s *TaskServer) requireTaskModelChange(
	ctx context.Context,
	taskID, provider, modelID, thinkingLevel string,
) error {
	current, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return mapError(err)
	}
	if current.ModelProvider == provider && current.ModelID == modelID &&
		current.ThinkingLevel == thinkingLevel {
		return nil
	}
	return s.requireTaskModelSelection(ctx, current)
}

func (s *TaskServer) connection(agentID ids.AgentID) (agentv1connect.TaskServiceClient, error) {
	conn, err := s.Hub.Connection(agentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return conn.Tasks, nil
}

func (s *TaskServer) connectionForActor(
	ctx context.Context,
	actorID ids.ActorID,
) (*tunnel.Connection, error) {
	agent, err := s.Agents.GetByActor(ctx, actorID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	conn, err := s.Hub.Connection(agent.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return conn, nil
}

func (s *TaskServer) activeTask(
	ctx context.Context,
	id string,
) (tasks.Task, *tunnel.Connection, error) {
	t, conn, err := s.sessionTask(ctx, id)
	if err != nil {
		return tasks.Task{}, nil, err
	}
	if t.State != tasks.StateRunning && t.State != tasks.StateInput &&
		t.State != tasks.StateSettled && t.State != tasks.StateFailed {
		return tasks.Task{}, nil, connect.NewError(
			connect.CodeFailedPrecondition,
			errors.New("task is not running"),
		)
	}
	return t, conn, nil
}

func (s *TaskServer) sessionTask(
	ctx context.Context,
	id string,
) (tasks.Task, *tunnel.Connection, error) {
	t, err := s.Tasks.Get(ctx, id)
	if err != nil {
		return tasks.Task{}, nil, mapError(err)
	}
	if t.SessionID == "" || t.AssigneeActorID == 0 {
		return tasks.Task{}, nil, connect.NewError(
			connect.CodeFailedPrecondition,
			errors.New("task has no running session"),
		)
	}
	conn, err := s.connectionForActor(ctx, t.AssigneeActorID)
	if err != nil {
		return tasks.Task{}, nil, err
	}
	return t, conn, nil
}

func runtimeStates(state tasks.State) bool {
	return state == tasks.StateRunning || state == tasks.StateInput ||
		state == tasks.StateSettled
}

func (s *TaskServer) enrichRuntime(
	ctx context.Context,
	list []tasks.Task,
	out []*agencyv1.Task,
) {
	byAgent := map[ids.ActorID][]int{}
	for i, t := range list {
		if !runtimeStates(t.State) || t.SessionID == "" || t.AssigneeActorID == 0 {
			continue
		}
		byAgent[t.AssigneeActorID] = append(byAgent[t.AssigneeActorID], i)
	}
	for actorID, idxs := range byAgent {
		conn, err := s.connectionForActor(ctx, actorID)
		if err != nil {
			continue
		}
		res, err := conn.Tasks.ListTasks(ctx, &agentv1.ListTasksRequest{})
		if err != nil {
			continue
		}
		byID := make(map[string]*agentv1.Task, len(res.Tasks))
		for _, at := range res.Tasks {
			byID[at.GetId()] = at
		}
		for _, i := range idxs {
			at := byID[list[i].SessionID]
			if at == nil {
				continue
			}
			out[i].Runtime = &agencyv1.TaskRuntime{
				RunInProgress:   at.GetRunInProgress(),
				WaitingForInput: at.GetWaitingForInput(),
				ProcessRunning:  at.GetProcessRunning(),
			}
			if state, ok := s.materializeRuntime(ctx, list[i], at); ok {
				out[i].State = state
			}
		}
	}
}

func (s *TaskServer) enrichWaitingReasons(
	ctx context.Context,
	list []tasks.Task,
	out []*agencyv1.Task,
) {
	if s.Scheduler == nil {
		return
	}
	for i := range list {
		if i >= len(out) {
			continue
		}
		switch out[i].GetState() {
		case agencyv1.TaskState_TASK_STATE_QUEUED,
			agencyv1.TaskState_TASK_STATE_WAITING:
			out[i].WaitingReason = s.Scheduler.WaitingReason(ctx, list[i])
		}
	}
}

func (s *TaskServer) runtime(ctx context.Context, t tasks.Task) (*agentv1.Task, bool) {
	if !runtimeStates(t.State) || t.SessionID == "" || t.AssigneeActorID == 0 {
		return nil, false
	}
	conn, err := s.connectionForActor(ctx, t.AssigneeActorID)
	if err != nil {
		return nil, false
	}
	res, err := conn.Tasks.ListTasks(ctx, &agentv1.ListTasksRequest{})
	if err != nil {
		return nil, false
	}
	for _, at := range res.Tasks {
		if at.GetId() == t.SessionID {
			return at, true
		}
	}
	return nil, false
}

func (s *TaskServer) settleGrace() time.Duration {
	if s.SettleGrace != 0 {
		return s.SettleGrace
	}
	return tasks.SettleGrace
}

func (s *TaskServer) materializeRuntime(
	ctx context.Context,
	t tasks.Task,
	at *agentv1.Task,
) (agencyv1.TaskState, bool) {
	if at == nil {
		return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
	}
	switch {
	case at.GetWaitingForInput():
		if t.State == tasks.StateInput {
			return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
		}
		if _, err := s.Tasks.MarkInput(ctx, t.ID); err != nil {
			log.Printf("[task %s] mark input: %v", t.ID, err)
			return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
		}
		return agencyv1.TaskState_TASK_STATE_INPUT, true
	case at.GetRunInProgress():

		switch t.State {
		case tasks.StateSettled, tasks.StateFailed:
			if _, err := s.Tasks.Resume(ctx, t.ID); err != nil {
				log.Printf("[task %s] resume: %v", t.ID, err)
				return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
			}
			return agencyv1.TaskState_TASK_STATE_RUNNING, true
		case tasks.StateInput:
			if _, err := s.Tasks.ClearInput(ctx, t.ID); err != nil {
				log.Printf("[task %s] clear input: %v", t.ID, err)
				return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
			}
			return agencyv1.TaskState_TASK_STATE_RUNNING, true
		default:
			return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
		}
	default:

		if t.RunMayBeStarting(time.Now(), s.settleGrace()) {
			return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
		}
		settled, err := s.Tasks.Settle(ctx, t.ID)
		if err != nil {
			log.Printf("[task %s] settle: %v", t.ID, err)
			return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
		}
		if !settled {

			if cur, err := s.Tasks.Get(ctx, t.ID); err == nil {
				if state := stateToProto(cur.State); state != stateToProto(t.State) {
					return state, true
				}
			}
			return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
		}
		return agencyv1.TaskState_TASK_STATE_SETTLED, true
	}
}

func (s *TaskServer) materializeStatus(
	ctx context.Context,
	agentID ids.AgentID,
	t tasks.Task,
	status agentv1.TaskStatus,
	errMsg string,
) {
	var err error
	switch status {
	case agentv1.TaskStatus_TASK_STATUS_WAITING_INPUT:
		_, err = s.Tasks.MarkInput(ctx, t.ID)
	case agentv1.TaskStatus_TASK_STATUS_IN_PROGRESS:

		if _, resumeErr := s.Tasks.Resume(ctx, t.ID); resumeErr != nil {
			err = resumeErr
			break
		}
		_, err = s.Tasks.ClearInput(ctx, t.ID)
	case agentv1.TaskStatus_TASK_STATUS_DONE:
		_, err = s.Tasks.Settle(ctx, t.ID)
	case agentv1.TaskStatus_TASK_STATUS_FAILED:

		msg := strings.TrimSpace(errMsg)
		if msg == "" {
			msg = tasks.RunFailedReason
		}
		err = s.Tasks.FailBySession(ctx, agentID, t.SessionID, msg)
	}
	if err != nil {
		log.Printf("[task %s] materialize status: %v", t.ID, err)
	}
}

func (s *TaskServer) checkAssignee(ctx context.Context, agentID ids.ActorID) error {
	if agentID == 0 {
		return nil
	}
	if _, err := s.Agents.GetByActor(ctx, agentID); err != nil {
		return mapAgentError(err)
	}
	return nil
}

func (s *TaskServer) wake() {
	if s.Scheduler != nil {
		s.Scheduler.Wake()
	}
}

func mapTemplateError(err error) *connect.Error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, templates.ErrTemplateMissing),
		errors.Is(err, templates.ErrVersionMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, templates.ErrArchived):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, projects.ErrForbidden),
		errors.Is(err, projects.ErrNotMember):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
}

func (s *TaskServer) canCloseOut(ctx context.Context, t tasks.Task) bool {
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return false
	}
	allowed, err := s.Tasks.CanCloseOut(ctx, t, actorID, auth.IsAdmin(ctx))
	return err == nil && allowed
}

func (s *TaskServer) canAnswerInput(ctx context.Context, t tasks.Task) bool {
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return false
	}
	allowed, err := s.Tasks.CanAnswerInput(ctx, t, actorID, auth.IsAdmin(ctx))
	return err == nil && allowed
}

func (s *TaskServer) canPrompt(ctx context.Context, t tasks.Task) bool {
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return false
	}
	allowed, err := s.Tasks.CanPrompt(ctx, t, actorID, auth.IsAdmin(ctx))
	return err == nil && allowed
}

func (s *TaskServer) requireCanCloseOut(ctx context.Context, t tasks.Task) error {
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return nil
	}
	allowed, err := s.Tasks.CanCloseOut(ctx, t, actorID, auth.IsAdmin(ctx))
	if err != nil {
		return mapError(err)
	}
	if !allowed {
		return connect.NewError(connect.CodePermissionDenied, tasks.ErrCannotCloseOut)
	}
	return nil
}

func (s *TaskServer) requireCanPrompt(ctx context.Context, t tasks.Task) error {
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return nil
	}
	allowed, err := s.Tasks.CanPrompt(ctx, t, actorID, auth.IsAdmin(ctx))
	if err != nil {
		return mapError(err)
	}
	if !allowed {
		return connect.NewError(connect.CodePermissionDenied, tasks.ErrCannotPrompt)
	}
	return nil
}

func mapError(err error) *connect.Error {
	switch {
	case errors.Is(err, tasks.ErrNotFound),
		errors.Is(err, tasks.ErrCommentNotFound),
		errors.Is(err, tasks.ErrHoldNotFound),
		errors.Is(err, projects.ErrProjectMissing),
		errors.Is(err, tags.ErrTagMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, tasks.ErrCommentForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, tasks.ErrInvalidPrompt),
		errors.Is(err, tasks.ErrInvalidModel),
		errors.Is(err, tasks.ErrProjectRequired),
		errors.Is(err, tasks.ErrSelfReference),
		errors.Is(err, tasks.ErrReferenceCycle),
		errors.Is(err, tasks.ErrLeadNotAllowed),
		errors.Is(err, tasks.ErrLeadRequired),
		errors.Is(err, tasks.ErrAutomatedLeader),
		errors.Is(err, tasks.ErrAutomatedBacklog),
		errors.Is(err, tasks.ErrCommentBodyRequired),
		errors.Is(err, tasks.ErrCommentAttachmentNotFound),
		errors.Is(err, tags.ErrInvalidName):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, tags.ErrTagExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, tasks.ErrNotEditable),
		errors.Is(err, tasks.ErrWrongState),
		errors.Is(err, tasks.ErrRunning),
		errors.Is(err, tasks.ErrPendingPrompt),
		errors.Is(err, tasks.ErrBacklogNotStarted),
		errors.Is(err, tasks.ErrNotCancelable),
		errors.Is(err, tasks.ErrLastPendingMessage),
		errors.Is(err, tasks.ErrFollowUpRequired),
		errors.Is(err, tasks.ErrHeld),
		errors.Is(err, tasks.ErrNotMember),
		errors.Is(err, tasks.ErrAssigneeViewer),
		errors.Is(err, tasks.ErrAssigneeMissingTag),
		errors.Is(err, tasks.ErrProjectHasNoAgents),
		errors.Is(err, tasks.ErrMergeProjectMismatch),
		errors.Is(err, tasks.ErrMergeAssigneeConflict),
		errors.Is(err, tasks.ErrMergeState),
		errors.Is(err, taskinput.ErrNoPendingDialog):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, tasks.ErrMergeTooFew):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func mapAgentError(err error) *connect.Error {
	if errors.Is(err, agents.ErrAgentMissing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func stateFromProto(s agencyv1.TaskState) tasks.State {
	switch s {
	case agencyv1.TaskState_TASK_STATE_BACKLOG:
		return tasks.StateBacklog
	case agencyv1.TaskState_TASK_STATE_WAITING:
		return tasks.StateWaiting
	case agencyv1.TaskState_TASK_STATE_QUEUED:
		return tasks.StateQueued
	case agencyv1.TaskState_TASK_STATE_RUNNING:
		return tasks.StateRunning
	case agencyv1.TaskState_TASK_STATE_INPUT:
		return tasks.StateInput
	case agencyv1.TaskState_TASK_STATE_SETTLED:
		return tasks.StateSettled
	case agencyv1.TaskState_TASK_STATE_FAILED:
		return tasks.StateFailed
	case agencyv1.TaskState_TASK_STATE_COMPLETED:
		return tasks.StateCompleted
	case agencyv1.TaskState_TASK_STATE_CANCELLED:
		return tasks.StateCancelled
	default:
		return ""
	}
}

func stateToProto(s tasks.State) agencyv1.TaskState {
	switch s {
	case tasks.StateBacklog:
		return agencyv1.TaskState_TASK_STATE_BACKLOG
	case tasks.StateWaiting:
		return agencyv1.TaskState_TASK_STATE_WAITING
	case tasks.StateQueued:
		return agencyv1.TaskState_TASK_STATE_QUEUED
	case tasks.StateRunning:
		return agencyv1.TaskState_TASK_STATE_RUNNING
	case tasks.StateInput:
		return agencyv1.TaskState_TASK_STATE_INPUT
	case tasks.StateSettled:
		return agencyv1.TaskState_TASK_STATE_SETTLED
	case tasks.StateFailed:
		return agencyv1.TaskState_TASK_STATE_FAILED
	case tasks.StateCompleted:
		return agencyv1.TaskState_TASK_STATE_COMPLETED
	case tasks.StateCancelled:
		return agencyv1.TaskState_TASK_STATE_CANCELLED
	default:
		return agencyv1.TaskState_TASK_STATE_UNSPECIFIED
	}
}

func attributesFromProto(
	setPriority *agencyv1.TaskPriority,
	setSize *agencyv1.TaskSize,
	setAssignee *ids.ActorID,
	setPrompt *string,
	setPendingPrompt *string,
	setModel *agencyv1.TaskModelOverride,
	addTagIDs, removeTagIDs []string,
) tasks.AttributeSet {
	var attrs tasks.AttributeSet
	if setPriority != nil {
		p := priorityFromProto(*setPriority)
		attrs.Priority = &p
	}
	if setSize != nil {
		sz := sizeFromProto(*setSize)
		attrs.Size = &sz
	}
	if setAssignee != nil {
		attrs.AssigneeActorID = setAssignee
	}
	if setPrompt != nil {
		attrs.Prompt = setPrompt
	}
	if setPendingPrompt != nil {
		attrs.PendingPrompt = setPendingPrompt
	}
	if setModel != nil {
		attrs.Model = &tasks.ModelOverride{
			Provider:      setModel.GetProvider(),
			ModelID:       setModel.GetModelId(),
			ThinkingLevel: setModel.GetThinkingLevel(),
		}
	}
	attrs.AddTagIDs = addTagIDs
	attrs.RemoveTagIDs = removeTagIDs
	return attrs
}

func priorityFromProto(p agencyv1.TaskPriority) int64 {
	switch p {
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

func priorityToProto(p int64) agencyv1.TaskPriority {
	switch {
	case p < tasks.PriorityNormal:
		return agencyv1.TaskPriority_TASK_PRIORITY_LOW
	case p >= tasks.PriorityCritical:
		return agencyv1.TaskPriority_TASK_PRIORITY_CRITICAL
	case p > tasks.PriorityNormal:
		return agencyv1.TaskPriority_TASK_PRIORITY_HIGH
	default:
		return agencyv1.TaskPriority_TASK_PRIORITY_NORMAL
	}
}

func sizeFromProto(s agencyv1.TaskSize) string {
	switch s {
	case agencyv1.TaskSize_TASK_SIZE_SMALL:
		return tasks.SizeSmall
	case agencyv1.TaskSize_TASK_SIZE_MEDIUM:
		return tasks.SizeMedium
	case agencyv1.TaskSize_TASK_SIZE_BIG:
		return tasks.SizeBig
	case agencyv1.TaskSize_TASK_SIZE_EPIC:
		return tasks.SizeEpic
	default:
		return ""
	}
}

func sizeToProto(s string) agencyv1.TaskSize {
	switch s {
	case tasks.SizeSmall:
		return agencyv1.TaskSize_TASK_SIZE_SMALL
	case tasks.SizeMedium:
		return agencyv1.TaskSize_TASK_SIZE_MEDIUM
	case tasks.SizeBig:
		return agencyv1.TaskSize_TASK_SIZE_BIG
	case tasks.SizeEpic:
		return agencyv1.TaskSize_TASK_SIZE_EPIC
	default:
		return agencyv1.TaskSize_TASK_SIZE_UNSPECIFIED
	}
}

func (s *TaskServer) enrichPendingMessages(
	ctx context.Context,
	taskID string,
	state tasks.State,
	out *agencyv1.Task,
) {
	queued, err := s.Tasks.PendingMessages(ctx, taskID)
	if err != nil || len(queued) == 0 {
		return
	}
	var byMessage map[string][]attachments.Attachment
	if s.Attachments != nil {
		if list, err := s.Attachments.TaskAttachments(ctx, taskID); err == nil {
			byMessage = map[string][]attachments.Attachment{}
			for _, a := range list {
				if a.MessageID != "" {
					byMessage[a.MessageID] = append(byMessage[a.MessageID], a)
				}
			}
		}
	}
	out.PendingMessages = make([]*agencyv1.TaskPendingMessage, len(queued))
	for i, message := range queued {
		pm := &agencyv1.TaskPendingMessage{
			Id:        message.ID,
			Text:      message.Text,
			ActorId:   actorid.Format(message.ActorID),
			CreatedAt: timestamppb.New(message.CreatedAt),
			CanCancel: tasks.CanCancelPendingMessage(state, queued, message),
		}
		for _, a := range byMessage[message.MessageID] {
			pm.Attachments = append(pm.Attachments, attachments.ToProto(a))
		}
		out.PendingMessages[i] = pm
	}
}

func toProtoTask(t tasks.Task) *agencyv1.Task {
	out := &agencyv1.Task{
		Id:                   t.ID,
		Seq:                  t.Seq,
		Name:                 t.Name,
		Prompt:               t.Prompt,
		ProjectId:            t.ProjectID,
		OriginTaskId:         t.OriginTaskID,
		LeaderActorId:        actorid.Format(t.LeaderActorID),
		NeedsOperator:        t.NeedsOperator,
		ReactionReminders:    int32(t.ReactionReminders),
		PendingSteering:      t.PendingSteering,
		AssigneeActorId:      actorid.Format(t.AssigneeActorID),
		AssigneePinned:       t.AssigneePinned,
		Priority:             priorityToProto(t.Priority),
		Size:                 sizeToProto(t.Size),
		State:                stateToProto(t.State),
		SessionId:            t.SessionID,
		PendingPrompt:        t.PendingPrompt,
		PendingPromptActorId: actorid.Format(t.PendingPromptActorID),
		Error:                t.Error,
		ModelProvider:        t.ModelProvider,
		ModelId:              t.ModelID,
		ThinkingLevel:        t.ThinkingLevel,
		Automated:            t.Automated,
		Workspace:            t.Workspace,
		AutoTitle:            t.AutoTitle,
		Archived:             t.Archived,
		Blockers:             int32(t.Blockers),
		Progress:             progressToProto(t.Progress),
		CreatorActorId:       actorid.Format(t.CreatorActorID),
		Subscribed:           t.Subscribed,
		Favorite:             t.Favorite,
		UnreadComments:       int32(t.UnreadComments),
		CreatedAt:            timestamppb.New(t.CreatedAt),
		UpdatedAt:            timestamppb.New(t.UpdatedAt),
	}
	for _, teamID := range t.TeamIDs {
		out.Teams = append(out.Teams, &agencyv1.TaskTeam{
			Id: int32(teamID),
		})
	}
	if t.LeaderActorID != 0 {
		out.Leader = &agencyv1.TaskLeader{
			ActorId: actorid.Format(t.LeaderActorID),
		}
	}
	for _, tag := range t.Tags {
		out.Tags = append(out.Tags, tagToProto(tag))
	}
	for _, link := range t.Links {
		out.Links = append(out.Links, &agencyv1.TaskLink{
			Url:   link.URL,
			Label: link.Label,
		})
	}
	for _, ref := range t.References {
		out.References = append(out.References, toProtoReference(ref))
	}
	for _, ref := range t.ReferencedBy {
		out.ReferencedBy = append(out.ReferencedBy, toProtoReference(ref))
	}
	for _, hold := range t.Holds {
		out.Holds = append(out.Holds, &agencyv1.TaskHold{
			Plugin:     hold.PluginID,
			PluginName: hold.PluginName,
			Key:        hold.Key,
			Reason:     hold.Reason,
			Url:        hold.URL,
			CreatedAt:  timeOrNil(hold.CreatedAt),
		})
	}
	out.QueuedAt = timeOrNil(t.QueuedAt)
	out.StartedAt = timeOrNil(t.StartedAt)
	out.SettledAt = timeOrNil(t.SettledAt)
	out.StateUpdatedAt = timeOrNil(t.StateUpdatedAt)
	out.TtcMs = t.TtcMs
	out.SessionDeletedAt = timeOrNil(t.SessionDeletedAt)
	out.ArchivedAt = timeOrNil(t.ArchivedAt)
	if t.Template != nil {
		out.Template = &agencyv1.TaskTemplateInfo{
			TemplateId:   t.Template.TemplateID,
			TemplateName: t.Template.TemplateName,
			Version:      t.Template.TemplateVersion,
			ArgsJson:     t.Template.ArgsJSON,
		}
	}
	return out
}

func progressToProto(p tasks.Progress) *agencyv1.TaskProgress {
	if p.Total == 0 && p.Summary == "" && p.UpdatedAt.IsZero() {
		return nil
	}
	out := &agencyv1.TaskProgress{
		Summary:   p.Summary,
		Done:      int32(p.Done),
		Total:     int32(p.Total),
		UpdatedAt: timeOrNil(p.UpdatedAt),
	}
	for _, item := range p.Items {
		out.Items = append(out.Items, &agencyv1.TaskProgressItem{
			Text:   item.Text,
			Status: agencyv1.TaskProgressItemStatus(item.Status),
		})
	}
	return out
}

func stripProgressItems(tasksOut []*agencyv1.Task) {
	for _, task := range tasksOut {
		if task.Progress != nil {
			task.Progress.Items = nil
		}
	}
}

func tagToProto(tag tasks.Tag) *agencyv1.Tag {
	return &agencyv1.Tag{
		Id:          tag.ID,
		ProjectId:   tag.ProjectID,
		Name:        tag.Name,
		Description: tag.Description,
		Color:       tag.Color,
		Archived:    tag.Archived,
		IsRequired:  tag.Required,
	}
}

func toProtoReference(ref tasks.Reference) *agencyv1.TaskReference {
	out := &agencyv1.TaskReference{
		Id:                    ref.ID,
		Name:                  ref.Name,
		Prompt:                ref.Prompt,
		State:                 stateToProto(ref.State),
		Kind:                  referenceKindToProto(ref.Kind),
		Priority:              priorityToProto(ref.Priority),
		Size:                  sizeToProto(ref.Size),
		WakeOnResolveWhenMany: ref.WakeOnResolveWhenMany,
	}
	for _, tag := range ref.Tags {
		out.Tags = append(out.Tags, tagToProto(tag))
	}
	return out
}

func referenceInputsFromProto(in []*agencyv1.TaskReferenceInput) []tasks.ReferenceInput {
	out := make([]tasks.ReferenceInput, 0, len(in))
	for _, ref := range in {
		out = append(out, tasks.ReferenceInput{
			TaskID:                ref.GetReferencedTaskId(),
			Kind:                  referenceKindFromProto(ref.GetKind()),
			WakeOnResolveWhenMany: ref.GetWakeOnResolveWhenMany(),
		})
	}
	return out
}

func holdReleaseModeFromProto(mode agencyv1.TaskHoldReleaseMode) tasks.ReleaseMode {
	switch mode {
	case agencyv1.TaskHoldReleaseMode_TASK_HOLD_RELEASE_MODE_QUEUE:
		return tasks.ReleaseQueue
	case agencyv1.TaskHoldReleaseMode_TASK_HOLD_RELEASE_MODE_COMPLETE:
		return tasks.ReleaseComplete
	default:
		return tasks.ReleaseSettle
	}
}

func referenceKindFromProto(kind agencyv1.TaskReferenceKind) tasks.ReferenceKind {
	switch kind {
	case agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_BLOCKING:
		return tasks.ReferenceBlocking
	case agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_BLOCKED_BY:
		return tasks.ReferenceBlockedBy
	case agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_BLOCKS:
		return tasks.ReferenceBlocks
	case agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_SUPERSEDES:
		return tasks.ReferenceSupersedes
	case agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_SUPERSEDED_BY:
		return tasks.ReferenceSupersededBy
	default:
		return tasks.ReferenceInformational
	}
}

func referenceKindToProto(kind tasks.ReferenceKind) agencyv1.TaskReferenceKind {
	switch kind {
	case tasks.ReferenceBlocking, tasks.ReferenceBlockedBy, tasks.ReferenceBlocks:
		return agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_BLOCKING
	case tasks.ReferenceSupersedes, tasks.ReferenceSupersededBy:
		return agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_SUPERSEDES
	default:
		return agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_INFORMATIONAL
	}
}

func toProtoLog(entry tasks.Log) *agencyv1.TaskLog {
	return &agencyv1.TaskLog{
		Id:       entry.ID,
		At:       timestamppb.New(entry.At),
		Actor:    entry.Actor,
		ActorId:  actorid.Format(entry.ActorID),
		Kind:     entry.Kind,
		Field:    entry.Field,
		OldValue: entry.OldValue,
		NewValue: entry.NewValue,
		Message:  entry.Message,
		DataJson: entry.DataJSON,
	}
}

func int32sToInt64s(ids []int32) []int64 {
	if ids == nil {
		return nil
	}
	out := make([]int64, len(ids))
	for i, id := range ids {
		out[i] = int64(id)
	}
	return out
}

func timeOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func taskRef(t tasks.Task) *agentv1.TaskRef {
	if t.ProjectID == "" || t.Seq == 0 {
		return nil
	}
	return &agentv1.TaskRef{ProjectId: t.ProjectID, Seq: t.Seq}
}
