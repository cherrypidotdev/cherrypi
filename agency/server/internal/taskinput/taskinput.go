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

package taskinput

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
)

type Connector interface {
	Connection(agentID ids.AgentID) (*tunnel.Connection, error)
}

type ActorInfo struct {
	Username string
	Name     string
	Kind     string
}

type ActorResolver func(ctx context.Context, id ids.ActorID) (ActorInfo, error)

type Service struct {
	Tasks  *tasks.Service
	Agents *agents.Registry
	Hub    Connector
	Actors ActorResolver

	OnAnswered func(ctx context.Context, answered Answered)
}

type Answered struct {
	Task    tasks.Task
	Dialog  Dialog
	Answer  Answer
	Actor   ActorInfo
	ActorID ids.ActorID
}

func (a Answered) Summary() AnswerSummary {
	return Summarize(a.Dialog, a.Answer)
}

func (a Answered) AnsweredBy() map[string]any {
	out := map[string]any{
		"id":       "",
		"username": "",
		"name":     "",
		"kind":     "",
	}
	if a.ActorID != 0 {
		out["id"] = actorid.Format(a.ActorID)
	}
	if a.Actor.Username != "" {
		out["username"] = a.Actor.Username
	}
	if a.Actor.Name != "" {
		out["name"] = a.Actor.Name
	}
	if a.Actor.Kind != "" {
		out["kind"] = a.Actor.Kind
	}
	return out
}

type AnswerSummary struct {
	Text          string                `json:"text"`
	Value         string                `json:"value,omitempty"`
	Confirmed     *bool                 `json:"confirmed,omitempty"`
	Cancelled     bool                  `json:"cancelled,omitempty"`
	Questionnaire []QuestionnaireAnswer `json:"questionnaire,omitempty"`
}

func Summarize(dialog Dialog, in Answer) AnswerSummary {
	out := AnswerSummary{Value: in.Value}
	switch {
	case in.Cancelled:
		out.Cancelled = true
		out.Text = "Cancelled"
	case in.Confirmed != nil:
		out.Confirmed = in.Confirmed
		if *in.Confirmed {
			out.Text = "Yes"
		} else {
			out.Text = "No"
		}
	default:
		pairs := in.Questionnaire
		if len(pairs) == 0 {
			pairs = questionnaireAnswersFromValue(dialog.Questionnaire, in.Value)
		}
		out.Questionnaire = pairs
		if len(pairs) > 0 {
			out.Text = summarizePairs(pairs)
		} else {
			out.Text = in.Value
		}
	}
	return out
}

func questionnaireAnswersFromValue(
	q *Questionnaire,
	value string,
) []QuestionnaireAnswer {
	if q == nil || strings.TrimSpace(value) == "" {
		return nil
	}
	var payload struct {
		Answers []struct {
			Index   int     `json:"index"`
			Options []int   `json:"options"`
			Custom  *string `json:"custom"`
		} `json:"answers"`
	}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return nil
	}
	out := make([]QuestionnaireAnswer, 0, len(payload.Answers))
	for _, item := range payload.Answers {
		if item.Index < 0 || item.Index >= len(q.Questions) {
			continue
		}
		question := q.Questions[item.Index]
		label := strings.TrimSpace(question.Header)
		if label == "" {
			label = strings.TrimSpace(question.Question)
		}
		answer := ""
		if item.Custom != nil {
			answer = *item.Custom
		} else {
			labels := make([]string, 0, len(item.Options))
			for _, index := range item.Options {
				if index >= 0 && index < len(question.Options) {
					labels = append(labels, question.Options[index].Label)
				}
			}
			answer = strings.Join(labels, ", ")
		}
		out = append(out, QuestionnaireAnswer{Question: label, Answer: answer})
	}
	return out
}

func summarizePairs(pairs []QuestionnaireAnswer) string {
	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		parts = append(parts, strings.TrimSpace(pair.Question)+": "+pair.Answer)
	}
	return strings.Join(parts, "; ")
}

type Dialog struct {
	RequestID string `json:"request_id"`

	Method      string   `json:"method"`
	Title       string   `json:"title,omitempty"`
	Message     string   `json:"message,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []string `json:"options,omitempty"`
	Timeout     uint32   `json:"timeout,omitempty"`

	Questionnaire *Questionnaire `json:"questionnaire,omitempty"`

	Raw string `json:"raw,omitempty"`
}

var ErrNoPendingDialog = errors.New("task has no pending input dialog")

type pendingDialogJSON struct {
	ID          string   `json:"id"`
	Method      string   `json:"method"`
	Title       string   `json:"title"`
	Message     string   `json:"message"`
	Placeholder string   `json:"placeholder"`
	Prefill     string   `json:"prefill"`
	Options     []string `json:"options"`
	Timeout     uint32   `json:"timeout"`
}

func (s *Service) Pending(
	ctx context.Context,
	t tasks.Task,
) (Dialog, bool, error) {
	if t.SessionID == "" || t.AssigneeActorID == 0 {
		return Dialog{}, false, nil
	}
	conn, err := s.connection(ctx, t.AssigneeActorID)
	if err != nil {
		return Dialog{}, false, err
	}
	return s.pendingFor(ctx, conn, t.SessionID)
}

func (s *Service) pendingFor(
	ctx context.Context,
	conn *tunnel.Connection,
	sessionID string,
) (Dialog, bool, error) {
	res, err := conn.Tasks.ListTasks(ctx, &agentv1.ListTasksRequest{})
	if err != nil {
		return Dialog{}, false, err
	}
	for _, at := range res.GetTasks() {
		if at.GetId() != sessionID {
			continue
		}
		if !at.GetWaitingForInput() {
			return Dialog{}, false, nil
		}
		return ParseDialog(at.GetPendingDialogJson()), true, nil
	}
	return Dialog{}, false, nil
}

func ParseDialog(raw string) Dialog {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Dialog{}
	}
	var in pendingDialogJSON
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return Dialog{Raw: raw}
	}
	d := Dialog{
		RequestID:   in.ID,
		Method:      in.Method,
		Title:       in.Title,
		Message:     in.Message,
		Placeholder: in.Placeholder,
		Options:     in.Options,
		Timeout:     in.Timeout,
		Raw:         raw,
	}
	d.Questionnaire = ParseQuestionnaire(in.Prefill)
	return d
}

type Answer struct {
	RequestID string

	Value string

	Confirmed *bool

	Cancelled bool

	Questionnaire []QuestionnaireAnswer
}

type QuestionnaireAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type answeredBy struct {
	ID string `json:"id"`

	Kind string `json:"kind,omitempty"`

	Username string `json:"username,omitempty"`
	Name     string `json:"name,omitempty"`
}

func (s *Service) Answer(
	ctx context.Context,
	t tasks.Task,
	in Answer,
) error {
	if t.SessionID == "" || t.AssigneeActorID == 0 {
		return ErrNoPendingDialog
	}
	conn, err := s.connection(ctx, t.AssigneeActorID)
	if err != nil {
		return err
	}

	dialog, pending, pendingErr := s.pendingFor(ctx, conn, t.SessionID)
	if pendingErr != nil {
		log.Printf("[task %s] read pending dialog: %v", t.ID, pendingErr)
	}
	if pending && dialog.RequestID == "" {
		dialog.RequestID = in.RequestID
	}
	inner := &agentv1.RespondToExtensionUiRequest{
		Id:        t.SessionID,
		RequestId: in.RequestID,
	}
	switch {
	case in.Cancelled:
		inner.Response = &agentv1.RespondToExtensionUiRequest_Cancelled{
			Cancelled: true,
		}
	case in.Confirmed != nil:
		inner.Response = &agentv1.RespondToExtensionUiRequest_Confirmed{
			Confirmed: *in.Confirmed,
		}
	default:
		inner.Response = &agentv1.RespondToExtensionUiRequest_Value{
			Value: in.Value,
		}
	}
	if len(in.Questionnaire) > 0 {
		inner.Questionnaire = make(
			[]*agentv1.QuestionnaireAnswer, 0, len(in.Questionnaire),
		)
		for _, qa := range in.Questionnaire {
			inner.Questionnaire = append(
				inner.Questionnaire,
				&agentv1.QuestionnaireAnswer{
					Question: qa.Question,
					Answer:   qa.Answer,
				},
			)
		}
	}

	answerer := s.resolveAnswerer(ctx)

	stamped := s.stamp(inner, answerer.stamp)
	if _, err := conn.Tasks.RespondToExtensionUi(ctx, inner); err != nil {
		return err
	}
	if count := max(stamped, len(in.Questionnaire)); count > 0 {
		s.record(ctx, t, in.RequestID, count)
	}

	if _, err := s.Tasks.ClearInput(ctx, t.ID); err != nil {
		log.Printf("[task %s] clear input: %v", t.ID, err)
	}
	if pending && s.OnAnswered != nil {
		s.OnAnswered(ctx, Answered{
			Task:    t,
			Dialog:  dialog,
			Answer:  in,
			Actor:   answerer.info,
			ActorID: answerer.id,
		})
	}
	return nil
}

type answerer struct {
	id    ids.ActorID
	info  ActorInfo
	stamp *answeredBy
}

func (s *Service) resolveAnswerer(ctx context.Context) answerer {
	actorID, ok := auth.ActorID(ctx)
	if !ok || actorID == 0 {
		return answerer{}
	}
	out := answerer{
		id:    actorID,
		stamp: &answeredBy{ID: actorid.Format(actorID)},
	}
	if s.Actors != nil {
		if info, err := s.Actors(ctx, actorID); err == nil {
			out.info = info
			out.stamp.Username = info.Username
			out.stamp.Name = info.Name
			out.stamp.Kind = info.Kind
		}
	}
	return out
}

func (s *Service) stamp(
	inner *agentv1.RespondToExtensionUiRequest,
	by *answeredBy,
) int {
	value, ok := inner.Response.(*agentv1.RespondToExtensionUiRequest_Value)
	if !ok || by == nil {
		return 0
	}
	stamped, ok := stampAnsweredBy(value.Value, by)
	if !ok {
		return 0
	}
	value.Value = stamped
	return answerCount(stamped)
}

func (s *Service) record(
	ctx context.Context,
	t tasks.Task,
	requestID string,
	answers int,
) {
	s.Tasks.RecordQuestionnaireResponse(ctx, t.ID, requestID, answers)
}

func answerCount(value string) int {
	var payload struct {
		Answers []json.RawMessage `json:"answers"`
	}
	if json.Unmarshal([]byte(value), &payload) != nil {
		return 0
	}
	return len(payload.Answers)
}

func (s *Service) connection(
	ctx context.Context,
	actorID ids.ActorID,
) (*tunnel.Connection, error) {
	if s.Agents == nil {
		return nil, errors.New("agent registry is not configured")
	}
	if s.Hub == nil {
		return nil, errors.New("agent tunnel is not configured")
	}
	agent, err := s.Agents.GetByActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return s.Hub.Connection(agent.ID)
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

type Questionnaire struct {
	Questions []Question `json:"questions"`
}

type Question struct {
	Question    string    `json:"question"`
	Header      string    `json:"header,omitempty"`
	MultiSelect bool      `json:"multi_select,omitempty"`
	Options     []QOption `json:"options"`
}

type QOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Recommended bool   `json:"recommended,omitempty"`
}

type questionnairePrefill struct {
	Kind      string `json:"kind"`
	Questions []struct {
		Question    string `json:"question"`
		Header      string `json:"header"`
		MultiSelect bool   `json:"multiSelect"`
		Options     []struct {
			Label       string `json:"label"`
			Description string `json:"description"`
			Recommended bool   `json:"recommended"`
		} `json:"options"`
	} `json:"questions"`
}

func ParseQuestionnaire(prefill string) *Questionnaire {
	prefill = strings.TrimSpace(prefill)
	if prefill == "" {
		return nil
	}
	var raw questionnairePrefill
	if err := json.Unmarshal([]byte(prefill), &raw); err != nil {
		return nil
	}
	if raw.Kind != "cherrypi.questionnaire" || len(raw.Questions) == 0 {
		return nil
	}
	out := &Questionnaire{Questions: make([]Question, 0, len(raw.Questions))}
	for _, q := range raw.Questions {
		question := Question{
			Question:    q.Question,
			Header:      q.Header,
			MultiSelect: q.MultiSelect,
			Options:     make([]QOption, 0, len(q.Options)),
		}
		for _, o := range q.Options {
			question.Options = append(question.Options, QOption{
				Label:       o.Label,
				Description: o.Description,
				Recommended: o.Recommended,
			})
		}
		out.Questions = append(out.Questions, question)
	}
	return out
}

type LabelAnswer struct {
	Question string `json:"question"`

	Options []string `json:"options"`

	Custom *string `json:"custom"`

	Notes string `json:"notes"`
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

func (q *Questionnaire) AnswerValue(
	answers []LabelAnswer,
	note string,
) (Answer, error) {
	if q == nil {
		return Answer{}, invalid(
			"this dialog is not a questionnaire; answer it with a value",
		)
	}
	if len(answers) == 0 {
		return Answer{}, invalid(
			"no answers were sent; answer every question (%s)",
			q.questionList(),
		)
	}
	byIndex := map[int]wireAnswer{}
	order := make([]int, 0, len(answers))
	for _, in := range answers {
		index, err := q.resolveQuestion(in.Question)
		if err != nil {
			return Answer{}, err
		}
		if _, dup := byIndex[index]; dup {
			return Answer{}, invalid(
				"question %s is answered twice; send one answer per question",
				q.describe(index),
			)
		}
		wire, err := q.resolveAnswer(index, in)
		if err != nil {
			return Answer{}, err
		}
		byIndex[index] = wire
		order = append(order, index)
	}
	missing := make([]string, 0)
	for index := range q.Questions {
		if _, ok := byIndex[index]; !ok {
			missing = append(missing, q.describe(index))
		}
	}
	if len(missing) > 0 {
		return Answer{}, invalid(
			"missing an answer for %s; every question needs one "+
				"(send \"custom\" for a free-form answer)",
			strings.Join(missing, ", "),
		)
	}
	sort.Ints(order)
	payload := make([]wireAnswer, 0, len(order))
	pairs := make([]QuestionnaireAnswer, 0, len(order))
	for _, index := range order {
		wire := byIndex[index]
		payload = append(payload, wire)
		pairs = append(pairs, QuestionnaireAnswer{
			Question: q.Questions[index].Question,
			Answer:   q.answerLabel(wire),
		})
	}
	value := struct {
		Answers []wireAnswer `json:"answers"`
		Note    string       `json:"note,omitempty"`
	}{Answers: payload, Note: strings.TrimSpace(note)}
	encoded, err := json.Marshal(value)
	if err != nil {
		return Answer{}, err
	}
	return Answer{Value: string(encoded), Questionnaire: pairs}, nil
}

type wireAnswer struct {
	Index   int     `json:"index"`
	Options []int   `json:"options,omitempty"`
	Custom  *string `json:"custom,omitempty"`
	Notes   string  `json:"notes,omitempty"`
	labels  []string
	custom  bool
}

func (q *Questionnaire) resolveQuestion(ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0, invalid(
			"an answer has no \"question\"; address each question by its "+
				"header, its 1-based index, or its text (%s)",
			q.questionList(),
		)
	}
	if n, ok := parseIndex(ref); ok {
		if n < 1 || n > len(q.Questions) {
			return 0, invalid(
				"question index %s is out of range; the questionnaire has "+
					"%d questions (%s)",
				ref, len(q.Questions), q.questionList(),
			)
		}
		return n - 1, nil
	}
	normalized := normalize(ref)
	matches := make([]int, 0, 1)
	for index, question := range q.Questions {
		if normalize(question.Header) == normalized ||
			normalize(question.Question) == normalized {
			matches = append(matches, index)
		}
	}
	switch len(matches) {
	case 0:
		return 0, invalid(
			"no question matches %q; address each question by its header, "+
				"its 1-based index, or its text (%s)",
			ref, q.questionList(),
		)
	case 1:
		return matches[0], nil
	default:
		labels := make([]string, 0, len(matches))
		for _, index := range matches {
			labels = append(labels, q.describe(index))
		}
		return 0, invalid(
			"%q matches %d questions (%s); use the 1-based index instead",
			ref, len(matches), strings.Join(labels, ", "),
		)
	}
}

func (q *Questionnaire) resolveAnswer(
	index int,
	in LabelAnswer,
) (wireAnswer, error) {
	question := q.Questions[index]
	wire := wireAnswer{
		Index:  index,
		Notes:  strings.TrimSpace(in.Notes),
		custom: in.Custom != nil,
	}
	if in.Custom != nil {
		if len(in.Options) > 0 {
			return wireAnswer{}, invalid(
				"question %s has both \"options\" and \"custom\"; send one or "+
					"the other",
				q.describe(index),
			)
		}
		custom := *in.Custom
		wire.Custom = &custom
		return wire, nil
	}
	if len(in.Options) == 0 {
		return wireAnswer{}, invalid(
			"question %s has no answer; send one of its option labels "+
				"(%s) or \"custom\" for a free-form answer",
			q.describe(index), question.optionList(),
		)
	}
	if !question.MultiSelect && len(in.Options) > 1 {
		return wireAnswer{}, invalid(
			"question %s takes a single option but %d were sent (%s); "+
				"send exactly one",
			q.describe(index), len(in.Options),
			strings.Join(in.Options, ", "),
		)
	}
	chosen := make([]int, 0, len(in.Options))
	labels := make([]string, 0, len(in.Options))
	for _, label := range in.Options {
		optionIndex, ok := question.findOption(label)
		if !ok {
			return wireAnswer{}, invalid(
				"question %s has no option %q; its options are %s (or send "+
					"\"custom\" for a free-form answer)",
				q.describe(index), strings.TrimSpace(label),
				question.optionList(),
			)
		}
		if containsInt(chosen, optionIndex) {
			continue
		}
		chosen = append(chosen, optionIndex)
		labels = append(labels, question.Options[optionIndex].Label)
	}
	sort.Ints(chosen)
	sort.Slice(labels, func(a, b int) bool { return labels[a] < labels[b] })
	wire.Options = chosen
	wire.labels = labels
	return wire, nil
}

func (q Question) findOption(label string) (int, bool) {
	normalized := normalize(label)
	for index, option := range q.Options {
		if normalize(option.Label) == normalized {
			return index, true
		}
	}
	return 0, false
}

func (q Question) optionList() string {
	labels := make([]string, 0, len(q.Options))
	for _, option := range q.Options {
		labels = append(labels, fmt.Sprintf("%q", option.Label))
	}
	return strings.Join(labels, ", ")
}

func (q *Questionnaire) describe(index int) string {
	question := q.Questions[index]
	if header := strings.TrimSpace(question.Header); header != "" {
		return fmt.Sprintf("%d (%q)", index+1, header)
	}
	text := strings.TrimSpace(question.Question)
	if len(text) > 40 {
		text = text[:39] + "…"
	}
	return fmt.Sprintf("%d (%q)", index+1, text)
}

func (q *Questionnaire) questionList() string {
	parts := make([]string, 0, len(q.Questions))
	for index := range q.Questions {
		parts = append(parts, q.describe(index))
	}
	return strings.Join(parts, ", ")
}

func (q *Questionnaire) answerLabel(wire wireAnswer) string {
	if wire.Custom != nil {
		return *wire.Custom
	}
	return strings.Join(wire.labels, ", ")
}

func normalize(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func parseIndex(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
		if n > 1<<16 {
			return 0, false
		}
	}
	return n, true
}

func containsInt(values []int, want int) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
