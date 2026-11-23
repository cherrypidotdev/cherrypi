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

package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"

	"cherrypi.vladfaust.com/agent/internal/agency"
)

const defaultListLimit = 50

const defaultHistoryLimit = 20

type toolOutput struct {
	message proto.Message
	raw     []byte
}

func protoOutput(message proto.Message) toolOutput { return toolOutput{message: message} }

func jsonOutput(value any) (toolOutput, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return toolOutput{}, err
	}
	return toolOutput{raw: raw}, nil
}

func (s *Session) agencyTool(
	name, label, description string,
	parameters *ai.Schema,
	run func(ctx context.Context, a args) (toolOutput, error),
) agent.AgentTool {
	return agent.AgentTool{
		Name:        name,
		Label:       label,
		Description: description,
		Parameters:  parameters,
		Execute: func(ctx context.Context, _ string, params map[string]any, _ agent.ToolUpdateFunc) (agent.AgentToolResult, error) {
			out, err := run(ctx, args(params))
			if err != nil {
				return errorText(err.Error()), nil
			}
			if out.raw != nil {
				return structuredJSON(out.raw), nil
			}
			return structuredMessage(out.message), nil
		},
	}
}

func (s *Session) agencyClient() (*agency.Client, error) {
	if s.cfg.Agency == nil {
		return nil, fmt.Errorf("the agency client is not configured")
	}
	return s.cfg.Agency, nil
}

func structuredMessage(message proto.Message) agent.AgentToolResult {
	if message == nil {
		return text("OK")
	}
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(message)
	if err != nil {
		return text("OK")
	}
	return structuredJSON(raw)
}

func structuredJSON(raw []byte) agent.AgentToolResult {
	result := text(string(raw))
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err == nil {
		result.Details = parsed
		result.StructuredContent = parsed
		return result
	}
	var list []any
	if err := json.Unmarshal(raw, &list); err == nil {
		result.Details = map[string]any{"items": list}
		result.StructuredContent = result.Details
	}
	return result
}

type args map[string]any

func (a args) str(name string) string {
	value, _ := a[name].(string)
	return strings.TrimSpace(value)
}

func (a args) present(name string) bool {
	value, ok := a[name]
	if !ok || value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
}

func (a args) number(name string) (float64, bool) {
	switch value := a[name].(type) {
	case float64:
		return value, true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func (a args) boolean(name string) (bool, bool) {
	switch value := a[name].(type) {
	case bool:
		return value, true
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

func (a args) list(name string) []any {
	value, _ := a[name].([]any)
	return value
}

func (a args) strings(name string) []string {
	switch value := a[name].(type) {
	case string:
		return stringList([]any{value})
	case []any:
		return stringList(value)
	case []string:
		return stringList(stringsToAny(value))
	}
	return nil
}

func stringsToAny(values []string) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

func (a args) object(name string) map[string]any {
	return objectArg(a, name)
}

func (a args) raw(name string) string {
	value, _ := a[name].(string)
	return value
}

func stringList(input []any) []string {
	out := make([]string, 0, len(input))
	for _, raw := range input {
		if value, ok := raw.(string); ok && strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func references(input []any) []*agencyv1.TaskReferenceInput {
	out := make([]*agencyv1.TaskReferenceInput, 0, len(input))
	for _, raw := range input {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		taskID, _ := entry["task_id"].(string)
		kind, _ := entry["kind"].(string)
		wake, _ := entry["wake_on_resolve_when_many"].(bool)
		out = append(out, &agencyv1.TaskReferenceInput{
			ReferencedTaskId:      strings.TrimSpace(taskID),
			Kind:                  referenceKind(kind),
			WakeOnResolveWhenMany: wake,
		})
	}
	return out
}

func referenceKind(value string) agencyv1.TaskReferenceKind {
	switch enumName(value) {
	case "blocked_by", "blocking":
		return agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_BLOCKED_BY
	case "blocks":
		return agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_BLOCKS
	case "supersedes":
		return agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_SUPERSEDES
	case "superseded_by":
		return agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_SUPERSEDED_BY
	default:
		return agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_INFORMATIONAL
	}
}

func enumName(value string) string {
	value = strings.Trim(enumSeparators.Replace(strings.ToLower(strings.TrimSpace(value))), "_")
	for _, prefix := range []string{
		"task_size_", "task_priority_", "task_state_", "prompt_scope_",
		"prompt_proposal_status_", "agent_log_level_", "task_reference_kind_",
		"size_", "priority_", "state_", "scope_", "status_", "level_", "kind_",
	} {
		if trimmed := strings.TrimPrefix(value, prefix); trimmed != value && trimmed != "" {
			return trimmed
		}
	}
	return value
}

var enumSeparators = strings.NewReplacer("-", "_", " ", "_", ".", "_", "/", "_")

func taskPriority(value string) (agencyv1.TaskPriority, bool) {
	switch enumName(value) {
	case "low":
		return agencyv1.TaskPriority_TASK_PRIORITY_LOW, true
	case "normal":
		return agencyv1.TaskPriority_TASK_PRIORITY_NORMAL, true
	case "high":
		return agencyv1.TaskPriority_TASK_PRIORITY_HIGH, true
	case "critical":
		return agencyv1.TaskPriority_TASK_PRIORITY_CRITICAL, true
	}
	return agencyv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED, false
}

func taskSize(value string) (agencyv1.TaskSize, bool) {
	switch enumName(value) {
	case "small":
		return agencyv1.TaskSize_TASK_SIZE_SMALL, true
	case "medium":
		return agencyv1.TaskSize_TASK_SIZE_MEDIUM, true
	case "big":
		return agencyv1.TaskSize_TASK_SIZE_BIG, true
	case "epic":
		return agencyv1.TaskSize_TASK_SIZE_EPIC, true
	}
	return agencyv1.TaskSize_TASK_SIZE_UNSPECIFIED, false
}

func taskState(value string) (agencyv1.TaskState, bool) {
	switch enumName(value) {
	case "backlog":
		return agencyv1.TaskState_TASK_STATE_BACKLOG, true
	case "waiting":
		return agencyv1.TaskState_TASK_STATE_WAITING, true
	case "queued":
		return agencyv1.TaskState_TASK_STATE_QUEUED, true
	case "running":
		return agencyv1.TaskState_TASK_STATE_RUNNING, true
	case "input":
		return agencyv1.TaskState_TASK_STATE_INPUT, true
	case "settled":
		return agencyv1.TaskState_TASK_STATE_SETTLED, true
	case "failed":
		return agencyv1.TaskState_TASK_STATE_FAILED, true
	case "completed":
		return agencyv1.TaskState_TASK_STATE_COMPLETED, true
	case "cancelled":
		return agencyv1.TaskState_TASK_STATE_CANCELLED, true
	}
	return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
}

func closeOutOutcome(value string) (agencyv1.TaskState, bool) {
	switch enumName(value) {
	case "completed":
		return agencyv1.TaskState_TASK_STATE_COMPLETED, true
	case "cancelled":
		return agencyv1.TaskState_TASK_STATE_CANCELLED, true
	default:
		return agencyv1.TaskState_TASK_STATE_UNSPECIFIED, false
	}
}

func promptScope(value string) agencyv1.PromptScope {
	switch enumName(value) {
	case "agent":
		return agencyv1.PromptScope_PROMPT_SCOPE_AGENT
	case "agent_project":
		return agencyv1.PromptScope_PROMPT_SCOPE_AGENT_PROJECT
	case "project":
		return agencyv1.PromptScope_PROMPT_SCOPE_PROJECT
	case "group":
		return agencyv1.PromptScope_PROMPT_SCOPE_GROUP
	case "group_project":
		return agencyv1.PromptScope_PROMPT_SCOPE_GROUP_PROJECT
	case "group_team":
		return agencyv1.PromptScope_PROMPT_SCOPE_GROUP_TEAM
	default:
		return agencyv1.PromptScope_PROMPT_SCOPE_UNSPECIFIED
	}
}

func proposalStatus(value string) agencyv1.PromptProposalStatus {
	switch enumName(value) {
	case "pending":
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_PENDING
	case "approved":
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_APPROVED
	case "rejected":
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_REJECTED
	case "withdrawn":
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_WITHDRAWN
	default:
		return agencyv1.PromptProposalStatus_PROMPT_PROPOSAL_STATUS_UNSPECIFIED
	}
}

func logLevel(value string) agencyv1.AgentLogLevel {
	switch enumName(value) {
	case "debug":
		return agencyv1.AgentLogLevel_AGENT_LOG_LEVEL_DEBUG
	case "info":
		return agencyv1.AgentLogLevel_AGENT_LOG_LEVEL_INFO
	case "warn":
		return agencyv1.AgentLogLevel_AGENT_LOG_LEVEL_WARN
	case "error":
		return agencyv1.AgentLogLevel_AGENT_LOG_LEVEL_ERROR
	default:
		return agencyv1.AgentLogLevel_AGENT_LOG_LEVEL_UNSPECIFIED
	}
}

const referenceKindDescription = "Edge kind from this task's perspective: " +
	"`blocked_by` gates this task's queuing, `blocks` makes the referenced task " +
	"wait for this one, `supersedes` cancels the referenced task, and " +
	"`superseded_by` cancels this one. Defaults to informational."

func referenceSchema(desc string) *ai.Schema {
	return ai.ArrayOf(ai.Object(
		ai.Prop("task_id", ai.String("Referenced task id.")),
		ai.Opt(
			"kind",
			ai.EnumOf(
				"informational",
				"blocked_by",
				"blocks",
				"supersedes",
				"superseded_by",
				"blocking",
			).Describe(referenceKindDescription),
		),
		ai.Opt(
			"wake_on_resolve_when_many",
			ai.Boolean(
				"Blocking edges only: start the task when this edge resolves "+
					"even though other blockers are still open. Default false, "+
					"which waits for every blocker.",
			),
		),
	), desc)
}

func linkSchema(desc string) *ai.Schema {
	return ai.ArrayOf(ai.Object(
		ai.Prop("url", ai.String("Absolute http(s) URL.")),
		ai.Prop("label", ai.String(`Short label, e.g. "PR".`)),
	), desc)
}

func scopeSchema(desc string) *ai.Schema {
	return ai.EnumOf(
		"agent",
		"agent_project",
		"project",
		"group",
		"group_project",
		"group_team",
	).Describe(desc)
}

func noArgumentsSchema() *ai.Schema {
	closed := false
	return &ai.Schema{
		Type:              "object",
		Properties:        map[string]*ai.Schema{},
		AdditionalAllowed: &closed,
	}
}

func (s *Session) agencyTools() []agent.AgentTool {
	return []agent.AgentTool{
		s.listTasksTool(),
		s.getTaskTool(),
		s.getTaskHistoryTool(),
		s.createBacklogTaskTool(),
		s.queueChildTaskTool(),
		s.updateTaskTool(),
		s.queueTaskTool(),
		s.closeOutTaskTool(),
		s.listAgentsTool(),
		s.readAgentLogsTool(),
		s.agencyMetadataTool(),
		s.readAgencyLogsTool(),
		s.getPromptTool(),
		s.listPromptVersionsTool(),
		s.proposePromptChangeTool(),
		s.listPromptProposalsTool(),
		s.listProjectsTool(),
		s.getTaskInputTool(),
	}
}

func (s *Session) inputTools() []agent.AgentTool {
	if !s.interactive {
		return []agent.AgentTool{
			s.answerOriginTaskQuestionsTool(),
			s.completeOriginTaskTool(),
			s.cancelOriginTaskTool(),
		}
	}
	return []agent.AgentTool{s.respondToTaskInputTool()}
}

func (s *Session) getTaskInputTool() agent.AgentTool {
	return s.agencyTool(
		"getTaskInput",
		"Get Task Input",
		"Read the input dialog a CherryPi task is blocked on: its method, message, options and, for a questionnaire, the questions. It has the same access gate as getTaskHistory: an agent may read its own tasks, and another task when it has the can-list permission in the task's project. Answering it stays lead-only (see respondToTaskInput). A task that is not waiting answers with `waiting: false` and a hint instead of failing, because the run may have resumed.",
		ai.Object(ai.Prop("taskId", ai.String("The task id to read."))),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Tasks.GetTaskInput(ctx, &agencyv1.GetTaskInputRequest{
				Id: a.str("taskId"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) respondToTaskInputTool() agent.AgentTool {
	return s.agencyTool(
		"respondToTaskInput",
		"Respond To Task Input",
		"Answer the input dialog a CherryPi task is blocked on, so the task continues. Only a lead of the task may answer. Send exactly one of `value` (select, input, editor), `confirmed` (confirm), `answers` (questionnaire: one entry per question, addressing it by header or 1-based index and naming an option label, or a `custom` answer) or `cancelled`. Pass the `requestId` from getTaskInput to guard against answering a dialog that was replaced.",
		ai.Object(
			ai.Prop("taskId", ai.String("The task id to answer.")),
			ai.Opt(
				"requestId",
				ai.String("The request id from getTaskInput; refuses a replaced dialog."),
			),
			ai.Opt("value", ai.String("The text, or one listed option, to send.")),
			ai.Opt("confirmed", ai.Boolean("The answer to a confirm dialog.")),
			ai.Opt(
				"answers",
				ai.ArrayOf(ai.Object(
					ai.Prop(
						"question",
						ai.String("The question's header, its 1-based index, or its text."),
					),
					ai.Opt(
						"options",
						ai.ArrayOf(ai.String(), "The chosen option labels."),
					),
					ai.Opt("custom", ai.String(`The "Type something." answer.`)),
					ai.Opt("notes", ai.String("A note attached to this answer.")),
				), "One entry per questionnaire question."),
			),
			ai.Opt("note", ai.String("A questionnaire-level final note.")),
			ai.Opt("cancelled", ai.Boolean("Decline the dialog instead of answering.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			answer := &agencyv1.TaskInputAnswer{
				RequestId: a.str("requestId"),
				Value:     a.raw("value"),
				Note:      a.raw("note"),
				Cancelled: false,
			}
			if confirmed, ok := a.boolean("confirmed"); ok {
				answer.Confirmed = &confirmed
			}
			if cancelled, ok := a.boolean("cancelled"); ok {
				answer.Cancelled = cancelled
			}
			if a.present("answers") {
				raw, err := json.Marshal(a.list("answers"))
				if err != nil {
					return toolOutput{}, fmt.Errorf("answers: %w", err)
				}
				answer.AnswersJson = string(raw)
			}
			res, err := client.Tasks.RespondToTaskInput(
				ctx,
				&agencyv1.RespondToTaskInputRequest{Id: a.str("taskId"), Answer: answer},
			)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) listTasksTool() agent.AgentTool {
	return s.agencyTool(
		"listTasks",
		"List Tasks",
		"List CherryPi tasks. With a projectId, lists that project's tasks when this agent has the can-list permission (otherwise only the agent's own tasks in it); without one, lists only this agent's own tasks. Optionally filter by assigned agent id, state, or a free-text search.",
		ai.Object(
			ai.Opt("projectId", ai.String("Project id, slug, or task prefix to list tasks in.")),
			ai.Opt("agentId", ai.String("Only tasks assigned to this agent id.")),
			ai.Opt(
				"state",
				ai.String(
					"Only tasks in this state (backlog, queued, running, settled, input, failed, completed, cancelled).",
				),
			),
			ai.Opt(
				"search",
				ai.String("Match task names, prompts, task ids, or numbers."),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			state, _ := taskState(a.str("state"))
			res, err := client.Tasks.ListTasks(ctx, &agencyv1.ListTasksRequest{
				ProjectId: a.str("projectId"),
				AgentId:   a.str("agentId"),
				State:     state,
				Search:    a.str("search"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) getTaskTool() agent.AgentTool {
	return s.agencyTool(
		"getTask",
		"Get Task",
		"Get one CherryPi task by id: its prompt, state, priority, size, tags, references, links, assignee, leader, origin chain, and automation kind. The conversation history is not included; read it with getTaskHistory. An agent may always read its own tasks, and another task when it has the can-list permission in the task's project.",
		ai.Object(ai.Prop("taskId", ai.String("The task id to read."))),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Tasks.GetTask(ctx, &agencyv1.GetTaskRequest{
				Id: a.str("taskId"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

type toolHistoryPage struct {
	TaskID     string             `json:"task_id"`
	Events     []toolHistoryEvent `json:"events"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type toolHistoryEvent struct {
	Type       string `json:"type"`
	At         string `json:"at,omitempty"`
	Text       string `json:"text,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	Phase      string `json:"phase,omitempty"`
	Args       string `json:"args,omitempty"`
	Result     string `json:"result,omitempty"`
	IsError    bool   `json:"is_error,omitempty"`
}

func (s *Session) getTaskHistoryTool() agent.AgentTool {
	tool := s.agencyTool(
		"getTaskHistory",
		"Get Task History",
		"Read one page of a CherryPi task's persisted conversation, newest page first: user prompts, assistant text and thinking, and tool calls. Events within a page stay chronological; pass the returned next_cursor as cursor to page back to older entries. The page defaults to the latest 20 entries and caps at 1000, but may hold fewer when large messages consume its byte budget. An oversized message keeps its start and end around a `start of message <truncated text> end of message` marker. An agent may always read its own tasks, and another task when it has the can-list permission in the task's project.",
		ai.Object(
			ai.Prop("taskId", ai.String("The task id to read.")),
			ai.Opt(
				"limit",
				ai.Number("Maximum entries per page; defaults to 20 and caps at 1000."),
			),
			ai.Opt(
				"cursor",
				ai.String(
					"Cursor from a previous page's next_cursor; omit for the newest page.",
				),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			limit := defaultHistoryLimit
			if value, ok := a.number("limit"); ok && value > 0 {
				limit = int(value)
			}
			taskID := a.str("taskId")
			res, err := client.Tasks.GetTaskHistory(
				ctx,
				&agencyv1.GetTaskHistoryRequest{
					Id:          taskID,
					Before:      a.str("cursor"),
					Limit:       uint32(limit),
					ModelFacing: true,
				},
			)
			if err != nil {
				return toolOutput{}, err
			}
			page := toolHistoryPage{TaskID: taskID, NextCursor: res.GetNextBefore()}
			for _, event := range res.GetEvents() {
				page.Events = append(page.Events, toolHistoryEventOf(event))
			}
			return jsonOutput(page)
		},
	)

	tool.OutputSchema = historyOutputSchema()
	return tool
}

func historyOutputSchema() *ai.Schema {
	return ai.Object(
		ai.Prop("task_id", ai.String()),
		ai.Prop("events", ai.ArrayOf(ai.Object(
			ai.Prop("type", ai.String()),
			ai.Opt("at", ai.String()),
			ai.Opt("text", ai.String()),
			ai.Opt("tool_call_id", ai.String()),
			ai.Opt("tool_name", ai.String()),
			ai.Opt("phase", ai.String()),
			ai.Opt("args", ai.String()),
			ai.Opt("result", ai.String()),
			ai.Opt("is_error", ai.Boolean()),
		))),
		ai.Prop("next_cursor", ai.String()),
	)
}

func toolHistoryEventOf(event *agentv1.TaskEvent) toolHistoryEvent {
	out := toolHistoryEvent{}
	if at := event.GetAt(); at != nil {
		out.At = at.AsTime().UTC().Format(time.RFC3339Nano)
	}
	switch payload := event.GetEvent().(type) {
	case *agentv1.TaskEvent_UserMessage:
		out.Type = "user_message"
		out.Text = payload.UserMessage.GetText()
	case *agentv1.TaskEvent_Text:
		out.Type = "text"
		out.Text = payload.Text.GetDelta()
	case *agentv1.TaskEvent_Thinking:
		out.Type = "thinking"
		out.Text = payload.Thinking.GetDelta()
	case *agentv1.TaskEvent_Tool:
		out.Type = "tool_call"
		out.ToolCallID = payload.Tool.GetToolCallId()
		out.ToolName = payload.Tool.GetToolName()
		out.Phase = toolPhaseName(payload.Tool.GetPhase())
		out.Args = payload.Tool.GetArgsJson()
		out.Result = payload.Tool.GetResultJson()
		out.IsError = payload.Tool.GetIsError()
	}
	return out
}

func toolPhaseName(phase agentv1.ToolPhase) string {
	switch phase {
	case agentv1.ToolPhase_TOOL_PHASE_STARTED:
		return "started"
	case agentv1.ToolPhase_TOOL_PHASE_UPDATED:
		return "updated"
	case agentv1.ToolPhase_TOOL_PHASE_FINISHED:
		return "finished"
	default:
		return "unspecified"
	}
}

func (s *Session) createBacklogTaskTool() agent.AgentTool {
	return s.agencyTool(
		"create_backlog_task",
		"Create Backlog Task",
		"Create a CherryPi task that does not block the task you are running. "+
			"By default it lands in backlog, the editable pre-queue state; pass "+
			"`queue: true` to start it as soon as an agent has a slot. It may "+
			"still carry its own blockers (including this task). To create work "+
			"this task must wait for, use queue_child_task instead.",
		taskCreateSchema(false),
		func(ctx context.Context, a args) (toolOutput, error) {
			return s.createTaskRequest(ctx, a, false)
		},
	)
}

func (s *Session) queueChildTaskTool() agent.AgentTool {
	return s.agencyTool(
		"queue_child_task",
		"Queue Child Task",
		"Create a task that blocks the task you are running, and queue it "+
			"immediately. This task parks as waiting until the new task is "+
			"completed or cancelled (call queueTask to park it right away "+
			"instead of waiting for your turn to end). Use it for the work this "+
			"task depends on.",
		taskCreateSchema(true),
		func(ctx context.Context, a args) (toolOutput, error) {
			return s.createTaskRequest(ctx, a, true)
		},
	)
}

func taskCreateSchema(child bool) *ai.Schema {
	props := []ai.Field{
		ai.Prop("projectId", ai.String("Project id or slug.")),
		ai.Opt(
			"prompt",
			ai.String(
				"The task prompt, or extra instructions appended to a template's rendered prompt.",
			),
		),
		ai.Opt(
			"templateId",
			ai.String(
				"Task template id from the project. The agency renders it with templateArgs.",
			),
		),
		ai.Opt(
			"templateVersion",
			ai.Number("Template version to render; omit or 0 for the latest."),
		),
		ai.Opt(
			"templateArgs",
			jsonValueSchema(
				"Template argument values as a JSON object string, keyed by argument key. "+
					"Strings, numbers and booleans are accepted.",
			),
		),
		ai.Opt(
			"assigneeId",
			ai.String(
				"Actor id, agent id, username or name of a project member to assign the task to. Use a member from listProjects; pass the username when a number names two members. Omit to let the scheduler pick a member agent.",
			),
		),
		ai.Opt("name", ai.String("Optional display name.")),
		ai.Opt("priority", ai.EnumOf("low", "normal", "high", "critical")),
		ai.Opt("size", ai.EnumOf("small", "medium", "big", "epic")),
		ai.Opt(
			"tagIds",
			ai.ArrayOf(
				ai.String(),
				"Tags to attach, by id or name (`unity` or `#unity`).",
			),
		),
		ai.Opt(
			"references",
			referenceSchema(
				"Tasks this task references. `blocked_by` and `blocks` gate queuing; the inverse kinds store the edge on the referenced task.",
			),
		),
	}

	if !child {
		props = append(props, ai.Opt(
			"queue",
			ai.Boolean(
				"Queue the task immediately instead of leaving it in backlog.",
			),
		))
	}
	return ai.Object(props...)
}

func (s *Session) createTaskRequest(
	ctx context.Context,
	a args,
	child bool,
) (toolOutput, error) {
	client, err := s.agencyClient()
	if err != nil {
		return toolOutput{}, err
	}
	refs := references(a.list("references"))

	projectID := a.str("projectId")
	tags := a.strings("tagIds")
	assignee := a.str("assigneeId")
	if len(tags) > 0 || assignee != "" {
		project, err := s.resolveProject(ctx, client, projectID)
		if err != nil {
			return toolOutput{}, err
		}
		projectID = project.GetId()
	}
	tagIDs, err := s.resolveTagIDs(ctx, client, projectID, tags)
	if err != nil {
		return toolOutput{}, err
	}
	assigneeID, err := s.resolveAssigneeID(ctx, client, projectID, assignee)
	if err != nil {
		return toolOutput{}, err
	}
	state := agencyv1.TaskState_TASK_STATE_BACKLOG
	if child {
		if own := s.agencyTaskID; own != "" {

			refs = append(refs, &agencyv1.TaskReferenceInput{
				ReferencedTaskId: own,
				Kind:             agencyv1.TaskReferenceKind_TASK_REFERENCE_KIND_BLOCKS,
			})
		}
		state = agencyv1.TaskState_TASK_STATE_QUEUED
	} else if queued, _ := a.boolean("queue"); queued {
		state = agencyv1.TaskState_TASK_STATE_QUEUED
	}
	request := &agencyv1.CreateTaskRequest{
		ProjectId:       projectID,
		Prompt:          a.str("prompt"),
		Name:            a.str("name"),
		AssigneeActorId: assigneeID,
		TagIds:          tagIDs,
		References:      refs,
		TemplateId:      a.str("templateId"),
		State:           state,

		OriginTaskId: s.agencyTaskID,
	}
	if priority, ok := taskPriority(a.str("priority")); ok {
		request.Priority = priority
	}
	if size, ok := taskSize(a.str("size")); ok {
		request.Size = size
	}
	if version, ok := a.number("templateVersion"); ok {
		request.TemplateVersion = int64(version)
	}
	if templateArgs := a.object("templateArgs"); len(templateArgs) > 0 {
		raw, err := json.Marshal(templateArgs)
		if err != nil {
			return toolOutput{}, fmt.Errorf("templateArgs: %w", err)
		}
		request.TemplateArgsJson = string(raw)
	}
	res, err := client.Tasks.CreateTask(ctx, request)
	if err != nil {
		return toolOutput{}, err
	}
	return protoOutput(res), nil
}

func (s *Session) originTaskID(
	ctx context.Context,
	explicit string,
) (string, error) {
	if id := strings.TrimSpace(explicit); id != "" {
		return id, nil
	}
	client, err := s.agencyClient()
	if err != nil {
		return "", err
	}
	res, err := client.Tasks.GetTask(ctx, &agencyv1.GetTaskRequest{Id: s.agencyTaskID})
	if err != nil {
		return "", err
	}
	origin := res.GetTask().GetOriginTaskId()
	if origin == "" {
		return "", fmt.Errorf(
			"this task has no origin; pass taskId explicitly",
		)
	}
	return origin, nil
}

func (s *Session) answerOriginTaskQuestionsTool() agent.AgentTool {
	return s.agencyTool(
		"answer_origin_task_questions",
		"Answer Origin Task Questions",
		"Answer the pending dialog of the task this reaction was created for. "+
			"Read it first with getTaskInput. Send exactly one of `value` "+
			"(select, input, editor), `confirmed` (confirm), `answers` "+
			"(questionnaire: one entry per question, addressing it by header or "+
			"1-based index and naming an option label, or a `custom` answer) or "+
			"`cancelled`. The origin task id defaults to this task's origin.",
		ai.Object(
			ai.Opt("taskId", ai.String("The origin task id; defaults to this task's origin.")),
			ai.Opt(
				"requestId",
				ai.String("The request id from getTaskInput; refuses a replaced dialog."),
			),
			ai.Opt("value", ai.String("The text, or one listed option, to send.")),
			ai.Opt("confirmed", ai.Boolean("The answer to a confirm dialog.")),
			ai.Opt(
				"answers",
				ai.ArrayOf(ai.Object(
					ai.Prop(
						"question",
						ai.String("The question's header, its 1-based index, or its text."),
					),
					ai.Opt(
						"options",
						ai.ArrayOf(ai.String(), "The chosen option labels."),
					),
					ai.Opt("custom", ai.String(`The "Type something." answer.`)),
					ai.Opt("notes", ai.String("A note attached to this answer.")),
				), "One entry per questionnaire question."),
			),
			ai.Opt("note", ai.String("A questionnaire-level final note.")),
			ai.Opt("cancelled", ai.Boolean("Decline the dialog instead of answering.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			taskID, err := s.originTaskID(ctx, a.str("taskId"))
			if err != nil {
				return toolOutput{}, err
			}
			answer := &agencyv1.TaskInputAnswer{
				RequestId: a.str("requestId"),
				Value:     a.raw("value"),
				Note:      a.raw("note"),
			}
			if confirmed, ok := a.boolean("confirmed"); ok {
				answer.Confirmed = &confirmed
			}
			if cancelled, ok := a.boolean("cancelled"); ok {
				answer.Cancelled = cancelled
			}
			if a.present("answers") {
				raw, err := json.Marshal(a.list("answers"))
				if err != nil {
					return toolOutput{}, fmt.Errorf("answers: %w", err)
				}
				answer.AnswersJson = string(raw)
			}
			res, err := client.Tasks.RespondToTaskInput(
				ctx,
				&agencyv1.RespondToTaskInputRequest{Id: taskID, Answer: answer},
			)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) completeOriginTaskTool() agent.AgentTool {
	return s.originCloseOutTool(
		"complete_origin_task",
		"Complete Origin Task",
		"Accept the origin task's outcome and close it out as completed. "+
			"Authority comes from the reaction link, so this works even when the "+
			"origin's leader changed. A running origin must settle or be stopped "+
			"first.",
		agencyv1.TaskState_TASK_STATE_COMPLETED,
	)
}

func (s *Session) cancelOriginTaskTool() agent.AgentTool {
	return s.originCloseOutTool(
		"cancel_origin_task",
		"Cancel Origin Task",
		"Abandon the origin task and close it out as cancelled, for example when "+
			"its work is obsolete or was replaced. Authority comes from the "+
			"reaction link.",
		agencyv1.TaskState_TASK_STATE_CANCELLED,
	)
}

func (s *Session) originCloseOutTool(
	name, label, description string,
	outcome agencyv1.TaskState,
) agent.AgentTool {
	return s.agencyTool(
		name,
		label,
		description,
		ai.Object(
			ai.Opt("taskId", ai.String("The origin task id; defaults to this task's origin.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			taskID, err := s.originTaskID(ctx, a.str("taskId"))
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Tasks.CloseOutTask(
				ctx,
				&agencyv1.CloseOutTaskRequest{
					Id:      taskID,
					Outcome: outcome,
				},
			)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) updateTaskTool() agent.AgentTool {
	return s.agencyTool(
		"updateTask",
		"Update Task",
		"Update a CherryPi task's name, links, tags, references, priority, size, or assignee. The agent may always update the task it is running; updating any other task needs the can-update permission in its project. Links, tag ids, and references replace the existing list. The result omits the task's original prompt: it is already the first message in the task's history.",
		ai.Object(
			ai.Prop("taskId", ai.String("The task id to update.")),
			ai.Opt("name", ai.String("New display name.")),
			ai.Opt("links", linkSchema("The complete link list; empty clears it.")),
			ai.Opt(
				"tagIds",
				ai.ArrayOf(
					ai.String(),
					"The complete tag list, by id or name; empty clears it.",
				),
			),
			ai.Opt(
				"references",
				referenceSchema(
					"The complete outgoing reference list; an empty list clears it. Incoming edges belong to the task that created them, so an inverse kind here adds an edge this list does not replace.",
				),
			),
			ai.Opt(
				"priority",
				ai.EnumOf("low", "normal", "high", "critical"),
			),
			ai.Opt("size", ai.EnumOf("small", "medium", "big", "epic")),
			ai.Opt(
				"assigneeId",
				ai.String(
					"Pin the task to this project member, by actor id, agent id, username or name; empty clears the pin.",
				),
			),
			ai.Opt(
				"state",
				ai.EnumOf("completed", "cancelled").Describe(
					"Close the task out: `completed` or `cancelled`. Only a lead or an administrator may close out. Queueing is a separate `queueTask` call.",
				),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			taskID := a.str("taskId")
			if taskID == "" {
				return toolOutput{}, fmt.Errorf("taskId is required")
			}
			if a.present("name") {
				if _, err := client.Tasks.RenameTask(
					ctx,
					&agencyv1.RenameTaskRequest{
						Id:   taskID,
						Name: a.str("name"),
					},
				); err != nil {
					return toolOutput{}, err
				}
			}
			if priority, ok := taskPriority(a.str("priority")); ok {
				if _, err := client.Tasks.SetTaskPriority(
					ctx,
					&agencyv1.SetTaskPriorityRequest{
						Id:       taskID,
						Priority: priority,
					},
				); err != nil {
					return toolOutput{}, err
				}
			}
			if size, ok := taskSize(a.str("size")); ok {
				if _, err := client.Tasks.SetTaskSize(
					ctx,
					&agencyv1.SetTaskSizeRequest{
						Id:   taskID,
						Size: size,
					},
				); err != nil {
					return toolOutput{}, err
				}
			}

			if value, ok := a["assigneeId"]; ok && value != nil {
				assignee, isString := value.(string)
				if !isString {
					return toolOutput{}, fmt.Errorf("assigneeId must be a string")
				}
				assignee = strings.TrimSpace(assignee)
				if assignee != "" {
					current, err := client.Tasks.GetTask(
						ctx,
						&agencyv1.GetTaskRequest{Id: taskID},
					)
					if err != nil {
						return toolOutput{}, err
					}
					assignee, err = s.resolveAssigneeID(
						ctx,
						client,
						current.GetTask().GetProjectId(),
						assignee,
					)
					if err != nil {
						return toolOutput{}, err
					}
				}
				if _, err := client.Tasks.UpdateTask(
					ctx,
					&agencyv1.UpdateTaskRequest{
						Id:                 taskID,
						SetAssigneeActorId: &assignee,
					},
				); err != nil {
					return toolOutput{}, err
				}
			}
			if a.present("links") {
				links := make([]*agencyv1.TaskLink, 0, len(a.list("links")))
				for _, raw := range a.list("links") {
					entry, ok := raw.(map[string]any)
					if !ok {
						continue
					}
					linkURL, _ := entry["url"].(string)
					label, _ := entry["label"].(string)
					links = append(links, &agencyv1.TaskLink{
						Url:   strings.TrimSpace(linkURL),
						Label: strings.TrimSpace(label),
					})
				}
				if _, err := client.Tasks.SetTaskLinks(
					ctx,
					&agencyv1.SetTaskLinksRequest{
						Id:    taskID,
						Links: links,
					},
				); err != nil {
					return toolOutput{}, err
				}
			}
			if a.present("tagIds") {
				if err := s.replaceTaskTags(
					ctx,
					client,
					taskID,
					a.strings("tagIds"),
				); err != nil {
					return toolOutput{}, err
				}
			}
			if a.present("references") {
				if err := s.replaceTaskReferences(
					ctx,
					client,
					taskID,
					references(a.list("references")),
				); err != nil {
					return toolOutput{}, err
				}
			}
			if a.present("state") {
				outcome, ok := closeOutOutcome(a.str("state"))
				if !ok {
					return toolOutput{}, fmt.Errorf(
						"state %q is not a close-out outcome; use completed or cancelled",
						a.str("state"),
					)
				}
				if _, err := client.Tasks.CloseOutTask(
					ctx,
					&agencyv1.CloseOutTaskRequest{
						Id:      taskID,
						Outcome: outcome,
					},
				); err != nil {
					return toolOutput{}, err
				}
			}

			res, err := client.Tasks.GetTask(ctx, &agencyv1.GetTaskRequest{
				Id: taskID,
			})
			if err != nil {
				return toolOutput{}, err
			}
			if res.GetTask() != nil {
				res.Task.Prompt = ""
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) resolveProject(
	ctx context.Context,
	client *agency.Client,
	value string,
) (*agencyv1.Project, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("projectId is required")
	}
	res, err := client.Projects.GetProject(
		ctx,
		&agencyv1.GetProjectRequest{Id: value},
	)
	if err != nil {
		return nil, err
	}
	return res.GetProject(), nil
}

func (s *Session) resolveTagIDs(
	ctx context.Context,
	client *agency.Client,
	projectID string,
	values []string,
) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if projectID == "" {
		return nil, fmt.Errorf("tags need a project to resolve against")
	}
	res, err := client.Tags.ListTags(
		ctx,
		&agencyv1.ListTagsRequest{ProjectId: projectID, IncludeArchived: true},
	)
	if err != nil {
		return nil, err
	}
	tags := res.GetTags()
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		if !tag.GetArchived() {
			names = append(names, "#"+tag.GetName())
		}
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "#"))
		var match *agencyv1.Tag
		for _, tag := range tags {
			if strings.EqualFold(tag.GetId(), strings.TrimSpace(value)) ||
				strings.EqualFold(tag.GetName(), name) {
				match = tag
				break
			}
		}
		if match == nil {
			if len(names) == 0 {
				return nil, fmt.Errorf(
					"tag %q is not in the project, which has no tags yet",
					value,
				)
			}
			return nil, fmt.Errorf(
				"tag %q is not in the project; use one of: %s",
				value,
				strings.Join(names, ", "),
			)
		}
		if seen[match.GetId()] {
			continue
		}
		seen[match.GetId()] = true
		out = append(out, match.GetId())
	}
	return out, nil
}

type projectMemberRecord struct {
	agentID string
	agent   *agencyv1.Agent
}

func (s *Session) resolveAssigneeID(
	ctx context.Context,
	client *agency.Client,
	projectID string,
	value string,
) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if projectID == "" {
		return "", fmt.Errorf("an assignee needs a project to resolve against")
	}
	res, err := client.Projects.ListProjectAgents(
		ctx,
		&agencyv1.ListProjectAgentsRequest{ProjectId: projectID},
	)
	if err != nil {
		return "", err
	}

	members := make([]projectMemberRecord, 0, len(res.GetAgents()))
	for _, member := range res.GetAgents() {
		agent, err := s.projectMember(ctx, client, member.GetAgentId())
		if err != nil {
			continue
		}
		members = append(members, projectMemberRecord{
			agentID: member.GetAgentId(),
			agent:   agent,
		})
	}
	matches := assigneeMatches(members, value)
	switch len(matches) {
	case 0:
		return "", fmt.Errorf(
			"assignee %q is not a member of the project; use one of: %s",
			value,
			assigneeChoices(members),
		)
	case 1:
		return matches[0].agent.GetActorId(), nil
	default:
		return "", fmt.Errorf(
			"assignee %q matches several members; pass the member's username: %s",
			value,
			assigneeChoices(matches),
		)
	}
}

func (s *Session) projectMember(
	ctx context.Context,
	client *agency.Client,
	agentID string,
) (*agencyv1.Agent, error) {
	res, err := client.Agents.GetAgent(
		ctx,
		&agencyv1.GetAgentRequest{Id: agentID},
	)
	if err != nil {
		return nil, err
	}
	return res.GetAgent(), nil
}

func assigneeMatches(members []projectMemberRecord, value string) []projectMemberRecord {
	var out []projectMemberRecord
	for _, member := range members {
		if memberNames(member, value) {
			out = append(out, member)
		}
	}
	return out
}

func memberNames(member projectMemberRecord, value string) bool {
	agent := member.agent
	for _, candidate := range []string{
		member.agentID,
		agent.GetActorId(),
		agent.GetUsername(),
		agent.GetDisplayName(),
	} {
		if candidate != "" && strings.EqualFold(candidate, value) {
			return true
		}
	}
	return false
}

func assigneeChoices(members []projectMemberRecord) string {
	if len(members) == 0 {
		return "the project has no agent members"
	}
	out := make([]string, 0, len(members))
	for _, member := range members {
		agent := member.agent
		label := agent.GetUsername()
		if label == "" {
			label = agent.GetDisplayName()
		}
		if label == "" {
			label = member.agentID
		}
		out = append(
			out,
			fmt.Sprintf("%s (actor %s, agent %s)", label, agent.GetActorId(), member.agentID),
		)
	}
	return strings.Join(out, ", ")
}

func (s *Session) replaceTaskTags(
	ctx context.Context,
	client *agency.Client,
	taskID string,
	want []string,
) error {
	current, err := client.Tasks.GetTask(ctx, &agencyv1.GetTaskRequest{
		Id: taskID,
	})
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, tag := range current.GetTask().GetTags() {
		have[tag.GetId()] = true
	}
	want, err = s.resolveTagIDs(
		ctx,
		client,
		current.GetTask().GetProjectId(),
		want,
	)
	if err != nil {
		return err
	}
	keep := map[string]bool{}
	var add, remove []string
	for _, id := range want {
		keep[id] = true
		if !have[id] {
			add = append(add, id)
		}
	}
	for id := range have {
		if !keep[id] {
			remove = append(remove, id)
		}
	}
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}
	_, err = client.Tasks.BulkUpdateTask(ctx, &agencyv1.BulkUpdateTaskRequest{
		Ids:          []string{taskID},
		AddTagIds:    add,
		RemoveTagIds: remove,
	})
	return err
}

func (s *Session) replaceTaskReferences(
	ctx context.Context,
	client *agency.Client,
	taskID string,
	want []*agencyv1.TaskReferenceInput,
) error {
	current, err := client.Tasks.GetTask(ctx, &agencyv1.GetTaskRequest{
		Id: taskID,
	})
	if err != nil {
		return err
	}
	type edge struct {
		id   string
		kind agencyv1.TaskReferenceKind
	}
	wanted := map[edge]bool{}
	byID := map[string]bool{}
	for _, ref := range want {
		if ref.GetReferencedTaskId() == "" {
			continue
		}
		wanted[edge{id: ref.GetReferencedTaskId(), kind: ref.GetKind()}] = true
		byID[ref.GetReferencedTaskId()] = true
	}

	for _, ref := range current.GetTask().GetReferences() {
		sameKind := wanted[edge{id: ref.GetId(), kind: ref.GetKind()}]
		if sameKind && byID[ref.GetId()] {
			continue
		}
		if _, err := client.Tasks.RemoveTaskReference(
			ctx,
			&agencyv1.RemoveTaskReferenceRequest{
				Id:               taskID,
				ReferencedTaskId: ref.GetId(),
			},
		); err != nil {
			return err
		}
	}
	for _, ref := range want {
		if ref.GetReferencedTaskId() == "" {
			continue
		}
		if _, err := client.Tasks.AddTaskReference(
			ctx,
			&agencyv1.AddTaskReferenceRequest{
				Id:               taskID,
				ReferencedTaskId: ref.GetReferencedTaskId(),
				Kind:             ref.GetKind(),
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) queueTaskTool() agent.AgentTool {
	return s.agencyTool(
		"queueTask",
		"Queue Task",
		"Move a CherryPi task to queued so the scheduler starts it as soon as an agent has a free slot. The agent may queue the task it is running; queueing any other task needs the can-update permission in its project. Queuing a task that holds an unresolved blocking reference parks it in `waiting` instead (slot-free), and it is queued again automatically when the last blocker is completed or cancelled — this is how a task waits for the blocking tasks it just created with queue_child_task. Without a blocker, a task that already started is queued by sending it a follow-up instead.",
		ai.Object(ai.Prop("taskId", ai.String("The task id to queue."))),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Tasks.QueueTask(ctx, &agencyv1.QueueTaskRequest{
				Id: a.str("taskId"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) closeOutTaskTool() agent.AgentTool {
	return s.agencyTool(
		"closeOutTask",
		"Close Out Task",
		"Complete or cancel a CherryPi task. Only a lead or an administrator may close out; an agent-created task adds its creator agent as a lead, so an agent can close out the work it spawned. A running task must settle or be stopped first.",
		ai.Object(
			ai.Prop("taskId", ai.String("The task id to close.")),
			ai.Prop(
				"outcome",
				ai.EnumOf("completed", "cancelled").Describe(
					"`completed` when the work is done, `cancelled` when abandoned.",
				),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			outcome, ok := closeOutOutcome(a.str("outcome"))
			if !ok {
				return toolOutput{}, fmt.Errorf(
					"outcome %q is not a close-out outcome; use completed or cancelled",
					a.str("outcome"),
				)
			}
			res, err := client.Tasks.CloseOutTask(
				ctx,
				&agencyv1.CloseOutTaskRequest{
					Id:      a.str("taskId"),
					Outcome: outcome,
				},
			)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) listAgentsTool() agent.AgentTool {
	return s.agencyTool(
		"listAgents",
		"List Agents",
		"Admin agents only. List every agent with its metadata: role, presence, build version, protocol version, the address its tunnel connected from, and last seen time. Read-only.",
		noArgumentsSchema(),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Agency.ListAgents(
				ctx,
				&agencyv1.ListAgencyAgentsRequest{},
			)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) readAgentLogsTool() agent.AgentTool {
	return s.agencyTool(
		"readAgentLogs",
		"Read Agent Logs",
		"Admin agents only. Read one agent's persisted operational logs (process lines, task/session lifecycle, connectivity), newest first. Unfiltered by default; pass excludeMessages to omit routine lines. Get ids from listAgents. Read-only.",
		ai.Object(
			ai.Prop(
				"agentId",
				ai.String("The numeric agent id from listAgents."),
			),
			ai.Opt(
				"limit",
				ai.Number("Maximum lines; defaults to 200 and caps at 1000."),
			),
			ai.Opt(
				"minLevel",
				ai.EnumOf("debug", "info", "warn", "error").
					Describe("Only lines at or above this severity."),
			),
			ai.Opt(
				"beforeId",
				ai.Number(
					"Page older than this log id (the last row's id from a previous call).",
				),
			),
			ai.Opt(
				"excludeMessages",
				ai.ArrayOf(
					ai.String(`A message substring to omit, e.g. "pi started". Case-insensitive.`),
					`Message substrings to omit. Unfiltered by default; the dashboard uses ["pi started", "pi exited", "pi stopped"].`,
				),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			request := &agencyv1.ListAgentLogsRequest{
				AgentId:  a.str("agentId"),
				MinLevel: logLevel(a.str("minLevel")),
			}
			if limit, ok := a.number("limit"); ok && limit > 0 {
				request.Limit = int32(limit)
			}
			if before, ok := a.number("beforeId"); ok {
				request.BeforeId = int64(before)
			}
			res, err := client.Agents.ListAgentLogs(ctx, request)
			if err != nil {
				return toolOutput{}, err
			}

			exclude := stringList(a.list("excludeMessages"))
			if len(exclude) == 0 {
				return protoOutput(res), nil
			}
			filtered := &agencyv1.ListAgentLogsResponse{}
			for _, line := range res.GetLogs() {
				if excludedMessage(line.GetMessage(), exclude) {
					continue
				}
				filtered.Logs = append(filtered.Logs, line)
			}
			return protoOutput(filtered), nil
		},
	)
}

// excludedMessage reports whether a log line carries one of the patterns.
func excludedMessage(message string, patterns []string) bool {
	lowered := strings.ToLower(message)
	for _, pattern := range patterns {
		if pattern = strings.ToLower(strings.TrimSpace(pattern)); pattern == "" {
			continue
		}
		if strings.Contains(lowered, pattern) {
			return true
		}
	}
	return false
}

func (s *Session) agencyMetadataTool() agent.AgentTool {
	return s.agencyTool(
		"getAgencyMetadata",
		"Get Agency Metadata",
		"Admin agents only. Read the agency's own metadata: build version, process id, start time and uptime, plus this agent's role. Read-only.",
		noArgumentsSchema(),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Agency.GetMetadata(
				ctx,
				&agencyv1.GetAgencyMetadataRequest{},
			)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) readAgencyLogsTool() agent.AgentTool {
	return s.agencyTool(
		"readAgencyLogs",
		"Read Agency Logs",
		"Admin agents only. Read the tail of the agency's current process log (everything the process wrote to stdout/stderr, timestamp-prefixed). Read-only.",
		ai.Object(
			ai.Opt(
				"lines",
				ai.Number("How many newest lines; defaults to 200 and caps at 2000."),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			request := &agencyv1.ReadAgencyLogsRequest{}
			if lines, ok := a.number("lines"); ok && lines > 0 {
				request.Lines = uint32(lines)
			}
			res, err := client.Agency.ReadLogs(ctx, request)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) getPromptTool() agent.AgentTool {
	return s.agencyTool(
		"getPrompt",
		"Get Prompt",
		"Read a versioned CherryPi prompt and whether this agent may propose a change. `scope` is `agent` (your own global prompt), `agent_project` (your prompt for one project), `project` (a project's shared prompt), `group` (an agent group's prompt), `group_project` (a group × project prompt), or `group_team` (a group × team prompt). Pass `projectId`, `groupId` and/or `teamId` as the scope requires. Group prompts are readable by any agent; only a member may propose. The response includes the content, version, author, and the caller's can_edit/can_propose/can_approve flags.",
		ai.Object(
			ai.Prop("scope", scopeSchema("Which prompt to read.")),
			ai.Opt(
				"projectId",
				ai.String(
					"Project id or slug; required for agent_project, project and group_project.",
				),
			),
			ai.Opt(
				"groupId",
				ai.Number(
					"Agent group id; required for group, group_project and group_team.",
				),
			),
			ai.Opt("teamId", ai.Number("Team id; required for group_team.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			request := &agencyv1.GetPromptRequest{
				Scope:     promptScope(a.str("scope")),
				ProjectId: a.str("projectId"),
			}
			if groupID, ok := a.number("groupId"); ok {
				request.GroupId = int64(groupID)
			}
			if teamID, ok := a.number("teamId"); ok {
				request.TeamId = int64(teamID)
			}
			res, err := client.Prompts.GetPrompt(ctx, request)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) listPromptVersionsTool() agent.AgentTool {
	return s.agencyTool(
		"listPromptVersions",
		"List Prompt Versions",
		"Read a CherryPi prompt's version history, newest first, with each version's author, approver, note, and content. Use it to understand how the prompt evolved before proposing a change. Pass the same scope/projectId/groupId/teamId as getPrompt.",
		ai.Object(
			ai.Prop("scope", scopeSchema("Which prompt's history to read.")),
			ai.Opt(
				"projectId",
				ai.String(
					"Project id or slug; required for agent_project, project and group_project.",
				),
			),
			ai.Opt(
				"groupId",
				ai.Number(
					"Agent group id; required for group, group_project and group_team.",
				),
			),
			ai.Opt("teamId", ai.Number("Team id; required for group_team.")),
			ai.Opt(
				"limit",
				ai.Number("Maximum versions; defaults to 50 and caps at 200."),
			),
			ai.Opt(
				"beforeVersion",
				ai.Number("Page older than this version (0 = from latest)."),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			request := &agencyv1.ListPromptVersionsRequest{
				Scope:     promptScope(a.str("scope")),
				ProjectId: a.str("projectId"),
				Limit:     defaultListLimit,
			}
			if groupID, ok := a.number("groupId"); ok {
				request.GroupId = int64(groupID)
			}
			if teamID, ok := a.number("teamId"); ok {
				request.TeamId = int64(teamID)
			}
			if limit, ok := a.number("limit"); ok && limit > 0 {
				request.Limit = uint32(limit)
			}
			if before, ok := a.number("beforeVersion"); ok {
				request.BeforeVersion = int64(before)
			}
			res, err := client.Prompts.ListPromptVersions(ctx, request)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) proposePromptChangeTool() agent.AgentTool {
	return s.agencyTool(
		"proposePromptChange",
		"Propose Prompt Change",
		"Propose a complete replacement for a CherryPi prompt. Use `scope=agent` for your own global prompt, `agent_project` for your prompt in one project, `project` for a project's shared prompt (project members only), or `group`/`group_project`/`group_team` for a group you belong to (group owners approve; the agent auto-approve setting does not apply to groups). The change becomes a pending proposal that an owner approves. Include a short `note` explaining why.",
		ai.Object(
			ai.Prop("scope", scopeSchema("Which prompt to propose a change to.")),
			ai.Prop(
				"content",
				ai.String("The complete replacement prompt content."),
			),
			ai.Opt("note", ai.String("Why this change is useful.")),
			ai.Opt(
				"projectId",
				ai.String(
					"Project id or slug; required for agent_project, project and group_project.",
				),
			),
			ai.Opt(
				"groupId",
				ai.Number(
					"Agent group id; required for group, group_project and group_team.",
				),
			),
			ai.Opt("teamId", ai.Number("Team id; required for group_team.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			request := &agencyv1.CreatePromptProposalRequest{
				Scope:     promptScope(a.str("scope")),
				ProjectId: a.str("projectId"),

				Content: a.raw("content"),
				Note:    a.str("note"),
			}
			if groupID, ok := a.number("groupId"); ok {
				request.GroupId = int64(groupID)
			}
			if teamID, ok := a.number("teamId"); ok {
				request.TeamId = int64(teamID)
			}
			res, err := client.Prompts.CreatePromptProposal(ctx, request)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) listPromptProposalsTool() agent.AgentTool {
	return s.agencyTool(
		"listPromptProposals",
		"List Prompt Proposals",
		"List prompt change proposals. By default returns your own proposals across scopes, newest first. Pass `mine=false` with a scope and its ids to list a target's proposals when you are allowed to approve them.",
		ai.Object(
			ai.Opt(
				"scope",
				scopeSchema("Optional target scope filter."),
			),
			ai.Opt(
				"projectId",
				ai.String(
					"Project id or slug; used with agent_project, project and group_project scopes.",
				),
			),
			ai.Opt(
				"groupId",
				ai.Number(
					"Agent group id; used with group, group_project and group_team scopes.",
				),
			),
			ai.Opt("teamId", ai.Number("Team id; used with the group_team scope.")),
			ai.Opt(
				"status",
				ai.EnumOf("pending", "approved", "rejected", "withdrawn").
					Describe("Only proposals in this status."),
			),
			ai.Opt(
				"mine",
				ai.Boolean(
					"Only your own proposals. Defaults to true; set false with a scope to list a target's proposals for approval.",
				),
			),
			ai.Opt(
				"limit",
				ai.Number("Maximum proposals; defaults to 50 and caps at 200."),
			),
			ai.Opt("beforeId", ai.Number("Page older than this proposal id.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			mine, ok := a.boolean("mine")
			if !ok {
				mine = true
			}
			request := &agencyv1.ListPromptProposalsRequest{
				Scope:     promptScope(a.str("scope")),
				ProjectId: a.str("projectId"),
				Status:    proposalStatus(a.str("status")),
				Mine:      mine,
				Limit:     defaultListLimit,
			}
			if groupID, ok := a.number("groupId"); ok {
				request.GroupId = int64(groupID)
			}
			if teamID, ok := a.number("teamId"); ok {
				request.TeamId = int64(teamID)
			}
			if limit, ok := a.number("limit"); ok && limit > 0 {
				request.Limit = uint32(limit)
			}
			if before, ok := a.number("beforeId"); ok {
				request.BeforeId = int64(before)
			}
			res, err := client.Prompts.ListPromptProposals(ctx, request)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) listProjectsTool() agent.AgentTool {
	return s.agencyTool(
		"listProjects",
		"List Projects",
		"List the CherryPi projects this agent can work in, with the agent's own task permissions in each, the project's description, and every agent member's identity (`actor_id`, name, username) and permissions. Use a member's `actor_id` when assigning a task.",
		noArgumentsSchema(),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}

			res, err := client.Projects.ListAgentMemberships(
				ctx,
				&agencyv1.ListAgentMembershipsRequest{},
			)
			if err != nil {
				return toolOutput{}, err
			}

			people := map[string]*agencyv1.Agent{}
			view := projectsView{
				Projects: make([]projectView, 0, len(res.GetMemberships())),
			}
			for _, membership := range res.GetMemberships() {
				project := membership.GetProject()
				entry := projectView{
					ID:              project.GetId(),
					Name:            project.GetName(),
					Slug:            project.GetSlug(),
					Description:     project.GetDescription(),
					Prefix:          project.GetPrefix(),
					ParentProjectID: project.GetParentProjectId(),
					Archived:        project.GetArchivedAt() != nil,
					Permissions: permissionsView{
						CanListTasks:   membership.GetCanListTasks(),
						CanCreateTasks: membership.GetCanCreateTasks(),
						CanUpdateTasks: membership.GetCanUpdateTasks(),
					},
				}
				members, err := client.Projects.ListProjectAgents(
					ctx,
					&agencyv1.ListProjectAgentsRequest{
						ProjectId: project.GetId(),
					},
				)
				if err == nil {
					for _, member := range members.GetAgents() {
						person := people[member.GetAgentId()]
						if person == nil {
							got, getErr := client.Agents.GetAgent(
								ctx,
								&agencyv1.GetAgentRequest{Id: member.GetAgentId()},
							)
							if getErr == nil {
								person = got.GetAgent()
								people[member.GetAgentId()] = person
							}
						}
						entry.Members = append(entry.Members, projectMember{
							AgentID:     member.GetAgentId(),
							ActorID:     person.GetActorId(),
							Name:        person.GetDisplayName(),
							Username:    person.GetUsername(),
							Description: person.GetDescription(),
							Permissions: permissionsView{
								CanListTasks:   member.GetCanListTasks(),
								CanCreateTasks: member.GetCanCreateTasks(),
								CanUpdateTasks: member.GetCanUpdateTasks(),
							},
						})
					}
				}
				view.Projects = append(view.Projects, entry)
			}
			return jsonOutput(view)
		},
	)
}

type projectsView struct {
	Projects []projectView `json:"projects"`
}

type projectView struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	Description     string          `json:"description,omitempty"`
	Prefix          string          `json:"prefix,omitempty"`
	ParentProjectID string          `json:"parent_project_id,omitempty"`
	Archived        bool            `json:"archived,omitempty"`
	Permissions     permissionsView `json:"permissions"`
	Members         []projectMember `json:"members,omitempty"`
}

type projectMember struct {
	AgentID     string          `json:"agent_id"`
	ActorID     string          `json:"actor_id"`
	Name        string          `json:"name,omitempty"`
	Username    string          `json:"username,omitempty"`
	Description string          `json:"description,omitempty"`
	Permissions permissionsView `json:"permissions"`
}

type permissionsView struct {
	CanListTasks   bool `json:"can_list_tasks"`
	CanCreateTasks bool `json:"can_create_tasks"`
	CanUpdateTasks bool `json:"can_update_tasks"`
}
