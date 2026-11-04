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
	"encoding/json"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
)

type answeredBy struct {
	ID string `json:"id"`

	Username string `json:"username,omitempty"`
	Name     string `json:"name,omitempty"`
}

func (s *TaskServer) stampQuestionnaireAnswer(
	ctx context.Context,
	inner *agentv1.RespondToExtensionUiRequest,
) bool {
	value, ok := inner.Response.(*agentv1.RespondToExtensionUiRequest_Value)
	if !ok {
		return false
	}
	by := s.answeringActor(ctx)
	if by == nil {
		return false
	}
	stamped, ok := stampAnsweredBy(value.Value, by)
	if !ok {
		return false
	}
	value.Value = stamped
	return true
}

func (s *TaskServer) answeringActor(ctx context.Context) *answeredBy {
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return nil
	}
	by := &answeredBy{ID: actorid.Format(actorID)}
	if s.Actors != nil {
		if info, err := s.Actors(ctx, actorID); err == nil {
			by.Username = info.Username
			by.Name = info.Name
		}
	}
	return by
}

func (s *TaskServer) recordQuestionnaireAnswer(
	ctx context.Context,
	t tasks.Task,
	requestID string,
	answers int,
) {
	s.Tasks.RecordQuestionnaireResponse(ctx, t.ID, requestID, answers)
}

func stampAnsweredBy(value string, by *answeredBy) (string, bool) {
	if value == "" {
		return value, false
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return value, false
	}
	if _, ok := payload["answers"]; !ok {
		return value, false
	}
	raw, err := json.Marshal(by)
	if err != nil {
		return value, false
	}
	payload["answeredBy"] = raw
	out, err := json.Marshal(payload)
	if err != nil {
		return value, false
	}
	return string(out), true
}
