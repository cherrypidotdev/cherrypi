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

// Realtime fan-out for the notifications inbox. The agency server is the only
// writer (agents push attention signals over the tunnel), so a small in-process
// hub is enough to relay new entries to connected browser clients; there is no
// broker to keep in sync.
package notifications

import (
	"sync"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

// subscriberBuffer bounds how many notifications may queue for one slow
// browser before older ones are dropped. The inbox is the source of truth, so
// dropping a realtime hint only costs a refresh.
const subscriberBuffer = 16

// subscriber is one connected browser, keyed by the actor it authenticated as.
// A targeted entry is delivered only to that actor; a broadcast entry reaches
// every subscriber.
type subscriber struct {
	actorID ids.ActorID
	ch      chan Notification
}

// hub is a fan-out of notifications to subscribed WebSocket clients.
type hub struct {
	mu     sync.Mutex
	subs   map[int]subscriber
	nextID int
}

func newHub() *hub {
	return &hub{subs: map[int]subscriber{}}
}

// subscribe returns a channel of new notifications for actorID plus a cancel
// function. The channel is closed by cancel and never by publish.
func (h *hub) subscribe(actorID ids.ActorID) (<-chan Notification, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.nextID
	h.nextID++
	ch := make(chan Notification, subscriberBuffer)
	h.subs[id] = subscriber{actorID: actorID, ch: ch}

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if existing, ok := h.subs[id]; ok {
			delete(h.subs, id)
			close(existing.ch)
		}
	}
}

// publish delivers n to the matching subscribers, dropping it for any that is
// full. A targeted entry goes only to its recipient; a broadcast entry (zero
// recipient) goes to everyone. It never blocks.
func (h *hub) publish(n Notification) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, sub := range h.subs {
		if n.RecipientActorID != 0 && sub.actorID != n.RecipientActorID {
			continue
		}
		select {
		case sub.ch <- n:
		default:
		}
	}
}
