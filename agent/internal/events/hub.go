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

// Package events is the agent's in-process event hub. Session webhook-style
// events (input requested, state changed) and usage deltas are published here
// and streamed to the agency over the tunnel, which turns them into
// notifications, webhooks, and usage aggregates.
package events

import (
	"sync"
	"time"
)

// Message is one published event. Payload carries the event-specific fields;
// `session_id` is always present for session-derived events.
type Message struct {
	Kind    string         `json:"kind"`
	At      time.Time      `json:"at"`
	Payload map[string]any `json:"payload"`
}

// Hub is a fan-out of messages to subscribers. Subscribers that cannot keep up
// drop messages rather than blocking publishers.
type Hub struct {
	mu   sync.Mutex
	subs map[chan Message]struct{}
}

// New builds an empty hub.
func New() *Hub {
	return &Hub{subs: map[chan Message]struct{}{}}
}

// Publish sends a message to every subscriber.
func (h *Hub) Publish(kind string, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}

	msg := Message{Kind: kind, At: time.Now(), Payload: payload}

	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- msg:
		default:
			// Slow subscriber: drop this event.
		}
	}
}

// Subscribe returns a channel of messages and an unsubscribe function.
func (h *Hub) Subscribe() (<-chan Message, func()) {
	ch := make(chan Message, 64)

	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		if _, ok := h.subs[ch]; ok {
			delete(h.subs, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
}
