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

// Package webhook delivers fire-and-forget JSON notifications to the agency's
// configured URLs. Webhooks are agency-level and may be filtered to one agent
// (NULL agent_id = all agents). Delivery is best-effort; retries and HMAC
// signing are deferred.
package webhook

import (
	"bytes"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
)

// Supported events.
const (
	// EventUserInputRequested fires when an extension dialog blocks a session.
	EventUserInputRequested = "user_input_requested"
	// EventAgentStateChanged fires on session state transitions.
	EventAgentStateChanged = "agent_state_changed"
)

// AllowedEvents is the set of events a webhook may subscribe to.
var AllowedEvents = []string{EventUserInputRequested, EventAgentStateChanged}

// ValidEvent reports whether name is a supported event.
func ValidEvent(name string) bool {
	for _, e := range AllowedEvents {
		if e == name {
			return true
		}
	}
	return false
}

// Notifier posts events to the matching webhooks.
type Notifier struct {
	store  store.Store
	client *http.Client
}

// NewNotifier builds a notifier over the webhook store.
func NewNotifier(st store.Store) *Notifier {
	return &Notifier{store: st, client: &http.Client{Timeout: 10 * time.Second}}
}

// Notify delivers `event` from `agentID` with `payload` to every subscribed
// webhook, fire-and-forget. It never blocks the caller.
func (n *Notifier) Notify(event string, agentID ids.AgentID, payload map[string]any) {
	body := map[string]any{"event": event, "actor_id": agentID}
	for key, value := range payload {
		body[key] = value
	}

	data, err := json.Marshal(body)
	if err != nil {
		log.Printf("[webhook] marshal %s: %v", event, err)
		return
	}

	go n.deliver(event, agentID, data)
}

// SamplePayload returns the sample body for an event's test payload.
func SamplePayload(event string) (map[string]any, bool) {
	switch event {
	case EventUserInputRequested:
		return map[string]any{
			"request_id": "test-request",
			"method":     "confirm",
			"title":      "CherryPi Agency test",
			"message":    "This is a test dialog from CherryPi Agency.",
			"options":    []string{"yes", "no"},
		}, true
	case EventAgentStateChanged:
		return map[string]any{"state": "run_settled", "status": 2}, true
	default:
		return nil, false
	}
}

// Send posts one event payload synchronously and returns the target's HTTP
// status code. A transport failure returns status 0 with the error.
func (n *Notifier) Send(
	ctx context.Context,
	url, secret, event string,
	payload map[string]any,
) (int, error) {
	body := map[string]any{"event": event}
	for key, value := range payload {
		body[key] = value
	}

	data, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	return n.post(ctx, url, secret, data)
}

func (n *Notifier) deliver(event string, agentID ids.AgentID, data []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	hooks, err := n.store.ListWebhooks(ctx)
	if err != nil {
		log.Printf("[webhook] list: %v", err)
		return
	}

	for _, hook := range hooks {
		if hook.AgentID.Valid && hook.AgentID.Int64 != int64(agentID) {
			continue
		}
		if !subscribed(hook.Events, event) {
			continue
		}
		_, _ = n.post(ctx, hook.Url, hook.Secret, data)
	}
}

func (n *Notifier) post(ctx context.Context, url, secret string, data []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		log.Printf("[webhook] request %s: %v", url, err)
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-CherryPi-Secret", secret)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		log.Printf("[webhook] post %s: %v", url, err)
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		log.Printf("[webhook] post %s: status %d", url, resp.StatusCode)
	}
	return resp.StatusCode, nil
}

// subscribed reports whether a comma-separated event list includes event.
func subscribed(events, event string) bool {
	for _, e := range strings.Split(events, ",") {
		if strings.TrimSpace(e) == event {
			return true
		}
	}
	return false
}
