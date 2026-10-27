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

const writeTimeout = 10 * time.Second

type Event struct {
	Type string `json:"type"`

	Unread int64 `json:"unread,omitempty"`

	Notification *Entry `json:"notification,omitempty"`
}

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

func (s *Service) WSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// Behind the auth middleware; accept any origin so the Vite dev server (a

			OriginPatterns: []string{"*"},
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		actorID, _ := auth.ActorID(r.Context())
		events, unsubscribe := s.hub.subscribe(actorID)
		defer unsubscribe()

		go func() {
			for {
				if _, _, err := conn.Read(ctx); err != nil {
					cancel()
					return
				}
			}
		}()

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
