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

import "time"

// Status mirrors agent.v1.SessionStatus. Values 1–4 are persisted in the
// sessions table (lifecycle); 5–6 are derived client-facing refinements of an
// active session and are never persisted. StatusCancelled is legacy: no longer
// set, and reported to clients as StatusDone.
type Status int64

const (
	StatusActive    Status = 1
	StatusDone      Status = 2
	StatusFailed    Status = 3
	StatusCancelled Status = 4

	// Refinements of StatusActive, computed from the session's live flags.
	StatusInProgress   Status = 5
	StatusWaitingInput Status = 6
)

// Terminal reports whether a status ends the session's life.
func Terminal(status Status) bool {
	return status == StatusDone || status == StatusFailed || status == StatusCancelled
}

// ClientStatus refines a persisted status into the value clients see: terminal
// statuses pass through (cancelled is merged into done), while an active
// session becomes waiting-input, in-progress, or done (no run in flight).
func ClientStatus(status Status, pendingPrompt, waiting bool) Status {
	if status == StatusCancelled {
		return StatusDone
	}
	if Terminal(status) {
		return status
	}
	switch {
	case waiting:
		return StatusWaitingInput
	case pendingPrompt:
		return StatusInProgress
	default:
		return StatusDone
	}
}

// ToolPhase mirrors agent.v1.ToolPhase.
type ToolPhase int64

const (
	ToolPhaseUnspecified ToolPhase = 0
	ToolStarted          ToolPhase = 1
	ToolUpdated          ToolPhase = 2
	ToolFinished         ToolPhase = 3
)

// EventKind discriminates the normalized events streamed to watchers.
type EventKind int

const (
	EventStatus EventKind = iota
	EventRunStarted
	EventRunSettled
	EventText
	EventThinking
	EventTool
	EventLog
	EventExtensionUI
	EventUser
	EventQueue
)

// Event is the normalized view of one Pi RPC record (or a lifecycle change).
type Event struct {
	Kind      EventKind
	At        time.Time
	Status    Status
	Error     string
	Delta     string
	Text      string
	Tool      *ToolEvent
	Line      string
	Extension *ExtensionUIEvent
	Queue     *QueueEvent
}

// QueueEvent mirrors Pi's queue_update: prompts accepted but not injected yet.
type QueueEvent struct {
	Steering []string
	FollowUp []string
}

// ExtensionUIEvent is a forwarded Pi extension-UI request. Dialog methods
// block the run until the client answers; the rest are fire-and-forget.
type ExtensionUIEvent struct {
	RequestID   string
	Method      string
	Title       string
	Message     string
	Placeholder string
	Prefill     string
	Options     []string
	Timeout     uint32
	RawJSON     string
}

type ToolEvent struct {
	CallID     string
	Name       string
	Phase      ToolPhase
	ArgsJSON   string
	ResultJSON string
	// DetailsJSON is the tool result's structured `details`, when it has one.
	// It survives history replay so the console can render a tool's structured
	// output (the questionnaire's answers).
	DetailsJSON string
	IsError     bool
}

func statusEvent(status Status, err string) Event {
	return Event{Kind: EventStatus, At: time.Now(), Status: status, Error: err}
}
