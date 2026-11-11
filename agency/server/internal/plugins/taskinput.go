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

package plugins

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

type TaskInputAnswer struct {
	Kind          string
	Value         string
	Confirmed     bool
	Questionnaire []QuestionnaireAnswer
}

type QuestionnaireAnswer struct {
	Question string
	Answer   string
}

const (
	InputAnswerValue     = "value"
	InputAnswerConfirmed = "confirmed"
	InputAnswerCancelled = "cancelled"
)

func (s *Service) RespondTaskInput(
	ctx context.Context,
	pluginID, taskID, requestID string,
	answer TaskInputAnswer,
) error {
	p, err := s.Get(ctx, pluginID)
	if err != nil {
		return err
	}
	if !p.Enabled {
		return errors.New("plugin is disabled")
	}
	taskID = strings.TrimSpace(taskID)
	requestID = strings.TrimSpace(requestID)
	if taskID == "" {
		return errors.New("task_id is required")
	}
	if requestID == "" {
		return errors.New("request_id is required")
	}
	switch answer.Kind {
	case InputAnswerValue, InputAnswerConfirmed, InputAnswerCancelled:
	default:
		return fmt.Errorf("unknown input answer kind %q", answer.Kind)
	}
	if s.Tasks == nil {
		return errors.New("tasks are unavailable")
	}
	t, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if t.State != tasks.StateInput {
		return fmt.Errorf("task %s is not waiting for input", taskID)
	}
	allowed, err := s.Tasks.CanAnswerInput(ctx, t, ids.ActorID(p.ActorID), false)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("only a lead or an administrator may answer this task")
	}
	if s.RespondInput == nil {
		return errors.New("this agency cannot answer task input")
	}
	return s.RespondInput(ctx, p, t, requestID, answer)
}
