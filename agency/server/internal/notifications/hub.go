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

package notifications

import (
	"sync"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

const subscriberBuffer = 16

type subscriber struct {
	actorID ids.ActorID
	ch      chan Notification
}

type hub struct {
	mu     sync.Mutex
	subs   map[int]subscriber
	nextID int
}

func newHub() *hub {
	return &hub{subs: map[int]subscriber{}}
}

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
