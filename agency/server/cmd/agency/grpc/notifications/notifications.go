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

package notificationsgrpc

import (
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"time"

	"cherrypi.vladfaust.com/agency/internal/notifications"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func caller(ctx context.Context) ids.ActorID {
	id, _ := auth.ActorID(ctx)
	return id
}

type NotificationServer struct {
	Notifications *notifications.Service
}

func (s *NotificationServer) ListNotifications(
	ctx context.Context,
	req *agencyv1.ListNotificationsRequest,
) (*agencyv1.ListNotificationsResponse, error) {
	list, err := s.Notifications.List(ctx, caller(ctx), int64(req.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := make([]*agencyv1.Notification, len(list))
	for i, n := range list {
		out[i] = &agencyv1.Notification{
			Id:        n.ID,
			AgentId:   actorid.Format(n.AgentID),
			TaskId:    n.TaskID,
			ProjectId: n.ProjectID,
			Kind:      n.Kind,
			Severity:  n.Severity,
			Title:     n.Title,
			Body:      n.Body,
			CreatedAt: timestamppb.New(n.CreatedAt),
			ReadAt:    timestampOrNil(n.ReadAt),
		}
	}
	return &agencyv1.ListNotificationsResponse{Notifications: out}, nil
}

func (s *NotificationServer) GetUnreadCount(
	ctx context.Context,
	_ *agencyv1.GetUnreadCountRequest,
) (*agencyv1.GetUnreadCountResponse, error) {
	count, err := s.Notifications.UnreadCount(ctx, caller(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.GetUnreadCountResponse{Count: count}, nil
}

func (s *NotificationServer) MarkNotificationRead(
	ctx context.Context,
	req *agencyv1.MarkNotificationReadRequest,
) (*agencyv1.MarkNotificationReadResponse, error) {
	if err := s.Notifications.MarkRead(ctx, caller(ctx), req.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.MarkNotificationReadResponse{}, nil
}

func (s *NotificationServer) MarkTaskNotificationsRead(
	ctx context.Context,
	req *agencyv1.MarkTaskNotificationsReadRequest,
) (*agencyv1.MarkTaskNotificationsReadResponse, error) {
	if err := s.Notifications.MarkTaskRead(ctx, caller(ctx), req.TaskId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.MarkTaskNotificationsReadResponse{}, nil
}

func (s *NotificationServer) MarkAllNotificationsRead(
	ctx context.Context,
	_ *agencyv1.MarkAllNotificationsReadRequest,
) (*agencyv1.MarkAllNotificationsReadResponse, error) {
	if err := s.Notifications.MarkAllRead(ctx, caller(ctx)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.MarkAllNotificationsReadResponse{}, nil
}

func (s *NotificationServer) DeleteNotification(
	ctx context.Context,
	req *agencyv1.DeleteNotificationRequest,
) (*agencyv1.DeleteNotificationResponse, error) {
	if err := s.Notifications.Delete(ctx, caller(ctx), req.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.DeleteNotificationResponse{}, nil
}

func (s *NotificationServer) DeleteAllNotifications(
	ctx context.Context,
	_ *agencyv1.DeleteAllNotificationsRequest,
) (*agencyv1.DeleteAllNotificationsResponse, error) {
	if err := s.Notifications.DeleteAll(ctx, caller(ctx)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.DeleteAllNotificationsResponse{}, nil
}

func (s *NotificationServer) GetInboxPreferences(
	ctx context.Context,
	_ *agencyv1.GetInboxPreferencesRequest,
) (*agencyv1.GetInboxPreferencesResponse, error) {
	kinds, err := s.Notifications.InboxDisabledKinds(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.GetInboxPreferencesResponse{DisabledKinds: kinds}, nil
}

func (s *NotificationServer) SetInboxPreferences(
	ctx context.Context,
	req *agencyv1.SetInboxPreferencesRequest,
) (*agencyv1.SetInboxPreferencesResponse, error) {
	if err := s.Notifications.SetInboxDisabledKinds(ctx, req.DisabledKinds); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	kinds, err := s.Notifications.InboxDisabledKinds(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.SetInboxPreferencesResponse{DisabledKinds: kinds}, nil
}

func timestampOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
