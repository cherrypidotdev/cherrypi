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

package realtime

import (
	"sync"

	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
)

const subscriberBuffer = 16

type Bus struct {
	mu     sync.Mutex
	subs   map[int]chan agencyv1.RealtimeKind
	nextID int
}

func New() *Bus {
	return &Bus{subs: map[int]chan agencyv1.RealtimeKind{}}
}

func (b *Bus) Subscribe() (<-chan agencyv1.RealtimeKind, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++
	ch := make(chan agencyv1.RealtimeKind, subscriberBuffer)
	b.subs[id] = ch

	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if existing, ok := b.subs[id]; ok {
			delete(b.subs, id)
			close(existing)
		}
	}
}

func (b *Bus) Publish(kind agencyv1.RealtimeKind) {
	if b == nil || kind == agencyv1.RealtimeKind_REALTIME_KIND_UNSPECIFIED {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.subs {
		select {
		case ch <- kind:
		default:
		}
	}
}
