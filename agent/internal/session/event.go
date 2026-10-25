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

type Status int64

const (
	StatusActive    Status = 1
	StatusDone      Status = 2
	StatusFailed    Status = 3
	StatusCancelled Status = 4

	StatusInProgress   Status = 5
	StatusWaitingInput Status = 6
)

func Terminal(status Status) bool {
	return status == StatusDone || status == StatusFailed || status == StatusCancelled
}

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

type ToolPhase int64

const (
	ToolPhaseUnspecified ToolPhase = 0
	ToolStarted          ToolPhase = 1
	ToolUpdated          ToolPhase = 2
	ToolFinished         ToolPhase = 3
)

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

type QueueEvent struct {
	Steering []string
	FollowUp []string
}

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

	DetailsJSON string
	IsError     bool
}

func statusEvent(status Status, err string) Event {
	return Event{Kind: EventStatus, At: time.Now(), Status: status, Error: err}
}
