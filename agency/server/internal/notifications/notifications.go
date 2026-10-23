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

// Package notifications is the agency's persistent inbox. Attention signals
// pushed by agents and user-directed invitations become notifications that
// survive long absences even with no webhooks configured.
package notifications

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
)

// keyInboxDisabledKinds is the settings row holding the comma-separated kinds
// the inbox should not store.
const keyInboxDisabledKinds = "notification_inbox_disabled_kinds"

// Notification is one inbox entry.
type Notification struct {
	ID      int64
	AgentID ids.AgentID
	TaskID  string
	// RecipientActorID targets the entry at one actor. Zero broadcasts it to
	// every reader (legacy entries and org-wide signals).
	RecipientActorID ids.ActorID
	// ProjectID is the project of the entry's task, when it has one.
	ProjectID string
	Kind      string
	Severity  string
	Title     string
	Body      string
	CreatedAt time.Time
	ReadAt    time.Time
}

// Service reads and writes the notifications table and fans new entries out to
// connected browser clients through its hub.
type Service struct {
	store store.Store
	hub   *hub
}

// New builds the service.
func New(st store.Store) *Service { return &Service{store: st, hub: newHub()} }

// Create records a notification and pushes it to every subscribed browser.
//
// A kind the inbox preferences disable is not stored: it is only pushed, so
// the live sound/desktop alerts still fire without leaving an inbox row. The
// push carries a zero id because there is no row to reference.
//
// A repeated `input_requested` signal for a task that already has an unread
// one for the same recipient is folded into that entry instead of adding a
// row: a questionnaire asks one question per dialog, but the user only needs
// one inbox entry per wait. The fold does not push, so it does not alert again
// either.
func (s *Service) Create(ctx context.Context, n Notification) error {
	createdAt := time.Now()
	if s.skipInbox(ctx, n.Kind) {
		n.CreatedAt = createdAt
		s.hub.publish(n)
		return nil
	}
	if n.Kind == "input_requested" && n.AgentID != 0 && n.TaskID != "" {
		row, err := s.store.RefreshUnreadNotification(ctx, store.RefreshUnreadNotificationParams{
			Title:            n.Title,
			Body:             n.Body,
			CreatedAt:        createdAt.UnixMilli(),
			AgentID:          sql.NullInt64{Int64: int64(n.AgentID), Valid: true},
			TaskID:           n.TaskID,
			Kind:             n.Kind,
			RecipientActorID: nullActorID(n.RecipientActorID),
		})
		switch {
		case err == nil:
			n.ID = row.ID
			n.CreatedAt = createdAt
			return nil
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}
	}

	row, err := s.store.CreateNotification(ctx, store.CreateNotificationParams{
		AgentID:          sql.NullInt64{Int64: int64(n.AgentID), Valid: n.AgentID != 0},
		TaskID:           n.TaskID,
		Kind:             n.Kind,
		Severity:         n.Severity,
		Title:            n.Title,
		Body:             n.Body,
		CreatedAt:        createdAt.UnixMilli(),
		RecipientActorID: nullActorID(n.RecipientActorID),
	})
	if err != nil {
		return err
	}

	n.ID = row.ID
	n.CreatedAt = createdAt
	s.hub.publish(n)
	return nil
}

// CreateForTask records a task notification for each recipient (its leads plus
// its subscribers). With no recipients it falls back to one broadcast entry,
// so a legacy or unattended task is not silently dropped.
func (s *Service) CreateForTask(
	ctx context.Context,
	taskID string,
	recipients []ids.ActorID,
	n Notification,
) error {
	n.TaskID = taskID
	if len(recipients) == 0 {
		return s.Create(ctx, n)
	}
	for _, recipient := range recipients {
		row := n
		row.RecipientActorID = recipient
		if err := s.Create(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// InboxDisabledKinds returns the notification kinds the inbox does not store.
func (s *Service) InboxDisabledKinds(ctx context.Context) ([]string, error) {
	rows, err := s.store.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.Key == keyInboxDisabledKinds {
			return splitKinds(row.Value), nil
		}
	}
	return nil, nil
}

// SetInboxDisabledKinds persists the kinds the inbox does not store.
func (s *Service) SetInboxDisabledKinds(ctx context.Context, kinds []string) error {
	return s.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   keyInboxDisabledKinds,
		Value: strings.Join(cleanKinds(kinds), ","),
	})
}

// skipInbox reports whether a kind is disabled; a settings read failure keeps
// the inbox storing everything rather than silently dropping signals.
func (s *Service) skipInbox(ctx context.Context, kind string) bool {
	kinds, err := s.InboxDisabledKinds(ctx)
	if err != nil {
		return false
	}
	for _, disabled := range kinds {
		if disabled == kind {
			return true
		}
	}
	return false
}

// splitKinds parses a comma-separated settings value, dropping blanks.
func splitKinds(value string) []string {
	return cleanKinds(strings.Split(value, ","))
}

// cleanKinds trims and dedupes a kind list, keeping order.
func cleanKinds(kinds []string) []string {
	seen := make(map[string]bool, len(kinds))
	out := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		kind = strings.TrimSpace(kind)
		if kind == "" || seen[kind] {
			continue
		}
		seen[kind] = true
		out = append(out, kind)
	}
	return out
}

// List returns the newest notifications visible to recipient (its own plus the
// broadcast rows), capped at limit.
func (s *Service) List(
	ctx context.Context,
	recipient ids.ActorID,
	limit int64,
) ([]Notification, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.store.ListNotifications(ctx, int64(recipient), limit)
	if err != nil {
		return nil, err
	}

	out := make([]Notification, len(rows))
	for i, row := range rows {
		n := Notification{
			ID:        row.ID,
			Kind:      row.Kind,
			Severity:  row.Severity,
			Title:     row.Title,
			Body:      row.Body,
			CreatedAt: time.UnixMilli(row.CreatedAt),
		}
		if row.TaskID.Valid {
			n.TaskID = row.TaskID.String
		}
		if row.ProjectID.Valid {
			n.ProjectID = row.ProjectID.String
		}
		if row.AgentID.Valid {
			n.AgentID = ids.AgentID(row.AgentID.Int64)
		}
		if row.RecipientActorID.Valid {
			n.RecipientActorID = ids.ActorID(row.RecipientActorID.Int64)
		}
		if row.ReadAt.Valid {
			n.ReadAt = time.UnixMilli(row.ReadAt.Int64)
		}
		out[i] = n
	}
	return out, nil
}

// UnreadCount returns the number of unread notifications visible to recipient.
func (s *Service) UnreadCount(ctx context.Context, recipient ids.ActorID) (int64, error) {
	return s.store.CountUnreadNotifications(ctx, int64(recipient))
}

// MarkRead marks one of the recipient's notifications read.
func (s *Service) MarkRead(ctx context.Context, recipient ids.ActorID, id int64) error {
	return s.store.MarkNotificationRead(ctx, store.MarkNotificationReadParams{
		ReadAt:           sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:               id,
		RecipientActorID: nullActorID(recipient),
	})
}

// MarkTaskRead marks every unread notification scoped to one task read, so
// visiting the task clears its signals without touching the rest of the inbox.
func (s *Service) MarkTaskRead(ctx context.Context, recipient ids.ActorID, taskID string) error {
	return s.store.MarkTaskNotificationsRead(
		ctx,
		store.MarkTaskNotificationsReadParams{
			ReadAt:           sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			TaskID:           taskID,
			RecipientActorID: nullActorID(recipient),
		},
	)
}

// MarkAllRead marks every notification visible to recipient read.
func (s *Service) MarkAllRead(ctx context.Context, recipient ids.ActorID) error {
	return s.store.MarkAllNotificationsRead(
		ctx,
		sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		int64(recipient),
	)
}

// Delete removes one of the recipient's notifications.
func (s *Service) Delete(ctx context.Context, recipient ids.ActorID, id int64) error {
	return s.store.DeleteNotification(ctx, id, int64(recipient))
}

// DeleteAll empties the recipient's inbox (and the broadcast entries).
func (s *Service) DeleteAll(ctx context.Context, recipient ids.ActorID) error {
	return s.store.DeleteAllNotifications(ctx, int64(recipient))
}

// nullActorID wraps an actor id for a recipient column; zero stays NULL.
func nullActorID(actorID ids.ActorID) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0}
}
