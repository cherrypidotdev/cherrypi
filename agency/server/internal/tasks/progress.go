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

package tasks

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	maxProgressItems = 50
	maxProgressChars = 240
)

type ProgressItemStatus int64

const (
	ProgressItemPending    ProgressItemStatus = 1
	ProgressItemInProgress ProgressItemStatus = 2
	ProgressItemDone       ProgressItemStatus = 3
)

func (s ProgressItemStatus) String() string {
	switch s {
	case ProgressItemInProgress:
		return "in_progress"
	case ProgressItemDone:
		return "done"
	default:
		return "pending"
	}
}

type ProgressItem struct {
	Text   string
	Status ProgressItemStatus
}

type Progress struct {
	Summary   string
	Items     []ProgressItem
	Done      int
	Total     int
	UpdatedAt time.Time
}

func (s *Service) SetProgress(
	ctx context.Context,
	taskID, summary string,
	items []ProgressItem,
) (Task, error) {
	if err := s.applyProgress(ctx, taskID, summary, items); err != nil {
		return Task{}, err
	}
	return s.Get(ctx, taskID)
}

func (s *Service) applyProgress(
	ctx context.Context,
	taskID, summary string,
	items []ProgressItem,
) error {
	clean := normalizeProgress(summary, items)
	params := store.SetTaskPlanParams{
		TaskID:    taskID,
		Summary:   clean.Summary,
		UpdatedAt: time.Now().UnixMilli(),
	}
	for i, item := range clean.Items {
		params.Items = append(params.Items, store.TaskPlanItem{
			Position: int64(i),
			Text:     item.Text,
			Status:   store.ProgressItemStatus(item.Status),
		})
	}
	return s.store.SetTaskProgress(ctx, params)
}

func (s *Service) SetProgressBySession(
	ctx context.Context,
	agentID ids.AgentID,
	sessionID, summary string,
	items []ProgressItem,
) error {
	taskID, err := s.store.TaskIDBySession(ctx, store.TaskIDBySessionParams{
		AgentID:   int64(agentID),
		SessionID: sessionID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.applyProgress(ctx, taskID, summary, items)
}

func (s *Service) progressForTask(ctx context.Context, taskID string) (Progress, error) {
	plan, err := s.store.GetTaskProgress(ctx, taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return emptyProgress(), nil
	}
	if err != nil {
		return Progress{}, err
	}
	return toProgress(plan), nil
}

func (s *Service) allProgress(ctx context.Context) (map[string]Progress, error) {
	plans, err := s.store.ListTaskProgress(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Progress, len(plans))
	for taskID, plan := range plans {
		out[taskID] = toProgress(plan)
	}
	return out, nil
}

func emptyProgress() Progress {
	return Progress{Items: []ProgressItem{}}
}

func progressOrEmpty(progress Progress) Progress {
	if progress.Items == nil {
		progress.Items = []ProgressItem{}
	}
	return progress
}

func toProgress(plan store.TaskPlan) Progress {
	out := Progress{
		Summary: plan.Summary,
		Items:   make([]ProgressItem, 0, len(plan.Items)),
	}
	if plan.UpdatedAt != 0 {
		out.UpdatedAt = time.UnixMilli(plan.UpdatedAt)
	}
	for _, item := range plan.Items {
		status := ProgressItemStatus(item.Status)
		if status != ProgressItemInProgress && status != ProgressItemDone {
			status = ProgressItemPending
		}
		out.Items = append(out.Items, ProgressItem{Text: item.Text, Status: status})
		if status == ProgressItemDone {
			out.Done++
		}
	}
	out.Total = len(out.Items)
	return out
}

func normalizeProgress(summary string, items []ProgressItem) Progress {
	out := Progress{Summary: truncateProgressText(summary)}
	for _, item := range items {
		text := truncateProgressText(item.Text)
		if text == "" {
			continue
		}
		status := item.Status
		if status != ProgressItemInProgress && status != ProgressItemDone {
			status = ProgressItemPending
		}
		out.Items = append(out.Items, ProgressItem{Text: text, Status: status})
		if status == ProgressItemDone {
			out.Done++
		}
		if len(out.Items) == maxProgressItems {
			break
		}
	}
	out.Total = len(out.Items)
	return out
}

func truncateProgressText(text string) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= maxProgressChars {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:maxProgressChars]))
}
