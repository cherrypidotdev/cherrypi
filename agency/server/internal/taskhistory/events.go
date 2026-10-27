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

package taskhistory

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"cherrypi.vladfaust.com/agency/internal/tasks"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultPageSize = 200
	maxPageSize     = 1000
)

type historyEntry struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
}

type historyMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Details    json.RawMessage `json:"details"`
	IsError    bool            `json:"isError"`
}

type historyBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type promptEnvelope struct {
	SendingActor envelopeSender `json:"sendingActor"`
	MessageBody  envelopeBody   `json:"messageBody"`
}

type envelopeSender struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

type envelopeBody struct {
	Text        string               `json:"text"`
	Attachments []envelopeAttachment `json:"attachments"`
}

type envelopeAttachment struct {
	ID       int64  `json:"id"`
	Filename string `json:"filename"`
}

func userMessage(text string) *agentv1.UserMessage {
	message := &agentv1.UserMessage{Text: text}
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{") {
		return message
	}
	var env promptEnvelope
	if json.NewDecoder(strings.NewReader(trimmed)).Decode(&env) != nil {
		return message
	}
	if env.SendingActor.Username == "" && env.SendingActor.ID == 0 &&
		env.SendingActor.Name == "" {
		return message
	}
	message.Body = env.MessageBody.Text
	message.Sender = &agentv1.PromptSender{
		ActorId:  env.SendingActor.ID,
		Username: env.SendingActor.Username,
		Name:     env.SendingActor.Name,
		Kind:     env.SendingActor.Kind,
	}
	for _, attachment := range env.MessageBody.Attachments {
		message.Attachments = append(message.Attachments, &agentv1.MessageAttachment{
			Id:       attachment.ID,
			Filename: attachment.Filename,
		})
	}
	return message
}

func Page(
	taskID string,
	entries []tasks.SessionHistoryEntry,
	before int64,
	limit int,
) ([]*agentv1.TaskEvent, int64) {
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	older := entries
	if before > 0 {
		older = nil
		for _, e := range entries {
			if e.Seq < before {
				older = append(older, e)
			}
		}
	}

	var nextBefore int64
	if len(older) > limit {
		older = older[len(older)-limit:]
		nextBefore = older[0].Seq
	} else if len(older) > 0 {
		nextBefore = 0
	}

	var events []*agentv1.TaskEvent
	for _, e := range older {
		for _, event := range parseLine(e.Raw) {
			event.TaskId = taskID
			events = append(events, event)
		}
	}
	return events, nextBefore
}

const truncatedMessageMarker = "\n\nstart of message <truncated text> end of message\n\n"

func PageBounded(
	taskID string,
	entries []tasks.SessionHistoryEntry,
	before int64,
	limit int,
	maxMessageBytes int,
	maxPageBytes int,
) ([]*agentv1.TaskEvent, int64) {
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	older := entries
	if before > 0 {
		older = nil
		for _, e := range entries {
			if e.Seq < before {
				older = append(older, e)
			}
		}
	}

	window := older
	if len(window) > limit {
		window = window[len(window)-limit:]
	}

	type pageEntry struct {
		seq    int64
		events []*agentv1.TaskEvent
	}
	page := make([]pageEntry, 0, len(window))
	totalBytes := 0
	stopped := false
	for i := len(window) - 1; i >= 0; i-- {
		e := window[i]
		events := parseLine(e.Raw)
		truncateMessage(events, maxMessageBytes)
		size := messageBytes(events)
		if maxPageBytes > 0 && len(page) > 0 && totalBytes+size > maxPageBytes {
			stopped = true
			break
		}
		page = append(page, pageEntry{seq: e.Seq, events: events})
		totalBytes += size
	}

	var nextBefore int64
	switch {
	case stopped:

		nextBefore = page[len(page)-1].seq
	case len(older) > len(window):
		nextBefore = window[0].Seq
	}

	var out []*agentv1.TaskEvent
	for i := len(page) - 1; i >= 0; i-- {
		for _, event := range page[i].events {
			event.TaskId = taskID
			out = append(out, event)
		}
	}
	return out, nextBefore
}

func truncateMessage(events []*agentv1.TaskEvent, maxBytes int) {
	if maxBytes <= 0 {
		return
	}
	for messageBytes(events) > maxBytes {
		var largest *string
		for _, event := range events {
			for _, field := range messageFields(event) {
				if len(*field) > 0 && (largest == nil || len(*field) > len(*largest)) {
					largest = field
				}
			}
		}
		if largest == nil || len(*largest) <= len(truncatedMessageMarker) {
			return
		}

		excess := messageBytes(events) - maxBytes
		target := len(*largest) - excess
		if target < len(truncatedMessageMarker) {
			target = len(truncatedMessageMarker)
		}
		if target >= len(*largest) {
			target = len(*largest) - 1
		}
		*largest = truncateMiddle(*largest, target)
	}
}

func messageBytes(events []*agentv1.TaskEvent) int {
	total := 0
	for _, event := range events {
		for _, field := range messageFields(event) {
			total += len(*field)
		}
	}
	return total
}

func messageFields(event *agentv1.TaskEvent) []*string {
	switch payload := event.GetEvent().(type) {
	case *agentv1.TaskEvent_Text:
		if payload.Text != nil {
			return []*string{&payload.Text.Delta}
		}
	case *agentv1.TaskEvent_Thinking:
		if payload.Thinking != nil {
			return []*string{&payload.Thinking.Delta}
		}
	case *agentv1.TaskEvent_UserMessage:
		if payload.UserMessage != nil {
			return []*string{&payload.UserMessage.Text}
		}
	case *agentv1.TaskEvent_Tool:
		if payload.Tool != nil {
			return []*string{&payload.Tool.ArgsJson, &payload.Tool.ResultJson}
		}
	}
	return nil
}

func truncateMiddle(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	if maxBytes <= len(truncatedMessageMarker) {
		return truncatedMessageMarker
	}

	budget := maxBytes - len(truncatedMessageMarker)
	headEnd := budget / 2
	for headEnd > 0 && !utf8.RuneStart(s[headEnd]) {
		headEnd--
	}
	tailStart := len(s) - (budget - budget/2)
	for tailStart < len(s) && !utf8.RuneStart(s[tailStart]) {
		tailStart++
	}
	return s[:headEnd] + truncatedMessageMarker + s[tailStart:]
}

// parseLine decodes one JSONL line into zero or more console events. It is the

func parseLine(line string) []*agentv1.TaskEvent {
	if line == "" {
		return nil
	}
	var entry historyEntry
	if json.Unmarshal([]byte(line), &entry) != nil || entry.Type != "message" {
		return nil
	}
	var msg historyMessage
	if json.Unmarshal(entry.Message, &msg) != nil {
		return nil
	}
	at := parseTime(entry.Timestamp)

	switch msg.Role {
	case "user":
		if text := contentText(msg.Content); text != "" {
			return []*agentv1.TaskEvent{
				{
					At: at,
					Event: &agentv1.TaskEvent_UserMessage{
						UserMessage: userMessage(text),
					},
				},
			}
		}
	case "assistant":
		var blocks []historyBlock
		if json.Unmarshal(msg.Content, &blocks) != nil {
			return nil
		}
		var events []*agentv1.TaskEvent
		for _, block := range blocks {
			switch block.Type {
			case "text":
				if block.Text != "" {
					events = append(events, &agentv1.TaskEvent{
						At:    at,
						Event: &agentv1.TaskEvent_Text{Text: &agentv1.AgentText{Delta: block.Text}},
					})
				}
			case "thinking":
				if block.Thinking != "" {
					events = append(events, &agentv1.TaskEvent{
						At: at,
						Event: &agentv1.TaskEvent_Thinking{
							Thinking: &agentv1.AgentThinking{Delta: block.Thinking},
						},
					})
				}
			case "toolCall":
				events = append(events, &agentv1.TaskEvent{
					At: at,
					Event: &agentv1.TaskEvent_Tool{Tool: &agentv1.ToolExecution{
						ToolCallId: block.ID,
						ToolName:   block.Name,
						Phase:      agentv1.ToolPhase_TOOL_PHASE_STARTED,
						ArgsJson:   string(block.Arguments),
					}},
				})
			}
		}
		return events
	case "toolResult":
		return []*agentv1.TaskEvent{{
			At: at,
			Event: &agentv1.TaskEvent_Tool{Tool: &agentv1.ToolExecution{
				ToolCallId:  msg.ToolCallID,
				ToolName:    msg.ToolName,
				Phase:       agentv1.ToolPhase_TOOL_PHASE_FINISHED,
				ResultJson:  contentText(msg.Content),
				DetailsJson: string(msg.Details),
				IsError:     msg.IsError,
			}},
		}}
	}
	return nil
}

func contentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return single
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	out := ""
	for _, block := range blocks {
		if block.Type == "text" {
			out += block.Text
		}
	}
	return out
}

func parseTime(value string) *timestamppb.Timestamp {
	if value == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return timestamppb.New(t)
}
