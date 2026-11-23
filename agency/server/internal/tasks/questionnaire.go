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
	"fmt"
	"log"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

type QuestionnaireResponse struct {
	ID        int64
	RequestID string
	ActorID   ids.ActorID
	At        time.Time
}

func (s *Service) RecordQuestionnaireResponse(
	ctx context.Context,
	taskID, requestID string,
	answers int,
) {

	if taskID == "" || answers == 0 {
		return
	}
	actorID := actorIDFrom(ctx)
	at := time.Now().UnixMilli()
	err := s.store.RecordQuestionnaireResponse(
		ctx,
		store.RecordQuestionnaireResponseParams{
			TaskID:    taskID,
			RequestID: requestID,
			ActorID:   nullID(actorID),
			At:        at,
		},
	)
	if err != nil {
		log.Printf(
			"[tasks] failed to record questionnaire response for %s: %v",
			taskID,
			err,
		)
	}
	s.recordLog(
		ctx, actorFrom(ctx), at,
		taskID, LogQuestionnaire, "answered", "", "",
		fmt.Sprintf("Questionnaire answered (%d %s)", answers, answerCount(answers)),
		map[string]any{"request_id": requestID, "answers": answers},
	)
}

func (s *Service) QuestionnaireResponses(
	ctx context.Context,
	taskID string,
) ([]QuestionnaireResponse, error) {
	rows, err := s.store.ListQuestionnaireResponses(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]QuestionnaireResponse, len(rows))
	for i, row := range rows {
		out[i] = QuestionnaireResponse{
			ID:        row.ID,
			RequestID: row.RequestID,
			ActorID:   ids.ActorID(row.ActorID.Int64),
			At:        time.UnixMilli(row.At),
		}
	}
	return out, nil
}

func answerCount(n int) string {
	if n == 1 {
		return "answer"
	}
	return "answers"
}
