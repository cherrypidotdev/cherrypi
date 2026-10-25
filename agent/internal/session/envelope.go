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
	"encoding/json"
	"strings"
)

type PromptSender struct {
	ActorID  int64
	Username string
	Name     string

	Kind string
}

type PromptAttachment struct {
	ID       int64
	Filename string
	Path     string
}

type promptEnvelope struct {
	SendingActor envelopeSender `json:"sendingActor"`
	MessageBody  envelopeBody   `json:"messageBody"`
}

type envelopeSender struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Kind     string `json:"kind,omitempty"`
}

type envelopeBody struct {
	Text        string               `json:"text"`
	Attachments []envelopeAttachment `json:"attachments,omitempty"`
}

type envelopeAttachment struct {
	ID       int64  `json:"id"`
	Filename string `json:"filename"`
	Path     string `json:"path"`
}

type ParsedPrompt struct {
	Body        string
	Sender      *PromptSender
	Attachments []PromptAttachment
}

func BuildPromptEnvelope(
	text string,
	attachments []PromptAttachment,
	sender *PromptSender,
) string {
	actor := envelopeSender{Username: "system", Name: "CherryPi", Kind: "system"}
	if sender != nil && (sender.Username != "" || sender.ActorID != 0 || sender.Kind != "") {
		actor = envelopeSender{
			ID:       sender.ActorID,
			Username: sender.Username,
			Name:     sender.Name,
			Kind:     sender.Kind,
		}
		if actor.Kind == "" {
			actor.Kind = "user"
		}
	}
	body := envelopeBody{Text: text}
	for _, attachment := range attachments {
		body.Attachments = append(body.Attachments, envelopeAttachment(attachment))
	}
	raw, err := json.Marshal(promptEnvelope{SendingActor: actor, MessageBody: body})
	if err != nil {
		return text
	}
	return string(raw)
}

func ParsePromptEnvelope(text string) (ParsedPrompt, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{") {
		return ParsedPrompt{}, false
	}
	var env promptEnvelope
	if err := json.NewDecoder(strings.NewReader(trimmed)).Decode(&env); err != nil {
		return ParsedPrompt{}, false
	}
	if env.SendingActor.Username == "" && env.SendingActor.ID == 0 &&
		env.SendingActor.Name == "" {
		return ParsedPrompt{}, false
	}
	parsed := ParsedPrompt{
		Body: env.MessageBody.Text,
		Sender: &PromptSender{
			ActorID:  env.SendingActor.ID,
			Username: env.SendingActor.Username,
			Name:     env.SendingActor.Name,
			Kind:     env.SendingActor.Kind,
		},
	}
	for _, attachment := range env.MessageBody.Attachments {
		parsed.Attachments = append(parsed.Attachments, PromptAttachment(attachment))
	}
	return parsed, true
}
