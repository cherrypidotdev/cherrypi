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
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
)

const keyInboxDisabledKinds = "notification_inbox_disabled_kinds"

type Notification struct {
	ID      int64
	AgentID ids.AgentID
	TaskID  string

	RecipientActorID ids.ActorID

	ProjectID string
	Kind      string
	Severity  string
	Title     string
	Body      string
	CreatedAt time.Time
	ReadAt    time.Time
}

type Service struct {
	store store.Store
	hub   *hub
}

func New(st store.Store) *Service { return &Service{store: st, hub: newHub()} }

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

func (s *Service) SetInboxDisabledKinds(ctx context.Context, kinds []string) error {
	return s.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   keyInboxDisabledKinds,
		Value: strings.Join(cleanKinds(kinds), ","),
	})
}

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

func splitKinds(value string) []string {
	return cleanKinds(strings.Split(value, ","))
}

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

func (s *Service) UnreadCount(ctx context.Context, recipient ids.ActorID) (int64, error) {
	return s.store.CountUnreadNotifications(ctx, int64(recipient))
}

func (s *Service) MarkRead(ctx context.Context, recipient ids.ActorID, id int64) error {
	return s.store.MarkNotificationRead(ctx, store.MarkNotificationReadParams{
		ReadAt:           sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:               id,
		RecipientActorID: nullActorID(recipient),
	})
}

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

func (s *Service) MarkAllRead(ctx context.Context, recipient ids.ActorID) error {
	return s.store.MarkAllNotificationsRead(
		ctx,
		sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		int64(recipient),
	)
}

func (s *Service) Delete(ctx context.Context, recipient ids.ActorID, id int64) error {
	return s.store.DeleteNotification(ctx, id, int64(recipient))
}

func (s *Service) DeleteAll(ctx context.Context, recipient ids.ActorID) error {
	return s.store.DeleteAllNotifications(ctx, int64(recipient))
}

func nullActorID(actorID ids.ActorID) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0}
}
