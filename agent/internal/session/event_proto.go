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
	"encoding/json"
	"time"

	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TaskEventProto(sessionID string, ev Event) *agentv1.TaskEvent {
	out := &agentv1.TaskEvent{
		TaskId: sessionID,
		At:     timestamppb.New(ev.At),
	}

	switch ev.Kind {
	case EventStatus:
		out.Event = &agentv1.TaskEvent_StatusChanged{
			StatusChanged: &agentv1.TaskStatusChanged{
				Status: agentv1.TaskStatus(ev.Status),
				Error:  ev.Error,
			},
		}
	case EventRunStarted:
		out.Event = &agentv1.TaskEvent_RunStarted{RunStarted: &agentv1.RunStarted{}}
	case EventRunSettled:
		out.Event = &agentv1.TaskEvent_RunSettled{RunSettled: &agentv1.RunSettled{}}
	case EventText:
		out.Event = &agentv1.TaskEvent_Text{Text: &agentv1.AgentText{Delta: ev.Delta}}
	case EventThinking:
		thinking := &agentv1.AgentThinking{Delta: ev.Delta}

		if ev.ThinkingSpan != nil {
			thinking.DurationMs = ev.ThinkingSpan.DurationMs
			thinking.Chars = ev.ThinkingSpan.Chars
			thinking.Estimated = ev.ThinkingSpan.Estimated
		}
		out.Event = &agentv1.TaskEvent_Thinking{Thinking: thinking}
	case EventThinkingSpan:

		thinking := &agentv1.AgentThinking{}
		if ev.ThinkingSpan != nil {
			thinking.DurationMs = ev.ThinkingSpan.DurationMs
			thinking.Chars = ev.ThinkingSpan.Chars
			thinking.Estimated = ev.ThinkingSpan.Estimated
		}
		out.Event = &agentv1.TaskEvent_Thinking{Thinking: thinking}
	case EventLog:
		out.Event = &agentv1.TaskEvent_Log{Log: &agentv1.LogLine{Line: ev.Line}}
	case EventTool:
		out.Event = &agentv1.TaskEvent_Tool{
			Tool: &agentv1.ToolExecution{
				ToolCallId:  ev.Tool.CallID,
				ToolName:    ev.Tool.Name,
				Phase:       agentv1.ToolPhase(ev.Tool.Phase),
				ArgsJson:    ev.Tool.ArgsJSON,
				ResultJson:  ev.Tool.ResultJSON,
				DetailsJson: ev.Tool.DetailsJSON,
				IsError:     ev.Tool.IsError,
			},
		}
	case EventExtensionUI:
		out.Event = &agentv1.TaskEvent_ExtensionUi{
			ExtensionUi: &agentv1.ExtensionUiRequest{
				RequestId:   ev.Extension.RequestID,
				Method:      ev.Extension.Method,
				Title:       ev.Extension.Title,
				Message:     ev.Extension.Message,
				Placeholder: ev.Extension.Placeholder,
				Prefill:     ev.Extension.Prefill,
				Options:     ev.Extension.Options,
				Timeout:     ev.Extension.Timeout,
				RawJson:     ev.Extension.RawJSON,
			},
		}
	case EventUser:
		message := &agentv1.UserMessage{Text: ev.Text}

		if parsed, ok := ParsePromptEnvelope(ev.Text); ok {
			message.Body = parsed.Body
			if parsed.Sender != nil {
				message.Sender = &agentv1.PromptSender{
					ActorId:  parsed.Sender.ActorID,
					Username: parsed.Sender.Username,
					Name:     parsed.Sender.Name,
					Kind:     parsed.Sender.Kind,
				}
			}
			for _, attachment := range parsed.Attachments {
				message.Attachments = append(message.Attachments, &agentv1.MessageAttachment{
					Id:       attachment.ID,
					Filename: attachment.Filename,
				})
			}
		}
		out.Event = &agentv1.TaskEvent_UserMessage{UserMessage: message}
	case EventPending:
		pending := &agentv1.PendingActivity{}
		if ev.Pending != nil {
			pending.Kind = agentv1.PendingActivity_Kind(ev.Pending.Kind)
			pending.Model = ev.Pending.Model
			pending.Provider = ev.Pending.Provider
			pending.Error = ev.Pending.Error
			pending.Attempt = ev.Pending.Attempt
			pending.MaxAttempts = ev.Pending.MaxAttempts
			pending.Reason = ev.Pending.Reason
		}
		out.Event = &agentv1.TaskEvent_PendingActivity{PendingActivity: pending}
	case EventModelWait:
		wait := &agentv1.ModelWait{}
		if ev.ModelWait != nil {
			wait.Kind = agentv1.ModelWait_Kind(ev.ModelWait.Kind)
			wait.Model = ev.ModelWait.Model
			wait.Provider = ev.ModelWait.Provider
			wait.DurationMs = ev.ModelWait.DurationMs
			wait.Error = ev.ModelWait.Error
			wait.Attempt = ev.ModelWait.Attempt
			wait.Reason = ev.ModelWait.Reason
		}
		out.Event = &agentv1.TaskEvent_ModelWait{ModelWait: wait}
	case EventQueue:
		messages := &agentv1.PendingMessages{}
		for _, entry := range ev.Queue.Steering {
			messages.QueuedSteering = append(messages.QueuedSteering, &agentv1.QueuedMessage{
				Id:   entry.ID,
				Text: entry.Text,
			})
		}

		//nolint:staticcheck // the deprecated field is the whole point here
		for _, entry := range ev.Queue.Steering {
			messages.Steering = append(messages.Steering, entry.Text)
		}
		out.Event = &agentv1.TaskEvent_PendingMessages{PendingMessages: messages}
	}

	return out
}

func DomainEventProto(kind string, payload map[string]any) *agentv1.TaskEvent {
	raw, _ := json.Marshal(payload)
	return &agentv1.TaskEvent{
		At: timestamppb.Now(),
		Event: &agentv1.TaskEvent_Domain{Domain: &agentv1.TaskDomainEvent{
			Kind:        kind,
			PayloadJson: string(raw),
		}},
	}
}

func eventKind(ev Event) string {
	switch ev.Kind {
	case EventUser:
		return "user"
	case EventText:
		return "text"
	case EventThinking:
		return "thinking"
	case EventThinkingSpan:

		return "thinking"
	case EventTool:
		return "tool"
	case EventStatus, EventRunStarted, EventRunSettled:
		return "status"
	case EventModelWait:

		return "model_wait"
	}
	return ""
}

func messageEvents(message agent.AgentMessage) []Event {
	at := time.Now()
	timestamp := messageTimestamp(message)
	if timestamp > 0 {
		at = time.UnixMilli(timestamp)
	}

	switch msg := message.(type) {
	case ai.UserMessage:
		if text := userMessageText(msg); text != "" {
			return []Event{{Kind: EventUser, At: at, Text: text}}
		}
	case *ai.UserMessage:
		if text := userMessageText(msg); text != "" {
			return []Event{{Kind: EventUser, At: at, Text: text}}
		}
	case ai.AssistantMessage:
		return assistantMessageEvents(msg.Content, at)
	case *ai.AssistantMessage:
		if msg != nil {
			return assistantMessageEvents(msg.Content, at)
		}
	case ai.ToolResultMessage:
		return toolResultEvents(msg, at)
	case *ai.ToolResultMessage:
		if msg != nil {
			return toolResultEvents(*msg, at)
		}
	}
	return nil
}

func assistantMessageEvents(content ai.ContentList, at time.Time) []Event {
	var out []Event
	for _, block := range content {
		switch value := block.(type) {
		case ai.TextContent:
			if value.Text != "" {
				out = append(out, Event{Kind: EventText, At: at, Delta: value.Text})
			}
		case *ai.TextContent:
			if value != nil && value.Text != "" {
				out = append(out, Event{Kind: EventText, At: at, Delta: value.Text})
			}
		case ai.ThinkingContent:
			if value.Thinking != "" {
				out = append(out, Event{Kind: EventThinking, At: at, Delta: value.Thinking})
			}
		case *ai.ThinkingContent:
			if value != nil && value.Thinking != "" {
				out = append(out, Event{Kind: EventThinking, At: at, Delta: value.Thinking})
			}
		case ai.ToolCall:
			out = append(out, toolCallEvent(value, at))
		case *ai.ToolCall:
			if value != nil {
				out = append(out, toolCallEvent(*value, at))
			}
		}
	}
	return out
}

func toolCallEvent(call ai.ToolCall, at time.Time) Event {
	args, _ := json.Marshal(call.Arguments)
	return Event{Kind: EventTool, At: at, Tool: &ToolEvent{
		CallID:   call.ID,
		Name:     call.Name,
		Phase:    ToolStarted,
		ArgsJSON: string(args),
	}}
}

func toolResultEvents(message ai.ToolResultMessage, at time.Time) []Event {
	details := ""
	if message.Details != nil {
		if raw, err := json.Marshal(message.Details); err == nil {
			details = string(raw)
		}
	}
	return []Event{{Kind: EventTool, At: at, Tool: &ToolEvent{
		CallID:      message.ToolCallID,
		Name:        message.ToolName,
		Phase:       ToolFinished,
		ResultJSON:  contentListText(message.Content),
		DetailsJSON: details,
		IsError:     message.IsError,
	}}}
}

func contentListText(content ai.ContentList) string {
	var out string
	for _, block := range content {
		if text, ok := block.(ai.TextContent); ok {
			out += text.Text
		}
	}
	return out
}

func messageTimestamp(message agent.AgentMessage) int64 {
	switch msg := message.(type) {
	case ai.UserMessage:
		return msg.Timestamp
	case *ai.UserMessage:
		if msg != nil {
			return msg.Timestamp
		}
	case ai.AssistantMessage:
		return msg.Timestamp
	case *ai.AssistantMessage:
		if msg != nil {
			return msg.Timestamp
		}
	case ai.ToolResultMessage:
		return msg.Timestamp
	case *ai.ToolResultMessage:
		if msg != nil {
			return msg.Timestamp
		}
	}
	return 0
}
