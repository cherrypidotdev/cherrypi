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

// Package events consumes an agent's event stream over its tunnel and turns
// the messages into the agency's usage aggregate, notifications, and webhook
// deliveries.
package events

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"cherrypi.vladfaust.com/agency/internal/agentlogs"
	"cherrypi.vladfaust.com/agency/internal/artifacts"
	"cherrypi.vladfaust.com/agency/internal/notifications"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	"cherrypi.vladfaust.com/agency/internal/webhook"
)

// Message mirrors the agent's event hub payload.
type Message struct {
	Kind    string         `json:"kind"`
	At      time.Time      `json:"at"`
	Payload map[string]any `json:"payload"`
}

// Consumer handles one agent's events.
type Consumer struct {
	Store         store.Store
	Notifier      *webhook.Notifier
	Notifications *notifications.Service
	// Tasks, when set, is told when a run fails so the owning task moves to
	// failed and frees its concurrency slot.
	Tasks *tasks.Service
	// Logs, when set, persists the agent's operational log records.
	Logs *agentlogs.Service
	// OnTaskFailed, when set, is called after a task is marked failed (the
	// scheduler uses it to reconsider the queue).
	OnTaskFailed func()
	// Artifacts stores the outputs an agent publishes for its task.
	Artifacts *artifacts.Service
}

// Run consumes the agent's event stream until ctx is cancelled, reconnecting
// on failure.
func (c *Consumer) Run(ctx context.Context, conn *tunnel.Connection) {
	for ctx.Err() == nil {
		if err := c.consume(ctx, conn); err != nil && ctx.Err() == nil {
			log.Printf("[events %d] %v (retrying)", conn.ID, err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (c *Consumer) consume(ctx context.Context, conn *tunnel.Connection) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://agent/api/rest/events", nil)
	if err != nil {
		return err
	}

	resp, err := conn.HTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("events stream: status %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	for {
		var msg Message
		if err := dec.Decode(&msg); err != nil {
			return err
		}
		c.dispatch(ctx, conn.ID, msg)
	}
}

func (c *Consumer) dispatch(ctx context.Context, agentID ids.AgentID, msg Message) {
	sessionID := stringValue(msg.Payload["session_id"])

	switch msg.Kind {
	case "log":
		c.insertLog(ctx, agentID, msg)

	case "usage":
		_ = c.Store.InsertAgentUsage(ctx, store.InsertAgentUsageParams{
			AgentID:      sql.NullInt64{Int64: int64(agentID), Valid: true},
			SessionID:    sessionID,
			Model:        stringValue(msg.Payload["model"]),
			InputTokens:  int64Value(msg.Payload["input_tokens"]),
			OutputTokens: int64Value(msg.Payload["output_tokens"]),
			Cost:         floatValue(msg.Payload["cost"]),
			ActiveMs:     int64Value(msg.Payload["active_ms"]),
			At:           int64Value(msg.Payload["at"]),
		})

	case "user_input_requested":
		c.notifyTask(
			ctx,
			agentID,
			c.taskID(ctx, agentID, sessionID),
			"input_requested",
			"warning",
			"Input requested",
			stringValue(msg.Payload["title"]),
		)
		c.Notifier.Notify(msg.Kind, agentID, msg.Payload)

	case "task_link":
		if c.Tasks != nil {
			if err := c.Tasks.SetLinksBySession(
				tasks.WithActor(ctx, tasks.ActorAgent),
				agentID,
				sessionID,
				linksValue(msg.Payload["links"]),
			); err != nil {
				log.Printf("[events %d] set task links %s: %v", agentID, sessionID, err)
			}
		}

	case "task_artifact":
		// The agent published task outputs (A2A artifacts); store them on the
		// task that owns its Pi session.
		if c.Artifacts != nil {
			taskID := c.taskID(ctx, agentID, sessionID)
			if taskID != "" {
				if err := c.Artifacts.Apply(
					ctx,
					taskID,
					replaceValue(msg.Payload["replace"]),
					artifactsValue(msg.Payload["artifacts"]),
				); err != nil {
					log.Printf("[events %d] set task artifacts %s: %v", agentID, sessionID, err)
				}
			}
		}

	case "agent_state_changed":
		if stringValue(msg.Payload["state"]) == "failed" {
			errMsg := stringValue(msg.Payload["error"])
			c.notifyTask(
				ctx,
				agentID,
				c.taskID(ctx, agentID, sessionID),
				"failed",
				"error",
				"Task failed",
				errMsg,
			)
			if c.Tasks != nil {
				if err := c.Tasks.FailBySession(ctx, agentID, sessionID, errMsg); err != nil {
					log.Printf("[events %d] fail task %s: %v", agentID, sessionID, err)
				} else if c.OnTaskFailed != nil {
					c.OnTaskFailed()
				}
			}
		}
		c.Notifier.Notify(msg.Kind, agentID, msg.Payload)
	}
}

// notifyTask records a task notification for the task's recipients (its leads
// plus its subscribers). Without a resolvable task, or with no recipients, it
// falls back to a broadcast entry so the signal is not lost.
func (c *Consumer) notifyTask(
	ctx context.Context,
	agentID ids.AgentID,
	taskID, kind, severity, title, body string,
) {
	var recipients []ids.ActorID
	if c.Tasks != nil && taskID != "" {
		recipients, _ = c.Tasks.Recipients(ctx, taskID)
	}
	_ = c.Notifications.CreateForTask(ctx, taskID, recipients, notifications.Notification{
		AgentID:  agentID,
		Kind:     kind,
		Severity: severity,
		Title:    title,
		Body:     body,
	})
}

// taskID resolves an agent's Pi session id to the agency task that owns it,
// returning "" when unknown so the inbox row renders unlinked.
func (c *Consumer) taskID(ctx context.Context, agentID ids.AgentID, sessionID string) string {
	if c.Tasks == nil || sessionID == "" {
		return ""
	}
	id, err := c.Tasks.TaskIDBySession(ctx, agentID, sessionID)
	if err != nil {
		return ""
	}
	return id
}

// insertLog persists one agent log record. The known fields become columns;
// every other payload key is kept as structured attributes.
func (c *Consumer) insertLog(ctx context.Context, agentID ids.AgentID, msg Message) {
	if c.Logs == nil {
		return
	}

	attrs := map[string]any{}
	for key, value := range msg.Payload {
		switch key {
		case "level", "message", "task_id", "session_id", "at":
		default:
			attrs[key] = value
		}
	}

	if err := c.Logs.Insert(ctx, agentlogs.Log{
		AgentID:   agentID,
		At:        msg.At.UnixMilli(),
		Level:     agentlogs.ParseLevel(stringValue(msg.Payload["level"])),
		Message:   stringValue(msg.Payload["message"]),
		TaskID:    stringValue(msg.Payload["task_id"]),
		SessionID: stringValue(msg.Payload["session_id"]),
		Attrs:     attrs,
	}); err != nil {
		log.Printf("[events %d] insert log: %v", agentID, err)
	}
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

// linksValue decodes the `links` array of a task_link event payload. The
// agent sends the task's complete URL list each time.
func linksValue(v any) []tasks.Link {
	items, _ := v.([]any)
	out := make([]tasks.Link, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, tasks.Link{
			URL:   stringValue(m["url"]),
			Label: stringValue(m["label"]),
		})
	}
	return out
}

// replaceValue reports whether an update is a full replacement; absent means
// true, matching the replacement tool's default.
func replaceValue(v any) bool {
	b, ok := v.(bool)
	if !ok {
		return true
	}
	return b
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}

// artifactsValue decodes the `artifacts` array of a task_artifact payload.
func artifactsValue(v any) []artifacts.Update {
	items, _ := v.([]any)
	out := make([]artifacts.Update, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		lastChunk := true
		if raw, ok := m["last_chunk"]; ok {
			lastChunk = boolValue(raw)
		}
		out = append(out, artifacts.Update{
			ID:           stringValue(m["id"]),
			Name:         stringValue(m["name"]),
			Description:  stringValue(m["description"]),
			MetadataJSON: stringValue(m["metadata_json"]),
			Append:       boolValue(m["append"]),
			LastChunk:    lastChunk,
			Parts:        artifactPartsValue(m["parts"]),
		})
	}
	return out
}

func artifactPartsValue(v any) []artifacts.Part {
	items, _ := v.([]any)
	out := make([]artifacts.Part, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, artifacts.Part{
			Kind:         stringValue(m["kind"]),
			Text:         stringValue(m["text"]),
			DataJSON:     stringValue(m["data"]),
			URL:          stringValue(m["url"]),
			AttachmentID: int64Value(m["attachment_id"]),
			Filename:     stringValue(m["filename"]),
			MediaType:    stringValue(m["media_type"]),
			MetadataJSON: stringValue(m["metadata_json"]),
		})
	}
	return out
}

func floatValue(v any) float64 {
	f, _ := v.(float64)
	return f
}

func int64Value(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	default:
		return 0
	}
}
