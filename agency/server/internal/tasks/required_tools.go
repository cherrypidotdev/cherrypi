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
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	MaxRequiredTools = 8

	MaxRequiredToolNameLength = 64
)

var requiredToolNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

var (
	ErrRequiredToolsTooMany = fmt.Errorf(
		"at most %d required tools are allowed", MaxRequiredTools,
	)

	ErrInvalidRequiredTool = fmt.Errorf(
		"a tool name must be 1-%d characters of a-z, A-Z, 0-9, '_' or '-'",
		MaxRequiredToolNameLength,
	)
)

func normalizeRequiredTools(tools []string) ([]string, error) {
	if len(tools) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(tools))
	seen := map[string]bool{}
	for _, tool := range tools {
		name := strings.TrimSpace(tool)
		if name == "" {
			continue
		}
		if !requiredToolNamePattern.MatchString(name) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidRequiredTool, name)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	if len(out) > MaxRequiredTools {
		return nil, ErrRequiredToolsTooMany
	}
	return out, nil
}

func sameRequiredTools(a, b []string) bool { return slices.Equal(a, b) }

func parseRequiredTools(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var tools []string
	if err := json.Unmarshal([]byte(raw), &tools); err != nil {
		return nil
	}
	normalized, err := normalizeRequiredTools(tools)
	if err != nil {
		return nil
	}
	return normalized
}

func marshalRequiredTools(tools []string) string {
	if len(tools) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(tools)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func (s *Service) SetRequiredTools(
	ctx context.Context,
	taskID string,
	tools []string,
) (Task, error) {
	current, err := s.Get(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	return s.setRequiredTools(ctx, current, tools)
}

func (s *Service) SetRequiredToolsBySession(
	ctx context.Context,
	agentID ids.AgentID,
	sessionID string,
	tools []string,
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
	_, err = s.SetRequiredTools(ctx, taskID, tools)
	return err
}

func (s *Service) setRequiredTools(
	ctx context.Context,
	task Task,
	tools []string,
) (Task, error) {
	normalized, err := normalizeRequiredTools(tools)
	if err != nil {
		return Task{}, err
	}
	if sameRequiredTools(normalized, task.RequiredTools) {
		return task, nil
	}
	if err := s.store.SetTaskRequiredTools(
		ctx, task.ID, marshalRequiredTools(normalized),
	); err != nil {
		return Task{}, err
	}
	s.changed(ctx, task.ID)
	return s.Get(ctx, task.ID)
}

func (s *Service) ClearRequiredTools(ctx context.Context, taskID string) error {
	current, err := s.Get(ctx, taskID)
	if err != nil {
		return err
	}
	return s.clearRequiredTools(ctx, current)
}

func (s *Service) clearRequiredTools(ctx context.Context, task Task) error {
	if len(task.RequiredTools) == 0 {
		return nil
	}
	if err := s.store.SetTaskRequiredTools(ctx, task.ID, "[]"); err != nil {
		return err
	}
	s.changed(ctx, task.ID)
	return nil
}

func (s *Service) ClearRequiredToolsBySession(
	ctx context.Context,
	agentID ids.AgentID,
	sessionID string,
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
	return s.ClearRequiredTools(ctx, taskID)
}
