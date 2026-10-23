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

// PromptSender is who a prompt is attributed to. The agency resolves it from
// the authenticated actor and passes it with every prompt; the agent renders
// it as the envelope's `sendingActor`.
type PromptSender struct {
	ActorID  int64
	Username string
	Name     string
	// Kind is "user", "agent", or "system".
	Kind string
}

// PromptAttachment is one file the agent wrote into the workspace for a prompt.
// The envelope carries the id, the final filename, and the on-disk path so the
// model can open it; the console reads the id and filename off the parsed
// UserMessage.
type PromptAttachment struct {
	ID       int64
	Filename string
	Path     string
}

// promptEnvelope is the message shape Pi receives: the sender plus the plain
// text and the files it carried. It mirrors the `{sendingActor, messageBody}`
// contract in docs/auth.md.
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

// envelopeAttachment is an attachment as it appears in the envelope Pi sees.
// `path` is the on-disk location the model opens; a legacy-style trailer is no
// longer appended to `text`.
type envelopeAttachment struct {
	ID       int64  `json:"id"`
	Filename string `json:"filename"`
	Path     string `json:"path"`
}

// ParsedPrompt is a decoded envelope: the plain body, the sender, and the files
// the prompt carried.
type ParsedPrompt struct {
	Body        string
	Sender      *PromptSender
	Attachments []PromptAttachment
}

// BuildPromptEnvelope renders the actor-tagged envelope handed to Pi. A nil or
// empty sender becomes a system placeholder, so a legacy agency that sends no
// sender still produces a well-formed message.
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

// ParsePromptEnvelope decodes the envelope from a user message. A legacy plain
// prompt that is not an envelope (or arbitrary JSON) reports false, so a caller
// falls back to the raw text and its attachment trailer.
//
// Pi may append a note after the envelope (for example the image-size
// annotation on a prompt that carried a picture), so only the first JSON value
// is decoded and the trailing text is ignored.
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
