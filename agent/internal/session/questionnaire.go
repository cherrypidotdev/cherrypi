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
	"strconv"
	"strings"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
)

const (
	questionnaireDialogKind = "cherrypi.questionnaire"

	questionnaireUnavailableKind = "cherrypi.questionnaire.unavailable"

	questionnaireNonInteractiveReason = "this task is non-interactive, so the " +
		"questionnaire was not shown. Ask the user in plain text instead."
	questionnaireNoUIError = "Error: UI not available (running in non-interactive " +
		"mode). Ask the questions as plain chat text instead."

	maxQuestions    = 10
	minOptions      = 2
	maxOptions      = 4
	maxHeaderLength = 16
	maxLabelLength  = 60
)

var reservedOptionLabels = []string{"Other", "Type something."}

const questionnaireToolDescription = `Ask the user one or more structured questions during execution. Use when you need to:
1. Gather user preferences or requirements
2. Clarify ambiguous instructions
3. Get decisions on implementation choices as you work
4. Offer choices to the user about what direction to take

Usage notes:
- Up to ` + "10" + ` questions per call (2-4 options each). Group every clarifying question into one call instead of stacking several calls.
- Every question always offers a "Type something." custom-answer row; do NOT author "Other" or "Type something." labels yourself. An empty custom answer is a valid answer.
- The user can attach a free-form note to any answer, including a custom one.
- Question text and option descriptions are rendered as Markdown (bold, links, lists, tables, fenced mermaid diagrams); raw HTML is NOT rendered. Put option details and trade-offs in the question body itself, not in the option description.
- If you recommend an option, set ` + "`recommended: true`" + ` on it. The user sees it highlighted; you may also keep it first in the list.
- Set ` + "`multiSelect: true`" + ` when several answers are valid.`

var questionnairePromptGuidelines = []string{
	"Use ask_user_question whenever the user's request is underspecified and " +
		"you cannot proceed without concrete decisions — you can ask up to " +
		"10 questions per invocation.",
	"Each question MUST have 2-4 options. Every option requires a concise label " +
		"(1-5 words) and a description explaining what the choice means or its " +
		`trade-offs. Every question gets an automatically appended "Type ` +
		`something." custom-answer row, and an answer is always required — an ` +
		`empty custom answer counts. Do NOT author "Other" or "Type something." ` +
		"labels yourself.",
	"Set multiSelect: true when multiple answers are valid. Mark the option you " +
		"recommend with `recommended: true` so it is highlighted. Question text " +
		"and option descriptions are Markdown (tables and fenced mermaid diagrams " +
		"included); raw HTML is not rendered. Put option details and trade-offs " +
		"in the question body itself.",
	"Do not stack multiple ask_user_question calls back-to-back — group all " +
		"clarifying questions into one invocation.",
}

func questionnaireOptionSchema() *ai.Schema {
	label := ai.String(
		"MAX 60 CHARACTERS — hard limit, requests over the limit are rejected. " +
			"The display text for this option that the user will see and select. " +
			"Should be concise (1-5 words) and clearly describe the choice.",
	)
	label.MaxLength = intPtr(maxLabelLength)
	return ai.Object(
		ai.Prop("label", label),
		ai.Prop(
			"description",
			ai.String(
				"What this option means or what will happen if chosen. Useful for "+
					"providing context about trade-offs or implications.",
			),
		),
		ai.Opt(
			"recommended",
			ai.Boolean(
				"Set to true to mark this as the recommended answer. The UI "+
					"highlights it so the user can see the suggested choice at a glance.",
			).WithDefault(false),
		),
	)
}

func questionnaireQuestionSchema() *ai.Schema {
	header := ai.String(
		"MAX 16 CHARACTERS — hard limit, requests over the limit are rejected. " +
			`Very short chip/tag shown on the question tab. Examples: "Auth method", ` +
			`"Library", "Approach".`,
	)
	header.MaxLength = intPtr(maxHeaderLength)
	options := ai.ArrayOf(
		questionnaireOptionSchema(),
		`The available choices for this question. Must have 2-4 options. The `+
			`"Type something." custom-answer row is appended automatically — do NOT `+
			"author it.",
	)
	options.MinItems = intPtr(minOptions)
	options.MaxItems = intPtr(maxOptions)
	return ai.Object(
		ai.Prop(
			"question",
			ai.String(
				"The complete question to ask the user. Clear, specific, and ending "+
					"with a question mark. Rendered as Markdown (emphasis, links, "+
					"lists, tables, fenced mermaid diagrams); raw HTML is not "+
					"rendered. Put option details and trade-offs here, not in the "+
					"option description. If multiSelect is true, phrase it "+
					"accordingly.",
			),
		),
		ai.Prop("header", header),
		ai.Prop("options", options),
		ai.Opt(
			"multiSelect",
			ai.Boolean(
				"Set to true to let the user select several options. The user can "+
					"still add a note to the answer.",
			).WithDefault(false),
		),
	)
}

func intPtr(value int) *int { return &value }

type questionnaireQuestion struct {
	ID          string
	Question    string
	Header      string
	MultiSelect bool
	Options     []questionnaireOption
}

type questionnaireOption struct {
	Label       string
	Description string
	Recommended bool
}

// normalizeQuestionnaire canonicalizes the model's arguments: line terminators

func normalizeQuestionnaire(params map[string]any) []questionnaireQuestion {
	raw, _ := params["questions"].([]any)
	if len(raw) > maxQuestions {
		raw = raw[:maxQuestions]
	}
	out := make([]questionnaireQuestion, 0, len(raw))
	for index, entry := range raw {
		question, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		normalized := questionnaireQuestion{
			ID:       fmt.Sprintf("q%d", index+1),
			Question: normalizeNewlines(stringArg(question, "question")),
			Header:   stringArg(question, "header"),
		}
		multiSelect, _ := question["multiSelect"].(bool)
		normalized.MultiSelect = multiSelect
		options, _ := question["options"].([]any)
		for _, rawOption := range options {
			option, ok := rawOption.(map[string]any)
			if !ok {
				continue
			}
			recommended, _ := option["recommended"].(bool)
			normalized.Options = append(normalized.Options, questionnaireOption{
				Label:       stringArg(option, "label"),
				Description: stringArg(option, "description"),
				Recommended: recommended,
			})
		}
		out = append(out, normalized)
	}
	return out
}

func normalizeNewlines(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func validateQuestionnaire(questions []questionnaireQuestion) string {
	if len(questions) == 0 {
		return "Error: at least one question is required."
	}
	if len(questions) > maxQuestions {
		return "Error: at most " + strconv.Itoa(maxQuestions) + " questions per call."
	}
	for index, question := range questions {
		number := strconv.Itoa(index + 1)
		if strings.TrimSpace(question.Question) == "" {
			return "Error: question " + number + " is empty."
		}
		if len([]rune(question.Header)) > maxHeaderLength {
			return "Error: question " + number + " header exceeds " +
				strconv.Itoa(maxHeaderLength) + " characters."
		}
		if len(question.Options) < minOptions || len(question.Options) > maxOptions {
			return "Error: question " + number + " must have " + strconv.Itoa(minOptions) +
				"-" + strconv.Itoa(maxOptions) + " options."
		}
		seen := make(map[string]bool, len(question.Options))
		for _, option := range question.Options {
			label := strings.TrimSpace(option.Label)
			if label == "" {
				return "Error: question " + number + " has an empty option label."
			}
			if len([]rune(label)) > maxLabelLength {
				return "Error: question " + number + ` option "` + label + `" exceeds ` +
					strconv.Itoa(maxLabelLength) + " characters."
			}
			if containsString(reservedOptionLabels, label) {
				return "Error: question " + number + ` uses reserved option label "` +
					label + `".`
			}
			if seen[label] {
				return "Error: question " + number + ` repeats option label "` +
					label + `".`
			}
			seen[label] = true
		}
	}
	return ""
}

func containsString(list []string, value string) bool {
	for _, entry := range list {
		if entry == value {
			return true
		}
	}
	return false
}

type questionnaireAnswer struct {
	Index   int     `json:"index"`
	Options []int   `json:"options"`
	Custom  *string `json:"custom"`
	Notes   string  `json:"notes"`
}

type questionnaireAnsweredBy struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type questionnaireResponse struct {
	Answers    []questionnaireAnswer    `json:"answers"`
	Note       string                   `json:"note"`
	AnsweredBy *questionnaireAnsweredBy `json:"answeredBy"`
	Kind       string                   `json:"kind"`
	Message    string                   `json:"message"`
}

type answerKind string

const (
	answerKindOption answerKind = "option"
	answerKindCustom answerKind = "custom"
	answerKindMulti  answerKind = "multi"
)

type questionAnswer struct {
	QuestionID    string     `json:"questionId"`
	QuestionIndex int        `json:"questionIndex"`
	Question      string     `json:"question"`
	Kind          answerKind `json:"kind"`
	Answer        *string    `json:"answer"`
	Selected      []string   `json:"selected,omitempty"`
	Notes         string     `json:"notes,omitempty"`
}

func parseQuestionnaire(
	raw string,
	questions []questionnaireQuestion,
) (questionnaireResponse, []questionAnswer) {
	var payload questionnaireResponse
	_ = json.Unmarshal([]byte(raw), &payload)
	byIndex := make(map[int]questionnaireAnswer, len(payload.Answers))
	for _, answer := range payload.Answers {
		byIndex[answer.Index] = answer
	}
	answers := make([]questionAnswer, 0, len(questions))
	for index, question := range questions {
		answers = append(
			answers,
			answerFromWire(question, index, byIndex[index]),
		)
	}
	payload.Note = strings.TrimSpace(payload.Note)
	return payload, answers
}

func answerFromWire(
	question questionnaireQuestion,
	index int,
	wire questionnaireAnswer,
) questionAnswer {
	out := questionAnswer{
		QuestionID:    question.ID,
		QuestionIndex: index,
		Question:      question.Question,
		Notes:         wire.Notes,
	}
	if strings.TrimSpace(wire.Notes) == "" {
		out.Notes = ""
	}

	if wire.Custom != nil {
		out.Kind = answerKindCustom
		out.Answer = wire.Custom
		return out
	}
	indices := make([]int, 0, len(wire.Options))
	for _, optionIndex := range wire.Options {
		if optionIndex >= 0 && optionIndex < len(question.Options) {
			indices = append(indices, optionIndex)
		}
	}
	if question.MultiSelect {
		out.Kind = answerKindMulti
		out.Selected = make([]string, 0, len(indices))
		for _, optionIndex := range indices {
			out.Selected = append(out.Selected, question.Options[optionIndex].Label)
		}
		return out
	}
	out.Kind = answerKindOption
	chosen := ""
	if len(indices) > 0 {
		chosen = question.Options[indices[0]].Label
	}
	out.Answer = &chosen
	return out
}

func answeredByLabel(by *questionnaireAnsweredBy) string {
	if by == nil {
		return ""
	}
	if by.Name != "" && by.Username != "" {
		return by.Name + " (@" + by.Username + ")"
	}
	if by.Name != "" {
		return by.Name
	}
	if by.Username != "" {
		return "@" + by.Username
	}
	if by.ID != "" {
		return "actor " + by.ID
	}
	return ""
}

func buildQuestionnaireEnvelope(
	answers []questionAnswer,
	cancelled bool,
	finalNote string,
	answeredBy *questionnaireAnsweredBy,
) string {
	if cancelled {
		return "User declined to answer questions"
	}
	segments := make([]string, 0, len(answers))
	for _, answer := range answers {
		value := ""
		if answer.Kind == answerKindMulti {
			value = strings.Join(answer.Selected, ", ")
		} else if answer.Answer != nil {
			value = *answer.Answer
		}
		// The upstream envelope quotes literally, so a multi-line question keeps

		segment := `"` + answer.Question + `"="` + value + `"`
		if answer.Notes != "" {
			segment += ". user notes: " + answer.Notes
		}
		segments = append(segments, segment+".")
	}
	if len(segments) == 0 {
		return "User declined to answer questions"
	}
	note := strings.TrimSpace(finalNote)
	noteSegment := ""
	if note != "" {
		noteSegment = " User's final note: " + note
	}

	lead := "User has answered your questions: "
	if who := answeredByLabel(answeredBy); who != "" {
		lead = "User \"" + who + "\" has answered your questions: "
	}
	return lead + strings.Join(segments, " ") + noteSegment +
		" You can now continue with the user's answers in mind."
}

func questionnaireResult(
	value, requestID string,
	answers []questionAnswer,
	cancelled bool,
	unavailable bool,
	finalNote string,
	answeredBy *questionnaireAnsweredBy,
) agent.AgentToolResult {
	details := map[string]any{
		"answers":   answers,
		"cancelled": cancelled,
	}
	if requestID != "" {
		details["requestId"] = requestID
	}
	if unavailable {
		details["unavailable"] = true
	}
	if note := strings.TrimSpace(finalNote); note != "" {
		details["finalNote"] = note
	}
	if answeredBy != nil {
		details["answeredBy"] = answeredBy
	}
	result := text(value)
	result.Details = details
	return result
}

func (s *Session) askUserQuestion() agent.AgentTool {
	return agent.AgentTool{
		Name:             "ask_user_question",
		Label:            "Ask User Question",
		Description:      questionnaireToolDescription,
		PromptGuidelines: questionnairePromptGuidelines,
		Parameters: ai.Object(ai.Prop(
			"questions",
			func() *ai.Schema {
				schema := ai.ArrayOf(
					questionnaireQuestionSchema(),
					"Questions to ask (1-10). All questions are shown as tabs and "+
						"submitted together.",
				)
				schema.MinItems = intPtr(1)
				schema.MaxItems = intPtr(maxQuestions)
				return schema
			}(),
		)),
		Execute: func(ctx context.Context, _ string, params map[string]any, _ agent.ToolUpdateFunc) (agent.AgentToolResult, error) {
			questions := normalizeQuestionnaire(params)
			if invalid := validateQuestionnaire(questions); invalid != "" {
				return questionnaireResult(invalid, "", nil, true, false, "", nil), nil
			}
			prefill, err := json.Marshal(map[string]any{
				"kind":      questionnaireDialogKind,
				"questions": questionnaireWire(questions),
			})
			if err != nil {
				return errorText(err.Error()), nil
			}
			title := fmt.Sprintf(
				"CherryPi questionnaire (%d %s)",
				len(questions),
				plural(len(questions), "question", "questions"),
			)
			raw, requestID, cancelled := s.askDialog(ctx, title, string(prefill))
			if cancelled {
				return questionnaireResult(
					buildQuestionnaireEnvelope(nil, true, "", nil),
					requestID,
					nil,
					true,
					false,
					"",
					nil,
				), nil
			}
			payload, answers := parseQuestionnaire(raw, questions)
			if payload.Kind == questionnaireUnavailableKind {
				reason := payload.Message
				if reason == "" {
					reason = questionnaireNonInteractiveReason
				}
				return questionnaireResult(
					"Error: "+reason,
					requestID,
					nil,
					false,
					true,
					"",
					nil,
				), nil
			}
			return questionnaireResult(
				buildQuestionnaireEnvelope(
					answers,
					false,
					payload.Note,
					payload.AnsweredBy,
				),
				requestID,
				answers,
				false,
				false,
				payload.Note,
				payload.AnsweredBy,
			), nil
		},
	}
}

func questionnaireWire(questions []questionnaireQuestion) []map[string]any {
	out := make([]map[string]any, 0, len(questions))
	for _, question := range questions {
		options := make([]map[string]any, 0, len(question.Options))
		for _, option := range question.Options {
			options = append(options, map[string]any{
				"label":       option.Label,
				"description": option.Description,
				"recommended": option.Recommended,
			})
		}
		out = append(out, map[string]any{
			"id":          question.ID,
			"question":    question.Question,
			"header":      question.Header,
			"options":     options,
			"multiSelect": question.MultiSelect,
		})
	}
	return out
}

func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return many
}
