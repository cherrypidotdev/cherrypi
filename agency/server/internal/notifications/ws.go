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
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// writeTimeout bounds one WebSocket frame write so a wedged client cannot pin a
// handler goroutine forever.
const writeTimeout = 10 * time.Second

// Event is one message pushed to a browser over the realtime WebSocket. Its
// JSON shape is the wire contract the SPA consumes; keep it in sync with the
// SDK's `AttentionEvent`.
type Event struct {
	// Type is "hello" on connect or "notification" for a new inbox entry.
	Type string `json:"type"`
	// Unread is the current unread count, set on the initial "hello".
	Unread int64 `json:"unread,omitempty"`
	// Notification carries the entry for a "notification" event.
	Notification *Entry `json:"notification,omitempty"`
}

// Entry is a notification as the browser sees it.
type Entry struct {
	ID        int64       `json:"id"`
	AgentID   ids.AgentID `json:"agentId"`
	TaskID    string      `json:"taskId"`
	Kind      string      `json:"kind"`
	Severity  string      `json:"severity"`
	Title     string      `json:"title"`
	Body      string      `json:"body"`
	CreatedAt string      `json:"createdAt"`
	ReadAt    *time.Time  `json:"readAt,omitempty"`
}

// WSHandler upgrades to a WebSocket and pushes each new notification as it is
// created. It is mounted behind the agency's basic-auth middleware, so only an
// authenticated browser can subscribe. The client need not send anything; the
// first frame is a "hello" carrying the current unread count.
func (s *Service) WSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// Behind basic auth; accept any origin so the Vite dev server (a
			// different origin) can attach too.
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// A targeted entry reaches only its recipient; the authenticated actor
		// is the stream's address.
		actorID, _ := auth.ActorID(r.Context())
		events, unsubscribe := s.hub.subscribe(actorID)
		defer unsubscribe()

		// A read loop notices the client closing the tab and lets the library
		// answer keepalive pings. Incoming application frames are ignored.
		go func() {
			for {
				if _, _, err := conn.Read(ctx); err != nil {
					cancel()
					return
				}
			}
		}()

		// A stale unread count is not worth refusing the stream over; the client
		// refreshes the inbox on connect regardless.
		unread, _ := s.UnreadCount(ctx, actorID)
		if err := writeEvent(ctx, conn, Event{Type: "hello", Unread: unread}); err != nil {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case n, ok := <-events:
				if !ok {
					return
				}
				event := Event{Type: "notification", Notification: browserEntry(n)}
				if err := writeEvent(ctx, conn, event); err != nil {
					return
				}
			}
		}
	})
}

func writeEvent(ctx context.Context, conn *websocket.Conn, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, data)
}

func browserEntry(n Notification) *Entry {
	entry := &Entry{
		ID:        n.ID,
		AgentID:   n.AgentID,
		TaskID:    n.TaskID,
		Kind:      n.Kind,
		Severity:  n.Severity,
		Title:     n.Title,
		Body:      n.Body,
		CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if !n.ReadAt.IsZero() {
		readAt := n.ReadAt.UTC()
		entry.ReadAt = &readAt
	}
	return entry
}
