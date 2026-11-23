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
	"context"
	"database/sql"
	"errors"
	"time"

	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
	"cherrypi.vladfaust.com/agent/internal/session"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type taskRow = sqlitedb.Task

type TaskServer struct {
	Tasks *session.Manager
}

func (s *TaskServer) CreateTask(
	ctx context.Context,
	req *agentv1.CreateTaskRequest,
) (*agentv1.CreateTaskResponse, error) {
	row, err := s.Tasks.Create(ctx, req.Prompt, session.CreateOptions{
		Workspace:          req.Workspace,
		ModelProvider:      req.ModelProvider,
		ModelID:            req.ModelId,
		ThinkingLevel:      req.ThinkingLevel,
		Interactive:        req.Interactive,
		Title:              req.Title,
		AutoTitle:          req.AutoTitle,
		AppendSystemPrompt: req.AppendSystemPrompt,
		Attachments:        sessionAttachments(req.Attachments),
		AgencyTaskID:       req.TaskId,
		AgencyProjectID:    req.Ref.GetProjectId(),
		AgencySeq:          req.Ref.GetSeq(),
		Sender:             sessionSender(req.Sender),
		PluginTools:        sessionPluginTools(req.Tools),
		RequiredTools:      req.RequiredTools,
	})
	if err != nil {
		return nil, connectError(err)
	}

	return &agentv1.CreateTaskResponse{Task: s.toProtoTask(row)}, nil
}

func (s *TaskServer) AddToTask(
	ctx context.Context,
	req *agentv1.AddToTaskRequest,
) (*agentv1.AddToTaskResponse, error) {
	if err := s.Tasks.Add(ctx, req.Id, req.Prompt, session.AddOptions{
		Attachments:     sessionAttachments(req.Attachments),
		AgencyTaskID:    req.TaskId,
		AgencyProjectID: req.Ref.GetProjectId(),
		AgencySeq:       req.Ref.GetSeq(),
		Sender:          sessionSender(req.Sender),
		RequiredTools:   req.RequiredTools,
		Model: session.ModelSelection{
			Provider:      req.ModelProvider,
			ModelID:       req.ModelId,
			ThinkingLevel: req.ThinkingLevel,
		},
	}); err != nil {
		return nil, connectError(err)
	}

	return &agentv1.AddToTaskResponse{}, nil
}

func sessionAttachments(refs []*agentv1.AttachmentRef) []session.Attachment {
	if len(refs) == 0 {
		return nil
	}
	out := make([]session.Attachment, 0, len(refs))
	for _, ref := range refs {
		out = append(out, session.Attachment{
			ID:          ref.Id,
			Filename:    ref.Filename,
			ContentType: ref.ContentType,
			Size:        ref.Size,
			URL:         ref.Url,
		})
	}
	return out
}

func sessionPluginTools(tools []*agentv1.PluginTool) []session.PluginTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]session.PluginTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, session.PluginTool{
			Name:        tool.GetName(),
			Description: tool.GetDescription(),
			InputSchema: tool.GetInputSchema(),
		})
	}
	return out
}

func sessionSender(sender *agentv1.PromptSender) *session.PromptSender {
	if sender == nil {
		return nil
	}
	return &session.PromptSender{
		ActorID:  sender.GetActorId(),
		Username: sender.GetUsername(),
		Name:     sender.GetName(),
		Kind:     sender.GetKind(),
	}
}

func (s *TaskServer) RetryTask(
	ctx context.Context,
	req *agentv1.RetryTaskRequest,
) (*agentv1.RetryTaskResponse, error) {
	if err := s.Tasks.Retry(
		ctx, req.Id, req.Ref.GetProjectId(), req.Ref.GetSeq(), req.RequiredTools,
		session.ModelSelection{
			Provider:      req.ModelProvider,
			ModelID:       req.ModelId,
			ThinkingLevel: req.ThinkingLevel,
		},
	); err != nil {
		return nil, connectError(err)
	}

	row, err := s.Tasks.Get(ctx, req.Id)
	if err != nil {
		return nil, connectError(err)
	}

	return &agentv1.RetryTaskResponse{Task: s.toProtoTask(row)}, nil
}

func (s *TaskServer) StopTask(
	ctx context.Context,
	req *agentv1.StopTaskRequest,
) (*agentv1.StopTaskResponse, error) {
	if err := s.Tasks.Stop(ctx, req.Id, req.Actor); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.StopTaskResponse{}, nil
}

func (s *TaskServer) InterruptToolCall(
	ctx context.Context,
	req *agentv1.InterruptToolCallRequest,
) (*agentv1.InterruptToolCallResponse, error) {
	if err := s.Tasks.InterruptToolCall(
		ctx,
		req.Id,
		req.ToolCallId,
		req.Actor,
	); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.InterruptToolCallResponse{}, nil
}

func (s *TaskServer) DeleteTask(
	ctx context.Context,
	req *agentv1.DeleteTaskRequest,
) (*agentv1.DeleteTaskResponse, error) {
	if err := s.Tasks.Delete(ctx, req.Id); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.DeleteTaskResponse{}, nil
}

func (s *TaskServer) RenameTask(
	ctx context.Context,
	req *agentv1.RenameTaskRequest,
) (*agentv1.RenameTaskResponse, error) {
	row, err := s.Tasks.Rename(ctx, req.Id, req.Name)
	if err != nil {
		return nil, connectError(err)
	}

	return &agentv1.RenameTaskResponse{Task: s.toProtoTask(row)}, nil
}

func (s *TaskServer) ListTasks(
	ctx context.Context,
	_ *agentv1.ListTasksRequest,
) (*agentv1.ListTasksResponse, error) {
	rows, err := s.Tasks.List(ctx)
	if err != nil {
		return nil, connectError(err)
	}

	out := make([]*agentv1.Task, len(rows))
	for i, row := range rows {
		out[i] = s.toProtoTask(row)
	}

	return &agentv1.ListTasksResponse{Tasks: out}, nil
}

func (s *TaskServer) WatchTask(
	ctx context.Context,
	req *agentv1.WatchTaskRequest,
	stream *connect.ServerStream[agentv1.WatchTaskResponse],
) error {
	ch, unsub, err := s.Tasks.Watch(ctx, req.Id, req.ReplayHistory)
	if err != nil {
		return connectError(err)
	}
	defer unsub()

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			msg := &agentv1.WatchTaskResponse{Event: session.TaskEventProto(req.Id, ev)}
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}
}

func (s *TaskServer) GetTaskHistory(
	ctx context.Context,
	req *agentv1.GetTaskHistoryRequest,
) (*agentv1.GetTaskHistoryResponse, error) {
	events, nextBefore, err := s.Tasks.History(ctx, req.Id, req.Before, int(req.Limit))
	if err != nil {
		return nil, connectError(err)
	}

	out := make([]*agentv1.TaskEvent, len(events))
	for i, ev := range events {
		out[i] = session.TaskEventProto(req.Id, ev)
	}

	return &agentv1.GetTaskHistoryResponse{Events: out, NextBefore: nextBefore}, nil
}

func (s *TaskServer) GetTaskConfig(
	ctx context.Context,
	req *agentv1.GetTaskConfigRequest,
) (*agentv1.GetTaskConfigResponse, error) {
	cfg, err := s.Tasks.Config(ctx, req.Id)
	if err != nil {
		return nil, connectError(err)
	}
	return &agentv1.GetTaskConfigResponse{Config: toProtoConfig(cfg)}, nil
}

func (s *TaskServer) GetTaskDefaults(
	ctx context.Context,
	_ *agentv1.GetTaskDefaultsRequest,
) (*agentv1.GetTaskDefaultsResponse, error) {
	cfg, err := s.Tasks.Defaults(ctx)
	if err != nil {
		return nil, connectError(err)
	}
	return &agentv1.GetTaskDefaultsResponse{Defaults: toProtoConfig(cfg)}, nil
}

func (s *TaskServer) SetTaskModel(
	ctx context.Context,
	req *agentv1.SetTaskModelRequest,
) (*agentv1.SetTaskModelResponse, error) {
	cfg, err := s.Tasks.SetModel(ctx, req.Id, req.Provider, req.ModelId)
	if err != nil {
		return nil, connectError(err)
	}
	return &agentv1.SetTaskModelResponse{Config: toProtoConfig(cfg)}, nil
}

func (s *TaskServer) SetTaskThinkingLevel(
	ctx context.Context,
	req *agentv1.SetTaskThinkingLevelRequest,
) (*agentv1.SetTaskThinkingLevelResponse, error) {
	cfg, err := s.Tasks.SetThinkingLevel(ctx, req.Id, req.Level)
	if err != nil {
		return nil, connectError(err)
	}
	return &agentv1.SetTaskThinkingLevelResponse{Config: toProtoConfig(cfg)}, nil
}

func (s *TaskServer) RespondToExtensionUi(
	ctx context.Context,
	req *agentv1.RespondToExtensionUiRequest,
) (*agentv1.RespondToExtensionUiResponse, error) {
	var resp session.ExtensionResponse
	switch r := req.Response.(type) {
	case *agentv1.RespondToExtensionUiRequest_Value:
		resp = session.ExtensionResponse{Kind: session.ExtensionResponseValue, Value: r.Value}
	case *agentv1.RespondToExtensionUiRequest_Confirmed:
		resp = session.ExtensionResponse{
			Kind:      session.ExtensionResponseConfirmed,
			Confirmed: r.Confirmed,
		}
	default:
		resp = session.ExtensionResponse{Kind: session.ExtensionResponseCancelled}
	}
	if len(req.Questionnaire) > 0 {
		resp.Questionnaire = make([]session.QuestionnaireAnswer, 0, len(req.Questionnaire))
		for _, qa := range req.Questionnaire {
			resp.Questionnaire = append(resp.Questionnaire, session.QuestionnaireAnswer{
				Question: qa.Question,
				Answer:   qa.Answer,
			})
		}
	}

	if err := s.Tasks.RespondExtensionUI(ctx, req.Id, req.RequestId, resp); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.RespondToExtensionUiResponse{}, nil
}

func (s *TaskServer) ClearQueue(
	ctx context.Context,
	req *agentv1.ClearQueueRequest,
) (*agentv1.ClearQueueResponse, error) {
	if err := s.Tasks.ClearQueue(ctx, req.Id); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.ClearQueueResponse{}, nil
}

func (s *TaskServer) RemoveQueuedMessage(
	ctx context.Context,
	req *agentv1.RemoveQueuedMessageRequest,
) (*agentv1.RemoveQueuedMessageResponse, error) {
	if err := s.Tasks.RemoveQueued(ctx, req.Id, req.QueueId); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.RemoveQueuedMessageResponse{}, nil
}

func (s *TaskServer) RestartTaskSession(
	ctx context.Context,
	req *agentv1.RestartTaskSessionRequest,
) (*agentv1.RestartTaskSessionResponse, error) {
	if err := s.Tasks.Restart(ctx, req.Id); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.RestartTaskSessionResponse{}, nil
}

func (s *TaskServer) CompactTask(
	ctx context.Context,
	req *agentv1.CompactTaskRequest,
) (*agentv1.CompactTaskResponse, error) {
	if err := s.Tasks.Compact(ctx, req.Id); err != nil {
		return nil, connectError(err)
	}
	return &agentv1.CompactTaskResponse{}, nil
}

func connectError(err error) error {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, session.ErrSessionBusy):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, session.ErrNothingToCompact):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, session.ErrCompactionDisabled):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, session.ErrInvalidPrompt):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, session.ErrInvalidName):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, session.ErrNoPendingDialog):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, session.ErrToolCallNotRunning):

		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, session.ErrInvalidCursor):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, session.ErrInvalidModel):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, session.ErrInvalidThinkingLevel):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, session.ErrShuttingDown):

		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func (s *TaskServer) toProtoTask(row taskRow) *agentv1.Task {
	waiting, dialog := s.Tasks.WaitingForInput(row.ID)
	if !waiting && row.WaitingForInput != 0 {
		waiting = true
		dialog = row.PendingDialog.String
	}
	return &agentv1.Task{
		Id: row.ID,

		Status: agentv1.TaskStatus(
			session.ClientStatus(session.Status(row.Status), row.PendingPrompt.Valid, waiting),
		),
		Prompt:            row.Prompt,
		Name:              row.Name,
		Workspace:         row.Workspace,
		SessionId:         row.SessionID,
		Error:             row.Error.String,
		CreatedAt:         timestamp(row.CreatedAt),
		UpdatedAt:         timestamp(row.UpdatedAt),
		EndedAt:           timestamp(row.EndedAt.Int64),
		Interactive:       row.Interactive != 0,
		WaitingForInput:   waiting,
		PendingDialogJson: dialog,
		ProcessRunning:    s.Tasks.ProcessRunning(row.ID),

		RunInProgress: row.PendingPrompt.Valid,
	}
}

func timestamp(ms int64) *timestamppb.Timestamp {
	if ms <= 0 {
		return nil
	}
	return timestamppb.New(time.UnixMilli(ms))
}

func toProtoConfig(cfg session.SessionConfig) *agentv1.TaskConfig {
	models := make([]*agentv1.TaskModel, len(cfg.Models))
	for i, m := range cfg.Models {
		models[i] = &agentv1.TaskModel{
			Provider:       m.Provider,
			Id:             m.ID,
			Name:           m.Name,
			Reasoning:      m.Reasoning,
			ThinkingLevels: m.ThinkingLevels,
		}
	}
	return &agentv1.TaskConfig{
		ModelProvider:  cfg.ModelProvider,
		ModelId:        cfg.ModelID,
		ModelName:      cfg.ModelName,
		Models:         models,
		ThinkingLevel:  cfg.ThinkingLevel,
		ThinkingLevels: cfg.ThinkingLevels,
	}
}
