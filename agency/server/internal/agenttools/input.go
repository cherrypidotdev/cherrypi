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

package agenttools

import (
	"errors"
	"net/http"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/taskinput"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"connectrpc.com/connect"
)

type taskInputJSON struct {
	TaskID  string `json:"task_id"`
	State   string `json:"state"`
	Waiting bool   `json:"waiting"`

	RequestID   string                   `json:"request_id,omitempty"`
	Method      string                   `json:"method,omitempty"`
	Title       string                   `json:"title,omitempty"`
	Message     string                   `json:"message,omitempty"`
	Placeholder string                   `json:"placeholder,omitempty"`
	Options     []string                 `json:"options,omitempty"`
	Timeout     uint32                   `json:"timeout,omitempty"`
	Question    *taskinput.Questionnaire `json:"questionnaire,omitempty"`

	Raw string `json:"raw,omitempty"`
	// Hint is the one-line instruction for the model: how to answer this

	Hint string `json:"hint,omitempty"`
}

func (h *Handler) taskInput(w http.ResponseWriter, r *http.Request, id string) {
	if h.Input == nil {
		writeError(w, http.StatusServiceUnavailable, "task input is unavailable")
		return
	}
	caller, ok := auth.ActorID(r.Context())
	if !ok || caller == 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	task, err := h.Tasks.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if !h.mayAnswerInput(r, task, caller) {
		writeError(
			w,
			http.StatusForbidden,
			"only a lead of this task may read or answer its input",
		)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getTaskInput(w, r, task)
	case http.MethodPost:
		h.answerTaskInput(w, r, task)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) getTaskInput(
	w http.ResponseWriter,
	r *http.Request,
	task tasks.Task,
) {
	dialog, waiting, err := h.Input.Pending(r.Context(), task)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	out := taskInputJSON{
		TaskID:  task.ID,
		State:   string(task.State),
		Waiting: waiting,
	}
	if !waiting {
		out.Hint = "the task is not waiting for input right now; " +
			"read it again with getTask or getTaskHistory"
		writeJSON(w, http.StatusOK, out)
		return
	}
	out.RequestID = dialog.RequestID
	out.Method = dialog.Method
	out.Title = dialog.Title
	out.Message = dialog.Message
	out.Placeholder = dialog.Placeholder
	out.Options = dialog.Options
	out.Timeout = dialog.Timeout
	out.Question = dialog.Questionnaire
	out.Raw = dialog.Raw
	out.Hint = answerHint(dialog)
	writeJSON(w, http.StatusOK, out)
}

func answerHint(dialog taskinput.Dialog) string {
	switch {
	case dialog.Questionnaire != nil:
		return "answer with \"answers\": one entry per question, " +
			"each addressing the question by its header (or 1-based index) " +
			"and naming an option label, or sending \"custom\""
	case dialog.Method == "confirm":
		return "answer with \"confirmed\": true or false"
	case dialog.Method == "select":
		return "answer with \"value\": exactly one of the listed options"
	case dialog.Method == "input", dialog.Method == "editor":
		return "answer with \"value\": the text to send"
	default:
		return "answer with \"value\""
	}
}

type answerInput struct {
	RequestID string `json:"request_id"`

	Answers []taskinput.LabelAnswer `json:"answers"`

	Note string `json:"note"`

	Value string `json:"value"`

	Confirmed *bool `json:"confirmed"`

	Cancelled bool `json:"cancelled"`
}

type answerResult struct {
	TaskID    string `json:"task_id"`
	Answered  bool   `json:"answered"`
	Cancelled bool   `json:"cancelled,omitempty"`
	State     string `json:"state"`

	Hint string `json:"hint,omitempty"`
}

func (h *Handler) answerTaskInput(
	w http.ResponseWriter,
	r *http.Request,
	task tasks.Task,
) {
	var in answerInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := taskActorContext(r)
	dialog, waiting, err := h.Input.Pending(ctx, task)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if !waiting {
		writeError(
			w,
			http.StatusConflict,
			"the task is not waiting for input right now (it may have been "+
				"answered already); read it again with getTaskInput",
		)
		return
	}
	if strings.TrimSpace(in.RequestID) != "" &&
		strings.TrimSpace(in.RequestID) != dialog.RequestID {
		writeError(
			w,
			http.StatusConflict,
			"request_id "+strings.TrimSpace(in.RequestID)+
				" does not match the pending dialog ("+dialog.RequestID+
				"); read it again with getTaskInput",
		)
		return
	}

	answer, err := dialogAnswer(dialog, in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Input.Answer(ctx, task, answer); err != nil {
		writeInputError(w, err)
		return
	}
	task, err = h.Tasks.Get(ctx, task.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := answerResult{
		TaskID:    task.ID,
		Answered:  true,
		Cancelled: answer.Cancelled,
		State:     string(task.State),
	}
	if !answer.Cancelled {
		out.Hint = "the task continues from your answer; its run is in flight again"
	}
	writeJSON(w, http.StatusOK, out)
}

func dialogAnswer(
	dialog taskinput.Dialog,
	in answerInput,
) (taskinput.Answer, error) {
	answer := taskinput.Answer{RequestID: dialog.RequestID}
	if in.Cancelled {
		if len(in.Answers) > 0 || in.Value != "" || in.Confirmed != nil {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "cancelled cannot be combined with an answer; " +
					"send either cancelled or the dialog's answer",
			}
		}
		answer.Cancelled = true
		return answer, nil
	}
	if dialog.Questionnaire != nil {
		if in.Confirmed != nil || in.Value != "" {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a questionnaire; answer it with " +
					"\"answers\" (question + option label, or custom), not " +
					"\"value\" or \"confirmed\"",
			}
		}
		resolved, err := dialog.Questionnaire.AnswerValue(in.Answers, in.Note)
		if err != nil {
			return taskinput.Answer{}, err
		}
		resolved.RequestID = dialog.RequestID
		return resolved, nil
	}
	switch dialog.Method {
	case "confirm":
		if in.Confirmed == nil {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a confirm; answer it with " +
					"\"confirmed\": true or false",
			}
		}
		if len(in.Answers) > 0 || in.Value != "" || in.Note != "" {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "a confirm dialog takes only \"confirmed\"; drop the " +
					"other fields",
			}
		}
		confirmed := *in.Confirmed
		answer.Confirmed = &confirmed
		return answer, nil
	case "select", "input", "editor":
		if len(in.Answers) > 0 {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a plain " + dialog.Method +
					"; answer it with \"value\", not \"answers\"",
			}
		}
		if in.Confirmed != nil {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a plain " + dialog.Method +
					"; answer it with \"value\", not \"confirmed\"",
			}
		}
		if dialog.Method == "select" && strings.TrimSpace(in.Value) == "" {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog is a select; answer it with \"value\" " +
					"set to one of: " + optionList(dialog.Options),
			}
		}
		answer.Value = in.Value
		return answer, nil
	default:
		if len(in.Answers) > 0 {
			return taskinput.Answer{}, &taskinput.ValidationError{
				Message: "this dialog does not take \"answers\"; answer it " +
					"with \"value\" or \"confirmed\"",
			}
		}
		answer.Value = in.Value
		if in.Confirmed != nil {
			confirmed := *in.Confirmed
			answer.Confirmed = &confirmed
		}
		return answer, nil
	}
}

func optionList(options []string) string {
	if len(options) == 0 {
		return "no options"
	}
	quoted := make([]string, 0, len(options))
	for _, option := range options {
		quoted = append(quoted, "\""+option+"\"")
	}
	return strings.Join(quoted, ", ")
}

func (h *Handler) mayAnswerInput(
	r *http.Request,
	task tasks.Task,
	caller ids.ActorID,
) bool {
	allowed, err := h.Tasks.CanAnswerInput(r.Context(), task, caller, false)
	return err == nil && allowed
}

func inputErrorCode(err error) int {
	var invalid *taskinput.ValidationError
	switch {
	case errors.As(err, &invalid):
		return http.StatusBadRequest
	case errors.Is(err, taskinput.ErrNoPendingDialog):
		return http.StatusConflict
	case connect.CodeOf(err) == connect.CodeFailedPrecondition:
		return http.StatusConflict
	default:
		return http.StatusBadGateway
	}
}

func writeInputError(w http.ResponseWriter, err error) {
	writeError(w, inputErrorCode(err), err.Error())
}
