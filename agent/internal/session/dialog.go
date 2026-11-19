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
	"log/slog"
	"sync/atomic"
	"time"
)

func (s *Session) askDialog(ctx context.Context, title, prefill string) (string, bool) {
	requestID := fmt.Sprintf("dialog-%d", dialogSeq.Add(1))
	answers := make(chan ExtensionResponse, 1)

	req := extensionUIRequest{
		Type:    "extension_ui_request",
		ID:      requestID,
		Method:  "editor",
		Title:   title,
		Prefill: prefill,
	}
	raw, _ := json.Marshal(req)
	req.RawJSON = string(raw)

	s.mu.Lock()
	if s.dialogs == nil {
		s.dialogs = map[string]chan ExtensionResponse{}
	}
	s.dialogs[requestID] = answers
	s.waiting = true
	s.waitStartedAt = time.Now()
	s.lastDialogJSON = req.RawJSON
	s.pendingExt[requestID] = req
	timeout := s.dialogTimeout
	s.mu.Unlock()

	s.persistWaiting(req.RawJSON)
	s.emit(Event{Kind: EventExtensionUI, At: time.Now(), Extension: extensionEvent(req)})

	var expired <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		expired = timer.C
	}

	select {
	case answer := <-answers:
		s.mu.Lock()
		delete(s.dialogs, requestID)
		delete(s.pendingExt, requestID)
		s.lastDialogJSON = ""
		s.accumulateWaitLocked(time.Now())
		s.waiting = len(s.dialogs) > 0
		s.mu.Unlock()
		s.clearWaitingStore(context.Background())
		if answer.Kind == ExtensionResponseCancelled {
			return "", true
		}
		return answer.Value, false
	case <-expired:
		s.closeDialog(requestID)
		return "", true
	case <-ctx.Done():
		s.closeDialog(requestID)
		return "", true
	}
}

func (s *Session) closeDialog(requestID string) {
	s.mu.Lock()
	delete(s.dialogs, requestID)
	delete(s.pendingExt, requestID)
	s.lastDialogJSON = ""
	s.accumulateWaitLocked(time.Now())
	s.waiting = len(s.dialogs) > 0
	s.mu.Unlock()
	s.clearWaitingStore(context.Background())
	s.notifyAgentState("waiting_resolved", StatusInProgress)
}

var dialogSeq atomic.Uint64

func (s *Session) answerLocalDialog(requestID string, resp ExtensionResponse) bool {
	s.mu.Lock()
	answers, ok := s.dialogs[requestID]
	s.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case answers <- resp:
	default:
	}
	return true
}

func (s *Session) resumeWithDialogAnswer(question, answer string) {
	message := "The user answered the questionnaire"
	if question != "" {
		message = "The user answered \"" + question + "\""
	}
	go func() {
		if err := s.promptImages(context.Background(), message+":\n\n"+answer, nil); err != nil {
			slog.Warn("resume after dialog", "session_id", s.ID, "error", err)
		}
	}()
}

func savedDialogTitle(raw string) string {
	var req extensionUIRequest
	if json.Unmarshal([]byte(raw), &req) != nil {
		return ""
	}
	if req.Method != "editor" {
		return ""
	}
	var payload struct {
		Kind      string `json:"kind"`
		Questions []struct {
			Question string `json:"question"`
		} `json:"questions"`
	}
	if json.Unmarshal([]byte(req.Prefill), &payload) != nil {
		return ""
	}
	if len(payload.Questions) == 0 {
		return ""
	}
	return payload.Questions[0].Question
}
