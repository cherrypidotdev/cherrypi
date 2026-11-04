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

package events

import (
	"sync"
	"time"
)

type Message struct {
	Seq     uint64         `json:"seq"`
	Kind    string         `json:"kind"`
	At      time.Time      `json:"at"`
	Payload map[string]any `json:"payload"`
}

const LogKind = "log"

const (
	backlogLimit = 256

	logReplayLimit = 2048
)

type Hub struct {
	mu   sync.Mutex
	seq  uint64
	subs map[chan Message]struct{}

	backlog []Message

	logs []Message
}

func New() *Hub {
	return &Hub{subs: map[chan Message]struct{}{}}
}

func (h *Hub) Publish(kind string, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.seq++
	msg := Message{Seq: h.seq, Kind: kind, At: time.Now(), Payload: payload}

	if kind == LogKind {
		h.logs = append(h.logs, msg)
		if len(h.logs) > logReplayLimit {
			h.logs = h.logs[len(h.logs)-logReplayLimit:]
		}
	}

	if len(h.subs) == 0 {
		h.backlog = append(h.backlog, msg)
		if len(h.backlog) > backlogLimit {
			h.backlog = h.backlog[len(h.backlog)-backlogLimit:]
		}
		return
	}
	for ch := range h.subs {
		select {
		case ch <- msg:
		default:

		}
	}
}

func (h *Hub) Subscribe(since uint64) (<-chan Message, func()) {
	ch := make(chan Message, backlogLimit+logReplayLimit+64)

	h.mu.Lock()
	last := since
	if since == 0 {
		for _, msg := range h.backlog {
			select {
			case ch <- msg:
			default:
			}
			if msg.Seq > last {
				last = msg.Seq
			}
		}
		h.backlog = nil
	} else {
		for _, msg := range h.logs {
			if msg.Seq <= last {
				continue
			}
			select {
			case ch <- msg:
			default:
			}
			last = msg.Seq
		}
	}
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
