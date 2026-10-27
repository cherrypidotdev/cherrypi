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
	"strconv"
	"time"

	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/artifacts"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/scheduler"
	"cherrypi.vladfaust.com/agency/internal/tags"
	"cherrypi.vladfaust.com/agency/internal/taskhistory"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	agentv1connect "cherrypi.vladfaust.com/shared/gen/buf/agent/v1/agentv1connect"
	"connectrpc.com/connect"
	"github.com/google/uuid"
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

	Actors func(ctx context.Context, id ids.ActorID) (ActorInfo, error)

	TeamName func(ctx context.Context, id int64) (string, error)
}

func (s *TaskServer) ListTasks(
	ctx context.Context,
	req *agencyv1.ListTasksRequest,
) (*agencyv1.ListTasksResponse, error) {
	list, err := s.Tasks.List(ctx, tasks.Filter{
		ProjectID: req.ProjectId,
		AgentID:   actorid.ParseAgent(req.AgentId),
		State:     stateFromProto(req.State),
		TagID:     req.TagId,
		Search:    req.Search,
		Size:      sizeFromProto(req.Size),
		Archived:  req.Archived,
	})
	if err != nil {
		return nil, mapError(err)
	}
	s.enrichNames(ctx, list)
	out := make([]*agencyv1.Task, len(list))
	for i, t := range list {
		out[i] = toProtoTask(t)
	}
	s.enrichRuntime(ctx, list, out)
	return &agencyv1.ListTasksResponse{Tasks: out}, nil
}

func (s *TaskServer) GetTask(
	ctx context.Context,
	req *agencyv1.GetTaskRequest,
) (*agencyv1.GetTaskResponse, error) {
	t, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	list := []tasks.Task{t}
	s.enrichNames(ctx, list)
	t = list[0]
	out := toProtoTask(t)
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
				out = toProtoTask(refreshed)
				out.Runtime = rt
			}
		}
	}
	s.enrichAttachments(ctx, out.Id, out)
	s.enrichAttribution(ctx, out)
	return &agencyv1.GetTaskResponse{Task: out}, nil
}

func (s *TaskServer) enrichAttachments(ctx context.Context, taskID string, out *agencyv1.Task) {
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
	cache := map[ids.ActorID]ActorInfo{}
	actor := func(raw string) (ActorInfo, bool) {
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
	if info, ok := actor(out.CreatorActorId); ok {
		out.CreatorName = displayActorName(info)
		out.CreatorPictureUrl = info.PictureURL
	}
	if info, ok := actor(out.PendingPromptActorId); ok {
		out.PendingPromptActorName = displayActorName(info)
	}
	if s.TeamName != nil {
		for _, team := range out.Teams {
			if name, err := s.TeamName(ctx, int64(team.Id)); err == nil {
				team.Name = name
			}
		}
	}
	for _, lead := range out.Leads {
		if info, ok := actor(lead.ActorId); ok {
			lead.Username = info.Username
			lead.Name = displayActorName(info)
			lead.Kind = info.Kind
			lead.PictureUrl = info.PictureURL
		}
	}
	for _, message := range out.Messages {
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
	if req.ModelProvider != "" || req.ModelId != "" || req.ThinkingLevel != "" {
		if req.AssigneeActorId == "" {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("assign an agent before choosing a model"),
			)
		}
	}
	t, err := s.Tasks.Create(ctx, tasks.CreateParams{
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
		Workspace:       req.Workspace,
		AutoTitle:       req.AutoTitle,
		State:           stateFromProto(req.State),
		TeamIDs:         int32sToInt64s(req.TeamIds),
		LeadActorIDs:    actorIDsFromStrings(req.LeadActorIds),
	})
	if err != nil {
		return nil, mapError(err)
	}
	if t.State == tasks.StateQueued {
		s.wake()
	}
	return &agencyv1.CreateTaskResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) UpdateTask(
	ctx context.Context,
	req *agencyv1.UpdateTaskRequest,
) (*agencyv1.UpdateTaskResponse, error) {
	if attrs := attributesFromProto(
		req.SetPriority,
		req.SetSize,
		actorid.ParseActorPtr(req.SetAssigneeActorId),
		req.SetPrompt,
		req.SetPendingPrompt,
		nil,
		nil,
	); !attrs.Empty() {
		if req.Prompt != "" || req.ProjectId != "" {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				errors.New(
					"an attribute update must not include the draft prompt or project fields",
				),
			)
		}
		if err := s.checkAssignee(
			ctx,
			actorid.ParseActor(req.GetSetAssigneeActorId()),
		); err != nil {
			return nil, err
		}
		t, err := s.Tasks.UpdateAttributes(ctx, req.Id, attrs)
		if err != nil {
			return nil, mapError(err)
		}

		s.wake()
		return &agencyv1.UpdateTaskResponse{Task: toProtoTask(t)}, nil
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
	return &agencyv1.UpdateTaskResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) QueueTask(
	ctx context.Context,
	req *agencyv1.QueueTaskRequest,
) (*agencyv1.QueueTaskResponse, error) {
	t, err := s.Tasks.Queue(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	s.wake()
	return &agencyv1.QueueTaskResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) ListTaskReferences(
	ctx context.Context,
	req *agencyv1.ListTaskReferencesRequest,
) (*agencyv1.ListTaskReferencesResponse, error) {
	refs, err := s.Tasks.References(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	s.enrichNames(ctx, refs)
	out := make([]*agencyv1.Task, len(refs))
	for i, t := range refs {
		out[i] = toProtoTask(t)
	}
	s.enrichRuntime(ctx, refs, out)
	return &agencyv1.ListTaskReferencesResponse{Tasks: out}, nil
}

func (s *TaskServer) AddTaskReference(
	ctx context.Context,
	req *agencyv1.AddTaskReferenceRequest,
) (*agencyv1.AddTaskReferenceResponse, error) {
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
	return &agencyv1.AddTaskReferenceResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) RemoveTaskReference(
	ctx context.Context,
	req *agencyv1.RemoveTaskReferenceRequest,
) (*agencyv1.RemoveTaskReferenceResponse, error) {
	t, err := s.Tasks.RemoveReference(ctx, req.Id, req.ReferencedTaskId)
	if err != nil {
		return nil, mapError(err)
	}

	s.wake()
	return &agencyv1.RemoveTaskReferenceResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) PromoteTask(
	ctx context.Context,
	req *agencyv1.PromoteTaskRequest,
) (*agencyv1.PromoteTaskResponse, error) {
	t, err := s.Tasks.Promote(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.PromoteTaskResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) DemoteTask(
	ctx context.Context,
	req *agencyv1.DemoteTaskRequest,
) (*agencyv1.DemoteTaskResponse, error) {
	t, err := s.Tasks.Demote(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DemoteTaskResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) DequeueTask(
	ctx context.Context,
	req *agencyv1.DequeueTaskRequest,
) (*agencyv1.DequeueTaskResponse, error) {
	t, err := s.Tasks.Dequeue(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DequeueTaskResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) CloseOutTask(
	ctx context.Context,
	req *agencyv1.CloseOutTaskRequest,
) (*agencyv1.CloseOutTaskResponse, error) {
	var (
		t   tasks.Task
		err error
	)
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
	return &agencyv1.CloseOutTaskResponse{Task: toProtoTask(t)}, nil
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
	return &agencyv1.ReopenTaskResponse{Task: toProtoTask(t)}, nil
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
	return &agencyv1.MergeTasksResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) ArchiveTask(
	ctx context.Context,
	req *agencyv1.ArchiveTaskRequest,
) (*agencyv1.ArchiveTaskResponse, error) {
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
	return &agencyv1.ArchiveTaskResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) UnarchiveTask(
	ctx context.Context,
	req *agencyv1.UnarchiveTaskRequest,
) (*agencyv1.UnarchiveTaskResponse, error) {
	t, err := s.Tasks.Unarchive(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	s.wake()
	return &agencyv1.UnarchiveTaskResponse{Task: toProtoTask(t)}, nil
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
	return &agencyv1.CleanupTaskSessionResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) cleanupTaskSession(ctx context.Context, t tasks.Task) error {
	if t.SessionID == "" || !t.SessionDeletedAt.IsZero() {
		return nil
	}
	if t.AssigneeActorID == 0 {
		return errors.New("task has no assigned agent")
	}
	conn, err := s.connectionForActor(ctx, t.AssigneeActorID)
	if err != nil {
		return err
	}
	if _, err := conn.Tasks.DeleteTask(
		ctx,
		&agentv1.DeleteTaskRequest{Id: t.SessionID},
	); err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		return fmt.Errorf("delete agent session: %w", err)
	}
	if _, err := s.Tasks.MarkSessionDeleted(ctx, t.ID); err != nil {
		return err
	}
	return nil
}

func (s *TaskServer) RenameTask(
	ctx context.Context,
	req *agencyv1.RenameTaskRequest,
) (*agencyv1.RenameTaskResponse, error) {
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
	return &agencyv1.RenameTaskResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) SetTaskPriority(
	ctx context.Context,
	req *agencyv1.SetTaskPriorityRequest,
) (*agencyv1.SetTaskPriorityResponse, error) {
	t, err := s.Tasks.SetPriority(ctx, req.Id, priorityFromProto(req.Priority))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.SetTaskPriorityResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) SetTaskSize(
	ctx context.Context,
	req *agencyv1.SetTaskSizeRequest,
) (*agencyv1.SetTaskSizeResponse, error) {
	t, err := s.Tasks.SetSize(ctx, req.Id, sizeFromProto(req.Size))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.SetTaskSizeResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) SetTaskLinks(
	ctx context.Context,
	req *agencyv1.SetTaskLinksRequest,
) (*agencyv1.SetTaskLinksResponse, error) {
	links := make([]tasks.Link, len(req.GetLinks()))
	for i, link := range req.GetLinks() {
		links[i] = tasks.Link{URL: link.GetUrl(), Label: link.GetLabel()}
	}
	t, err := s.Tasks.SetLinks(ctx, req.Id, links)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.SetTaskLinksResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) SetTaskTeams(
	ctx context.Context,
	req *agencyv1.SetTaskTeamsRequest,
) (*agencyv1.SetTaskTeamsResponse, error) {
	t, err := s.Tasks.SetTaskTeams(ctx, req.Id, int32sToInt64s(req.TeamIds))
	if err != nil {
		return nil, mapError(err)
	}
	out := toProtoTask(t)
	s.enrichAttribution(ctx, out)
	return &agencyv1.SetTaskTeamsResponse{Task: out}, nil
}

func (s *TaskServer) SetTaskLeads(
	ctx context.Context,
	req *agencyv1.SetTaskLeadsRequest,
) (*agencyv1.SetTaskLeadsResponse, error) {
	t, err := s.Tasks.SetTaskLeads(ctx, req.Id, actorIDsFromStrings(req.LeadActorIds))
	if err != nil {
		return nil, mapError(err)
	}
	out := toProtoTask(t)
	s.enrichAttribution(ctx, out)
	return &agencyv1.SetTaskLeadsResponse{Task: out}, nil
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
	out := toProtoTask(t)
	s.enrichAttribution(ctx, out)
	return &agencyv1.SetTaskSubscriptionResponse{Task: out}, nil
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
	results := s.Tasks.UpdateManyAttributes(ctx, req.GetIds(), attrs)
	out := make([]*agencyv1.BulkUpdateTaskResult, 0, len(results))
	for _, r := range results {
		item := &agencyv1.BulkUpdateTaskResult{Id: r.ID}
		if r.Error != nil {
			item.Error = r.Error.Error()
		} else {
			item.Task = toProtoTask(r.Task)
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

	messageID := req.MessageId
	if messageID == "" {
		messageID = uuid.NewString()
	}

	if !t.SessionDeletedAt.IsZero() {
		if _, err := s.Tasks.RequeueWithPrompt(ctx, t.ID, req.Prompt, actorID); err != nil {
			return nil, mapError(err)
		}
		s.recordMessage(ctx, t.ID, messageID, req.Prompt, actorID, req.AttachmentIds)
		s.wake()
		return &agencyv1.AddToTaskResponse{MessageId: messageID}, nil
	}

	switch t.State {
	case tasks.StateCompleted, tasks.StateCancelled,
		tasks.StateSettled, tasks.StateFailed:
		if _, err := s.Tasks.RequeueWithPrompt(ctx, t.ID, req.Prompt, actorID); err != nil {
			return nil, mapError(err)
		}
		s.recordMessage(ctx, t.ID, messageID, req.Prompt, actorID, req.AttachmentIds)
		s.wake()
		return &agencyv1.AddToTaskResponse{MessageId: messageID}, nil
	}

	if t.State != tasks.StateRunning {
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			errors.New("task has no running session"),
		)
	}
	conn, err := s.connectionForActor(ctx, t.AssigneeActorID)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Tasks.AddToTask(ctx, &agentv1.AddToTaskRequest{
		Id:          t.SessionID,
		Prompt:      req.Prompt,
		Attachments: s.attachmentRefs(ctx, req.AttachmentIds),
		TaskId:      t.ID,
		Ref:         taskRef(t),
		Sender:      s.promptSender(ctx, actorID),
	}); err != nil {
		return nil, err
	}
	s.recordMessage(ctx, t.ID, messageID, req.Prompt, actorID, req.AttachmentIds)
	return &agencyv1.AddToTaskResponse{MessageId: messageID}, nil
}

func (s *TaskServer) promptSender(
	ctx context.Context,
	actorID ids.ActorID,
) *agentv1.PromptSender {
	sender, err := s.Tasks.Sender(ctx, actorID)
	if err != nil {
		return nil
	}
	return &agentv1.PromptSender{
		ActorId:  int64(sender.ActorID),
		Username: sender.Username,
		Name:     sender.Name,
		Kind:     sender.Kind,
	}
}

func (s *TaskServer) attachmentRefs(
	ctx context.Context,
	ids []int64,
) []*agentv1.AttachmentRef {
	if s.Attachments == nil || len(ids) == 0 {
		return nil
	}
	refs := make([]*agentv1.AttachmentRef, 0, len(ids))
	for _, id := range ids {
		a, err := s.Attachments.Get(ctx, id)
		if err != nil {
			continue
		}
		refs = append(refs, &agentv1.AttachmentRef{
			Id:          a.ID,
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Size:        a.Size,
			Url:         attachments.URLFor(a.ID),
		})
	}
	return refs
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
	t, conn, err := s.activeTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Tasks.StopTask(ctx, &agentv1.StopTaskRequest{Id: t.SessionID}); err != nil {
		return nil, err
	}
	return &agencyv1.StopTaskResponse{}, nil
}

func (s *TaskServer) RetryTask(
	ctx context.Context,
	req *agencyv1.RetryTaskRequest,
) (*agencyv1.RetryTaskResponse, error) {
	t, conn, err := s.activeTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Tasks.RetryTask(ctx, &agentv1.RetryTaskRequest{
		Id:  t.SessionID,
		Ref: taskRef(t),
	}); err != nil {
		return nil, err
	}
	if t.State == tasks.StateSettled || t.State == tasks.StateFailed {
		if _, err := s.Tasks.Resume(ctx, t.ID); err != nil {
			log.Printf("[task %s] resume: %v", t.ID, err)
		}
		t, _ = s.Tasks.Get(ctx, t.ID)
	}
	return &agencyv1.RetryTaskResponse{Task: toProtoTask(t)}, nil
}

func (s *TaskServer) WatchTask(
	ctx context.Context,
	req *agencyv1.WatchTaskRequest,
	stream *connect.ServerStream[agencyv1.WatchTaskResponse],
) error {
	t, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return mapError(err)
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
				t.ID,
				agentv1.TaskStatus_TASK_STATUS_IN_PROGRESS,
			)
		case event.GetRunSettled() != nil:
			if _, err := s.Tasks.Settle(ctx, t.ID); err != nil {
				log.Printf("[task %s] settle: %v", t.ID, err)
			}
		case event.GetStatusChanged() != nil:
			s.materializeStatus(ctx, t.ID, event.GetStatusChanged().GetStatus())
		}
		if err := stream.Send(
			&agencyv1.WatchTaskResponse{Event: event},
		); err != nil {
			return err
		}
	}
	return inner.Err()
}

func (s *TaskServer) GetTaskHistory(
	ctx context.Context,
	req *agencyv1.GetTaskHistoryRequest,
) (*agencyv1.GetTaskHistoryResponse, error) {
	t, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, mapError(err)
	}
	entries, err := s.Tasks.SessionHistory(ctx, t.ID)
	if err != nil {
		return nil, mapError(err)
	}
	if len(entries) > 0 {
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
		events, next := taskhistory.Page(t.ID, entries, before, int(req.Limit))
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
		return nil, err
	}
	return &agencyv1.GetTaskHistoryResponse{
		Events:     res.Events,
		NextBefore: res.NextBefore,
	}, nil
}

func (s *TaskServer) GetTaskConfig(
	ctx context.Context,
	req *agencyv1.GetTaskConfigRequest,
) (*agencyv1.GetTaskConfigResponse, error) {
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
	t, conn, err := s.sessionTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	inner := &agentv1.RespondToExtensionUiRequest{
		Id:        t.SessionID,
		RequestId: req.RequestId,
	}
	switch resp := req.Response.(type) {
	case *agencyv1.RespondToExtensionUiRequest_Value:
		inner.Response = &agentv1.RespondToExtensionUiRequest_Value{Value: resp.Value}
	case *agencyv1.RespondToExtensionUiRequest_Confirmed:
		inner.Response = &agentv1.RespondToExtensionUiRequest_Confirmed{
			Confirmed: resp.Confirmed,
		}
	case *agencyv1.RespondToExtensionUiRequest_Cancelled:
		inner.Response = &agentv1.RespondToExtensionUiRequest_Cancelled{
			Cancelled: resp.Cancelled,
		}
	}
	if len(req.Questionnaire) > 0 {
		inner.Questionnaire = make([]*agentv1.QuestionnaireAnswer, 0, len(req.Questionnaire))
		for _, qa := range req.Questionnaire {
			inner.Questionnaire = append(inner.Questionnaire, &agentv1.QuestionnaireAnswer{
				Question: qa.Question,
				Answer:   qa.Answer,
			})
		}
	}
	if _, err := conn.Tasks.RespondToExtensionUi(ctx, inner); err != nil {
		return nil, err
	}

	if _, err := s.Tasks.ClearInput(ctx, t.ID); err != nil {
		log.Printf("[task %s] clear input: %v", t.ID, err)
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
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return nil
	}
	allowed, err := s.Tasks.CanPrompt(ctx, t, actorID, auth.IsAdmin(ctx))
	if err != nil {
		return mapError(err)
	}
	if !allowed {
		return connect.NewError(
			connect.CodePermissionDenied,
			errors.New("not allowed to change this task's model"),
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
	if current.AssigneeActorID == 0 &&
		(provider != "" || modelID != "" || thinkingLevel != "") {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("assign an agent before choosing a model"),
		)
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
		if _, err := s.Tasks.Settle(ctx, t.ID); err != nil {
			log.Printf("[task %s] settle: %v", t.ID, err)
			return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
		}
		return agencyv1.TaskState_TASK_STATE_SETTLED, true
	}
}

func (s *TaskServer) materializeStatus(
	ctx context.Context,
	id string,
	status agentv1.TaskStatus,
) {
	var err error
	switch status {
	case agentv1.TaskStatus_TASK_STATUS_WAITING_INPUT:
		_, err = s.Tasks.MarkInput(ctx, id)
	case agentv1.TaskStatus_TASK_STATUS_IN_PROGRESS:

		if _, resumeErr := s.Tasks.Resume(ctx, id); resumeErr != nil {
			err = resumeErr
			break
		}
		_, err = s.Tasks.ClearInput(ctx, id)
	case agentv1.TaskStatus_TASK_STATUS_DONE:
		_, err = s.Tasks.Settle(ctx, id)
	}
	if err != nil {
		log.Printf("[task %s] materialize status: %v", id, err)
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

func mapError(err error) *connect.Error {
	switch {
	case errors.Is(err, tasks.ErrNotFound),
		errors.Is(err, projects.ErrProjectMissing),
		errors.Is(err, tags.ErrTagMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, tasks.ErrInvalidPrompt),
		errors.Is(err, tasks.ErrProjectRequired),
		errors.Is(err, tasks.ErrSelfReference),
		errors.Is(err, tasks.ErrReferenceCycle),
		errors.Is(err, tasks.ErrLeadNotAllowed),
		errors.Is(err, tags.ErrInvalidName):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, tags.ErrTagExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, tasks.ErrNotDraft),
		errors.Is(err, tasks.ErrWrongState),
		errors.Is(err, tasks.ErrRunning),
		errors.Is(err, tasks.ErrNotMember),
		errors.Is(err, tasks.ErrProjectHasNoAgents),
		errors.Is(err, tasks.ErrMergeProjectMismatch),
		errors.Is(err, tasks.ErrMergeAssigneeConflict),
		errors.Is(err, tasks.ErrMergeState):
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
	case agencyv1.TaskState_TASK_STATE_DRAFT:
		return tasks.StateDraft
	case agencyv1.TaskState_TASK_STATE_BACKLOG:
		return tasks.StateBacklog
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
	case tasks.StateDraft:
		return agencyv1.TaskState_TASK_STATE_DRAFT
	case tasks.StateBacklog:
		return agencyv1.TaskState_TASK_STATE_BACKLOG
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

func toProtoTask(t tasks.Task) *agencyv1.Task {
	out := &agencyv1.Task{
		Id:                   t.ID,
		Seq:                  t.Seq,
		Name:                 t.Name,
		Prompt:               t.Prompt,
		ProjectId:            t.ProjectID,
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
		Interactive:          t.Interactive,
		Workspace:            t.Workspace,
		AutoTitle:            t.AutoTitle,
		Archived:             t.Archived,
		Blockers:             int32(t.Blockers),
		CreatorActorId:       actorid.Format(t.CreatorActorID),
		Subscribed:           t.Subscribed,
		CreatedAt:            timestamppb.New(t.CreatedAt),
		UpdatedAt:            timestamppb.New(t.UpdatedAt),
	}
	for _, teamID := range t.TeamIDs {
		out.Teams = append(out.Teams, &agencyv1.TaskTeam{
			Id: int32(teamID),
		})
	}
	for _, leadID := range t.LeadActorIDs {
		out.Leads = append(out.Leads, &agencyv1.TaskLead{
			ActorId: actorid.Format(leadID),
		})
	}
	for _, tag := range t.Tags {
		out.Tags = append(out.Tags, &agencyv1.Tag{
			Id:          tag.ID,
			ProjectId:   tag.ProjectID,
			Name:        tag.Name,
			Description: tag.Description,
			Color:       tag.Color,
			Archived:    tag.Archived,
		})
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
	out.QueuedAt = timeOrNil(t.QueuedAt)
	out.StartedAt = timeOrNil(t.StartedAt)
	out.SettledAt = timeOrNil(t.SettledAt)
	out.StateUpdatedAt = timeOrNil(t.StateUpdatedAt)
	out.SessionDeletedAt = timeOrNil(t.SessionDeletedAt)
	out.ArchivedAt = timeOrNil(t.ArchivedAt)
	return out
}

func toProtoReference(ref tasks.Reference) *agencyv1.TaskReference {
	return &agencyv1.TaskReference{
		Id:     ref.ID,
		Name:   ref.Name,
		Prompt: ref.Prompt,
		State:  stateToProto(ref.State),
		Kind:   referenceKindToProto(ref.Kind),
	}
}

func referenceInputsFromProto(in []*agencyv1.TaskReferenceInput) []tasks.ReferenceInput {
	out := make([]tasks.ReferenceInput, 0, len(in))
	for _, ref := range in {
		out = append(out, tasks.ReferenceInput{
			TaskID: ref.GetReferencedTaskId(),
			Kind:   referenceKindFromProto(ref.GetKind()),
		})
	}
	return out
}

func referenceKindFromProto(kind agencyv1.TaskReferenceKind) tasks.ReferenceKind {
	switch kind {
	case agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_BLOCKING:
		return tasks.ReferenceBlocking
	case agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_SUPERSEDES:
		return tasks.ReferenceSupersedes
	default:
		return tasks.ReferenceInformational
	}
}

func referenceKindToProto(kind tasks.ReferenceKind) agencyv1.TaskReferenceKind {
	switch kind {
	case tasks.ReferenceBlocking:
		return agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_BLOCKING
	case tasks.ReferenceSupersedes:
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

func actorIDsFromStrings(in []string) []ids.ActorID {
	out := make([]ids.ActorID, 0, len(in))
	for _, raw := range in {
		if id := actorid.ParseActor(raw); id != 0 {
			out = append(out, id)
		}
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
