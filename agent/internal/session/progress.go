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
	"strconv"
	"strings"
	"time"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
)

// Task progress is CherryPi's plan: an optional one-line summary plus an

const (
	maxProgressItems = 50
	maxProgressChars = 240
	maxSettleNudges  = 2
)

const (
	ProgressPending    = "pending"
	ProgressInProgress = "in_progress"
	ProgressDone       = "done"
)

type ProgressTodo struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

type Progress struct {
	Summary string         `json:"summary"`
	Todos   []ProgressTodo `json:"todos"`
}

const carryOn = " Then carry on with the work immediately in the same turn: a " +
	"plan update is bookkeeping, never a stopping point, so do not end the run " +
	"right after updating the plan."

func clampProgress(value string) string {
	text := strings.TrimSpace(value)
	if runes := []rune(text); len(runes) > maxProgressChars {
		return strings.TrimSpace(string(runes[:maxProgressChars]))
	}
	return text
}

func normalizeProgressStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "in_progress", "in-progress", "inprogress":
		return ProgressInProgress
	case "done", "completed":
		return ProgressDone
	default:
		return ProgressPending
	}
}

func normalizeProgress(summary string, todos []ProgressTodo) Progress {
	out := Progress{Summary: clampProgress(summary)}
	for _, item := range todos {
		text := clampProgress(item.Text)
		if text == "" {
			continue
		}
		out.Todos = append(out.Todos, ProgressTodo{
			Text:   text,
			Status: normalizeProgressStatus(item.Status),
		})
		if len(out.Todos) == maxProgressItems {
			break
		}
	}
	return out
}

func renderProgress(heading string, progress Progress) string {
	if len(progress.Todos) == 0 {
		if progress.Summary != "" {
			return heading + ": no plan. Summary: " + progress.Summary
		}
		return heading + ": no plan yet."
	}
	done, active := 0, 0
	lines := make([]string, 0, len(progress.Todos))
	for _, item := range progress.Todos {
		mark := "[ ]"
		switch item.Status {
		case ProgressDone:
			mark = "[x]"
			done++
		case ProgressInProgress:
			mark = "[~]"
			active++
		}
		lines = append(lines, mark+" "+item.Text)
	}
	summary := ""
	if progress.Summary != "" {
		summary = " — " + progress.Summary
	}
	out := heading + ": " + strconv.Itoa(
		done,
	) + "/" + strconv.Itoa(
		len(progress.Todos),
	) + " done" + summary +
		"\n" + strings.Join(
		lines,
		"\n",
	)
	if active > 1 {
		out += "\nNote: " + strconv.Itoa(
			active,
		) + " items are in_progress; prefer a single current step."
	}
	return out
}

func progressReminder(content string) ai.Message {

	return ai.NewSystemText(content, time.Now().UnixMilli())
}

func (s *Session) progressState() Progress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

func (s *Session) setProgress(progress Progress) {
	s.mu.Lock()
	s.progress = progress
	s.progressTouchedRun = true

	s.progressPlanCovers = s.progressPlanOnly
	s.progressCoveredBlock = nil
	s.mu.Unlock()
	s.notify("task_progress", map[string]any{
		"summary": progress.Summary,
		"todos":   progress.Todos,
	})
}

func (s *Session) beforeToolCall(
	ctx context.Context,
	c agent.BeforeToolCallContext,
) *agent.BeforeToolCallResult {

	if blocked := s.requiredToolGate(ctx, c); blocked != nil {
		return blocked
	}

	if c.ToolCall.Name == "reportTaskProgress" {
		s.notePlanBlock(c.AssistantMessage)
		return nil
	}
	if blockStartsWithPlan(c.AssistantMessage) {
		s.notePlanBlock(c.AssistantMessage)
		return nil
	}
	if s.consumePlanCover(c.AssistantMessage) {
		return nil
	}
	return planGateBlock()
}

func (s *Session) notePlanBlock(message *ai.AssistantMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progressPlanOnly = planOnlyBlock(message)
}

func (s *Session) consumePlanCover(message *ai.AssistantMessage) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if message != nil && message == s.progressCoveredBlock {
		return true
	}
	if !s.progressPlanCovers {
		return false
	}
	s.progressPlanCovers = false
	s.progressCoveredBlock = message
	return true
}

func blockStartsWithPlan(message *ai.AssistantMessage) bool {
	if message == nil {
		return false
	}
	for _, content := range message.Content {
		if call, ok := content.(ai.ToolCall); ok {
			return call.Name == "reportTaskProgress"
		}
	}
	return false
}

func planOnlyBlock(message *ai.AssistantMessage) bool {
	if message == nil {
		return false
	}
	for _, content := range message.Content {
		if call, ok := content.(ai.ToolCall); ok && call.Name != "reportTaskProgress" {
			return false
		}
	}
	return true
}

func planGateBlock() *agent.BeforeToolCallResult {
	return &agent.BeforeToolCallResult{
		Block: true,
		Reason: "Plan reminder: a block of tool calls must begin with " +
			"`reportTaskProgress`. Call it first, in the same message as the " +
			"tools, with the complete list (3–8 ordered items, mark finished " +
			"ones `done` and the current one `in_progress`), then call the tool." +
			carryOn,
	}
}

func (s *Session) shouldContinueBeforeSettle() (ai.Message, bool) {
	progress := s.progressState()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.progressSettleNudges >= maxSettleNudges {
		return nil, false
	}
	missing := len(progress.Todos) == 0
	unfinished := false
	for _, item := range progress.Todos {
		if item.Status != ProgressDone {
			unfinished = true
			break
		}
	}
	stale := !s.progressTouchedRun
	if !missing && !unfinished && !stale {
		return nil, false
	}
	s.progressSettleNudges++
	reason := "you did not update the plan during this run"
	switch {
	case missing:
		reason = "you have not published a plan for this task"
	case unfinished:
		reason = "your plan still has unfinished items"
	}
	return progressReminder("Plan reminder: " + reason + ". Before finishing, call " +
		"`reportTaskProgress` with the complete list: mark every completed step " +
		"`done`, leave nothing you finished as `pending`, and replace the list if " +
		"the task changed." + carryOn), true
}

func (s *Session) beginPlanRun() []agent.AgentMessage {
	progress := s.progressState()
	s.mu.Lock()
	s.progressTouchedRun = false
	s.progressPlanOnly = false
	s.progressPlanCovers = false
	s.progressCoveredBlock = nil
	s.progressSettleNudges = 0
	s.mu.Unlock()
	if len(progress.Todos) > 0 {
		return nil
	}
	return []agent.AgentMessage{
		progressReminder("Plan reminder: this task has no plan yet. Publish one with " +
			"`reportTaskProgress` before doing the work: 3–8 ordered items, each " +
			"`pending` (a genuinely one-step task gets one item). Every block of " +
			"tool calls must begin with it, so keep it current: mark an item " +
			"`in_progress` when you begin it and `done` when it lands."),
	}
}
